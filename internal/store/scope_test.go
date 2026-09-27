// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

var testScope = Scope{Environment: "https://sandbox.straddle.com", Account: "acct_test"}

// rawDB opens the store file directly for schema inspection and for
// seeding legacy shapes that the scoped API deliberately cannot write.
func rawDB(t *testing.T, dbPath string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func openScoped(t *testing.T, dbPath string, scope Scope) *Store {
	t.Helper()
	s, err := Open(dbPath, scope)
	if err != nil {
		t.Fatalf("open %+v: %v", scope, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestOpenRejectsScopeWithoutEnvironment(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data.db")
	for _, scope := range []Scope{{}, {Account: "acct_a"}, {Environment: "  "}} {
		if s, err := Open(dbPath, scope); err == nil {
			_ = s.Close()
			t.Fatalf("Open(%+v) succeeded; legacy rows would be readable", scope)
		}
	}
}

func TestSameResourceIDCoexistsAcrossScopes(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data.db")
	scopes := map[string]Scope{
		"sandbox/a":    {Environment: "https://sandbox.straddle.com", Account: "acct_a"},
		"sandbox/b":    {Environment: "https://sandbox.straddle.com", Account: "acct_b"},
		"sandbox/none": {Environment: "https://sandbox.straddle.com"},
		"production/a": {Environment: "https://production.straddle.com", Account: "acct_a"},
	}
	stores := map[string]*Store{}
	for name, scope := range scopes {
		stores[name] = openScoped(t, dbPath, scope)
		body := json.RawMessage(`{"id":"ch_shared","description":"marker ` + strings.ReplaceAll(name, "/", "_") + `"}`)
		if _, _, err := stores[name].UpsertBatch("charges", []json.RawMessage{body}); err != nil {
			t.Fatalf("%s upsert: %v", name, err)
		}
		if err := stores[name].SaveSyncState("charges", "cursor-"+name, 1); err != nil {
			t.Fatal(err)
		}
	}
	// Rewriting one scope must not disturb the others' rows or FTS entries.
	if _, _, err := stores["sandbox/a"].UpsertBatch("charges", []json.RawMessage{json.RawMessage(`{"id":"ch_shared","description":"marker sandbox_a"}`)}); err != nil {
		t.Fatal(err)
	}

	for name, s := range stores {
		marker := "marker " + strings.ReplaceAll(name, "/", "_")
		got, err := s.Get("charges", "ch_shared")
		if err != nil || !strings.Contains(string(got), marker) {
			t.Fatalf("%s Get = %s, %v; want its own body", name, got, err)
		}
		list, err := s.List("charges", 0)
		if err != nil || len(list) != 1 {
			t.Fatalf("%s List = %d rows, %v; want 1", name, len(list), err)
		}
		if n, err := s.Count("charges"); err != nil || n != 1 {
			t.Fatalf("%s Count = %d, %v; want 1", name, n, err)
		}
		if status, err := s.Status(); err != nil || !reflect.DeepEqual(status, map[string]int{"charges": 1}) {
			t.Fatalf("%s Status = %v, %v", name, status, err)
		}
		hits, err := s.Search("marker", 10)
		if err != nil || len(hits) != 1 || !strings.Contains(string(hits[0]), marker) {
			t.Fatalf("%s Search = %q, %v; want only its own row", name, hits, err)
		}
		typed := 0
		if err := s.ScanTable(context.Background(), "charges", func(id string, data []byte) {
			typed++
			if !strings.Contains(string(data), marker) {
				t.Errorf("%s ScanTable returned %s", name, data)
			}
		}); err != nil || typed != 1 {
			t.Fatalf("%s ScanTable rows = %d, %v; want 1", name, typed, err)
		}
		if cursor, _, _, err := s.GetSyncState("charges"); err != nil || cursor != "cursor-"+name {
			t.Fatalf("%s cursor = %q, %v; want its own", name, cursor, err)
		}
	}

	if err := stores["sandbox/b"].ClearSyncCursors(); err != nil {
		t.Fatal(err)
	}
	if cursor, _, _, _ := stores["sandbox/a"].GetSyncState("charges"); cursor != "cursor-sandbox/a" {
		t.Fatalf("clearing sandbox/b cursors reset sandbox/a to %q", cursor)
	}
}

// TestLegacyRowsMigrateInPlaceAndStayHidden seeds a schema-v2 database the
// way the previous binary wrote it, then opens it with a real scope.
func TestLegacyRowsMigrateInPlaceAndStayHidden(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data.db")
	raw := rawDB(t, dbPath)
	for _, stmt := range []string{
		`CREATE TABLE resources (id TEXT NOT NULL, resource_type TEXT NOT NULL, data JSON NOT NULL, synced_at DATETIME DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (resource_type, id))`,
		`CREATE TABLE sync_state (resource_type TEXT PRIMARY KEY, last_cursor TEXT, last_synced_at DATETIME, total_count INTEGER DEFAULT 0)`,
		`CREATE TABLE "charges" ("id" TEXT PRIMARY KEY, "data" JSON NOT NULL, "synced_at" DATETIME DEFAULT CURRENT_TIMESTAMP, "response_type" TEXT)`,
		resourcesFTSCreateSQL,
		`INSERT INTO resources (id, resource_type, data) VALUES ('ch_old', 'charges', '{"id":"ch_old","description":"legacy marker"}'), ('cus_old', 'customers', '{"id":"cus_old"}')`,
		`INSERT INTO "charges" ("id", "data") VALUES ('ch_old', '{"id":"ch_old","description":"legacy marker"}')`,
		`INSERT INTO sync_state (resource_type, last_cursor) VALUES ('charges', 'legacy-cursor')`,
		`INSERT INTO resources_fts (id, resource_type, content) VALUES ('ch_old', 'charges', 'legacy marker')`,
		`PRAGMA user_version = 2`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("seed %q: %v", stmt, err)
		}
	}

	for attempt := range 2 {
		s := openScoped(t, dbPath, testScope)
		if v, err := s.SchemaVersion(); err != nil || v != StoreSchemaVersion {
			t.Fatalf("schema version = %d, %v", v, err)
		}
		if hidden, err := s.HiddenLegacyCount(); err != nil || hidden != 2 {
			t.Fatalf("attempt %d hidden = %d, %v; want 2", attempt, hidden, err)
		}
		if _, err := s.Get("charges", "ch_old"); err == nil {
			t.Fatal("legacy row is readable through a scope")
		}
		if hits, _ := s.Search("legacy", 10); len(hits) != 0 {
			t.Fatalf("legacy row is searchable: %q", hits)
		}
		if cursor, _, _, _ := s.GetSyncState("charges"); cursor != "" {
			t.Fatalf("legacy cursor resumed in a scope: %q", cursor)
		}
		_ = s.Close()
	}

	var rows, typed int
	if err := raw.QueryRow(`SELECT COUNT(*) FROM resources WHERE scope_environment = ''`).Scan(&rows); err != nil || rows != 2 {
		t.Fatalf("legacy resources kept = %d, %v; want 2", rows, err)
	}
	if err := raw.QueryRow(`SELECT COUNT(*) FROM "charges" WHERE scope_environment = ''`).Scan(&typed); err != nil || typed != 1 {
		t.Fatalf("legacy charges kept = %d, %v; want 1", typed, err)
	}

	resynced := openScoped(t, dbPath, testScope)
	if _, _, err := resynced.UpsertBatch("charges", []json.RawMessage{json.RawMessage(`{"id":"ch_old","description":"fresh marker"}`)}); err != nil {
		t.Fatal(err)
	}
	if got, err := resynced.Get("charges", "ch_old"); err != nil || !strings.Contains(string(got), "fresh") {
		t.Fatalf("resynced row = %s, %v", got, err)
	}
}

func queryAll(t *testing.T, snap *Snapshot, query string) ([]string, error) {
	t.Helper()
	rows, err := snap.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []string
	for rows.Next() {
		values := make([]sql.NullString, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		parts := make([]string, len(values))
		for i, v := range values {
			parts[i] = v.String
		}
		out = append(out, strings.Join(parts, "|"))
	}
	return out, rows.Err()
}

func TestSnapshotExposesOnlyItsScope(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "data.db")
	scopeA := Scope{Environment: "https://sandbox.straddle.com", Account: "acct_a"}
	for scope, marker := range map[Scope]string{
		scopeA: "alpha",
		{Environment: "https://sandbox.straddle.com", Account: "acct_b"}:    "bravo",
		{Environment: "https://production.straddle.com", Account: "acct_a"}: "prodmarker",
	} {
		s := openScoped(t, dbPath, scope)
		items := []json.RawMessage{
			json.RawMessage(`{"id":"cus_shared","name":"` + marker + `","external_id":"ext_` + marker + `"}`),
		}
		if _, _, err := s.UpsertBatch("customers", items); err != nil {
			t.Fatal(err)
		}
		_ = s.Close()
	}
	if _, err := rawDB(t, dbPath).Exec(`INSERT INTO resources (id, resource_type, data) VALUES ('legacy_row', 'customers', '{"name":"legacymarker"}')`); err != nil {
		t.Fatal(err)
	}

	snap, err := OpenSnapshot(context.Background(), dbPath, scopeA)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	cols, err := queryAll(t, snap, `SELECT name FROM pragma_table_info('customers') ORDER BY cid`)
	if err != nil {
		t.Fatal(err)
	}
	wantCols := []string{"id", "data", "synced_at", "response_type", "created_at", "email", "external_id", "name", "phone", "status", "type", "updated_at"}
	if !reflect.DeepEqual(cols, wantCols) {
		t.Fatalf("snapshot customers columns = %v, want the unscoped shape %v", cols, wantCols)
	}
	indexes, err := queryAll(t, snap, `SELECT name FROM pragma_index_list('customers') WHERE origin = 'c' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(indexes, []string{"idx_customers_created_at", "idx_customers_external_id", "idx_customers_updated_at"}) {
		t.Fatalf("snapshot indexes = %v", indexes)
	}

	for query, want := range map[string][]string{
		`SELECT id, name FROM customers`: {"cus_shared|alpha"},
		`WITH c AS (SELECT id FROM customers) SELECT r.id FROM c JOIN resources r ON r.id = c.id`:                            {"cus_shared"},
		`SELECT id FROM resources_fts WHERE resources_fts MATCH 'alpha'`:                                                     {"cus_shared"},
		`SELECT count(*) FROM resources WHERE data LIKE '%bravo%' OR data LIKE '%prodmarker%' OR data LIKE '%legacymarker%'`: {"0"},
		`SELECT count(*) FROM resources_fts WHERE resources_fts MATCH 'bravo OR prodmarker'`:                                 {"0"},
		`SELECT name FROM pragma_database_list`:                                                                              {"main"},
	} {
		got, err := queryAll(t, snap, query)
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("%s = %v, %v; want %v", query, got, err, want)
		}
	}

	for _, query := range []string{
		`SELECT * FROM src.customers`,
		`SELECT count(*) FROM pragma_table_info('customers', 'src')`,
		`INSERT INTO customers (id, data) VALUES ('x', '{}')`,
		`CREATE TEMP TABLE smuggled (x)`,
	} {
		if got, err := queryAll(t, snap, query); err == nil {
			t.Errorf("%s succeeded with %v; want rejection", query, got)
		}
	}
	stats, err := queryAll(t, snap, `SELECT DISTINCT name FROM dbstat`)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(stats)
	for _, table := range stats {
		if strings.Contains(table, "src") {
			t.Fatalf("dbstat exposed %s", table)
		}
	}
}
