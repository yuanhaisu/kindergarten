// 登录/登出路径的授权缓存清理接线。
//
// 覆盖范围：clearAuth —— 设置页登出、http.js 的 401 失效、同意拒绝三条路径最终都汇到这里。
// 未覆盖：silentLogin 里的清理调用依赖 @/api/* 别名，Node 无法直接解析；
// 该调用点由编译产物与运行时桩验证（见 docs/tests/TEST_CATALOG.md TC-NEW-ENT-0020）。
import test from "node:test"
import assert from "node:assert/strict"

const storage = new Map()

globalThis.uni = {
  getStorageSync(key) {
    return storage.has(key) ? storage.get(key) : ""
  },
  setStorageSync(key, value) {
    storage.set(key, value)
  },
  removeStorageSync(key) {
    storage.delete(key)
  }
}

const { clearAuth } = await import("../src/utils/auth.js")

function seed() {
  storage.clear()
  storage.set("parent_token", "token-a")
  storage.set("parent_refresh_token", "refresh-a")
  storage.set("parent_user", { id: 1, name: "Parent A" })
  storage.set("entitlement_cache", { modules: ["growth"], expires_at: "2026-12-31T00:00:00Z" })
  storage.set("entitlement_cache_at", 1700000000000)
  storage.set("parent_unrelated", "keep")
}

test("clearAuth 清理登录态与授权缓存", () => {
  seed()

  clearAuth()

  assert.equal(storage.has("parent_token"), false)
  assert.equal(storage.has("parent_refresh_token"), false)
  assert.equal(storage.has("parent_user"), false)
  assert.equal(storage.has("entitlement_cache"), false)
  assert.equal(storage.has("entitlement_cache_at"), false)
  assert.equal(storage.get("parent_unrelated"), "keep")
})
