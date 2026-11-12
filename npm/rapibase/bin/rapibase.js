#!/usr/bin/env node
// Shim of the `rapibase` npm package: finds the binary of this platform in
// its optional dependency (@rapibase/cli-<os>-<arch>) and runs it with the
// same arguments, stdio and exit code. No postinstall (spec §10.2).
'use strict'

const { spawnSync } = require('node:child_process')
const fs = require('node:fs')

const platform = process.platform // linux | darwin | win32
const arch = process.arch // x64 | arm64
const exe = platform === 'win32' ? 'rapibase.exe' : 'rapibase'
const pkg = `@rapibase/cli-${platform}-${arch}`

function findBinary() {
  // RAPIBASE_BINARY: a local build, for development and for platforms
  // without a package
  if (process.env.RAPIBASE_BINARY) return process.env.RAPIBASE_BINARY
  try {
    return require.resolve(`${pkg}/bin/${exe}`)
  } catch {
    return null
  }
}

const bin = findBinary()
if (!bin) {
  const supported = ['linux-x64', 'linux-arm64', 'darwin-x64', 'darwin-arm64', 'win32-x64', 'win32-arm64']
  const msg = supported.includes(`${platform}-${arch}`)
    ? `the package ${pkg} is missing. It is an optional dependency: reinstall without --no-optional / --omit=optional (npm i -g rapibase@latest).`
    : `there is no Rapibase CLI build for ${platform}-${arch} yet. Supported: ${supported.join(', ')}.`
  console.error(`rapibase: ${msg}`)
  process.exit(1)
}

if (platform !== 'win32') {
  // some package managers drop the executable bit when unpacking
  try {
    const mode = fs.statSync(bin).mode
    if ((mode & 0o111) === 0) fs.chmodSync(bin, 0o755)
  } catch {
    // read-only store: try to run it anyway
  }
}

const res = spawnSync(bin, process.argv.slice(2), { stdio: 'inherit', windowsHide: false })
if (res.error) {
  console.error(`rapibase: could not run ${bin}: ${res.error.message}`)
  process.exit(1)
}
if (res.signal) {
  // die the same way the binary did (Ctrl+C → 130 in the shell)
  process.kill(process.pid, res.signal)
}
process.exit(res.status ?? 1)
