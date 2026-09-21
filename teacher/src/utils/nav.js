/**
 * 老师端的底部导航。
 *
 * 老师端**没有 tabBar**（见 src/pages.json，无 tabBar 字段），底栏是各页自己
 * 用 navigateTo 模拟的。那样每次点 tab 都会 push 一个新页面，栈长度无上限增长，
 * 超过微信的 10 层上限后导航会**静默失败** —— 点了没反应，也没有任何报错。
 *
 * goTab 取代它：
 * - 目标已在页面栈里 → navigateBack 回去（保留原有页面状态）
 * - 目标就是当前页   → 滚动置顶
 * - 都不满足        → redirectTo 替换栈顶，栈深恒为 1
 */
export const TEACHER_TABS = [
  "/pages/index/index",
  "/pages/class/index",
  "/pages/messages/index",
  "/pages/profile/index",
]

export function goTab(path) {
  const pages = getCurrentPages()
  const route = path.replace(/^\//, "")
  const idx = pages.findIndex((p) => p.route === route)

  if (idx >= 0) {
    const delta = pages.length - 1 - idx
    if (delta > 0) {
      uni.navigateBack({ delta })
      return
    }
    uni.pageScrollTo({ scrollTop: 0, duration: 300 })
    return
  }

  uni.redirectTo({ url: path })
}
