// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package apisync_test

import (
	"go/format"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/apisync"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

func TestGenerateEndpointFile(t *testing.T) {
	t.Parallel()

	commandSurface := surface.Surface{
		Endpoint:    "widgets.list",
		OperationID: "ListWidgets",
		Method:      "GET",
		Path:        "/v1/widgets/{id}",
		PathParams:  []string{"id"},
		Flags: []surface.Flag{
			{
				Name:        "status",
				In:          surface.InQuery,
				Key:         "status",
				Kind:        surface.KindString,
				Array:       true,
				Style:       surface.StyleForm,
				Explode:     true,
				Required:    true,
				Enum:        []string{"active", "inactive"},
				Description: "Filter by status.",
				Default:     "active",
			},
			{
				Name:        "request-id",
				In:          surface.InHeader,
				Key:         "Request-Id",
				Kind:        surface.KindString,
				Format:      "uuid",
				Style:       surface.StyleSimple,
				Description: "Trace one request.",
			},
		},
		AcceptsAccountHeader: true,
		ReadOnly:             true,
	}
	op := apisync.Operation{
		Key:         apisync.OperationKey(commandSurface.Method, commandSurface.Path),
		OperationID: commandSurface.OperationID,
		Endpoint:    commandSurface.Endpoint,
		Method:      commandSurface.Method,
		Path:        commandSurface.Path,
		Summary:     "List widgets. Additional detail is omitted.",
		PathParameters: []apisync.Parameter{
			{Name: "id", In: "path", Required: true, SchemaType: "string"},
		},
		ReadOnly: true,
	}

	for _, tc := range []struct {
		name    string
		surface surface.Surface
		op      apisync.Operation
		wantErr string
	}{
		{name: "renders declarative command", surface: commandSurface, op: op},
		{
			name:    "rejects unsupported operation",
			surface: commandSurface,
			op: func() apisync.Operation {
				unsupported := op
				unsupported.Method = "TRACE"
				unsupported.Key = apisync.OperationKey(unsupported.Method, unsupported.Path)
				return unsupported
			}(),
			wantErr: "unsupported HTTP method TRACE",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			file, err := apisync.GenerateEndpointFile(tc.surface, tc.op, t.TempDir())
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("GenerateEndpointFile error = %v, want containing %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GenerateEndpointFile: %v", err)
			}
			if _, err := format.Source([]byte(file.Content)); err != nil {
				t.Fatalf("format generated source: %v\n%s", err, file.Content)
			}
			for _, want := range []string{
				`registerGeneratedEndpoint("widgets.list", newWidgetsListCmd)`,
				"registerSurface(surface.Surface{",
				"s := surface.Surface{",
				"PathParams:",
				`[]string{"id"}`,
				"Enum:",
				`[]string{"active", "inactive"}`,
				`"mcp:read-only":`,
				"Format:",
				`"uuid"`,
				"bind := bindSurface(cmd, flags, s)",
				"return executeSurface(cmd, flags, s, req)",
				`applyOverlay("widgets.list", cmd)`,
			} {
				if !strings.Contains(file.Content, want) {
					t.Fatalf("generated content missing %q:\n%s", want, file.Content)
				}
			}
			for _, pattern := range []string{
				`HasBody:\s+false`,
				`BodyRequired:\s+false`,
				`AcceptsAccountHeader:\s+true`,
				`ReadOnly:\s+true`,
			} {
				if !regexp.MustCompile(pattern).MatchString(file.Content) {
					t.Fatalf("generated content does not match %q:\n%s", pattern, file.Content)
				}
			}
			for _, unwanted := range []string{"flags.newClient()", "GetWithHeaders(", "PostWithParamsAndHeaders("} {
				if strings.Contains(file.Content, unwanted) {
					t.Fatalf("generated content contains legacy request assembly %q:\n%s", unwanted, file.Content)
				}
			}
		})
	}
}

func TestGenerateAllReconcilesGeneratedFiles(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		dryRun bool
	}{
		{name: "writes changes", dryRun: false},
		{name: "dry run reports without writing", dryRun: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := t.TempDir()
			cliDir := filepath.Join(repo, "internal", "cli")
			if err := os.MkdirAll(cliDir, 0o755); err != nil {
				t.Fatalf("mkdir internal/cli: %v", err)
			}
			specPath := writeGenerationSpec(t)
			operations, err := apisync.LoadSpec(specPath)
			if err != nil {
				t.Fatalf("LoadSpec: %v", err)
			}
			surfaces, unsupported, err := apisync.DeriveSurfaces(specPath)
			if err != nil {
				t.Fatalf("DeriveSurfaces: %v", err)
			}
			if len(unsupported) != 1 || unsupported[0].Operation.Key != "POST /v1/uploads" {
				t.Fatalf("unsupported = %#v, want POST /v1/uploads", unsupported)
			}

			currentOp := operationByKey(t, operations, "GET /v1/widgets/{id}")
			currentSurface := surfaceByKey(t, surfaces, currentOp.Key)
			currentFile, err := apisync.GenerateEndpointFile(currentSurface, currentOp, cliDir)
			if err != nil {
				t.Fatalf("GenerateEndpointFile current: %v", err)
			}
			writeGeneratedFile(t, currentFile)

			staleOp := apisync.Operation{
				Key:         "DELETE /v1/retired-widgets/{id}",
				OperationID: "DeleteRetiredWidget",
				Endpoint:    "retired-widgets.delete",
				Method:      "DELETE",
				Path:        "/v1/retired-widgets/{id}",
				PathParameters: []apisync.Parameter{
					{Name: "id", In: "path", Required: true, SchemaType: "string"},
				},
			}
			staleSurface := surface.Surface{
				Endpoint:    staleOp.Endpoint,
				OperationID: staleOp.OperationID,
				Method:      staleOp.Method,
				Path:        staleOp.Path,
				PathParams:  []string{"id"},
			}
			staleFile, err := apisync.GenerateEndpointFile(staleSurface, staleOp, cliDir)
			if err != nil {
				t.Fatalf("GenerateEndpointFile stale: %v", err)
			}
			writeGeneratedFile(t, staleFile)
			newlyUnsupportedOp := apisync.Operation{
				Key:         "POST /v1/uploads",
				OperationID: "CreateUpload",
				Endpoint:    "uploads.create",
				Method:      "POST",
				Path:        "/v1/uploads",
			}
			newlyUnsupportedSurface := surface.Surface{
				Endpoint:    newlyUnsupportedOp.Endpoint,
				OperationID: newlyUnsupportedOp.OperationID,
				Method:      newlyUnsupportedOp.Method,
				Path:        newlyUnsupportedOp.Path,
			}
			newlyUnsupportedFile, err := apisync.GenerateEndpointFile(newlyUnsupportedSurface, newlyUnsupportedOp, cliDir)
			if err != nil {
				t.Fatalf("GenerateEndpointFile newly unsupported: %v", err)
			}
			writeGeneratedFile(t, newlyUnsupportedFile)

			handAuthoredPath := filepath.Join(cliDir, "bridge_create_tan.go")
			handAuthored := `package cli

var bridgeCreateTANAnnotations = map[string]string{
	"straddle:endpoint": "bridge.create-tan",
	"straddle:operation-id": "CreateTAN",
	"straddle:method": "POST",
	"straddle:path": "/v1/bridge/tan",
}
`
			if err := os.WriteFile(handAuthoredPath, []byte(handAuthored), 0o644); err != nil {
				t.Fatalf("write hand-authored command: %v", err)
			}

			result, err := apisync.GenerateAll(specPath, repo, tc.dryRun)
			if err != nil {
				t.Fatalf("GenerateAll: %v", err)
			}
			newPath := filepath.Join(cliDir, "widgets_create.go")
			if want := []string{newPath}; !reflect.DeepEqual(result.Generated, want) {
				t.Fatalf("Generated = %#v, want %#v", result.Generated, want)
			}
			if want := []string{staleFile.Path, newlyUnsupportedFile.Path}; !reflect.DeepEqual(result.Deleted, want) {
				t.Fatalf("Deleted = %#v, want %#v", result.Deleted, want)
			}
			if want := []string{currentFile.Path}; !reflect.DeepEqual(result.Unchanged, want) {
				t.Fatalf("Unchanged = %#v, want %#v", result.Unchanged, want)
			}
			if len(result.UnsupportedOperations) != 1 || result.UnsupportedOperations[0].Operation.Key != newlyUnsupportedOp.Key {
				t.Fatalf("UnsupportedOperations = %#v, want %s", result.UnsupportedOperations, newlyUnsupportedOp.Key)
			}
			if result.DryRun != tc.dryRun {
				t.Fatalf("DryRun = %t, want %t", result.DryRun, tc.dryRun)
			}
			if _, err := os.Stat(handAuthoredPath); err != nil {
				t.Fatalf("hand-authored annotated command was changed: %v", err)
			}

			if tc.dryRun {
				if _, err := os.Stat(newPath); !os.IsNotExist(err) {
					t.Fatalf("dry run wrote %s, stat error = %v", newPath, err)
				}
				if _, err := os.Stat(staleFile.Path); err != nil {
					t.Fatalf("dry run deleted %s: %v", staleFile.Path, err)
				}
				if _, err := os.Stat(newlyUnsupportedFile.Path); err != nil {
					t.Fatalf("dry run deleted newly unsupported %s: %v", newlyUnsupportedFile.Path, err)
				}
				return
			}
			if _, err := os.Stat(newPath); err != nil {
				t.Fatalf("generated file missing: %v", err)
			}
			if _, err := os.Stat(staleFile.Path); !os.IsNotExist(err) {
				t.Fatalf("stale generated file still exists, stat error = %v", err)
			}
			if _, err := os.Stat(newlyUnsupportedFile.Path); !os.IsNotExist(err) {
				t.Fatalf("newly unsupported generated file still exists, stat error = %v", err)
			}
		})
	}
}

func writeGenerationSpec(t *testing.T) string {
	t.Helper()
	return writeSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets/{id}": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "GetWidget",
					"summary": "Get widget",
					"parameters": [
						{"name": "id", "in": "path", "required": true, "schema": {"type": "string"}}
					]
				}
			},
			"/v1/widgets": {
				"post": {
					"tags": ["Widgets"],
					"operationId": "CreateWidget",
					"summary": "Create widget"
				}
			},
			"/v1/uploads": {
				"post": {
					"tags": ["Uploads"],
					"operationId": "CreateUpload",
					"summary": "Upload a file",
					"requestBody": {"required": true, "content": {"multipart/form-data": {}}}
				}
			}
		}
	}`)
}

func operationByKey(t *testing.T, operations []apisync.Operation, key string) apisync.Operation {
	t.Helper()
	for _, op := range operations {
		if op.Key == key {
			return op
		}
	}
	t.Fatalf("operation %s not found", key)
	return apisync.Operation{}
}

func surfaceByKey(t *testing.T, surfaces []surface.Surface, key string) surface.Surface {
	t.Helper()
	for _, commandSurface := range surfaces {
		if apisync.OperationKey(commandSurface.Method, commandSurface.Path) == key {
			return commandSurface
		}
	}
	t.Fatalf("surface %s not found", key)
	return surface.Surface{}
}

func writeGeneratedFile(t *testing.T, file apisync.GeneratedFile) {
	t.Helper()
	if err := os.WriteFile(file.Path, []byte(file.Content), 0o644); err != nil {
		t.Fatalf("write generated file: %v", err)
	}
}
