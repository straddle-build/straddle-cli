// Locates the straddle binary for this platform. Releases ship it in the
// @straddlecom/cli-<platform>-<arch> optional dependency; vendor/ holds a copy
// that bin/straddle.js downloaded when that package was not installed.
'use strict';

const fs = require('node:fs');
const path = require('node:path');

function binaryPath() {
  const exe = process.platform === 'win32' ? 'straddle.exe' : 'straddle';
  try {
    return require.resolve(`@straddlecom/cli-${process.platform}-${process.arch}/bin/${exe}`);
  } catch {
    // Missing or unsupported platform package; try the downloaded copy.
  }
  const vendored = path.join(__dirname, 'vendor', exe);
  return fs.existsSync(vendored) ? vendored : null;
}

module.exports = { binaryPath };
