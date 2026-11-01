import { requestWithAuth, uploadFile } from "@/api/http"

// options 透传给 requestWithAuth（如 { silent: true }）：相册页自己渲染整页错误态，
// 不再叠加全局 toast。其余调用方不传时行为不变。
export function fetchTeacherAlbumStats(options = {}) {
  return requestWithAuth({ url: "/teacher/albums/stats", method: "GET", ...options })
}

// params.status: draft / published / unpublished；不传返回全部状态（老师端管理视图）。
export function fetchTeacherAlbums(params = {}, options = {}) {
  return requestWithAuth({ url: "/teacher/albums", method: "GET", data: params, ...options })
}

export function fetchTeacherAlbum(id, options = {}) {
  return requestWithAuth({ url: `/teacher/albums/${id}`, method: "GET", ...options })
}

export function createTeacherAlbum(data, options = {}) {
  return requestWithAuth({ url: "/teacher/albums", method: "POST", data, silent: options.silent })
}

// 编辑：请求体与创建一致，照片/参与者都是全量替换语义（见 10-album.md）。
export function updateTeacherAlbum(id, data, options = {}) {
  return requestWithAuth({
    url: `/teacher/albums/${id}`,
    method: "PUT",
    data,
    silent: options.silent
  })
}

// 发布草稿或重新上架已下架相册；已发布再次发布会返回 422。
export function publishTeacherAlbum(id, options = {}) {
  return requestWithAuth({
    url: `/teacher/albums/${id}/publish`,
    method: "POST",
    silent: options.silent
  })
}

export function unpublishTeacherAlbum(id, options = {}) {
  return requestWithAuth({
    url: `/teacher/albums/${id}/unpublish`,
    method: "POST",
    silent: options.silent
  })
}

// 软删除：相册从老师端与家长端消失，行保留 180 天供审计。
export function deleteTeacherAlbum(id, options = {}) {
  return requestWithAuth({
    url: `/teacher/albums/${id}`,
    method: "DELETE",
    silent: options.silent
  })
}

// 相册照片先走统一上传通道（素材管理基线：biz_type=album），
// 上传响应里的 id 就是相册契约的 photos[].file_id（不是 URL、不是其它 ID）。
export function uploadAlbumPhoto(filePath, onProgressUpdate = null) {
  return uploadFile({
    url: "/common/upload",
    filePath,
    formData: { biz_type: "album" },
    silent: true,
    onProgressUpdate
  })
}
