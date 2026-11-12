// Builds the npm packages of a release from GoReleaser's output:
//
//   node npm/build.mjs <version> [dist]
//
// It writes npm/out/<package>/ for `rapibase` (the shim) and one
// @rapibase/cli-<os>-<arch> per binary, all with the same version. The
// release workflow then publishes the platform packages first and the shim
// last, so `npx rapibase@<version>` never resolves to missing binaries.
import { readFileSync, writeFileSync, mkdirSync, copyFileSync, rmSync, chmodSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
const root = join(here, '..')
const [, , rawVersion, distArg = 'dist'] = process.argv
if (!rawVersion) {
  console.error('usage: node npm/build.mjs <version> [dist]')
  process.exit(2)
}
const version = rawVersion.replace(/^v/, '')
if (!/^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$/.test(version)) {
  console.error(`invalid version ${rawVersion}`)
  process.exit(2)
}
const dist = join(root, distArg)
const out = join(here, 'out')
rmSync(out, { recursive: true, force: true })

const goToNode = {
  os: { linux: 'linux', darwin: 'darwin', windows: 'win32' },
  arch: { amd64: 'x64', arm64: 'arm64' },
}
const shim = JSON.parse(readFileSync(join(here, 'rapibase', 'package.json'), 'utf8'))

const artifacts = JSON.parse(readFileSync(join(dist, 'artifacts.json'), 'utf8'))
const binaries = artifacts.filter((a) => a.type === 'Binary' && a.name.startsWith('rapibase'))
const platforms = []
for (const a of binaries) {
  const os = goToNode.os[a.goos]
  const cpu = goToNode.arch[a.goarch]
  if (!os || !cpu) continue
  const name = `@rapibase/cli-${os}-${cpu}`
  const exe = os === 'win32' ? 'rapibase.exe' : 'rapibase'
  const dir = join(out, `cli-${os}-${cpu}`)
  mkdirSync(join(dir, 'bin'), { recursive: true })
  copyFileSync(join(root, a.path), join(dir, 'bin', exe))
  if (os !== 'win32') chmodSync(join(dir, 'bin', exe), 0o755)
  copyFileSync(join(root, 'LICENSE'), join(dir, 'LICENSE'))
  writeFileSync(
    join(dir, 'package.json'),
    JSON.stringify(
      {
        name,
        version,
        description: `The Rapibase CLI binary for ${os}-${cpu}. Install \`rapibase\` instead.`,
        homepage: shim.homepage,
        repository: shim.repository,
        license: shim.license,
        os: [os],
        cpu: [cpu],
        files: ['bin', 'LICENSE'],
        preferUnplugged: true,
      },
      null,
      2,
    ) + '\n',
  )
  platforms.push({ name, dir })
}

const expected = Object.keys(shim.optionalDependencies)
const built = platforms.map((p) => p.name)
const missing = expected.filter((n) => !built.includes(n))
if (missing.length) {
  console.error(`missing binaries for: ${missing.join(', ')}`)
  process.exit(1)
}

// the shim package, pinned to the platform packages of the same version
const shimDir = join(out, 'rapibase')
mkdirSync(join(shimDir, 'bin'), { recursive: true })
copyFileSync(join(here, 'rapibase', 'bin', 'rapibase.js'), join(shimDir, 'bin', 'rapibase.js'))
copyFileSync(join(root, 'README.md'), join(shimDir, 'README.md'))
copyFileSync(join(root, 'LICENSE'), join(shimDir, 'LICENSE'))
const pkg = { ...shim, version, optionalDependencies: Object.fromEntries(expected.map((n) => [n, version])) }
writeFileSync(join(shimDir, 'package.json'), JSON.stringify(pkg, null, 2) + '\n')

// publish order for the workflow: platforms first, shim last
writeFileSync(join(out, 'order.txt'), [...platforms.map((p) => p.dir), shimDir].join('\n') + '\n')
console.log(`npm packages ${version}: ${[...built, 'rapibase'].join(', ')}`)
