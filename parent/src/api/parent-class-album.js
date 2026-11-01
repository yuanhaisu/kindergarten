import { requestWithAuth } from "@/api/http"

/**
 * 班级相册（backend/api/api-parent/13-class-albums.md）。
 *
 * 服务端只返回已发布相册，并按「当前孩子 + 班内共享」算好
 * visible_photo_count / cover_urls / photos；端上不做二次过滤。
 */

// scope 缺省即 mine；all 只在园所开启全班视图时由服务端放行（否则 403）。
const SCOPES = new Set(["mine", "all"])

function withScope(data, scope) {
  if (SCOPES.has(scope)) data.scope = scope
  return data
}

// student_id 省略时服务端取该家长的第一个孩子；多孩家庭由调用方显式传入。
export function fetchParentClassAlbums(params = {}) {
  const data = { page: params.page, page_size: params.page_size }
  if (params.student_id) data.student_id = params.student_id
  withScope(data, params.scope)
  return requestWithAuth({
    url: "/parent/class-albums",
    method: "GET",
    data
  })
}

// 草稿/下架/跨班/留存过期统一 404（服务端不区分「无权」与「不存在」）。
export function fetchParentClassAlbum(id, params = {}) {
  const data = {}
  if (params.student_id) data.student_id = params.student_id
  withScope(data, params.scope)
  return requestWithAuth({
    url: `/parent/class-albums/${id}`,
    method: "GET",
    data
  })
}
