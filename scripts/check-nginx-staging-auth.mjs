#!/usr/bin/env node
/**
 * check-nginx-staging-auth.mjs
 * 校验 deploy/nginx/kindergarten.conf.tmpl 的 staging 访问门禁契约（ticket #53 / #79）。
 *
 * 模板由 envsubst 渲染后人工部署；`nginx -t` 只能证明语法成立，证明不了
 * 「staging 有门禁、生产没有」。这里按 server 块做结构断言，在提交前而不是
 * 部署后暴露权限回归。零依赖，只用 Node 内建模块。
 *
 * #79 追加：staging 的豁免不是可选项——
 * - test-admin 的 /api/v1/、/health、/ready 必须 auth_basic off，否则管理端
 *   Bearer 请求会被 Basic Auth 覆盖；
 * - test-www 的 /privacy、/terms、/404.html 与静态资源必须 auth_basic off，
 *   否则家长端 web-view 打开政策页会 401；
 * - test-www 的 location / 必须同时具备门禁、clean URL rewrite 与目录页 try_files。
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
    return null
  }
  const { body } = found[0]
  if (!/auth_basic\s+"[^"]+";/.test(body)) {
    failures.push(`${name}: 缺少 auth_basic "realm"（staging 必须要求凭据）`)
  }
  if (!/auth_basic_user_file\s+\S+;/.test(body)) {
    failures.push(`${name}: 缺少 auth_basic_user_file`)
  }
  return found[0]
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

const AUTH_OFF = /\bauth_basic\s+off\s*;/
// 用一条样例资源路径验证 ~* 规则是否真的覆盖静态扩展名，
// 避免只检查「有没有 auth_basic off」却漏掉整段 location 被删。
const ASSET_SAMPLE = '/_nuxt/entry.css'

function parseLocations(serverBody) {
  const locations = []
  // 块开括号后必为空白/行尾；量词花括号（如 {2,3}）后面不是空白，因此不会被误认。
  const opener = /\blocation\s+([^{}\n]+?)\s*\{(?=\s|$)/g
  let match
  while ((match = opener.exec(serverBody)) !== null) {
    const openIndex = match.index + match[0].length - 1
    let depth = 0
    let index = openIndex
    for (; index < serverBody.length; index += 1) {
      if (serverBody[index] === '{') depth += 1
      else if (serverBody[index] === '}') {
        depth -= 1
        if (depth === 0) break
      }
    }
    if (depth !== 0) throw new Error('location 块花括号不配对')

    locations.push({
      spec: match[1].trim().replace(/\s+/g, ' '),
      body: serverBody.slice(openIndex + 1, index),
    })
    opener.lastIndex = index + 1
  }
  return locations
}

function safeLocations(server, label) {
  try {
    return parseLocations(server.body)
  } catch (err) {
    failures.push(`${label}: 解析 location 失败 — ${err.message}`)
    return null
  }
}

function requireLocation(locations, label, spec) {
  const matches = locations.filter((location) => location.spec === spec)
  if (matches.length !== 1) {
    failures.push(`${label}: 应有且仅有一个 "location ${spec}"（实际 ${matches.length} 个）`)
    return null
  }
  return matches[0]
}

function expectOffLocation(locations, label, spec) {
  const location = requireLocation(locations, label, spec)
  if (!location) return
  if (!AUTH_OFF.test(location.body)) {
    failures.push(
      `${label}: "location ${spec}" 必须 auth_basic off（豁免缺失会让 Bearer / 健康检查 / 政策页请求被 Basic Auth 拦下）`,
    )
  }
}

function coversStaticAsset(spec) {
  if (!spec.startsWith('~*')) return false
  try {
    return new RegExp(spec.slice(2).trim(), 'i').test(ASSET_SAMPLE)
  } catch {
    return false
  }
}

function expectStaticAssetsOff(locations, label) {
  const regexLocations = locations.filter((location) => location.spec.startsWith('~*'))
  if (!regexLocations.some((location) => coversStaticAsset(location.spec))) {
    failures.push(
      `${label}: 缺少覆盖静态扩展名（如 *.css）的 ~* location；政策页资源会被门禁挡成 401`,
    )
  }
  for (const location of regexLocations) {
    if (!AUTH_OFF.test(location.body)) {
      failures.push(`${label}: 静态资源 "location ${location.spec}" 必须 auth_basic off`)
    }
  }
}

function expectGatedRoot(locations, label) {
  const root = requireLocation(locations, label, '/')
  if (!root) return
  if (!/auth_basic\s+"[^"]+";/.test(root.body)) {
    failures.push(`${label}: "location /" 缺少 auth_basic "realm"`)
  }
  if (!/auth_basic_user_file\s+\S+;/.test(root.body)) {
    failures.push(`${label}: "location /" 缺少 auth_basic_user_file`)
  }
  if (!/rewrite\s+\^\/\(\.\+\)\/\$\s+\/\$1\s+permanent\s*;/.test(root.body)) {
    failures.push(`${label}: "location /" 缺少 clean URL rewrite（^/(.+)/$ → /$1 permanent）`)
  }
  if (!/try_files\s+\$uri\s+\$uri\/index\.html\s+\$uri\/\s+=404\s*;/.test(root.body)) {
    failures.push(`${label}: "location /" 缺少目录页 try_files（$uri $uri/index.html $uri/ =404）`)
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

const testAdmin = expectAuth(`test-admin.${DOMAIN}`)
const testWww = expectAuth(`test-www.${DOMAIN}`)

if (testAdmin) {
  const locations = safeLocations(testAdmin, 'test-admin')
  if (locations) {
    for (const spec of ['/api/v1/', '= /health', '= /ready']) {
      expectOffLocation(locations, 'test-admin', spec)
    }
  }
}

if (testWww) {
  const locations = safeLocations(testWww, 'test-www')
  if (locations) {
    for (const spec of ['= /privacy', '= /terms', '= /404.html']) {
      expectOffLocation(locations, 'test-www', spec)
    }
    expectStaticAssetsOff(locations, 'test-www')
    expectGatedRoot(locations, 'test-www')
  }
}

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
  '[check-nginx-staging-auth] staging: test-admin / test-www 有 auth_basic；test-admin API/健康检查与 test-www 政策页/404/静态资源豁免；prod admin / www / apex 与 test-api 无；80 端口仅 ACME + 301',
)
