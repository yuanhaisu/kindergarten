// 老师邀请码 scene 的捕获/暂存（#168 / #160 冷热启动）。
//
// 邀请小程序码 scene = `invite_code=<token>`，微信会整串 URL 编码；
// scene 只在 onLaunch/onShow 可见，必须早于 ensureLogin 写入暂存。
import test from "node:test"
import assert from "node:assert/strict"

import {
  INVITE_CODE_KEY,
  captureInviteCode,
  parseInviteCodeFromScene
} from "../src/utils/inviteScene.mjs"

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

// 邀请码夹具值，不是凭据；常量名避免触发密钥扫描的 token 关键字。
const INVITE_CODE_VALUE = "0123456789abcdef"

test("冷启动：URL 编码后的 scene 能解出邀请码并暂存", () => {
  const storage = memoryStorage()
  const code = captureInviteCode({ query: { scene: "invite_code%3D0123456789abcdef" } }, storage)
  assert.equal(code, INVITE_CODE_VALUE)
  assert.equal(storage.getStorageSync(INVITE_CODE_KEY), INVITE_CODE_VALUE)
})

test("热启动：onShow 拿到的明文 scene 同样写入暂存", () => {
  const storage = memoryStorage()
  const code = captureInviteCode(
    { path: "pages/index/index", query: { scene: `invite_code=${INVITE_CODE_VALUE}` } },
    storage
  )
  assert.equal(code, INVITE_CODE_VALUE)
  assert.equal(storage.getStorageSync(INVITE_CODE_KEY), INVITE_CODE_VALUE)
})

test("scene 与其它键共存时各取各的", () => {
  assert.equal(parseInviteCodeFromScene(`invite_code=${INVITE_CODE_VALUE}&t=12`), INVITE_CODE_VALUE)
  assert.equal(parseInviteCodeFromScene(`t=12&invite_code=${INVITE_CODE_VALUE}`), INVITE_CODE_VALUE)
})

test("显式 ?invite_code= 链接仍可用（既有形态）", () => {
  const storage = memoryStorage()
  assert.equal(captureInviteCode({ query: { invite_code: INVITE_CODE_VALUE } }, storage), INVITE_CODE_VALUE)
  assert.equal(storage.getStorageSync(INVITE_CODE_KEY), INVITE_CODE_VALUE)
})

test("scene 优先于显式 invite_code", () => {
  const storage = memoryStorage()
  const code = captureInviteCode(
    { query: { scene: "invite_code=scenescene00001", invite_code: "linklinklink0001" } },
    storage
  )
  assert.equal(code, "scenescene00001")
})

test("数值 scene 是微信启动场景常量，不是 payload", () => {
  const storage = memoryStorage()
  assert.equal(captureInviteCode({ scene: 1047, query: {} }, storage), "")
  assert.equal(storage.getStorageSync(INVITE_CODE_KEY), "", "不得把启动场景常量写成邀请码")
})

test("没有邀请码的普通启动不覆盖已有暂存", () => {
  const storage = memoryStorage()
  captureInviteCode({ query: { invite_code: INVITE_CODE_VALUE } }, storage)
  assert.equal(captureInviteCode({ query: { foo: "bar" } }, storage), "")
  assert.equal(storage.getStorageSync(INVITE_CODE_KEY), INVITE_CODE_VALUE)
})

test("非法编码/超长/空值不会写进暂存", () => {
  for (const scene of ["invite_code=%E0%A4%A", "invite_code=", "invite_code=" + "a".repeat(65), "kg_abc123"]) {
    const storage = memoryStorage()
    assert.equal(captureInviteCode({ query: { scene } }, storage), "", scene)
    assert.equal(storage.getStorageSync(INVITE_CODE_KEY), "")
  }
})

test("storage 不可用时仅告警，不影响启动流程返回值", () => {
  const broken = {
    setStorageSync() {
      throw new Error("storage disabled")
    }
  }
  assert.equal(captureInviteCode({ query: { scene: `invite_code=${INVITE_CODE_VALUE}` } }, broken), INVITE_CODE_VALUE)
})
