import { requestWithAuth } from "@/api/http"

export function fetchGrowthFeed(params = {}) {
  return requestWithAuth({
    url: "/parent/growth/feed",
    method: "GET",
    data: params
  })
}

export function fetchGrowthDetail(id) {
  return requestWithAuth({
    url: `/parent/growth/feed/${id}`,
    method: "GET"
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
