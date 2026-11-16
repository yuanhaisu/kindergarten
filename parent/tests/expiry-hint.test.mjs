import test from "node:test"
import assert from "node:assert/strict"

import { expiryHintText, EXPIRY_WARNING_DAYS } from "../src/utils/expiry-hint.mjs"

const NOW = Date.parse("2026-11-16T00:00:00Z")
const RENEW = "请联系园所管理员续费"

test("空值与非法日期不显示", () => {
  assert.equal(expiryHintText(null, NOW, RENEW), "")
  assert.equal(expiryHintText("not-a-date", NOW, RENEW), "")
})

test("超过 30 天不显示", () => {
  assert.equal(expiryHintText("2026-12-17T00:00:01Z", NOW, RENEW), "")
})

test("恰好 30 天显示提示", () => {
  const text = expiryHintText("2026-12-16T00:00:00Z", NOW, RENEW)
  assert.match(text, /将于 2026-12-16 到期/)
  assert.ok(EXPIRY_WARNING_DAYS === 30)
})

test("剩余不足一天仍显示在期提示", () => {
  const text = expiryHintText("2026-11-16T12:00:00Z", NOW, RENEW)
  assert.match(text, /将于 2026-11-16 到期/)
})

test("已过期 1 分钟必须显示已到期", () => {
  assert.match(expiryHintText("2026-11-15T23:59:00Z", NOW, RENEW), /已到期/)
})

test("已过期 1 天显示已到期", () => {
  assert.match(expiryHintText("2026-11-15T00:00:00Z", NOW, RENEW), /已到期/)
})
