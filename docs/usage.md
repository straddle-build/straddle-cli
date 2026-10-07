# Use the Straddle CLI

Configure authentication and account context, query synced data, and handle API responses in scripts. Start with the [README](../README.md) for installation and a first sandbox read.

## Install with another method

The CLI publishes prebuilt binaries for macOS, Linux, and Windows on x64 and ARM64. Choose the method that fits your environment.

### Use the shell installer

On macOS or Linux, install a checksum-verified release:

```sh
curl -fsSL https://raw.githubusercontent.com/straddle-build/straddle-cli/main/install.sh | sh
```

The installer writes to `~/.local/bin`. Add that directory to `PATH`, or set `STRADDLE_INSTALL_DIR` to another installation directory. It verifies the archive against the release's `checksums.txt` before installation.

### Download a binary

Download the archive for your system from [GitHub releases](https://github.com/straddle-build/straddle-cli/releases), extract it, and put the binary on `PATH`. On macOS or Linux, give the extracted binary execute permission if needed:

```sh
chmod +x straddle
```

macOS release binaries from `1.0.3` onward are Developer ID signed and notarized. Gatekeeper needs a network connection to retrieve the notarization ticket on the first launch of a quarantined download. Update an earlier installation to a signed release.

### Install with Go or build from source

Use the Go version required by [`go.mod`](../go.mod) to install from the module:

```sh
go install github.com/straddle-build/straddle-cli/cmd/straddle@latest
```

To build a checkout:

```sh
git clone https://github.com/straddle-build/straddle-cli
cd straddle-cli
make build
bin/straddle --help
```

See [Contributing](../CONTRIBUTING.md) for the development checks.

## Configure authentication

Set a sandbox API key from the [Straddle dashboard](https://dashboard.straddle.com). In Bash or Zsh:

```sh
export STRADDLE_API_KEY="YOUR_SANDBOX_API_KEY"
export STRADDLE_ENVIRONMENT=sandbox
```

In PowerShell:

```powershell
$env:STRADDLE_API_KEY = "YOUR_SANDBOX_API_KEY"
$env:STRADDLE_ENVIRONMENT = "sandbox"
```

To save a token across shell sessions, replace the placeholder in this command:

```sh
straddle auth set-token "YOUR_SANDBOX_API_KEY"
```

`auth set-token` replaces all saved credentials in the configuration file, including credentials saved by earlier versions. The file uses owner-only permissions. An exported `STRADDLE_API_KEY` takes precedence; the command saves only the token you pass. The token argument can remain in shell history, so use an environment variable supplied by your secret manager when that matters for your workflow.

Saved credentials live at `~/.config/straddle/config.toml`. Set `STRADDLE_CONFIG` or use `--config` to choose another file. Saves are atomic. A resolvable symlink keeps its link and replaces the target; a dangling or unresolvable link causes the save to fail.

`straddle auth logout` clears saved credentials, including a saved notification polling token. Unset exported credentials separately when ending a shell session's authentication.

### Choose an API environment

The default base URL is `https://{environment}.straddle.com`. `STRADDLE_ENVIRONMENT` replaces the placeholder and defaults to `sandbox`.

| Environment | Base URL | Key |
| --- | --- | --- |
| Sandbox | `https://sandbox.straddle.com` | Sandbox API key |
| Production | `https://production.straddle.com` | Production API key |

`STRADDLE_BASE_URL` overrides the base URL. An existing override continues to control the target even after you change `STRADDLE_ENVIRONMENT`. API targets require HTTPS, with HTTP allowed for loopback test servers.

### Set request headers

Configure static headers under `headers` in `config.toml`. Command-specific header values take precedence.

API requests use `User-Agent: straddle-cli/<version>`. Override it through the configured headers or a command-specific header. Notification polling and forwarding add `(events)`, webhook output delivery adds `(deliver)`, and feedback submissions add `(feedback)` to the CLI user agent.

## Work with embedded accounts

Set your integration type with `straddle setup --type account`, `saas`, or `marketplace`. The CLI saves this choice in `platform.toml`, separately from credentials.

The following table shows when the CLI uses `Straddle-Account-Id`.

| Integration | Account header behavior |
| --- | --- |
| Direct account | Omits the header. |
| SaaS platform | Scopes customer, paykey, bridge, payment, review, and funding-event operations that accept the header. |
| Marketplace | Scopes payment and funding-event operations. Customer, paykey, and bridge operations use the platform context without the header. |

Account-management and onboarding operations carry account IDs in their paths or bodies. Platform ID, Organization ID, and Account ID identify different resources.

For platform operations, select an acting account or override it for a single command:

```sh
straddle use-account "ACCOUNT_ID"
straddle customers list --account "ACCOUNT_ID" --json
```

The customer-list example applies to a SaaS integration. For a marketplace, omit `--account` on customer operations. Run `straddle use-account --clear` to clear the saved acting account.

The default platform file is `~/.config/straddle/platform.toml`. `STRADDLE_PLATFORM_CONFIG` overrides it. Otherwise, `STRADDLE_CONFIG` relocates it beside the selected configuration file.

## Read all pages

List commands return one page by default and warn when response totals show more results. Read every remaining customer page with:

```sh
straddle customers list --all --json
```

For numbered endpoints, `--page-number` selects the starting page and `--page-size` controls page size. The CLI preserves filters and account scoping across requests. When an Embed response omits totals, a short page ends the read.

The CLI returns an error if a later request fails or pagination cannot prove that the result is complete. Numbered reads stop at 10,000 pages; narrow the filters or increase the supported page size for larger result sets.

`--rate-limit` sets a requests-per-second maximum shared by concurrent calls. The limiter slows after HTTP `429` responses and recovers toward that maximum. `0` disables the limit. Invalid pacing values fail before a request.

## Sync and query local data

Create or refresh the local mirror before querying it:

```sh
straddle sync
straddle search "example@example.com" --data-source local --json
```

`sync` reads API data into SQLite and keeps checkpoints for later incremental runs. Its summary lists denied resources and other failures. Add `--strict` when a script needs a nonzero exit for any resource failure:

```sh
straddle sync --strict
```

The default exit policy can succeed when some resources sync and other non-critical resources fail. Every selected resource failing always produces a nonzero exit. Use the sync summary to confirm which resources reached the local store.

`--resources` selects resource types. Selecting a parent also syncs its dependent resources. Selecting only a dependent requires its parent rows to exist from an earlier sync. Use `straddle sync --help` for full resync, time windows, page limits, and concurrency options.

The local store separates rows by API environment and acting account. Changing `use-account`, `--account`, or the environment selects another context. Sync that context before using local reports.

### Choose live or local reads

For read commands that support data-source selection, use the following modes.

| Flag | Behavior |
| --- | --- |
| `--data-source auto` | Uses the live API where available and writes successful reads to the local store, with local fallback on supported network failures. |
| `--data-source live` | Requires an API response and skips local-store write-through. |
| `--data-source local` | Reads synced data. |

`search` uses local full-text search when an API search endpoint is unavailable. `reconcile`, `pipeline`, `returns`, `review-queue`, `cashflow`, and `expiring` analyze the synced store. `sql` runs read-only queries against that store.

### Analyze payment activity

Use the following commands after a sync:

```sh
straddle reconcile --outstanding --json
straddle pipeline --cancelable --json
straddle returns --days 30 --repeat-offenders --json
straddle cashflow --days 30 --weekly --json
straddle review-queue --type customers --json
straddle expiring --days 14 --json
```

Reconciliation matches payments to funding events. Pipeline groups payment statuses; `--cancelable` selects `created`, `scheduled`, and `on_hold`. Payments that reach `pending` are locked, so confirm current status through the API before taking an action based on a local report.

Returns groups failed and reversed payments with their recorded reason codes. Cashflow groups charge and payout volume by each payment's creation date, including days with no activity. Review queue sorts customers and paykeys awaiting review by age. Expiring identifies paykeys near expiry and blocked paykeys eligible for unblocking.

## Choose output and handle errors

Use `--json` for structured output, `--compact` for fewer fields, or `--select id,status` to choose fields on each returned resource. A selector that matches nothing reports the mismatch. Envelope paths such as `results.id` are not resource selectors.

Human-readable output is the terminal default. Piped output stays JSON, including with `--human-friendly`. Explicit machine-format flags take precedence over human formatting.

`--agent` sets `--json --compact --no-input --no-color --yes`. Explicit values override these defaults. For example, `--agent --json=false --compact=false` selects human output in a terminal. Commands still accept their inputs through flags or stdin.

`--dry-run` prints an API request without sending it. Use `--data-source live` for API read previews to skip local-store initialization and write-through. Required inputs and structured values are still validated. Write commands accept JSON through `--stdin` when their help lists it. `--idempotent` handles an already-existing create result as a successful no-op; `--ignore-missing` does the same for a missing delete target.

The CLI uses the following exit codes.

| Code | Meaning |
| --- | --- |
| `0` | Success |
| `1` | Other command failure |
| `2` | Usage error |
| `3` | Resource not found |
| `4` | Authentication error |
| `5` | API error |
| `7` | Rate limited |
| `10` | Configuration error |

## Read and forward notification events

Get the notification polling URL and token from the Straddle dashboard. Use the sandbox endpoint for development. Set `STRADDLE_POLLING_URL` to the URL containing `{consumer_id}` and `STRADDLE_POLLING_TOKEN` to its token. Alternatively, save `polling_url` and `polling_token` in `config.toml`.

Read new events and forward their payloads to a local handler:

```sh
straddle events tail --consumer local-handler --from-now --forward-to http://localhost:3000/webhooks
```

The polling token authenticates the notification endpoint separately from `STRADDLE_API_KEY`. `--polling-url` overrides the environment and saved URL. Use a distinct consumer name for each terminal or teammate.

A new consumer replays retained history unless you pass `--from-now`. A consumer with a saved position always resumes there. `--account-id` filters displayed events; the consumer still advances past other accounts' events.

Tail processes events in order and commits the consumer position after printing or forwarding. A failing forwarding target retries with increasing delays, starting at `--interval` and capped at 30 seconds. Later events stay uncommitted during that retry.

Terminal output shows one event per line. Piped output, `--json`, and `--agent` produce newline-delimited JSON with `offset`, `timestamp`, `event_type`, and `payload`. Forwarding also adds `forward_status`.

## Browse and call API endpoints

Inspect interfaces and command options from the installed CLI:

```sh
straddle api
straddle api customers
straddle customers list --help
```

Use the raw API command for an endpoint without a dedicated command. After configuring sandbox authentication, preview a customer read:

```sh
straddle api get /v1/customers --param page_size=1 --dry-run --agent
```

Raw requests share the friendly commands' authentication, account scoping, request preview, verification, redaction, and output behavior. The method can be `GET`, `POST`, `PUT`, `PATCH`, or `DELETE`. Repeat `--param key=value` for query parameters or `--header key=value` for headers. `--stdin` accepts a JSON request body for `POST`, `PUT`, and `PATCH`.

### Supply structured values and files

Use the current command help for request fields. The following input rules apply to the corresponding commands:

- `customers create` accepts JSON objects for `--compliance-profile` and `--metadata`.
- `customers update` requires a valid customer status, including during a dry run. With `--stdin`, include `status` in the JSON body; stdin replaces body flags and profile values. Read the existing customer first if you need to preserve its status.
- `linked-bank-accounts create` expects `--purposes` as a JSON array such as `'["charges","payouts"]'`.
- `charges refund` creates a linked payout for a paid charge. Inspect the request fields with `straddle charges refund --help` before preparing it.
- `charges upload-authorization-proof` and `payouts upload-authorization-proof` accept `--file` for PDF, PNG, JPEG, DOC, or DOCX documents up to 10 MiB. File validation runs locally; `--dry-run` describes the file without printing its contents.

## Troubleshoot a command

Use the following checks for common failures.

| Symptom | Next step |
| --- | --- |
| Authentication error or HTTP `401` | Use `straddle doctor` to inspect credential presence and the selected target. Confirm that the key belongs to that environment. |
| Request reaches the wrong environment | Inspect `STRADDLE_ENVIRONMENT` and any `STRADDLE_BASE_URL` override. |
| Platform request returns HTTP `403` or unexpected account data | Inspect `straddle setup` and `straddle use-account`, then check the account-scoping rules for that operation. |
| Local search or reconciliation is empty | Sync the selected environment and acting account, then inspect the sync summary. |
| A payment cannot be cancelled | Read its current status. A payment in `pending` or a later state is locked. |
| A paykey has expired | Inspect `straddle expiring` and reconnect the bank account as appropriate before retrying a payment. |
| HTTP `423` while tailing events | Give each reader a distinct `--consumer`. A prior uncommitted batch can retain its lease for about five minutes; tail retries until it expires. |
| Resource not found | Confirm the ID and account context with the corresponding list command. |

For a reproducible CLI problem, [open an issue](https://github.com/straddle-build/straddle-cli/issues) with the CLI version, command, and sanitized error. Follow the [security policy](../SECURITY.md) for vulnerabilities.
