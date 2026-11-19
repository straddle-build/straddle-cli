# Production-write confirmation (EG-1002)

Production consent is a separate decision from output format and unattended execution. `--agent` means JSON, compact output, no color, and no interactive input. It never means permission to write to production.

## What requires consent

A production write is a live API request with method `POST`, `PUT`, `PATCH`, or `DELETE` whose resolved destination is outside the known sandbox (`sandbox.straddle.com`) and loopback (`localhost` or a loopback IP). Unknown remote destinations require consent as well. This conservative rule covers production aliases and custom base URLs without maintaining a list of commands or production hosts. The actual URL wins over `STRADDLE_ENVIRONMENT`: a configured production URL cannot become exempt merely because the environment variable still says sandbox. Loopback remains available for local mock servers.

The client already distinguishes writes from read-only requests, including its explicit POST-query read path. Consent follows that write classification. Generated endpoints, hand-authored endpoints, raw `api` calls, JSON stdin, multipart uploads, and JSONL imports all use the shared client. Local configuration and SQLite writes are outside this production API rule. Output delivery and feedback are separate non-API facilities and are unchanged.

`--dry-run` and the existing verification no-op mode do not send a write and do not ask for production consent. Production reads do not ask either. A write-preserving redirect is checked against its actual destination, so a sandbox response cannot silently authorize a production write.

## A person at a terminal

Before the first production write, stderr shows the method, destination URL without query or credentials, command path, and selected acting account when present. The prompt asks:

```text
Production write: DELETE https://production.straddle.com/v1/customers/<id>
Command: straddle customers delete
Type yes to allow this command's writes to https://production.straddle.com:
```

Type `yes` to continue. Blank input, a different answer, an input error, or end of input refuses the write. There is no default approval. The confirmation covers this command invocation's writes to that origin, including all import records; retries do not ask again. A different origin needs a separate confirmation. The prompt does not show request bodies, credentials, or query parameters. Use `--dry-run` to inspect request contents before executing.

## A script in CI

Pass `--yes` explicitly to authorize the invocation's production writes, normally together with `--no-input`. For example:

```sh
straddle customers delete <id> --no-input --yes
```

`--no-input` controls interaction only. It never grants consent. `--yes=false` does not grant consent. Profiles already exclude `--yes` and `--agent`; that restriction remains in place. There is no environment variable or saved setting for production consent.

## An agent

Inspect the intended write with `--dry-run --agent`. When authorized to execute it, use `--agent --yes`. Agent mode alone refuses a production write, including when the agent happens to run in a terminal. An explicit `--no-input=false` can opt back into the same terminal prompt; it still does not grant consent.

## No terminal

Without `--yes`, stdin must be a real terminal for the prompt to run. Pipes, redirected files, and `/dev/null` are not interactive consent. Piping `yes` does not authorize a production write. JSON or JSONL supplied through stdin remains request input; the CLI does not open `/dev/tty` to seek approval elsewhere. Missing or refused consent exits 2 before the production request is sent and emits guidance to pass `--yes` deliberately or use `--dry-run`.

## Compatibility changes to review

- Production scripts and agents that previously used only `--agent` must add explicit `--yes`. They now fail before sending the write.
- Production commands that previously sent a write immediately now prompt on terminal stdin and refuse without terminal stdin unless `--yes` is present.
- Custom remote API hosts are treated conservatively as production; their unattended writes also need `--yes`. Local loopback mock servers and the known sandbox remain exempt.
- `profile delete --agent` now also needs explicit `--yes`, because removing agent-implied consent applies to the existing local deletion guard. The profile-specific guard is otherwise unchanged.
- Successful JSON and agent response formats, account-header policy, request idempotency, sandbox usage, and dry-run output retain their existing behavior.

## Implementation and evidence

The root pre-run resolves profiles and agent defaults, then installs a consent policy for that invocation. `rootFlags.newClient` attaches it to the shared client and redirect path. The client's live-write boundary calls it once before entering the existing idempotency and retry loop. A refused redirect confirmation is non-retryable, so the source request is not repeated. Import treats a consent refusal as a command error rather than continuing through the remaining records.

Regression tests execute the real command tree with only the HTTP boundary replaced. They cover every mutation verb, explicit script and agent consent, profile restrictions, generated and hand-authored commands, multipart, import, target overrides, redirects, reads, dry runs, and verification no-ops. A child command under a real pseudoterminal proves that a person is asked and that affirmative, negative, and blank answers affect whether a request is sent.

Local validation passed with the pinned Go 1.26.10: the full `go test ./...` suite, CLI build, `go vet ./...`, golangci-lint, the full-history secret scan, and `doctor --json` / `agent-context --pretty` smoke checks. Help and command-tree snapshots changed only for the two flag descriptions; successful response snapshots matched.

The vulnerability scan reported the pre-existing standard-library issue [GO-2026-6859](https://pkg.go.dev/vuln/GO-2026-6859) in the pinned Go 1.26.10, fixed in Go 1.26.11. This change leaves the toolchain and release configuration untouched. That scan remains a delivery limitation for the separate validation pipeline to resolve.
