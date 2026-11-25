import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const SCRIPT = path.join(repoRoot, 'scripts/check-migrations-readme.mjs')

function run(dir) {
  return spawnSync(process.execPath, [SCRIPT, dir], { encoding: 'utf8' })
}

/**
 * 迁移目录 fixture：sqlFiles 决定实际迁移文件，rows 决定 README 表格行。
 * rows 省略时按 sqlFiles 生成一一对应的行；显式传入可用于制造漏列/多列。
 */
function writeFixture(t, { sqlFiles = [], rows, writeReadme = true }) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'migrations-readme-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))

  for (const file of sqlFiles) {
    fs.writeFileSync(path.join(dir, file), '-- fixture\n-- +goose Up\n')
  }
  // 迁移目录里混着非 .sql 文件（embed.go），检查必须忽略。
  fs.writeFileSync(path.join(dir, 'embed.go'), 'package migrations\n')

  if (writeReadme) {
    const tableRows =
      rows ??
      sqlFiles.map((file) => [file.slice(0, 5), `用途 ${file.slice(0, 5)}`])
    const table = [
      '# fixture',
      '',
      '## 版本清单',
      '',
      '| 版本 | 用途 |',
      '| --- | --- |',
      ...tableRows.map(([version, purpose]) => `| ${version} | ${purpose} |`),
      '',
    ].join('\n')
    fs.writeFileSync(path.join(dir, 'README.md'), table)
  }

  return dir
}

test('README 版本清单与迁移文件一致时通过', (t) => {
  const dir = writeFixture(t, {
    sqlFiles: ['00001_baseline.sql', '00002_second.sql', '00003_third.sql'],
  })

  const result = run(dir)

  assert.equal(result.status, 0, result.stderr)
  assert.match(result.stdout, /OK/)
})

test('README 漏列版本时失败并打印缺失版本号', (t) => {
  const dir = writeFixture(t, {
    sqlFiles: ['00001_baseline.sql', '00002_second.sql'],
    rows: [['00001', '基线']],
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /缺少/)
  assert.match(result.stderr, /00002/)
})

test('README 多列版本时失败并打印多出版本号', (t) => {
  const dir = writeFixture(t, {
    sqlFiles: ['00001_baseline.sql'],
    rows: [
      ['00001', '基线'],
      ['00099', '不存在的迁移'],
    ],
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /多出/)
  assert.match(result.stderr, /00099/)
})

test('README 重复列出版本时失败', (t) => {
  const dir = writeFixture(t, {
    sqlFiles: ['00001_baseline.sql'],
    rows: [
      ['00001', '基线'],
      ['00001', '重复行'],
    ],
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /重复/)
  assert.match(result.stderr, /00001/)
})

test('版本行缺少用途说明时失败', (t) => {
  const dir = writeFixture(t, {
    sqlFiles: ['00001_baseline.sql'],
    rows: [['00001', '']],
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /用途/)
  assert.match(result.stderr, /00001/)
})

test('缺少 README 时失败且不打印堆栈', (t) => {
  const dir = writeFixture(t, {
    sqlFiles: ['00001_baseline.sql'],
    writeReadme: false,
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /README\.md/)
  assert.doesNotMatch(result.stderr, /\n\s+at /)
})
