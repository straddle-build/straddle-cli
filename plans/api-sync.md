# Straddle CLI API Sync

ME-663 keeps the published CLI synchronized with immutable releases of `@straddle/straddle-api` in Scalar.

## What happens

1. The API contract publisher sends `straddle-contract-published` with an exact contract version and published SHA-256 digest.
2. The CLI workflow downloads that exact Scalar version and verifies its declared version and digest before using it.
3. A daily recovery run discovers Scalar's current version, then re-fetches the exact versioned artifact. A manual run accepts an exact version.
4. If the version and bytes already match either `main` or that version's pending synchronization branch, the workflow exits successfully without changing the repository. Changed bytes for an already-seen version fail.
5. For every new version, the workflow:
   - records additions, changes, removals, and unsupported operations for review;
   - generates supported new endpoint commands;
   - updates `spec.yaml` and `contract.lock.json`;
   - runs lock verification, endpoint coverage, tests, and vetting;
   - opens a normal pull request for human review.
6. Merging the generated version-specific `automation/api-sync-*` pull request creates the next CLI patch tag.
7. The existing release workflow publishes the updated CLI through GitHub Releases and npm, plus Homebrew when its repository token is configured. The shell installer and `go install` resolve the GitHub release automatically.

Changed, removed, or unsupported operations never suppress the synchronization pull request. They stay visible in its summary and workflow artifacts so a reviewer can decide what the CLI should do.

## Checked-in contract

- `spec.yaml` contains the exact Scalar bytes.
- `contract.lock.json` records the contract version, immutable Registry reference, and SHA-256 digest.
- CLI endpoint annotations record stable `operationId` values for coverage checks.
- Operations the generic generator cannot represent are reported explicitly. They do not require a separate policy file.

## Boundaries

- The CLI binary never fetches Scalar at runtime.
- Synchronization pull requests are never auto-merged.
- The CLI does not claim or publish SDK support.
- The CLI does not require read access to the private contract repository.
- The workflow does not accept arbitrary contract URLs.
- Drift reporting is review evidence, not a claim of complete OpenAPI semantic equivalence.

## Verification

```bash
go run ./cmd/gen-endpoint verify-lock --spec spec.yaml
go run ./cmd/gen-endpoint check --spec spec.yaml --repo .
go test ./...
go vet ./...
```
