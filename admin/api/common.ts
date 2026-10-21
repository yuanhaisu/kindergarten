import type { AuthUser, UploadedFile } from '~/types/api'

/** 当前用户资料（三端通用） */
export function getUserProfile() {
  return useHttp().get<AuthUser & { created_at?: string }>('/common/user/profile')
}

/**
 * 统一上传通道（多端共用）。
 * 返回的 `id` 即业务的 file_id 引用（如人脸底库提交 /admin/students/:id/faces 的 file_ids）。
 */
export function uploadFile(file: File, bizType: string) {
  const fd = new FormData()
  fd.append('file', file)
  fd.append('biz_type', bizType)
  return useHttp()
    .raw.post<UploadedFile>('/common/upload', fd)
    .then((r) => r.data as UploadedFile)
}
