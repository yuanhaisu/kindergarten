/** 浏览器下载 Blob */
export function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

/** 从 Content-Disposition 解析文件名 */
export function parseFilenameFromDisposition(cd: string | undefined, fallback: string) {
  if (!cd) return fallback
  const m = /filename\*?=(?:UTF-8'')?["']?([^"';]+)/i.exec(cd)
  return m?.[1] ? decodeURIComponent(m[1]) : fallback
}
