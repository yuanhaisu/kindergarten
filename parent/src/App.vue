<script setup>
import { onLaunch } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"

onLaunch(() => {
  ensureLogin().catch(() => {
    // 启动阶段静默登录失败时，业务页会继续引导用户授权。
  })
})
</script>

<style lang="scss">
// CSS Custom Properties for runtime theme switching
@import "kindergarten-design-tokens/tokens-parent.css";

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
