#!/usr/bin/env node
/**
 * check-content.mjs
 * 官网构建产物的上线内容门禁（ticket #128 / spec #126）。
 *
 * 为什么要检查构建产物而不是源码：`nuxt generate` 退出 0 只说明打包成功，
 * 不代表访客拿到的 HTML 里 ICP 号、主体、邮箱、canonical 还在，也不代表
 * 未发布的小程序码与「立即使用」没有混进来。门禁在打包机上拦截，
 * 而不是上线后靠人眼抽查。
 *
 * 零依赖，只用 Node 内建模块。
 *
 * 用法：
 *   node scripts/check-content.mjs [产物目录] [--production]
 *
 * 产物目录默认是 website/.output/public（相对本脚本所在项目解析，不随调用目录漂移）。
 * --production 追加法务 preflight：隐私政策 / 服务条款仍标「版本：草案」或
 * 「生效日期：待定」时拒绝发布；暂存模式允许草案。
 *
 * 退出码：0 = 通过；1 = 命中规则或产物不可读。
 */
import { existsSync, readFileSync, statSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const projectRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const args = process.argv.slice(2)
const production = args.includes('--production')
const positional = args.filter((arg) => !arg.startsWith('--'))
const target = resolve(positional[0] || join(projectRoot, '.output/public'))

const PAGES = [
  { key: 'home', label: '首页', file: 'index.html' },
  { key: 'privacy', label: '隐私政策', file: 'privacy/index.html' },
  { key: 'terms', label: '服务条款', file: 'terms/index.html' },
  { key: 'notFound', label: '404 页面', file: '404.html' },
]

const PRODUCT_NAME = '养成星球'
// 上线事实：ICP 备案已于 2026-11-13 通过，主体与邮箱随备案信息固定。
const ICP_NUMBER = '湘ICP备2026054245号-1'
const LEGAL_ENTITY = '湘潭市岳塘区稚序软件开发工作室'
const CONTACT_EMAIL = 'contact@nurtureplanet.cn'
const PRODUCTION_HOST = 'https://www.nurtureplanet.cn'

const failures = []

function fail(message) {
  failures.push(message)
}

function pageLabel(key) {
  const page = PAGES.find((item) => item.key === key)
  return `${page.label}（${page.file}）`
}

function decodeEntities(text) {
  const named = { amp: '&', lt: '<', gt: '>', quot: '"', apos: "'", nbsp: ' ' }
  return text.replace(/&(#x?[0-9a-f]+|[a-z]+);/gi, (match, entity) => {
    if (entity[0] === '#') {
      const code =
        entity[1] === 'x' || entity[1] === 'X'
          ? Number.parseInt(entity.slice(2), 16)
          : Number.parseInt(entity.slice(1), 10)
      return Number.isInteger(code) && code >= 0 && code <= 0x10ffff
        ? String.fromCodePoint(code)
        : match
    }
    return named[entity.toLowerCase()] ?? match
  })
}

// 先去掉脚本/样式/注释，再剥标签，避免把 JSON-LD 或压缩 bundle 里的字符串
// 当成访客可见的文案。
function toVisibleText(html) {
  return decodeEntities(
    html
      .replace(/<!--[\s\S]*?-->/g, ' ')
      .replace(/<(script|style)\b[^>]*>[\s\S]*?<\/\1\s*>/gi, ' ')
      .replace(/<[^>]+>/g, ' '),
  )
    .replace(/\s+/g, ' ')
    .trim()
}

function requireFact(visible, key, needle, description) {
  const text = visible[key]
  if (text === undefined) return // 页面缺失已在上面报过
  if (!text.includes(needle)) fail(`${pageLabel(key)}：缺少${description}`)
}

function attr(tag, name) {
  const match = tag.match(
    new RegExp(`\\b${name}\\s*=\\s*(?:"([^"]*)"|'([^']*)'|([^\\s"'>]+))`, 'i'),
  )
  return match ? (match[1] ?? match[2] ?? match[3]) : null
}

function canonicalHref(html) {
  for (const tag of html.match(/<link\b[^>]*>/gi) ?? []) {
    const rel = attr(tag, 'rel')
    if (rel && rel.split(/\s+/).includes('canonical')) return attr(tag, 'href')
  }
  return null
}

function checkCanonical(sources) {
  for (const page of PAGES) {
    if (page.key === 'notFound') continue // 404 不参与 SEO canonical
    const html = sources[page.key]
    if (html === undefined) continue
    const href = canonicalHref(html)
    if (!href) {
      fail(`${pageLabel(page.key)}：缺少 canonical 链接`)
      continue
    }
    let origin
    try {
      origin = new URL(href).origin
    } catch {
      origin = null
    }
    if (origin !== PRODUCTION_HOST) {
      fail(
        `${pageLabel(page.key)}：canonical 指向「${href}」，生产站要求「${PRODUCTION_HOST}」`,
      )
    }
  }
}

function checkSitemap(xml) {
  if (xml === undefined) return
  const locations = [...xml.matchAll(/<loc>\s*([^<]+?)\s*<\/loc>/gi)].map((match) => match[1])
  if (locations.length === 0) {
    fail('sitemap.xml：没有任何 <loc> 条目')
    return
  }
  for (const location of locations) {
    let origin
    try {
      origin = new URL(location).origin
    } catch {
      origin = null
    }
    if (origin !== PRODUCTION_HOST) {
      fail(`sitemap.xml：<loc> 使用了非生产域名「${location}」（要求 ${PRODUCTION_HOST}）`)
    }
  }
}

// 法务草案标记：隐私政策 / 服务条款顶部的 meta 行是硬编码的，
// 只把 legalDraft 改成 false 不会清掉它。生产发布前必须两条都消失。
const DRAFT_MARKERS = ['版本：草案', '生效日期：待定']

function checkLegalDraftMarkers(visible) {
  for (const key of ['privacy', 'terms']) {
    const text = visible[key]
    if (text === undefined) continue
    for (const marker of DRAFT_MARKERS) {
      if (text.includes(marker)) {
        fail(
          `${pageLabel(key)}：仍标注「${marker}」，生产发布前须完成法务复核并填写生效日期`,
        )
      }
    }
  }
}

// 班内共享语义钉住：线上后端仍支持 allow_class_share，家长端设置页仍可关闭。
// 文案必须与已上线行为一致；#88 移除开关的改版落地时，网站文案与本门禁
// 必须在同一次变更里更新，不能提前改口。
const CLASS_SHARE_TERM = '班内共享'
const CLASS_SHARE_DEFAULT = '默认开启'
const CLASS_SHARE_WITHDRAW = ['可随时撤回', '可随时关闭']
// 未来的「没有家长开关」文案：出现即说明文案先于产品改了。
// 覆盖 不提供…开关 / 无法关闭 / 不能关闭 / 不可关闭 / 没有…开关 等变体。
const CLASS_SHARE_DRIFT =
  /(?:不提供|无法|不能|不可)[^。；\n]{0,30}(?:开关|关闭)|没有[^。；\n]{0,20}开关/

function checkClassShare(visible) {
  for (const key of ['home', 'privacy']) {
    const text = visible[key]
    if (text === undefined) continue
    if (!text.includes(CLASS_SHARE_TERM)) {
      fail(`${pageLabel(key)}：缺少「${CLASS_SHARE_TERM}」说明（当前后端仍支持家长关闭共享）`)
      continue
    }
    if (!text.includes(CLASS_SHARE_DEFAULT)) {
      fail(`${pageLabel(key)}：「${CLASS_SHARE_TERM}」仍应说明默认开启`)
    }
    if (!CLASS_SHARE_WITHDRAW.some((phrase) => text.includes(phrase))) {
      fail(`${pageLabel(key)}：缺少「可随时撤回 / 可随时关闭」，当前家长仍可关闭共享`)
    }
  }
}

// 小程序尚未发布：只禁止「实际的小程序码素材」与指向它的行动召唤。
// 隐私政策里对微信第三方的文字披露（「小程序码生成 / 小程序码参数」）是
// 合规说明，不是 QR 素材；企业微信二维码已获准，均放行。
const SMALL_PROGRAM_QR = /小程序码|miniprogram|wxacode|xcx/i
const WECOM_QR = /企业微信|wecom|work\.weixin/i

// 图片素材候选：src、srcset（逐条 URL）与 alt。浏览器可能选中 srcset 里的
// 任意候选，只看 src 会被 `src="占位.webp" srcset="小程序码.webp 2x"` 绕过。
function imageCandidates(html) {
  const candidates = []
  for (const tag of html.match(/<(?:img|source)\b[^>]*>/gi) ?? []) {
    const values = []
    for (const name of ['src', 'srcset', 'alt']) {
      const value = attr(tag, name)
      if (!value) continue
      if (name === 'srcset') {
        for (const entry of value.split(',')) {
          const url = entry.trim().split(/\s+/)[0]
          if (url) values.push(url)
        }
      } else {
        values.push(value)
      }
    }
    candidates.push({ tag, values })
  }
  return candidates
}

// CSS background-image 的 url()：内联 style 与 <style> 块都要扫，否则把
// QR 换成背景图即可绕过素材检查。只看 background 声明，避开字体等资源。
function backgroundImageUrls(html) {
  const declarations = []
  for (const match of html.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style\s*>/gi)) {
    declarations.push(match[1])
  }
  for (const match of html.matchAll(/\bstyle\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))/gi)) {
    declarations.push(match[1] ?? match[2] ?? match[3])
  }

  const urls = []
  for (const css of declarations) {
    for (const declaration of css.matchAll(/background(?:-image)?\s*:\s*([^;{}]+)/gi)) {
      for (const url of declaration[1].matchAll(/url\(\s*(?:"([^"]*)"|'([^']*)'|([^)'"]+))\s*\)/gi)) {
        const value = (url[1] ?? url[2] ?? url[3] ?? '').trim()
        if (value) urls.push(value)
      }
    }
  }
  return urls
}

const QR_CTA_STRONG = /扫码/
const QR_CTA_GENERIC = /立即|体验|查看/

// 行动召唤：「扫码」本身已足够指向 QR；其余动词只在链接/按钮携带图片时
// 才算 QR CTA（src 不含关键词也要能识别），避免把普通文本链接误判。
function checkCta(html, key) {
  for (const block of html.match(/<(a|button)\b[^>]*>[\s\S]*?<\/\1\s*>/gi) ?? []) {
    // 企业微信二维码已获准：整段带 wecom 标记的 CTA 放行。
    if (WECOM_QR.test(block)) continue
    const label = toVisibleText(block)
    const altText = [...block.matchAll(/<img\b[^>]*>/gi)]
      .map((match) => attr(match[0], 'alt') ?? '')
      .join(' ')
    const text = `${label} ${altText}`.trim()
    if (!text) continue
    const hasImage = /<(?:img|picture|source)\b/i.test(block)
    if (!QR_CTA_STRONG.test(text) && !(hasImage && QR_CTA_GENERIC.test(text))) continue
    fail(`${pageLabel(key)}：出现小程序行动召唤「${text}」，小程序尚未发布`)
  }
}

function checkForbidden(html, visibleText, key) {
  if (visibleText.includes('立即使用')) {
    fail(`${pageLabel(key)}：出现「立即使用」行动召唤，小程序尚未发布`)
  }

  const drift = visibleText.match(CLASS_SHARE_DRIFT)
  if (drift) {
    fail(
      `${pageLabel(key)}：出现「${drift[0]}」文案，当前后端仍支持家长关闭班内共享（等 #88 落地再改）`,
    )
  }

  for (const { tag, values } of imageCandidates(html)) {
    const source = values.join(' ')
    if (WECOM_QR.test(source)) continue // 企业微信二维码允许
    if (values.some((value) => SMALL_PROGRAM_QR.test(value))) {
      fail(`${pageLabel(key)}：出现小程序码素材 ${tag.trim()}，小程序尚未发布`)
    }
  }

  for (const url of backgroundImageUrls(html)) {
    if (WECOM_QR.test(url)) continue
    if (SMALL_PROGRAM_QR.test(url)) {
      fail(`${pageLabel(key)}：出现小程序码背景素材「${url}」，小程序尚未发布`)
    }
  }

  checkCta(html, key)
}

function loadText(relative) {
  const file = join(target, relative)
  if (!existsSync(file) || !statSync(file).isFile()) return null
  try {
    return readFileSync(file, 'utf8')
  } catch (err) {
    fail(`${relative}：无法读取 — ${err.message}`)
    return null
  }
}

function main() {
  if (!existsSync(target) || !statSync(target).isDirectory()) {
    console.error(`[check-content] FAIL 产物目录不存在或不是目录：${target}`)
    console.error('[check-content] 先执行 npm run generate，或在参数里指定产物目录')
    process.exit(1)
  }

  const sources = {}
  for (const page of PAGES) {
    const content = loadText(page.file)
    if (content === null) fail(`${pageLabel(page.key)}：产物缺失`)
    else sources[page.key] = content
  }

  const assets = {}
  for (const asset of ['robots.txt', 'sitemap.xml']) {
    const content = loadText(asset)
    if (content === null) fail(`${asset}：产物缺失`)
    else assets[asset] = content
  }

  const visible = {}
  for (const [key, html] of Object.entries(sources)) visible[key] = toVisibleText(html)

  // 上线事实：首页、隐私政策、服务条款都必须能读到同一组备案与联系信息。
  requireFact(visible, 'home', PRODUCT_NAME, `产品名「${PRODUCT_NAME}」`)
  requireFact(visible, 'notFound', PRODUCT_NAME, `产品名「${PRODUCT_NAME}」（品牌 404）`)
  for (const key of ['home', 'privacy', 'terms']) {
    requireFact(visible, key, ICP_NUMBER, ` ICP 备案号「${ICP_NUMBER}」`)
    requireFact(visible, key, LEGAL_ENTITY, `主体名称「${LEGAL_ENTITY}」`)
    requireFact(visible, key, CONTACT_EMAIL, `联系邮箱「${CONTACT_EMAIL}」`)
  }

  checkCanonical(sources)
  checkSitemap(assets['sitemap.xml'])
  checkClassShare(visible)
  if (production) checkLegalDraftMarkers(visible)
  for (const page of PAGES) {
    if (sources[page.key] === undefined) continue
    checkForbidden(sources[page.key], visible[page.key], page.key)
  }

  if (failures.length > 0) {
    console.error(`[check-content] FAIL ${target}`)
    for (const failure of failures) console.error(`[check-content] ${failure}`)
    process.exit(1)
  }

  console.log(
    `[check-content] OK ${PAGES.length} 个页面，robots.txt 与 sitemap.xml 存在（${production ? '生产模式' : '暂存模式'}）：${target}`,
  )
}

main()
