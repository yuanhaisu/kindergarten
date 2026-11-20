#!/usr/bin/env node
/**
 * Project-local secret scanner.
 *
 * Default mode scans the staged diff plus untracked files. `--files` scans
 * explicit paths (used by the supervisor against the worker manifest).
 * Finds are reported as file:line + rule name only; the matched value is never
 * printed.
 */
import { execFileSync } from 'node:child_process'
import { existsSync, readFileSync, statSync } from 'node:fs'
import { resolve } from 'node:path'

const ROOT = process.cwd()
const MAX_FILE_BYTES = 1_000_000

const SKIP = [
  /(^|\/)node_modules\//,
  /(^|\/)\.git\//,
  /(^|\/)dist\//,
  /(^|\/)build\//,
  /(^|\/)\.claude\/runs\//,
  /(^|\/)\.claude\/locks\//,
  /(^|\/)package-lock\.json$/,
  /(^|\/)pnpm-lock\.yaml$/,
  /(^|\/)yarn\.lock$/,
]

const FILE_RULES = [
  { rule: 'sensitive-file', test: (p) => /(^|\/)\.env$/.test(p) },
  { rule: 'sensitive-file', test: (p) => /(^|\/)\.env\.(local|development|production|test|staging|prod|dev|uat)$/.test(p) },
  { rule: 'sensitive-file', test: (p) => /(^|\/)configs\/app\.local\.ini$/.test(p) },
  { rule: 'sensitive-file', test: (p) => /(^|\/)\.mcp\.json$/.test(p) },
  { rule: 'sensitive-file', test: (p) => /\.(pem|key|p12|jks|pfx)$/i.test(p) },
  { rule: 'sensitive-file', test: (p) => /(^|\/)id_(rsa|dsa|ecdsa|ed25519)(\.|$)/.test(p) },
]

const CONTENT_RULES = [
  ['private-key', /-----BEGIN (?:RSA |EC |OPENSSH |DSA |PGP )?PRIVATE KEY-----/],
  ['anthropic-key', /\bsk-ant-[A-Za-z0-9_-]{20,}\b/],
  ['openai-key', /\bsk-[A-Za-z0-9_-]{20,}\b/],
  ['aws-access-key', /\bAKIA[0-9A-Z]{16}\b/],
  ['alibaba-access-key', /\bLTAI[0-9A-Za-z]{12,}\b/],
  ['github-token', /\bgh[pousr]_[0-9A-Za-z]{36,}\b/],
  ['slack-token', /\bxox[baprs]-[0-9A-Za-z-]{10,}\b/],
  ['mysql-dsn-password', /\bmysql:\/\/[^:\s/]+:[^@\s/]+@/i],
  // Go/MySQL 驱动常见 DSN：账号:口令@tcp(主机:端口)/库名
  ['go-mysql-dsn', /\b[A-Za-z0-9_]+:[^@\s]{6,}@tcp\([^)\s]+\)/i],
  ['jwt-secret', /\bJWT_SECRET\s*[:=]\s*['"][^'"]{8,}['"]/i],
  ['wechat-secret', /\bWECHAT_[A-Z0-9_]*(?:SECRET|KEY)\s*[:=]\s*['"][^'"]{8,}['"]/i],
  // app.ini 风格：parent_secret = <32 位十六进制>
  ['wechat-app-secret', /\b[a-z_]*secret\s*[:=]\s*['"]?[0-9a-f]{32}['"]?\s*$/i],
  ['oss-secret', /\bOSS_[A-Z0-9_]*(?:SECRET|KEY)\s*[:=]\s*['"][^'"]{8,}['"]/i],
  ['password-assignment', /\b(?:password|passwd|pwd|secret|token|api[_-]?key)\s*[:=]\s*['"][^'"]{6,}['"]/i],
  // ini 风格不加引号：password = <明文口令> / access_key_id = <明文 AK>
  [
    'ini-credential-assignment',
    /^\s*(?:password|passwd|pwd|secret|secret_key|secret_id|access_key_id|access_key_secret|token|api[_-]?key)\s*=\s*[^\s$"'<>*{}()[\].;,]{8,}\s*$/i,
  ],
]

const PLACEHOLDER =
  /(?:\b(?:x{3,}|\*{3,}|change[-_ ]?me|replace[-_ ]?me|your[-_ ][a-z0-9_-]*(?:key|secret|token|password)|example|placeholder|redacted|dummy|sample)\b|\$\{[^}]*\}|<[^>]{1,}>)/i

// --all 模式下已知且经人工确认的假阳性。只按「规则 + 文件」精确放行，
// 不做整目录豁免，避免新增泄漏被同一目录的规则一起吞掉。
const ALLOW = [
  // vendored 的 agent/skill 文档与示例代码，口令是教学用假值
  { rule: 'password-assignment', file: /^\.claude\/(?:agents|skills)\// },
  // 两端 .env.* 是刻意入库的 Vite 公开变量（只有 API 域名），不含密钥
  { rule: 'sensitive-file', file: /^(?:parent|teacher)\/\.env\.(?:development|production|test)$/ },
  // .mcp.json 只含本机工具路径，无密钥
  { rule: 'sensitive-file', file: /^\.mcp\.json$/ },
  // 改密表单里的字段名，值来自变量而非字面量
  { rule: 'password-assignment', file: /^admin\/pages\/admins\/index\.vue$/ },
  // 测试夹具
  { rule: 'password-assignment', file: /^server\/internal\/config\/config_test\.go$/ },
  // 平台会话单测里的假 token（new-access / new-refresh），不是凭据
  { rule: 'password-assignment', file: /^admin\/tests\/unit\/usePlatformHttp\.test\.ts$/ },
  // 平台账号服务测试的假口令（Abc12345），不是凭据
  { rule: 'password-assignment', file: /^server\/internal\/service\/platform_account_test\.go$/ },
]

function isAllowed(finding) {
  return ALLOW.some(({ rule, file }) => rule === finding.rule && file.test(finding.file))
}

function git(args) {
  return execFileSync('git', args, { cwd: ROOT, encoding: 'utf8', maxBuffer: 32 * 1024 * 1024 })
}

function normalize(file) {
  return file.replaceAll('\\', '/').replace(/^\.\//, '')
}

function isSkipped(file) {
  const p = normalize(file)
  return SKIP.some((re) => re.test(p))
}

function isBinary(file) {
  try {
    const buf = readFileSync(resolve(ROOT, file))
    return buf.subarray(0, 8192).includes(0)
  } catch {
    return true
  }
}

function tooLarge(file) {
  try {
    return statSync(resolve(ROOT, file)).size > MAX_FILE_BYTES
  } catch {
    return true
  }
}

function fileFindings(file) {
  const p = normalize(file)
  const findings = []
  for (const { rule, test } of FILE_RULES) {
    if (test(p)) {
      findings.push({ file: p, line: 1, rule })
      break
    }
  }
  return findings
}

function lineFindings(file, line, lineNo) {
  if (PLACEHOLDER.test(line)) return []
  for (const [rule, re] of CONTENT_RULES) {
    if (re.test(line)) return [{ file: normalize(file), line: lineNo, rule }]
  }
  return []
}

function scanFile(file) {
  const p = normalize(file)
  if (isSkipped(p) || isBinary(p) || tooLarge(p)) return []
  const findings = fileFindings(p)
  const text = readFileSync(resolve(ROOT, p), 'utf8')
  const lines = text.split(/\r?\n/)
  for (let i = 0; i < lines.length; i++) {
    findings.push(...lineFindings(p, lines[i], i + 1))
  }
  return findings
}

function stagedFiles() {
  const out = git(['diff', '--cached', '--name-only', '--diff-filter=ACMR'])
  return out.split(/\r?\n/).filter(Boolean)
}

function untrackedFiles() {
  const out = git(['ls-files', '--others', '--exclude-standard'])
  return out.split(/\r?\n/).filter(Boolean)
}

function allTrackedFiles() {
  const out = git(['ls-files'])
  return out.split(/\r?\n/).filter(Boolean)
}

function stagedAddedLines(files) {
  if (files.length === 0) return []
  const diff = git(['diff', '--cached', '--no-color', '--unified=0', '--', ...files])
  const rows = []
  let current = null
  let newLine = 0
  for (const raw of diff.split(/\r?\n/)) {
    if (raw.startsWith('+++ b/')) {
      current = raw.slice(6)
      continue
    }
    if (raw.startsWith('+++ /dev/null')) {
      current = null
      continue
    }
    const hunk = /^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@/.exec(raw)
    if (hunk) {
      newLine = Number(hunk[1])
      continue
    }
    if (!current) continue
    if (raw.startsWith('+') && !raw.startsWith('+++')) {
      rows.push({ file: current, line: newLine, text: raw.slice(1) })
      newLine += 1
    } else if (raw.startsWith(' ')) {
      newLine += 1
    }
  }
  return rows
}

function parseArgs(argv) {
  const opts = { staged: false, all: false, json: false, quiet: false, files: [], help: false }
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]
    if (arg === '--staged') opts.staged = true
    else if (arg === '--all') opts.all = true
    else if (arg === '--json') opts.json = true
    else if (arg === '--quiet') opts.quiet = true
    else if (arg === '--help' || arg === '-h') opts.help = true
    else if (arg === '--files') {
      while (i + 1 < argv.length && !argv[i + 1].startsWith('--')) {
        opts.files.push(argv[++i])
      }
    } else if (arg.startsWith('--files=')) {
      opts.files.push(arg.slice('--files='.length))
    } else {
      throw new Error(`Unknown argument: ${arg}`)
    }
  }
  if (!opts.staged && !opts.all && opts.files.length === 0) opts.staged = true
  return opts
}

function printHelp() {
  console.log(`Usage:
  node scripts/check-secrets.mjs [--staged] [--all] [--files <path...>] [--json] [--quiet]

Modes:
  --staged        Scan staged added lines + untracked files (default)
  --all           Scan all tracked text files
  --files <...>   Scan the given files entirely (used by supervisor)

Exit codes:
  0  no findings
  1  findings
  2  usage/tool error`)
}

function main() {
  const opts = parseArgs(process.argv.slice(2))
  if (opts.help) {
    printHelp()
    return
  }

  const findings = []

  if (opts.staged) {
    const files = stagedFiles()
    for (const file of files) findings.push(...fileFindings(file))
    for (const row of stagedAddedLines(files)) {
      findings.push(...lineFindings(row.file, row.text, row.line))
    }
    for (const file of untrackedFiles()) {
      findings.push(...scanFile(file))
    }
  }

  if (opts.all) {
    for (const file of allTrackedFiles()) findings.push(...scanFile(file))
  }

  if (opts.files.length > 0) {
    for (const file of opts.files) {
      if (!existsSync(resolve(ROOT, file))) continue
      findings.push(...scanFile(file))
    }
  }

  const unique = []
  const seen = new Set()
  for (const finding of findings) {
    if (isAllowed(finding)) continue
    const key = `${finding.file}:${finding.line}:${finding.rule}`
    if (seen.has(key)) continue
    seen.add(key)
    unique.push(finding)
  }

  if (opts.json) {
    console.log(JSON.stringify({ findings: unique, count: unique.length }, null, 2))
  } else if (!opts.quiet) {
    if (unique.length === 0) {
      console.log('Secret scan: 0 findings')
    } else {
      console.log(`Secret scan: ${unique.length} finding(s)`)
      for (const f of unique) console.log(`  ${f.file}:${f.line}  [${f.rule}]`)
    }
  }

  if (unique.length > 0) process.exitCode = 1
}

try {
  main()
} catch (error) {
  console.error(`check-secrets: ${error.message}`)
  process.exitCode = 2
}
