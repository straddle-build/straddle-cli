// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/client"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

type boundRequest struct {
	Path    string
	Query   url.Values
	Headers map[string]string
	Body    any
}

type surfaceFlagBinding struct {
	definition   surface.Flag
	stringValue  string
	intValue     int
	floatValue   float64
	boolValue    bool
	stringValues []string
	intValues    []int
}

type surfaceFlagValue struct {
	body any
	wire []string
}

func bindSurface(cmd *cobra.Command, s surface.Surface) func(args []string) (boundRequest, error) {
	bindings := make([]*surfaceFlagBinding, 0, len(s.Flags))
	for _, definition := range s.Flags {
		binding := &surfaceFlagBinding{definition: definition}
		binding.register(cmd)
		if definition.Required {
			cmd.Flags().Lookup(definition.Name).Annotations = map[string][]string{"straddle:required": {"true"}}
		}
		bindings = append(bindings, binding)
	}

	var stdinBody bool
	isForm := hasFormFlag(s)
	if s.HasBody && !isForm {
		cmd.Flags().BoolVar(&stdinBody, "stdin", false, "Read JSON request body from stdin")
	}

	return func(args []string) (boundRequest, error) {
		hasBody := (s.HasBody && !isForm) || cmd.Annotations["straddle:body"] == "true"
		readStdin := stdinBody
		if hasBody && !s.HasBody {
			readStdin, _ = cmd.Flags().GetBool("stdin")
		}
		req := boundRequest{
			Path:    s.Path,
			Query:   url.Values{},
			Headers: map[string]string{},
		}
		if hasBody {
			req.Body = map[string]any{}
		}
		if len(args) < len(s.PathParams) {
			return req, usageErr(fmt.Errorf("%s is required", s.PathParams[len(args)]))
		}
		for i, name := range s.PathParams {
			req.Path = replacePathParam(req.Path, name, args[i])
		}

		if !readStdin {
			for _, binding := range bindings {
				if binding.definition.Required && !cmd.Flags().Changed(binding.definition.Name) {
					return req, fmt.Errorf("required flag %q not set", binding.definition.Name)
				}
			}
		}

		if readStdin {
			stdinData, err := io.ReadAll(cmd.InOrStdin())
			if err != nil {
				return req, fmt.Errorf("reading stdin: %w", err)
			}
			var body map[string]any
			if err := json.Unmarshal(stdinData, &body); err != nil {
				return req, fmt.Errorf("parsing stdin JSON: %w", err)
			}
			req.Body = body
		}

		var form client.MultipartForm
		for _, binding := range bindings {
			if err := binding.validateExplicitInput(cmd, req.Body, readStdin); err != nil {
				return req, usageErr(err)
			}
			definition := binding.definition
			if readStdin && definition.In == surface.InBody || !binding.included(cmd) {
				continue
			}
			if definition.In == surface.InForm {
				part, err := formFile(cmd, definition, binding.stringValue)
				if err != nil {
					return req, err
				}
				form.Files = append(form.Files, part)
				continue
			}
			value, err := binding.value()
			if err != nil {
				return req, err
			}
			if err := validateSurfaceEnum(cmd, definition, value.wire); err != nil {
				return req, err
			}
			switch definition.In {
			case surface.InQuery:
				for _, item := range value.wire {
					req.Query.Add(definition.Key, item)
				}
			case surface.InHeader:
				req.Headers[definition.Key] = strings.Join(value.wire, ",")
			case surface.InBody:
				setSurfaceBodyValue(req.Body.(map[string]any), definition.Key, value.body)
			}
		}
		if len(form.Files) > 0 {
			req.Body = form
		}
		return req, nil
	}
}

func (b *surfaceFlagBinding) register(cmd *cobra.Command) {
	definition := b.definition
	if definition.Array && definition.In == surface.InBody {
		b.stringValue = definition.Default
		cmd.Flags().StringVar(&b.stringValue, definition.Name, b.stringValue, surfaceFlagUsage(definition))
		return
	}
	if definition.Array {
		switch definition.Kind {
		case surface.KindString:
			b.stringValues = stringSliceDefault(definition.Default)
			cmd.Flags().StringSliceVar(&b.stringValues, definition.Name, b.stringValues, definition.Description)
		case surface.KindInteger:
			b.intValues = intSliceDefault(definition.Default)
			cmd.Flags().IntSliceVar(&b.intValues, definition.Name, b.intValues, definition.Description)
		default:
			panic(fmt.Sprintf("unsupported array flag kind %q for --%s", definition.Kind, definition.Name))
		}
		return
	}

	switch definition.Kind {
	case surface.KindString, surface.KindFile:
		b.stringValue = definition.Default
		cmd.Flags().StringVar(&b.stringValue, definition.Name, b.stringValue, definition.Description)
	case surface.KindJSON:
		b.stringValue = definition.Default
		cmd.Flags().StringVar(&b.stringValue, definition.Name, b.stringValue, surfaceFlagUsage(definition))
	case surface.KindInteger:
		b.intValue, _ = strconv.Atoi(definition.Default)
		cmd.Flags().IntVar(&b.intValue, definition.Name, b.intValue, definition.Description)
	case surface.KindNumber:
		b.floatValue, _ = strconv.ParseFloat(definition.Default, 64)
		cmd.Flags().Float64Var(&b.floatValue, definition.Name, b.floatValue, definition.Description)
	case surface.KindBoolean:
		b.boolValue, _ = strconv.ParseBool(definition.Default)
		cmd.Flags().BoolVar(&b.boolValue, definition.Name, b.boolValue, definition.Description)
	default:
		panic(fmt.Sprintf("unsupported flag kind %q for --%s", definition.Kind, definition.Name))
	}
}

func (b *surfaceFlagBinding) included(cmd *cobra.Command) bool {
	if b.definition.In == surface.InQuery && b.definition.Name == "page-number" {
		return true
	}
	return cmd.Flags().Changed(b.definition.Name) || b.definition.Default != ""
}

func (b *surfaceFlagBinding) value() (surfaceFlagValue, error) {
	if b.definition.Array && b.definition.In == surface.InBody {
		return b.jsonArrayValue()
	}
	if b.definition.Array {
		switch b.definition.Kind {
		case surface.KindString:
			return surfaceFlagValue{body: b.stringValues, wire: append([]string(nil), b.stringValues...)}, nil
		case surface.KindInteger:
			wire := make([]string, len(b.intValues))
			for i, value := range b.intValues {
				wire[i] = strconv.Itoa(value)
			}
			return surfaceFlagValue{body: b.intValues, wire: wire}, nil
		}
	}
	switch b.definition.Kind {
	case surface.KindString:
		return surfaceFlagValue{body: b.stringValue, wire: []string{b.stringValue}}, nil
	case surface.KindInteger:
		return surfaceFlagValue{body: b.intValue, wire: []string{strconv.Itoa(b.intValue)}}, nil
	case surface.KindNumber:
		wire := strconv.FormatFloat(b.floatValue, 'g', -1, 64)
		return surfaceFlagValue{body: b.floatValue, wire: []string{wire}}, nil
	case surface.KindBoolean:
		wire := strconv.FormatBool(b.boolValue)
		return surfaceFlagValue{body: b.boolValue, wire: []string{wire}}, nil
	case surface.KindJSON:
		return b.jsonValue()
	default:
		panic(fmt.Sprintf("unsupported flag kind %q for --%s", b.definition.Kind, b.definition.Name))
	}
}

func (b *surfaceFlagBinding) jsonValue() (surfaceFlagValue, error) {
	definition := b.definition
	var parsed any
	err := json.Unmarshal([]byte(b.stringValue), &parsed)
	if !definition.Object {
		if err != nil {
			return surfaceFlagValue{}, usageErr(fmt.Errorf("--%s expects a JSON value: %w", definition.Name, err))
		}
		return surfaceFlagValue{body: parsed, wire: []string{b.stringValue}}, nil
	}
	if _, isObject := parsed.(map[string]any); err != nil || (parsed != nil && !isObject) {
		return surfaceFlagValue{}, usageErr(fmt.Errorf("--%s expects a JSON object, for example --%s '%s'", definition.Name, definition.Name, jsonObjectExample(definition)))
	}
	return surfaceFlagValue{body: parsed, wire: []string{b.stringValue}}, nil
}

// surfaceFlagUsage is the help text for a flag: its description plus, for
// JSON-shaped body flags, the expected shape and a copyable example.
func surfaceFlagUsage(definition surface.Flag) string {
	switch {
	case definition.Array && definition.In == surface.InBody:
		return jsonArrayUsage(definition)
	case definition.Kind == surface.KindJSON && definition.Object:
		return jsonObjectUsage(definition)
	default:
		return definition.Description
	}
}

func jsonObjectUsage(definition surface.Flag) string {
	usage := strings.TrimRight(strings.TrimSpace(definition.Description), ".")
	if usage != "" {
		usage += "."
	}
	if usage == "" {
		usage = "JSON object."
	} else if !strings.Contains(strings.ToLower(usage), "json object") && !strings.Contains(strings.ToLower(usage), "an object") {
		usage += " JSON object."
	}
	return fmt.Sprintf("%s Example: --%s '%s'", usage, definition.Name, jsonObjectExample(definition))
}

func jsonObjectExample(definition surface.Flag) string {
	if definition.Name == "compliance-profile" {
		return `{"ein":"12-3456789","legal_business_name":"Acme Corp LLC"}`
	}
	return `{"key":"value"}`
}

func (b *surfaceFlagBinding) jsonArrayValue() (surfaceFlagValue, error) {
	definition := b.definition
	raw := []byte(b.stringValue)
	switch definition.Kind {
	case surface.KindString:
		var values []string
		if err := json.Unmarshal(raw, &values); err == nil && values != nil {
			return surfaceFlagValue{body: values, wire: values}, nil
		}
	case surface.KindInteger:
		var values []int
		if err := json.Unmarshal(raw, &values); err == nil && values != nil {
			wire := make([]string, len(values))
			for i, value := range values {
				wire[i] = strconv.Itoa(value)
			}
			return surfaceFlagValue{body: values, wire: wire}, nil
		}
	default:
		panic(fmt.Sprintf("unsupported array flag kind %q for --%s", definition.Kind, definition.Name))
	}
	return surfaceFlagValue{}, usageErr(fmt.Errorf("--%s expects a JSON array of %ss, for example --%s '%s'", definition.Name, definition.Kind, definition.Name, jsonArrayExample(definition)))
}

func jsonArrayUsage(definition surface.Flag) string {
	usage := strings.TrimSpace(definition.Description + " JSON array of " + string(definition.Kind) + "s")
	if len(definition.Enum) > 0 {
		usage += "; items one of: " + strings.Join(definition.Enum, ", ")
	}
	return fmt.Sprintf("%s. Example: --%s '%s'", usage, definition.Name, jsonArrayExample(definition))
}

func jsonArrayExample(definition surface.Flag) string {
	items := definition.Enum
	if len(items) > 2 {
		items = items[:2]
	}
	if definition.Kind == surface.KindInteger {
		if len(items) == 0 {
			items = []string{"1"}
		}
		return "[" + strings.Join(items, ",") + "]"
	}
	if len(items) == 0 {
		items = []string{"value"}
	}
	example, _ := json.Marshal(items)
	return string(example)
}

func validateSurfaceEnum(cmd *cobra.Command, definition surface.Flag, values []string) error {
	allowedValues := definition.Enum
	if flag := cmd.Flags().Lookup(definition.Name); flag != nil {
		if annotated, exists := flag.Annotations["straddle:enum"]; exists && len(annotated) > 0 {
			allowedValues = annotated
		}
	}
	if len(allowedValues) == 0 {
		return nil
	}
	for _, value := range values {
		valid := false
		for _, allowed := range allowedValues {
			if value == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid value %q for --%s (allowed: %s)", value, definition.Name, strings.Join(allowedValues, ", "))
		}
	}
	return nil
}

func setSurfaceBodyValue(body map[string]any, pointer string, value any) {
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	current := body
	for i, part := range parts {
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		if i == len(parts)-1 {
			current[part] = value
			return
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
}

func stringSliceDefault(raw string) []string {
	if raw == "" {
		return nil
	}
	var values []string
	if json.Unmarshal([]byte(raw), &values) == nil {
		return values
	}
	return []string{raw}
}

func intSliceDefault(raw string) []int {
	if raw == "" {
		return nil
	}
	var values []int
	if json.Unmarshal([]byte(raw), &values) == nil {
		return values
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil
	}
	return []int{value}
}

func executeSurface(cmd *cobra.Command, flags *rootFlags, s surface.Surface, req boundRequest) error {
	c, err := flags.newClient()
	if err != nil {
		return err
	}
	if s.Method == "GET" {
		resourceType := strings.SplitN(s.Endpoint, ".", 2)[0]
		if resource := cmd.Annotations["straddle:resource"]; resource != "" {
			resourceType = resource
		}
		var data json.RawMessage
		var provenance DataProvenance
		if cmd.Flags().Lookup("all") != nil {
			fetchAll, flagErr := cmd.Flags().GetBool("all")
			if flagErr != nil {
				return flagErr
			}
			data, provenance, err = resolvePaginatedReadWithValues(cmd.Context(), c, flags, resourceType, req.Path, req.Query, req.Headers, fetchAll, "page_number", "", "")
		} else {
			data, provenance, err = resolveReadWithValues(cmd.Context(), c, flags, resourceType, false, req.Path, req.Query, req.Headers)
		}
		if err != nil {
			return classifyAPIError(err, flags)
		}
		return printSurfaceReadOutput(cmd, flags, data, provenance)
	}

	data, statusCode, err := c.DoWithValues(s.Method, req.Path, req.Query, req.Body, req.Headers)
	if err != nil {
		if s.Method == "DELETE" {
			return classifyDeleteError(err, flags)
		}
		return classifyAPIError(err, flags)
	}
	return printGeneratedMutationOutput(cmd, flags, s.Method, s.Endpoint, req.Path, statusCode, data)
}

func printSurfaceReadOutput(cmd *cobra.Command, flags *rootFlags, data json.RawMessage, provenance DataProvenance) error {
	// Unwrap the Straddle response envelope for HUMAN display only when the
	// command opts in via straddle:unwrap-response. Machine output (JSON/csv/
	// plain/quiet/select) keeps the full envelope (see `data` below) so meta
	// (api_request_id, pagination) stays accessible to consumers — see
	// unwrapSingleKeyArray's multi-key pass-through policy.
	humanData := data
	if cmd.Annotations["straddle:unwrap-response"] == "true" {
		humanData = extractResponseData(data)
	}
	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		var items []json.RawMessage
		if json.Unmarshal(humanData, &items) != nil {
			items = []json.RawMessage{humanData}
		}
		printProvenance(cmd, len(items), provenance)
	}
	if flags.asJSON || (!isTerminal(cmd.OutOrStdout()) && !flags.csv && !flags.quiet && !flags.plain) {
		filtered, err := projectOutput(data, flags)
		if err != nil {
			return err
		}
		wrapped, err := wrapWithProvenance(filtered, provenance)
		if err != nil {
			return err
		}
		return printOutput(cmd.OutOrStdout(), wrapped, true)
	}
	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		var items []map[string]any
		if json.Unmarshal(humanData, &items) == nil && len(items) > 0 {
			if err := printAutoTable(cmd.OutOrStdout(), items); err != nil {
				return err
			}
			if len(items) >= 25 {
				fmt.Fprintf(cmd.ErrOrStderr(), "\nShowing %d results. To narrow: add --limit, --json --select, or filter flags.\n", len(items))
			}
			return nil
		}
	}
	// Human-terminal fall-through (table declined, e.g. a single-object
	// response) renders the unwrapped payload so the inner resource is shown;
	// machine fall-through (csv/plain/quiet/select) keeps the full envelope.
	displayData := data
	if wantsHumanTable(cmd.OutOrStdout(), flags) {
		displayData = humanData
	}
	return printOutputWithFlags(cmd.OutOrStdout(), displayData, flags)
}
