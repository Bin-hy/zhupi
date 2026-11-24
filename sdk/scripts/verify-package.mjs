#!/usr/bin/env node
// 发布前校验：包内容、体积、安装、类型解析与运行时依赖。
// 用法：在 sdk/ 下执行 node scripts/verify-package.mjs

import { execFileSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync, unlinkSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const sdkRoot = dirname(dirname(fileURLToPath(import.meta.url)))
const LIMIT_TARBALL = 300 * 1024
const LIMIT_UNPACKED = 1.5 * 1024 * 1024
const REQUIRED = [
  'dist/zhupi.esm.js',
  'dist/zhupi.js',
  'dist/index.d.ts',
  'README.md',
  'LICENSE',
  'package.json',
]

function run(cmd, args, options = {}) {
  return execFileSync(cmd, args, {
    cwd: options.cwd ?? sdkRoot,
    encoding: 'utf8',
    stdio: options.stdio ?? ['ignore', 'pipe', 'pipe'],
  })
}

function fail(message) {
  console.error(`\n✗ ${message}`)
  process.exitCode = 1
}

function assert(condition, message) {
  if (!condition) throw new Error(message)
}

let packDir = ''
let tarball = ''

try {
  console.log(`SDK 目录: ${sdkRoot}`)

  const packed = JSON.parse(run('npm', ['pack', '--json']))
  const info = Array.isArray(packed) ? packed[0] : packed
  assert(info, 'npm pack 未返回包信息')
  tarball = join(sdkRoot, info.filename)
  const files = info.files.map((f) => f.path)

  console.log(`包文件数: ${files.length}`)
  console.log(`tarball: ${(info.size / 1024).toFixed(1)} KB`)
  console.log(`解包: ${(info.unpackedSize / 1024 / 1024).toFixed(2)} MB`)

  for (const required of REQUIRED) {
    assert(files.includes(required), `缺少必需文件: ${required}`)
  }
  for (const file of files) {
    assert(!file.endsWith('.map'), `包内不应包含 sourcemap: ${file}`)
    assert(!file.endsWith('.test.d.ts'), `包内不应包含测试声明: ${file}`)
  }
  assert(info.size <= LIMIT_TARBALL, `tarball 超过 300KB: ${info.size} bytes`)
  assert(info.unpackedSize <= LIMIT_UNPACKED, `解包体积超过 1.5MB: ${info.unpackedSize} bytes`)
  console.log('✓ 包内容与体积')

  packDir = mkdtempSync(join(tmpdir(), 'zhupi-consumer-'))
  writeFileSync(
    join(packDir, 'package.json'),
    JSON.stringify({ name: 'zhupi-consumer-check', private: true, type: 'module' }, null, 2),
  )
  writeFileSync(
    join(packDir, 'tsconfig.json'),
    JSON.stringify(
      {
        compilerOptions: {
          target: 'ES2020',
          module: 'ESNext',
          moduleResolution: 'Bundler',
          lib: ['ES2020', 'DOM', 'DOM.Iterable'],
          strict: true,
          noEmit: true,
          skipLibCheck: true,
        },
        include: ['index.ts'],
      },
      null,
      2,
    ),
  )
  writeFileSync(
    join(packDir, 'index.ts'),
    [
      "import { init } from 'zhupi-sdk'",
      "import type {",
      '  Annotation,',
      '  AnnotationComment,',
      '  Anchor,',
      '  AuthProvider,',
      '  CreateAnnotationInput,',
      '  EventMap,',
      '  Identity,',
      '  QuoteSelector,',
      '  SessionView,',
      '  SiteConfig,',
      '  ThemeMode,',
      '  User,',
      "} from 'zhupi-sdk'",
      '',
      "const theme: ThemeMode = 'auto'",
      "const instance = init({ siteKey: 'site-key', theme })",
      '',
      'type Collected = [',
      '  Annotation,',
      '  AnnotationComment,',
      '  Anchor,',
      '  AuthProvider,',
      '  CreateAnnotationInput,',
      '  EventMap,',
      '  Identity,',
      '  QuoteSelector,',
      '  SessionView,',
      '  SiteConfig,',
      '  User,',
      ']',
      '',
      'void instance',
      'export type { Collected }',
      '',
    ].join('\n'),
  )

  run('npm', ['install', tarball, '--no-audit', '--no-fund', '--ignore-scripts'], { cwd: packDir })
  console.log('✓ 临时消费者安装')

  const tsc = join(sdkRoot, 'node_modules', '.bin', 'tsc')
  run(tsc, ['--noEmit', '-p', 'tsconfig.json'], { cwd: packDir })
  console.log('✓ ESM 导入与公开类型编译通过')

  const tree = JSON.parse(
    run('npm', ['ls', '--omit=dev', '--depth=0', '--json'], { cwd: packDir }),
  )
  const runtimeDeps = Object.keys(tree.dependencies ?? {})
  assert(
    runtimeDeps.length === 1 && runtimeDeps[0] === 'zhupi-sdk',
    `运行时依赖应只有 zhupi-sdk，实际: ${runtimeDeps.join(', ') || '(空)'}`,
  )
  console.log('✓ 运行时依赖为空')

  console.log('\n全部检查通过。')
} catch (error) {
  fail(error instanceof Error ? error.message : String(error))
} finally {
  if (tarball) {
    try {
      unlinkSync(tarball)
    } catch {
      /* ignore */
    }
  }
  if (packDir) {
    rmSync(packDir, { recursive: true, force: true })
  }
}
