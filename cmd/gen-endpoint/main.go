// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/straddle-build/straddle-cli/internal/apisync"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: gen-endpoint <candidate-status|check|drift|generate|surfaces|verify-lock|version> [flags]")
	}
	switch args[0] {
	case "candidate-status":
		return runCandidateStatus(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "drift":
		return runDrift(args[1:], stdout, stderr)
	case "generate":
		return runGenerate(args[1:], stdout, stderr)
	case "surfaces":
		return runSurfaces(args[1:], stdout, stderr)
	case "verify-lock":
		return runVerifyLock(args[1:], stdout, stderr)
	case "version":
		return runVersion(args[1:], stdout, stderr)
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func runCandidateStatus(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("candidate-status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lockPath := fs.String("lock", "contract.lock.json", "contract lock")
	specPath := fs.String("spec", "", "candidate Scalar contract")
	version := fs.String("version", "", "expected exact contract version")
	publishedSHA256 := fs.String("published-sha256", "", "optional publisher-provided digest")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *specPath == "" || *version == "" {
		return errors.New("candidate-status requires --spec and --version")
	}
	status, err := apisync.InspectContractCandidate(*lockPath, *specPath, apisync.ContractCandidateExpectation{
		Version:         *version,
		PublishedSHA256: *publishedSHA256,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, status)
	return err
}

func runVerifyLock(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("verify-lock", flag.ContinueOnError)
	fs.SetOutput(stderr)
	lockPath := fs.String("lock", "contract.lock.json", "contract lock")
	specPath := fs.String("spec", "spec.yaml", "pinned Scalar contract")
	if err := fs.Parse(args); err != nil {
		return err
	}
	lock, err := apisync.VerifyContractLock(*lockPath, *specPath)
	if err != nil {
		return err
	}
	return writeJSON(stdout, lock)
}

func runVersion(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.SetOutput(stderr)
	specPath := fs.String("spec", "", "OpenAPI document")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *specPath == "" {
		return errors.New("version requires --spec")
	}
	version, err := apisync.LoadSpecVersion(*specPath)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(stdout, version)
	return err
}

func runCheck(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(stderr)
	specPath := fs.String("spec", "spec.yaml", "OpenAPI spec lockfile")
	repo := fs.String("repo", ".", "repository root")
	agent := fs.Bool("agent", false, "emit JSON")
	reviewDrift := fs.Bool("review-drift", false, "allow removed or renamed operations as review evidence")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := apisync.CheckSpecAgainstRepo(*specPath, *repo)
	if err != nil {
		return err
	}
	if *agent {
		if err := writeJSON(stdout, result); err != nil {
			return err
		}
	} else {
		writeCheckSummary(stdout, result)
	}
	if !result.OK && (!*reviewDrift || result.HasBlockingIssues()) {
		return fmt.Errorf("endpoint coverage check failed: %d missing, %d extra, %d duplicate groups, %d invalid annotations, %d operationId mismatches", len(result.Missing), len(result.Extra), len(result.DuplicateAnnotations), len(result.InvalidAnnotations), len(result.OperationIDMismatches))
	}
	return nil
}

func runDrift(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("drift", flag.ContinueOnError)
	fs.SetOutput(stderr)
	base := fs.String("base", "", "base OpenAPI spec")
	head := fs.String("head", "", "head OpenAPI spec")
	target := fs.String("target", "", "alias for --head")
	repo := fs.String("repo", ".", "repository root, reserved for workflow callers")
	agent := fs.Bool("agent", false, "emit JSON")
	outPath := fs.String("out", "", "write JSON drift result to this file")
	summaryPath := fs.String("summary", "", "write text summary to this file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	_ = repo
	if *head == "" && *target != "" {
		*head = *target
	}
	if *base == "" || *head == "" {
		return errors.New("drift requires --base and --head")
	}
	result, err := apisync.DriftSpecs(*base, *head)
	if err != nil {
		return err
	}
	if *outPath != "" {
		if err := writeJSONFile(*outPath, result); err != nil {
			return err
		}
	}
	if *summaryPath != "" {
		if err := os.WriteFile(*summaryPath, []byte(driftSummary(result)), 0o600); err != nil {
			return err
		}
	}
	if *agent {
		return writeJSON(stdout, result)
	}
	_, err = io.WriteString(stdout, driftSummary(result))
	return err
}

func runSurfaces(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("surfaces", flag.ContinueOnError)
	fs.SetOutput(stderr)
	specPath := fs.String("spec", "", "OpenAPI spec")
	fs.Bool("agent", false, "emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *specPath == "" {
		return errors.New("surfaces requires --spec")
	}
	surfaces, unsupported, err := apisync.DeriveSurfaces(*specPath)
	if err != nil {
		return err
	}
	sort.Slice(surfaces, func(i, j int) bool {
		if surfaces[i].Path != surfaces[j].Path {
			return surfaces[i].Path < surfaces[j].Path
		}
		return surfaces[i].Method < surfaces[j].Method
	})
	if err := writeJSON(stdout, surfaces); err != nil {
		return err
	}
	if len(unsupported) == 0 {
		return nil
	}
	summaries := make([]string, len(unsupported))
	for i, operation := range unsupported {
		summaries[i] = operation.Operation.Key + ": " + strings.Join(operation.Reasons, ", ")
	}
	return fmt.Errorf("surface derivation found %d unsupported operations: %s", len(unsupported), strings.Join(summaries, "; "))
}

func runGenerate(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	specPath := fs.String("spec", "spec.yaml", "OpenAPI spec to generate from")
	repo := fs.String("repo", ".", "repository root")
	agent := fs.Bool("agent", false, "emit JSON")
	dryRun := fs.Bool("dry-run", false, "show file changes without applying them")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := apisync.GenerateAll(*specPath, *repo, *dryRun)
	if err != nil {
		return err
	}
	if *agent {
		return writeJSON(stdout, result)
	}
	writeGenerateSummary(stdout, result)
	if len(result.UnsupportedOperations) == 0 {
		return nil
	}
	summaries := make([]string, len(result.UnsupportedOperations))
	for i, operation := range result.UnsupportedOperations {
		summaries[i] = operation.Operation.Key + ": " + strings.Join(operation.Reasons, ", ")
	}
	return fmt.Errorf("generation found %d unsupported operations: %s", len(result.UnsupportedOperations), strings.Join(summaries, "; "))
}

func writeJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func writeJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil && filepath.Dir(path) != "." {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func writeCheckSummary(w io.Writer, result apisync.CheckResult) {
	status := "ok"
	if !result.OK {
		status = "failed"
	}
	fmt.Fprintf(w, "endpoint coverage: %s\n", status)
	fmt.Fprintf(w, "spec_operations: %d\n", result.SpecOperations)
	fmt.Fprintf(w, "annotated_endpoints: %d\n", result.AnnotatedEndpoints)
	fmt.Fprintf(w, "missing: %d\n", len(result.Missing))
	fmt.Fprintf(w, "extra: %d\n", len(result.Extra))
	fmt.Fprintf(w, "duplicate_annotations: %d\n", len(result.DuplicateAnnotations))
	fmt.Fprintf(w, "invalid_annotations: %d\n", len(result.InvalidAnnotations))
	fmt.Fprintf(w, "operation_id_mismatches: %d\n", len(result.OperationIDMismatches))
	fmt.Fprintf(w, "unsupported_operations: %d\n", len(result.UnsupportedOperations))
}

func driftSummary(result apisync.DriftResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "api drift summary\n")
	fmt.Fprintf(&b, "base_operations: %d\n", result.BaseOperations)
	fmt.Fprintf(&b, "head_operations: %d\n", result.HeadOperations)
	fmt.Fprintf(&b, "supported_additions: %d\n", len(result.SupportedAdditions))
	fmt.Fprintf(&b, "changes: %d\n", len(result.Changes))
	fmt.Fprintf(&b, "removals: %d\n", len(result.Removals))
	fmt.Fprintf(&b, "unsupported_operations: %d\n", len(result.UnsupportedOperations))
	fmt.Fprintf(&b, "no_drift: %t\n", result.NoDrift)
	return b.String()
}

func writeGenerateSummary(w io.Writer, result apisync.GenerateResult) {
	fmt.Fprintf(w, "generated: %d\n", len(result.Generated))
	fmt.Fprintf(w, "deleted: %d\n", len(result.Deleted))
	fmt.Fprintf(w, "unchanged: %d\n", len(result.Unchanged))
	fmt.Fprintf(w, "unsupported: %d\n", len(result.UnsupportedOperations))
	fmt.Fprintf(w, "dry_run: %t\n", result.DryRun)
}
