# Local store and sync model

## Overview

The CLI keeps a local SQLite store so it can search, reconcile, and analyze Straddle data offline. This is a major product feature, not just a cache.

The store lives in `internal/store/` and is opened through `internal/store/store.go`.

## What the store is used for

The local database supports:

- search across synced resources
- analytics commands such as reconciliation and cashflow reporting
- the `search` and `sql` commands
- offline read paths when live API access is unavailable or unnecessary

## Storage characteristics

Key implementation details from `internal/store/store.go`:

- it uses `modernc.org/sqlite`, so the repo stays pure Go and cross-compilable
- the database runs in WAL mode with a 5 second busy timeout, set through the driver's `_pragma` DSN parameters
- write access is serialized with a mutex
- a `resources_fts` FTS5 virtual table provides full-text search
- the schema version is tracked with `PRAGMA user_version`

## Scope

Every row belongs to one scope: the API environment (the lowercase origin of the resolved base URL, so sandbox, production and a local server never mix) and the selected platform acting account (`--account`, else `use-account`; direct accounts always use none). An empty account is its own platform-level context, not a wildcard. The scope is fixed when a `Store` opens and every read, search, sync cursor and write filters or stamps it; the raw database handle is not exposed.

Local scope is separate from the `Straddle-Account-Id` header. A marketplace fetches customers without the header, but rows captured while acting as one account stay under that account. Sync applies the header policy per request and sends the acting account only where the operation accepts it. Live reads skip the HTTP response cache when the header sent differs from the local account, so a cached response never crosses acting accounts.

Schema version 3 rebuilds each table in place with `scope_environment` and `scope_account` leading its primary key, so the same resource ID coexists across scopes. Rows written before scoping keep an empty environment; they stay in the file, no scope can read them, and `doctor` and local-read provenance (`meta.hidden_legacy_records`, present only when nonzero) report their count.

`straddle sql` runs on an in-memory snapshot of the current scope, copied in one read transaction with the unscoped column order, rowids, plain column indexes and the FTS index, then detached from the file and set `query_only`. Its cost grows with the current scope's row count.

## Schema evolution

The store includes migration and backfill logic so older databases can be upgraded in place. The code explicitly handles added columns that newer binaries expect.

That means changes to the store are operationally sensitive:

- adding/removing columns affects migrations
- typed resource tables and fallback resource storage both matter
- search and analytics code may depend on those tables being populated

## Sync model

Although sync orchestration spans more than one file, the overall pattern is straightforward:

1. authenticate and connect to Straddle
2. fetch resources from the API
3. upsert them into the local database
4. use the synced store for subsequent search/analytics commands

If the store is empty, commands like search or reconciliation will not be useful until sync runs.

Successful API reads in the default `--data-source auto` mode also update the local store. The write-through path stores list results and individual resources returned inside the API's `data` envelope, making them available for later offline lookup and search without another sync.

The write-through path never stores unmasked or revealed responses because those responses contain sensitive data. A local write failure does not fail the API request.

## Where to start in code

- `internal/store/store.go`
- `internal/cli/sync.go`
- `internal/cli/search.go`
- `internal/cli/straddle_reconcile.go`
- `internal/cli/straddle_cashflow.go`
