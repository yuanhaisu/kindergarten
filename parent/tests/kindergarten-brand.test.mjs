// 当前园所品牌的客户端归一与缓存（#160 L14/L235）。
//
// 公开品牌接口的 brand_color 会直接进 inline style，logo 是 10 分钟签名 URL：
// 这里锁住「只收 #RRGGBB」「缓存不存 logo」「缓存有过期与条数上限」三条契约。
import test from "node:test"
import assert from "node:assert/strict"

import {
  BRAND_CACHE_STORAGE_KEY,
  BRAND_CACHE_TTL_MS,
  normalizeBrandColor,
  pickPublicBrand,
  readCachedBrand,
  resolveCurrentKindergarten,
  tintBrandColor,
  writeCachedBrand
} from "../src/utils/kindergartenBrand.mjs"

function memoryStorage() {
  const map = new Map()
  return {
    getStorageSync(key) {
      return map.has(key) ? map.get(key) : ""
    },
    setStorageSync(key, value) {
      map.set(key, value)
    }
  }
}

const BRAND = {
  public_code: "k3m9x2qa7t",
  name: "阳光幼儿园",
  logo_url: "https://oss.example.com/logo.png?sign=abc",
  brand_color: "#ff8800"
}

test("品牌色只接受 #RRGGBB，其余一律清空", () => {
  assert.equal(normalizeBrandColor("#ff8800"), "#FF8800")
  assert.equal(normalizeBrandColor("ff8800"), "#FF8800")
  assert.equal(normalizeBrandColor("  #FF8800  "), "#FF8800")
  for (const raw of ["", null, "#fff", "#ff88001", "red", "rgb(1,2,3)", "url(x)", "#ff8800;background:url(x)"]) {
    assert.equal(normalizeBrandColor(raw), "", String(raw))
  }
})

test("品牌柔底按主色向白色混合，非法色不产出样式", () => {
  assert.equal(tintBrandColor("#FF8800"), "#FFF5EB")
  assert.equal(tintBrandColor("#000000", 0.5), "#808080")
  assert.equal(tintBrandColor("nope"), "")
})

test("成员列表按 id 找当前园，兼容数字/字符串 id", () => {
  const items = [{ kindergarten_id: 4, name: "阳光幼儿园", public_code: "k3m9x2qa7t" }]
  assert.equal(resolveCurrentKindergarten(items, 4)?.name, "阳光幼儿园")
  assert.equal(resolveCurrentKindergarten(items, "4")?.name, "阳光幼儿园")
  assert.equal(resolveCurrentKindergarten(items, 7), null)
  assert.equal(resolveCurrentKindergarten(items, 0), null)
  assert.equal(resolveCurrentKindergarten(null, 4), null)
})

test("公开品牌只保留四个公开字段", () => {
  const brand = pickPublicBrand({
    ...BRAND,
    address: "银湖南路 1 号",
    phone: "13900000000",
    principal: "李园长"
  })
  assert.deepEqual(Object.keys(brand).sort(), ["brand_color", "logo_url", "name", "public_code"])
  assert.equal(brand.brand_color, "#FF8800")
  assert.equal(brand.name, "阳光幼儿园")
})

test("缺少 public_code 或 name 视为无效响应", () => {
  assert.equal(pickPublicBrand({ ...BRAND, public_code: " " }), null)
  assert.equal(pickPublicBrand({ ...BRAND, name: "" }), null)
  assert.equal(pickPublicBrand(null), null)
})

test("缓存只存名称与主色：签名 logo 不跨页复用", () => {
  const storage = memoryStorage()
  writeCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, BRAND.public_code, pickPublicBrand(BRAND), 1000)

  const cached = readCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, BRAND.public_code, 2000)
  assert.equal(cached.name, "阳光幼儿园")
  assert.equal(cached.brand_color, "#FF8800")
  assert.equal(cached.logo_url, "", "签名 URL 有过期时间，不得写进缓存")
})

test("缓存过期或键不符时返回 null", () => {
  const storage = memoryStorage()
  writeCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, BRAND.public_code, BRAND, 1000)
  assert.ok(readCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, BRAND.public_code, 1000), "写入当刻必须命中")
  assert.equal(
    readCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, BRAND.public_code, 1000 + BRAND_CACHE_TTL_MS + 1),
    null
  )
  assert.equal(readCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, "other", 2000), null)
})

test("缓存条数有上限，超出的按最旧淘汰", () => {
  const storage = memoryStorage()
  for (let i = 0; i < 12; i += 1) {
    writeCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, `code${i}`, { ...BRAND, public_code: `code${i}` }, 1000 + i)
  }
  assert.equal(readCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, "code0", 2000), null, "最旧两条必须被淘汰")
  assert.equal(readCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, "code1", 2000), null)
  assert.ok(readCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, "code2", 2000))
  assert.ok(readCachedBrand(storage, BRAND_CACHE_STORAGE_KEY, "code11", 2000))
})

test("storage 不可用只告警，不影响页面继续", () => {
  const broken = {
    getStorageSync() {
      throw new Error("storage disabled")
    },
    setStorageSync() {
      throw new Error("storage disabled")
    }
  }
  assert.equal(readCachedBrand(broken, BRAND_CACHE_STORAGE_KEY, BRAND.public_code), null)
  assert.doesNotThrow(() =>
    writeCachedBrand(broken, BRAND_CACHE_STORAGE_KEY, BRAND.public_code, BRAND)
  )
})
