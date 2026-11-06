// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package config_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/straddle-build/straddle-cli/internal/client"
	"github.com/straddle-build/straddle-cli/internal/config"
)

func TestSaveTokensReplacesStoredCredentials(t *testing.T) {
	cases := []struct {
		name string
		file string
		env  string
	}{
		{name: "new file"},
		{name: "environment override", env: "fixture-old-env"},
		{name: "legacy api key", file: "api_key = 'fixture-old-file'\n"},
		{name: "legacy header", file: "auth_header = 'Bearer fixture-old-header'\n"},
		{name: "all credential sources", file: "api_key = 'fixture-old-file'\nauth_header = 'Bearer fixture-old-header'\naccess_token = 'fixture-old-token'\n", env: "fixture-old-env"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STRADDLE_API_KEY", tc.env)
			t.Setenv("STRADDLE_BASE_URL", "https://sandbox.straddle.com")
			path := filepath.Join(t.TempDir(), "config.toml")
			if tc.file != "" {
				if err := os.WriteFile(path, []byte(tc.file), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := cfg.SaveTokens("", "", "fixture-new-token", "", time.Time{}); err != nil {
				t.Fatal(err)
			}
			wantActive := "Bearer fixture-new-token"
			if tc.env != "" {
				wantActive = "Bearer " + tc.env
			}
			if got := cfg.AuthHeader(); got != wantActive {
				t.Errorf("active credential = %q, want %q", got, wantActive)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "fixture-old-") {
				t.Error("replaced or environment-derived credential remained on disk")
			}
			t.Setenv("STRADDLE_API_KEY", "")
			reloaded, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := reloaded.AuthHeader(); got != "Bearer fixture-new-token" {
				t.Errorf("reloaded credential = %q, want newly saved token", got)
			}
		})
	}
}

func TestLegacyAPIKeyLoadsUntilReplaced(t *testing.T) {
	t.Setenv("STRADDLE_API_KEY", "")
	t.Setenv("STRADDLE_BASE_URL", "https://sandbox.straddle.com")
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("api_key = 'fixture-legacy-key'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.AuthHeader(); got != "Bearer fixture-legacy-key" {
		t.Fatalf("legacy credential = %q", got)
	}
}

func TestClearTokensClearsSavedAndRuntimeCredentials(t *testing.T) {
	t.Setenv("STRADDLE_API_KEY", "fixture-env")
	t.Setenv("STRADDLE_BASE_URL", "https://sandbox.straddle.com")
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveTokens("fixture-client", "fixture-secret", "fixture-access", "fixture-refresh", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ClearTokens(); err != nil {
		t.Fatal(err)
	}
	if cfg.AuthHeader() != "" {
		t.Fatal("logout left an active credential on the loaded config")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "fixture-") {
		t.Fatal("logout left credentials on disk")
	}
	reloaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.AuthHeader(); got != "Bearer fixture-env" {
		t.Fatalf("exported environment credential should still apply on fresh load: %q", got)
	}
	t.Setenv("STRADDLE_API_KEY", "")
	reloaded, err = config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.AuthHeader() != "" {
		t.Fatal("logout did not survive reload after environment removal")
	}
}

func TestRotatedTokenAuthenticatesHTTPRequests(t *testing.T) {
	t.Setenv("STRADDLE_VERIFY", "")
	t.Setenv("STRADDLE_VERIFY_LIVE_HTTP", "")
	t.Setenv("STRADDLE_API_KEY", "fixture-env")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"authorization": r.Header.Get("Authorization")})
	}))
	defer server.Close()
	t.Setenv("STRADDLE_BASE_URL", server.URL)
	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveTokens("", "", "fixture-saved", "", time.Time{}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ env, want string }{
		{"fixture-env", "Bearer fixture-env"},
		{"", "Bearer fixture-saved"},
	} {
		t.Setenv("STRADDLE_API_KEY", tc.env)
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		c := client.New(cfg, time.Second, 0)
		c.NoCache = true
		body, err := c.Get("/v1/customers", nil)
		if err != nil {
			t.Fatal(err)
		}
		var response map[string]string
		if err := json.Unmarshal(body, &response); err != nil {
			t.Fatal(err)
		}
		if got := response["authorization"]; got != tc.want {
			t.Errorf("Authorization = %q, want %q", got, tc.want)
		}
	}
}

// Polling settings: environment overrides the file field by field, saving
// an API token keeps the file's polling settings without persisting the
// environment's, and logout removes the stored polling token.
func TestPollingSettings(t *testing.T) {
	const file = "polling_url = 'https://file.example/consumer/{consumer_id}'\npolling_token = 'fixture-file-token'\n"
	cases := []struct {
		name      string
		envURL    string
		envToken  string
		wantURL   string
		wantToken string
	}{
		{name: "config file", wantURL: "https://file.example/consumer/{consumer_id}", wantToken: "fixture-file-token"},
		{name: "environment token only", envToken: "fixture-env-token", wantURL: "https://file.example/consumer/{consumer_id}", wantToken: "fixture-env-token"},
		{name: "environment URL and token", envURL: "https://env.example/consumer/{consumer_id}", envToken: "fixture-env-token", wantURL: "https://env.example/consumer/{consumer_id}", wantToken: "fixture-env-token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STRADDLE_API_KEY", "")
			t.Setenv("STRADDLE_BASE_URL", "https://sandbox.straddle.com")
			t.Setenv("STRADDLE_POLLING_URL", tc.envURL)
			t.Setenv("STRADDLE_POLLING_TOKEN", tc.envToken)
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(file), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if gotURL, gotToken := cfg.Polling(); gotURL != tc.wantURL || gotToken != tc.wantToken {
				t.Errorf("Polling() = %q, %q; want %q, %q", gotURL, gotToken, tc.wantURL, tc.wantToken)
			}

			if err := cfg.SaveTokens("", "", "fixture-api-token", "", time.Time{}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "file.example") || !strings.Contains(string(data), "fixture-file-token") {
				t.Errorf("saving an API token dropped the stored polling settings:\n%s", data)
			}
			if strings.Contains(string(data), "env.example") || strings.Contains(string(data), "fixture-env-token") {
				t.Errorf("saving persisted environment-only polling settings:\n%s", data)
			}

			if err := cfg.ClearTokens(); err != nil {
				t.Fatal(err)
			}
			data, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "fixture-") {
				t.Errorf("logout left a credential on disk:\n%s", data)
			}
			if !strings.Contains(string(data), "file.example") {
				t.Errorf("logout dropped the polling URL, which is not a credential:\n%s", data)
			}
		})
	}
}

// TestSaveTokensDoesNotPersistBaseURLOverride pins the invariant that
// STRADDLE_BASE_URL is a runtime-only override: SaveTokens must write the
// file's base_url back to disk unchanged, never the env value. Otherwise a
// one-time shell environment (e.g. a verify-mode mock URL) permanently
// redirects every later command — including authenticated ones — to the
// wrong host. Mirrors the envAPIKey/envPolling* unexported-field pattern.
func TestSaveTokensDoesNotPersistBaseURLOverride(t *testing.T) {
	const fileBaseURL = "https://sandbox.straddle.com"
	const envBaseURL = "http://127.0.0.1:9999" // loopback http passes RequireSecureURL

	cases := []struct {
		name string
		file string
	}{
		{name: "no file base_url", file: ""},
		{name: "file base_url present", file: "base_url = '" + fileBaseURL + "'\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STRADDLE_API_KEY", "")
			t.Setenv("STRADDLE_BASE_URL", envBaseURL)
			path := filepath.Join(t.TempDir(), "config.toml")
			if tc.file != "" {
				if err := os.WriteFile(path, []byte(tc.file), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			// The env override must be visible at runtime via the accessor...
			if got := cfg.EffectiveBaseURL(); got != envBaseURL {
				t.Fatalf("EffectiveBaseURL() before save = %q, want %q", got, envBaseURL)
			}
			// ...but must NOT mutate the serialized field.
			if cfg.BaseURL == envBaseURL {
				t.Fatalf("cfg.BaseURL leaked env override %q before save", envBaseURL)
			}
			if err := cfg.SaveTokens("", "", "fixture-token", "", time.Time{}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), envBaseURL) {
				t.Errorf("SaveTokens persisted the env base URL override to disk:\n%s", data)
			}
			if tc.file != "" && !strings.Contains(string(data), fileBaseURL) {
				t.Errorf("SaveTokens dropped the file's base_url:\n%s", data)
			}
			// A fresh load with no env var must see the file's base_url, not the
			// poisoned override — this is the regression that silently redirected
			// later authenticated commands.
			t.Setenv("STRADDLE_BASE_URL", "")
			reloaded, err := config.Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := reloaded.EffectiveBaseURL(); got == envBaseURL {
				t.Fatalf("reloaded EffectiveBaseURL() = %q, env override leaked across save", got)
			}
			if tc.file != "" && reloaded.EffectiveBaseURL() != fileBaseURL {
				t.Fatalf("reloaded EffectiveBaseURL() = %q, want file value %q", reloaded.EffectiveBaseURL(), fileBaseURL)
			}
		})
	}
}

// TestClearTokensDoesNotPersistBaseURLOverride pins the same invariant for
// the logout path: ClearTokens must not write the env base URL override to
// disk either.
func TestClearTokensDoesNotPersistBaseURLOverride(t *testing.T) {
	const fileBaseURL = "https://sandbox.straddle.com"
	const envBaseURL = "http://127.0.0.1:9999"

	t.Setenv("STRADDLE_API_KEY", "")
	t.Setenv("STRADDLE_BASE_URL", envBaseURL)
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("base_url = '"+fileBaseURL+"'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SaveTokens("", "", "fixture-token", "", time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.ClearTokens(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), envBaseURL) {
		t.Errorf("ClearTokens persisted the env base URL override to disk:\n%s", data)
	}
	if !strings.Contains(string(data), fileBaseURL) {
		t.Errorf("ClearTokens dropped the file's base_url:\n%s", data)
	}
	t.Setenv("STRADDLE_BASE_URL", "")
	reloaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.EffectiveBaseURL(); got != fileBaseURL {
		t.Fatalf("reloaded EffectiveBaseURL() = %q, want file value %q", got, fileBaseURL)
	}
}

// TestEffectiveBaseURL pins the accessor's layering: the env override wins
// over the file value; without it, the file value (or default) is returned.
func TestEffectiveBaseURL(t *testing.T) {
	t.Setenv("STRADDLE_API_KEY", "")
	t.Setenv("STRADDLE_ENVIRONMENT", "")

	t.Run("env override wins over file value", func(t *testing.T) {
		t.Setenv("STRADDLE_BASE_URL", "http://127.0.0.1:8080")
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("base_url = 'https://sandbox.straddle.com'\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := cfg.EffectiveBaseURL(), "http://127.0.0.1:8080"; got != want {
			t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
		}
		if got := cfg.BaseURL; got != "https://sandbox.straddle.com" {
			t.Fatalf("BaseURL = %q, want file value unchanged", got)
		}
	})

	t.Run("file value when no env override", func(t *testing.T) {
		t.Setenv("STRADDLE_BASE_URL", "")
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, []byte("base_url = 'https://sandbox.straddle.com'\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := cfg.EffectiveBaseURL(), "https://sandbox.straddle.com"; got != want {
			t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
		}
	})

	t.Run("default when neither set", func(t *testing.T) {
		t.Setenv("STRADDLE_BASE_URL", "")
		cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := cfg.EffectiveBaseURL(), "https://{environment}.straddle.com"; got != want {
			t.Fatalf("EffectiveBaseURL() = %q, want %q", got, want)
		}
	})
}

// TestClientUsesEffectiveBaseURL pins that the client targets the env
// override, not the file value — so requests (and the response cache key)
// stay accurate when STRADDLE_BASE_URL is set.
func TestClientUsesEffectiveBaseURL(t *testing.T) {
	t.Setenv("STRADDLE_API_KEY", "")
	t.Setenv("STRADDLE_BASE_URL", "")
	t.Setenv("STRADDLE_ENVIRONMENT", "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"host": r.Host})
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("base_url = 'https://sandbox.straddle.com'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("STRADDLE_BASE_URL", server.URL)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	c := client.New(cfg, time.Second, 0)
	c.NoCache = true
	body, err := c.Get("/v1/customers", nil)
	if err != nil {
		t.Fatal(err)
	}
	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(server.URL, "http://127.0.0.1") && !strings.HasPrefix(server.URL, "http://localhost") {
		t.Fatalf("test server URL %q is not loopback", server.URL)
	}
	// The request reached the env-override server, not sandbox.straddle.com.
	if !strings.Contains(server.URL, resp["host"]) {
		t.Fatalf("client hit host %q, want the env-override server %q", resp["host"], server.URL)
	}
}
