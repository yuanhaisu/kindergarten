// 小程序码 scene 形如 `t=123`，可能被 URL 编码成 `t%3D123`（或整串被编码一次）。
// 首页与任务详情必须共用同一份解析：两处各写一份，编码差异迟早漂移。
export function parseSceneTaskId(scene) {
  if (!scene) return 0
  let raw = String(scene)
  try {
    raw = decodeURIComponent(raw)
  } catch (e) {
    // 非法编码就按原样解析，不要因此吞掉整个入口
  }
  const m = raw.match(/(?:^|&)t=(\d+)/)
  return m ? Number(m[1]) : 0
}
