// 原生 custom-tab-bar 的选中态只能由页面主动回写：切 tab 时组件实例不重建，
// 组件内 attached / pageLifetimes.show 拿到的栈顶路由可能还是旧页面，高亮会停在首页。
// uni-app 的页面由 Component 包装，原生页实例挂在 Vue 实例的 $scope 上；getTabBar 是原生页方法。
export function syncTabBar(selected) {
  const pages = getCurrentPages()
  const page = pages[pages.length - 1]
  if (!page) return

  const scope = page.$vm?.$scope
  const tabBar =
    (typeof scope?.getTabBar === "function" && scope.getTabBar()) ||
    (typeof page.getTabBar === "function" && page.getTabBar()) ||
    null
  // 拿不到组件实例时静默跳过：高亮偏差不该阻断页面本身的加载
  if (tabBar) tabBar.setData({ selected })
}
