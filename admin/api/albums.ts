import type { PageParams, PageResult } from '~/types/api'

/** 与后端 domain.AlbumStatus 同值（design D3 状态机） */
export type AlbumStatus = 'draft' | 'published' | 'unpublished'
/** 与后端 domain.AlbumCategory 同值；中文标签由服务端 category_label 给出 */
export type AlbumCategory = 'activity' | 'daily' | 'work' | 'festival'

export interface AlbumCreator {
  id: number
  nickname: string
}

export interface AlbumStudentBrief {
  id: number
  name: string
  avatar: string
}

export interface AlbumListItem {
  id: number
  class_id: number
  class_name: string
  title: string
  category: AlbumCategory
  category_label: string
  album_date: string
  photo_count: number
  status: AlbumStatus
  published_at: string | null
  creator: AlbumCreator
}

export interface AlbumPhoto {
  id: number
  file_id: number | null
  file_url: string
  thumb_url: string
  width: number
  height: number
  sort_num: number
  students: AlbumStudentBrief[]
}

export interface AlbumDetail extends AlbumListItem {
  photos: AlbumPhoto[]
  participants: AlbumStudentBrief[]
}

export interface AlbumListParams extends PageParams {
  class_id?: number
  status?: AlbumStatus
  category?: AlbumCategory
}

export interface AlbumUnpublishResult {
  id: number
  status: AlbumStatus
  /** false = 调用前已是 unpublished（幂等成功，不重复写审计） */
  changed: boolean
}

export function getAlbums(params?: AlbumListParams) {
  return useHttp().get<PageResult<AlbumListItem>>('/admin/albums', { params })
}

export function getAlbumDetail(id: number) {
  return useHttp().get<AlbumDetail>(`/admin/albums/${id}`)
}

export function unpublishAlbum(id: number) {
  return useHttp().post<AlbumUnpublishResult>(`/admin/albums/${id}/unpublish`)
}
