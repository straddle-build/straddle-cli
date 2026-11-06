// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// decodeSyncNDJSONLines splits captured sync stdout into one JSON object per
// line and fails the test if any non-empty line is not valid JSON. This is
// the contract sync's default (non --human-friendly) mode must uphold: every
// stdout line is a standalone JSON event so line-by-line consumers (the
// standard way to consume NDJSON) never choke on a malformed line and lose
// subsequent events. Modelled on decodeTailEvents in tail_test.go, which
// guards the same contract for the tail command.
func decodeSyncNDJSONLines(t *testing.T, out []byte) []map[string]any {
	t.Helper()
	var events []map[string]any
	for i, line := range bytes.Split(out, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var ev map[string]any
		if err := json.Unmarshal(line, &ev); err != nil {
			t.Fatalf("stdout line %d is not valid JSON: %v\nline=%s\nfull stdout:\n%s", i, err, string(line), out)
		}
		events = append(events, ev)
	}
	return events
}

// findSyncWarningEvent returns the first sync_warning event whose resource
// matches resource, or nil if none. Used to inspect the warning payload
// after asserting the whole stream is valid NDJSON.
func findSyncWarningEvent(events []map[string]any, resource string) map[string]any {
	for _, ev := range events {
		if ev["event"] != "sync_warning" {
			continue
		}
		if r, _ := ev["resource"].(string); r == resource {
			return ev
		}
	}
	return nil
}

// TestSyncAccessWarningNDJSONValidFlatPath is the regression test for the
// "access-denied warnings emit invalid NDJSON" bug on the flat
// syncResource path. Before the fix, a 403 response with a multi-line or
// backslash-containing body was interpolated into a hand-rolled JSON
// template whose only escaping was strings.ReplaceAll(w.Message, `"`, `\"`).
// Raw newlines split the warning across multiple stdout lines (breaking
// NDJSON framing), and backslash sequences like `\"` terminated the JSON
// string early — both producing lines that are not valid JSON. This test
// drives the real sync command against an httptest server returning 403
// with hostile bodies and asserts every emitted stdout line decodes as JSON
// and the warning payload carries the body verbatim (newlines and
// backslashes preserved, not corrupted).
func TestSyncAccessWarningNDJSONValidFlatPath(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			"multiline_html_body",
			"<html>\n403 Forbidden\n</html>\n",
		},
		{
			"backslash_quote_body",
			`{"error": "quote"}`,
		},
		{
			"trailing_backslash_body",
			`error: bad\`,
		},
		{
			"mixed_newlines_and_backslashes",
			"line1\nline2\\with\\backslashes\r\nand\r\nCRLF",
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()

			dbPath := filepath.Join(t.TempDir(), "data.db")
			res := runSyncCommand(t, dbPath, server.URL, "--resources", "accounts")

			// Every stdout line must be valid JSON (the NDJSON contract).
			events := decodeSyncNDJSONLines(t, res.stdout)

			// The warning must be present and carry the body verbatim.
			warn := findSyncWarningEvent(events, "accounts")
			if warn == nil {
				t.Fatalf("no sync_warning event for accounts in stdout:\n%s", res.stdout)
			}
			if warn["reason"] != "forbidden" {
				t.Errorf("sync_warning reason = %v, want \"forbidden\"", warn["reason"])
			}
			if warn["status"] != float64(403) {
				t.Errorf("sync_warning status = %v, want 403", warn["status"])
			}
			if warn["message"] != tc.body {
				t.Errorf("sync_warning message = %q, want %q (body must be preserved verbatim)", warn["message"], tc.body)
			}
			// The flat path warning must not carry a parent field.
			if _, ok := warn["parent"]; ok {
				t.Errorf("sync_warning for flat path must not carry a parent field, got: %v", warn["parent"])
			}
		})
	}
}

// TestSyncAccessWarningNDJSONValidDependentPath is the regression test for
// the dependent-resource path (syncDependentResource). The same
// hand-rolled Fprintf template with quote-only escaping existed there;
// a per-parent 403 with a multi-line or backslash-containing body
// corrupted the NDJSON stream identically. This test seeds an accounts
// parent row, then syncs capability_requests against a server that 403s
// every per-parent call with a hostile body, and asserts every stdout
// line is valid JSON and the warning carries the parent field.
func TestSyncAccessWarningNDJSONValidDependentPath(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			"multiline_html_body",
			"<html>\n403 Forbidden\n</html>\n",
		},
		{
			"backslash_quote_body",
			`{"error": "quote"}`,
		},
		{
			"trailing_backslash_body",
			`error: bad\`,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			server := depTestServer(t, `[{"id":"acc1"}]`, func(parentID string) (int, string) {
				return http.StatusForbidden, tc.body
			})
			defer server.Close()

			dbPath := filepath.Join(t.TempDir(), "data.db")
			seedAccountsSync(t, dbPath, server.URL)

			res := runSyncCommand(t, dbPath, server.URL, "--resources", "capability_requests")

			// Every stdout line must be valid JSON (the NDJSON contract).
			events := decodeSyncNDJSONLines(t, res.stdout)

			// The dependent warning must be present, carry the parent
			// field, and preserve the body verbatim.
			warn := findSyncWarningEvent(events, "capability_requests")
			if warn == nil {
				t.Fatalf("no sync_warning event for capability_requests in stdout:\n%s", res.stdout)
			}
			if warn["reason"] != "forbidden" {
				t.Errorf("sync_warning reason = %v, want \"forbidden\"", warn["reason"])
			}
			if warn["status"] != float64(403) {
				t.Errorf("sync_warning status = %v, want 403", warn["status"])
			}
			if warn["parent"] != "acc1" {
				t.Errorf("sync_warning parent = %v, want \"acc1\"", warn["parent"])
			}
			if warn["message"] != tc.body {
				t.Errorf("sync_warning message = %q, want %q (body must be preserved verbatim)", warn["message"], tc.body)
			}
		})
	}
}

// TestSyncAccessWarningNDJSONValidBenignBody guards the happy path: a
// single-line JSON 403 body (the only shape existing tests exercised before
// the fix) must still produce a valid sync_warning line with the expected
// fields. This ensures the json.Marshal fix did not change the output shape
// for the common case.
func TestSyncAccessWarningNDJSONValidBenignBody(t *testing.T) {
	body := `{"error":"forbidden","reason":"insufficient scope"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	dbPath := filepath.Join(t.TempDir(), "data.db")
	res := runSyncCommand(t, dbPath, server.URL, "--resources", "accounts")

	events := decodeSyncNDJSONLines(t, res.stdout)
	warn := findSyncWarningEvent(events, "accounts")
	if warn == nil {
		t.Fatalf("no sync_warning event for accounts in stdout:\n%s", res.stdout)
	}
	if warn["reason"] != "forbidden" {
		t.Errorf("sync_warning reason = %v, want \"forbidden\"", warn["reason"])
	}
	if warn["status"] != float64(403) {
		t.Errorf("sync_warning status = %v, want 403", warn["status"])
	}
	if warn["message"] != body {
		t.Errorf("sync_warning message = %q, want %q", warn["message"], body)
	}
}

// TestSyncWarningJSONUnit directly exercises the syncWarningJSON helper to
// pin its output contract: the result is a single valid JSON line with the
// expected field values, and hostile message content (newlines, backslashes,
// quotes) is fully JSON-escaped. This guards the helper independently of the
// sync command wiring so a future refactor that bypasses json.Marshal is
// caught here.
func TestSyncWarningJSONUnit(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		resource string
		parent   string
		w        *accessWarning
	}{
		{
			"flat_forbidden",
			"accounts",
			"",
			&accessWarning{Status: 403, Reason: "forbidden", Message: `{"error":"forbidden"}`},
		},
		{
			"dependent_forbidden",
			"capability_requests",
			"acc1",
			&accessWarning{Status: 403, Reason: "forbidden", Message: "denied\nfor parent"},
		},
		{
			"backslash_and_quote",
			"accounts",
			"",
			&accessWarning{Status: 400, Reason: "insufficient_access", Message: `bad\"quote\`},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := syncWarningJSON(tc.resource, tc.parent, tc.w)
			// Must be a single valid JSON line.
			var ev map[string]any
			if err := json.Unmarshal([]byte(out), &ev); err != nil {
				t.Fatalf("syncWarningJSON produced invalid JSON: %v\noutput: %s", err, out)
			}
			if ev["event"] != "sync_warning" {
				t.Errorf("event = %v, want \"sync_warning\"", ev["event"])
			}
			if ev["resource"] != tc.resource {
				t.Errorf("resource = %v, want %q", ev["resource"], tc.resource)
			}
			if ev["reason"] != tc.w.Reason {
				t.Errorf("reason = %v, want %q", ev["reason"], tc.w.Reason)
			}
			if ev["message"] != tc.w.Message {
				t.Errorf("message = %v, want %q", ev["message"], tc.w.Message)
			}
			if int(ev["status"].(float64)) != tc.w.Status {
				t.Errorf("status = %v, want %d", ev["status"], tc.w.Status)
			}
			// parent field: present only when non-empty (omitempty).
			if tc.parent != "" {
				if ev["parent"] != tc.parent {
					t.Errorf("parent = %v, want %q", ev["parent"], tc.parent)
				}
			} else {
				if _, ok := ev["parent"]; ok {
					t.Errorf("parent should be omitted for flat path, got: %v", ev["parent"])
				}
			}
			// Must not contain a raw newline (would break NDJSON framing).
			if bytes.Contains([]byte(out), []byte("\n")) {
				t.Errorf("syncWarningJSON output contains a raw newline (breaks NDJSON framing):\n%s", out)
			}
		})
	}
}
