#!/usr/bin/env node
/**
 * check-admin-static.mjs
 * 管理端静态产物的发布前检查。零依赖，只用 Node 内建模块。
 *
 * 为什么在构建后跑：`nuxt generate` 退出 0 只说明打包没报错，不代表产物能直接
 * 发布 —— 远程字体会拖慢国内首屏，残留的 localhost 地址会让线上页面请求本机，
 * 缺入口或 `_nuxt` 资源则直接白屏。这些都要在打包机上拦下，而不是等 staging 抽查。
 *
 * 用法：
 *   node scripts/check-admin-static.mjs [产物目录]   # 默认 .output/public
 *
 * 退出码：0 = 通过，1 = 命中规则（CI-friendly）。失败信息会列出规则与命中文件。
 */
import { existsSync, readFileSync, readdirSync, statSync } from 'node:fs'
import { extname, join, relative, resolve } from 'node:path'

const TEXT_EXTENSIONS = new Set([
  '.html', '.htm', '.js', '.mjs', '.css', '.json', '.txt', '.xml', '.svg', '.webmanifest', '.map',
])
const FONT_HOSTS = ['fonts.googleapis.com', 'fonts.gstatic.com']
const LOCAL_URLS = ['http://127.0.0.1', 'http://localhost']

// 已知安全的 localhost 片段：扫描前从行内剔除，同一行其余内容仍然检查。
// 只放行完整表达式而不是整行/整文件，避免真实地址和库代码混在一个压缩块里时被一起放过。
const ALLOWED_SNIPPETS = [
  // axios 浏览器适配器：拿不到 window.location 时（如 Worker）用它解析相对 URL，
  // 不是部署目标地址。将来 axios 改写法时这里会失配，门禁会失败而不是静默放过。
  /window\.location\.href\s*\|\|\s*["']http:\/\/localhost["']/g,
]

const target = resolve(process.argv[2] || '.output/public')
const failures = []
const seen = new Set()

function addFailure(rule, file, detail, line) {
  // 同一文件、同一规则、同一命中值只报一次，避免压缩后的单行 bundle 刷屏
  const key = `${rule}\0${file}\0${detail}`
  if (seen.has(key)) return
  seen.add(key)
  failures.push({ rule, file, detail, line })
}

function* walk(dir) {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) yield* walk(full)
    else if (entry.isFile()) yield full
  }
}

function rel(file) {
  return relative(target, file).split('\\').join('/')
}

function checkEntry() {
  const entry = join(target, 'index.html')
  if (!existsSync(entry) || !statSync(entry).isFile()) {
    addFailure('missing-entry', 'index.html', '缺少入口文件 index.html')
    return null
  }
  return entry
}

function checkAssets() {
  const dir = join(target, '_nuxt')
  if (!existsSync(dir) || !statSync(dir).isDirectory()) {
    addFailure('missing-assets', '_nuxt', '缺少 _nuxt 资源目录')
    return []
  }
  const assets = [...walk(dir)].filter((file) => ['.js', '.css'].includes(extname(file).toLowerCase()))
  if (assets.length === 0) {
    addFailure('missing-assets', '_nuxt', '_nuxt 目录下没有任何 JS/CSS 资源')
  }
  return assets
}

function scrubAllowed(line) {
  let out = line
  for (const pattern of ALLOWED_SNIPPETS) out = out.replace(pattern, '')
  return out
}

function checkContent() {
  let scanned = 0
  for (const file of walk(target)) {
    if (!TEXT_EXTENSIONS.has(extname(file).toLowerCase())) continue
    let text
    try {
      text = readFileSync(file, 'utf8')
    } catch (err) {
      addFailure('unreadable', rel(file), `无法读取：${err.message}`)
      continue
    }
    scanned += 1
    const lines = text.split('\n')
    for (const [index, line] of lines.entries()) {
      for (const host of FONT_HOSTS) {
        if (line.includes(host)) addFailure('remote-font', rel(file), host, index + 1)
      }
      const scrubbed = scrubAllowed(line)
      for (const url of LOCAL_URLS) {
        if (scrubbed.includes(url)) addFailure('localhost-url', rel(file), url, index + 1)
      }
    }
  }
  return scanned
}

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function main() {
  if (!existsSync(target) || !statSync(target).isDirectory()) {
    console.error(`[check-admin-static] FAIL 产物目录不存在或不是目录：${target}`)
    console.error('[check-admin-static] 先执行 npm run generate，或在参数里指定产物目录')
    process.exit(1)
  }

  const entry = checkEntry()
  const assets = checkAssets()
  const scanned = checkContent()

  if (failures.length > 0) {
    console.error(`[check-admin-static] FAIL ${target}`)
    for (const { rule, file, detail, line } of failures) {
      const location = line ? `${file}:${line}` : file
      console.error(`[check-admin-static] ${rule}: ${location} — ${detail}`)
    }
    console.error(`[check-admin-static] ${failures.length} 项规则未通过，产物不可发布`)
    process.exit(1)
  }

  const assetBytes = assets.reduce((sum, file) => sum + statSync(file).size, 0)
  console.log(`[check-admin-static] OK ${target}`)
  console.log(`[check-admin-static] entry: index.html (${formatBytes(statSync(entry).size)})`)
  console.log(`[check-admin-static] assets: ${assets.length} js/css (${formatBytes(assetBytes)})`)
  console.log(`[check-admin-static] scanned ${scanned} text files`)
}

main()
