// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
package apisync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/surface"
)

func TestDeriveSurfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		spec string
		want func(*testing.T, []surface.Surface, []UnsupportedOperation)
	}{
		{
			name: "query array uses item enum and OpenAPI defaults",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    get:
      operationId: listWidgets
      tags: [widgets]
      parameters:
        - name: status
          in: query
          schema:
            type: array
            items:
              type: string
              enum: [open, closed]
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				requireFlag(t, got, surface.Flag{
					Name:    "status",
					In:      surface.InQuery,
					Key:     "status",
					Kind:    surface.KindString,
					Array:   true,
					Style:   surface.StyleForm,
					Explode: true,
					Enum:    []string{"open", "closed"},
				})
			},
		},
		{
			name: "scalar formats are preserved",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      parameters:
        - name: customer_id
          in: query
          schema:
            type: string
            format: uuid
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                starts_on:
                  type: string
                  format: date
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				requireFlag(t, got, surface.Flag{
					Name:    "customer-id",
					In:      surface.InQuery,
					Key:     "customer_id",
					Kind:    surface.KindString,
					Style:   surface.StyleForm,
					Explode: true,
					Format:  "uuid",
				})
				requireFlag(t, got, surface.Flag{
					Name:   "starts-on",
					In:     surface.InBody,
					Key:    "/starts_on",
					Kind:   surface.KindString,
					Format: "date",
				})
			},
		},
		{
			name: "nested required property has required ancestors",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [config]
              properties:
                config:
                  type: object
                  required: [auto_hold]
                  properties:
                    auto_hold:
                      type: boolean
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				requireFlag(t, got, surface.Flag{
					Name:     "config-auto-hold",
					In:       surface.InBody,
					Key:      "/config/auto_hold",
					Kind:     surface.KindBoolean,
					Required: true,
				})
			},
		},
		{
			name: "optional request body never yields required flags",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets/{id}/hold:
    put:
      operationId: holdWidget
      tags: [widgets]
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [reason]
              properties:
                reason:
                  type: string
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				if got.BodyRequired {
					t.Fatalf("BodyRequired = true, want false")
				}
				requireFlag(t, got, surface.Flag{
					Name: "reason",
					In:   surface.InBody,
					Key:  "/reason",
					Kind: surface.KindString,
				})
			},
		},
		{
			name: "nested required property has optional ancestor",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                config:
                  type: object
                  required: [auto_hold]
                  properties:
                    auto_hold:
                      type: boolean
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				requireFlag(t, got, surface.Flag{
					Name: "config-auto-hold",
					In:   surface.InBody,
					Key:  "/config/auto_hold",
					Kind: surface.KindBoolean,
				})
			},
		},
		{
			name: "metadata is one JSON flag",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                metadata:
                  type: object
                  additionalProperties:
                    type: string
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				requireFlag(t, got, surface.Flag{
					Name: "metadata",
					In:   surface.InBody,
					Key:  "/metadata",
					Kind: surface.KindJSON,
				})
			},
		},
		{
			name: "account header is represented by the surface bit",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    get:
      operationId: listWidgets
      tags: [widgets]
      parameters:
        - name: Straddle-Account-Id
          in: header
          schema:
            type: string
        - name: Request-Id
          in: header
          schema:
            type: string
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				if !got.AcceptsAccountHeader {
					t.Fatal("AcceptsAccountHeader = false, want true")
				}
				if flagByName(got.Flags, "straddle-account-id") != nil {
					t.Fatalf("Flags = %#v, want no straddle-account-id flag", got.Flags)
				}
				requireFlag(t, got, surface.Flag{
					Name:  "request-id",
					In:    surface.InHeader,
					Key:   "Request-Id",
					Kind:  surface.KindString,
					Style: surface.StyleSimple,
				})
			},
		},
		{
			name: "delete retains its request body",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets/{id}:
    delete:
      operationId: deleteWidget
      tags: [widgets]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [reason]
              properties:
                reason:
                  type: string
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				if !got.HasBody || !got.BodyRequired {
					t.Fatalf("HasBody = %t, BodyRequired = %t, want both true", got.HasBody, got.BodyRequired)
				}
			},
		},
		{
			name: "unrepresentable schema identifies its pointer",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                config:
                  type: object
                  properties:
                    mystery: {}
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				if len(surfaces) != 1 {
					t.Fatalf("surfaces = %#v, want the partial surface retained", surfaces)
				}
				if len(unsupported) != 1 {
					t.Fatalf("unsupported = %#v, want one operation", unsupported)
				}
				if !surfaceReasonContains(unsupported[0].Reasons, "/config/mystery") {
					t.Fatalf("reasons = %#v, want JSON pointer", unsupported[0].Reasons)
				}
			},
		},
		{
			name: "array path parameter is unsupported",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets/{tags}:
    get:
      operationId: listWidgetsByTags
      tags: [widgets]
      parameters:
        - name: tags
          in: path
          required: true
          style: simple
          explode: false
          schema:
            type: array
            items:
              type: string
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				if len(unsupported) != 1 {
					t.Fatalf("unsupported = %#v, want one array-path operation", unsupported)
				}
				if unsupported[0].Operation.Key != "GET /v1/widgets/{tags}" {
					t.Fatalf("unsupported key = %q, want GET /v1/widgets/{tags}", unsupported[0].Operation.Key)
				}
				if !surfaceReasonContains(unsupported[0].Reasons, `path parameter "tags" uses unsupported schema type array`) {
					t.Fatalf("reasons = %#v, want array path schema-type reason", unsupported[0].Reasons)
				}
			},
		},
		{
			name: "referenced array and object path parameters are unsupported",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets/{tags}:
    get:
      operationId: listWidgetsByTags
      tags: [widgets]
      parameters:
        - {name: tags, in: path, required: true, schema: {$ref: "#/components/schemas/TagList"}}
  /v1/widgets/by-filter/{filter}:
    get:
      operationId: listWidgetsByFilter
      tags: [widgets]
      parameters:
        - {name: filter, in: path, required: true, schema: {$ref: "#/components/schemas/Filter"}}
components:
  schemas:
    TagList:
      type: array
      items: {type: string}
    Filter:
      type: object
      properties:
        status: {type: string}
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				want := map[string]string{
					"GET /v1/widgets/by-filter/{filter}": `path parameter "filter" uses unsupported schema type object`,
					"GET /v1/widgets/{tags}":             `path parameter "tags" uses unsupported schema type array`,
				}
				if len(unsupported) != len(want) {
					t.Fatalf("unsupported = %#v, want both referenced path parameters rejected", unsupported)
				}
				for _, op := range unsupported {
					if !surfaceReasonContains(op.Reasons, want[op.Operation.Key]) {
						t.Fatalf("%s reasons = %#v, want %q", op.Operation.Key, op.Reasons, want[op.Operation.Key])
					}
				}
			},
		},
		{
			name: "array query parameter remains supported end to end",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    get:
      operationId: listWidgets
      tags: [widgets]
      parameters:
        - name: status
          in: query
          schema:
            type: array
            items:
              type: string
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				requireFlag(t, got, surface.Flag{
					Name:    "status",
					In:      surface.InQuery,
					Key:     "status",
					Kind:    surface.KindString,
					Array:   true,
					Style:   surface.StyleForm,
					Explode: true,
				})
			},
		},
		{
			name: "multipart binary property becomes a streamed file flag",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets/{id}/proof:
    post:
      operationId: uploadWidgetProof
      tags: [widgets]
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema:
              type: object
              required: [File]
              properties:
                File:
                  type: string
                  description: The document. More detail.
                  contentMediaType: application/octet-stream
            encoding:
              File:
                contentType: application/pdf, image/png
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				if !got.HasBody || !got.BodyRequired {
					t.Fatalf("surface = %#v, want a required body", got)
				}
				requireFlag(t, got, surface.Flag{
					Name:        "file",
					In:          surface.InForm,
					Key:         "File",
					Kind:        surface.KindFile,
					Required:    true,
					Enum:        []string{"application/pdf", "image/png"},
					Description: "The document.",
				})
			},
		},
		{
			name: "multipart upload may own a stdin query parameter",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets/proof:
    post:
      operationId: uploadWidgetProof
      tags: [widgets]
      parameters:
        - {name: stdin, in: query, schema: {type: string}}
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                File: {type: string, format: binary}
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				requireFlag(t, got, surface.Flag{Name: "stdin", In: surface.InQuery, Key: "stdin", Kind: surface.KindString, Style: surface.StyleForm, Explode: true})
				requireFlag(t, got, surface.Flag{Name: "file", In: surface.InForm, Key: "File", Kind: surface.KindFile})
			},
		},
		{
			name: "multipart text property stays unsupported",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                File: {type: string, format: binary}
                note: {type: string}
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				if len(unsupported) != 1 || !surfaceReasonContains(unsupported[0].Reasons, `multipart property "note" is not a file`) {
					t.Fatalf("unsupported = %#v, want the text property rejected", unsupported)
				}
			},
		},
		{
			name: "allOf enums intersect regardless of member order",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      parameters:
        - name: mode
          in: query
          schema:
            allOf:
              - {type: string, enum: [a, b, c]}
              - {enum: [b, c, d]}
      requestBody:
        content:
          application/json:
            schema:
              allOf:
                - type: object
                  properties:
                    status: {type: string, enum: [a, b, c]}
                - type: object
                  properties:
                    status: {type: string, enum: [b, c, d]}
                    reverse:
                      allOf:
                        - {type: string, enum: [b, c, d]}
                        - {type: string, enum: [a, b, c]}
                    tier:
                      allOf:
                        - {type: integer, enum: [1, 2]}
                        - {type: integer, enum: [2.0, 3]}
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				got := requireSingleSupportedSurface(t, surfaces, unsupported)
				requireFlag(t, got, surface.Flag{
					Name:    "mode",
					In:      surface.InQuery,
					Key:     "mode",
					Kind:    surface.KindString,
					Style:   surface.StyleForm,
					Explode: true,
					Enum:    []string{"b", "c"},
				})
				requireFlag(t, got, surface.Flag{Name: "status", In: surface.InBody, Key: "/status", Kind: surface.KindString, Enum: []string{"b", "c"}})
				requireFlag(t, got, surface.Flag{Name: "reverse", In: surface.InBody, Key: "/reverse", Kind: surface.KindString, Enum: []string{"b", "c"}})
				requireFlag(t, got, surface.Flag{Name: "tier", In: surface.InBody, Key: "/tier", Kind: surface.KindInteger, Enum: []string{"2"}})
			},
		},
		{
			name: "disjoint allOf enums leave the field unsupported",
			spec: `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name: {type: string}
                status:
                  allOf:
                    - {type: string, enum: [a, b]}
                    - {type: string, enum: [c, d]}
`,
			want: func(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) {
				t.Helper()
				if len(surfaces) != 1 || len(unsupported) != 1 {
					t.Fatalf("surfaces = %#v, unsupported = %#v, want one partial surface reported unsupported", surfaces, unsupported)
				}
				if !surfaceReasonContains(unsupported[0].Reasons, "conflicting allOf enums at /status") {
					t.Fatalf("reasons = %#v, want conflicting allOf enums at /status", unsupported[0].Reasons)
				}
				if flag := flagByName(surfaces[0].Flags, "status"); flag != nil {
					t.Fatalf("status flag = %#v, want no flag for a field no value satisfies", *flag)
				}
				requireFlag(t, surfaces[0], surface.Flag{Name: "name", In: surface.InBody, Key: "/name", Kind: surface.KindString})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			surfaces, unsupported, err := DeriveSurfaces(writeSurfaceSpec(t, test.spec))
			if err != nil {
				t.Fatalf("DeriveSurfaces: %v", err)
			}
			test.want(t, surfaces, unsupported)
		})
	}
}

func TestDriftSpecsReportsReferencedSchemaFieldChanges(t *testing.T) {
	t.Parallel()

	base := writeSurfaceSpec(t, `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateWidget'
components:
  schemas:
    CreateWidget:
      type: object
      properties:
        status:
          type: string
          enum: [open, closed]
`)
	head := writeSurfaceSpec(t, `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateWidget'
components:
  schemas:
    CreateWidget:
      type: object
      required: [amount]
      properties:
        amount:
          type: integer
        status:
          type: string
          enum: [open]
`)

	result, err := DriftSpecs(base, head)
	if err != nil {
		t.Fatalf("DriftSpecs: %v", err)
	}
	if len(result.Changes) != 1 {
		t.Fatalf("Changes = %#v, want one operation change", result.Changes)
	}
	fields := result.Changes[0].Fields
	if len(fields) != 2 {
		t.Fatalf("Fields = %#v, want two field changes", fields)
	}
	if fields[0].Flag != "amount" || fields[0].Kind != "added" {
		t.Fatalf("Fields[0] = %#v, want amount added", fields[0])
	}
	if fields[1].Flag != "status" || fields[1].Kind != "changed" {
		t.Fatalf("Fields[1] = %#v, want status changed", fields[1])
	}
}

func TestDriftSpecsReportsJSONObjectShapeChange(t *testing.T) {
	t.Parallel()

	spec := func(branch string) string {
		return `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                config:
                  anyOf:
                    - type: object
                    - type: ` + branch + `
`
	}
	result, err := DriftSpecs(writeSurfaceSpec(t, spec(`"null"`)), writeSurfaceSpec(t, spec("string")))
	if err != nil {
		t.Fatalf("DriftSpecs: %v", err)
	}
	if len(result.Changes) != 1 || len(result.Changes[0].Fields) != 1 {
		t.Fatalf("Changes = %#v, want one config field change", result.Changes)
	}
	if field := result.Changes[0].Fields[0]; field.Flag != "config" || !strings.Contains(field.Detail, "object") {
		t.Fatalf("field = %#v, want config change detail naming the object shape", field)
	}
}

func writeSurfaceSpec(t *testing.T, spec string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(spec)+"\n"), 0o600); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return path
}

func requireSingleSupportedSurface(t *testing.T, surfaces []surface.Surface, unsupported []UnsupportedOperation) surface.Surface {
	t.Helper()
	if len(unsupported) != 0 {
		t.Fatalf("unsupported = %#v, want none", unsupported)
	}
	if len(surfaces) != 1 {
		t.Fatalf("surfaces = %#v, want one", surfaces)
	}
	return surfaces[0]
}

func requireFlag(t *testing.T, got surface.Surface, want surface.Flag) {
	t.Helper()
	flag := flagByName(got.Flags, want.Name)
	if flag == nil {
		t.Fatalf("Flags = %#v, want %q", got.Flags, want.Name)
	}
	if flag.In != want.In || flag.Key != want.Key || flag.Kind != want.Kind || flag.Array != want.Array || flag.Style != want.Style || flag.Explode != want.Explode || flag.Required != want.Required || flag.Format != want.Format {
		t.Fatalf("flag %q = %#v, want %#v", want.Name, *flag, want)
	}
	if strings.Join(flag.Enum, ",") != strings.Join(want.Enum, ",") {
		t.Fatalf("flag %q enum = %#v, want %#v", want.Name, flag.Enum, want.Enum)
	}
}

func flagByName(flags []surface.Flag, name string) *surface.Flag {
	for i := range flags {
		if flags[i].Name == name {
			return &flags[i]
		}
	}
	return nil
}

func surfaceReasonContains(reasons []string, want string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, want) {
			return true
		}
	}
	return false
}

func TestDeriveSurfacesAcceptsParameterizedJSONBody(t *testing.T) {
	path := writeSurfaceSpec(t, `
openapi: 3.1.0
paths:
  /v1/widgets:
    post:
      operationId: createWidget
      tags: [widgets]
      requestBody:
        required: true
        content:
          application/json; charset=utf-8:
            schema:
              type: object
              properties:
                name:
                  type: string
`)
	surfaces, unsupported, err := DeriveSurfaces(path)
	if err != nil {
		t.Fatal(err)
	}
	got := requireSingleSupportedSurface(t, surfaces, unsupported)
	if !got.HasBody || !got.BodyRequired {
		t.Fatalf("body = (%t, %t), want required body", got.HasBody, got.BodyRequired)
	}
	requireFlag(t, got, surface.Flag{Name: "name", In: surface.InBody, Key: "/name", Kind: surface.KindString})
}
