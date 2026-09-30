// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package demo_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDemoChargeScript(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}

	binary := filepath.Join(repoRoot, "straddle")
	_, hadBinary := os.Stat(binary)

	t.Cleanup(func() {
		if os.IsNotExist(hadBinary) {
			_ = os.Remove(binary)
		}
	})

	script := filepath.Join(repoRoot, "demo", "demo-charge.sh")
	cmd := exec.Command(bash, script)
	cmd.Dir = repoRoot

	tempConfig := t.TempDir()
	cmd.Env = append(os.Environ(),
		"STRADDLE_CONFIG="+filepath.Join(tempConfig, "config.toml"),
		"STRADDLE_PLATFORM_CONFIG="+filepath.Join(tempConfig, "platform.toml"),
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("demo-charge.sh failed: %v\noutput:\n%s", err, string(out))
	}

	outStr := string(out)
	if !strings.Contains(outStr, "dry run - no request sent") {
		t.Fatalf("demo-charge.sh missing dry run output:\n%s", outStr)
	}
	if !strings.Contains(outStr, "pk_demo") {
		t.Fatalf("demo-charge.sh missing paykey in output:\n%s", outStr)
	}
}

func TestDemoTapeTemplate(t *testing.T) {
	content, err := os.ReadFile("demo.tape.tmpl")
	if err != nil {
		t.Fatalf("read demo.tape.tmpl: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "{{CUSTOMER_ID}}") {
		t.Errorf("demo.tape.tmpl missing {{CUSTOMER_ID}} placeholder")
	}
	if !strings.Contains(s, "{{REPO_DIR}}") {
		t.Errorf("demo.tape.tmpl missing {{REPO_DIR}} placeholder")
	}
}
