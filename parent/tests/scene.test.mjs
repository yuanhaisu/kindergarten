// 小程序码 scene 解析（#176）：园所入口码 kg_<public_code> 与既有任务码 t=<id> 共存。
import test from "node:test"
import assert from "node:assert/strict"

import {
  decodeScene,
  parseScenePublicCode,
  parseSceneTaskId
} from "../src/utils/scene.mjs"

test("任务码行为保持不变", () => {
  assert.equal(parseSceneTaskId("t=42"), 42)
  assert.equal(parseSceneTaskId("t%3D42"), 42, "整串被编码一次也要能解析")
  assert.equal(parseSceneTaskId("from=qr&t=7"), 7)
  assert.equal(parseSceneTaskId("kg_abc123defg"), 0, "园所码不得被当任务码")
  assert.equal(parseSceneTaskId(""), 0)
})

test("园所码 kg_<public_code>", () => {
  assert.equal(parseScenePublicCode("kg_abc123defg"), "abc123defg")
  // 整串被 URL 编码一次（%5F = _）
  assert.equal(parseScenePublicCode("kg%5Fabc123defg"), "abc123defg")
  // 与任务码共存：各取各的，不互相吞掉
  assert.equal(parseScenePublicCode("kg_abc123defg&t=12"), "abc123defg")
  assert.equal(parseSceneTaskId("kg_abc123defg&t=12"), 12)
})

test("非园所码不得误判", () => {
  assert.equal(parseScenePublicCode("t=12"), "")
  assert.equal(parseScenePublicCode("kg_"), "")
  assert.equal(parseScenePublicCode("kg_ABC123"), "", "公开码是小写字母表")
  assert.equal(parseScenePublicCode("kg_abci"), "", "i/l/o/u 不在 Crockford 字母表")
  assert.equal(parseScenePublicCode("xkg_abc123defg"), "", "必须是独立段")
  assert.equal(parseScenePublicCode(null), "")
})

test("非法编码按原样解析而不是抛错", () => {
  assert.equal(decodeScene("%E0%A4%A"), "%E0%A4%A")
  assert.equal(parseScenePublicCode("kg_abc123defg%"), "abc123defg")
})
