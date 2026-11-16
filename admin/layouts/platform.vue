<template>
  <div class="min-h-screen flex bg-bg-page">
    <aside class="w-56 bg-brand-sidebar flex flex-col">
      <div class="px-5 py-5 text-white">
        <div class="text-base font-semibold">养成星球</div>
        <div class="text-xs text-white/60">平台运营</div>
      </div>
      <nav class="flex-1 px-2 space-y-1">
        <NuxtLink
          v-for="item in menus"
          :key="item.path"
          :to="item.path"
          class="block px-3 py-2 rounded-md text-sm transition-colors"
          :class="isActive(item.path) ? 'bg-brand-sidebarHover text-white font-medium' : 'text-white/80 hover:bg-brand-sidebarHover hover:text-white'"
        >
          {{ item.title }}
        </NuxtLink>
      </nav>
      <button
        class="m-3 px-3 py-2 rounded-md text-sm text-white/80 hover:text-white bg-white/10 hover:bg-white/20 transition-colors"
        @click="logout"
      >
        退出登录
      </button>
    </aside>
    <main class="flex-1 overflow-auto p-6">
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { platformLogout } from '~/api/platform'
import { PLATFORM_TOKEN_COOKIE, buildAuthCookieOptions } from '~/utils/authCookie'

const menus = [
  { title: '园所', path: '/platform/kindergartens' },
  { title: '套餐', path: '/platform/plans' },
  { title: '订单', path: '/platform/orders' },
  { title: '分销商', path: '/platform/distributors' },
  { title: '佣金台账', path: '/platform/commissions' },
]

const route = useRoute()
function isActive(path: string) {
  return route.path.startsWith(path)
}

async function logout() {
  try {
    await platformLogout()
  } catch {
    // 网络失败也要清本地登录态
  }
  useCookie<string | null>(PLATFORM_TOKEN_COOKIE, buildAuthCookieOptions()).value = null
  await navigateTo('/platform/login')
}
</script>
