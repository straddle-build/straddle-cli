package scripts

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const team = "ABCDE12345"

// Stand-ins for the macOS tools the release script drives. Each reads its
// scenario from the environment so one stub set covers every case.
var stubs = map[string]string{
	"security": `#!/usr/bin/env bash
printf '%b' "$STUB_IDENTITIES"`,
	"codesign": `#!/usr/bin/env bash
case "$1" in
--force) for last; do :; done; printf 'signed' >>"$last" ;;
--display) printf '%b' "$STUB_SIGNATURE" >&2 ;;
esac`,
	"ditto": `#!/usr/bin/env bash
for last; do :; done; : >"$last"`,
	"xcrun": `#!/usr/bin/env bash
if [ "$2" = submit ]; then
  printf '{"id":"sub-1","message":"done","status":"%s"}\n' "$STUB_STATUS"
else
  echo "notary log for $3"
fi`,
}

func TestMacOSSignNotarize(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	script, err := filepath.Abs("macos-sign-notarize.sh")
	if err != nil {
		t.Fatal(err)
	}

	identity := `  1) 0123ABCD "Developer ID Application: Straddle (` + team + `)"\n`
	signature := "Authority=Developer ID Application: Straddle (" + team + ")\\n" +
		"TeamIdentifier=" + team + "\\n" +
		"CodeDirectory v=20500 size=1 flags=0x10000(runtime) hashes=1\\n" +
		"Timestamp=Sep 27, 2026\\n"

	tests := []struct {
		name       string
		env        map[string]string
		wantErr    string
		wantSigned bool
	}{
		{name: "accepted notarization succeeds", wantSigned: true},
		{name: "missing app-specific password fails before signing", env: map[string]string{"APPLE_APP_SPECIFIC_PASSWORD": ""}, wantErr: "APPLE_APP_SPECIFIC_PASSWORD is not set"},
		{name: "missing keychain fails before signing", env: map[string]string{"APPLE_SIGNING_KEYCHAIN": ""}, wantErr: "APPLE_SIGNING_KEYCHAIN is not set"},
		{name: "no identity for team fails before signing", env: map[string]string{"STUB_IDENTITIES": `  1) 99 "Developer ID Application: Other (ZZZZZ99999)"\n`}, wantErr: "found 0"},
		{name: "ambiguous identities fail before signing", env: map[string]string{"STUB_IDENTITIES": identity + identity}, wantErr: "found 2"},
		{name: "missing hardened runtime fails", env: map[string]string{"STUB_SIGNATURE": strings.Replace(signature, "(runtime)", "(none)", 1)}, wantErr: "hardened runtime", wantSigned: true},
		{name: "missing secure timestamp fails", env: map[string]string{"STUB_SIGNATURE": strings.Replace(signature, "Timestamp=", "Signed Time=", 1)}, wantErr: "secure timestamp", wantSigned: true},
		{name: "rejected notarization fails", env: map[string]string{"STUB_STATUS": "Invalid"}, wantErr: "status 'Invalid'", wantSigned: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stubDir := filepath.Join(dir, "bin")
			if err := os.Mkdir(stubDir, 0o755); err != nil {
				t.Fatal(err)
			}
			for name, body := range stubs {
				if err := os.WriteFile(filepath.Join(stubDir, name), []byte(body+"\n"), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			binary := filepath.Join(dir, "straddle")
			if err := os.WriteFile(binary, []byte("macho"), 0o755); err != nil {
				t.Fatal(err)
			}

			env := map[string]string{
				"PATH":                        stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
				"APPLE_SIGNING_KEYCHAIN":      filepath.Join(dir, "release.keychain-db"),
				"APPLE_ID":                    "release@example.com",
				"APPLE_APP_SPECIFIC_PASSWORD": "app-specific-secret",
				"APPLE_TEAM_ID":               team,
				"STUB_IDENTITIES":             identity,
				"STUB_SIGNATURE":              signature,
				"STUB_STATUS":                 "Accepted",
			}
			for k, v := range tc.env {
				env[k] = v
			}
			cmd := exec.Command(bash, script, binary)
			for k, v := range env {
				cmd.Env = append(cmd.Env, k+"="+v)
			}
			var out bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &out
			runErr := cmd.Run()

			if tc.wantErr == "" && runErr != nil {
				t.Fatalf("want success, got %v:\n%s", runErr, out.String())
			}
			if tc.wantErr != "" {
				if runErr == nil {
					t.Fatalf("want failure containing %q, got success:\n%s", tc.wantErr, out.String())
				}
				if !strings.Contains(out.String(), tc.wantErr) {
					t.Fatalf("want output containing %q, got:\n%s", tc.wantErr, out.String())
				}
			}
			if strings.Contains(out.String(), "app-specific-secret") {
				t.Fatalf("output leaked the app-specific password:\n%s", out.String())
			}
			got, err := os.ReadFile(binary)
			if err != nil {
				t.Fatal(err)
			}
			if signed := bytes.HasSuffix(got, []byte("signed")); signed != tc.wantSigned {
				t.Fatalf("binary signed = %v, want %v", signed, tc.wantSigned)
			}
		})
	}
}
