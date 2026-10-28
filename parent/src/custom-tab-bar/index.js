// 微信原生自定义 tabBar：本目录被 @dcloudio/uni-mp-weixin 原样复制到 dist，
// 不经过 uni/Vue 编译，因此这里只能用原生 Component + wx.* API。
// 图标码位与 shared-ui BottomNav 一致（remixicon U+EE1C / U+EDA5 / U+EC79 / U+F264）。
const TABS = [
  { pagePath: "pages/index/index", text: "首页", icon: "" },
  { pagePath: "pages/growth/index", text: "成长", icon: "" },
  { pagePath: "pages/pickup/index", text: "接送", icon: "" },
  { pagePath: "pages/profile/index", text: "我的", icon: "" }
]

// 每个 tab 页各持有一个 tabBar 组件实例，隐藏页的实例仍在内存里；
// 只改当前页那一份，其它页会留着上一次的高亮，切回去就错位（首页没有 onShow 同步，最容易中招）。
// 模块级变量在小程序里是单例，点击时把所有已创建的实例一起改。
const TAB_BARS = new Set()

Component({
  // 让 app.wxss 里的字体/基础样式可以作用到本组件（原生 tabBar 默认样式隔离）
  options: {
    styleIsolation: "apply-shared"
  },

  data: {
    selected: 0,
    list: TABS
  },

  lifetimes: {
    attached() {
      TAB_BARS.add(this)
    },
    detached() {
      TAB_BARS.delete(this)
    }
  },

  methods: {
    onTabTap(e) {
      const index = Number(e.currentTarget.dataset.index)
      if (!Number.isInteger(index) || index === this.data.selected) return

      const tab = TABS[index]
      if (!tab) return

      // 先点亮再切换：switchTab 后当前页立即隐藏，等不到目标页 onShow 回写。
      TAB_BARS.forEach((bar) => bar.setData({ selected: index }))
      wx.switchTab({ url: "/" + tab.pagePath })
    }
  }
})
