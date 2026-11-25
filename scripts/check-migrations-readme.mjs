#!/usr/bin/env node
/**
 * 迁移 README 漂移检查（ticket #184）
 *
 * `server/migrations/README.md` 的「版本清单」表格必须与 `*.sql` 文件一一对应。
 * 新增/删除迁移却忘记改 README 时，在 CI 阶段失败并打印缺失/多出的版本号，
 * 避免文档再次停在旧版本（历史问题：README 停在 00007，实际已到 00015）。
 *
 * 约定：版本行的第一个单元格是 5 位版本号，第二个单元格是一句话用途。
 * 目录中的非 .sql 文件（embed.go）不参与比对。
 *
 * 用法：
 *   node scripts/check-migrations-readme.mjs [迁移目录]
 *
 * 默认目录为 `server/migrations`。退出码：0 = 一致；1 = 漂移或不可读。
 * 只读文件系统，不连接数据库。
 */
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const targetDir = resolve(process.argv[2] || join(repoRoot, 'server/migrations'))

const SQL_FILE_RE = /^(\d{5})_.+\.sql$/
const README_ROW_RE = /^\|\s*(\d{5})\s*\|\s*(.*?)\s*\|\s*$/
const LABEL = '[check-migrations-readme]'

function readMigrationFiles(dir) {
  const versions = new Set()
  for (const name of readdirSync(dir)) {
    const match = SQL_FILE_RE.exec(name)
    if (match) versions.add(match[1])
  }
  return versions
}

function readReadmeManifest(readmePath) {
  const text = readFileSync(readmePath, 'utf8')
  const rows = []
  for (const line of text.split(/\r?\n/)) {
    const match = README_ROW_RE.exec(line)
    if (match) rows.push({ version: match[1], purpose: match[2] })
  }
  return rows
}

function sorted(versions) {
  return [...versions].sort()
}

function main() {
  let files
  let rows
  try {
    files = readMigrationFiles(targetDir)
    rows = readReadmeManifest(join(targetDir, 'README.md'))
  } catch (error) {
    console.error(`${LABEL} ERROR 无法读取 ${targetDir}: ${error.message}`)
    process.exit(1)
  }

  const failures = []
  const seen = new Set()
  const listed = new Set()

  for (const { version, purpose } of rows) {
    if (seen.has(version)) {
      failures.push(`README 版本重复: ${version}`)
      continue
    }
    seen.add(version)
    listed.add(version)
    if (purpose.trim() === '') {
      failures.push(`README 版本缺少用途说明: ${version}`)
    }
  }

  const missing = sorted([...files].filter((version) => !listed.has(version)))
  const extra = sorted([...listed].filter((version) => !files.has(version)))

  if (missing.length > 0) {
    failures.push(`README 缺少迁移版本: ${missing.join(', ')}`)
  }
  if (extra.length > 0) {
    failures.push(`README 多出迁移版本（无对应 .sql 文件）: ${extra.join(', ')}`)
  }

  if (failures.length > 0) {
    console.error(`${LABEL} FAIL ${targetDir}`)
    for (const failure of failures) {
      console.error(`${LABEL} ${failure}`)
    }
    process.exit(1)
  }

  const versions = sorted(files)
  const range = versions.length > 0 ? `${versions[0]}–${versions[versions.length - 1]}` : '空'
  console.log(`${LABEL} OK ${targetDir}（${versions.length} 个版本，${range}）`)
}

main()
