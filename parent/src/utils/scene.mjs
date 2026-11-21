// 小程序码 scene 的统一解析（#176）。
//
// scene 形如 `t=123`（任务分享码）或 `kg_<public_code>`（园所入口码），
// 可能被 URL 编码成 `t%3D123` / `kg%5Fabc...`（或整串被编码一次）。
// 首页、任务详情与 App 启动捕获必须共用这一份：各写一份，编码差异迟早漂移。
//
// 纯函数、无 uni 依赖，便于 node --test 直接覆盖冷/热启动拿到的各种形状。

// decodeURIComponent 遇到非法编码会抛错；按原样解析，不能因此吞掉整个入口。
export function decodeScene(scene) {
  if (!scene) return ""
  const raw = String(scene)
  try {
    return decodeURIComponent(raw)
  } catch (e) {
    return raw
  }
}

// 公开码字母表与后端 GeneratePublicCode 一致（Crockford base32，去掉 i/l/o/u）。
const PUBLIC_CODE_RE = /^[0-9abcdefghjkmnpqrstvwxyz]{1,24}$/

// 单独导出校验：入口暂存/决策也要用同一份字母表，避免两处正则各自漂移。
export function isPublicCode(value) {
  return PUBLIC_CODE_RE.test(String(value || ""))
}

// 园所入口码：`kg_<public_code>`。与任务码共存于同一 scene 时各取各的，
// 例如 `kg_abc123defg&t=12` 同时得到园所码与任务 id。
export function parseScenePublicCode(scene) {
  const raw = decodeScene(scene)
  if (!raw) return ""
  const m = raw.match(/(?:^|&)kg_([0-9a-z]+)/)
  if (!m) return ""
  return isPublicCode(m[1]) ? m[1] : ""
}

// 任务分享码：`t=<digits>`，可带其它键。保持 #163 起的既有行为。
export function parseSceneTaskId(scene) {
  const raw = decodeScene(scene)
  if (!raw) return 0
  const m = raw.match(/(?:^|&)t=(\d+)/)
  return m ? Number(m[1]) : 0
}
