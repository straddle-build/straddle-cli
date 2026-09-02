# Straddle CLI

Use Straddle's Pay by Bank and Embed APIs from your terminal. The `straddle` command also supports a local SQLite copy of payment data, offline search, settlement and return analytics, and structured JSON output for scripts and agents.

This package installs the standalone Go CLI. For the TypeScript SDK, use [`@straddlecom/straddle`](https://www.npmjs.com/package/@straddlecom/straddle).

## Install

```sh
npm install -g @straddlecom/cli
straddle --help
```

Or try it without a global installation:

```sh
npx @straddlecom/cli --help
```

Requires Node.js 18 or newer and macOS, Linux, or Windows on x64 or ARM64. Installation needs internet access to GitHub Releases, plus `tar` on macOS/Linux or PowerShell on Windows. You do not need Go installed.

## Get started

```sh
straddle --version
straddle doctor --help
straddle doctor
```

`doctor` checks your configuration and connectivity. It does not create payments. Without an API key, it reports the missing configuration. Set `STRADDLE_API_KEY` and `STRADDLE_ENVIRONMENT=sandbox` to work with your sandbox account. Never commit API keys.

For authentication, account setup, and available commands, see the [CLI documentation](https://github.com/straddle-build/straddle-cli#authentication). Use `--help` on any command to inspect its options before running it.

## How installation works

The installer downloads the Go binary for your operating system and architecture from the matching [GitHub release](https://github.com/straddle-build/straddle-cli/releases). It checks the archive's SHA-256 digest against that release's `checksums.txt` before extracting it. The `straddle` command then runs that binary.

If npm install scripts are disabled, the launcher attempts this download the first time you run `straddle`.

## Links

- [Source and full documentation](https://github.com/straddle-build/straddle-cli)
- [Release downloads](https://github.com/straddle-build/straddle-cli/releases)
- [Report an issue](https://github.com/straddle-build/straddle-cli/issues)

Licensed under Apache-2.0. See [LICENSE](LICENSE).
