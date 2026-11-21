// 老师邀请码的启动参数解析（#168 / #160「冷/热启动 scene」）。
//
// 邀请小程序码的 scene 是 `invite_code=<token>`，微信通常整串 URL 编码成
// `invite_code%3D<token>`。scene 只在启动/唤醒时可见，因此必须在 ensureLogin()
// 之前写进暂存——登录流程会用同一份 storage 完成绑定（见 utils/login.js）。
//
// 纯函数、storage 注入，便于 node --test 覆盖冷/热启动与编码差异。
export const INVITE_CODE_KEY = "invite_code"

// decodeURIComponent 遇到非法编码会抛错；按原样解析，不能让一个坏 scene 吞掉整个启动。
export function decodeScene(scene) {
  if (!scene) return ""
  const raw = String(scene)
  try {
    return decodeURIComponent(raw)
  } catch (e) {
    return raw
  }
}

// 当前后端邀请码是 16 位 hex（generateInviteCode），解析这里放宽到 URL 安全字符，
// 换格式时不至于整条邀请链路失联；长度上限同时挡住把整段异常 scene 写进 storage。
const INVITE_CODE_RE = /^[0-9A-Za-z_-]{4,64}$/

function normalizeInviteCode(value) {
  const code = String(value || "").trim()
  return INVITE_CODE_RE.test(code) ? code : ""
}

// 从 scene 中取 `invite_code=<token>`，允许与其它键共存（如 `&t=1`）。
export function parseInviteCodeFromScene(scene) {
  const raw = decodeScene(scene)
  if (!raw) return ""
  const m = raw.match(/(?:^|&)invite_code=([^&#]+)/)
  if (!m) return ""
  let code = m[1]
  try {
    code = decodeURIComponent(code)
  } catch (e) {
    // 参数值单独编码异常时保留原值，交给下面的字母表校验。
  }
  return normalizeInviteCode(code)
}

// 从一次启动/唤醒的 options 里提取邀请码并暂存，返回识别到的邀请码（没有则空串）。
//
// 只认 query.scene 与 query.invite_code：options.scene 是微信的启动场景常量
// （如 1047 = 扫小程序码），不是业务 payload，绝不能当邀请码解析。
export function captureInviteCode(options, storage) {
  const query = options && options.query ? options.query : {}
  const code = parseInviteCodeFromScene(query.scene) || normalizeInviteCode(query.invite_code)
  if (!code) return ""
  if (storage) {
    try {
      storage.setStorageSync(INVITE_CODE_KEY, code)
    } catch (err) {
      // 存储不可用只影响跨启动保留；本次启动仍可按返回值继续处理入口。
      console.warn("[invite] persist invite code failed:", err?.message || err)
    }
  }
  return code
}
