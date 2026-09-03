// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
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
			name:     "dry run skips required flag",
			required: true,
			args:     []string{"widget-1", "--dry-run"},
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
			wantErr:  "parsing --metadata JSON: unexpected end of JSON input",
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
			{Name: "metadata", In: surface.InBody, Key: "/metadata", Kind: surface.KindJSON},
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
	bind := bindSurface(cmd, flags, s)
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
