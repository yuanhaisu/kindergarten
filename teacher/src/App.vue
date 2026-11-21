<script setup>
import { onLaunch } from "@dcloudio/uni-app"
import { ensureConsent, ensureLogin, hasTeacherRole } from "@/utils/login"
import { getUser } from "@/utils/auth"

// 待绑定账号（有会话、无 teacher 角色）不能停留在工作台：业务接口会统一 40302。
// 登录页负责在持有邀请码时完成绑定。
function redirectToBindIfNeeded() {
  const user = getUser()
  if (user && !hasTeacherRole(user)) {
    uni.reLaunch({ url: "/pages/auth/login" })
    return true
  }
  return false
}

onLaunch(() => {
  const options = uni.getLaunchOptionsSync()
  if (options?.query?.invite_code) {
    uni.setStorageSync('invite_code', options.query.invite_code)
  }
  ensureLogin()
    .then(() => {
      if (redirectToBindIfNeeded()) return
      return ensureConsent()
    })
    .catch(() => {
      // 启动阶段静默登录失败时，业务页会继续引导用户授权；
      // 但已拿到待绑定会话却绑定失败时，必须回登录页才能重试。
      redirectToBindIfNeeded()
    })
})
</script>

<style lang="scss">
// CSS Custom Properties for runtime theme switching
@import "kindergarten-design-tokens/tokens-teacher.css";

// Remix Icon 字体（图标是 PUA 码位，缺此字体整页图标渲染成方块）
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
