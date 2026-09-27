#!/usr/bin/env bash
# Signs one darwin release binary with the Developer ID Application identity
# in the release keychain (hardened runtime, secure timestamp), then requires
# Apple's notary service to return Accepted. GoReleaser runs this as a build
# post hook, so any failure stops the release before archives, checksums or
# publication exist.
#
# Apple cannot staple a ticket to a bare Mach-O or a ZIP, so nothing is
# stapled: Gatekeeper finds the ticket online on first launch.
set -euo pipefail

bin=${1:?usage: macos-sign-notarize.sh <darwin-binary>}

fail() {
	echo "macos-sign-notarize: $*" >&2
	exit 1
}

for var in APPLE_SIGNING_KEYCHAIN APPLE_ID APPLE_APP_SPECIFIC_PASSWORD APPLE_TEAM_ID; do
	[ -n "${!var:-}" ] || fail "$var is not set; refusing to release an unsigned macOS binary"
done
[ -f "$bin" ] || fail "binary not found: $bin"

# Derive the identity from the isolated keychain instead of a configured name.
identities=$(security find-identity -v -p codesigning "$APPLE_SIGNING_KEYCHAIN" |
	grep -F "\"Developer ID Application: " | grep -F "($APPLE_TEAM_ID)\"" || true)
count=$(printf '%s' "$identities" | grep -c . || true)
[ "$count" = 1 ] || fail "expected exactly one Developer ID Application identity for team $APPLE_TEAM_ID, found $count"
identity=$(printf '%s\n' "$identities" | awk '{print $2}')

codesign --force --options runtime --timestamp --keychain "$APPLE_SIGNING_KEYCHAIN" --sign "$identity" "$bin"

codesign --verify --strict --verbose=2 "$bin"
details=$(codesign --display --verbose=4 "$bin" 2>&1)
grep -q '^Authority=Developer ID Application: ' <<<"$details" || fail "$bin is not signed by a Developer ID Application identity"
grep -q "^TeamIdentifier=$APPLE_TEAM_ID\$" <<<"$details" || fail "$bin is not signed by team $APPLE_TEAM_ID"
grep -q '^CodeDirectory .*flags=.*(runtime)' <<<"$details" || fail "$bin lacks the hardened runtime flag"
grep -q '^Timestamp=' <<<"$details" || fail "$bin lacks a secure timestamp"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
ditto -c -k --keepParent "$bin" "$work/submission.zip"

auth=(--apple-id "$APPLE_ID" --team-id "$APPLE_TEAM_ID" --password "$APPLE_APP_SPECIFIC_PASSWORD")
# notarytool can exit 0 for a completed but rejected submission, so the
# recorded status decides, not the exit code.
xcrun notarytool submit "$work/submission.zip" "${auth[@]}" --wait --timeout 30m --output-format json >"$work/result.json" ||
	fail "notarytool submit failed for $bin: $(cat "$work/result.json")"
status=$(sed -n 's/.*"status" *: *"\([^"]*\)".*/\1/p' "$work/result.json")
id=$(sed -n 's/.*"id" *: *"\([^"]*\)".*/\1/p' "$work/result.json")
if [ "$status" != Accepted ]; then
	[ -z "$id" ] || xcrun notarytool log "$id" "${auth[@]}" >&2 || true
	fail "notarization of $bin returned status '${status:-unknown}' (submission ${id:-unknown}), want Accepted"
fi
echo "macos-sign-notarize: $bin signed and notarized (submission $id)"
