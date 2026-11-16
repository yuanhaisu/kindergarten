#!/usr/bin/env node
/**
 * check-nginx-staging-auth.mjs
 * 校验 deploy/nginx/kindergarten.conf.tmpl 的 staging 访问门禁契约（ticket #53）。
 *
 * 模板由 envsubst 渲染后人工部署；`nginx -t` 只能证明语法成立，证明不了
 * 「staging 有门禁、生产没有」。这里按 server 块做结构断言，在提交前而不是
 * 部署后暴露权限回归。零依赖，只用 Node 内建模块。
 *
 * 用法：
 *   node scripts/check-nginx-staging-auth.mjs [模板路径]
 *
 * 退出码：0 = 通过；1 = 断言失败或模板不可读。
 */
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const DEFAULT_TEMPLATE = resolve(repoRoot, 'deploy/nginx/kindergarten.conf.tmpl')
const target = resolve(process.argv[2] || DEFAULT_TEMPLATE)

// envsubst 占位符先换成无花括号的标记，避免和 server 块的花括号混淆。
const DOMAIN = '__VAR_DOMAIN__'

function normalize(text) {
  return text
    .replace(/\$\{([A-Z0-9_]+)\}/g, (_, name) => `__VAR_${name}__`)
    .replace(/#[^\n]*/g, '')
}

function parseServerBlocks(text) {
  const blocks = []
  const opener = /\bserver\s*\{/g
  let match
  while ((match = opener.exec(text)) !== null) {
    const openIndex = match.index + match[0].length - 1
    let depth = 0
    let index = openIndex
    for (; index < text.length; index += 1) {
      if (text[index] === '{') depth += 1
      else if (text[index] === '}') {
        depth -= 1
        if (depth === 0) break
      }
    }
    if (depth !== 0) throw new Error('server 块花括号不配对')

    const body = text.slice(openIndex + 1, index)
    const listen = (body.match(/(?:^|\n)\s*listen\s+([^;]+);/) || [])[1] || ''
    const names = ((body.match(/(?:^|\n)\s*server_name\s+([^;]+);/) || [])[1] || '')
      .trim()
      .split(/\s+/)
      .filter(Boolean)
    blocks.push({ body, listen: listen.trim(), names })
    opener.lastIndex = index + 1
  }
  return blocks
}

const failures = []

function httpsBlock(name) {
  return blocks.filter((block) => block.listen.startsWith('443') && block.names.includes(name))
}

function expectAuth(name) {
  const found = httpsBlock(name)
  if (found.length !== 1) {
    failures.push(`${name}: 应有且仅有一个 443 server 块（实际 ${found.length} 个）`)
    return
  }
  const { body } = found[0]
  if (!/auth_basic\s+"[^"]+";/.test(body)) {
    failures.push(`${name}: 缺少 auth_basic "realm"（staging 必须要求凭据）`)
  }
  if (!/auth_basic_user_file\s+\S+;/.test(body)) {
    failures.push(`${name}: 缺少 auth_basic_user_file`)
  }
}

function expectNoAuth(name, label) {
  const found = httpsBlock(name)
  if (found.length !== 1) {
    failures.push(`${label}: 应有且仅有一个 443 server 块（实际 ${found.length} 个）`)
    return
  }
  if (/auth_basic/.test(found[0].body)) {
    failures.push(`${label}: 生产站点不应启用 auth_basic（#53 只作用于 staging）`)
  }
}

let template
try {
  template = readFileSync(target, 'utf8')
} catch (err) {
  console.error(`[check-nginx-staging-auth] FAIL 读取模板失败：${target} — ${err.message}`)
  process.exit(1)
}

let blocks
try {
  blocks = parseServerBlocks(normalize(template))
} catch (err) {
  console.error(`[check-nginx-staging-auth] FAIL 解析模板失败：${target} — ${err.message}`)
  process.exit(1)
}

if (blocks.length === 0) {
  console.error(`[check-nginx-staging-auth] FAIL 未找到任何 server 块：${target}`)
  process.exit(1)
}

expectAuth(`test-admin.${DOMAIN}`)
expectAuth(`test-www.${DOMAIN}`)

expectNoAuth(`admin.${DOMAIN}`, 'prod admin')
expectNoAuth(`www.${DOMAIN}`, 'prod www')
expectNoAuth(DOMAIN, 'prod apex')
expectNoAuth(`test-api.${DOMAIN}`, 'test-api')

const port80 = blocks.filter((block) => block.listen.split(/\s+/)[0] === '80')
if (port80.length !== 1) {
  failures.push(`80 端口块: 应有且仅有一个（实际 ${port80.length} 个）`)
} else {
  const { body } = port80[0]
  if (!body.includes('location ^~ /.well-known/acme-challenge/')) {
    failures.push('80 端口块: 缺少 ACME HTTP-01 location')
  }
  if (!body.includes('root /var/www/acme;')) {
    failures.push('80 端口块: ACME location 应指向 /var/www/acme')
  }
  if (!body.includes('return 301 https://$host$request_uri;')) {
    failures.push('80 端口块: 其余请求应 301 到 HTTPS')
  }
  if (/auth_basic/.test(body)) {
    failures.push('80 端口块: 不应出现 auth_basic（ACME 校验必须匿名）')
  }
}

if (failures.length > 0) {
  console.error(`[check-nginx-staging-auth] FAIL ${target}`)
  for (const failure of failures) {
    console.error(`[check-nginx-staging-auth] ${failure}`)
  }
  process.exit(1)
}

console.log(`[check-nginx-staging-auth] OK ${target}`)
console.log(
  '[check-nginx-staging-auth] staging: test-admin / test-www 有 auth_basic；prod admin / www / apex 与 test-api 无；80 端口仅 ACME + 301',
)
