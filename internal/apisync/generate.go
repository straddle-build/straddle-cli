// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
package apisync

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/straddle-build/straddle-cli/internal/surface"
)

type GeneratedFile struct {
	Path      string    `json:"path"`
	Operation Operation `json:"operation"`
	Content   string    `json:"-"`
}

type GenerateResult struct {
	Generated             []string               `json:"generated"`
	Deleted               []string               `json:"deleted"`
	Unchanged             []string               `json:"unchanged"`
	UnsupportedOperations []UnsupportedOperation `json:"unsupported"`
	DryRun                bool                   `json:"dry_run"`
}

type generatedEndpointFile struct {
	Path string
	Key  string
}

func GenerateAll(specPath, repo string, dryRun bool) (GenerateResult, error) {
	result := GenerateResult{
		Generated:             []string{},
		Deleted:               []string{},
		Unchanged:             []string{},
		UnsupportedOperations: []UnsupportedOperation{},
		DryRun:                dryRun,
	}
	operations, err := LoadSpec(specPath)
	if err != nil {
		return result, err
	}
	surfaces, unsupported, err := DeriveSurfaces(specPath)
	if err != nil {
		return result, err
	}
	result.UnsupportedOperations = append(result.UnsupportedOperations, unsupported...)

	operationsByKey := make(map[string]Operation, len(operations))
	for _, op := range operations {
		operationsByKey[op.Key] = op
	}
	unsupportedKeys := make(map[string]bool, len(unsupported))
	for _, item := range unsupported {
		unsupportedKeys[item.Operation.Key] = true
	}
	supportedKeys := make(map[string]bool, len(surfaces)-len(unsupported))
	for _, commandSurface := range surfaces {
		key := OperationKey(commandSurface.Method, commandSurface.Path)
		if !unsupportedKeys[key] {
			supportedKeys[key] = true
		}
	}

	outDir := filepath.Join(repo, "internal", "cli")
	existingGenerated, err := inventoryGeneratedEndpointFiles(outDir)
	if err != nil {
		return result, err
	}
	generatedPathByKey := make(map[string]string, len(existingGenerated))
	generatedKeyByPath := make(map[string]string, len(existingGenerated))
	for _, existing := range existingGenerated {
		generatedKeyByPath[existing.Path] = existing.Key
		if supportedKeys[existing.Key] {
			if previous := generatedPathByKey[existing.Key]; previous != "" {
				return result, fmt.Errorf("supported operation %s is registered by both %s and %s", existing.Key, previous, existing.Path)
			}
			generatedPathByKey[existing.Key] = existing.Path
			continue
		}
		result.Deleted = append(result.Deleted, existing.Path)
		if dryRun {
			continue
		}
		if err := os.Remove(existing.Path); err != nil {
			return result, fmt.Errorf("deleting %s: %w", existing.Path, err)
		}
	}

	for _, commandSurface := range surfaces {
		key := OperationKey(commandSurface.Method, commandSurface.Path)
		if unsupportedKeys[key] {
			continue
		}
		op, ok := operationsByKey[key]
		if !ok {
			return result, fmt.Errorf("derived surface %s has no matching operation", key)
		}
		basePath := filepath.Join(outDir, fileName(op))
		resolvedPath, err := resolveGeneratedPath(basePath, key, generatedPathByKey, generatedKeyByPath, supportedKeys)
		if err != nil {
			return result, err
		}
		funcName := functionName(op)
		if resolvedPath != basePath {
			funcName = generatedFunctionName(op)
		}
		file, err := generateEndpointFile(commandSurface, op, outDir, funcName)
		if err != nil {
			return result, fmt.Errorf("generating %s: %w", key, err)
		}
		file.Path = resolvedPath
		generatedPathByKey[key] = file.Path
		generatedKeyByPath[file.Path] = key
		current, err := os.ReadFile(file.Path) // #nosec G304 -- generated source paths are rooted under the explicit repository.
		if err == nil && bytes.Equal(current, []byte(file.Content)) {
			result.Unchanged = append(result.Unchanged, file.Path)
			continue
		}
		if err != nil && !os.IsNotExist(err) {
			return result, fmt.Errorf("reading %s: %w", file.Path, err)
		}
		result.Generated = append(result.Generated, file.Path)
		if dryRun {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(file.Path), 0o750); err != nil {
			return result, fmt.Errorf("creating %s: %w", filepath.Dir(file.Path), err)
		}
		if err := os.WriteFile(file.Path, []byte(file.Content), 0o644); err != nil { // #nosec G306 -- generated source files are intended repo artifacts.
			return result, fmt.Errorf("writing %s: %w", file.Path, err)
		}
	}

	sort.Strings(result.Generated)
	sort.Strings(result.Deleted)
	sort.Strings(result.Unchanged)
	return result, nil
}

func GenerateEndpointFile(commandSurface surface.Surface, op Operation, outDir string) (GeneratedFile, error) {
	return generateEndpointFile(commandSurface, op, outDir, functionName(op))
}

func generateEndpointFile(commandSurface surface.Surface, op Operation, outDir, funcName string) (GeneratedFile, error) {
	reasons := append(UnsupportedReasons(op), surfaceUnsupportedReasons(commandSurface)...)
	if len(reasons) > 0 {
		key := op.Key
		if key == "" {
			key = OperationKey(commandSurface.Method, commandSurface.Path)
		}
		return GeneratedFile{}, fmt.Errorf("unsupported operation %s: %s", key, strings.Join(uniqueSortedStrings(reasons), "; "))
	}
	data := fileTemplateData{
		FuncName:       funcName,
		CommandUse:     commandUse(op),
		Short:          firstSentence(op),
		Example:        example(op),
		Endpoint:       commandSurface.Endpoint,
		OperationID:    commandSurface.OperationID,
		Method:         commandSurface.Method,
		Path:           commandSurface.Path,
		ReadOnly:       commandSurface.ReadOnly,
		SurfaceLiteral: renderSurfaceLiteral(commandSurface),
	}
	var buf bytes.Buffer
	if err := endpointTemplate.Execute(&buf, data); err != nil {
		return GeneratedFile{}, err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("formatting generated %s: %w", op.Key, err)
	}
	return GeneratedFile{
		Path:      filepath.Join(outDir, fileName(op)),
		Operation: op,
		Content:   string(formatted),
	}, nil
}

func inventoryGeneratedEndpointFiles(dir string) ([]generatedEndpointFile, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return []generatedEndpointFile{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("stat %s: %w", dir, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}
	var files []generatedEndpointFile
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path) // #nosec G304 -- inventory is limited to the explicit repository's CLI source directory.
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		if !hasGeneratedSurfaceRegistration(content) {
			continue
		}
		key, err := registeredSurfaceKey(path, content)
		if err != nil {
			return nil, err
		}
		files = append(files, generatedEndpointFile{Path: path, Key: key})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func resolveGeneratedPath(basePath, key string, generatedPathByKey, generatedKeyByPath map[string]string, supportedKeys map[string]bool) (string, error) {
	if existing := generatedPathByKey[key]; existing != "" {
		return existing, nil
	}
	available, err := generatedPathAvailable(basePath, generatedKeyByPath, supportedKeys)
	if err != nil {
		return "", err
	}
	if available {
		return basePath, nil
	}
	ext := filepath.Ext(basePath)
	alternate := strings.TrimSuffix(basePath, ext) + "_generated" + ext
	available, err = generatedPathAvailable(alternate, generatedKeyByPath, supportedKeys)
	if err != nil {
		return "", err
	}
	if available {
		return alternate, nil
	}
	return "", fmt.Errorf("generated paths %s and %s for %s are both occupied", basePath, alternate, key)
}

func generatedPathAvailable(path string, generatedKeyByPath map[string]string, supportedKeys map[string]bool) (bool, error) {
	if owner, ok := generatedKeyByPath[path]; ok {
		return !supportedKeys[owner], nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("stat %s: %w", path, err)
	}
	return false, nil
}

func registeredSurfaceKey(path string, content []byte) (string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, content, 0)
	if err != nil {
		return "", fmt.Errorf("parsing generated file %s: %w", path, err)
	}
	var method string
	var apiPath string
	var registrationCount int
	var registrationErr error
	ast.Inspect(file, func(node ast.Node) bool {
		if registrationErr != nil {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok || name.Name != "registerSurface" {
			return true
		}
		registrationCount++
		if len(call.Args) != 1 {
			registrationErr = fmt.Errorf("registerSurface in %s has %d arguments, want 1", path, len(call.Args))
			return false
		}
		literal, ok := call.Args[0].(*ast.CompositeLit)
		if !ok {
			registrationErr = fmt.Errorf("registerSurface in %s does not contain a surface literal", path)
			return false
		}
		for _, element := range literal.Elts {
			field, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			fieldName, ok := field.Key.(*ast.Ident)
			if !ok || fieldName.Name != "Method" && fieldName.Name != "Path" {
				continue
			}
			value, ok := field.Value.(*ast.BasicLit)
			if !ok || value.Kind != token.STRING {
				registrationErr = fmt.Errorf("%s field in %s is not a string literal", fieldName.Name, path)
				return false
			}
			unquoted, err := strconv.Unquote(value.Value)
			if err != nil {
				registrationErr = fmt.Errorf("parsing %s field in %s: %w", fieldName.Name, path, err)
				return false
			}
			if fieldName.Name == "Method" {
				method = unquoted
			} else {
				apiPath = unquoted
			}
		}
		return false
	})
	if registrationErr != nil {
		return "", registrationErr
	}
	if registrationCount != 1 {
		return "", fmt.Errorf("generated file %s has %d surface registrations, want 1", path, registrationCount)
	}
	if method == "" || apiPath == "" {
		return "", fmt.Errorf("registered surface in %s must contain Method and Path", path)
	}
	return OperationKey(method, apiPath), nil
}

func hasGeneratedSurfaceRegistration(content []byte) bool {
	marker := []byte("registerSurface(surface.Surface{")
	for _, line := range bytes.Split(content, []byte{'\n'}) {
		if bytes.Equal(bytes.TrimSpace(line), marker) {
			return true
		}
	}
	return false
}

type fileTemplateData struct {
	FuncName       string
	CommandUse     string
	Short          string
	Example        string
	Endpoint       string
	OperationID    string
	Method         string
	Path           string
	ReadOnly       bool
	SurfaceLiteral string
}

var endpointTemplate = template.Must(template.New("endpoint").Parse(`// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func init() {
	registerGeneratedEndpoint({{ printf "%q" .Endpoint }}, {{ .FuncName }})
	registerSurface({{ .SurfaceLiteral }})
}

func {{ .FuncName }}(flags *rootFlags) *cobra.Command {
	s := {{ .SurfaceLiteral }}
	cmd := &cobra.Command{
		Use:     {{ printf "%q" .CommandUse }},
		Short:   {{ printf "%q" .Short }},
		Example: {{ printf "%q" .Example }},
		Annotations: map[string]string{
			"straddle:endpoint":     {{ printf "%q" .Endpoint }},
			"straddle:operation-id": {{ printf "%q" .OperationID }},
			"straddle:method":       {{ printf "%q" .Method }},
			"straddle:path":         {{ printf "%q" .Path }},
			{{- if .ReadOnly }}
			"mcp:read-only":         "true",
			{{- end }}
		},
	}
	bind := bindSurface(cmd, flags, s)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		req, err := bind(args)
		if err != nil {
			return err
		}
		return executeSurface(cmd, flags, s, req)
	}
	applyOverlay({{ printf "%q" .Endpoint }}, cmd)
	return cmd
}
`))

func renderSurfaceLiteral(commandSurface surface.Surface) string {
	var b strings.Builder
	b.WriteString("surface.Surface{\n")
	fmt.Fprintf(&b, "\tEndpoint: %q,\n", commandSurface.Endpoint)
	fmt.Fprintf(&b, "\tOperationID: %q,\n", commandSurface.OperationID)
	fmt.Fprintf(&b, "\tMethod: %q,\n", commandSurface.Method)
	fmt.Fprintf(&b, "\tPath: %q,\n", commandSurface.Path)
	b.WriteString("\tPathParams: []string{")
	writeQuotedStrings(&b, commandSurface.PathParams)
	b.WriteString("},\n")
	b.WriteString("\tFlags: []surface.Flag{\n")
	for _, flag := range commandSurface.Flags {
		b.WriteString("\t\t{\n")
		if flag.Name != "" {
			fmt.Fprintf(&b, "\t\t\tName: %q,\n", flag.Name)
		}
		if flag.In != "" {
			fmt.Fprintf(&b, "\t\t\tIn: %s,\n", surfaceInLiteral(flag.In))
		}
		if flag.Key != "" {
			fmt.Fprintf(&b, "\t\t\tKey: %q,\n", flag.Key)
		}
		if flag.Kind != "" {
			fmt.Fprintf(&b, "\t\t\tKind: %s,\n", surfaceKindLiteral(flag.Kind))
		}
		if flag.Array {
			b.WriteString("\t\t\tArray: true,\n")
		}
		if flag.Style != "" {
			fmt.Fprintf(&b, "\t\t\tStyle: %s,\n", surfaceStyleLiteral(flag.Style))
		}
		if flag.Explode {
			b.WriteString("\t\t\tExplode: true,\n")
		}
		if flag.Required {
			b.WriteString("\t\t\tRequired: true,\n")
		}
		if len(flag.Enum) > 0 {
			b.WriteString("\t\t\tEnum: []string{")
			writeQuotedStrings(&b, flag.Enum)
			b.WriteString("},\n")
		}
		if flag.Description != "" {
			fmt.Fprintf(&b, "\t\t\tDescription: %q,\n", flag.Description)
		}
		if flag.Format != "" {
			fmt.Fprintf(&b, "\t\t\tFormat: %q,\n", flag.Format)
		}
		if flag.Default != "" {
			fmt.Fprintf(&b, "\t\t\tDefault: %q,\n", flag.Default)
		}
		b.WriteString("\t\t},\n")
	}
	b.WriteString("\t},\n")
	fmt.Fprintf(&b, "\tHasBody: %t,\n", commandSurface.HasBody)
	fmt.Fprintf(&b, "\tBodyRequired: %t,\n", commandSurface.BodyRequired)
	fmt.Fprintf(&b, "\tAcceptsAccountHeader: %t,\n", commandSurface.AcceptsAccountHeader)
	fmt.Fprintf(&b, "\tReadOnly: %t,\n", commandSurface.ReadOnly)
	b.WriteString("}")
	return b.String()
}

func writeQuotedStrings(b *strings.Builder, values []string) {
	for i, value := range values {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "%q", value)
	}
}

func surfaceInLiteral(value surface.In) string {
	switch value {
	case surface.InQuery:
		return "surface.InQuery"
	case surface.InHeader:
		return "surface.InHeader"
	case surface.InBody:
		return "surface.InBody"
	default:
		return fmt.Sprintf("surface.In(%q)", value)
	}
}

func surfaceKindLiteral(value surface.Kind) string {
	switch value {
	case surface.KindString:
		return "surface.KindString"
	case surface.KindInteger:
		return "surface.KindInteger"
	case surface.KindNumber:
		return "surface.KindNumber"
	case surface.KindBoolean:
		return "surface.KindBoolean"
	case surface.KindJSON:
		return "surface.KindJSON"
	default:
		return fmt.Sprintf("surface.Kind(%q)", value)
	}
}

func surfaceStyleLiteral(value surface.Style) string {
	switch value {
	case surface.StyleForm:
		return "surface.StyleForm"
	case surface.StyleSimple:
		return "surface.StyleSimple"
	default:
		return fmt.Sprintf("surface.Style(%q)", value)
	}
}

func fileName(op Operation) string {
	name := op.Endpoint
	if name == "" {
		name = strings.ToLower(op.Method) + "-" + strings.Trim(op.Path, "/")
	}
	name = strings.NewReplacer(".", "_", "/", "_", "{", "", "}", "", " ", "_", "-", "-").Replace(name)
	return name + ".go"
}

func functionName(op Operation) string {
	base := op.Endpoint
	if base == "" {
		base = strings.ToLower(op.Method) + " " + op.Path
	}
	parts := strings.FieldsFunc(base, func(r rune) bool {
		return r == '.' || r == '-' || r == '_' || r == '/' || r == ' ' || r == '{' || r == '}'
	})
	var b strings.Builder
	b.WriteString("new")
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	b.WriteString("Cmd")
	return b.String()
}

func generatedFunctionName(op Operation) string {
	return strings.TrimSuffix(functionName(op), "Cmd") + "GeneratedCmd"
}

func commandUse(op Operation) string {
	name := op.Endpoint
	if dot := strings.LastIndex(name, "."); dot >= 0 && dot < len(name)-1 {
		name = name[dot+1:]
	}
	if name == "" {
		name = strings.ToLower(op.Method)
	}
	for _, param := range op.PathParameters {
		name += " <" + param.Name + ">"
	}
	return name
}

func firstSentence(op Operation) string {
	text := strings.TrimSpace(op.Summary)
	if text == "" {
		text = strings.TrimSpace(op.Description)
	}
	if text == "" {
		return op.Method + " " + op.Path
	}
	text = strings.ReplaceAll(text, "\n", " ")
	if len(text) > 120 {
		text = strings.TrimSpace(text[:117]) + "..."
	}
	return text
}

func example(op Operation) string {
	example := "  straddle " + strings.ReplaceAll(op.Endpoint, ".", " ")
	for _, param := range op.PathParameters {
		example += " <" + param.Name + ">"
	}
	return example
}

func flagName(param Parameter) string {
	return strings.ToLower(strings.ReplaceAll(param.Name, "_", "-"))
}

func generatedHeaderParameters(params []Parameter) []Parameter {
	headers := make([]Parameter, 0, len(params))
	for _, param := range params {
		if strings.EqualFold(param.Name, "Straddle-Account-Id") {
			continue
		}
		headers = append(headers, param)
	}
	return headers
}

func paramVarName(param Parameter) string {
	parts := strings.FieldsFunc(param.Name, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	if len(parts) == 0 {
		return "flagValue"
	}
	var b strings.Builder
	b.WriteString("flag")
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(part[1:])
		}
	}
	return b.String()
}
