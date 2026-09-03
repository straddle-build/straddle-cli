// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
package apisync

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

var methodOrder = map[string]int{
	"GET":     0,
	"POST":    1,
	"PUT":     2,
	"PATCH":   3,
	"DELETE":  4,
	"HEAD":    5,
	"OPTIONS": 6,
	"TRACE":   7,
}

type Operation struct {
	Key                   string      `json:"key"`
	OperationID           string      `json:"operation_id"`
	Endpoint              string      `json:"endpoint,omitempty"`
	Method                string      `json:"method"`
	Path                  string      `json:"path"`
	Summary               string      `json:"summary,omitempty"`
	Description           string      `json:"description,omitempty"`
	Tags                  []string    `json:"tags,omitempty"`
	PathParameters        []Parameter `json:"path_parameters,omitempty"`
	QueryParameters       []Parameter `json:"query_parameters,omitempty"`
	HeaderParameters      []Parameter `json:"header_parameters,omitempty"`
	RequestBodyRequired   bool        `json:"request_body_required,omitempty"`
	RequestBodyMediaTypes []string    `json:"request_body_media_types,omitempty"`
	RequestBodyRef        string      `json:"request_body_ref,omitempty"`
	ReadOnly              bool        `json:"read_only"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
	SchemaType  string `json:"schema_type,omitempty"`
	Style       string `json:"style,omitempty"`
	Explode     bool   `json:"explode,omitempty"`
}

type rawDocument struct {
	OpenAPI    string                                `json:"openapi"`
	Paths      map[string]map[string]json.RawMessage `json:"paths"`
	Components struct {
		Parameters map[string]rawParameter    `json:"parameters"`
		Schemas    map[string]json.RawMessage `json:"schemas"`
	} `json:"components"`
}

type parsedDocument struct {
	raw           rawDocument
	operations    []Operation
	rawOperations map[string]parsedOperation
}

type parsedOperation struct {
	operation  rawOperation
	parameters []rawParameter
}

type rawOperation struct {
	Tags        []string        `json:"tags"`
	Summary     string          `json:"summary"`
	Description string          `json:"description"`
	OperationID string          `json:"operationId"`
	Parameters  []rawParameter  `json:"parameters"`
	RequestBody *rawRequestBody `json:"requestBody"`
}

type rawParameter struct {
	Ref         string          `json:"$ref"`
	Name        string          `json:"name"`
	In          string          `json:"in"`
	Required    bool            `json:"required"`
	Description string          `json:"description"`
	Style       string          `json:"style"`
	Explode     *bool           `json:"explode"`
	Schema      json.RawMessage `json:"schema"`
}

type rawRequestBody struct {
	Ref      string                  `json:"$ref"`
	Required bool                    `json:"required"`
	Content  map[string]rawMediaType `json:"content"`
}

type rawMediaType struct {
	Schema json.RawMessage `json:"schema"`
}

func LoadSpec(path string) ([]Operation, error) {
	data, err := os.ReadFile(path) // #nosec G304: spec paths are explicit local CLI/workflow inputs.
	if err != nil {
		return nil, fmt.Errorf("reading spec %s: %w", path, err)
	}
	return ParseSpec(data)
}

func LoadSpecVersion(path string) (string, error) {
	data, err := os.ReadFile(path) // #nosec G304: spec paths are explicit local CLI/workflow inputs.
	if err != nil {
		return "", fmt.Errorf("reading spec %s: %w", path, err)
	}
	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return "", fmt.Errorf("parsing OpenAPI document: %w", err)
	}
	var document struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(jsonData, &document); err != nil {
		return "", fmt.Errorf("parsing OpenAPI document: %w", err)
	}
	if !exactVersionPattern.MatchString(document.Info.Version) {
		return "", fmt.Errorf("OpenAPI info.version must be exact semver")
	}
	return document.Info.Version, nil
}

func ParseSpec(data []byte) ([]Operation, error) {
	doc, err := parseDocument(data)
	if err != nil {
		return nil, err
	}
	return append([]Operation(nil), doc.operations...), nil
}

func loadParsedDocument(path string) (*parsedDocument, error) {
	data, err := os.ReadFile(path) // #nosec G304: spec paths are explicit local CLI/workflow inputs.
	if err != nil {
		return nil, fmt.Errorf("reading spec %s: %w", path, err)
	}
	return parseDocument(data)
}

func parseDocument(data []byte) (*parsedDocument, error) {
	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("parsing OpenAPI document: %w", err)
	}
	var rawDoc rawDocument
	if err := json.Unmarshal(jsonData, &rawDoc); err != nil {
		return nil, fmt.Errorf("parsing OpenAPI document: %w", err)
	}
	if strings.TrimSpace(rawDoc.OpenAPI) == "" {
		return nil, fmt.Errorf("missing openapi version")
	}
	if len(rawDoc.Paths) == 0 {
		return nil, fmt.Errorf("missing paths")
	}

	doc := &parsedDocument{
		raw:           rawDoc,
		rawOperations: make(map[string]parsedOperation),
	}
	for path, item := range rawDoc.Paths {
		pathParams, err := parseRawParameters(item["parameters"], path, rawDoc.Components.Parameters)
		if err != nil {
			return nil, err
		}
		for method, raw := range item {
			method = strings.ToUpper(method)
			if _, ok := methodOrder[method]; !ok {
				continue
			}
			var ro rawOperation
			if err := json.Unmarshal(raw, &ro); err != nil {
				return nil, fmt.Errorf("parsing %s %s operation: %w", method, path, err)
			}
			op := Operation{
				Key:         OperationKey(method, path),
				OperationID: ro.OperationID,
				Endpoint:    deriveEndpoint(ro.OperationID, ro.Tags),
				Method:      method,
				Path:        path,
				Summary:     ro.Summary,
				Description: ro.Description,
				Tags:        append([]string(nil), ro.Tags...),
				ReadOnly:    method == "GET" || method == "HEAD",
			}
			operationParams, err := resolveRawParameters(ro.Parameters, op.Key, rawDoc.Components.Parameters)
			if err != nil {
				return nil, err
			}
			parameters := mergeRawParameters(pathParams, operationParams)
			for _, p := range parameters {
				explode := p.Explode != nil && *p.Explode
				param := Parameter{
					Name:        p.Name,
					In:          p.In,
					Required:    p.Required,
					Description: p.Description,
					SchemaType:  rawSchemaType(p.Schema),
					Style:       p.Style,
					Explode:     explode,
				}
				switch p.In {
				case "path":
					op.PathParameters = append(op.PathParameters, param)
				case "query":
					op.QueryParameters = append(op.QueryParameters, param)
				case "header":
					op.HeaderParameters = append(op.HeaderParameters, param)
				default:
					op.QueryParameters = append(op.QueryParameters, param)
				}
			}
			if ro.RequestBody != nil {
				op.RequestBodyRef = strings.TrimSpace(ro.RequestBody.Ref)
				op.RequestBodyRequired = ro.RequestBody.Required
				for mediaType := range ro.RequestBody.Content {
					op.RequestBodyMediaTypes = append(op.RequestBodyMediaTypes, mediaType)
				}
				sort.Strings(op.RequestBodyMediaTypes)
			}
			doc.operations = append(doc.operations, op)
			doc.rawOperations[op.Key] = parsedOperation{
				operation:  ro,
				parameters: parameters,
			}
		}
	}
	SortOperations(doc.operations)
	return doc, nil
}

func rawSchemaType(raw json.RawMessage) string {
	var schema struct {
		Type json.RawMessage `json:"type"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		return ""
	}
	var single string
	if err := json.Unmarshal(schema.Type, &single); err == nil {
		return single
	}
	var multiple []string
	if err := json.Unmarshal(schema.Type, &multiple); err != nil {
		return ""
	}
	for _, schemaType := range multiple {
		if schemaType != "null" {
			if single != "" {
				return ""
			}
			single = schemaType
		}
	}
	return single
}

func parseRawParameters(raw json.RawMessage, context string, components map[string]rawParameter) ([]rawParameter, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var params []rawParameter
	if err := json.Unmarshal(raw, &params); err != nil {
		return nil, fmt.Errorf("parsing %s parameters: %w", context, err)
	}
	return resolveRawParameters(params, context, components)
}

func resolveRawParameters(params []rawParameter, context string, components map[string]rawParameter) ([]rawParameter, error) {
	resolved := make([]rawParameter, 0, len(params))
	for _, param := range params {
		if param.Ref == "" {
			resolved = append(resolved, param)
			continue
		}
		const prefix = "#/components/parameters/"
		if !strings.HasPrefix(param.Ref, prefix) {
			return nil, fmt.Errorf("unsupported %s parameter reference %s", context, param.Ref)
		}
		component, ok := components[strings.TrimPrefix(param.Ref, prefix)]
		if !ok {
			return nil, fmt.Errorf("missing %s parameter reference %s", context, param.Ref)
		}
		resolved = append(resolved, component)
	}
	return resolved, nil
}

func mergeRawParameters(pathParameters, operationParameters []rawParameter) []rawParameter {
	merged := append([]rawParameter(nil), pathParameters...)
	indexes := make(map[string]int, len(merged))
	for i, parameter := range merged {
		indexes[parameter.In+"\x00"+parameter.Name] = i
	}
	for _, parameter := range operationParameters {
		key := parameter.In + "\x00" + parameter.Name
		if index, ok := indexes[key]; ok {
			merged[index] = parameter
			continue
		}
		indexes[key] = len(merged)
		merged = append(merged, parameter)
	}
	return merged
}

func OperationKey(method, path string) string {
	return strings.ToUpper(method) + " " + path
}

func SortOperations(ops []Operation) {
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Path != ops[j].Path {
			return ops[i].Path < ops[j].Path
		}
		if methodOrder[ops[i].Method] != methodOrder[ops[j].Method] {
			return methodOrder[ops[i].Method] < methodOrder[ops[j].Method]
		}
		return ops[i].OperationID < ops[j].OperationID
	})
}

func deriveEndpoint(operationID string, tags []string) string {
	if operationID == "" {
		return ""
	}
	resource := "endpoint"
	if len(tags) > 0 && strings.TrimSpace(tags[0]) != "" {
		resource = kebab(tags[0])
		switch resource {
		case "charge":
			resource = "charges"
		case "payout":
			resource = "payouts"
		}
	}
	action, rest := splitAction(operationID)
	if action == "" {
		return resource + "." + kebab(operationID)
	}
	restKebab := kebab(rest)
	if restKebab == "" || restMatchesResource(restKebab, resource) {
		return resource + "." + action
	}
	return resource + "." + action + "-" + restKebab
}

func splitAction(operationID string) (string, string) {
	prefixes := []string{"Create", "Update", "Delete", "List", "Get", "Hold", "Release", "Cancel", "Resubmit", "Refund", "Upload", "Onboard", "Refresh", "Reveal", "Unmask", "Simulate"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(operationID), strings.ToLower(prefix)) && len(operationID) > len(prefix) {
			return strings.ToLower(prefix), operationID[len(prefix):]
		}
	}
	return "", operationID
}

func restMatchesResource(rest, resource string) bool {
	trimmed := strings.TrimSuffix(resource, "s")
	return rest == resource || rest == trimmed || rest+"s" == resource
}

func kebab(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	var prevDash bool
	for i, r := range s {
		if r == '_' || r == '-' || r == ' ' || r == '.' || r == '/' {
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
			continue
		}
		if i > 0 && isUpperASCII(r) && !prevDash {
			b.WriteByte('-')
		}
		b.WriteRune(toLowerASCII(r))
		prevDash = false
	}
	return strings.Trim(b.String(), "-")
}

func isUpperASCII(r rune) bool { return r >= 'A' && r <= 'Z' }
func toLowerASCII(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + ('a' - 'A')
	}
	return r
}
