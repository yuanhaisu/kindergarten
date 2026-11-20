<template>
  <div class="min-h-screen bg-platform-bg">
    <!-- 移动端遮罩：侧栏以抽屉形式出现，关闭后不占用布局宽度 -->
    <div
      v-if="mobileOpen"
      class="fixed inset-0 z-40 bg-platform-overlay lg:hidden"
      @click="mobileOpen = false"
    />

    <aside
      class="fixed inset-y-0 left-0 z-50 flex w-sidebar flex-col border-r border-platform-border bg-platform-card transition-[transform,width] duration-200"
      :class="[
        mobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0',
        collapsed ? 'lg:w-sidebarCollapsed' : 'lg:w-sidebar',
      ]"
    >
      <div
        class="flex h-16 shrink-0 items-center border-b border-platform-border px-5"
        :class="collapsed ? 'lg:justify-center lg:px-0' : ''"
      >
        <img src="/brand-badge.png" alt="" class="h-9 w-9 shrink-0 rounded-full">
        <div v-if="!collapsed" class="ml-3 min-w-0">
          <div class="truncate text-sm font-semibold text-platform-textPrimary">养成星球</div>
          <div class="truncate text-xs text-platform-textMuted">平台运营</div>
        </div>
        <button
          class="ml-auto flex h-8 w-8 items-center justify-center rounded-md text-platform-textMuted hover:bg-platform-bg hover:text-platform-textPrimary lg:hidden"
          aria-label="关闭导航"
          @click="mobileOpen = false"
        >
          <el-icon :size="18"><Close /></el-icon>
        </button>
      </div>

      <nav class="flex-1 overflow-y-auto px-3 py-4">
        <div v-if="!collapsed" class="px-3 pb-2 text-xs font-medium text-platform-textMuted">运营台</div>
        <NuxtLink
          v-for="item in menus"
          :key="item.path"
          :to="item.path"
          :title="collapsed ? item.title : ''"
          class="mb-1 flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium transition-colors"
          :class="[
            collapsed ? 'lg:justify-center lg:px-0' : '',
            isActive(item.path)
              ? 'bg-platform-brandBg text-platform-brandDark'
              : 'text-platform-textMuted hover:bg-platform-bg hover:text-platform-textPrimary',
          ]"
        >
          <el-icon :size="18" class="shrink-0">
            <component :is="item.icon" />
          </el-icon>
          <span v-if="!collapsed" class="truncate">{{ item.title }}</span>
        </NuxtLink>
      </nav>

      <div class="border-t border-platform-border p-3">
        <div
          class="flex items-center gap-3 rounded-md px-2 py-2"
          :class="collapsed ? 'lg:justify-center lg:px-0' : ''"
        >
          <div
            class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-platform-brandBg text-sm font-semibold text-platform-brand"
          >
            {{ initial }}
          </div>
          <div v-if="!collapsed" class="min-w-0">
            <div class="truncate text-sm font-medium text-platform-textPrimary">{{ displayName }}</div>
            <div class="text-xs text-platform-textMuted">平台账号</div>
          </div>
        </div>
      </div>
    </aside>

    <div
      class="flex min-h-screen flex-col transition-[padding] duration-200"
      :class="collapsed ? 'lg:pl-sidebarCollapsed' : 'lg:pl-sidebar'"
    >
      <header
        class="sticky top-0 z-30 flex h-16 shrink-0 items-center justify-between border-b border-platform-border bg-platform-card px-4 md:px-6"
      >
        <div class="flex min-w-0 items-center gap-2">
          <button
            class="flex h-9 w-9 items-center justify-center rounded-md text-platform-textMuted hover:bg-platform-bg hover:text-platform-textPrimary lg:hidden"
            aria-label="打开导航"
            @click="openMobileNav"
          >
            <el-icon :size="20"><Menu /></el-icon>
          </button>
          <button
            class="hidden h-9 w-9 items-center justify-center rounded-md text-platform-textMuted hover:bg-platform-bg hover:text-platform-textPrimary lg:flex"
            :aria-label="collapsed ? '展开侧栏' : '收起侧栏'"
            @click="collapsed = !collapsed"
          >
            <el-icon :size="18">
              <Expand v-if="collapsed" />
              <Fold v-else />
            </el-icon>
          </button>
          <div class="flex min-w-0 items-center gap-1.5 text-sm text-platform-textMuted">
            <span class="hidden sm:inline">平台运营</span>
            <span class="hidden text-platform-textHint sm:inline">/</span>
            <!-- 移动端标题由页面 H1 承担，顶栏只留角色名，避免同名重复。 -->
            <span class="truncate font-medium text-platform-textPrimary">
              <span class="hidden sm:inline">{{ currentTitle }}</span>
              <span class="sm:hidden">平台运营</span>
            </span>
          </div>
        </div>

        <el-dropdown trigger="click" @command="onCommand">
          <button
            class="flex items-center gap-2 rounded-md px-2 py-1.5 text-left hover:bg-platform-bg"
            aria-label="账号菜单"
          >
            <div
              class="flex h-9 w-9 items-center justify-center rounded-full bg-platform-brandBg text-sm font-semibold text-platform-brand"
            >
              {{ initial }}
            </div>
            <div class="hidden min-w-0 sm:block">
              <div class="truncate text-sm font-medium text-platform-textPrimary">{{ displayName }}</div>
              <div class="text-xs text-platform-textMuted">平台账号</div>
            </div>
            <el-icon :size="14" class="text-platform-textMuted"><ArrowDown /></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </header>

      <main class="flex-1 p-4 md:p-6">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ArrowDown,
  Box,
  Close,
  Coin,
  DataLine,
  Document,
  Expand,
  Fold,
  Menu,
  OfficeBuilding,
  Share,
  Tickets,
  User,
} from '@element-plus/icons-vue'
import { platformLogout } from '~/api/platform/auth'
import { PLATFORM_TOKEN_COOKIE, buildAuthCookieOptions } from '~/utils/authCookie'

// 侧栏与弹层由 EP 生成，样式统一挂在 body 上，避免 teleport 逃出页面作用域。
useHead({ bodyAttrs: { class: 'platform-surface' } })

const menus = [
  { title: '看板', path: '/platform', icon: DataLine },
  { title: '园所', path: '/platform/kindergartens', icon: OfficeBuilding },
  { title: '套餐', path: '/platform/plans', icon: Box },
  { title: '订单', path: '/platform/orders', icon: Tickets },
  { title: '分销商', path: '/platform/distributors', icon: Share },
  { title: '佣金台账', path: '/platform/commissions', icon: Coin },
  { title: '账号', path: '/platform/accounts', icon: User },
  { title: '审计', path: '/platform/logs', icon: Document },
]

const route = useRoute()
const mobileOpen = ref(false)
const collapsed = ref(false)
const displayName = ref('平台运营')

// 登录页写入显示名；没有时退回角色名，不把账号信息塞进 token 或 JWT。
onMounted(() => {
  const name = localStorage.getItem('platform_user_name')
  if (name) displayName.value = name
})

const initial = computed(() => displayName.value.trim().charAt(0) || '平')

const currentTitle = computed(() => {
  if (route.path === '/platform') return '运营看板'
  return menus.find((item) => item.path !== '/platform' && route.path.startsWith(item.path))?.title || '平台运营'
})

function isActive(path: string) {
  // 看板位于 /platform 根路径，用前缀匹配会让它在所有平台页都高亮。
  if (path === '/platform') return route.path === '/platform'
  return route.path.startsWith(path)
}

watch(
  () => route.path,
  () => {
    mobileOpen.value = false
  },
)

/** 移动端抽屉始终显示完整菜单，即使桌面端刚被折叠过。 */
function openMobileNav() {
  collapsed.value = false
  mobileOpen.value = true
}

async function onCommand(command: string) {
  if (command === 'logout') await logout()
}

async function logout() {
  try {
    await platformLogout()
  } catch {
    // 网络失败也要清本地登录态
  }
  localStorage.removeItem('platform_user_name')
  useCookie<string | null>(PLATFORM_TOKEN_COOKIE, buildAuthCookieOptions()).value = null
  await navigateTo('/platform/login')
}
</script>
