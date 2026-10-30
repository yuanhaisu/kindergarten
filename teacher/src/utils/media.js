/**
 * 选图统一入口（相册上传与成长记录共用）。
 *
 * 用户取消不提示；其它失败给一次可感知的 toast，然后统一返回空数组，
 * 调用方不需要区分「取消」和「失败」。返回 [{ path, size }]，size 供
 * 各页做 20MB 预检。
 */
export function pickImages({ count = 1 } = {}) {
  return new Promise(resolve => {
    uni.chooseImage({
      count,
      sizeType: ["compressed", "original"],
      sourceType: ["album", "camera"],
      success(res) {
        const paths = res.tempFilePaths || []
        const files = res.tempFiles || []
        resolve(paths.map((path, index) => ({ path, size: Number(files[index]?.size || 0) })))
      },
      fail(err) {
        const msg = String(err?.errMsg || "")
        if (!/cancel/i.test(msg)) uni.showToast({ title: "选择照片失败", icon: "none" })
        resolve([])
      }
    })
  })
}
