// 园所入口 scene 的捕获、暂存与落园决策（#176）。
//
// 关键时序：小程序码的 scene 只在「冷启动」经 App.onLaunch 的
// getLaunchOptionsSync/options 可见；「热启动」（已在后台被扫码唤起）走
// App.onShow 的 options。两处都必须在登录之前、独立于登录写入暂存，
// 否则登录/绑定流程跑完时入口已经丢了。
//
// storage 由调用方注入（小程序里是 uni），纯逻辑便于 node --test 覆盖。
import { isPublicCode, parseScenePublicCode } from "./scene.mjs"

export const PENDING_ENTRY_KEY = "parent_pending_entry"

// 入口码 7 天有效：足够「今天扫码、明天登录」，又不会让一个久远的码
// 在换设备后突然把会话切走。
export const PENDING_ENTRY_TTL_MS = 7 * 24 * 60 * 60 * 1000

// 与后端公开码字母表一致（Crockford base32，去掉 i/l/o/u）；校验统一走 scene.mjs。
function normalizePublicCode(value) {
  const code = String(value || "").trim().toLowerCase()
  return isPublicCode(code) ? code : ""
}

function writePending(storage, code, now) {
  try {
    storage.setStorageSync(PENDING_ENTRY_KEY, JSON.stringify({ public_code: code, captured_at: now }))
  } catch (err) {
    // 存储不可用只影响跨启动保留；本次启动仍可按返回值继续处理入口。
    console.warn("[entry] persist pending kindergarten failed:", err?.message || err)
  }
}

// 从一次启动/唤醒的 options 里提取园所入口并暂存。
//
// 只认 query.scene 与 query.public_code：options.scene 是微信的启动场景常量
// （如 1047 = 扫小程序码），不是业务 payload，绝不能当园所码解析。
// 返回识别到的公开码（没有则返回空串），便于调用方打日志。
export function captureEntryScene(options, storage, now = Date.now()) {
  const query = options && options.query ? options.query : {}
  const code = parseScenePublicCode(query.scene) || normalizePublicCode(query.public_code)
  if (!code) return ""
  if (storage) writePending(storage, code, now)
  return code
}

// 读取待处理入口；过期或损坏时清除并返回 null，避免坏数据反复触发入口流程。
export function readPendingEntry(storage, now = Date.now()) {
  if (!storage) return null
  let raw = ""
  try {
    raw = storage.getStorageSync(PENDING_ENTRY_KEY)
  } catch (err) {
    console.warn("[entry] read pending kindergarten failed:", err?.message || err)
    return null
  }
  if (!raw) return null

  let parsed = null
  try {
    parsed = typeof raw === "string" ? JSON.parse(raw) : raw
  } catch (err) {
    // 坏数据按「无入口」处理并清除；留日志便于排查写入方。
    console.warn("[entry] pending kindergarten is corrupted:", err?.message || err)
    parsed = null
  }

  const code = normalizePublicCode(parsed && parsed.public_code)
  const capturedAt = Number(parsed && parsed.captured_at)
  if (!code || !Number.isFinite(capturedAt) || now - capturedAt > PENDING_ENTRY_TTL_MS) {
    clearPendingEntry(storage)
    return null
  }
  return { public_code: code, captured_at: capturedAt }
}

export function clearPendingEntry(storage) {
  if (!storage) return
  try {
    storage.removeStorageSync(PENDING_ENTRY_KEY)
  } catch (err) {
    console.warn("[entry] clear pending kindergarten failed:", err?.message || err)
  }
}

// 落园决策：只依据「我的园所」成员列表，绝不枚举其它园所。
// - 目标园在列表且不是当前园 → 自动切换（#169）
// - 目标园就是当前园 → 无需动作
// - 不在列表 → 绑定引导（携带公开码，绑定即落到该园，#167）
export function resolveEntryAction({ publicCode, memberships, currentKindergartenID } = {}) {
  const code = normalizePublicCode(publicCode)
  if (!code) return { action: "none" }

  const list = Array.isArray(memberships) ? memberships : []
  const match = list.find((item) => normalizePublicCode(item && item.public_code) === code)
  if (!match) return { action: "bind", publicCode: code }

  const kindergartenID = Number(match.kindergarten_id) || 0
  if (!kindergartenID) return { action: "bind", publicCode: code }
  if (kindergartenID === (Number(currentKindergartenID) || 0)) {
    return { action: "none", kindergartenID, name: match.name || "" }
  }
  return { action: "switch", kindergartenID, name: match.name || "" }
}
