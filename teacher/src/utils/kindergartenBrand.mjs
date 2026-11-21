// 当前园所品牌（#160 L14/L235）。
//
// 公开品牌接口只返回 { public_code, name, logo_url, brand_color }；客户端拿它给
// 「当前园」展示名称/logo/主色。品牌色会直接进 inline style，因此在客户端再归一
// 一次：只接受 #RRGGBB，其余一律退回 token 样式。
//
// 纯函数 + 显式注入 storage，便于 node --test 覆盖，不依赖 uni 运行时。

export const BRAND_CACHE_STORAGE_KEY = "kg_public_brand_cache"
export const BRAND_CACHE_TTL_MS = 6 * 60 * 60 * 1000
export const BRAND_CACHE_MAX_ENTRIES = 10

const HEX_COLOR_RE = /^#?([0-9a-fA-F]{6})$/

export function normalizeBrandColor(raw) {
  const match = HEX_COLOR_RE.exec(String(raw || "").trim())
  return match ? `#${match[1].toUpperCase()}` : ""
}

function channelOf(hex, offset) {
  return parseInt(hex.slice(offset, offset + 2), 16)
}

// 主色向白色混合出浅底（默认 8%），给 hero 这类品牌面垫底。
// 不用 rgba：小程序里叠在暖色渐变上会发灰，实色更稳定。
export function tintBrandColor(color, weight = 0.08) {
  const hex = normalizeBrandColor(color)
  if (!hex) return ""
  const ratio = Math.min(Math.max(Number(weight) || 0, 0), 1)
  const mix = (channel) => Math.round(channel * ratio + 255 * (1 - ratio))
  const parts = [1, 3, 5].map((offset) =>
    mix(channelOf(hex, offset)).toString(16).padStart(2, "0")
  )
  return `#${parts.join("").toUpperCase()}`
}

// 成员列表（GET /common/user/kindergartens）里按园所 id 找当前园。
// id 可能来自 storage 或 query，统一转数字比较。
export function resolveCurrentKindergarten(items, kindergartenID) {
  const id = Number(kindergartenID)
  if (!(id > 0) || !Array.isArray(items)) return null
  return items.find((item) => Number(item?.kindergarten_id) === id) || null
}

// 收窄公开品牌响应：只留四个公开字段，并做类型/空白归一。
// 名称或公开码缺失视为无效响应，页面退回 token 兜底。
export function pickPublicBrand(payload) {
  if (!payload || typeof payload !== "object") return null
  const publicCode = String(payload.public_code || "").trim()
  const name = String(payload.name || "").trim()
  if (!publicCode || !name) return null
  return {
    public_code: publicCode,
    name,
    logo_url: String(payload.logo_url || "").trim(),
    brand_color: normalizeBrandColor(payload.brand_color)
  }
}

// logo 是 10 分钟签名 URL，跨页缓存它必然过期，所以缓存只保留名称与主色；
// 读回时 logo_url 为空，页面先显示字形，本次拉取成功后再换成 logo。
export function readCachedBrand(storage, cacheKey, publicCode, now = Date.now()) {
  const code = String(publicCode || "").trim()
  if (!code || !storage) return null
  try {
    const raw = storage.getStorageSync(cacheKey)
    const map = raw && typeof raw === "object" ? raw : {}
    const entry = map[code]
    if (!entry || !entry.brand) return null
    const at = Number(entry.at)
    if (!(at > 0) || now - at > BRAND_CACHE_TTL_MS) return null
    const brand = pickPublicBrand({ ...entry.brand, logo_url: "" })
    return brand
  } catch (err) {
    return null
  }
}

export function writeCachedBrand(storage, cacheKey, publicCode, brand, now = Date.now()) {
  const code = String(publicCode || "").trim()
  const normalized = pickPublicBrand(brand)
  if (!code || !normalized || !storage) return
  try {
    const raw = storage.getStorageSync(cacheKey)
    const map = raw && typeof raw === "object" ? raw : {}
    map[code] = {
      brand: {
        public_code: normalized.public_code,
        name: normalized.name,
        brand_color: normalized.brand_color
      },
      at: now
    }
    const codes = Object.keys(map)
    if (codes.length > BRAND_CACHE_MAX_ENTRIES) {
      codes
        .sort((a, b) => Number(map[a]?.at || 0) - Number(map[b]?.at || 0))
        .slice(0, codes.length - BRAND_CACHE_MAX_ENTRIES)
        .forEach((stale) => delete map[stale])
    }
    storage.setStorageSync(cacheKey, map)
  } catch (err) {
    // 缓存不可用只影响一次多余请求，不影响品牌展示本身。
    console.warn("[brand] persist brand cache failed:", err?.message || err)
  }
}
