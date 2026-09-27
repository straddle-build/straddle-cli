// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// doctorSetup creates an isolated config + environment for a doctor run.
// homeDir isolates the cache DB (defaultDBPath derives from $HOME) and the
// config fallback path. The caller-provided configToml is written to the
// STRADDLE_CONFIG path; an empty string writes an (empty) config file so
// config.Load succeeds without a base_url. STRADDLE_API_KEY / base URL
// overrides are the caller's responsibility (passed via t.Setenv after this
// helper runs, or via the returned config path).
func doctorSetup(t *testing.T, configToml string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("STRADDLE_PLATFORM_CONFIG", filepath.Join(home, "platform.toml"))
	t.Setenv("STRADDLE_BASE_URL", "")
	t.Setenv("STRADDLE_VERIFY", "")
	t.Setenv("STRADDLE_VERIFY_LIVE_HTTP", "")
	configPath := filepath.Join(home, "config.toml")
	t.Setenv("STRADDLE_CONFIG", configPath)
	if err := os.WriteFile(configPath, []byte(configToml), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return configPath
}

// runDoctor builds a fresh root command tree, disables color so human-mode
// FAIL indicators render as the literal string "FAIL", and executes the
// given args. Returns (stdout, stderr, err) — err is the RunE return value
// (e.g. the --fail-on=error trigger), unmodified by the root error wrapper.
func runDoctor(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	origNoColor := noColor
	origHumanFriendly := humanFriendly
	noColor = true
	humanFriendly = false
	t.Cleanup(func() {
		noColor = origNoColor
		humanFriendly = origHumanFriendly
	})

	cmd := RootCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// TestDoctorValueIsError pins the shared failure-classification keyword set.
// This is the single source of truth used by both the human renderer's red
// FAIL indicator and the --fail-on exit gate; keeping them in sync is the
// fix for the "not configured" divergence bug.
func TestDoctorValueIsError(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{name: "auth not configured", s: "not configured", want: true},
		{name: "api not configured with guidance", s: "not configured (set base_url in config file)", want: true},
		{name: "error prefix", s: "error: bad toml", want: true},
		{name: "embedded error", s: "client init error: boom", want: true},
		{name: "unreachable", s: "unreachable: dial tcp: connection refused", want: true},
		{name: "invalid", s: "invalid config: missing field", want: true},
		{name: "missing keyword", s: "ERROR missing required: STRADDLE_API_KEY", want: true},
		{name: "reachable does not contain unreachable", s: "reachable", want: false},
		{name: "reachable with status", s: "reachable (HTTP 200 at /)", want: false},
		{name: "configured", s: "configured", want: false},
		{name: "env vars ok", s: "OK 1/1 available", want: false},
		{name: "credentials not verified is not an error", s: "present, not verified. Run straddle accounts list to confirm the token works end-to-end.", want: false},
		{name: "verify normal operation", s: "normal operation", want: false},
		{name: "blocked interstitial is not error-classified", s: "blocked by Cloudflare interstitial — the configured transport reached the wall.", want: false},
		{name: "empty string", s: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := doctorValueIsError(tt.s); got != tt.want {
				t.Fatalf("doctorValueIsError(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

// TestDoctorExitForFailOn exercises the --fail-on exit gate directly. The
// "not configured" cases are the regression: before the fix the gate did not
// include "not configured" in its keyword set, so a report whose only
// failure-classified value was "not configured" left worstError=false and
// returned nil — violating the --fail-on=error contract while the renderer
// painted the same state red.
func TestDoctorExitForFailOn(t *testing.T) {
	tests := []struct {
		name    string
		failOn  string
		report  map[string]any
		wantErr bool
	}{
		{
			name:    "empty failOn never fails even with not configured",
			failOn:  "",
			report:  map[string]any{"api": "not configured (set base_url in config file)"},
			wantErr: false,
		},
		{
			name:    "error trips on api not configured — regression",
			failOn:  "error",
			report:  map[string]any{"api": "not configured (set base_url in config file)"},
			wantErr: true,
		},
		{
			name:    "error trips on unreachable — existing keyword guard",
			failOn:  "error",
			report:  map[string]any{"api": "unreachable: dial tcp: connection refused"},
			wantErr: true,
		},
		{
			name:    "error trips on client init error — existing keyword guard",
			failOn:  "error",
			report:  map[string]any{"api": "client init error: missing transport"},
			wantErr: true,
		},
		{
			name:    "error trips on env vars missing — existing keyword guard",
			failOn:  "error",
			report:  map[string]any{"env_vars": "ERROR missing required: STRADDLE_API_KEY"},
			wantErr: true,
		},
		{
			name:    "reachable does not trip error",
			failOn:  "error",
			report:  map[string]any{"api": "reachable"},
			wantErr: false,
		},
		{
			name:    "healthy report does not trip error",
			failOn:  "error",
			report:  map[string]any{"auth": "configured", "api": "reachable (HTTP 200 at /)", "env_vars": "OK 1/1 available"},
			wantErr: false,
		},
		{
			name:    "credentials not verified does not trip error",
			failOn:  "error",
			report:  map[string]any{"credentials": "present, not verified. Run straddle accounts list to confirm the token works end-to-end."},
			wantErr: false,
		},
		{
			name:    "stale error-level does not trip from a stale cache alone",
			failOn:  "error",
			report:  map[string]any{"cache": map[string]any{"status": "stale"}},
			wantErr: false,
		},
		{
			name:    "error trips on cache status error map",
			failOn:  "error",
			report:  map[string]any{"cache": map[string]any{"status": "error", "error": "boom"}},
			wantErr: true,
		},
		{
			name:    "stale trips on stale cache",
			failOn:  "stale",
			report:  map[string]any{"cache": map[string]any{"status": "stale"}},
			wantErr: true,
		},
		{
			name:    "stale does not trip on unknown cache",
			failOn:  "stale",
			report:  map[string]any{"cache": map[string]any{"status": "unknown"}},
			wantErr: false,
		},
		{
			name:    "stale also trips on error-level values",
			failOn:  "stale",
			report:  map[string]any{"api": "unreachable: dial tcp"},
			wantErr: true,
		},
		{
			name:    "unknown failOn value errors",
			failOn:  "warn",
			report:  map[string]any{"api": "reachable"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := doctorExitForFailOn(tt.failOn, tt.report)
			if tt.wantErr && err == nil {
				t.Fatalf("doctorExitForFailOn(%q, %v) = nil, want non-nil", tt.failOn, tt.report)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("doctorExitForFailOn(%q, %v) = %v, want nil", tt.failOn, tt.report, err)
			}
		})
	}
}

// TestDoctorFailOnErrorExitsNonZeroOnNotConfiguredAPI is the end-to-end
// regression for the bug: with credentials present and an explicit empty
// base_url in the config, the doctor prints "FAIL API: not configured ..."
// and --fail-on=error MUST exit non-zero. Before the shared classifier, the
// gate missed "not configured" and returned exit 0.
func TestDoctorFailOnErrorExitsNonZeroOnNotConfiguredAPI(t *testing.T) {
	doctorSetup(t, "base_url = \"\"\n")
	t.Setenv("STRADDLE_API_KEY", "fake-key")

	stdout, _, err := runDoctor(t, []string{"doctor", "--fail-on=error"})
	if err == nil {
		t.Fatal("doctor --fail-on=error exited nil for a not-configured API; expected non-zero")
	}
	if !strings.Contains(err.Error(), "--fail-on=error triggered") {
		t.Fatalf("error = %v, want --fail-on=error triggered", err)
	}

	// The renderer must paint the same state as FAIL: the human output's
	// API line must carry the FAIL indicator and the not-configured value.
	apiLine := ""
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, "not configured (set base_url in config file)") {
			apiLine = line
			break
		}
	}
	if apiLine == "" {
		t.Fatalf("stdout missing API not-configured line; output:\n%s", stdout)
	}
	if !strings.Contains(apiLine, "FAIL") {
		t.Fatalf("API not-configured line is not FAIL-classified (renderer/gate divergence); line: %q", apiLine)
	}
	if !strings.Contains(apiLine, "API:") {
		t.Fatalf("expected line labeled API:; got %q", apiLine)
	}
}

// TestDoctorFailOnErrorExitsZeroWhenReachable is the happy-path regression:
// a reachable API (httptest 200) with credentials present must NOT trip
// --fail-on=error and must NOT render a FAIL indicator.
func TestDoctorFailOnErrorExitsZeroWhenReachable(t *testing.T) {
	doctorSetup(t, "")
	t.Setenv("STRADDLE_API_KEY", "test_key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	t.Setenv("STRADDLE_BASE_URL", server.URL)

	stdout, _, err := runDoctor(t, []string{"doctor", "--fail-on=error"})
	if err != nil {
		t.Fatalf("doctor --fail-on=error on a reachable API returned error: %v", err)
	}
	if strings.Contains(stdout, "FAIL") {
		t.Fatalf("reachable doctor output unexpectedly contains FAIL; output:\n%s", stdout)
	}
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Contains(line, "API:") && !strings.Contains(line, "reachable") {
			t.Fatalf("API line not reported reachable; line: %q", line)
		}
	}
}

// runtimeContextOf runs a command and decodes the runtime_context it reports.
func runtimeContextOf(t *testing.T, args ...string) runtimeContext {
	t.Helper()
	stdout, stderr, err := runDoctor(t, args)
	if err != nil {
		t.Fatalf("%v: %v\nstderr: %s", args, err, stderr)
	}
	var out struct {
		RuntimeContext *runtimeContext `json:"runtime_context"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil || out.RuntimeContext == nil {
		t.Fatalf("%v: no runtime_context (%v) in:\n%s", args, err, stdout)
	}
	return *out.RuntimeContext
}

// TestRuntimeContextReportsSelection checks that doctor and agent-context
// report the same environment, integration type and acting account a
// command would scope its local data to.
func TestRuntimeContextReportsSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()
	origin := server.URL
	str := func(s string) *string { return &s }

	tests := []struct {
		name      string
		config    string
		platform  string
		profile   string
		env       map[string]string
		args      []string
		offline   bool // agent-context only; doctor would dial the real host
		want      runtimeContext
		wantError string
	}{
		{
			name: "unset context is null, not guessed",
			env:  map[string]string{"STRADDLE_BASE_URL": origin},
			want: runtimeContext{Environment: str(origin)},
		},
		{
			name:    "STRADDLE_ENVIRONMENT selects the default host",
			env:     map[string]string{"STRADDLE_ENVIRONMENT": "production"},
			offline: true,
			want:    runtimeContext{Environment: str("https://production.straddle.com")},
		},
		{
			name:     "saas acts as use-account",
			platform: "integration_type = 'saas'\ncurrent_account = 'acct_sticky'\n",
			env:      map[string]string{"STRADDLE_BASE_URL": origin},
			want:     runtimeContext{Environment: str(origin), IntegrationType: str("saas"), ActingAccount: str("acct_sticky")},
		},
		{
			name:     "explicit --account overrides use-account",
			platform: "integration_type = 'saas'\ncurrent_account = 'acct_sticky'\n",
			env:      map[string]string{"STRADDLE_BASE_URL": origin},
			args:     []string{"--account", "acct_flag"},
			want:     runtimeContext{Environment: str(origin), IntegrationType: str("saas"), ActingAccount: str("acct_flag")},
		},
		{
			name:     "saved profile account overrides use-account",
			platform: "integration_type = 'saas'\ncurrent_account = 'acct_sticky'\n",
			profile:  "acct_profile",
			env:      map[string]string{"STRADDLE_BASE_URL": origin},
			args:     []string{"--profile", "p"},
			want:     runtimeContext{Environment: str(origin), IntegrationType: str("saas"), ActingAccount: str("acct_profile")},
		},
		{
			name:     "marketplace keeps its acting account",
			platform: "integration_type = 'marketplace'\ncurrent_account = 'acct_sticky'\n",
			env:      map[string]string{"STRADDLE_BASE_URL": origin},
			want:     runtimeContext{Environment: str(origin), IntegrationType: str("marketplace"), ActingAccount: str("acct_sticky")},
		},
		{
			name:     "direct account never acts as another account",
			platform: "integration_type = 'account'\ncurrent_account = 'acct_sticky'\n",
			env:      map[string]string{"STRADDLE_BASE_URL": origin},
			want:     runtimeContext{Environment: str(origin), IntegrationType: str("account")},
		},
		{
			name:      "direct account rejects --account",
			platform:  "integration_type = 'account'\n",
			env:       map[string]string{"STRADDLE_BASE_URL": origin},
			args:      []string{"--account", "acct_flag"},
			want:      runtimeContext{IntegrationType: str("account")},
			wantError: "remove --account",
		},
		{
			name:      "invalid integration type is reported, not dropped",
			platform:  "integration_type = 'reseller'\ncurrent_account = 'acct_sticky'\n",
			env:       map[string]string{"STRADDLE_BASE_URL": origin},
			want:      runtimeContext{Environment: str(origin), IntegrationType: str("reseller"), ActingAccount: str("acct_sticky")},
			wantError: `invalid integration type "reseller"`,
		},
		{
			name:   "STRADDLE_BASE_URL overrides saved base_url",
			config: "base_url = 'http://localhost:1'\n",
			env:    map[string]string{"STRADDLE_BASE_URL": origin},
			want:   runtimeContext{Environment: str(origin)},
		},
		{
			name:   "saved base_url is the target, not STRADDLE_ENVIRONMENT",
			config: "base_url = '" + origin + "'\n",
			env:    map[string]string{"STRADDLE_ENVIRONMENT": "production"},
			want:   runtimeContext{Environment: str(origin)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := filepath.Dir(doctorSetup(t, tc.config))
			t.Setenv("STRADDLE_API_KEY", "test_key")
			t.Setenv("STRADDLE_ENVIRONMENT", "")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if tc.platform != "" {
				if err := os.WriteFile(filepath.Join(home, "platform.toml"), []byte(tc.platform), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.profile != "" {
				s := &profileStore{Profiles: map[string]Profile{"p": {Name: "p", Values: map[string]string{"account": tc.profile}}}}
				if err := saveProfileStore(s); err != nil {
					t.Fatal(err)
				}
			}
			surfaces := map[string]runtimeContext{"agent-context": runtimeContextOf(t, append([]string{"agent-context"}, tc.args...)...)}
			if !tc.offline {
				surfaces["doctor"] = runtimeContextOf(t, append([]string{"doctor", "--agent"}, tc.args...)...)
			}
			for surface, got := range surfaces {
				if !strings.Contains(got.Error, tc.wantError) || (tc.wantError == "") != (got.Error == "") {
					t.Errorf("%s error = %q, want containing %q", surface, got.Error, tc.wantError)
				}
				got.Error = ""
				gotJSON, _ := json.Marshal(got)
				wantJSON, _ := json.Marshal(tc.want)
				if string(gotJSON) != string(wantJSON) {
					t.Errorf("%s runtime_context = %s, want %s", surface, gotJSON, wantJSON)
				}
			}
		})
	}
}

// TestAgentContextDoesNotOpenStore keeps agent-context offline: reporting
// the runtime context must not create the local store.
func TestAgentContextDoesNotOpenStore(t *testing.T) {
	home := filepath.Dir(doctorSetup(t, ""))
	runtimeContextOf(t, "agent-context")
	if _, err := os.Stat(filepath.Join(home, ".local")); !os.IsNotExist(err) {
		t.Fatalf("agent-context created local store directory (stat err %v)", err)
	}
}

// TestDoctorFailOnErrorTripsOnInvalidRuntimeContext keeps the exit gate
// honest when the integration type cannot be applied.
func TestDoctorFailOnErrorTripsOnInvalidRuntimeContext(t *testing.T) {
	home := filepath.Dir(doctorSetup(t, ""))
	if err := os.WriteFile(filepath.Join(home, "platform.toml"), []byte("integration_type = 'reseller'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	t.Setenv("STRADDLE_BASE_URL", server.URL)
	t.Setenv("STRADDLE_API_KEY", "test_key")
	stdout, _, err := runDoctor(t, []string{"doctor", "--fail-on=error"})
	if err == nil || !strings.Contains(err.Error(), "--fail-on=error triggered") {
		t.Fatalf("err = %v, want --fail-on=error triggered", err)
	}
	if !strings.Contains(stdout, "FAIL Runtime Context: error") {
		t.Fatalf("human output missing FAIL runtime context:\n%s", stdout)
	}
}
