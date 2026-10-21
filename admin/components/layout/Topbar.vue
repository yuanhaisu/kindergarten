<template>
  <header
    class="h-16 shrink-0 flex items-center justify-between px-4 bg-white border-b border-slate-200"
  >
    <!-- 左侧：折叠按钮 + 面包屑（分组 / 页面，兼任页面标题） -->
    <div class="flex items-center gap-3 min-w-0">
      <el-button text circle aria-label="折叠或展开侧栏" @click="app.toggleSidebar">
        <el-icon :size="20">
          <Fold v-if="!app.sidebarCollapsed" />
          <Expand v-else />
        </el-icon>
      </el-button>
      <Breadcrumb />
    </div>

    <!-- 中间：人员速查（学生 / 老师 / 家长） -->
    <div ref="searchWrap" class="relative hidden lg:block w-64">
      <div
        class="flex items-center h-10 rounded-xl border border-slate-200 bg-slate-50 px-3 focus-within:border-blue-400 focus-within:bg-white transition-colors"
      >
        <el-icon :size="16" color="#94A3B8">
          <Search />
        </el-icon>
        <input
          v-model="searchText"
          class="flex-1 ml-2 bg-transparent border-none outline-none text-sm text-slate-700 placeholder-slate-500"
          placeholder="搜索学生 / 老师 / 家长"
          aria-label="按姓名或手机号搜索学生、老师或家长"
          @input="scheduleSearch"
          @keydown.enter.prevent="runSearch"
          @keydown.esc="open = false"
          @focus="onFocus"
        />
      </div>

      <div
        v-if="open"
        class="absolute z-20 left-0 right-0 mt-2 rounded-xl bg-white border border-slate-200 shadow-lg py-2 max-h-80 overflow-auto"
      >
        <div v-if="searching" class="px-4 py-2 text-sm text-slate-600">查询中…</div>
        <div v-else-if="error" class="px-4 py-2">
          <div class="text-sm font-medium text-text-primary">人员查询失败</div>
          <button
            class="mt-2 h-8 px-4 rounded-full bg-text-primary text-text-onBrand text-xs font-semibold hover:opacity-90"
            @click="runSearch"
          >
            重试
          </button>
        </div>
        <div v-else-if="!hasSearched" class="px-4 py-2 text-sm text-slate-600">
          输入姓名或手机号，查找学生 / 老师 / 家长
        </div>
        <div v-else-if="!groups.length" class="px-4 py-2 text-sm text-slate-600">无匹配人员</div>
        <template v-else>
          <div v-for="g in groups" :key="g.kind">
            <div class="px-4 pt-2 pb-1 text-xs font-semibold text-slate-600">{{ g.label }}</div>
            <button
              v-for="r in g.items"
              :key="`${g.kind}-${r.id}`"
              class="w-full text-left px-4 py-2 text-sm text-slate-700 hover:bg-slate-50 transition-colors"
              @click="go(g.kind, r)"
            >
              {{ r.name }}
              <span v-if="r.sub" class="ml-2 text-xs text-slate-500">{{ r.sub }}</span>
            </button>
          </div>
        </template>
      </div>
    </div>

    <!-- 右侧：用户 + 退出 -->
    <div class="flex items-center gap-3">
      <span class="text-sm text-slate-600 max-w-[120px] truncate hidden sm:inline">
        {{ auth.user?.nickname || '管理员' }}
      </span>
      <el-button type="danger" link size="small" @click="logout">退出</el-button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { Expand, Fold, Search } from '@element-plus/icons-vue'
import { getParents } from '~/api/parents'
import { getStudents } from '~/api/students'
import { getTeachers } from '~/api/teachers'

const app = useAppStore()
const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.clearSession()
  router.push('/login')
}

// ---- 人员速查 ----
type Hit = { id: number; name: string; sub?: string }
type HitGroup = { kind: 'students' | 'teachers' | 'parents'; label: string; items: Hit[] }

const searchWrap = ref<HTMLElement | null>(null)
const searchText = ref('')
const open = ref(false)
const searching = ref(false)
const hasSearched = ref(false)
const error = ref(false)
const groups = ref<HitGroup[]>([])

let timer: ReturnType<typeof setTimeout> | null = null
/** 回包序号：慢请求不得覆盖新查询的结果 */
let seq = 0

function scheduleSearch() {
  if (timer) clearTimeout(timer)
  timer = setTimeout(runSearch, 300)
}

async function runSearch() {
  const q = searchText.value.trim()
  open.value = true
  if (!q) {
    hasSearched.value = false
    groups.value = []
    error.value = false
    return
  }
  const my = ++seq
  searching.value = true
  error.value = false
  try {
    const [st, te, pa] = await Promise.all([
      getStudents({ keyword: q, page: 1, page_size: 5 }),
      getTeachers({ keyword: q, page: 1, page_size: 5 }),
      getParents({ keyword: q, page: 1, page_size: 5 }),
    ])
    if (my !== seq) return
    const next: HitGroup[] = []
    const students: Hit[] = (st.list || []).map((s) => ({
      id: s.id,
      name: s.name,
      sub: s.class_name,
    }))
    const teachers: Hit[] = (te.list || []).map((t) => ({
      id: t.id,
      name: t.nickname,
      sub: t.phone,
    }))
    const parents: Hit[] = (pa.list || []).map((p) => ({
      id: p.id,
      name: p.nickname,
      sub: p.phone,
    }))
    if (students.length) next.push({ kind: 'students', label: '学生', items: students })
    if (teachers.length) next.push({ kind: 'teachers', label: '老师', items: teachers })
    if (parents.length) next.push({ kind: 'parents', label: '家长', items: parents })
    groups.value = next
    hasSearched.value = true
  } catch (e: unknown) {
    if (my !== seq) return
    // 不静默：下拉里给出失败态与重试入口
    error.value = true
    console.error('[topbar-search] 人员查询失败', e)
  } finally {
    if (my === seq) searching.value = false
  }
}

function go(kind: HitGroup['kind'], r: Hit) {
  open.value = false
  const kw = searchText.value.trim() || r.name
  router.push(`/${kind}?keyword=${encodeURIComponent(kw)}`)
}

function onFocus() {
  if (searchText.value.trim() && hasSearched.value) open.value = true
}

function onClickOutside(e: MouseEvent) {
  if (searchWrap.value && !searchWrap.value.contains(e.target as Node)) {
    open.value = false
  }
}

onMounted(() => document.addEventListener('mousedown', onClickOutside))
onUnmounted(() => {
  document.removeEventListener('mousedown', onClickOutside)
  if (timer) clearTimeout(timer)
})
</script>
