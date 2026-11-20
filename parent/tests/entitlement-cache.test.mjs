import test from "node:test"
import assert from "node:assert/strict"

import {
  clearEntitlementCache,
  entitlementCacheEpoch
} from "../src/utils/entitlement-cache.mjs"

function fakeStorage(initial = {}) {
  const data = new Map(Object.entries(initial))
  return {
    data,
    removeStorageSync(key) {
      data.delete(key)
    }
  }
}

test("清理授权缓存只移除 entitlement 两个 key", () => {
  const storage = fakeStorage({
    entitlement_cache: { modules: ["growth"], expires_at: "2026-12-31T00:00:00Z" },
    entitlement_cache_at: 1700000000000,
    parent_token: "t",
    parent_user: { id: 1 }
  })

  clearEntitlementCache(storage)

  assert.equal(storage.data.has("entitlement_cache"), false)
  assert.equal(storage.data.has("entitlement_cache_at"), false)
  assert.equal(storage.data.get("parent_token"), "t")
  assert.deepEqual(storage.data.get("parent_user"), { id: 1 })
})

test("清理一次递增缓存代数", () => {
  const before = entitlementCacheEpoch()

  clearEntitlementCache(fakeStorage())

  assert.equal(entitlementCacheEpoch(), before + 1)
})

test("连续清理按次数递增缓存代数", () => {
  const before = entitlementCacheEpoch()

  clearEntitlementCache(fakeStorage())
  clearEntitlementCache(fakeStorage())

  assert.equal(entitlementCacheEpoch(), before + 2)
})

test("没有 storage 时安全返回并递增缓存代数", () => {
  const before = entitlementCacheEpoch()

  assert.doesNotThrow(() => clearEntitlementCache())
  assert.doesNotThrow(() => clearEntitlementCache(null))

  assert.equal(entitlementCacheEpoch(), before + 2)
})

test("存储异常不阻断清理流程且留下告警", () => {
  const storage = {
    removeStorageSync() {
      throw new Error("storage unavailable")
    }
  }
  const originalWarn = console.warn
  const warnings = []
  console.warn = (...args) => warnings.push(args.join(" "))
  try {
    assert.doesNotThrow(() => clearEntitlementCache(storage))
  } finally {
    console.warn = originalWarn
  }

  assert.equal(warnings.length, 1)
  assert.match(warnings[0], /clear cache failed/)
  assert.match(warnings[0], /storage unavailable/)
})
