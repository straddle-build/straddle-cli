// Copyright 2026 hello-keith. Licensed under Apache-2.0. See LICENSE.

package store

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

// Snapshot is a read-only, in-memory copy of one scope's rows for arbitrary
// local SQL. It keeps the unscoped table and column shapes, rowids, plain
// column indexes and the FTS index, and it has no attached database, so
// other scopes, legacy rows and FTS shadow data are unreachable from any
// statement run against it.
//
// ponytail: every snapshot copies the current scope's rows, so memory and
// open time grow with that scope (about 360 ms and 75 MB for 50,000 charges
// on a laptop). The upgrade path is SQLite's native authorizer over scoped
// views, once the driver exposes sqlite3_set_authorizer.
type Snapshot struct {
	db   *sql.DB
	conn *sql.Conn
}

// OpenSnapshot copies scope's rows from the migrated store at dbPath into a
// private in-memory database inside one read transaction, so concurrent
// sync writes never produce a partially copied snapshot. The source is
// detached and the connection set to query_only before it is returned.
func OpenSnapshot(ctx context.Context, dbPath string, scope Scope) (*Snapshot, error) {
	if err := validateScope(scope); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file::memory:")
	if err != nil {
		return nil, fmt.Errorf("opening snapshot database: %w", err)
	}
	// Each pooled connection would get its own empty in-memory database.
	db.SetMaxOpenConns(1)
	conn, err := db.Conn(ctx)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	snapshot := &Snapshot{db: db, conn: conn}
	if err := snapshot.load(ctx, abs, scope); err != nil {
		_ = snapshot.Close()
		return nil, err
	}
	return snapshot, nil
}

func (s *Snapshot) load(ctx context.Context, path string, scope Scope) error {
	source := (&url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}).String()
	if _, err := s.conn.ExecContext(ctx, `ATTACH DATABASE ? AS src`, source); err != nil {
		return fmt.Errorf("attaching local database: %w", err)
	}
	if _, err := s.conn.ExecContext(ctx, `BEGIN`); err != nil {
		return err
	}
	if err := s.copyScope(ctx, scope); err != nil {
		_, _ = s.conn.ExecContext(ctx, `ROLLBACK`)
		return err
	}
	for _, stmt := range []string{`COMMIT`, `DETACH DATABASE src`, `PRAGMA query_only = 1`} {
		if _, err := s.conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sealing snapshot (%s): %w", stmt, err)
		}
	}
	return nil
}

func (s *Snapshot) copyScope(ctx context.Context, scope Scope) error {
	rows, err := s.conn.QueryContext(ctx, `SELECT name FROM src.sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'resources_fts%' ORDER BY name`)
	if err != nil {
		return fmt.Errorf("listing local tables: %w", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("listing local tables: %w", err)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, table := range tables {
		if err := s.copyTable(ctx, table, scope); err != nil {
			return fmt.Errorf("copying %s: %w", table, err)
		}
	}
	return s.copyFTS(ctx, scope)
}

func (s *Snapshot) copyTable(ctx context.Context, table string, scope Scope) error {
	columns, err := tableColumns(ctx, s.conn, "src", table)
	if err != nil {
		return err
	}
	if !hasScopeColumns(columns) {
		return fmt.Errorf("table is not scoped; open the store once to migrate it")
	}
	var defs, names []string
	pk := make([]string, len(columns)+1)
	for _, c := range columns {
		if isScopeColumn(c.name) {
			continue
		}
		defs = append(defs, strings.TrimSpace(quoteIdent(c.name)+" "+c.declType))
		names = append(names, quoteIdent(c.name))
		if c.pkOrder > 0 {
			pk[c.pkOrder] = quoteIdent(c.name)
		}
	}
	var key []string
	for _, name := range pk {
		if name != "" {
			key = append(key, name)
		}
	}
	if len(key) > 0 {
		defs = append(defs, "PRIMARY KEY ("+strings.Join(key, ", ")+")")
	}
	quoted := quoteIdent(table)
	list := strings.Join(names, ", ")
	if _, err := s.conn.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE main.%s (%s)`, quoted, strings.Join(defs, ", "))); err != nil {
		return err
	}
	if _, err := s.conn.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO main.%s (rowid, %s) SELECT rowid, %s FROM src.%s WHERE scope_environment = ? AND scope_account = ? ORDER BY rowid`, quoted, list, list, quoted),
		scope.Environment, scope.Account,
	); err != nil {
		return err
	}
	return s.copyIndexes(ctx, table)
}

// copyIndexes recreates the table's CREATE INDEX indexes on plain columns,
// skipping the scope index columns that no longer exist in the snapshot.
func (s *Snapshot) copyIndexes(ctx context.Context, table string) error {
	rows, err := s.conn.QueryContext(ctx, `SELECT name, "unique" FROM pragma_index_list(?, 'src') WHERE origin = 'c'`, table)
	if err != nil {
		return err
	}
	type index struct {
		name   string
		unique bool
	}
	var indexes []index
	for rows.Next() {
		var ix index
		if err := rows.Scan(&ix.name, &ix.unique); err != nil {
			_ = rows.Close()
			return err
		}
		indexes = append(indexes, ix)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, ix := range indexes {
		colRows, err := s.conn.QueryContext(ctx, `SELECT name FROM pragma_index_info(?, 'src') ORDER BY seqno`, ix.name)
		if err != nil {
			return err
		}
		var cols []string
		skip := false
		for colRows.Next() {
			var col sql.NullString
			if err := colRows.Scan(&col); err != nil {
				_ = colRows.Close()
				return err
			}
			if !col.Valid || isScopeColumn(col.String) {
				skip = true
			}
			cols = append(cols, quoteIdent(col.String))
		}
		if err := colRows.Err(); err != nil {
			_ = colRows.Close()
			return err
		}
		if err := colRows.Close(); err != nil {
			return err
		}
		if skip || len(cols) == 0 {
			continue
		}
		unique := ""
		if ix.unique {
			unique = "UNIQUE "
		}
		if _, err := s.conn.ExecContext(ctx, fmt.Sprintf(`CREATE %sINDEX main.%s ON %s (%s)`, unique, quoteIdent(ix.name), quoteIdent(table), strings.Join(cols, ", "))); err != nil {
			return fmt.Errorf("recreating index %s: %w", ix.name, err)
		}
	}
	return nil
}

func (s *Snapshot) copyFTS(ctx context.Context, scope Scope) error {
	if _, err := s.conn.ExecContext(ctx, strings.Replace(resourcesFTSCreateSQL, "resources_fts", "main.resources_fts", 1)); err != nil {
		return fmt.Errorf("creating snapshot search index: %w", err)
	}
	_, err := s.conn.ExecContext(ctx,
		`INSERT INTO main.resources_fts (rowid, id, resource_type, content) SELECT rowid, id, resource_type, content FROM src.resources_fts WHERE scope_environment = ? AND scope_account = ?`,
		scope.Environment, scope.Account,
	)
	if err != nil {
		return fmt.Errorf("copying search index: %w", err)
	}
	return nil
}

func isScopeColumn(name string) bool {
	return name == scopeColumns[0] || name == scopeColumns[1]
}

// Query runs one read-only statement against the snapshot.
func (s *Snapshot) Query(ctx context.Context, query string) (*sql.Rows, error) {
	return s.conn.QueryContext(ctx, query)
}

func (s *Snapshot) Close() error {
	connErr := s.conn.Close()
	if err := s.db.Close(); err != nil {
		return err
	}
	return connErr
}
