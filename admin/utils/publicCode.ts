/**
 * 园所家长入口码（`public_code`）的展示与复制。
 *
 * 后端在开通响应/列表/详情都返回 10 位 Crockford base32 小写码；旧 UI 因为
 * 类型未声明而全部丢弃（#204）。复制时必须先检查 `navigator.clipboard`：
 * HTTP staging 不是安全上下文，clipboard 为 undefined，直接调用 `writeText`
 * 会同步抛 TypeError（旧教师页正是这个写法），因此统一降级为「手动选中」。
 */

export type CopyOutcome = 'copied' | 'manual'

/** 归一化入口码：非字符串或纯空白都视为未生成。 */
export function normalizePublicCode(code?: string | null): string {
  return typeof code === 'string' ? code.trim() : ''
}

/** 展示用文案：空码统一显示「未生成」，避免模板直接渲染 undefined。 */
export function publicCodeText(code?: string | null): string {
  return normalizePublicCode(code) || '未生成'
}

export function hasPublicCode(code?: string | null): boolean {
  return normalizePublicCode(code) !== ''
}

/**
 * 复制入口码。
 *
 * 返回 `copied` 表示已写入剪贴板；返回 `manual` 表示不可用（无 clipboard API、
 * 写入被拒或码为空），调用方应提示用户手动选中码文本。
 */
export async function copyPublicCode(code?: string | null): Promise<CopyOutcome> {
  const text = normalizePublicCode(code)
  if (!text) return 'manual'

  const clipboard = typeof navigator === 'undefined' ? undefined : navigator.clipboard
  if (!clipboard || typeof clipboard.writeText !== 'function') return 'manual'

  try {
    await clipboard.writeText(text)
    return 'copied'
  } catch {
    return 'manual'
  }
}
