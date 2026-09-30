// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package scripts

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitIgnoreToolCaches(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available")
	}

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(git, "check-ignore", "-v", "cmd/straddle/.impeccable/hook.cache.json")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected cmd/straddle/.impeccable/hook.cache.json to be ignored by .gitignore: %v\noutput:\n%s", err, string(out))
	}
	if !strings.Contains(string(out), ".impeccable") {
		t.Fatalf("expected git check-ignore to match .impeccable pattern, got:\n%s", string(out))
	}

	nonIgnoredCmd := exec.Command(git, "check-ignore", "-v", "cmd/straddle/main.go")
	nonIgnoredCmd.Dir = repoRoot
	if err := nonIgnoredCmd.Run(); err == nil {
		t.Fatalf("expected cmd/straddle/main.go to NOT be ignored by .gitignore")
	}
}
