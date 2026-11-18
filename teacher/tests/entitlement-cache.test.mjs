import test from "node:test"
import assert from "node:assert/strict"

import {
  ENTITLEMENT_CACHE_KEY,
  ENTITLEMENT_CACHE_AT_KEY,
  clearEntitlementCache
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
    teacher_token: "t",
    teacher_user: { id: 1 }
  })

  clearEntitlementCache(storage)

  assert.equal(storage.data.has("entitlement_cache"), false)
  assert.equal(storage.data.has("entitlement_cache_at"), false)
  assert.equal(storage.data.get("teacher_token"), "t")
  assert.deepEqual(storage.data.get("teacher_user"), { id: 1 })
})

test("缓存 key 是两端共用的存储契约", () => {
  assert.equal(ENTITLEMENT_CACHE_KEY, "entitlement_cache")
  assert.equal(ENTITLEMENT_CACHE_AT_KEY, "entitlement_cache_at")
})

test("存储异常不阻断清理流程", () => {
  const storage = {
    removeStorageSync() {
      throw new Error("storage unavailable")
    }
  }

  assert.doesNotThrow(() => clearEntitlementCache(storage))
})

test("没有 storage 时安全返回", () => {
  assert.doesNotThrow(() => clearEntitlementCache())
  assert.doesNotThrow(() => clearEntitlementCache(null))
})
