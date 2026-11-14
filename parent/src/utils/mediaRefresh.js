/**
 * 媒体 URL 是 1 小时签名地址；图片加载失败通常是 URL 过期。
 * 返回一个带冷却的 error 处理器：重新拉取当前页数据，拿到新签名 URL。
 * 冷却避免真实 404 时无限重试。
 */
export function createMediaErrorHandler(reload, cooldownMs = 3000) {
  let refreshing = false
  return async function onMediaError() {
    if (refreshing) return
    refreshing = true
    try {
      await reload()
    } catch (err) {
      console.warn("[media] reload after image error failed", err)
    } finally {
      setTimeout(() => {
        refreshing = false
      }, cooldownMs)
    }
  }
}
