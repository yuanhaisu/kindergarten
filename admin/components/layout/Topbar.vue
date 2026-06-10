<template>
  <header
    class="h-16 shrink-0 flex items-center justify-between px-4 bg-white border-b border-slate-200"
  >
    <!-- 左侧：折叠按钮 + 页面标题 -->
    <div class="flex items-center gap-3">
      <el-button text circle @click="app.toggleSidebar">
        <el-icon :size="20">
          <Fold v-if="!app.sidebarCollapsed" />
          <Expand v-else />
        </el-icon>
      </el-button>
      <span class="text-xl font-bold text-slate-800 hidden sm:inline">{{ pageTitle }}</span>
    </div>

    <!-- 中间：搜索框 -->
    <div class="hidden lg:flex items-center h-10 w-64 rounded-xl border border-slate-200 bg-slate-50 px-3 focus-within:border-blue-400 focus-within:bg-white transition-colors">
      <el-icon :size="16" color="#94A3B8">
        <Search />
      </el-icon>
      <input
        class="flex-1 ml-2 bg-transparent border-none outline-none text-sm text-slate-700 placeholder-slate-400"
        placeholder="搜索..."
      />
    </div>

    <!-- 右侧：通知 + 用户 + 退出 -->
    <div class="flex items-center gap-3">
      <!-- 通知按钮 -->
      <el-button text circle class="relative">
        <el-icon :size="20" color="#64748B">
          <Bell />
        </el-icon>
        <span
          class="absolute top-1 right-1 w-2 h-2 rounded-full bg-red-500 ring-2 ring-white"
        />
      </el-button>

      <span class="text-sm text-slate-600 max-w-[120px] truncate hidden sm:inline">
        {{ auth.user?.nickname || '管理员' }}
      </span>
      <el-button type="danger" link size="small" @click="logout">退出</el-button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { Bell, Expand, Fold, Search } from '@element-plus/icons-vue'
import { ADMIN_MENUS } from '~/constants/menus'

const app = useAppStore()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

const pageTitle = computed(() => {
  const menu = ADMIN_MENUS.find((m) => {
    if (m.path === '/') return route.path === '/'
    return route.path.startsWith(m.path)
  })
  return menu?.title || '管理后台'
})

function logout() {
  auth.clearSession()
  router.push('/login')
}
</script>
