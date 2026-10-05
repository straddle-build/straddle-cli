#!/usr/bin/env node
// Launcher for @straddlecom/cli: executes the binary from this platform's
// @straddlecom/cli-<platform>-<arch> package, downloading it into vendor/
// first when that optional dependency was not installed.
'use strict';

const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { binaryPath } = require('../resolve.js');

let binPath = binaryPath();

if (binPath === null) {
  console.error(
    `@straddlecom/cli: @straddlecom/cli-${process.platform}-${process.arch} is not installed. ` +
      'npm skips this optional dependency under --omit=optional, with a lockfile written on ' +
      'another OS, or on an unsupported platform. Downloading the binary from GitHub Releases instead.'
  );
  const installer = path.join(__dirname, '..', 'install.js');
  const result = spawnSync(process.execPath, [installer], { stdio: 'inherit' });
  binPath = binaryPath();
  if (result.status !== 0 || binPath === null) {
    console.error(
      '@straddlecom/cli: binary install failed; see errors above. ' +
        'Reinstall with optional dependencies (npm install @straddlecom/cli --include=optional).'
    );
    process.exit(result.status === null || result.status === 0 ? 1 : result.status);
  }
}

const child = spawnSync(binPath, process.argv.slice(2), { stdio: 'inherit' });
process.exit(child.status === null ? 1 : child.status);
