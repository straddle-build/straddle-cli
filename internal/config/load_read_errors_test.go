// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoad_UnreadableConfigReturnsError pins the fix for the swallowed
// read-error bug: an existing-but-unreadable config (a directory at the
// path, or a permission-denied regular file) must surface an actionable
// error that names the path, not silently fall through to a default empty
// config. Before the fix every os.ReadFile error was treated like a
// missing file, so Load returned a zero-value config and nil error —
// making an unreadable config indistinguishable from a first-run install
// and letting auth set-token overwrite a file Load never read.
func TestLoad_UnreadableConfigReturnsError(t *testing.T) {
	t.Setenv("STRADDLE_API_KEY", "")
	t.Setenv("STRADDLE_BASE_URL", "")
	t.Setenv("STRADDLE_ENVIRONMENT", "")

	t.Run("directory at config path", func(t *testing.T) {
		// A directory at the config path fails os.ReadFile with "is a
		// directory" under any uid — including root — so this case pins the
		// fix in environments where file permissions are bypassed.
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatalf("mkdir at config path: %v", err)
		}
		cfg, err := Load(path)
		if err == nil {
			t.Fatalf("Load() with directory at config path returned nil error; read failure swallowed")
		}
		if cfg != nil {
			t.Fatalf("Load() returned non-nil config (%v) alongside an error; callers must not receive a usable config on read failure", cfg)
		}
		if !strings.Contains(err.Error(), path) {
			t.Fatalf("Load() error = %q; want it to name the config path %q", err.Error(), path)
		}
		if !strings.Contains(err.Error(), "reading") {
			t.Fatalf("Load() error = %q; want a 'reading' verb so the failure is actionable", err.Error())
		}
	})

	t.Run("permission-denied regular file", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permission-denied case does not apply when running as root (root bypasses file permissions)")
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		// A 0000 regular file is a real-world failure mode (chmod drift,
		// a botched backup restore) that is NOT a missing file.
		writeFileLoose(t, path, "api_key = 'fixture-key'\n", 0o000)
		cfg, err := Load(path)
		if err == nil {
			t.Fatalf("Load() with unreadable (0000) file returned nil error; read failure swallowed")
		}
		if cfg != nil {
			t.Fatalf("Load() returned non-nil config (%v) alongside an error", cfg)
		}
		if !strings.Contains(err.Error(), path) {
			t.Fatalf("Load() error = %q; want it to name the config path %q", err.Error(), path)
		}
	})

	t.Run("unreadable file error is not classified as not-exist", func(t *testing.T) {
		// Guard against a regression that re-widens the tolerance: the
		// directory error must not satisfy os.IsNotExist / errors.Is
		// (os.ErrNotExist), otherwise Load would tolerate it.
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatalf("mkdir at config path: %v", err)
		}
		_, readErr := os.ReadFile(path)
		if readErr == nil {
			t.Fatalf("os.ReadFile(directory) returned nil error; precondition for test is broken")
		}
		if os.IsNotExist(readErr) {
			t.Fatalf("os.ReadFile(directory) error %q satisfies os.ErrNotExist; Load would wrongly tolerate it", readErr)
		}
		_, loadErr := Load(path)
		if loadErr == nil {
			t.Fatalf("Load(directory) returned nil error; the read error was tolerated like a missing file")
		}
	})
}

// TestLoad_MissingConfigTolerated pins the deliberate fresh-install
// bootstrap: a genuinely absent config file must still return a usable
// default config and nil error. The fix tightened tolerance from "any
// read error" to "file absent", so this case must not regress.
func TestLoad_MissingConfigTolerated(t *testing.T) {
	t.Setenv("STRADDLE_API_KEY", "")
	t.Setenv("STRADDLE_BASE_URL", "")
	t.Setenv("STRADDLE_CONFIG", "")
	t.Setenv("STRADDLE_ENVIRONMENT", "")

	// Absent file in an existing parent directory.
	path := filepath.Join(t.TempDir(), "config.toml")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("precondition: config must not exist, stat err = %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() on absent config returned error %v; want nil (fresh-install bootstrap)", err)
	}
	if cfg == nil {
		t.Fatal("Load() on absent config returned nil config; want a default config")
	}
	if cfg.BaseURL != "https://{environment}.straddle.com" {
		t.Fatalf("BaseURL = %q, want default https://{environment}.straddle.com", cfg.BaseURL)
	}
	if cfg.Path != path {
		t.Fatalf("cfg.Path = %q, want %q", cfg.Path, path)
	}
	if got := cfg.AuthHeader(); got != "" {
		t.Fatalf("AuthHeader() = %q, want empty (no credentials on fresh install)", got)
	}
}

// TestLoad_ReadableFileStillLoads pins the happy path so the tightened
// error handling cannot regress the ordinary case: a readable TOML file
// is parsed and its fields populate the config exactly as before.
func TestLoad_ReadableFileStillLoads(t *testing.T) {
	t.Setenv("STRADDLE_API_KEY", "")
	t.Setenv("STRADDLE_BASE_URL", "")
	t.Setenv("STRADDLE_ENVIRONMENT", "")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	content := "base_url = 'https://sandbox.straddle.com'\napi_key = 'fixture-readable-key'\npolling_token = 'fixture-poll'\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write readable config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() on readable config returned error %v", err)
	}
	if cfg.BaseURL != "https://sandbox.straddle.com" {
		t.Fatalf("BaseURL = %q, want https://sandbox.straddle.com", cfg.BaseURL)
	}
	if got := cfg.AuthHeader(); got != "Bearer fixture-readable-key" {
		t.Fatalf("AuthHeader() = %q, want Bearer fixture-readable-key", got)
	}
	if cfg.PollingToken != "fixture-poll" {
		t.Fatalf("PollingToken = %q, want fixture-poll", cfg.PollingToken)
	}
}
