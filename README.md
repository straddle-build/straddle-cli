# Straddle CLI

Use Straddle's Pay by Bank and Embed APIs from your terminal, scripts, or coding agent. The `straddle` command also syncs payment data to a local SQLite store for search, reconciliation, return analysis, and cashflow reports.

## Install

On macOS with Homebrew, install the CLI and check its version:

```sh
brew install straddle-build/tap/straddle
straddle --version
```

The version command prints `straddle` followed by the installed version.

With Node.js 18 or later, use npm on macOS, Linux, or Windows:

```sh
npm install -g @straddlecom/cli
straddle --version
```

To inspect the commands without a global installation:

```sh
npx @straddlecom/cli --help
```

Both installation methods use a prebuilt binary. See [npm installation details](npm/README.md), [other installation methods](docs/usage.md#install-with-another-method), or [release downloads](https://github.com/straddle-build/straddle-cli/releases) for shell, Go, and source builds.

## Try an offline command

Print the supported sandbox outcomes before configuring an API key:

```sh
straddle sandbox outcomes --json
```

The result contains `customers`, `paykeys`, and `charges_payouts` arrays with the outcome values used in sandbox tests. This command reads the CLI's built-in reference.

## Authentication

Get a sandbox API key from **Developer > API Keys** in the [Straddle dashboard](https://dashboard.straddle.com). In Bash or Zsh, replace the placeholder with that key and select sandbox:

```sh
export STRADDLE_API_KEY="YOUR_SANDBOX_API_KEY"
export STRADDLE_ENVIRONMENT=sandbox
```

Sandbox keys authenticate against `https://sandbox.straddle.com`. Production uses `https://production.straddle.com` and a separate key. The CLI defaults to sandbox; setting the environment explicitly makes the target clear in scripts.

To save a key locally, use `straddle auth set-token`. An exported `STRADDLE_API_KEY` takes precedence over a saved token. See [authentication and configuration](docs/usage.md#configure-authentication) for PowerShell, saved credentials, and configuration paths, or the [API authentication guide](https://docs.straddle.com/api-reference/authentication) for key management.

## Choose your account context

Set the integration type that matches your key and application. The CLI saves this choice locally and uses it to decide when to send `Straddle-Account-Id`.

| Integration | Use it when | Command |
| --- | --- | --- |
| Direct account | Your business collects or sends its own payments | `straddle setup --type account` |
| SaaS platform | Your clients own their customers | `straddle setup --type saas` |
| Marketplace | Your platform owns customers across sellers | `straddle setup --type marketplace` |

For SaaS or marketplace integrations, select the embedded account for account-scoped operations. Replace `ACCOUNT_ID` with its account ID:

```sh
straddle use-account "ACCOUNT_ID"
```

`use-account` keeps that selection until you change it. `--account` overrides it for one command. Run `straddle setup` or `straddle use-account` without arguments to inspect the saved context. See [account scoping](docs/usage.md#work-with-embedded-accounts) for the operations each integration type scopes.

## Read sandbox customers

With your sandbox key and account context configured, check connectivity and request one customer:

```sh
straddle doctor
straddle customers list --page-size 1 --data-source live --json
```

The list command returns a JSON response from the sandbox API. An empty list is valid when the selected account has no customers. `doctor` reports configuration and connectivity; the authenticated list request confirms that your key can read customers.

List commands return one page by default. Add `--all` to fetch the remaining pages. Use each command's help to find its filters:

```sh
straddle customers list --help
```

## Sync and investigate payments

Sync data before using local search and analysis:

```sh
straddle sync
straddle reconcile --outstanding --json
```

`sync` reads API resources into the local SQLite store. `reconcile` then lists synced payments that aren't tied to a funding event. Read the sync summary for resources your key couldn't read or that failed to sync.

The store separates data by API environment and acting account. After changing either context, sync that context before using local reports. Synced data can contain customer and payment information; see [local data storage](SECURITY.md#local-data).

The following commands answer common operational questions using synced data.

| Task | Command |
| --- | --- |
| Find payments still in a cancelable state | `straddle pipeline --cancelable --json` |
| Inspect failed and reversed payments | `straddle returns --days 30 --json` |
| Compare charge and payout volume | `straddle cashflow --days 30 --json` |
| Find customers and paykeys awaiting review | `straddle review-queue --json` |
| Find expiring or unblock-eligible paykeys | `straddle expiring --days 14 --json` |

A local report reflects the last sync. Check a payment's current API status before acting on it. See [local data and analytics](docs/usage.md#sync-and-query-local-data) for freshness, strict sync checks, and search options.

## Output formats

Commands use human-readable output in a terminal and JSON when piped. Select JSON explicitly for scripts:

```sh
straddle customers list --json
straddle customers list --json --select id,name,status
```

`--select` applies to fields on each returned resource. `--compact` returns a smaller JSON result. See [output and exit codes](docs/usage.md#choose-output-and-handle-errors) for formatting precedence and error handling.

## Use the CLI with an agent

Find a command by capability, then inspect its options:

```sh
straddle which "settlement" --json
straddle reconcile --help
straddle agent-context --pretty
```

`which` returns matching commands from the CLI's capability index. `agent-context` describes the command tree, flags, and selected runtime context as versioned JSON.

For API commands, `--dry-run` validates the inputs and prints the request before you send it:

```sh
straddle customers list --page-size 1 --data-source live --dry-run --agent
```

`--data-source live` keeps this read preview out of the local store. `--agent` defaults to `--json --compact --no-input --no-color --yes`. Because it also enables `--yes`, use `--dry-run` when you want a request preview. Explicit flag values override agent defaults.

The repository's [CLI skill](SKILL.md) teaches coding agents to discover commands and use the CLI. Install it with the [skills CLI](https://github.com/vercel-labs/skills):

```sh
npx skills add straddle-build/straddle-cli
```

For a guided integration that coordinates planning, implementation, and testing, start with [Straddle Wizard](https://github.com/straddle-build/wizard).

## Continue building

Use the following guides for the next task:

- [Read and forward notification events](docs/usage.md#read-and-forward-notification-events) to test a local event handler.
- [Browse and call API endpoints](docs/usage.md#browse-and-call-api-endpoints) to discover the full command tree or use a raw API path.
- [Troubleshoot a command](docs/usage.md#troubleshoot-a-command) for authentication, account context, pagination, and local data issues.
- [Contribute](CONTRIBUTING.md) for development prerequisites and checks, and [Operations](OPERATIONS.md) for contract synchronization and releases.

Report bugs in [GitHub issues](https://github.com/straddle-build/straddle-cli/issues). Use the [security policy](SECURITY.md) to report vulnerabilities. Licensed under [Apache-2.0](LICENSE).
