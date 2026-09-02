// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
package apisync

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyContractLockBindsExactScalarBytesAndVersion(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	spec := []byte("openapi: 3.1.0\ninfo: {version: 1.2.3}\npaths: {}\n")
	lock := []byte(`{"schema_version":1,"contract_version":"1.2.3","registry_ref":"@straddle/straddle-api@1.2.3","published_sha256":"` + digestBytes(spec) + `"}`)
	lockPath := writeLockFixture(t, dir, "lock.json", lock)
	specPath := writeLockFixture(t, dir, "spec.yaml", spec)

	if _, err := VerifyContractLock(lockPath, specPath); err != nil {
		t.Fatalf("VerifyContractLock: %v", err)
	}
	if err := os.WriteFile(specPath, append(spec, '#'), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyContractLock(lockPath, specPath); err == nil {
		t.Fatal("tampered spec accepted")
	}
}

func TestInspectContractCandidateValidatesPublisherEvidenceAndVersionOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	locked := []byte("openapi: 3.1.0\ninfo: {version: 1.2.3}\npaths: {}\n")
	lock := []byte(`{"schema_version":1,"contract_version":"1.2.3","registry_ref":"@straddle/straddle-api@1.2.3","published_sha256":"` + digestBytes(locked) + `"}`)
	lockPath := writeLockFixture(t, dir, "lock.json", lock)

	tests := []struct {
		name        string
		spec        []byte
		expectation ContractCandidateExpectation
		want        ContractCandidateStatus
		wantErr     string
	}{
		{name: "current", spec: locked, expectation: ContractCandidateExpectation{Version: "1.2.3", PublishedSHA256: digestBytes(locked)}, want: ContractCandidateCurrent},
		{name: "new", spec: []byte("openapi: 3.1.0\ninfo: {version: 1.2.4}\npaths: {}\n"), expectation: ContractCandidateExpectation{Version: "1.2.4"}, want: ContractCandidateNew},
		{name: "stale pull request", spec: []byte("openapi: 3.1.0\ninfo: {version: 1.1.9}\npaths: {}\n"), expectation: ContractCandidateExpectation{Version: "1.1.9"}, wantErr: "older than pinned version"},
		{name: "mutated immutable version", spec: []byte("openapi: 3.1.0\ninfo: {version: 1.2.3}\npaths: {/v1/widgets: {}}\n"), expectation: ContractCandidateExpectation{Version: "1.2.3"}, wantErr: "changed bytes for an immutable version"},
		{name: "wrong declared version", spec: []byte("openapi: 3.1.0\ninfo: {version: 1.2.4}\npaths: {}\n"), expectation: ContractCandidateExpectation{Version: "1.2.5"}, wantErr: "does not match expected version"},
		{name: "wrong publisher digest", spec: []byte("openapi: 3.1.0\ninfo: {version: 1.2.4}\npaths: {}\n"), expectation: ContractCandidateExpectation{Version: "1.2.4", PublishedSHA256: strings.Repeat("a", 64)}, wantErr: "does not match publisher digest"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			specPath := writeLockFixture(t, dir, strings.ReplaceAll(test.name, " ", "-")+".yaml", test.spec)
			got, err := InspectContractCandidate(lockPath, specPath, test.expectation)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("InspectContractCandidate() = %q, %v, want %q", got, err, test.want)
			}
		})
	}
}

func TestInspectContractCandidateComparesLargeSemverComponentsWithoutOverflow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	lockedVersion := "999999999999999999999999999999.0.0"
	newVersion := "1000000000000000000000000000000.0.0"
	locked := []byte("openapi: 3.1.0\ninfo: {version: " + lockedVersion + "}\npaths: {}\n")
	lock := []byte(`{"schema_version":1,"contract_version":"` + lockedVersion + `","registry_ref":"@straddle/straddle-api@` + lockedVersion + `","published_sha256":"` + digestBytes(locked) + `"}`)
	lockPath := writeLockFixture(t, dir, "large-lock.json", lock)
	newSpec := []byte("openapi: 3.1.0\ninfo: {version: " + newVersion + "}\npaths: {}\n")
	specPath := writeLockFixture(t, dir, "large-spec.yaml", newSpec)

	got, err := InspectContractCandidate(lockPath, specPath, ContractCandidateExpectation{Version: newVersion})
	if err != nil || got != ContractCandidateNew {
		t.Fatalf("InspectContractCandidate() = %q, %v, want %q", got, err, ContractCandidateNew)
	}
}

func TestValidateContractLockRejectsMutableReferencesAndUnknownFields(t *testing.T) {
	t.Parallel()
	path := writeLockFixture(t, t.TempDir(), "lock.json", []byte(`{"schema_version":1,"contract_version":"latest","extra":true}`))
	if _, err := VerifyContractLock(path, path); err == nil {
		t.Fatal("mutable malformed lock accepted")
	}
}

func writeLockFixture(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
