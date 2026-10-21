import { requestWithAuth } from "@/api/http"

export function fetchGrowthFeed(params = {}) {
  return requestWithAuth({
    url: "/parent/growth/feed",
    method: "GET",
    data: params
  })
}

// studentId 决定服务端视角的「当前孩子」（记录可见性校验与徽章判定都依赖它）；
// 不传时服务端回退到第一个孩子——照片点名的「只看我的孩子」筛选需要它保持正确。
export function fetchGrowthDetail(id, studentId) {
  return requestWithAuth({
    url: `/parent/growth/feed/${id}`,
    method: "GET",
    data: studentId ? { student_id: studentId } : {}
  })
}

export function likeGrowth(id) {
  return requestWithAuth({
    url: `/parent/growth/feed/${id}/like`,
    method: "POST",
    data: {}
  })
}

export function unlikeGrowth(id) {
  return requestWithAuth({
    url: `/parent/growth/feed/${id}/like`,
    method: "DELETE"
  })
}

export function fetchGrowthComments(id, params = {}) {
  return requestWithAuth({
    url: `/parent/growth/feed/${id}/comments`,
    method: "GET",
    data: params
  })
}

export function postGrowthComment(id, data) {
  return requestWithAuth({
    url: `/parent/growth/feed/${id}/comments`,
    method: "POST",
    data
  })
}

export function fetchGrowthAlbum(params = {}) {
  return requestWithAuth({
    url: "/parent/growth/album",
    method: "GET",
    data: params
  })
}

export function fetchGrowthReport(params = {}) {
  return requestWithAuth({
    url: "/parent/growth/report",
    method: "GET",
    data: params
  })
}
