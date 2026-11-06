// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestEscapeFTSQuery pins the FTS5 escaping contract: every whitespace-
// separated token becomes a double-quoted phrase with internal double
// quotes doubled, so the bound MATCH argument is always a legal FTS5
// expression regardless of what metacharacters the input contains. The
// raw-passthrough bug aborted the command on unbalanced quotes/parens/
// stray punctuation; this table is the direct regression for that.
func TestEscapeFTSQuery(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain word", "payment", "\"payment\""},
		{"two words implicit AND", "hello world", "\"hello\" \"world\""},
		{"unbalanced trailing quote", "dlc\"foo", "\"dlc\"\"foo\""},
		{"unbalanced leading paren", "(unclosed", "\"(unclosed\""},
		{"trailing semicolon", "payment;", "\"payment;\""},
		{"stray closing paren", ")", "\")\""},
		{"unbalanced parens", ")(", "\")(\""},
		{"quote comma brace mix", "\",}", "\"\"\",}\""},
		{"lone double quote", "\"", "\"\"\"\""},
		{"phrase with internal quotes", "he said \"hi\"", "\"he\" \"said\" \"\"\"hi\"\"\""},
		{"leading and trailing spaces", "  spaced  ", "\"spaced\""},
		{"only spaces", "   ", ""},
		{"empty", "", ""},
		{"whitespace separated", "a\tb", "\"a\" \"b\""},
		{"asterisk prefix becomes literal", "pay*", "\"pay*\""},
		{"AND operator becomes literal", "payment AND failed", "\"payment\" \"AND\" \"failed\""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := escapeFTSQuery(c.in); got != c.want {
				t.Fatalf("escapeFTSQuery(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestSearchMetacharactersDoNotError is the end-to-end regression for the
// bug: a Store.Search call with a query containing unbalanced FTS5
// metacharacters used to abort with "SQL logic error: unterminated string"
// or "fts5: syntax error near …". After the escape fix, every such query
// must return (possibly empty) results and never an error.
func TestSearchMetacharactersDoNotError(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "data.db")
	s, err := Open(dbPath, testScope)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.Upsert("payments", "row-1", json.RawMessage(`{"id":"row-1","description":"payment marker"}`)); err != nil {
		t.Fatal(err)
	}

	for _, query := range []string{
		"dlc\"foo",
		"(unclosed",
		"payment;",
		")",
		")(",
		"\",}",
		"\"",
		"hello \"world",
		"a\"b\"c",
		"payment AND failed",
		"payment OR delay",
		"pay*",
	} {
		if _, err := s.Search(query, 50); err != nil {
			t.Fatalf("Search(%q): %v (must not error on FTS5 metacharacters)", query, err)
		}
	}
}

// TestSearchTypedMetacharactersDoNotError mirrors the untyped regression
// through SearchTyped, which funnels into the same search() helper and
// was equally broken by the raw-passthrough bug.
func TestSearchTypedMetacharactersDoNotError(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "data.db")
	s, err := Open(dbPath, testScope)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.Upsert("payments", "row-1", json.RawMessage(`{"id":"row-1","description":"payment marker"}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Upsert("charges", "row-2", json.RawMessage(`{"id":"row-2","description":"payment marker"}`)); err != nil {
		t.Fatal(err)
	}

	for _, query := range []string{"dlc\"foo", "(unclosed", "payment;", ")("} {
		if _, err := s.SearchTyped("payments", query, 50); err != nil {
			t.Fatalf("SearchTyped(%q): %v (must not error on FTS5 metacharacters)", query, err)
		}
	}
}

// TestSearchPlainTermsStillMatch guards the happy path: escaping the query
// into quoted phrases must not break matching for plain single-word
// queries (the only kind the docs and existing tests use). The search term
// and ids avoid the "pay"/"payment" stem because resource_type is an
// indexed FTS column: "payments" stems to "pay", so a search for
// "payment" would match every row of type payments, not just the one
// whose description contains it.
func TestSearchPlainTermsStillMatch(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "data.db")
	s, err := Open(dbPath, testScope)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.Upsert("payments", "row-1", json.RawMessage(`{"id":"row-1","description":"alpha signature"}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Upsert("payments", "row-2", json.RawMessage(`{"id":"row-2","description":"beta signature"}`)); err != nil {
		t.Fatal(err)
	}

	got, err := s.Search("alpha", 50)
	if err != nil {
		t.Fatalf("Search(plain): %v", err)
	}
	if len(got) != 1 || !bytes.Contains(got[0], []byte(`"row-1"`)) {
		t.Fatalf("Search(\"alpha\") = %s, want only row-1", got)
	}
}

// TestSearchMultiWordImplicitAND pins that multi-word queries still match
// only rows containing every term (FTS5 implicit-AND across phrases).
// Escaping wraps each token as a phrase but joins them with whitespace,
// preserving the documented "error timeout" → both-terms behavior.
func TestSearchMultiWordImplicitAND(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "data.db")
	s, err := Open(dbPath, testScope)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.Upsert("payments", "row-1", json.RawMessage(`{"id":"row-1","description":"error timeout marker"}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Upsert("payments", "row-2", json.RawMessage(`{"id":"row-2","description":"error only"}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Upsert("payments", "row-3", json.RawMessage(`{"id":"row-3","description":"timeout only"}`)); err != nil {
		t.Fatal(err)
	}

	got, err := s.Search("error timeout", 50)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) != 1 || !bytes.Contains(got[0], []byte(`"row-1"`)) {
		t.Fatalf("Search(\"error timeout\") = %s, want only row-1 (both terms)", got)
	}
}

// TestSearchEmptyQueryDoesNotError confirms an empty or whitespace-only
// query (which escapes to the empty string) returns empty results and no
// error rather than aborting the command on an FTS5 syntax error.
func TestSearchEmptyQueryDoesNotError(t *testing.T) {
	t.Parallel()
	dbPath := filepath.Join(t.TempDir(), "data.db")
	s, err := Open(dbPath, testScope)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.Upsert("payments", "row-1", json.RawMessage(`{"id":"row-1","description":"marker"}`)); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"", "   ", "\t"} {
		got, err := s.Search(query, 50)
		if err != nil {
			t.Fatalf("Search(%q): %v (empty query must not error)", query, err)
		}
		if len(got) != 0 {
			t.Fatalf("Search(%q) = %d rows, want 0 (no terms to match)", query, len(got))
		}
	}
}
