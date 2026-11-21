// 园所入口 scene 的捕获/暂存/落园决策（#176）。
//
// 冷启动经 App.onLaunch(options)，热启动经 App.onShow(options)；两者都在
// 登录之前、独立于登录执行。storage 由调用方注入（小程序里是 uni）。
import test from "node:test"
import assert from "node:assert/strict"

import {
  PENDING_ENTRY_KEY,
  PENDING_ENTRY_TTL_MS,
  captureEntryScene,
  readPendingEntry,
  clearPendingEntry,
  resolveEntryAction
} from "../src/utils/entryScene.mjs"

function memoryStorage() {
  const map = new Map()
  return {
    getStorageSync(key) {
      return map.has(key) ? map.get(key) : ""
    },
    setStorageSync(key, value) {
      map.set(key, value)
    },
    removeStorageSync(key) {
      map.delete(key)
    }
  }
}

const NOW = Date.UTC(2026, 10, 21, 8, 0, 0)

test("冷启动：onLaunch 的 query.scene 被暂存", () => {
  const storage = memoryStorage()
  const code = captureEntryScene({ query: { scene: "kg_abc123defg" } }, storage, NOW)
  assert.equal(code, "abc123defg")
  assert.deepEqual(JSON.parse(storage.getStorageSync(PENDING_ENTRY_KEY)), {
    public_code: "abc123defg",
    captured_at: NOW
  })
})

test("热启动：onShow 的 query.scene 同样被暂存", () => {
  const storage = memoryStorage()
  const code = captureEntryScene({ path: "pages/index/index", query: { scene: "kg%5Fabc123defg" } }, storage, NOW)
  assert.equal(code, "abc123defg")
  assert.equal(JSON.parse(storage.getStorageSync(PENDING_ENTRY_KEY)).public_code, "abc123defg")
})

test("数值 scene 是启动场景常量，不是 payload，不得误存", () => {
  const storage = memoryStorage()
  assert.equal(captureEntryScene({ scene: 1047, query: {} }, storage, NOW), "")
  assert.equal(storage.getStorageSync(PENDING_ENTRY_KEY), "")
})

test("普通 query 不带园所码时不覆盖待处理入口", () => {
  const storage = memoryStorage()
  captureEntryScene({ query: { scene: "kg_abc123defg" } }, storage, NOW)
  assert.equal(captureEntryScene({ query: { foo: "bar" } }, storage, NOW + 1000), "")
  assert.equal(JSON.parse(storage.getStorageSync(PENDING_ENTRY_KEY)).public_code, "abc123defg")
})

test("显式 ?public_code= 链接也作为入口（#167 已有形态）", () => {
  const storage = memoryStorage()
  assert.equal(captureEntryScene({ query: { public_code: "abc123defg" } }, storage, NOW), "abc123defg")
})

test("scene 优先于显式 public_code", () => {
  const storage = memoryStorage()
  const code = captureEntryScene({
    query: { scene: "kg_scenecde1", public_code: "linkcde0001" }
  }, storage, NOW)
  assert.equal(code, "scenecde1")
})

test("待处理入口有 TTL：过期即失效并清除", () => {
  const storage = memoryStorage()
  captureEntryScene({ query: { scene: "kg_abc123defg" } }, storage, NOW)
  assert.equal(readPendingEntry(storage, NOW + PENDING_ENTRY_TTL_MS - 1)?.public_code, "abc123defg")
  assert.equal(readPendingEntry(storage, NOW + PENDING_ENTRY_TTL_MS + 1), null)
  assert.equal(storage.getStorageSync(PENDING_ENTRY_KEY), "", "过期项必须清除")
})

test("损坏的暂存值不会把启动卡住", () => {
  const storage = memoryStorage()
  storage.setStorageSync(PENDING_ENTRY_KEY, "{not-json")
  assert.equal(readPendingEntry(storage, NOW), null)
  assert.equal(storage.getStorageSync(PENDING_ENTRY_KEY), "")
})

test("clearPendingEntry 只清入口键", () => {
  const storage = memoryStorage()
  storage.setStorageSync("parent_token", "t")
  captureEntryScene({ query: { scene: "kg_abc123defg" } }, storage, NOW)
  clearPendingEntry(storage)
  assert.equal(storage.getStorageSync(PENDING_ENTRY_KEY), "")
  assert.equal(storage.getStorageSync("parent_token"), "t")
})

test("落园决策：成员园且非当前园 → 自动切换", () => {
  const plan = resolveEntryAction({
    publicCode: "abc123defg",
    memberships: [
      { kindergarten_id: 7, name: "甲园", public_code: "zzzzzzzzzz" },
      { kindergarten_id: 9, name: "乙园", public_code: "abc123defg" }
    ],
    currentKindergartenID: 7
  })
  assert.deepEqual(plan, { action: "switch", kindergartenID: 9, name: "乙园" })
})

test("落园决策：已在目标园 → 无事可做", () => {
  const plan = resolveEntryAction({
    publicCode: "abc123defg",
    memberships: [{ kindergarten_id: 9, name: "乙园", public_code: "abc123defg" }],
    currentKindergartenID: 9
  })
  assert.equal(plan.action, "none")
})

test("落园决策：不在成员列表 → 绑定引导（不提供园所目录）", () => {
  const plan = resolveEntryAction({
    publicCode: "abc123defg",
    memberships: [{ kindergarten_id: 9, name: "乙园", public_code: "zzzzzzzzzz" }],
    currentKindergartenID: 9
  })
  assert.deepEqual(plan, { action: "bind", publicCode: "abc123defg" })
})

test("落园决策：没有入口码 → 不动", () => {
  assert.equal(resolveEntryAction({ publicCode: "", memberships: [], currentKindergartenID: 0 }).action, "none")
})
