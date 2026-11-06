// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
//
// Tests for the account-scope policy fix on `straddle tail` and
// `straddle import`. Both commands make real API calls but carry no
// straddle:path/straddle:method annotations, so before the fix
// straddleAccountPolicyTarget fell through to ("", "", false), Classify
// short-circuited to Forbid, and an explicit --account was spuriously
// rejected while a sticky current account was silently dropped. The fix
// resolves the policy target from the resource argument via
// syncResourcePath, matching the per-path acceptsAccountHeader lookup
// that annotated sibling commands use.
package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/straddle-build/straddle-cli/internal/straddleacct"
)

// newTailPolicyTestCmd builds a bare `tail` cobra command (no straddle:path /
// straddle:method annotations, only mcp:read-only) bound to the account flag,
// mirroring the real newTailCmd shape that resolveStraddleAccount sees.
func newTailPolicyTestCmd() (*cobra.Command, *rootFlags) {
	f := &rootFlags{}
	cmd := &cobra.Command{
		Use:         "tail",
		Annotations: map[string]string{"mcp:read-only": "true"},
	}
	cmd.Flags().StringVar(&f.straddleAccount, "account", "", "")
	return cmd, f
}

// newImportPolicyTestCmd builds a bare `import` cobra command bound to the
// account flag, mirroring the real newImportCmd shape.
func newImportPolicyTestCmd() (*cobra.Command, *rootFlags) {
	f := &rootFlags{}
	cmd := &cobra.Command{
		Use: "import",
	}
	cmd.Flags().StringVar(&f.straddleAccount, "account", "", "")
	return cmd, f
}

// TestStraddleAccountPolicyTargetTail verifies the policy target is resolvable
// for tail from the resource argument, instead of falling through to
// ("", "", false) which short-circuits Classify to Forbid. The path comes from
// syncResourcePath (so "funding-events" -> "/v1/funding_events"), the method is
// GET, and acceptsHeader matches the registered surface for that path+GET.
func TestStraddleAccountPolicyTargetTail(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		args        []string
		wantPath    string
		wantMethod  string
		wantAccepts bool
	}{
		{"customers header-capable", []string{"customers"}, "/v1/customers", "GET", true},
		{"payments header-capable", []string{"payments"}, "/v1/payments", "GET", true},
		{"paykeys header-capable", []string{"paykeys"}, "/v1/paykeys", "GET", true},
		{"funding-events header-capable", []string{"funding-events"}, "/v1/funding_events", "GET", true},
		{"linked-bank-accounts no header", []string{"linked-bank-accounts"}, "/v1/linked_bank_accounts", "GET", false},
		{"accounts no header", []string{"accounts"}, "/v1/accounts", "GET", false},
		{"organizations no header", []string{"organizations"}, "/v1/organizations", "GET", false},
		{"representatives no header", []string{"representatives"}, "/v1/representatives", "GET", false},
		{"unknown resource falls through", []string{"nonexistent"}, "", "", false},
		{"no args falls through", nil, "", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, _ := newTailPolicyTestCmd()
			path, method, accepts := straddleAccountPolicyTarget(cmd, tc.args)
			if path != tc.wantPath || method != tc.wantMethod || accepts != tc.wantAccepts {
				t.Errorf("straddleAccountPolicyTarget(tail %v) = (%q, %q, %v), want (%q, %q, %v)",
					tc.args, path, method, accepts, tc.wantPath, tc.wantMethod, tc.wantAccepts)
			}
		})
	}
}

// TestStraddleAccountPolicyTargetImport verifies the policy target is resolvable
// for import from the resource argument. The method is POST and acceptsHeader
// matches the registered surface (or the fallback) for that path+POST.
func TestStraddleAccountPolicyTargetImport(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		args        []string
		wantPath    string
		wantMethod  string
		wantAccepts bool
	}{
		{"customers header-capable", []string{"customers"}, "/v1/customers", "POST", true},
		{"paykeys fallback header-capable", []string{"paykeys"}, "/v1/paykeys", "POST", true},
		{"payments fallback header-capable", []string{"payments"}, "/v1/payments", "POST", true},
		{"funding-events fallback header-capable", []string{"funding-events"}, "/v1/funding_events", "POST", true},
		{"accounts no header", []string{"accounts"}, "/v1/accounts", "POST", false},
		{"organizations no header", []string{"organizations"}, "/v1/organizations", "POST", false},
		{"representatives no header", []string{"representatives"}, "/v1/representatives", "POST", false},
		{"linked-bank-accounts no header", []string{"linked-bank-accounts"}, "/v1/linked_bank_accounts", "POST", false},
		{"unknown resource falls through", []string{"nonexistent"}, "", "", false},
		{"no args falls through", nil, "", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cmd, _ := newImportPolicyTestCmd()
			path, method, accepts := straddleAccountPolicyTarget(cmd, tc.args)
			if path != tc.wantPath || method != tc.wantMethod || accepts != tc.wantAccepts {
				t.Errorf("straddleAccountPolicyTarget(import %v) = (%q, %q, %v), want (%q, %q, %v)",
					tc.args, path, method, accepts, tc.wantPath, tc.wantMethod, tc.wantAccepts)
			}
		})
	}
}

// isolateAccountPlatformConfig isolates the platform config to a temp dir and
// saves the given context, so resolveStraddleAccount reads a deterministic
// integration type and current account.
func isolateAccountPlatformConfig(t *testing.T, ctx straddleacct.Context) {
	t.Helper()
	isolateAPIConfig(t)
	if err := straddleacct.SaveContext(ctx); err != nil {
		t.Fatalf("save platform context: %v", err)
	}
}

// TestResolveStraddleAccountTailAcceptsAccount exercises resolveStraddleAccount
// directly on a tail-shaped command across integration types. This is the
// primary regression test: before the fix, every case returned ("", "", false)
// from straddleAccountPolicyTarget, Classify short-circuited to Forbid, and
// the explicit --account was rejected / the sticky was dropped.
func TestResolveStraddleAccountTailAcceptsAccount(t *testing.T) {
	// SaaS with --account: customers GET is Allow, so the flag value resolves.
	t.Run("saas_explicit_account_customers", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeSaaS})
		cmd, f := newTailPolicyTestCmd()
		if err := cmd.Flags().Set("account", "acct_flag"); err != nil {
			t.Fatal(err)
		}
		f.straddleAccount = "acct_flag"
		if err := resolveStraddleAccount(cmd, f, []string{"customers"}); err != nil {
			t.Fatalf("resolveStraddleAccount: %v", err)
		}
		if f.straddleAccountResolved != "acct_flag" {
			t.Errorf("resolved = %q, want acct_flag", f.straddleAccountResolved)
		}
	})

	// SaaS with sticky: customers GET is Allow, so the sticky value resolves
	// (no silent drop).
	t.Run("saas_sticky_customers", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{
			IntegrationType: straddleacct.TypeSaaS,
			CurrentAccount:  "acct_sticky",
		})
		cmd, f := newTailPolicyTestCmd()
		if err := resolveStraddleAccount(cmd, f, []string{"customers"}); err != nil {
			t.Fatalf("resolveStraddleAccount: %v", err)
		}
		if f.straddleAccountResolved != "acct_sticky" {
			t.Errorf("resolved = %q, want acct_sticky (sticky must not be dropped)", f.straddleAccountResolved)
		}
	})

	// Account integration type: customers GET is Forbid (a direct account has
	// no acting-account context), so an explicit --account must still error.
	t.Run("account_type_explicit_account_errors", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeAccount})
		cmd, f := newTailPolicyTestCmd()
		if err := cmd.Flags().Set("account", "acct_flag"); err != nil {
			t.Fatal(err)
		}
		f.straddleAccount = "acct_flag"
		err := resolveStraddleAccount(cmd, f, []string{"customers"})
		if err == nil {
			t.Fatal("account type with --account: error = nil, want forbidden error")
		}
		if !strings.Contains(err.Error(), "remove --account") {
			t.Fatalf("error = %q, want it to contain \"remove --account\"", err.Error())
		}
		if got := ExitCode(err); got != 2 {
			t.Fatalf("ExitCode = %d, want 2", got)
		}
	})

	// Marketplace: customers is customer-owned, so Forbid; --account must error.
	t.Run("marketplace_customers_explicit_account_errors", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeMarketplace})
		cmd, f := newTailPolicyTestCmd()
		if err := cmd.Flags().Set("account", "acct_flag"); err != nil {
			t.Fatal(err)
		}
		f.straddleAccount = "acct_flag"
		err := resolveStraddleAccount(cmd, f, []string{"customers"})
		if err == nil {
			t.Fatal("marketplace customers with --account: error = nil, want forbidden error")
		}
		if !strings.Contains(err.Error(), "remove --account") {
			t.Fatalf("error = %q, want it to contain \"remove --account\"", err.Error())
		}
	})

	// Marketplace: payments is not customer-owned, so Allow; --account resolves.
	t.Run("marketplace_payments_explicit_account_resolves", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeMarketplace})
		cmd, f := newTailPolicyTestCmd()
		if err := cmd.Flags().Set("account", "acct_flag"); err != nil {
			t.Fatal(err)
		}
		f.straddleAccount = "acct_flag"
		if err := resolveStraddleAccount(cmd, f, []string{"payments"}); err != nil {
			t.Fatalf("resolveStraddleAccount: %v", err)
		}
		if f.straddleAccountResolved != "acct_flag" {
			t.Errorf("resolved = %q, want acct_flag", f.straddleAccountResolved)
		}
	})

	// Unknown resource: falls through to ("", "", false) -> Forbid; but the
	// flag is empty so no error, and resolved is "" (RunE will error later).
	t.Run("unknown_resource_no_account", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeSaaS})
		cmd, f := newTailPolicyTestCmd()
		if err := resolveStraddleAccount(cmd, f, []string{"nonexistent"}); err != nil {
			t.Fatalf("resolveStraddleAccount: %v", err)
		}
		if f.straddleAccountResolved != "" {
			t.Errorf("resolved = %q, want \"\" for unknown resource", f.straddleAccountResolved)
		}
	})
}

// TestResolveStraddleAccountImportPolicy exercises resolveStraddleAccount
// directly on an import-shaped command. customers POST under SaaS is Require
// (saasCreateScoped), so an explicit account resolves and a missing account
// errors. accounts POST is Forbid (no header capability), so --account errors.
func TestResolveStraddleAccountImportPolicy(t *testing.T) {
	// SaaS with --account on customers: Require, resolves the flag value.
	t.Run("saas_explicit_account_customers", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeSaaS})
		cmd, f := newImportPolicyTestCmd()
		if err := cmd.Flags().Set("account", "acct_flag"); err != nil {
			t.Fatal(err)
		}
		f.straddleAccount = "acct_flag"
		if err := resolveStraddleAccount(cmd, f, []string{"customers"}); err != nil {
			t.Fatalf("resolveStraddleAccount: %v", err)
		}
		if f.straddleAccountResolved != "acct_flag" {
			t.Errorf("resolved = %q, want acct_flag", f.straddleAccountResolved)
		}
	})

	// SaaS with sticky on customers: Require, resolves the sticky value.
	t.Run("saas_sticky_customers", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{
			IntegrationType: straddleacct.TypeSaaS,
			CurrentAccount:  "acct_sticky",
		})
		cmd, f := newImportPolicyTestCmd()
		if err := resolveStraddleAccount(cmd, f, []string{"customers"}); err != nil {
			t.Fatalf("resolveStraddleAccount: %v", err)
		}
		if f.straddleAccountResolved != "acct_sticky" {
			t.Errorf("resolved = %q, want acct_sticky", f.straddleAccountResolved)
		}
	})

	// SaaS without account on customers: Require, no value -> error.
	t.Run("saas_no_account_customers_errors", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeSaaS})
		cmd, f := newImportPolicyTestCmd()
		err := resolveStraddleAccount(cmd, f, []string{"customers"})
		if err == nil {
			t.Fatal("saas import customers without account: error = nil, want required error")
		}
		if !strings.Contains(err.Error(), "an account id is required") {
			t.Fatalf("error = %q, want it to contain \"an account id is required\"", err.Error())
		}
		if got := ExitCode(err); got != 2 {
			t.Fatalf("ExitCode = %d, want 2", got)
		}
	})

	// SaaS with --account on accounts: Forbid (accounts POST has no header
	// capability), so --account must error.
	t.Run("saas_explicit_account_accounts_errors", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeSaaS})
		cmd, f := newImportPolicyTestCmd()
		if err := cmd.Flags().Set("account", "acct_flag"); err != nil {
			t.Fatal(err)
		}
		f.straddleAccount = "acct_flag"
		err := resolveStraddleAccount(cmd, f, []string{"accounts"})
		if err == nil {
			t.Fatal("saas import accounts with --account: error = nil, want forbidden error")
		}
		if !strings.Contains(err.Error(), "remove --account") {
			t.Fatalf("error = %q, want it to contain \"remove --account\"", err.Error())
		}
	})

	// Account integration type: Forbid, --account must error.
	t.Run("account_type_explicit_account_errors", func(t *testing.T) {
		isolateAccountPlatformConfig(t, straddleacct.Context{IntegrationType: straddleacct.TypeAccount})
		cmd, f := newImportPolicyTestCmd()
		if err := cmd.Flags().Set("account", "acct_flag"); err != nil {
			t.Fatal(err)
		}
		f.straddleAccount = "acct_flag"
		err := resolveStraddleAccount(cmd, f, []string{"customers"})
		if err == nil {
			t.Fatal("account type import with --account: error = nil, want forbidden error")
		}
		if !strings.Contains(err.Error(), "remove --account") {
			t.Fatalf("error = %q, want it to contain \"remove --account\"", err.Error())
		}
	})
}

// recordingServer returns an httptest server that records the
// Straddle-Account-Id header and request path/method on each request.
func recordingServer(t *testing.T) (*httptest.Server, *sync.Mutex, *string, *int) {
	t.Helper()
	var mu sync.Mutex
	var gotHeader string
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		gotHeader = r.Header.Get(straddleacct.Header)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"created"}`))
	}))
	t.Cleanup(server.Close)
	return server, &mu, &gotHeader, &requestCount
}

// TestImportAccountHeaderSentUnderSaaS is the end-to-end test: under SaaS,
// `import customers --account acct_x` sends POST /v1/customers with
// Straddle-Account-Id: acct_x. Before the fix, the policy target was
// ("", "", false) -> Forbid, so --account was rejected with "remove --account"
// and no request was sent.
func TestImportAccountHeaderSentUnderSaaS(t *testing.T) {
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("STRADDLE_API_KEY", "test_key")
	if err := straddleacct.SaveContext(straddleacct.Context{IntegrationType: straddleacct.TypeSaaS}); err != nil {
		t.Fatal(err)
	}
	server, mu, gotHeader, requestCount := recordingServer(t)
	t.Setenv("STRADDLE_BASE_URL", server.URL)

	input := filepath.Join(t.TempDir(), "customers.jsonl")
	if err := os.WriteFile(input, []byte("{\"name\":\"One\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := runRootForAPITest(t, []string{
		"import", "customers", "--input", input, "--account", "acct_x", "--json",
	}, "")
	if err != nil {
		t.Fatalf("import with --account under SaaS failed: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if *requestCount != 1 {
		t.Fatalf("HTTP requests = %d, want 1", *requestCount)
	}
	if *gotHeader != "acct_x" {
		t.Fatalf("Straddle-Account-Id = %q, want acct_x", *gotHeader)
	}
}

// TestImportPathNormalizedToV1 verifies the import request targets
// /v1/customers (the versioned, policy-matching path) not /customers.
func TestImportPathNormalizedToV1(t *testing.T) {
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("STRADDLE_API_KEY", "test_key")
	if err := straddleacct.SaveContext(straddleacct.Context{IntegrationType: straddleacct.TypeSaaS}); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPath = r.URL.Path
		mu.Unlock()
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"created"}`))
	}))
	defer server.Close()
	t.Setenv("STRADDLE_BASE_URL", server.URL)

	input := filepath.Join(t.TempDir(), "customers.jsonl")
	if err := os.WriteFile(input, []byte("{\"name\":\"One\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := runRootForAPITest(t, []string{
		"import", "customers", "--input", input, "--account", "acct_x", "--json",
	}, "")
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if gotPath != "/v1/customers" {
		t.Fatalf("request path = %q, want /v1/customers", gotPath)
	}
}

// TestImportForbiddenAccountFlagErrors verifies that under the account
// integration type, `import customers --account acct_x` is rejected with
// "remove --account" (exit code 2) and sends zero HTTP requests — the policy
// correctly forbids the header, and the command never runs.
func TestImportForbiddenAccountFlagErrors(t *testing.T) {
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("STRADDLE_API_KEY", "test_key")
	if err := straddleacct.SaveContext(straddleacct.Context{IntegrationType: straddleacct.TypeAccount}); err != nil {
		t.Fatal(err)
	}
	server, mu, _, requestCount := recordingServer(t)
	t.Setenv("STRADDLE_BASE_URL", server.URL)

	input := filepath.Join(t.TempDir(), "customers.jsonl")
	if err := os.WriteFile(input, []byte("{\"name\":\"One\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := runRootForAPITest(t, []string{
		"import", "customers", "--input", input, "--account", "acct_x", "--json",
	}, "")
	if err == nil {
		t.Fatal("import under account type with --account: error = nil, want forbidden error")
	}
	if !strings.Contains(err.Error(), "remove --account") {
		t.Fatalf("error = %q, want it to contain \"remove --account\"", err.Error())
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode = %d, want 2", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if *requestCount != 0 {
		t.Fatalf("HTTP requests = %d, want 0 (policy must block before any network call)", *requestCount)
	}
}

// TestImportRequiredAccountErrors verifies that under SaaS, `import customers`
// without --account and without a sticky account errors with "required"
// (exit code 2) and sends zero HTTP requests — customers POST is Require
// under SaaS, so a missing account is a hard error, not a silent unscoped call.
func TestImportRequiredAccountErrors(t *testing.T) {
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("STRADDLE_API_KEY", "test_key")
	if err := straddleacct.SaveContext(straddleacct.Context{IntegrationType: straddleacct.TypeSaaS}); err != nil {
		t.Fatal(err)
	}
	server, mu, _, requestCount := recordingServer(t)
	t.Setenv("STRADDLE_BASE_URL", server.URL)

	input := filepath.Join(t.TempDir(), "customers.jsonl")
	if err := os.WriteFile(input, []byte("{\"name\":\"One\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := runRootForAPITest(t, []string{
		"import", "customers", "--input", input, "--json",
	}, "")
	if err == nil {
		t.Fatal("import customers under SaaS without account: error = nil, want required error")
	}
	if !strings.Contains(err.Error(), "an account id is required") {
		t.Fatalf("error = %q, want it to contain \"an account id is required\"", err.Error())
	}
	if got := ExitCode(err); got != 2 {
		t.Fatalf("ExitCode = %d, want 2", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if *requestCount != 0 {
		t.Fatalf("HTTP requests = %d, want 0 (Require must block before any network call)", *requestCount)
	}
}
