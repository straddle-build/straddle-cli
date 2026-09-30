// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/straddle-build/straddle-cli/internal/apisync"
	"github.com/straddle-build/straddle-cli/internal/surface"
)

type capturedSurfaceRequest struct {
	method   string
	path     string
	rawQuery string
	query    url.Values
	headers  http.Header
	body     map[string]any
	err      error
}

func TestBindSurfaceCapturesRequests(t *testing.T) {
	cases := []struct {
		name        string
		method      string
		required    bool
		args        []string
		stdin       string
		wantErr     string
		wantRequest bool
		assert      func(*testing.T, capturedSurfaceRequest)
	}{
		{
			name:     "all flags set",
			required: true,
			args: []string{
				"widget-1",
				"--name", "example",
				"--limit", "12",
				"--active=false",
				"--status", "a",
				"--status", "b",
				"--mode", "fast",
				"--request-id", "req_123",
				"--amount", "125",
				"--config-auto-hold=false",
				"--metadata", `{"source":"test","rank":2}`,
				"--paykey", "pk_123",
			},
			wantRequest: true,
			assert: func(t *testing.T, got capturedSurfaceRequest) {
				t.Helper()
				if got.method != http.MethodPost {
					t.Fatalf("method = %q, want POST", got.method)
				}
				if got.path != "/v1/widgets/widget-1" {
					t.Fatalf("path = %q, want /v1/widgets/widget-1", got.path)
				}
				if !strings.Contains(got.rawQuery, "status=a&status=b") {
					t.Fatalf("query = %q, want repeated status keys", got.rawQuery)
				}
				wantQuery := url.Values{
					"active": {"false"},
					"limit":  {"12"},
					"mode":   {"fast"},
					"name":   {"example"},
					"status": {"a", "b"},
				}
				if !reflect.DeepEqual(got.query, wantQuery) {
					t.Fatalf("query = %#v, want %#v", got.query, wantQuery)
				}
				if got.headers.Get("Request-Id") != "req_123" {
					t.Fatalf("Request-Id = %q, want req_123", got.headers.Get("Request-Id"))
				}
				if got.body["amount"] != float64(125) {
					t.Fatalf("amount = %#v, want 125", got.body["amount"])
				}
				config, ok := got.body["config"].(map[string]any)
				if !ok {
					t.Fatalf("config = %#v, want object", got.body["config"])
				}
				if autoHold, ok := config["auto_hold"].(bool); !ok || autoHold {
					t.Fatalf("config.auto_hold = %#v, want false", config["auto_hold"])
				}
				metadata, ok := got.body["metadata"].(map[string]any)
				if !ok {
					t.Fatalf("metadata = %#v, want object", got.body["metadata"])
				}
				if metadata["source"] != "test" || metadata["rank"] != float64(2) {
					t.Fatalf("metadata = %#v, want source and rank", metadata)
				}
				if got.body["paykey"] != "pk_123" {
					t.Fatalf("paykey = %#v, want pk_123", got.body["paykey"])
				}
			},
		},
		{
			name:        "nothing set",
			args:        []string{"widget-1"},
			wantRequest: true,
			assert: func(t *testing.T, got capturedSurfaceRequest) {
				t.Helper()
				if got.rawQuery != "" {
					t.Fatalf("query = %q, want empty", got.rawQuery)
				}
				if len(got.body) != 0 {
					t.Fatalf("body = %#v, want empty object", got.body)
				}
			},
		},
		{
			name:        "explicit zero integer",
			args:        []string{"widget-1", "--amount", "0"},
			wantRequest: true,
			assert: func(t *testing.T, got capturedSurfaceRequest) {
				t.Helper()
				if amount, ok := got.body["amount"].(float64); !ok || amount != 0 {
					t.Fatalf("amount = %#v, want 0", got.body["amount"])
				}
			},
		},
		{
			name:        "explicit false boolean",
			args:        []string{"widget-1", "--config-auto-hold=false"},
			wantRequest: true,
			assert: func(t *testing.T, got capturedSurfaceRequest) {
				t.Helper()
				config, ok := got.body["config"].(map[string]any)
				if !ok {
					t.Fatalf("config = %#v, want object", got.body["config"])
				}
				if autoHold, ok := config["auto_hold"].(bool); !ok || autoHold {
					t.Fatalf("config.auto_hold = %#v, want false", config["auto_hold"])
				}
			},
		},
		{
			name:        "explicit empty string",
			args:        []string{"widget-1", "--paykey="},
			wantRequest: true,
			assert: func(t *testing.T, got capturedSurfaceRequest) {
				t.Helper()
				paykey, ok := got.body["paykey"]
				if !ok || paykey != "" {
					t.Fatalf("paykey = %#v, want empty string", paykey)
				}
			},
		},
		{
			name:     "stdin overrides body flags",
			required: true,
			args: []string{
				"widget-1",
				"--amount", "999",
				"--metadata", `{"source":"flag"}`,
				"--paykey", "pk_flag",
				"--stdin",
			},
			stdin:       `{"paykey":"pk_stdin","nested":{"value":true}}`,
			wantRequest: true,
			assert: func(t *testing.T, got capturedSurfaceRequest) {
				t.Helper()
				wantBody := map[string]any{
					"nested": map[string]any{"value": true},
					"paykey": "pk_stdin",
				}
				if !reflect.DeepEqual(got.body, wantBody) {
					t.Fatalf("body = %#v, want %#v", got.body, wantBody)
				}
			},
		},
		{
			name:     "required flag missing",
			required: true,
			args:     []string{"widget-1"},
			wantErr:  `required flag "paykey" not set`,
		},
		{
			name:     "enum violation",
			required: true,
			args:     []string{"widget-1", "--paykey", "pk_123", "--mode", "turbo"},
			wantErr:  `invalid value "turbo" for --mode (allowed: fast, safe)`,
		},
		{
			name:     "malformed JSON flag",
			required: true,
			args:     []string{"widget-1", "--paykey", "pk_123", "--metadata", "{"},
			wantErr:  `--metadata expects a JSON object, for example --metadata '{"key":"value"}'`,
		},
		{
			name:        "DELETE with body",
			method:      http.MethodDelete,
			required:    true,
			args:        []string{"widget-1", "--amount", "25", "--paykey", "pk_delete"},
			wantRequest: true,
			assert: func(t *testing.T, got capturedSurfaceRequest) {
				t.Helper()
				if got.method != http.MethodDelete {
					t.Fatalf("method = %q, want DELETE", got.method)
				}
				if got.body["amount"] != float64(25) || got.body["paykey"] != "pk_delete" {
					t.Fatalf("body = %#v, want DELETE body fields", got.body)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			method := tc.method
			if method == "" {
				method = http.MethodPost
			}
			s := testSurface(method, tc.required)
			captured := make(chan capturedSurfaceRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got := capturedSurfaceRequest{
					method:   r.Method,
					path:     r.URL.Path,
					rawQuery: r.URL.RawQuery,
					query:    r.URL.Query(),
					headers:  r.Header.Clone(),
				}
				body, err := io.ReadAll(r.Body)
				if err == nil && len(body) > 0 {
					err = json.Unmarshal(body, &got.body)
				}
				got.err = err
				captured <- got
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"widget-1"}`))
			}))
			defer server.Close()
			isolateSurfaceConfig(t, server.URL)

			_, _, err := runSurfaceCommand(t, s, tc.args, tc.stdin)
			if tc.wantErr != "" {
				if err == nil || err.Error() != tc.wantErr {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				select {
				case got := <-captured:
					t.Fatalf("unexpected request: %#v", got)
				default:
				}
				return
			}
			if err != nil {
				t.Fatalf("execute returned error: %v", err)
			}
			if !tc.wantRequest {
				select {
				case got := <-captured:
					t.Fatalf("unexpected request: %#v", got)
				default:
				}
				return
			}
			select {
			case got := <-captured:
				if got.err != nil {
					t.Fatalf("capturing request: %v", got.err)
				}
				if tc.assert != nil {
					tc.assert(t, got)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for request")
			}
		})
	}
}

func TestGeneratedCreateRequiredFlagsInBothModes(t *testing.T) {
	var requests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"created"}}`))
	}))
	defer server.Close()
	isolateSurfaceConfig(t, server.URL)
	t.Setenv("HOME", t.TempDir())

	for _, resource := range []string{"charges", "organizations"} {
		complete := goldenInvocations[resource+".create"]
		for omitted, argument := range complete {
			name := strings.SplitN(strings.TrimPrefix(argument, "--"), "=", 2)[0]
			for _, mode := range []string{"--json", "--agent"} {
				for _, dryRun := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/dry-run=%t", resource, name, mode, dryRun), func(t *testing.T) {
						args := []string{mode, "--no-cache", resource, "create"}
						if dryRun {
							args = append(args, "--dry-run")
						}
						args = append(args, complete[:omitted]...)
						args = append(args, complete[omitted+1:]...)
						_, _, err := runRootForAPITest(t, args, "")
						if err == nil || !strings.Contains(err.Error(), fmt.Sprintf(`required flag %q not set`, name)) {
							t.Errorf("error = %v, want missing required flag %q", err, name)
						}
						if got := requests.Load(); got != 0 {
							t.Fatalf("incomplete create sent %d HTTP requests", got)
						}
					})
				}
			}
		}
	}
}

func TestGeneratedBodyArrayFlagTakesJSONArray(t *testing.T) {
	var requests atomic.Int64
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		body = nil
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"created"}}`))
	}))
	defer server.Close()
	isolateSurfaceConfig(t, server.URL)
	t.Setenv("HOME", t.TempDir())
	create := append([]string{"--json", "--no-cache", "linked-bank-accounts", "create"}, goldenInvocations["linked-bank-accounts.create"]...)

	t.Run("JSON array becomes the body array", func(t *testing.T) {
		requests.Store(0)
		if _, _, err := runRootForAPITest(t, append(create, `--purposes=["charges","payouts"]`), ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requests.Load() != 1 || !reflect.DeepEqual(body["purposes"], []any{"charges", "payouts"}) {
			t.Fatalf("requests = %d, purposes = %#v, want one request with [charges payouts]", requests.Load(), body["purposes"])
		}
	})

	for _, tc := range []struct{ name, value, wantErr string }{
		{"comma list", "charges,payouts", "JSON array"},
		{"bare value", "charges", "JSON array"},
		{"wrong item type", "[1]", "JSON array"},
		{"null", "null", "JSON array"},
		{"unknown item", `["charges","bogus"]`, `"bogus"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests.Store(0)
			_, _, err := runRootForAPITest(t, append(create, "--purposes="+tc.value), "")
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want error mentioning %s", err, tc.wantErr)
			}
			if requests.Load() != 0 {
				t.Fatalf("invalid --purposes sent %d HTTP requests", requests.Load())
			}
			if tc.wantErr != "JSON array" {
				return
			}
			example, ok := copyableFlagExample(err.Error(), "purposes")
			if !ok {
				t.Fatalf("error %q has no copyable --purposes example", err)
			}
			if _, _, err := runRootForAPITest(t, append(create, "--purposes="+example), ""); err != nil || requests.Load() != 1 {
				t.Fatalf("suggested --purposes %s: error = %v, requests = %d, want accepted", example, err, requests.Load())
			}
		})
	}
}

// Every generated body array flag must keep its JSON shape, accepted values,
// and example in help, even when an endpoint overlay rewrites flag usage.
func TestGeneratedBodyArrayFlagHelpShowsShape(t *testing.T) {
	commands := map[string]*cobra.Command{}
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if endpoint := cmd.Annotations["straddle:endpoint"]; endpoint != "" {
			commands[endpoint] = cmd
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(RootCmd())

	checked := 0
	for _, s := range registeredSurfaces() {
		for _, definition := range s.Flags {
			if !definition.Array || definition.In != surface.InBody {
				continue
			}
			flag := commands[s.Endpoint].Flags().Lookup(definition.Name)
			example, ok := copyableFlagExample(flag.Usage, definition.Name)
			var items []any
			if !strings.Contains(flag.Usage, "JSON array") || !ok || json.Unmarshal([]byte(example), &items) != nil || len(items) == 0 {
				t.Errorf("%s --%s usage %q, want JSON array shape and a copyable non-empty array example", s.Endpoint, definition.Name, flag.Usage)
			}
			for _, value := range definition.Enum {
				if !strings.Contains(flag.Usage, value) {
					t.Errorf("%s --%s usage %q omits accepted value %q", s.Endpoint, definition.Name, flag.Usage, value)
				}
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no generated body array flags found")
	}
}

func TestGeneratedKindJSONObjectFlagTakesJSONObject(t *testing.T) {
	var requests atomic.Int64
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		body = nil
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"created"}}`))
	}))
	defer server.Close()
	isolateSurfaceConfig(t, server.URL)
	t.Setenv("HOME", t.TempDir())
	create := append([]string{"--json", "--no-cache", "customers", "create"}, goldenInvocations["customers.create"]...)

	t.Run("JSON object becomes body object", func(t *testing.T) {
		requests.Store(0)
		if _, _, err := runRootForAPITest(t, append(create, `--metadata={"key":"value","rank":2}`), ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantMetadata := map[string]any{"key": "value", "rank": float64(2)}
		if requests.Load() != 1 || !reflect.DeepEqual(body["metadata"], wantMetadata) {
			t.Fatalf("requests = %d, metadata = %#v, want one request with %#v", requests.Load(), body["metadata"], wantMetadata)
		}
	})

	t.Run("null is preserved for nullable contract", func(t *testing.T) {
		requests.Store(0)
		if _, _, err := runRootForAPITest(t, append(create, `--metadata=null`), ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requests.Load() != 1 || body["metadata"] != nil {
			t.Fatalf("requests = %d, metadata = %#v, want one request with nil metadata", requests.Load(), body["metadata"])
		}
	})

	t.Run("compliance-profile valid object", func(t *testing.T) {
		requests.Store(0)
		if _, _, err := runRootForAPITest(t, append(create, `--compliance-profile={"ein":"12-3456789","legal_business_name":"Acme Corp LLC"}`), ""); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantProfile := map[string]any{"ein": "12-3456789", "legal_business_name": "Acme Corp LLC"}
		if requests.Load() != 1 || !reflect.DeepEqual(body["compliance_profile"], wantProfile) {
			t.Fatalf("requests = %d, compliance_profile = %#v, want one request with %#v", requests.Load(), body["compliance_profile"], wantProfile)
		}
	})

	t.Run("dry-run previews valid body without HTTP", func(t *testing.T) {
		requests.Store(0)
		stdout, _, err := runRootForAPITest(t, append(create, "--dry-run", `--metadata={"key":"value"}`), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if requests.Load() != 0 {
			t.Fatalf("dry-run sent %d HTTP requests, want 0", requests.Load())
		}
		if !strings.Contains(stdout, `"dry_run": true`) {
			t.Fatalf("dry-run stdout = %q, want preview with metadata object", stdout)
		}
	})

	t.Run("dry-run rejects invalid object locally without HTTP", func(t *testing.T) {
		requests.Store(0)
		_, _, err := runRootForAPITest(t, append(create, "--dry-run", `--metadata=not-json`), "")
		if err == nil || !strings.Contains(err.Error(), "JSON object") {
			t.Fatalf("error = %v, want JSON object usage error", err)
		}
		if requests.Load() != 0 {
			t.Fatalf("dry-run sent %d HTTP requests, want 0", requests.Load())
		}
	})

	for _, tc := range []struct{ flagName, name, value, wantErr string }{
		{"metadata", "comma list", "key=value", "JSON object"},
		{"metadata", "bare value", "foobar", "JSON object"},
		{"metadata", "array value", `["a","b"]`, "JSON object"},
		{"metadata", "primitive string", `"text"`, "JSON object"},
		{"metadata", "primitive int", "42", "JSON object"},
		{"metadata", "primitive bool", "true", "JSON object"},
		{"metadata", "broken JSON", "{bad", "JSON object"},
		{"compliance-profile", "bare value", "foobar", "JSON object"},
		{"compliance-profile", "array value", `[1,2]`, "JSON object"},
		{"compliance-profile", "broken JSON", "{bad", "JSON object"},
	} {
		t.Run(tc.flagName+" "+tc.name, func(t *testing.T) {
			requests.Store(0)
			_, _, err := runRootForAPITest(t, append(create, "--"+tc.flagName+"="+tc.value), "")
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want error mentioning %s", err, tc.wantErr)
			}
			if requests.Load() != 0 {
				t.Fatalf("invalid --%s sent %d HTTP requests", tc.flagName, requests.Load())
			}
			example, ok := copyableFlagExample(err.Error(), tc.flagName)
			if !ok {
				t.Fatalf("error %q has no copyable --%s example", err, tc.flagName)
			}
			requests.Store(0)
			if _, _, err := runRootForAPITest(t, append(create, "--"+tc.flagName+"="+example), ""); err != nil || requests.Load() != 1 {
				t.Fatalf("suggested --%s %s: error = %v, requests = %d, want accepted", tc.flagName, example, err, requests.Load())
			}
		})
	}
}

func TestGeneratedKindJSONObjectFlagHelpShowsShape(t *testing.T) {
	commands := map[string]*cobra.Command{}
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		if endpoint := cmd.Annotations["straddle:endpoint"]; endpoint != "" {
			commands[endpoint] = cmd
		}
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(RootCmd())

	checked := 0
	for _, s := range registeredSurfaces() {
		for _, definition := range s.Flags {
			if definition.Kind != surface.KindJSON || !definition.Object {
				continue
			}
			flag := commands[s.Endpoint].Flags().Lookup(definition.Name)
			example, ok := copyableFlagExample(flag.Usage, definition.Name)
			var obj map[string]any
			if (!strings.Contains(flag.Usage, "JSON object") && !strings.Contains(flag.Usage, "object")) || !ok || json.Unmarshal([]byte(example), &obj) != nil || len(obj) == 0 {
				t.Errorf("%s --%s usage %q, want JSON object shape and a copyable non-empty object example", s.Endpoint, definition.Name, flag.Usage)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no generated KindJSON object flags found")
	}
}

// Contract-derived KindJSON flags cover raw arrays and mixed unions as well as
// objects; only schemas that are objects may reject non-object JSON.
func TestDerivedKindJSONFlagsKeepSchemaShape(t *testing.T) {
	specPath := filepath.Join(t.TempDir(), "spec.yaml")
	if err := os.WriteFile(specPath, []byte(`openapi: 3.1.0
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
                  type: [object, "null"]
                  additionalProperties:
                    type: string
                owner:
                  anyOf:
                    - $ref: "#/components/schemas/Profile"
                    - type: "null"
                labels:
                  anyOf:
                    - properties:
                        name:
                          type: string
                    - additionalProperties:
                        type: string
                rules:
                  type: array
                  items:
                    type: object
                    properties:
                      name:
                        type: string
                selector:
                  oneOf:
                    - type: string
                    - type: integer
                tree:
                  anyOf:
                    - $ref: "#/components/schemas/Node"
components:
  schemas:
    Profile:
      type: object
      properties:
        ein:
          type: string
    Node:
      anyOf:
        - $ref: "#/components/schemas/Node"
        - type: object
`), 0o600); err != nil {
		t.Fatal(err)
	}
	surfaces, unsupported, err := apisync.DeriveSurfaces(specPath)
	if err != nil || len(unsupported) != 0 || len(surfaces) != 1 {
		t.Fatalf("DeriveSurfaces = %#v, %#v, %v; want one supported surface", surfaces, unsupported, err)
	}
	derived := surfaces[0]

	var requests atomic.Int64
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		body = nil
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":"created"}}`))
	}))
	defer server.Close()
	isolateSurfaceConfig(t, server.URL)
	t.Setenv("HOME", t.TempDir())

	for _, tc := range []struct{ flag, key, value string }{
		{"rules", "rules", `[{"name":"daily"}]`},
		{"selector", "selector", `7`},
		{"selector", "selector", `"alpha"`},
		{"metadata", "metadata", `{"key":"value"}`},
		{"metadata", "metadata", `null`},
		{"owner", "owner", `{"ein":"12-3456789"}`},
		{"owner", "owner", `null`},
		{"labels", "labels", `"text"`},
		{"labels", "labels", `{"name":"a"}`},
		{"tree", "tree", `{"child":{}}`},
	} {
		t.Run("accepts "+tc.flag+"="+tc.value, func(t *testing.T) {
			var want any
			_ = json.Unmarshal([]byte(tc.value), &want)
			requests.Store(0)
			if _, _, err := runSurfaceCommand(t, derived, []string{"--dry-run", "--" + tc.flag + "=" + tc.value}, ""); err != nil || requests.Load() != 0 {
				t.Fatalf("dry-run error = %v, requests = %d; want accepted without HTTP", err, requests.Load())
			}
			if _, _, err := runSurfaceCommand(t, derived, []string{"--" + tc.flag + "=" + tc.value}, ""); err != nil || requests.Load() != 1 || !reflect.DeepEqual(body[tc.key], want) {
				t.Fatalf("live error = %v, requests = %d, body[%s] = %#v; want one request with %#v", err, requests.Load(), tc.key, body[tc.key], want)
			}
		})
	}

	for _, tc := range []struct{ flag, value string }{
		{"metadata", `["a"]`},
		{"owner", `"text"`},
		{"rules", `not-json`},
		{"selector", `{bad`},
	} {
		t.Run("rejects "+tc.flag+"="+tc.value, func(t *testing.T) {
			requests.Store(0)
			_, _, err := runSurfaceCommand(t, derived, []string{"--" + tc.flag + "=" + tc.value}, "")
			if ExitCode(err) != 2 || requests.Load() != 0 {
				t.Fatalf("error = %v (exit %d), requests = %d; want usage error without HTTP", err, ExitCode(err), requests.Load())
			}
			example, ok := copyableFlagExample(err.Error(), tc.flag)
			if !ok {
				return
			}
			if _, _, err := runSurfaceCommand(t, derived, []string{"--" + tc.flag + "=" + example}, ""); err != nil || requests.Load() != 1 {
				t.Fatalf("suggested --%s %s: error = %v, requests = %d; want accepted", tc.flag, example, err, requests.Load())
			}
		})
	}
}

func copyableFlagExample(text, name string) (string, bool) {
	_, rest, found := strings.Cut(text, "--"+name+" '")
	if !found {
		return "", false
	}
	example, _, found := strings.Cut(rest, "'")
	return example, found
}

func TestBindSurfaceRequiredWithDefault(t *testing.T) {
	surfaceWithRequiredDefault := surface.Surface{
		Endpoint:    "widgets.update",
		OperationID: "updateWidget",
		Method:      http.MethodPut,
		Path:        "/v1/widgets/{id}",
		PathParams:  []string{"id"},
		HasBody:     true,
		Flags: []surface.Flag{
			{Name: "name", In: surface.InBody, Key: "/name", Kind: surface.KindString, Required: true},
			{Name: "status", In: surface.InBody, Key: "/status", Kind: surface.KindString, Required: true, Default: "verified", Enum: []string{"pending", "review", "verified"}},
		},
	}

	run := func(t *testing.T, args []string, stdin string) (string, string, *capturedSurfaceRequest, error) {
		t.Helper()
		captured := make(chan capturedSurfaceRequest, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := capturedSurfaceRequest{method: r.Method, path: r.URL.Path}
			body, readErr := io.ReadAll(r.Body)
			if readErr == nil && len(body) > 0 {
				_ = json.Unmarshal(body, &got.body)
			}
			got.err = readErr
			captured <- got
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"widget-1"}`))
		}))
		defer server.Close()
		isolateSurfaceConfig(t, server.URL)
		stdout, stderr, err := runSurfaceCommand(t, surfaceWithRequiredDefault, args, stdin)
		var req *capturedSurfaceRequest
		select {
		case got := <-captured:
			req = &got
		default:
		}
		return stdout, stderr, req, err
	}

	t.Run("omitted status errors and sends nothing", func(t *testing.T) {
		_, _, req, err := run(t, []string{"widget-1", "--name", "example"}, "")
		if err == nil || !strings.Contains(err.Error(), `required flag "status" not set`) {
			t.Fatalf("error = %v, want %q", err, `required flag "status" not set`)
		}
		if req != nil {
			t.Fatalf("omitted --status reached the API; body=%#v", req.body)
		}
	})

	t.Run("explicit non-default value is sent", func(t *testing.T) {
		_, _, req, err := run(t, []string{"widget-1", "--name", "example", "--status", "review"}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req == nil {
			t.Fatal("expected a request to reach the server")
		}
		if req.body["status"] != "review" {
			t.Fatalf("status = %#v, want \"review\"", req.body["status"])
		}
	})

	t.Run("explicit default value is sent", func(t *testing.T) {
		_, _, req, err := run(t, []string{"widget-1", "--name", "example", "--status", "verified"}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if req == nil || req.body["status"] != "verified" {
			t.Fatalf("status = %#v, want \"verified\"", req.body)
		}
	})

	t.Run("invalid enum still rejected when supplied", func(t *testing.T) {
		_, _, req, err := run(t, []string{"widget-1", "--name", "example", "--status", "bogus"}, "")
		if err == nil || !strings.Contains(err.Error(), `invalid value "bogus" for --status`) {
			t.Fatalf("error = %v, want enum violation", err)
		}
		if req != nil {
			t.Fatalf("invalid enum reached the API; body=%#v", req.body)
		}
	})

	t.Run("stdin body overrides the required+default flag without error", func(t *testing.T) {
		_, _, req, err := run(t, []string{"widget-1", "--stdin"}, `{"status":"review"}`)
		if err != nil {
			t.Fatalf("stdin with explicit body status returned error: %v", err)
		}
		if req == nil {
			t.Fatal("expected a request to reach the server")
		}
		if req.body["status"] != "review" {
			t.Fatalf("status = %#v, want \"review\" from stdin", req.body["status"])
		}
	})

	t.Run("help does not advertise a default that is never applied", func(t *testing.T) {
		stdout, _, _, err := run(t, []string{"--help"}, "")
		if err != nil {
			t.Fatalf("help returned error: %v", err)
		}
		if strings.Contains(stdout, `(default "verified")`) {
			t.Fatalf("help for a required flag advertises a default the CLI rejects when omitted:\n%s", stdout)
		}
	})
}

func testSurface(method string, requiredPaykey bool) surface.Surface {
	return surface.Surface{
		Endpoint:    "widgets.update",
		OperationID: "updateWidget",
		Method:      method,
		Path:        "/v1/widgets/{id}",
		PathParams:  []string{"id"},
		HasBody:     true,
		Flags: []surface.Flag{
			{Name: "active", In: surface.InQuery, Key: "active", Kind: surface.KindBoolean},
			{Name: "limit", In: surface.InQuery, Key: "limit", Kind: surface.KindInteger},
			{Name: "mode", In: surface.InQuery, Key: "mode", Kind: surface.KindString, Enum: []string{"fast", "safe"}},
			{Name: "name", In: surface.InQuery, Key: "name", Kind: surface.KindString},
			{Name: "status", In: surface.InQuery, Key: "status", Kind: surface.KindString, Array: true, Style: surface.StyleForm, Explode: true},
			{Name: "request-id", In: surface.InHeader, Key: "Request-Id", Kind: surface.KindString},
			{Name: "amount", In: surface.InBody, Key: "/amount", Kind: surface.KindInteger},
			{Name: "config-auto-hold", In: surface.InBody, Key: "/config/auto_hold", Kind: surface.KindBoolean},
			{Name: "metadata", In: surface.InBody, Key: "/metadata", Kind: surface.KindJSON, Object: true},
			{Name: "paykey", In: surface.InBody, Key: "/paykey", Kind: surface.KindString, Required: requiredPaykey},
		},
	}
}

func runSurfaceCommand(t *testing.T, s surface.Surface, args []string, stdin string) (string, string, error) {
	t.Helper()
	flags := &rootFlags{}
	root := &cobra.Command{Use: "straddle", SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().BoolVar(&flags.dryRun, "dry-run", false, "Show request without sending")
	cmd := &cobra.Command{
		Use: "fixture <id>",
		Annotations: map[string]string{
			"straddle:endpoint":     s.Endpoint,
			"straddle:operation-id": s.OperationID,
			"straddle:method":       s.Method,
			"straddle:path":         s.Path,
		},
	}
	bind := bindSurface(cmd, s)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		req, err := bind(args)
		if err != nil {
			return err
		}
		return executeSurface(cmd, flags, s, req)
	}
	root.AddCommand(cmd)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(append([]string{"fixture"}, args...))
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

func isolateSurfaceConfig(t *testing.T, baseURL string) {
	t.Helper()
	configDir := t.TempDir()
	t.Setenv("STRADDLE_CONFIG", filepath.Join(configDir, "config.toml"))
	t.Setenv("STRADDLE_PLATFORM_CONFIG", filepath.Join(configDir, "platform.toml"))
	t.Setenv("STRADDLE_API_KEY", "test_key")
	t.Setenv("STRADDLE_BASE_URL", baseURL)
	t.Setenv("STRADDLE_VERIFY", "")
	t.Setenv("STRADDLE_VERIFY_LIVE_HTTP", "")
}

func TestValidateSurfaceEnum(t *testing.T) {
	definitionEnum := []string{"active", "rejected"}
	cases := []struct {
		name          string
		annotationSet bool
		annotation    []string
		values        []string
		definition    []string
		wantErr       string
	}{
		{
			name:       "no annotation uses definition enum invalid rejected",
			values:     []string{"invalid_garbage"},
			definition: definitionEnum,
			wantErr:    `invalid value "invalid_garbage" for --status (allowed: active, rejected)`,
		},
		{
			name:       "no annotation uses definition enum valid accepted",
			values:     []string{"active"},
			definition: definitionEnum,
		},
		{
			name:          "non-empty annotation overrides definition invalid rejected",
			annotationSet: true,
			annotation:    []string{"on", "off"},
			values:        []string{"maybe"},
			definition:    definitionEnum,
			wantErr:       `invalid value "maybe" for --status (allowed: on, off)`,
		},
		{
			name:          "non-empty annotation overrides definition valid accepted",
			annotationSet: true,
			annotation:    []string{"on", "off"},
			values:        []string{"off"},
			definition:    definitionEnum,
		},
		{
			name:          "empty annotation falls back to definition enum invalid rejected",
			annotationSet: true,
			annotation:    nil,
			values:        []string{"invalid_garbage"},
			definition:    definitionEnum,
			wantErr:       `invalid value "invalid_garbage" for --status (allowed: active, rejected)`,
		},
		{
			name:       "empty definition and no annotation skips validation",
			values:     []string{"anything_goes"},
			definition: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			definition := surface.Flag{Name: "status", Kind: surface.KindString, Enum: tc.definition}
			cmd := &cobra.Command{Use: "test"}
			cmd.Flags().String("status", "", "")
			if tc.annotationSet {
				flag := cmd.Flags().Lookup("status")
				if flag.Annotations == nil {
					flag.Annotations = map[string][]string{}
				}
				flag.Annotations["straddle:enum"] = tc.annotation
			}
			err := validateSurfaceEnum(cmd, definition, tc.values)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

var testOverlayEnumSeq atomic.Int64

func TestOverlayEnumSetWithoutEnumPanics(t *testing.T) {
	endpoint := fmt.Sprintf("test.enum-panic.%d", testOverlayEnumSeq.Add(1))
	registerCommandOverlay(endpoint, commandOverlay{
		flags: []flagOverlay{
			{name: "status", usage: "Status", enumSet: true},
		},
	})
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("status", "", "Status")
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when enumSet is set without enum values")
		}
		if msg := fmt.Sprint(r); !strings.Contains(msg, "enumSet") || !strings.Contains(msg, "enum") {
			t.Fatalf("unexpected panic message: %v", r)
		}
	}()
	applyOverlay(endpoint, cmd)
}

func TestCustomerReviewRequiredStatusFromProfile(t *testing.T) {
	cases := []struct {
		name       string
		profile    map[string]string
		args       []string
		wantStatus string
		wantError  bool
	}{
		{
			name:      "profile omits required default",
			profile:   map[string]string{},
			wantError: true,
		},
		{
			name:       "profile supplies decision",
			profile:    map[string]string{"status": "rejected"},
			wantStatus: "rejected",
		},
		{
			name:       "profile deliberately supplies schema default",
			profile:    map[string]string{"status": "verified"},
			wantStatus: "verified",
		},
		{
			name:       "command line overrides profile",
			profile:    map[string]string{"status": "rejected"},
			args:       []string{"--status", "verified"},
			wantStatus: "verified",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			captured := make(chan capturedSurfaceRequest, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got := capturedSurfaceRequest{method: r.Method, path: r.URL.Path}
				got.err = json.NewDecoder(r.Body).Decode(&got.body)
				captured <- got
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"id":"reviewed"}}`))
			}))
			defer server.Close()
			isolateSurfaceConfig(t, server.URL)
			t.Setenv("HOME", t.TempDir())
			if err := saveProfileStore(&profileStore{Profiles: map[string]Profile{
				"review": {Name: "review", Values: tc.profile},
			}}); err != nil {
				t.Fatal(err)
			}

			args := []string{"--json", "--no-cache", "--profile", "review", "customers", "review", "update-customer", goldenUUID}
			args = append(args, tc.args...)
			_, _, err := runRootForAPITest(t, args, "")
			if tc.wantError {
				if err == nil || !strings.Contains(err.Error(), "status") {
					t.Fatalf("error = %v, want missing status error", err)
				}
				select {
				case got := <-captured:
					t.Fatalf("omitted status reached the API: %#v", got)
				default:
				}
				return
			}
			if err != nil {
				t.Fatalf("review update: %v", err)
			}
			select {
			case got := <-captured:
				if got.err != nil {
					t.Fatalf("decoding request: %v", got.err)
				}
				if got.method != http.MethodPatch || got.path != "/v1/customers/"+goldenUUID+"/review" {
					t.Fatalf("request = %s %s, want PATCH /v1/customers/%s/review", got.method, got.path, goldenUUID)
				}
				wantBody := map[string]any{"status": tc.wantStatus}
				if !reflect.DeepEqual(got.body, wantBody) {
					t.Fatalf("body = %#v, want %#v", got.body, wantBody)
				}
			default:
				t.Fatal("review update did not reach the API")
			}
		})
	}
}

func TestBindSurfaceRequiredProfileFalse(t *testing.T) {
	cmd := &cobra.Command{Use: "fixture"}
	bind := bindSurface(cmd, surface.Surface{
		Path:    "/fixture",
		HasBody: true,
		Flags: []surface.Flag{
			{Name: "control", In: surface.InBody, Key: "/relationship/control", Kind: surface.KindBoolean, Required: true},
		},
	})
	if err := ApplyProfileToFlags(cmd, &Profile{Values: map[string]string{"control": "false"}}); err != nil {
		t.Fatal(err)
	}
	req, err := bind(nil)
	if err != nil {
		t.Fatalf("binding profile false: %v", err)
	}
	wantBody := map[string]any{"relationship": map[string]any{"control": false}}
	if !reflect.DeepEqual(req.Body, wantBody) {
		t.Fatalf("body = %#v, want %#v", req.Body, wantBody)
	}
}
