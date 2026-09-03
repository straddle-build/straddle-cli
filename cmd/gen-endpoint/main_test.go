// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/apisync"
)

func TestRunDriftAgentReportsNoShapeForIdenticalSpecs(t *testing.T) {
	t.Parallel()

	spec := writeCommandSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "ListWidgets",
					"summary": "List widgets"
				}
			}
		}
	}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"drift", "--base", spec, "--head", spec, "--agent"}, &stdout, &stderr); err != nil {
		t.Fatalf("run drift: %v\nstderr: %s", err, stderr.String())
	}

	var result apisync.DriftResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode drift JSON: %v\nstdout: %s", err, stdout.String())
	}
	if !result.NoDrift {
		t.Fatalf("NoDrift = false for identical specs: %#v", result)
	}
	if len(result.SupportedAdditions) != 0 || len(result.Changes) != 0 || len(result.Removals) != 0 || len(result.UnsupportedOperations) != 0 {
		t.Fatalf("identical specs emitted drift shape: %#v", result)
	}
}

func TestRunCandidateStatusValidatesPublisherDigest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	locked := []byte("openapi: 3.1.0\ninfo: {version: 1.2.3}\npaths: {}\n")
	lockPath := filepath.Join(dir, "contract.lock.json")
	lock := map[string]any{
		"schema_version":   1,
		"contract_version": "1.2.3",
		"registry_ref":     "@straddle/straddle-api@1.2.3",
		"published_sha256": commandDigest(locked),
	}
	lockData, err := json.Marshal(lock)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, lockData, 0o644); err != nil {
		t.Fatal(err)
	}
	candidate := []byte("openapi: 3.1.0\ninfo: {version: 1.2.4}\npaths: {}\n")
	candidatePath := filepath.Join(dir, "candidate.yaml")
	if err := os.WriteFile(candidatePath, candidate, 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err = run([]string{
		"candidate-status",
		"--lock", lockPath,
		"--spec", candidatePath,
		"--version", "1.2.4",
		"--published-sha256", commandDigest(candidate),
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run candidate-status: %v\nstderr: %s", err, stderr.String())
	}
	if stdout.String() != "new\n" {
		t.Fatalf("candidate status = %q, want new", stdout.String())
	}

	err = run([]string{
		"candidate-status",
		"--lock", lockPath,
		"--spec", candidatePath,
		"--version", "1.2.4",
		"--published-sha256", strings.Repeat("a", 64),
	}, &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "publisher digest") {
		t.Fatalf("digest mismatch error = %v", err)
	}
}

func TestRunGenerateAgentDryRunReportsRegenerationPlan(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	cliDir := filepath.Join(repo, "internal", "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("mkdir internal/cli: %v", err)
	}
	spec := writeCommandSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/zeta": {
				"post": {
					"tags": ["Zeta"],
					"operationId": "CreateZeta",
					"summary": "Create zeta"
				}
			},
			"/v1/widgets": {
				"get": {
					"tags": ["Widgets"],
					"operationId": "ListWidgets",
					"summary": "List widgets"
				}
			},
			"/v1/alpha": {
				"post": {
					"tags": ["Alpha"],
					"operationId": "CreateAlpha",
					"summary": "Create alpha"
				}
			},
			"/v1/upload": {
				"post": {
					"tags": ["Upload"],
					"operationId": "CreateUpload",
					"summary": "Upload a file",
					"requestBody": {"required": true, "content": {"multipart/form-data": {}}}
				}
			}
		}
	}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"generate", "--spec", spec, "--repo", repo, "--dry-run", "--agent"}, &stdout, &stderr); err != nil {
		t.Fatalf("run generate: %v\nstderr: %s", err, stderr.String())
	}

	var result apisync.GenerateResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode generate JSON: %v\nstdout: %s", err, stdout.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(stdout.Bytes(), &fields); err != nil {
		t.Fatalf("decode generate JSON fields: %v", err)
	}
	for _, name := range []string{"generated", "deleted", "unchanged", "unsupported", "dry_run"} {
		if _, ok := fields[name]; !ok {
			t.Fatalf("generate JSON missing %q: %s", name, stdout.String())
		}
	}
	if len(fields) != 5 {
		t.Fatalf("generate JSON fields = %#v", fields)
	}
	if !result.DryRun {
		t.Fatal("DryRun = false")
	}
	wantGenerated := []string{
		filepath.Join(cliDir, "alpha_create.go"),
		filepath.Join(cliDir, "widgets_list.go"),
		filepath.Join(cliDir, "zeta_create.go"),
	}
	if len(result.Generated) != len(wantGenerated) {
		t.Fatalf("Generated = %#v, want %#v", result.Generated, wantGenerated)
	}
	for i, want := range wantGenerated {
		if result.Generated[i] != want {
			t.Fatalf("Generated = %#v, want deterministic order %#v", result.Generated, wantGenerated)
		}
		if _, err := os.Stat(want); !os.IsNotExist(err) {
			t.Fatalf("dry-run wrote %s, stat err %v", want, err)
		}
	}
	if len(result.UnsupportedOperations) != 1 {
		t.Fatalf("UnsupportedOperations = %#v, want one non-JSON request body", result.UnsupportedOperations)
	}
	unsupported := result.UnsupportedOperations[0]
	if unsupported.Operation.Key != "POST /v1/upload" {
		t.Fatalf("unsupported key = %q, want %q", unsupported.Operation.Key, "POST /v1/upload")
	}
	if !strings.Contains(strings.Join(unsupported.Reasons, ", "), "request body lacks application/json content") {
		t.Fatalf("unsupported reasons = %#v", unsupported.Reasons)
	}
}

func TestRunGenerateWritesDeclarativeEndpoint(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	cliDir := filepath.Join(repo, "internal", "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("mkdir internal/cli: %v", err)
	}
	spec := writeCommandSpec(t, `{
		"openapi": "3.1.0",
		"paths": {
			"/v1/widgets": {
				"post": {
					"tags": ["Widgets"],
					"operationId": "CreateWidgets",
					"summary": "Create widget"
				}
			}
		}
	}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"generate", "--spec", spec, "--repo", repo, "--agent"}, &stdout, &stderr); err != nil {
		t.Fatalf("run generate: %v\nstderr: %s", err, stderr.String())
	}

	generatedPath := filepath.Join(cliDir, "widgets_create.go")
	content, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}
	got := string(content)
	for _, want := range []string{
		`registerGeneratedEndpoint("widgets.create", newWidgetsCreateCmd)`,
		"registerSurface(surface.Surface{",
		`"straddle:operation-id": "CreateWidgets"`,
		`applyOverlay("widgets.create", cmd)`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("generated file missing %q:\n%s", want, got)
		}
	}
}

func TestRunGenerateUnsupportedExitBehavior(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		agent   bool
		wantErr bool
	}{
		{name: "human exits non-zero", wantErr: true},
		{name: "agent reports and exits zero", agent: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := t.TempDir()
			if err := os.MkdirAll(filepath.Join(repo, "internal", "cli"), 0o755); err != nil {
				t.Fatalf("mkdir internal/cli: %v", err)
			}
			spec := writeCommandSpec(t, `{
				"openapi": "3.1.0",
				"paths": {
					"/v1/upload": {
						"post": {
							"tags": ["Upload"],
							"operationId": "CreateUpload",
							"requestBody": {"required": true, "content": {"multipart/form-data": {}}}
						}
					}
				}
			}`)
			args := []string{"generate", "--spec", spec, "--repo", repo}
			if tc.agent {
				args = append(args, "--agent")
			}
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run(args, &stdout, &stderr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("run generate error = %v, wantErr %t", err, tc.wantErr)
			}
			if tc.wantErr && !strings.Contains(err.Error(), "generation found 1 unsupported operations") {
				t.Fatalf("run generate error = %q", err)
			}
			if !strings.Contains(stdout.String(), "unsupported") {
				t.Fatalf("generate output missing unsupported operations: %s", stdout.String())
			}
		})
	}
}

func TestRunGenerateRejectsRemovedFlags(t *testing.T) {
	t.Parallel()

	for _, flagName := range []string{"--drift", "--supported-additions"} {
		t.Run(flagName, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := run([]string{"generate", flagName}, &stdout, &stderr)
			if err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
				t.Fatalf("run generate %s error = %v", flagName, err)
			}
		})
	}
}

func TestRunCheckReviewDriftAllowsRemovedAndRenamedOperations(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	cliDir := filepath.Join(repo, "internal", "cli")
	if err := os.MkdirAll(cliDir, 0o755); err != nil {
		t.Fatalf("mkdir internal/cli: %v", err)
	}
	annotations := map[string]string{
		"removed.go": "package cli\n\nvar removed = map[string]string{\"straddle:endpoint\": \"widgets.removed\", \"straddle:operation-id\": \"removedWidget\", \"straddle:method\": \"DELETE\", \"straddle:path\": \"/v1/widgets/{id}\"}\n",
		"renamed.go": "package cli\n\nvar renamed = map[string]string{\"straddle:endpoint\": \"widgets.get\", \"straddle:operation-id\": \"getWidget\", \"straddle:method\": \"GET\", \"straddle:path\": \"/v1/widgets/{id}\"}\n",
	}
	for name, content := range annotations {
		if err := os.WriteFile(filepath.Join(cliDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	spec := writeCommandSpec(t, "{\n\"openapi\":\"3.1.0\",\n\"paths\":{\"/v1/widgets/{id}\":{\"get\":{\"tags\":[\"Widgets\"],\"operationId\":\"fetchWidget\",\"parameters\":[{\"name\":\"id\",\"in\":\"path\",\"required\":true,\"schema\":{\"type\":\"string\"}}]}}}\n}")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := run([]string{"check", "--spec", spec, "--repo", repo, "--agent"}, &stdout, &stderr); err == nil {
		t.Fatal("strict coverage check accepted removed and renamed operations")
	}

	stdout.Reset()
	stderr.Reset()
	if err := run([]string{"check", "--spec", spec, "--repo", repo, "--review-drift", "--agent"}, &stdout, &stderr); err != nil {
		t.Fatalf("review-drift coverage check: %v\nstderr: %s", err, stderr.String())
	}
	var result apisync.CheckResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode coverage result: %v", err)
	}
	if len(result.Extra) != 1 || len(result.OperationIDMismatches) != 1 || len(result.Missing) != 0 {
		t.Fatalf("coverage result = %#v, want one removal and one operationId mismatch", result)
	}
}

func writeCommandSpec(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write spec: %v", err)
	}
	return path
}

func commandDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
