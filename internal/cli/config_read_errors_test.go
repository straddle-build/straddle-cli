// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unreadableConfigSetup isolates HOME/config env and places a directory at
// the STRADDLE_CONFIG path so config.Load hits a read error that is NOT a
// "file absent" error. A directory fails os.ReadFile ("is a directory")
// under any uid, including root, so this is the portable pin for the
// swallowed-read-error fix. Returns the config path.
func unreadableConfigSetup(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("STRADDLE_PLATFORM_CONFIG", filepath.Join(home, "platform.toml"))
	t.Setenv("STRADDLE_BASE_URL", "")
	t.Setenv("STRADDLE_VERIFY", "")
	t.Setenv("STRADDLE_VERIFY_LIVE_HTTP", "")
	configPath := filepath.Join(home, "config.toml")
	if err := os.Mkdir(configPath, 0o700); err != nil {
		t.Fatalf("mkdir at config path: %v", err)
	}
	// Point STRADDLE_CONFIG at the directory so Load resolves it as the
	// config path (flags.configPath is empty in runDoctor-built commands
	// unless --config is passed).
	t.Setenv("STRADDLE_CONFIG", configPath)
	return configPath
}

// TestDoctorSurfacesUnreadableConfigError is the end-to-end regression for
// the swallowed read-error bug: with a directory at the config path,
// doctor must report the config read failure as a FAIL error naming the
// path, not the silent "Config: ok" it printed before the fix
// (config.Load returned a default config and nil error, masking the I/O
// problem). --fail-on=error must trip on the config error alone.
func TestDoctorSurfacesUnreadableConfigError(t *testing.T) {
	configPath := unreadableConfigSetup(t)
	// A non-empty API key isolates the config error as the sole failure:
	// env_vars stays "OK" and the only FAIL line is the Config line.
	t.Setenv("STRADDLE_API_KEY", "fixture-key")

	t.Run("human output reports Config error and trips --fail-on=error", func(t *testing.T) {
		stdout, _, err := runDoctor(t, []string{"doctor", "--fail-on=error"})
		if err == nil {
			t.Fatal("doctor --fail-on=error exited nil with an unreadable config; expected non-zero")
		}
		if !strings.Contains(err.Error(), "--fail-on=error triggered") {
			t.Fatalf("error = %v, want --fail-on=error triggered", err)
		}
		if strings.Contains(stdout, "OK Config:") {
			t.Fatalf("doctor still paints Config as OK for an unreadable config; output:\n%s", stdout)
		}
		if !strings.Contains(stdout, "FAIL Config:") {
			t.Fatalf("doctor output missing FAIL Config line; output:\n%s", stdout)
		}
		// The FAIL line must carry the actionable read error, not the bare
		// path and not "ok".
		if !strings.Contains(stdout, "reading config") {
			t.Fatalf("doctor Config line missing the actionable read error; output:\n%s", stdout)
		}
		if !strings.Contains(stdout, configPath) {
			t.Fatalf("doctor output missing the config path %q; output:\n%s", configPath, stdout)
		}
	})

	t.Run("json report surfaces the config error, not ok", func(t *testing.T) {
		stdout, _, err := runDoctor(t, []string{"doctor", "--json"})
		if err != nil {
			t.Fatalf("doctor --json returned error: %v", err)
		}
		var report map[string]any
		if err := json.Unmarshal([]byte(stdout), &report); err != nil {
			t.Fatalf("decode doctor json: %v\n%s", err, stdout)
		}
		got, _ := report["config"].(string)
		if got == "ok" {
			t.Fatalf("doctor json config = %q; want an error for an unreadable config", got)
		}
		if !strings.Contains(got, "error") {
			t.Fatalf("doctor json config = %q; want an error-classified value", got)
		}
		if !strings.Contains(got, "reading config") {
			t.Fatalf("doctor json config = %q; want the actionable read error", got)
		}
		if _, ok := report["config_path"]; ok {
			t.Fatalf("doctor json must not report config_path when Load failed; got %v", report["config_path"])
		}
	})
}

// TestAuthStatusReportsConfigErrorNotNotConfigured pins that an unreadable
// config surfaces as a config error (exit 10), not the misleading
// "Not authenticated / no credentials configured" (exit 4) that a silently
// empty config produced before the fix. The actionable message names the
// read failure so the user knows the file — not their credentials — is the
// problem.
func TestAuthStatusReportsConfigErrorNotNotConfigured(t *testing.T) {
	unreadableConfigSetup(t)
	t.Setenv("STRADDLE_API_KEY", "")

	stdout, _, err := runDoctor(t, []string{"auth", "status"})
	if err == nil {
		t.Fatal("auth status exited nil for an unreadable config; expected a config error")
	}
	if got := ExitCode(err); got != 10 {
		t.Fatalf("ExitCode = %d, want 10 (config error); err = %v", got, err)
	}
	if !strings.Contains(err.Error(), "reading config") {
		t.Fatalf("error = %v, want an actionable read-config error", err)
	}
	if strings.Contains(err.Error(), "no credentials configured") {
		t.Fatalf("error = %v; an unreadable config must not be masked as 'no credentials configured'", err)
	}
	if strings.Contains(stdout, "Not authenticated") {
		t.Fatalf("auth status still reports Not authenticated for an unreadable config; output:\n%s", stdout)
	}
}

// TestAuthSetTokenDoesNotDestroyUnreadableConfig pins the data-loss
// consequence of the swallowed read error: before the fix an unreadable
// config file (permission denied / EACCES) survived Load as an empty
// in-memory config, then SaveTokens atomically replaced it with a file
// containing only the new token — silently destroying base_url, headers,
// api_key, polling_url and template vars. With the fix Load returns an
// error, so auth set-token never reaches SaveTokens and the original
// file is preserved byte-for-byte.
func TestAuthSetTokenDoesNotDestroyUnreadableConfig(t *testing.T) {
	t.Setenv("STRADDLE_API_KEY", "")
	t.Setenv("STRADDLE_BASE_URL", "")
	t.Setenv("STRADDLE_CONFIG", "")

	// Portable case: a directory at the config path makes Load fail, so
	// set-token returns a config error and never writes through it.
	t.Run("directory at config path errors and does not write", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatalf("mkdir at config path: %v", err)
		}
		var out, errOut bytes.Buffer
		cmd := newAuthSetTokenCmd(&rootFlags{configPath: path})
		cmd.SetOut(&out)
		cmd.SetErr(&errOut)
		cmd.SetArgs([]string{"sk_live_new"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("auth set-token exited nil for an unreadable config; expected a config error")
		}
		if got := ExitCode(err); got != 10 {
			t.Fatalf("ExitCode = %d, want 10 (config error); err = %v", got, err)
		}
		if !strings.Contains(err.Error(), "reading config") {
			t.Fatalf("error = %v, want an actionable read-config error", err)
		}
		if strings.Contains(out.String(), "Token saved") {
			t.Fatalf("auth set-token reported success despite the unreadable config; output: %s", out.String())
		}
		// The path must remain a directory (SaveTokens never ran, so no
		// file replaced it).
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Fatalf("stat config path: %v", statErr)
		}
		if !info.IsDir() {
			t.Fatalf("config path was overwritten from a directory to %v; SaveTokens must not run when Load fails", info.Mode())
		}
	})

	// The real destruction case: an unreadable regular file in a writable
	// directory. Before the fix the new token overwrote the original
	// contents; after the fix Load errors and the original survives. Root
	// bypasses file permissions, so this subcase only runs as non-root.
	t.Run("unreadable regular file is preserved, not overwritten", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("permission-denied case does not apply when running as root (root bypasses file permissions)")
		}
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		original := "base_url = 'https://sandbox.straddle.com'\n" +
			"api_key = 'fixture-original-key'\n" +
			"headers = { 'X-Tenant' = 'fixture-tenant' }\n" +
			"polling_url = 'https://poll.example/consumer/{consumer_id}'\n" +
			"polling_token = 'fixture-poll-token'\n"
		writeFileLoose(t, path, original, 0o000)

		var out, errOut bytes.Buffer
		cmd := newAuthSetTokenCmd(&rootFlags{configPath: path})
		cmd.SetOut(&out)
		cmd.SetErr(&errOut)
		cmd.SetArgs([]string{"sk_live_new"})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("auth set-token exited nil for an unreadable (0000) file; expected a config error")
		}
		if got := ExitCode(err); got != 10 {
			t.Fatalf("ExitCode = %d, want 10 (config error); err = %v", got, err)
		}
		if strings.Contains(out.String(), "Token saved") {
			t.Fatalf("auth set-token reported success despite the unreadable file; output: %s", out.String())
		}

		// Restore readability and confirm the original contents survived:
		// SaveTokens never ran, so the file must be byte-for-byte the
		// original and must not carry the new token.
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatalf("chmod config back to readable: %v", err)
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read preserved config: %v", readErr)
		}
		if string(data) != original {
			t.Fatalf("unreadable config was modified by set-token.\nwant:\n%s\ngot:\n%s", original, data)
		}
		if bytes.Contains(data, []byte("sk_live_new")) {
			t.Fatalf("the new token overwrote the unreadable config file; content:\n%s", data)
		}
	})
}
