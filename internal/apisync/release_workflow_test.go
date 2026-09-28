// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package apisync_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReleaseWorkflowRoutesTagPushesAndNPMRecoverySeparately(t *testing.T) {
	t.Parallel()

	workflow := readWorkflow(t, filepath.Join(testRepoRoot(t), ".github", "workflows", "release.yml"))
	if got := workflow.On.Push.Tags; len(got) != 1 || got[0] != "v*" {
		t.Fatalf("release tag triggers = %#v", got)
	}
	if input := workflow.On.WorkflowDispatch.Inputs["tag"]; !input.Required || input.Type != "string" {
		t.Fatalf("recovery tag input = %#v, want a required string", input)
	}
	if got := workflow.Jobs["release"].If; got != "github.event_name == 'push'" {
		t.Fatalf("release job condition = %q; a dispatch must never build, sign or publish release assets", got)
	}
	recovery := workflow.Jobs["npm-recovery"]
	if recovery.If != "github.event_name == 'workflow_dispatch' && github.ref == 'refs/heads/main'" {
		t.Fatalf("npm recovery condition = %q", recovery.If)
	}

	// The recovery job verifies, checks out the verified commit and publishes
	// the wrapper from that checkout; nothing else.
	var labels []string
	for _, step := range recovery.Steps {
		label := step.Name
		if label == "" {
			label = step.Uses
		}
		labels = append(labels, label)
	}
	wantSteps := []string{
		"actions/checkout@v7",
		"Verify published release",
		"Check out release source",
		"actions/setup-node@v6",
		"Use npm with trusted publishing support",
		"Publish npm wrapper",
	}
	if !reflect.DeepEqual(labels, wantSteps) {
		t.Fatalf("npm recovery steps = %#v, want %#v", labels, wantSteps)
	}
	steps := stepsByName(recovery)
	source := steps["Check out release source"]
	if source.With["ref"] != "${{ steps.release.outputs.commit }}" || source.With["path"] != "release-source" {
		t.Fatalf("release source checkout = %#v, want the verified tag commit in release-source", source.With)
	}
	publish := steps["Publish npm wrapper"]
	if publish.WorkingDirectory != "release-source/npm" || publish.Env["VERSION"] != "${{ steps.release.outputs.version }}" {
		t.Fatalf("npm publish = dir %q env %#v, want the verified version from release-source/npm", publish.WorkingDirectory, publish.Env)
	}
}

func TestNPMRecoveryVerifiesPublishedReleaseTag(t *testing.T) {
	t.Parallel()

	workflow := readWorkflow(t, filepath.Join(testRepoRoot(t), ".github", "workflows", "release.yml"))
	script := stepsByName(workflow.Jobs["npm-recovery"])["Verify published release"].Run
	repo, released, unmerged := releaseFixtureRepo(t)

	assets := `{"name":"checksums.txt"},` +
		`{"name":"straddle_1.2.3_darwin_amd64.tar.gz"},{"name":"straddle_1.2.3_darwin_arm64.tar.gz"},` +
		`{"name":"straddle_1.2.3_linux_amd64.tar.gz"},{"name":"straddle_1.2.3_linux_arm64.tar.gz"},` +
		`{"name":"straddle_1.2.3_windows_amd64.zip"},{"name":"straddle_1.2.3_windows_arm64.zip"}`
	published := `{"isDraft":false,"isPrerelease":false,"assets":[` + assets + `]}`

	tests := []struct {
		name    string
		tag     string
		release string // gh release view output; empty means no release
		wantErr string
		want    []string
	}{
		{name: "published release on main", tag: "v1.2.3", release: published, want: []string{"version=1.2.3", "commit=" + released}},
		{name: "branch name instead of tag", tag: "main", release: published, wantErr: "is not a vX.Y.Z release tag"},
		{name: "option injection", tag: "--help", release: published, wantErr: "is not a vX.Y.Z release tag"},
		{name: "prerelease tag", tag: "v1.2.3-rc.1", release: published, wantErr: "is not a vX.Y.Z release tag"},
		{name: "missing tag", tag: "v9.9.9", release: published, wantErr: "tag v9.9.9 does not exist"},
		{name: "tag outside main", tag: "v1.2.4", release: published, wantErr: "(" + unmerged + ") is not on main"},
		{name: "tag without release", tag: "v1.2.3", wantErr: "no GitHub release for v1.2.3"},
		{name: "draft release", tag: "v1.2.3", release: strings.Replace(published, `"isDraft":false`, `"isDraft":true`, 1), wantErr: "is a draft or prerelease"},
		{name: "prerelease release", tag: "v1.2.3", release: strings.Replace(published, `"isPrerelease":false`, `"isPrerelease":true`, 1), wantErr: "is a draft or prerelease"},
		{name: "missing archive", tag: "v1.2.3", release: strings.Replace(published, `{"name":"straddle_1.2.3_windows_arm64.zip"}`, `{"name":"notes.txt"}`, 1), wantErr: "are not the six archives plus checksums.txt"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			output := filepath.Join(t.TempDir(), "output")
			out, err := runReleaseScript(t, repo, script, map[string]string{
				"RECOVERY_TAG":  tc.tag,
				"GITHUB_OUTPUT": output,
				"MOCK_RELEASE":  tc.release,
			}, map[string]string{
				"gh": `[ "$*" = "release view $RECOVERY_TAG --json isDraft,isPrerelease,assets" ] || exit 64
[ -n "$MOCK_RELEASE" ] || { echo "release not found" >&2; exit 1; }
printf '%s\n' "$MOCK_RELEASE"`,
			})
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(out, tc.wantErr) {
					t.Fatalf("err = %v, output = %q, want failure containing %q", err, out, tc.wantErr)
				}
				if got := readLinesIfExists(t, output); got != nil {
					t.Fatalf("refused recovery wrote outputs %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("verify release: %v\n%s", err, out)
			}
			if got := readLines(t, output); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("outputs = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// releaseFixtureRepo builds a repository checked out on main with v1.2.3 on
// main and v1.2.4 on an unmerged branch, returning both tagged commits.
func releaseFixtureRepo(t *testing.T) (dir, released, unmerged string) {
	t.Helper()
	dir = t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	git("commit", "-q", "--allow-empty", "-m", "release")
	git("tag", "-a", "v1.2.3", "-m", "v1.2.3")
	released = git("rev-parse", "HEAD")
	git("switch", "-q", "-c", "side")
	git("commit", "-q", "--allow-empty", "-m", "unreviewed")
	git("tag", "v1.2.4")
	unmerged = git("rev-parse", "HEAD")
	git("switch", "-q", "main")
	return dir, released, unmerged
}

func runReleaseScript(t *testing.T, dir, script string, env, stubs map[string]string) (string, error) {
	t.Helper()
	binDir := filepath.Join(t.TempDir(), "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatalf("create stub bin: %v", err)
	}
	for name, body := range stubs {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte("#!/bin/bash\nset -euo pipefail\n"+body+"\n"), 0o755); err != nil {
			t.Fatalf("write %s stub: %v", name, err)
		}
	}
	cmd := exec.Command("bash", "-c", script)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "PATH="+binDir+":"+os.Getenv("PATH"))
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}
