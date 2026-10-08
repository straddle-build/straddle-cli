// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// depTestServer builds an httptest server for the dependent-resource error
// classification scenarios. The flat accounts list endpoint always returns
// accountsJSON (a JSON array) so the parent table can be seeded; each
// per-parent capability_requests call is routed through depResp, which
// returns the HTTP status and body to write for that parent ID. HTTP 422
// is used for hard errors (rather than 5xx) so the client does not retry
// with exponential backoff, keeping the tests fast.
func depTestServer(t *testing.T, accountsJSON string, depResp func(parentID string) (int, string)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/v1/accounts":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(accountsJSON))
		case strings.HasPrefix(r.URL.Path, "/v1/accounts/") && strings.HasSuffix(r.URL.Path, "/capability_requests"):
			parentID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/accounts/"), "/capability_requests")
			status, body := depResp(parentID)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// seedAccountsSync runs `straddle sync --resources accounts` against the
// server to populate the parent table. The dependent runner cannot make
// per-parent capability_requests calls until at least one parent row
// exists, so the dependent scenarios are driven as two runs against the
// same DB: first seed accounts, then sync the dependent alone.
func seedAccountsSync(t *testing.T, dbPath, serverURL string) {
	t.Helper()
	res := runSyncCommand(t, dbPath, serverURL, "--resources", "accounts")
	if res.err != nil {
		t.Fatalf("seed accounts sync failed: %v\nstdout: %s\nstderr: %s", res.err, res.stdout, res.stderr)
	}
}

// TestSyncDependentAllParentsHardErrorFails is the regression test for the
// "dependent resources report success when all parent fetches hard-fail" bug.
// Before the fix, a sync where the only selected resource is a dependent and
// every parent call returns a hard error (HTTP 422) exited 0 with sync_summary
// reporting success:1, errored:0 — indistinguishable from a clean sync. After
// the fix, the dependent surfaces as Err so the run-level exit-code policy's
// "all-resource failure exits non-zero" contract holds.
func TestSyncDependentAllParentsHardErrorFails(t *testing.T) {
	server := depTestServer(t, `[{"id":"acc1"}]`, func(parentID string) (int, string) {
		return http.StatusUnprocessableEntity, `{"error":"invalid","reason":"server rejected filter"}`
	})
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "data.db")
	seedAccountsSync(t, dbPath, server.URL)

	res := runSyncCommand(t, dbPath, server.URL, "--resources", "capability_requests")

	if res.exitCode == 0 {
		t.Errorf("exitCode = 0, want non-zero: every selected resource's parent call failed (all-resource failure must exit non-zero)\nstdout: %s\nstderr: %s", res.stdout, res.stderr)
	}
	summary := parseSyncSummary(t, res.stdout)
	if summary.Errored != 1 {
		t.Errorf("sync_summary errored = %d, want 1 (dependent hard failure must be classified as a resource error)\nstdout: %s", summary.Errored, res.stdout)
	}
	if summary.Success != 0 {
		t.Errorf("sync_summary success = %d, want 0 (dependent must not be counted as success when every parent hard-errored)\nstdout: %s", summary.Success, res.stdout)
	}
	if summary.Warned != 0 {
		t.Errorf("sync_summary warned = %d, want 0 (hard errors are Err, not Warn)\nstdout: %s", summary.Warned, res.stdout)
	}
	if summary.Resources != 1 {
		t.Errorf("sync_summary resources = %d, want 1\nstdout: %s", summary.Resources, res.stdout)
	}
	if !bytes.Contains(res.stdout, []byte(`"event":"sync_error"`)) {
		t.Errorf("stdout should contain a sync_error event for the failed parent\nstdout: %s", res.stdout)
	}
	if !bytes.Contains(res.stdout, []byte(`"resource":"capability_requests"`)) {
		t.Errorf("stdout should reference capability_requests\nstdout: %s", res.stdout)
	}
}

// TestSyncDependentAllParentsAccessDeniedStaysWarn guards the existing
// all-denied Warn branch against the fix. The fix restructured the shared
// per-parent error-handling block; this test ensures access denials (HTTP
// 403) still surface as Warn (not Err) and still exit non-zero via the
// all-warned exit-code policy. A hard error and an access denial must
// remain distinct classifications.
func TestSyncDependentAllParentsAccessDeniedStaysWarn(t *testing.T) {
	server := depTestServer(t, `[{"id":"acc1"}]`, func(parentID string) (int, string) {
		return http.StatusForbidden, `{"error":"forbidden","reason":"insufficient scope"}`
	})
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "data.db")
	seedAccountsSync(t, dbPath, server.URL)

	res := runSyncCommand(t, dbPath, server.URL, "--resources", "capability_requests")

	if res.exitCode == 0 {
		t.Errorf("exitCode = 0, want non-zero: all-warned dependent must exit non-zero\nstdout: %s\nstderr: %s", res.stdout, res.stderr)
	}
	summary := parseSyncSummary(t, res.stdout)
	if summary.Warned != 1 {
		t.Errorf("sync_summary warned = %d, want 1 (all-denied dependent must surface as Warn)\nstdout: %s", summary.Warned, res.stdout)
	}
	if summary.Errored != 0 {
		t.Errorf("sync_summary errored = %d, want 0 (access denials are Warn, not Err)\nstdout: %s", summary.Errored, res.stdout)
	}
	if summary.Success != 0 {
		t.Errorf("sync_summary success = %d, want 0 (all-denied dependent must not be counted as success)\nstdout: %s", summary.Success, res.stdout)
	}
	if !bytes.Contains(res.stdout, []byte(`"event":"sync_warning"`)) {
		t.Errorf("stdout should contain a sync_warning event for the access denial\nstdout: %s", res.stdout)
	}
}

// TestSyncDependentMixedDeniedAndHardErrorFails covers total failure split
// across both classifications: one parent access-denied, the other
// hard-errored, zero rows. Neither all-denied nor all-hard-errored holds on
// its own, yet nothing synced, so the dependent must surface as Err (a hard
// error outranks a denial) and the run must exit non-zero.
func TestSyncDependentMixedDeniedAndHardErrorFails(t *testing.T) {
	accountsJSON := `[{"id":"acc1"},{"id":"acc2"}]`
	server := depTestServer(t, accountsJSON, func(parentID string) (int, string) {
		switch parentID {
		case "acc1":
			return http.StatusForbidden, `{"error":"forbidden","reason":"insufficient scope"}`
		default:
			return http.StatusUnprocessableEntity, `{"error":"invalid","reason":"server rejected filter"}`
		}
	})
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "data.db")
	seedAccountsSync(t, dbPath, server.URL)

	res := runSyncCommand(t, dbPath, server.URL, "--resources", "capability_requests")

	if res.exitCode == 0 {
		t.Errorf("exitCode = 0, want non-zero: every parent failed (one denied, one hard error) and nothing synced\nstdout: %s\nstderr: %s", res.stdout, res.stderr)
	}
	summary := parseSyncSummary(t, res.stdout)
	if summary.Errored != 1 {
		t.Errorf("sync_summary errored = %d, want 1 (a hard error among all-failed parents must surface as Err)\nstdout: %s", summary.Errored, res.stdout)
	}
	if summary.Success != 0 {
		t.Errorf("sync_summary success = %d, want 0 (all-failed dependent must not be counted as success)\nstdout: %s", summary.Success, res.stdout)
	}
	if summary.Warned != 0 {
		t.Errorf("sync_summary warned = %d, want 0 (a hard error outranks the denial)\nstdout: %s", summary.Warned, res.stdout)
	}
	if !bytes.Contains(res.stdout, []byte(`"event":"sync_warning"`)) {
		t.Errorf("stdout should contain a per-parent sync_warning for the denied parent\nstdout: %s", res.stdout)
	}
	if !bytes.Contains(res.stdout, []byte(`"event":"sync_error"`)) {
		t.Errorf("stdout should contain a per-parent sync_error for the hard-errored parent\nstdout: %s", res.stdout)
	}
}

// TestSyncDependentPartialHardErrorStaysSuccess pins the per-parent
// resilience boundary: when only SOME parents hard-error (and at least one
// parent call succeeds, even with zero rows), the dependent still returns
// Err: nil and is counted as success. The guard only fires when every parent
// failed; this test guards against a future change that makes the
// all-failed guard fire too broadly and breaks the intentional per-parent
// resilience (which the all-denied pattern also relies on).
func TestSyncDependentPartialHardErrorStaysSuccess(t *testing.T) {
	accountsJSON := `[{"id":"acc1"},{"id":"acc2"}]`
	server := depTestServer(t, accountsJSON, func(parentID string) (int, string) {
		switch parentID {
		case "acc1":
			return http.StatusOK, `[]`
		default:
			return http.StatusUnprocessableEntity, `{"error":"invalid","reason":"server rejected filter"}`
		}
	})
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "data.db")
	seedAccountsSync(t, dbPath, server.URL)

	res := runSyncCommand(t, dbPath, server.URL, "--resources", "capability_requests")

	if res.exitCode != 0 {
		t.Errorf("exitCode = %d, want 0: a partial hard error (one parent succeeded) must not fail the dependent\nstdout: %s\nstderr: %s", res.exitCode, res.stdout, res.stderr)
	}
	summary := parseSyncSummary(t, res.stdout)
	if summary.Success != 1 {
		t.Errorf("sync_summary success = %d, want 1 (partial hard error must still count the dependent as success)\nstdout: %s", summary.Success, res.stdout)
	}
	if summary.Errored != 0 {
		t.Errorf("sync_summary errored = %d, want 0 (partial hard error must not surface as Err)\nstdout: %s", summary.Errored, res.stdout)
	}
	if summary.Warned != 0 {
		t.Errorf("sync_summary warned = %d, want 0 (partial hard error must not surface as Warn)\nstdout: %s", summary.Warned, res.stdout)
	}
	if !bytes.Contains(res.stdout, []byte(`"event":"sync_error"`)) {
		t.Errorf("stdout should still contain a per-parent sync_error for the failed parent\nstdout: %s", res.stdout)
	}
}
