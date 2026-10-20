# Operations

Local development commands, release process, and operational pointers for the Straddle CLI repo.

## Local development

| Task | Command |
|---|---|
| Build | `go build -o bin/straddle ./cmd/straddle` (or `make build`; never build to `/tmp`) |
| Test | `go test ./...` (or `make test`) |
| Vet | `go vet ./...` |
| Lint | `golangci-lint run` (or `make lint`) |
| Format | `gofmt -w <changed files>` (changed files only) |
| Contract lock | `go run ./cmd/gen-endpoint verify-lock --spec spec.yaml` |
| Endpoint coverage | `go run ./cmd/gen-endpoint check --spec spec.yaml --repo .` |
| Contract mock | `STRADDLE_CONTRACT_MOCK=1 go test ./internal/cli -run TestContractMockServer -v` |
| Vulnerability scan | `make vuln` |
| Secret scan | `go run github.com/zricethezav/gitleaks/v8@latest detect --log-opts=--all` |
| Runtime smoke | `go run ./cmd/straddle doctor --json` and `go run ./cmd/straddle agent-context --pretty` |
| Install to PATH | `make install` (`go install ./cmd/straddle`) |

See [Output Formats](docs/usage.md#choose-output-and-handle-errors) for agent defaults, explicit overrides, and human-friendly output behavior.

The contract mock check runs the registered `customers create --stdin --no-cache` command against Scalar using the `application/json` request example in `spec.yaml`. It prefers a media-type `example`, then the alphabetically first named media-type example, resolving local `#/components/examples/` references. When the media type has no example, it falls back to the referenced component schema's singular `example` or first plural `examples` value.

## CI

`.github/workflows/ci.yml` runs build, test, golangci-lint, govulncheck, and gitleaks (full history) on pushes to `main` and all PRs. PRs are additionally gated through the no-mistakes pipeline (`.no-mistakes.yaml`).

## API sync

`spec.yaml` contains the exact bytes of the immutable Scalar release named by `contract.lock.json`. The tooling derives each supported endpoint's command surface and compares drift field by field:

```bash
go run ./cmd/gen-endpoint verify-lock --spec spec.yaml
go run ./cmd/gen-endpoint surfaces --spec spec.yaml --agent
go run ./cmd/gen-endpoint check --spec spec.yaml --repo .
go run ./cmd/gen-endpoint drift --base spec.yaml --head <released-spec> --repo . --agent
go run ./cmd/gen-endpoint generate --spec spec.yaml --repo . --agent
```

When `allOf` members (or a `$ref` and its sibling keywords) each declare an `enum`, the derived flag accepts only values every member allows. If no value satisfies every member, the field gets no flag and its operation is reported unsupported with a `conflicting allOf enums` reason, the same treatment as conflicting `allOf` types.

`.github/workflows/api-sync.yml` receives `straddle-contract-published`, accepts an exact version for manual recovery, and checks Scalar daily for a missed event. Discovery may read Scalar's current release, but synchronization always downloads the exact versioned artifact. Publisher-triggered runs verify the publisher-provided digest before drift or generation. Scheduled and manual recovery runs compute the digest from the exact downloaded artifact and require no checksum input.

Every new contract version updates the YAML and lock in a normal human-reviewed PR. The workflow then regenerates every supported contract-derived endpoint file, overwrites existing generated files, and deletes owned generated files for operations that left the contract or became unsupported. Review evidence includes field-level flag additions, removals, and changes, plus counts for generated, deleted, unchanged, and unsupported operations. Repeated events and scheduled runs are green no-ops when the version and bytes already match `main` or the version-specific branch of an open synchronization PR. A stale branch without an open PR does not suppress PR creation. Changed bytes for an already-seen version fail. The workflow never auto-merges.

Configure `API_SYNC_BOT_TOKEN` with contents and pull-request write access. It creates the synchronization PR. After that PR merges, `.github/workflows/api-sync-release.yml` creates the next CLI patch tag only when the merge changed a file other than `spec.yaml` and `contract.lock.json`. A contract-only merge records the skipped release in the workflow summary. The existing release workflow publishes each new tag through every configured CLI distribution.

## Release

Releases are cut from `main` by tag. A merged version-specific `automation/api-sync-*` PR creates the next patch tag automatically; other releases may still be tagged manually.

1. Push a `vX.Y.Z` tag, or merge the generated contract synchronization PR.
2. `.github/workflows/release.yml` runs tests on a macOS runner, imports the Developer ID certificate into a temporary keychain, then GoReleaser builds the six os/arch binaries. A build post hook (`scripts/macos-sign-notarize.sh`) signs each darwin binary with hardened runtime and a secure timestamp and requires Apple notarization status `Accepted` before GoReleaser archives, checksums or publishes anything, so `checksums.txt` covers the signed bytes. Any missing credential, signature check or non-Accepted notarization fails the release before publication. GoReleaser then publishes the GitHub release (6 os/arch archives + `checksums.txt`). When `HOMEBREW_TAP_GITHUB_TOKEN` is configured, GoReleaser pushes `Casks/straddle.rb` to a `straddle-<version>` branch of `straddle-build/homebrew-tap` and opens a pull request into its `main`, which accepts changes only through pull requests. The token needs contents and pull-request write access to the tap. A maintainer merges that PR after the tap's CI passes; the generated stanza order does not satisfy the tap's `brew audit --strict`, so apply `brew style --fix` on the PR branch first. A cask step failure happens after the GitHub release is published and stops the workflow before npm, so recover npm as described below.
3. `scripts/npm-platform-packages.sh dist <version> <out-dir>` checks each archive against `checksums.txt` and builds seven npm packages from them: six scriptless binary packages, `@straddlecom/cli-<platform>-<arch>` for darwin, linux and win32 on arm64 and x64, and the `@straddlecom/cli` wrapper, which lists all six as optional dependencies at the same exact version. The job publishes the six platform packages, then the wrapper, with npm trusted publishing and provenance. A publish is skipped when that version is already on npm, so rerunning a partially published release finishes the rest. An npm publication failure fails the workflow.
4. `install.sh` and `go install github.com/straddle-build/straddle-cli/cmd/straddle@latest` resolve the new release with no further action.

Local dry run: `make release-snapshot` builds everything into `dist/` without publishing. Snapshots skip signing and notarization, so their darwin binaries are unsigned development builds, not release candidates.

### Apple signing setup

The release job needs these GitHub Actions repository secrets. The job runs only on pushed `v*` tags, so pull requests and npm recovery dispatches never receive them.

| Secret | Value |
|---|---|
| `APPLE_CERTIFICATE_P12_BASE64` | Base64 of a password-protected `.p12` export of the Developer ID Application certificate including its private key |
| `APPLE_CERTIFICATE_PASSWORD` | The `.p12` export password |
| `APPLE_ID` | Apple ID email used for notarization |
| `APPLE_APP_SPECIFIC_PASSWORD` | App-specific password for that Apple ID |
| `APPLE_TEAM_ID` | 10-character Apple Developer Team ID; the keychain must hold exactly one Developer ID Application identity for this team |

The signing identity is derived from the imported certificate, so no identity name secret is needed. A local `notarytool` keychain profile cannot be exported to CI; the Apple ID and app-specific password are supplied directly instead.

Apple cannot staple a notarization ticket to a bare executable or a ZIP, so release binaries are not stapled. On first launch of a quarantined binary, such as a Homebrew cask install, Gatekeeper looks the ticket up online. `install.sh` and npm downloads are not quarantined.

### npm setup (once per package)

The CLI is `@straddlecom/cli`; `@straddlecom/straddle` remains the TypeScript SDK. npm permissions belong to each package, so the SDK's trusted-publisher connection does not authorize CLI releases.

Before the first automated CLI release, an npm maintainer must publish `@straddlecom/cli@1.0.2` from `npm/`, aligned with Scalar contract 1.0.2. Do not bootstrap 0.1.1, any other version, or the placeholder `0.0.0`. Then configure the CLI package's Settings > Trusted Publisher for GitHub Actions:

- Organization: `straddle-build`
- Repository: `straddle-cli`
- Workflow filename: `release.yml`
- Environment: leave blank (the release job does not use a GitHub environment)

Allow `npm publish` in that connection. Subsequent CLI releases authenticate through GitHub OIDC without an `NPM_TOKEN`. See [npm trusted publishing](https://docs.npmjs.com/trusted-publishers/). Verify an actual automated publication before treating npm delivery as connected.

Each of the six platform packages needs the same connection, and npm accepts a trusted publisher only for a package that already exists. Before the first release that publishes them (1.0.4), an npm maintainer with publish rights on the `@straddlecom` scope bootstraps each one with an empty `0.0.0` placeholder, using npm 11.21.0 or newer for `npm trust`:

```bash
for p in darwin-arm64 darwin-x64 linux-arm64 linux-x64 win32-arm64 win32-x64; do
  name="@straddlecom/cli-$p" dir=$(mktemp -d)
  printf '{"name":"%s","version":"0.0.0","license":"Apache-2.0","repository":{"type":"git","url":"git+https://github.com/straddle-build/straddle-cli.git"}}\n' "$name" >"$dir/package.json"
  npm publish "$dir" --access public
  npm trust github "$name" --repo straddle-build/straddle-cli --file release.yml --allow-publish
done
```

The placeholder holds no binary. The wrapper pins its platform packages to its own exact version, so no `@straddlecom/cli` release installs `0.0.0`. Until every platform package is bootstrapped, a release publishes the GitHub release and fails at the first unconnected npm package; bootstrap it, then recover npm as described below.

### npm-only recovery for a published release

When a tag's GitHub release is published but its npm step did not run or stopped partway, dispatch `release.yml` from `main` with the tag, for example `gh workflow run release.yml --ref main -f tag=v1.0.4`. Only the `npm-recovery` job runs. It refuses a tag that is not `vX.Y.Z`, does not exist, is not on `main`, has no published non-draft, non-prerelease GitHub release, or whose assets are not exactly the six archives plus `checksums.txt`. It then checks out the tag's commit, downloads those assets, and runs that commit's `scripts/npm-platform-packages.sh` and the release job's publish step, so it builds the same seven packages and skips versions already on npm. Tags before v1.0.4 have no such script and cannot be recovered this way. It never runs GoReleaser, signs, rebuilds or changes release assets. Provenance records the dispatching `main` run of `release.yml`.

The Homebrew cask for such a release is added through a pull request to `straddle-build/homebrew-tap` using the published `checksums.txt`.

## Dependency maintenance

Dependabot (`.github/dependabot.yml`) runs weekly. Go module minor/patch updates are grouped as `go-minor-and-patch`, except `modernc.org/sqlite` — review SQLite updates separately because the local store depends on it. GitHub Actions updates are grouped together.

## Demo harness

`demo/` holds the VHS demo harness for marketing recordings (`demo.tape.tmpl`, `make-demo.sh`, `demo-charge.sh`). Demo scripts assume specific CLI output; re-check them when changing output formatting.
