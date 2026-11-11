<script setup>
import { onLaunch } from "@dcloudio/uni-app"
import { ensureConsent, ensureLogin } from "@/utils/login"
import serifFontBase64 from "@/assets/fonts/noto-serif-sc-subset.js"

// DNA 字体：Noto Serif SC 子集（仅静态标题字），以 data URI 注入。
// 小程序不允许读包内文件、loadFontFace 也不接受本地路径，只能内联 base64。
function loadSerifFont() {
  // #ifdef MP-WEIXIN
  try {
    wx.loadFontFace({
      global: true,
      family: "Noto Serif SC",
      source: `url("data:font/woff2;base64,${serifFontBase64.trim()}")`,
      success: () => {
        const app = getApp()
        if (app) app.globalData.serifFont = "loaded"
      },
      fail: (err) => {
        const app = getApp()
        if (app) app.globalData.serifFont = "failed"
        console.warn("[font] Noto Serif SC load failed:", err?.errMsg || err)
      }
    })
  } catch (e) {
    console.warn("[font] read subset failed:", e?.message || e)
  }
  // #endif
}

onLaunch(() => {
  loadSerifFont()
  ensureLogin()
    .then(() => ensureConsent())
    .catch(() => {
      // 启动阶段静默登录失败时，业务页会继续引导用户授权。
    })
})
</script>

<style lang="scss">
// CSS Custom Properties for runtime theme switching
@import "kindergarten-design-tokens/tokens-parent.css";

// Remix Icon 字体（在 App.vue 中 @ 别名可被 Vite 正确解析）
@import "@/static/fonts/remixicon.css";

page {
  min-height: 100%;
  background: $color-bg-page;
  color: $color-text-primary;
  font-family: -apple-system, BlinkMacSystemFont, "PingFang SC", "Microsoft YaHei", sans-serif;
}

view,
text,
button,
input,
textarea {
  box-sizing: border-box;
}

button {
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  line-height: 1.4;
}

button::after {
  border: 0;
}
</style>
