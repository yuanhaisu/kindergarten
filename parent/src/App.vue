<script>
// getApp().globalData 需要先声明；字体加载状态写入这里，便于排查「字体没生效」。
export default {
  globalData: {
    serifFont: "",
    latinSerifFont: ""
  }
}
</script>

<script setup>
import { onLaunch, onShow } from "@dcloudio/uni-app"
import { ensureConsent, ensureLogin } from "@/utils/login"
import { captureEntryScene } from "@/utils/entryScene.mjs"
import serifFontBase64 from "@/assets/fonts/noto-serif-sc-subset.js"
import latinSerifFontBase64 from "@/assets/fonts/instrument-serif-subset.js"

// DNA 字体：中文标题 Noto Serif SC，拉丁/数字 Instrument Serif，均以 data URI 注入。
// 小程序不允许读包内文件、loadFontFace 也不接受本地路径，只能内联 base64。
const DNA_FONTS = [
  { family: "Noto Serif SC", base64: serifFontBase64, statusKey: "serifFont" },
  { family: "Instrument Serif", base64: latinSerifFontBase64, statusKey: "latinSerifFont" }
]

function loadDnaFonts() {
  // #ifdef MP-WEIXIN
  DNA_FONTS.forEach(({ family, base64, statusKey }) => {
    try {
      wx.loadFontFace({
        global: true,
        family,
        source: `url("data:font/woff2;base64,${base64.trim()}")`,
        success: () => {
          const app = getApp()
          if (app) app.globalData[statusKey] = "loaded"
        },
        fail: (err) => {
          const app = getApp()
          if (app) app.globalData[statusKey] = "failed"
          console.warn(`[font] ${family} load failed:`, err?.errMsg || err)
        }
      })
    } catch (e) {
      console.warn(`[font] ${family} load failed:`, e?.message || e)
    }
  })
  // #endif
}

// 园所入口（#176）：scene 只在启动/唤醒时可见，必须早于登录写进暂存。
// 冷启动走 onLaunch(options)，热启动（后台被扫码唤起）只走 onShow(options)；
// 两处都在 ensureLogin 之前执行，登录/绑定流程跑完时入口仍可读。
function captureEntry(options) {
  const code = captureEntryScene(options, uni)
  if (code) console.info("[entry] pending kindergarten:", code)
}

onLaunch((options) => {
  // 部分基础库在 onLaunch 不给 options，回退 getLaunchOptionsSync（同老师端 invite_code）。
  captureEntry(options || uni.getLaunchOptionsSync())
  loadDnaFonts()
  ensureLogin()
    .then(() => ensureConsent())
    .catch(() => {
      // 启动阶段静默登录失败时，业务页会继续引导用户授权。
    })
})

onShow((options) => {
  captureEntry(options)
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
