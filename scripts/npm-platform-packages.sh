#!/usr/bin/env bash
# Builds one release's seven npm packages into <out-dir>: a scriptless
# @straddlecom/cli-<platform>-<arch> package per GoReleaser archive (cli-*/),
# and the @straddlecom/cli wrapper (cli/) pinning all six as optional
# dependencies. Publish the cli-* packages before cli. Each binary comes from
# its checksum-verified archive, so signed darwin binaries stay signed.
set -euo pipefail

fail() { echo "npm-platform-packages: $*" >&2; exit 1; }

[ $# -eq 3 ] || fail "usage: $0 <archives-dir> <version> <out-dir>"
archives=$1 version=$2 out=$3
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || fail "version '$version' is not semver"
[ -f "$archives/checksums.txt" ] || fail "$archives/checksums.txt not found"
repo=$(cd "$(dirname "$0")/.." && pwd)

# GoReleaser goos_goarch:Node process.platform-process.arch
targets="darwin_arm64:darwin-arm64 darwin_amd64:darwin-x64 linux_arm64:linux-arm64 linux_amd64:linux-x64 windows_arm64:win32-arm64 windows_amd64:win32-x64"

mkdir -p "$out"
optional_deps=()
for target in $targets; do
  goplatform=${target%%:*} platform=${target#*:}
  name="@straddlecom/cli-$platform"
  if [ "${platform%%-*}" = win32 ]; then
    archive="straddle_${version}_${goplatform}.zip" exe=straddle.exe
  else
    archive="straddle_${version}_${goplatform}.tar.gz" exe=straddle
  fi
  [ -f "$archives/$archive" ] || fail "$archives/$archive not found"

  expected=$(awk -v f="$archive" '$2 == f { print $1 }' "$archives/checksums.txt")
  [ -n "$expected" ] || fail "no checksum for $archive in checksums.txt"
  actual=$(shasum -a 256 "$archives/$archive" | awk '{ print $1 }')
  [ "$actual" = "$expected" ] || fail "checksum mismatch for $archive: expected $expected, got $actual"

  pkg="$out/cli-$platform"
  rm -rf "$pkg"
  mkdir -p "$pkg/bin"
  if [ "$exe" = straddle.exe ]; then
    unzip -q "$archives/$archive" "$exe" -d "$pkg/bin"
  else
    tar -xzf "$archives/$archive" -C "$pkg/bin" "$exe"
  fi
  chmod 0755 "$pkg/bin/$exe"
  cp "$repo/LICENSE" "$pkg/LICENSE"
  cat >"$pkg/package.json" <<EOF
{
  "name": "$name",
  "version": "$version",
  "description": "The straddle binary for ${platform/-/ }, installed by @straddlecom/cli",
  "license": "Apache-2.0",
  "repository": {
    "type": "git",
    "url": "git+https://github.com/straddle-build/straddle-cli.git"
  },
  "os": ["${platform%%-*}"],
  "cpu": ["${platform#*-}"],
  "preferUnplugged": true
}
EOF
  cat >"$pkg/README.md" <<EOF
# $name

The \`straddle\` binary for ${platform/-/ }. Install [\`@straddlecom/cli\`](https://www.npmjs.com/package/@straddlecom/cli) instead; npm selects this package for your platform automatically.
EOF
  optional_deps+=("optionalDependencies.$name=$version")
done

rm -rf "$out/cli"
cp -R "$repo/npm" "$out/cli"
rm -rf "$out/cli/vendor" "$out/cli/node_modules"
(cd "$out/cli" && npm pkg set "version=$version" "${optional_deps[@]}")
echo "npm-platform-packages: wrote @straddlecom/cli@$version and its six platform packages to $out"
