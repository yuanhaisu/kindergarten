import type { KindergartenInfo } from '~/types/api'

export function getKindergartenInfo() {
  return useHttp().get<KindergartenInfo>('/admin/kindergarten/info')
}

export function updateKindergartenInfo(body: Partial<KindergartenInfo>) {
  return useHttp().put<{ id: number; updated_at: string }>('/admin/kindergarten/info', body)
}

/** 家长端园所入口二维码（#205），契约见 backend/api/api-admin/18-settings.md */
export interface ParentEntryQRCode {
  public_code: string
  scene: string
  page: string
  env_version: string
  /** PNG 的 base64（不含 data URL 前缀） */
  image_base64: string
}

/**
 * 生成当前园所的家长入口二维码。目标园由登录态决定，不接受 kindergarten_id。
 * envVersion 省略时回落到服务端配置，避免前端猜错线上/体验版。
 */
export function getParentEntryQRCode(envVersion?: string) {
  return useHttp().get<ParentEntryQRCode>('/admin/kindergarten/parent-qrcode', {
    params: envVersion ? { env_version: envVersion } : undefined,
  })
}
