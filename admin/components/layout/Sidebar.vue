<template>
  <aside
    class="shrink-0 bg-white border-r border-slate-200 flex flex-col transition-[width] duration-200"
    :class="collapsed ? 'w-[64px]' : 'w-[250px]'"
  >
    <!-- Logo 区域 -->
    <div class="h-16 flex items-center px-6 overflow-hidden">
      <div
        class="w-8 h-8 shrink-0 rounded-lg bg-gradient-to-b from-blue-500 to-blue-600 flex items-center justify-center"
      >
        <el-icon color="#fff" :size="18">
          <StarFilled />
        </el-icon>
      </div>
      <span
        v-if="!collapsed"
        class="ml-3 text-lg font-bold text-slate-800 whitespace-nowrap"
      >养成星球</span>
    </div>

    <!-- 导航菜单 -->
    <el-scrollbar class="flex-1 px-3">
      <div class="py-4">
        <template v-for="group in visibleGroups" :key="group.label">
          <!-- 分组标签 -->
          <div
            v-if="!collapsed"
            class="text-xs font-semibold text-slate-500 px-3 pt-4 pb-2 first:pt-0"
          >
            {{ group.label }}
          </div>
          <div v-else class="h-2" />

          <!-- 菜单项 -->
          <NuxtLink
            v-for="item in group.children"
            :key="item.path"
            :to="item.path"
            class="flex items-center h-10 px-3 rounded-lg my-0.5 text-sm no-underline transition-colors"
            :class="[
              collapsed ? 'justify-center' : '',
              isActive(item.path)
                ? 'bg-blue-50 text-blue-500 font-semibold'
                : 'text-slate-700 font-medium hover:bg-slate-100',
            ]"
            :title="collapsed ? item.title : ''"
          >
            <el-icon :size="18" :color="isActive(item.path) ? '#3B82F6' : '#64748B'">
              <component :is="item.icon" />
            </el-icon>
            <span v-if="!collapsed" class="ml-3">{{ item.title }}</span>
          </NuxtLink>
        </template>
      </div>
    </el-scrollbar>

    <!-- 底部用户信息 -->
    <div class="border-t border-slate-200 px-3 py-4" :class="collapsed ? 'flex justify-center' : ''">
      <div class="flex items-center gap-3" :class="collapsed ? 'justify-center' : ''">
        <div
          class="w-10 h-10 shrink-0 rounded-full bg-blue-100 flex items-center justify-center"
        >
          <el-icon :size="16" color="#3B82F6">
            <UserFilled />
          </el-icon>
        </div>
        <div v-if="!collapsed" class="flex-1 min-w-0">
          <div class="text-sm font-semibold text-slate-800 truncate">
            {{ auth.user?.nickname || '管理员' }}
          </div>
          <div class="text-xs text-slate-500 truncate">
            {{ auth.user?.role || '系统管理员' }}
          </div>
        </div>
        <button
          v-if="!collapsed"
          class="shrink-0 w-8 h-8 flex items-center justify-center rounded-lg text-slate-400 hover:bg-slate-100 hover:text-slate-600"
          @click="goSettings"
        >
          <el-icon :size="18"><Setting /></el-icon>
        </button>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { Setting, StarFilled, UserFilled } from '@element-plus/icons-vue'
import { ADMIN_MENU_GROUPS } from '~/constants/menus'

const route = useRoute()
const app = useAppStore()
const auth = useAuthStore()
const perm = usePermissionStore()
const router = useRouter()

const collapsed = computed(() => app.sidebarCollapsed)

const visibleGroups = computed(() =>
  ADMIN_MENU_GROUPS
    .map((g) => ({
      ...g,
      children: g.children.filter((m) => !m.permission || perm.can(m.permission)),
    }))
    .filter((g) => g.children.length > 0),
)

function isActive(path: string) {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

function goSettings() {
  router.push('/settings')
}
</script>
