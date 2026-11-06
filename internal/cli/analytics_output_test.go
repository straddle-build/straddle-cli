// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package cli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/straddle-build/straddle-cli/internal/store"
)

// seedAnalyticsStore opens a fresh store at dbPath and inserts three messages
// (two with author_id "a1", one with "a2") so the summary, count, and
// group-by paths all have data to operate on.
func seedAnalyticsStore(t *testing.T, dbPath string) {
	t.Helper()
	db, err := store.Open(dbPath, testStoreScope(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []struct {
		id       string
		authorID string
	}{
		{"m1", "a1"},
		{"m2", "a1"},
		{"m3", "a2"},
	} {
		data, _ := json.Marshal(map[string]any{"id": m.id, "author_id": m.authorID})
		if err := db.Upsert("messages", m.id, data); err != nil {
			db.Close()
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestAnalyticsOutputHonorsSmartDefault verifies that the `straddle analytics`
// command follows the same "JSON when piped / machine-flagged, table when
// terminal" smart-default contract as every other read command. All three
// output paths (summary, count, group-by) are exercised.
//
// The test also implicitly verifies that output is written to
// cmd.OutOrStdout() rather than raw os.Stdout: cmd.SetOut(&stdout) only
// captures writes that go through the Cobra output writer, which is the same
// writer --deliver wires through PersistentPreRunE. Before the fix,
// analytics wrote to raw os.Stdout and the stdout buffer stayed empty.
func TestAnalyticsOutputHonorsSmartDefault(t *testing.T) {
	for _, tc := range []struct {
		name     string
		terminal bool
		args     []string
		wantJSON bool
	}{
		// Summary path (no --type): db.Status() -> map[string]int
		{"summary terminal default", true, nil, false},
		{"summary pipe default", false, nil, true},
		{"summary terminal json", true, []string{"--json"}, true},
		{"summary pipe json", false, []string{"--json"}, true},
		{"summary terminal compact", true, []string{"--compact"}, true},
		{"summary pipe compact", false, []string{"--compact"}, true},
		{"summary terminal agent", true, []string{"--agent"}, true},
		{"summary pipe agent", false, []string{"--agent"}, true},
		{"summary terminal agent defaults disabled", true, []string{"--agent", "--json=false", "--compact=false"}, false},

		// Count path (--type messages, no --group-by): db.Count() -> int
		{"count terminal default", true, []string{"--type", "messages"}, false},
		{"count pipe default", false, []string{"--type", "messages"}, true},
		{"count terminal json", true, []string{"--type", "messages", "--json"}, true},
		{"count pipe json", false, []string{"--type", "messages", "--json"}, true},
		{"count terminal compact", true, []string{"--type", "messages", "--compact"}, true},
		{"count pipe compact", false, []string{"--type", "messages", "--compact"}, true},

		// Group-by path (--type messages --group-by author_id): runGroupBy -> []kv
		{"groupby terminal default", true, []string{"--type", "messages", "--group-by", "author_id"}, false},
		{"groupby pipe default", false, []string{"--type", "messages", "--group-by", "author_id"}, true},
		{"groupby terminal json", true, []string{"--type", "messages", "--group-by", "author_id", "--json"}, true},
		{"groupby pipe json", false, []string{"--type", "messages", "--group-by", "author_id", "--json"}, true},
		{"groupby terminal compact", true, []string{"--type", "messages", "--group-by", "author_id", "--compact"}, true},
		{"groupby pipe compact", false, []string{"--type", "messages", "--group-by", "author_id", "--compact"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateAPIConfig(t)
			t.Setenv("HOME", t.TempDir())
			previousTerminal := isTerminalOverride
			previousColor, previousResource := noColor, currentResource
			isTerminalOverride = &tc.terminal
			t.Cleanup(func() {
				isTerminalOverride = previousTerminal
				noColor, currentResource = previousColor, previousResource
			})
			dbPath := filepath.Join(t.TempDir(), "analytics.db")
			seedAnalyticsStore(t, dbPath)

			cmd := RootCmd()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(append([]string{"analytics", "--db", dbPath}, tc.args...))
			if err := cmd.Execute(); err != nil {
				t.Fatalf("analytics failed: %v; stderr=%s", err, stderr.String())
			}

			if stdout.Len() == 0 {
				t.Fatalf("no output captured in stdout buffer; the command likely wrote to raw os.Stdout instead of cmd.OutOrStdout()")
			}

			if tc.wantJSON {
				if !json.Valid(stdout.Bytes()) {
					t.Fatalf("expected JSON output, got %q", stdout.String())
				}
			} else if json.Valid(stdout.Bytes()) {
				t.Fatalf("expected human table output, got JSON %q", stdout.String())
			}
		})
	}
}

// TestAnalyticsOutputContent verifies the actual JSON content for each output
// path when piped, so a regression that produces the wrong shape (or the
// human table) is caught with an actionable diff.
func TestAnalyticsOutputContent(t *testing.T) {
	for _, tc := range []struct {
		name  string
		args  []string
		check func(t *testing.T, out []byte)
	}{
		{
			"summary pipe",
			nil,
			func(t *testing.T, out []byte) {
				var status map[string]int
				if err := json.Unmarshal(out, &status); err != nil {
					t.Fatalf("summary JSON: %v\n%s", err, out)
				}
				if status["messages"] != 3 {
					t.Errorf("summary messages count = %d, want 3", status["messages"])
				}
			},
		},
		{
			"count pipe",
			[]string{"--type", "messages"},
			func(t *testing.T, out []byte) {
				var result map[string]any
				if err := json.Unmarshal(out, &result); err != nil {
					t.Fatalf("count JSON: %v\n%s", err, out)
				}
				if int(result["count"].(float64)) != 3 {
					t.Errorf("count = %v, want 3", result["count"])
				}
				if result["resource_type"] != "messages" {
					t.Errorf("resource_type = %v, want messages", result["resource_type"])
				}
			},
		},
		{
			"groupby pipe",
			[]string{"--type", "messages", "--group-by", "author_id"},
			func(t *testing.T, out []byte) {
				var groups []struct {
					Value string `json:"value"`
					Count int    `json:"count"`
				}
				if err := json.Unmarshal(out, &groups); err != nil {
					t.Fatalf("groupby JSON: %v\n%s", err, out)
				}
				if len(groups) != 2 {
					t.Fatalf("groupby groups = %d, want 2", len(groups))
				}
				// sorted by count desc: a1 (2) before a2 (1)
				if groups[0].Value != "a1" || groups[0].Count != 2 {
					t.Errorf("first group = %+v, want {a1 2}", groups[0])
				}
				if groups[1].Value != "a2" || groups[1].Count != 1 {
					t.Errorf("second group = %+v, want {a2 1}", groups[1])
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			isolateAPIConfig(t)
			t.Setenv("HOME", t.TempDir())
			terminal := false
			previousTerminal := isTerminalOverride
			previousColor, previousResource := noColor, currentResource
			isTerminalOverride = &terminal
			t.Cleanup(func() {
				isTerminalOverride = previousTerminal
				noColor, currentResource = previousColor, previousResource
			})
			dbPath := filepath.Join(t.TempDir(), "analytics.db")
			seedAnalyticsStore(t, dbPath)

			cmd := RootCmd()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SetArgs(append([]string{"analytics", "--db", dbPath}, tc.args...))
			if err := cmd.Execute(); err != nil {
				t.Fatalf("analytics failed: %v; stderr=%s", err, stderr.String())
			}
			tc.check(t, stdout.Bytes())
		})
	}
}

// TestAnalyticsOutputCapturedByDeliver verifies that analytics output is
// captured by the --deliver mechanism. PersistentPreRunE wires
// cmd.SetOut(io.MultiWriter(os.Stdout, deliverBuf)) when --deliver is set;
// before the fix, analytics wrote to raw os.Stdout and the deliver buffer
// stayed empty.
func TestAnalyticsOutputCapturedByDeliver(t *testing.T) {
	isolateAPIConfig(t)
	t.Setenv("HOME", t.TempDir())
	terminal := false
	previousTerminal := isTerminalOverride
	previousColor, previousResource := noColor, currentResource
	isTerminalOverride = &terminal
	t.Cleanup(func() {
		isTerminalOverride = previousTerminal
		noColor, currentResource = previousColor, previousResource
	})

	dbPath := filepath.Join(t.TempDir(), "analytics.db")
	seedAnalyticsStore(t, dbPath)

	var flags rootFlags
	cmd := newRootCmd(&flags)
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"analytics", "--db", dbPath, "--type", "messages", "--deliver", "file:" + filepath.Join(t.TempDir(), "out.json")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("analytics with --deliver failed: %v; stderr=%s", err, stderr.String())
	}
	if flags.deliverBuf == nil {
		t.Fatal("deliverBuf is nil; --deliver did not set up capture")
	}
	if !json.Valid(flags.deliverBuf.Bytes()) {
		t.Fatalf("deliver buffer should contain JSON, got %q", flags.deliverBuf.String())
	}
}
