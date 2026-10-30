// 绑定孩子的失败分档。
//
// 后端三条失败语义（server/internal/handler/http/parent_portal.go#ParentBindStudent）：
//   404 / 40400 未找到该学生   → 让用户改码重填才有意义
//   422 / 42200 已绑定或邀请码无效 → 清空输入并不能解决问题，不给「重新填写」
//   其他        绑定学生失败   → 保留输入，可直接重试
//
// http.js 已把响应体（code/message）原样 reject，这里只做分档，不改文案来源：
// 服务端文案是唯一真相，缺失时才用兜底。

const NOT_FOUND = "未找到该学生，请检查邀请码"
const ALREADY_BOUND = "已绑定或邀请码无效"
const INTERNAL = "绑定学生失败"

function errorCode(err) {
  return Number(err?.code)
}

export function toBindError(err) {
  const code = errorCode(err)
  const isNotFound = code === 40400 || code === 404
  const isAlreadyBound = code === 42200 || code === 422
  const fallback = isNotFound ? NOT_FOUND : isAlreadyBound ? ALREADY_BOUND : INTERNAL
  const message = String(err?.message || "").trim() || fallback

  return {
    // server 表示请求已到后端且被明确拒绝；客户端校验不经过这里。
    kind: "server",
    message,
    // 只有「查无此码」值得让用户清空重填；已绑定/服务异常清空无用。
    refill: isNotFound
  }
}
