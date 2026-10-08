// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_BaseURLTransportPolicy pins the credential transport boundary:
// non-loopback plaintext endpoints are rejected before the client can attach
// Authorization, while loopback HTTP remains available for hermetic tests and
// HTTPS production endpoints stay accepted.
func TestLoad_BaseURLTransportPolicy(t *testing.T) {
	cases := []struct {
		name        string
		baseURL     string
		wantErr     string
		wantBaseURL string
	}{
		{
			name:    "rejects non-loopback http",
			baseURL: "http://sandbox.straddle.com",
			wantErr: "must use https",
		},
		{
			name:        "accepts IPv4 loopback http",
			baseURL:     "http://127.0.0.1:8080",
			wantBaseURL: "http://127.0.0.1:8080",
		},
		{
			name:        "accepts localhost http",
			baseURL:     "http://localhost:3000",
			wantBaseURL: "http://localhost:3000",
		},
		{
			name:        "accepts https",
			baseURL:     "https://sandbox.straddle.com",
			wantBaseURL: "https://sandbox.straddle.com",
		},
		{
			name: "accepts templated default",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STRADDLE_BASE_URL", tc.baseURL)
			t.Setenv("STRADDLE_API_KEY", "")
			t.Setenv("STRADDLE_CONFIG", "")
			t.Setenv("STRADDLE_ENVIRONMENT", "")

			cfg, err := Load(filepath.Join(t.TempDir(), "config.toml"))
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("Load() error = nil, want substring %q", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Load() error = %q, want substring %q", err.Error(), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if tc.wantBaseURL != "" && cfg.BaseURL != tc.wantBaseURL {
				t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, tc.wantBaseURL)
			}
		})
	}
}

// TestAuthSourceMatchesSentCredential pins AuthSource to the credential
// AuthHeader sends, from Load onward. The response cache key reads the label
// before the first request; auth status and doctor read it after.
func TestAuthSourceMatchesSentCredential(t *testing.T) {
	cases := []struct {
		name       string
		file       string
		env        string
		wantHeader string
		wantSource string
	}{
		{
			name:       "file auth_header shadows exported env key",
			file:       "auth_header = 'Bearer fixture-file-header'\napi_key = 'fixture-file-key'\naccess_token = 'fixture-file-token'\n",
			env:        "fixture-env",
			wantHeader: "Bearer fixture-file-header",
			wantSource: "config",
		},
		{
			name:       "env key shadows file api_key and access_token",
			file:       "api_key = 'fixture-file-key'\naccess_token = 'fixture-file-token'\n",
			env:        "fixture-env",
			wantHeader: "Bearer fixture-env",
			wantSource: "env:STRADDLE_API_KEY",
		},
		{
			name:       "file api_key shadows access_token",
			file:       "api_key = 'fixture-file-key'\naccess_token = 'fixture-file-token'\n",
			wantHeader: "Bearer fixture-file-key",
			wantSource: "config",
		},
		{
			name:       "access_token alone",
			file:       "access_token = 'fixture-file-token'\n",
			wantHeader: "Bearer fixture-file-token",
			wantSource: "oauth2",
		},
		{
			name: "no credential",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("STRADDLE_API_KEY", tc.env)
			t.Setenv("STRADDLE_BASE_URL", "")
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(tc.file), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.AuthSource != tc.wantSource {
				t.Errorf("AuthSource after Load = %q, want %q", cfg.AuthSource, tc.wantSource)
			}
			if got := cfg.AuthHeader(); got != tc.wantHeader {
				t.Errorf("AuthHeader() = %q, want %q", got, tc.wantHeader)
			}
			if cfg.AuthSource != tc.wantSource {
				t.Errorf("AuthSource after AuthHeader = %q, want %q", cfg.AuthSource, tc.wantSource)
			}
		})
	}

	t.Run("constructed config with auth_header", func(t *testing.T) {
		cfg := &Config{AuthHeaderVal: "Bearer fixture-header"}
		if got := cfg.AuthHeader(); got != "Bearer fixture-header" {
			t.Fatalf("AuthHeader() = %q, want %q", got, "Bearer fixture-header")
		}
		if cfg.AuthSource != "config" {
			t.Fatalf("AuthSource = %q, want %q", cfg.AuthSource, "config")
		}
	})
}
