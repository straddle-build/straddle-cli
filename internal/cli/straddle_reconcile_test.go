// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.
//
// Tests for the reconcile command's outstanding bucket. The outstanding
// bucket must report only money that can still settle; payments in terminal
// statuses (failed before funding, cancelled, reversed without funding refs)
// must be excluded so the "Outstanding (not yet settled)" total does not
// overstate the cash still expected to arrive.
package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/store"
)

// seedReconcileStore opens a fresh store at dbPath, seeds it with the given
// payments and funding events (as UpsertBatch items), and closes it so the
// reconcile command reopens it read-only. The scope matches what the
// command resolves in the test environment.
func seedReconcileStore(t *testing.T, dbPath string, payments, events []json.RawMessage) {
	t.Helper()
	db, err := store.Open(dbPath, testStoreScope(t))
	if err != nil {
		t.Fatalf("open seed store: %v", err)
	}
	if len(payments) > 0 {
		if _, _, err := db.UpsertBatch("payments", payments); err != nil {
			t.Fatalf("seed payments: %v", err)
		}
	}
	if len(events) > 0 {
		if _, _, err := db.UpsertBatch("funding_events", events); err != nil {
			t.Fatalf("seed funding_events: %v", err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed store: %v", err)
	}
}

// runReconcileCmd drives RootCmd() with the given args and returns stdout.
// The environment is isolated so no real config or credentials are read.
func runReconcileCmd(t *testing.T, dbPath string, args ...string) (string, string, error) {
	t.Helper()
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	cmd := RootCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(append([]string{"reconcile", "--db", dbPath}, args...))
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

// TestReconcileOutstandingStatuses is a table-driven test that pins which
// statuses land in the outstanding bucket and which are excluded. It seeds a
// single unfunded payment in each status and checks the bucket. Only terminal
// statuses that can never settle (failed, reversed, cancelled) are excluded;
// settleable statuses (created, validating, scheduled, on_hold, pending) remain.
func TestReconcileOutstandingStatuses(t *testing.T) {
	cases := []struct {
		status      string
		outstanding bool
	}{
		{"created", true},
		{"validating", true},
		{"scheduled", true},
		{"on_hold", true},
		{"pending", true},
		{"failed", false},
		{"cancelled", false},
		{"reversed", false},
	}
	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			payments := []json.RawMessage{
				json.RawMessage(`{"id":"pay_` + tc.status + `","payment_type":"charge","status":"` + tc.status + `","amount":10000,"currency":"USD"}`),
			}
			dbPath := filepath.Join(t.TempDir(), "reconcile.db")
			seedReconcileStore(t, dbPath, payments, nil)

			stdout, stderr, err := runReconcileCmd(t, dbPath, "--outstanding", "--json")
			if err != nil {
				t.Fatalf("reconcile --outstanding --json failed: %v\nstderr: %s", err, stderr)
			}

			var bucket reconcileBucket
			if err := json.Unmarshal([]byte(stdout), &bucket); err != nil {
				t.Fatalf("expected JSON bucket, got: %s\nerr: %v", stdout, err)
			}
			if tc.outstanding {
				if bucket.PaymentCount != 1 {
					t.Errorf("status %q should be outstanding: payment_count = %d, want 1\nbucket: %s",
						tc.status, bucket.PaymentCount, stdout)
				}
				if bucket.PaymentTotal != 10000 {
					t.Errorf("status %q should be outstanding: payment_total = %d, want 10000\nbucket: %s",
						tc.status, bucket.PaymentTotal, stdout)
				}
			} else {
				if bucket.PaymentCount != 0 {
					t.Errorf("status %q must NOT be outstanding: payment_count = %d, want 0\nbucket: %s",
						tc.status, bucket.PaymentCount, stdout)
				}
				if bucket.PaymentTotal != 0 {
					t.Errorf("status %q must NOT be outstanding: payment_total = %d, want 0\nbucket: %s",
						tc.status, bucket.PaymentTotal, stdout)
				}
			}
		})
	}
}

// TestReconcileOutstandingJSONExcludesTerminalStatuses is the core regression
// test: `straddle reconcile --outstanding --json` must not include failed,
// cancelled, or reversed payments in the outstanding bucket. Only created and
// scheduled payments (which can still settle) should appear, and the total
// must reflect only their amounts. Funded payments are grouped under their
// funding event, not in the outstanding bucket.
func TestReconcileOutstandingJSONExcludesTerminalStatuses(t *testing.T) {
	payments := []json.RawMessage{
		json.RawMessage(`{"id":"pay_created","payment_type":"charge","status":"created","amount":10000,"currency":"USD"}`),
		json.RawMessage(`{"id":"pay_scheduled","payment_type":"charge","status":"scheduled","amount":5000,"currency":"USD"}`),
		json.RawMessage(`{"id":"pay_failed","payment_type":"charge","status":"failed","amount":50000,"currency":"USD","status_details":{"code":"R01","reason":"nsf"}}`),
		json.RawMessage(`{"id":"pay_cancelled","payment_type":"payout","status":"cancelled","amount":30000,"currency":"USD"}`),
		json.RawMessage(`{"id":"pay_reversed","payment_type":"charge","status":"reversed","amount":20000,"currency":"USD"}`),
		json.RawMessage(`{"id":"pay_paid","payment_type":"charge","status":"paid","amount":40000,"currency":"USD","funding_id":"fe_1"}`),
		json.RawMessage(`{"id":"pay_pending","payment_type":"payout","status":"pending","amount":25000,"currency":"USD","funding_id":"fe_1"}`),
	}
	events := []json.RawMessage{
		json.RawMessage(`{"id":"fe_1","amount":65000,"direction":"inbound","status":"settled","payment_count":2}`),
	}
	dbPath := filepath.Join(t.TempDir(), "reconcile.db")
	seedReconcileStore(t, dbPath, payments, events)

	stdout, stderr, err := runReconcileCmd(t, dbPath, "--outstanding", "--json")
	if err != nil {
		t.Fatalf("reconcile --outstanding --json failed: %v\nstderr: %s", err, stderr)
	}

	var bucket reconcileBucket
	if err := json.Unmarshal([]byte(stdout), &bucket); err != nil {
		t.Fatalf("expected JSON bucket, got: %s\nerr: %v", stdout, err)
	}

	// Only created + scheduled payments can settle; their combined amount is
	// $150.00 (10000 + 5000 cents). Failed ($500), cancelled ($300), and
	// reversed ($200) must be excluded.
	if bucket.PaymentCount != 2 {
		t.Errorf("outstanding payment_count = %d, want 2 (created + scheduled only)\nbucket: %s",
			bucket.PaymentCount, stdout)
	}
	if bucket.PaymentTotal != 15000 {
		t.Errorf("outstanding payment_total = %d, want 15000 (created 10000 + scheduled 5000)\nbucket: %s",
			bucket.PaymentTotal, stdout)
	}

	gotIDs := make(map[string]bool, len(bucket.Payments))
	for _, p := range bucket.Payments {
		gotIDs[p.ID] = true
	}
	for _, want := range []string{"pay_created", "pay_scheduled"} {
		if !gotIDs[want] {
			t.Errorf("outstanding bucket missing %q\nbucket: %s", want, stdout)
		}
	}
	for _, banned := range []string{"pay_failed", "pay_cancelled", "pay_reversed", "pay_paid", "pay_pending"} {
		if gotIDs[banned] {
			t.Errorf("outstanding bucket must not include terminal/funded payment %q\nbucket: %s", banned, stdout)
		}
	}
}

// TestReconcileFundedTerminalStillGrouped verifies that a reversed payment
// WITH funding refs (funded then reversed) is still grouped under its funding
// event — the fix only skips terminal payments that have no funding refs.
// This guards against the fix over-skipping funded terminal payments from
// their funding-event groups.
func TestReconcileFundedTerminalStillGrouped(t *testing.T) {
	payments := []json.RawMessage{
		// A reversed payment that was funded then reversed: it carries
		// funding_id and must land under fe_1, not be skipped.
		json.RawMessage(`{"id":"pay_reversed_funded","payment_type":"charge","status":"reversed","amount":40000,"currency":"USD","funding_id":"fe_1"}`),
		// A settleable unfunded payment for the outstanding bucket.
		json.RawMessage(`{"id":"pay_created","payment_type":"charge","status":"created","amount":10000,"currency":"USD"}`),
	}
	events := []json.RawMessage{
		json.RawMessage(`{"id":"fe_1","amount":40000,"direction":"inbound","status":"settled","payment_count":1}`),
	}
	dbPath := filepath.Join(t.TempDir(), "reconcile.db")
	seedReconcileStore(t, dbPath, payments, events)

	stdout, stderr, err := runReconcileCmd(t, dbPath, "--json")
	if err != nil {
		t.Fatalf("reconcile --json failed: %v\nstderr: %s", err, stderr)
	}

	var result reconcileResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("expected JSON result, got: %s\nerr: %v", stdout, err)
	}

	// The reversed-but-funded payment must be in the fe_1 group.
	if len(result.FundingEvents) != 1 {
		t.Fatalf("expected 1 funding event, got %d\nresult: %s", len(result.FundingEvents), stdout)
	}
	g := result.FundingEvents[0]
	if g.PaymentCount != 1 || g.PaymentTotal != 40000 {
		t.Errorf("fe_1 group = %d payments $%d, want 1 payment $40000 (reversed-but-funded must stay grouped)\nresult: %s",
			g.PaymentCount, g.PaymentTotal, stdout)
	}
	if len(g.Payments) != 1 || g.Payments[0].ID != "pay_reversed_funded" {
		t.Errorf("fe_1 must contain pay_reversed_funded\nresult: %s", stdout)
	}

	// The outstanding bucket must contain only the created payment.
	if result.Outstanding.PaymentCount != 1 || result.Outstanding.PaymentTotal != 10000 {
		t.Errorf("outstanding = %d payments $%d, want 1 payment $10000\nresult: %s",
			result.Outstanding.PaymentCount, result.Outstanding.PaymentTotal, stdout)
	}
}
