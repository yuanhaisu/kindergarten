import { fetchPublicKindergarten } from "@/api/public"
import {
  BRAND_CACHE_STORAGE_KEY,
  pickPublicBrand,
  readCachedBrand,
  resolveCurrentKindergarten,
  writeCachedBrand
} from "./kindergartenBrand.mjs"

// 取「当前园」的公开品牌（#160 L14/L235）。
//
// public_code 只从本人成员列表里取：家长端不请求任何园所目录（见
// tests/no-directory.test.mjs）。品牌失败不阻断园所列表——有缓存用缓存，
// 没有就返回 null，页面按 token 样式兜底。
export async function loadCurrentKindergartenBrand(memberships, kindergartenID) {
  const current = resolveCurrentKindergarten(memberships, kindergartenID)
  const code = String(current?.public_code || "").trim()
  if (!code) return null

  const cached = readCachedBrand(uni, BRAND_CACHE_STORAGE_KEY, code)
  try {
    const res = await fetchPublicKindergarten(code)
    const brand = pickPublicBrand(res?.data)
    if (!brand) return cached
    // 内存态保留本次签名后的 logo_url；缓存只写名称/主色（见 kindergartenBrand.mjs）。
    writeCachedBrand(uni, BRAND_CACHE_STORAGE_KEY, code, brand)
    return brand
  } catch (err) {
    console.warn("[brand] load public brand failed:", err?.message || err)
    return cached
  }
}
