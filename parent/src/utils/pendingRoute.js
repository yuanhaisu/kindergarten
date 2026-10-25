// 未登录 / 登录态失效时保留用户原本要去的站内路径，登录成功后回跳。
//
// 只保留小程序内部 `/pages/` 路径：消息的 action_path 与页面参数都可能来自服务端，
// 不能让一个外部 URL 借「回跳」被打开。

const KEY = "pending_route"

// 家长端 tabBar 四页（pages.json → tabBar.list）。这些路径只能用 switchTab。
const TAB_PAGES = [
  "/pages/index/index",
  "/pages/growth/index",
  "/pages/pickup/index",
  "/pages/profile/index"
]

function isInternalPath(path) {
  return typeof path === "string" &&
    path.indexOf("/pages/") === 0 &&
    path.indexOf("://") === -1
}

export function rememberPendingRoute(path) {
  if (!isInternalPath(path)) return
  try {
    uni.setStorageSync(KEY, path)
  } catch (e) {
    // 存储不可用只影响回跳，不影响登录本身
  }
}

export function takePendingRoute() {
  try {
    const path = uni.getStorageSync(KEY)
    uni.removeStorageSync(KEY)
    return isInternalPath(path) ? path : ""
  } catch (e) {
    return ""
  }
}

// 登录成功 / 已登录时调用：有保留路径就回跳并返回 true；
// 返回 false 表示调用方应走默认首页。
export function resumePendingRoute() {
  const path = takePendingRoute()
  if (!path) return false
  const pathname = path.split("?")[0]
  if (TAB_PAGES.indexOf(pathname) !== -1) {
    uni.switchTab({ url: pathname })
    return true
  }
  // reLaunch 而不是 navigateTo：把登录页从栈里移除，回跳后返回键不会回到登录页。
  uni.reLaunch({ url: path })
  return true
}
