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
