import { requestWithAuth, uploadFile } from "@/api/http"

// options 透传给 requestWithAuth（如 { silent: true }）：相册页自己渲染整页错误态，
// 不再叠加全局 toast。其余调用方不传时行为不变。
export function fetchTeacherAlbumStats(options = {}) {
  return requestWithAuth({ url: "/teacher/albums/stats", method: "GET", ...options })
}

export function fetchTeacherAlbums(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/albums", method: "GET", data: params, ...options })
}

export function fetchTeacherAlbum(id, options = {}) {
  return requestWithAuth({ url: `/teacher/albums/${id}`, method: "GET", ...options })
}

export function createTeacherAlbum(data, options = {}) {
  return requestWithAuth({ url: "/teacher/albums", method: "POST", data, silent: options.silent })
}

// 相册照片先走统一上传通道（素材管理基线：biz_type=album），
// 再把返回的 file_url 关联到 POST /teacher/albums。
export function uploadAlbumPhoto(filePath, onProgressUpdate = null) {
  return uploadFile({
    url: "/common/upload",
    filePath,
    formData: { biz_type: "album" },
    silent: true,
    onProgressUpdate
  })
}
