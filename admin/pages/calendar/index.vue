<template>
  <div class="space-y-4">
    <el-card shadow="never">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span>园级日历</span>
          <el-radio-group v-model="statusFilter" size="small" @change="load">
            <el-radio-button value="">全部</el-radio-button>
            <el-radio-button value="draft">草稿</el-radio-button>
            <el-radio-button value="published">已发布</el-radio-button>
          </el-radio-group>
        </div>
      </template>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
        <div v-loading="loading" class="min-w-0">
          <el-calendar v-model="calendarDate">
            <template #date-cell="{ data }">
              <div
                class="cal-cell"
                :class="{ 'is-adjacent': data.type !== 'current-month', 'is-picked': data.isSelected }"
              >
                <span class="cal-num">{{ Number(data.day.slice(8)) }}</span>
                <div class="cal-dots">
                  <span
                    v-for="(dot, i) in dotsFor(data.day)"
                    :key="i"
                    class="cal-dot"
                    :class="{ 'is-unknown': !dot }"
                    :style="dot ? { backgroundColor: dot } : undefined"
                  />
                </div>
              </div>
            </template>
          </el-calendar>
          <div v-if="typeOptions.length" class="legend">
            <span v-for="t in typeOptions" :key="t.key" class="legend-item">
              <span class="cal-dot" :class="{ 'is-unknown': !t.color }" :style="t.color ? { backgroundColor: t.color } : undefined" />
              {{ t.label }}
            </span>
          </div>
        </div>

        <div class="min-w-0">
          <div class="mb-3 flex items-start justify-between gap-2">
            <div>
              <div class="text-lg font-semibold text-text-primary">{{ dayLabel }}</div>
              <div class="mt-0.5 text-xs text-text-muted">园级事件 · 全园可见</div>
            </div>
            <el-button type="primary" @click="openCreate">新建园级事件</el-button>
          </div>

          <div v-if="!dayEvents.length" class="empty">当日暂无园级安排</div>
          <div v-else class="space-y-3">
            <div v-for="row in dayEvents" :key="row.id" class="event-card">
              <div class="flex items-start justify-between gap-2">
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <span
                      class="type-dot"
                      :class="{ 'is-unknown': !row.color }"
                      :style="row.color ? { backgroundColor: row.color } : undefined"
                    />
                    <span class="event-title">{{ row.title }}</span>
                  </div>
                  <div class="event-meta">
                    {{ timeText(row) }}<template v-if="row.location"> · {{ row.location }}</template>
                  </div>
                </div>
                <span class="status-pill" :class="row.status === 'published' ? 'is-published' : 'is-draft'">
                  {{ row.status === 'published' ? '已发布' : '草稿' }}
                </span>
              </div>
              <div v-if="row.description" class="event-desc">{{ row.description }}</div>
              <div class="mt-2 flex justify-end gap-1">
                <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
                <el-button link type="danger" @click="remove(row)">删除</el-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="editId ? '编辑园级事件' : '新建园级事件'"
      width="560px"
      destroy-on-close
    >
      <el-form :model="form" label-width="88px">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" maxlength="60" show-word-limit />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.event_type" class="w-full" placeholder="选择类型">
            <el-option v-for="t in typeOptions" :key="t.key" :label="t.label" :value="t.key">
              <span class="flex items-center gap-2">
                <span
                  class="type-dot"
                  :class="{ 'is-unknown': !t.color }"
                  :style="t.color ? { backgroundColor: t.color } : undefined"
                />
                {{ t.label }}
              </span>
            </el-option>
          </el-select>
        </el-form-item>
        <el-form-item label="日期" required>
          <el-date-picker
            v-model="form.event_date"
            type="date"
            value-format="YYYY-MM-DD"
            :clearable="false"
            class="!w-full"
          />
        </el-form-item>
        <el-form-item label="全天">
          <el-switch v-model="form.is_all_day" />
        </el-form-item>
        <el-form-item v-if="!form.is_all_day" label="时间">
          <div class="flex items-center gap-2">
            <el-time-picker
              v-model="form.start_time"
              value-format="HH:mm"
              format="HH:mm"
              placeholder="开始"
              class="!w-32"
            />
            <span class="text-text-muted">-</span>
            <el-time-picker
              v-model="form.end_time"
              value-format="HH:mm"
              format="HH:mm"
              placeholder="结束"
              class="!w-32"
            />
          </div>
        </el-form-item>
        <el-form-item label="地点">
          <el-input v-model="form.location" maxlength="80" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="3" maxlength="500" show-word-limit />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio value="draft">草稿（不对外可见）</el-radio>
            <el-radio value="published">发布</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="showNotifySwitch" label="通知">
          <el-switch v-model="form.notify" />
          <span class="ml-2 text-xs text-text-muted">发布后通知全园家长</span>
        </el-form-item>
        <el-form-item v-else-if="editingPublished" label="通知">
          <span class="text-xs text-text-muted">修改已发布事件，保存时会询问是否通知家长</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">
          {{ form.status === 'published' ? '保存并发布' : '存为草稿' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import type {
  GardenCalendarEventItem,
  GardenCalendarInput,
} from '~/types/api'
import {
  createGardenEvent,
  deleteGardenEvent,
  listGardenEvents,
  updateGardenEvent,
} from '~/api/calendar'
import { listDictItems, listDictTypes } from '~/api/dict'

// 园级日历管理（ui-design-notes §3）：月历（数据驱动类型色圆点）+ 当日列表 + 弹窗表单。
// 类型不硬编码：下拉与色点都来自园所字典 calendar_event_type（含园所自定义项）。

const loading = ref(false)
const saving = ref(false)
const calendarDate = ref(new Date())
const statusFilter = ref<'' | 'draft' | 'published'>('')
const events = ref<GardenCalendarEventItem[]>([])
let loadedMonth = ''

const dialogVisible = ref(false)
const editId = ref<number | null>(null)
const editingPublished = ref(false)
const original = ref<GardenCalendarEventItem | null>(null)

interface TypeOption {
  key: string
  label: string
  color: string
}
const typeOptions = ref<TypeOption[]>([])

const form = reactive({
  title: '',
  description: '',
  event_type: 'activity',
  event_date: '',
  start_time: '',
  end_time: '',
  location: '',
  is_all_day: false,
  status: 'published' as 'draft' | 'published',
  notify: false,
})

const showNotifySwitch = computed(
  () => form.status === 'published' && !editingPublished.value,
)

// ── 日期辅助 ──────────────────────────────────────────────────────────

function fmtDate(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function monthKey(d: Date) {
  return `${d.getFullYear()}-${d.getMonth()}`
}

function monthWindow(d: Date) {
  const start = new Date(d.getFullYear(), d.getMonth() - 1, 1)
  const end = new Date(d.getFullYear(), d.getMonth() + 2, 0)
  return { start: fmtDate(start), end: fmtDate(end) }
}

const selectedDay = computed(() => fmtDate(calendarDate.value))

const dayLabel = computed(() => {
  const d = calendarDate.value
  const week = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][d.getDay()]
  return `${d.getMonth() + 1} 月 ${d.getDate()} 日 ${week}`
})

// ── 数据加载 ──────────────────────────────────────────────────────────

async function load() {
  loading.value = true
  try {
    const { start, end } = monthWindow(calendarDate.value)
    const res = await listGardenEvents({
      start,
      end,
      status: statusFilter.value || undefined,
    })
    events.value = res.list
    loadedMonth = monthKey(calendarDate.value)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载园级日历失败')
  } finally {
    loading.value = false
  }
}

async function loadTypeOptions() {
  try {
    const types = await listDictTypes()
    const type = types.list.find((t) => t.type_key === 'calendar_event_type')
    if (!type) return
    const items = await listDictItems(type.id)
    typeOptions.value = items.list
      .filter((i) => i.is_enabled)
      .map((i) => ({
        key: i.item_key,
        label: i.item_label,
        color: typeof i.extra?.color === 'string' ? i.extra.color : '',
      }))
    if (typeOptions.value.length && !typeOptions.value.some((t) => t.key === form.event_type)) {
      form.event_type = typeOptions.value[0].key
    }
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载事件类型字典失败')
  }
}

// 月历翻月（EP 的上一月/下一月会更新 v-model）时重新拉取；同月内点选日期不重载。
watch(calendarDate, (d) => {
  if (monthKey(d) !== loadedMonth) load()
})

// ── 当日列表 ──────────────────────────────────────────────────────────

const dayEvents = computed(() => {
  return events.value
    .filter((e) => e.event_date === selectedDay.value)
    .sort((a, b) => {
      if (a.is_all_day !== b.is_all_day) return a.is_all_day ? -1 : 1
      return (a.start_time || '').localeCompare(b.start_time || '')
    })
})

function dotsFor(day: string): string[] {
  // 按事件类型去重（同类型只画一个点），颜色取自字典
  const byType = new Map<string, string>()
  for (const e of events.value) {
    if (e.event_date === day && !byType.has(e.event_type)) byType.set(e.event_type, e.color || '')
  }
  return [...byType.values()].slice(0, 3)
}

function timeText(e: GardenCalendarEventItem) {
  if (e.is_all_day) return '全天'
  if (e.start_time && e.end_time) return `${e.start_time}-${e.end_time}`
  return e.start_time || '未设时间'
}

// ── 表单 ──────────────────────────────────────────────────────────────

function resetForm() {
  // 与老师端一致默认「活动」；园所若删除了该字典项则退回字典首项
  const preferActivity = typeOptions.value.some((t) => t.key === 'activity')
  Object.assign(form, {
    title: '',
    description: '',
    event_type: preferActivity ? 'activity' : typeOptions.value[0]?.key || 'activity',
    event_date: selectedDay.value,
    start_time: '',
    end_time: '',
    location: '',
    is_all_day: false,
    status: 'published',
    notify: false,
  })
}

function openCreate() {
  editId.value = null
  editingPublished.value = false
  original.value = null
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: GardenCalendarEventItem) {
  editId.value = row.id
  editingPublished.value = row.status === 'published'
  original.value = { ...row }
  Object.assign(form, {
    title: row.title,
    description: row.description || '',
    event_type: row.event_type,
    event_date: row.event_date,
    start_time: row.start_time || '',
    end_time: row.end_time || '',
    location: row.location || '',
    is_all_day: row.is_all_day,
    status: row.status,
    notify: false,
  })
  dialogVisible.value = true
}

function rangeText(start: string | null, end: string | null, allDay: boolean) {
  if (allDay) return '全天'
  if (!start) return '未设'
  return end ? `${start}-${end}` : start
}

function diffSummary(old: GardenCalendarEventItem, next: GardenCalendarInput) {
  const parts: string[] = []
  let substantive = false
  if (old.event_date !== next.event_date) {
    parts.push(`日期 ${old.event_date} → ${next.event_date}`)
    substantive = true
  }
  const oldRange = rangeText(old.start_time, old.end_time, old.is_all_day)
  const newRange = rangeText(next.start_time, next.end_time, next.is_all_day)
  if (old.is_all_day !== next.is_all_day || oldRange !== newRange) {
    parts.push(`时间 ${oldRange} → ${newRange}`)
    substantive = true
  }
  if ((old.location || '') !== next.location) {
    parts.push(`地点 ${old.location || '无'} → ${next.location || '无'}`)
    substantive = true
  }
  if (old.title !== next.title) parts.push('标题已更新')
  if ((old.description || '') !== next.description) parts.push('内容已更新')
  if (old.event_type !== next.event_type) parts.push('类型已更新')
  return { text: parts.join('；'), substantive }
}

// 已发布事件的保存确认：主按钮 = 推荐选择（实质变更默认通知，仅文案变更默认不通知），
// 取消按钮 = 另一选择，右上角关闭 = 放弃保存。
async function askNotify(diff: { text: string; substantive: boolean }): Promise<boolean | null> {
  const primaryIsNotify = diff.substantive
  const message = diff.substantive
    ? `本次变更：${diff.text}。建议通知家长，避免按旧安排行动。`
    : `本次仅 ${diff.text}，通常无需打扰家长。`
  try {
    await ElMessageBox.confirm(message, '通知全园家长？', {
      confirmButtonText: primaryIsNotify ? '通知家长' : '不通知，仅保存',
      cancelButtonText: primaryIsNotify ? '不通知，仅保存' : '通知家长',
      distinguishCancelAndClose: true,
      type: diff.substantive ? 'warning' : 'info',
    })
    return primaryIsNotify
  } catch (action) {
    if (action === 'cancel') return !primaryIsNotify
    return null
  }
}

async function submit() {
  if (!form.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  if (!form.event_date) {
    ElMessage.warning('请选择日期')
    return
  }
  const payload: GardenCalendarInput = {
    title: form.title.trim(),
    description: form.description.trim(),
    event_type: form.event_type,
    event_date: form.event_date,
    start_time: form.is_all_day ? '' : form.start_time,
    end_time: form.is_all_day ? '' : form.end_time,
    location: form.location.trim(),
    is_all_day: form.is_all_day,
    status: form.status,
    notify: form.status === 'published' && form.notify,
  }

  // 编辑已发布事件：有变更时当场决定是否通知家长（草稿→发布走表单开关）
  if (editId.value && editingPublished.value && form.status === 'published' && original.value) {
    const diff = diffSummary(original.value, payload)
    if (diff.text) {
      const notify = await askNotify(diff)
      if (notify === null) return
      payload.notify = notify
    } else {
      payload.notify = false
    }
  }

  saving.value = true
  try {
    if (editId.value) {
      await updateGardenEvent(editId.value, payload)
      ElMessage.success(payload.status === 'published' ? '已保存' : '已存为草稿')
    } else {
      await createGardenEvent(payload)
      ElMessage.success(payload.status === 'published' ? '已发布' : '已存为草稿')
    }
    dialogVisible.value = false
    // 日期可能被改到当前加载窗口之外：同步选中日期并重新加载
    if (payload.event_date !== selectedDay.value) {
      calendarDate.value = new Date(`${payload.event_date}T00:00:00`)
    }
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function remove(row: GardenCalendarEventItem) {
  try {
    await ElMessageBox.confirm(`确定删除「${row.title}」？删除后全端不再可见。`, '提示', {
      type: 'warning',
    })
    await deleteGardenEvent(row.id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* 用户取消 */
  }
}

onMounted(() => {
  load()
  loadTypeOptions()
})
</script>

<style scoped>
.cal-cell {
  @apply flex h-full flex-col items-center gap-1 pt-1;
}
.cal-cell.is-adjacent {
  @apply opacity-40;
}
.cal-num {
  @apply inline-flex h-5 w-5 items-center justify-center text-base text-text-body;
}
.cal-cell.is-picked .cal-num {
  @apply rounded-full bg-brand-primary font-semibold text-text-onBrand;
}
.cal-dots {
  @apply flex h-2 items-center justify-center gap-1;
}
.cal-dot {
  @apply inline-block h-1.5 w-1.5 rounded-full;
}
.cal-dot.is-unknown {
  @apply bg-text-hint;
}
.legend {
  @apply mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 border-t border-border-hairline pt-3;
}
.legend-item {
  @apply flex items-center gap-1.5 text-xs text-text-muted;
}
.empty {
  @apply rounded-lg border border-dashed border-border-hairline py-10 text-center text-sm text-text-muted;
}
.event-card {
  @apply rounded-lg border border-border-hairline bg-bg-card p-3 shadow-sm;
}
.type-dot {
  @apply inline-block h-2 w-2 shrink-0 rounded-full;
}
.type-dot.is-unknown {
  @apply bg-text-hint;
}
.event-title {
  @apply truncate text-md font-semibold text-text-primary;
}
.event-meta {
  @apply mt-1 text-xs text-text-muted;
}
.event-desc {
  @apply mt-1.5 line-clamp-2 text-xs leading-5 text-text-body;
}
.status-pill {
  @apply shrink-0 rounded-full px-2 py-0.5 text-xs font-medium;
}
.status-pill.is-published {
  @apply bg-semantic-successBg text-semantic-success;
}
.status-pill.is-draft {
  @apply bg-semantic-warningBg text-semantic-warning;
}
:deep(.el-calendar-table .el-calendar-day) {
  padding: 2px;
  height: 64px;
}
:deep(.el-calendar-table td.is-selected) {
  background: transparent;
}
:deep(.el-calendar-table td.is-selected .el-calendar-day) {
  @apply bg-brand-primaryBg;
}
</style>
