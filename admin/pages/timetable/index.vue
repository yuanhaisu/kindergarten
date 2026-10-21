<template>
  <div class="space-y-4">
    <!-- 设置条 -->
    <el-card shadow="never">
      <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
        <div class="flex items-center gap-2">
          <span class="text-sm font-medium text-text-primary">同步课表到日历</span>
          <el-switch v-model="enabled" :loading="switching" @change="onToggle" />
        </div>
        <el-divider direction="vertical" />
        <el-button :loading="syncing" @click="syncNow">立即同步</el-button>
        <span v-if="syncResult" class="text-xs text-text-muted">
          新增 {{ syncResult.created }} · 更新 {{ syncResult.updated }} · 删除
          {{ syncResult.deleted }} · 跳过（老师已调整）{{ syncResult.skipped_overridden }}
        </span>
        <span v-else class="text-xs text-text-muted">同步未来 4 周；此后每日自动滚动补齐</span>
      </div>
      <div
        v-if="!enabled"
        class="mt-3 rounded-lg bg-semantic-warningBg px-3 py-2 text-xs text-semantic-warning"
      >
        当前未开启：课表仍可维护，但不会出现在老师端 / 家长端日历中。
      </div>
    </el-card>

    <!-- 年级模板 -->
    <el-card shadow="never">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span>年级课表模板</span>
          <el-radio-group v-model="activeGrade" size="small">
            <el-radio-button v-for="g in GRADES" :key="g" :value="g">{{ g }}</el-radio-button>
          </el-radio-group>
        </div>
      </template>
      <p class="hint">
        点格子添加或编辑；保存即全量替换「{{ activeGrade }}」模板。开启课表时未来 4
        周自动按模板更新，老师已调整过的课程不受影响。
      </p>
      <div class="grid-wrap">
        <div class="grid-head">节次</div>
        <div v-for="wd in WEEKDAYS" :key="wd.value" class="grid-head">{{ wd.label }}</div>
        <template v-for="p in PERIODS" :key="p">
          <div class="grid-period">第 {{ p }} 节</div>
          <button
            v-for="wd in WEEKDAYS"
            :key="`${wd.value}-${p}`"
            type="button"
            class="grid-cell"
            @click="openTemplateCell(wd.value, p)"
          >
            <template v-if="templateEntryAt(wd.value, p)">
              <span class="cell-title">{{ templateEntryAt(wd.value, p)?.title }}</span>
              <span class="cell-sub">{{ cellSub(templateEntryAt(wd.value, p)) }}</span>
            </template>
            <span v-else class="cell-empty">+</span>
          </button>
        </template>
      </div>
      <div v-if="!templateEntries.length" class="mt-3 text-center text-xs text-text-muted">
        尚未维护课表，点格子添加
      </div>
    </el-card>

    <!-- 班级微调 -->
    <el-card shadow="never">
      <template #header>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span>班级微调</span>
          <el-select
            v-model="selectedClassId"
            placeholder="选择班级"
            class="!w-52"
            @change="onClassChange"
          >
            <el-option
              v-for="c in classOptions"
              :key="c.id"
              :label="`${c.name}（${c.grade}）`"
              :value="c.id"
            />
          </el-select>
        </div>
      </template>
      <div v-if="!selectedClassId" class="empty">选择班级后微调（替换 / 移除 / 新增），微调格带角标。</div>
      <template v-else>
        <p class="hint">
          呈现「{{ classGrade }}模板 + 本班微调」的结果；点格子设置。微调随保存全量提交，开启课表时自动同步。
        </p>
        <div class="grid-wrap">
          <div class="grid-head">节次</div>
          <div v-for="wd in WEEKDAYS" :key="wd.value" class="grid-head">{{ wd.label }}</div>
          <template v-for="p in PERIODS" :key="p">
            <div class="grid-period">第 {{ p }} 节</div>
            <button
              v-for="wd in WEEKDAYS"
              :key="`${wd.value}-${p}`"
              type="button"
              class="grid-cell"
              :class="{ 'is-overridden': isOverridden(wd.value, p) }"
              @click="openOverrideCell(wd.value, p)"
            >
              <span v-if="isOverridden(wd.value, p)" class="cell-badge">微调</span>
              <template v-if="effectiveCell(wd.value, p)?.action === 'remove'">
                <span class="cell-removed">不上此课</span>
              </template>
              <template v-else-if="effectiveEntry(wd.value, p)">
                <span class="cell-title">{{ effectiveEntry(wd.value, p)?.title }}</span>
                <span class="cell-sub">{{ cellSub(effectiveEntry(wd.value, p)) }}</span>
              </template>
              <span v-else class="cell-empty">+</span>
            </button>
          </template>
        </div>
      </template>
    </el-card>

    <!-- 格子编辑弹窗（模板 / 微调共用） -->
    <el-dialog v-model="cellDialog.visible" :title="cellDialogTitle" width="480px" destroy-on-close>
      <el-form label-width="72px">
        <el-form-item v-if="cellDialog.mode === 'override' && hasTemplateAtCell" label="方式">
          <el-radio-group v-model="cellDialog.action">
            <el-radio value="replace">替换本格</el-radio>
            <el-radio value="remove">不上此课</el-radio>
          </el-radio-group>
        </el-form-item>
        <template v-if="cellDialog.action !== 'remove'">
          <el-form-item label="课程" required>
            <el-input v-model="cellDialog.title" maxlength="40" placeholder="如：美术课" />
          </el-form-item>
          <el-form-item label="地点">
            <el-input v-model="cellDialog.location" maxlength="40" placeholder="如：美术教室" />
          </el-form-item>
          <el-form-item label="时间">
            <div class="flex items-center gap-2">
              <el-time-picker
                v-model="cellDialog.start_time"
                value-format="HH:mm"
                format="HH:mm"
                placeholder="开始"
                class="!w-28"
              />
              <span class="text-text-muted">-</span>
              <el-time-picker
                v-model="cellDialog.end_time"
                value-format="HH:mm"
                format="HH:mm"
                placeholder="结束"
                class="!w-28"
              />
            </div>
          </el-form-item>
        </template>
        <div v-else class="rounded-lg bg-bg-page px-3 py-2 text-xs text-text-muted">
          该格将不再出现在本班日历中（模板默认课程不受影响）。
        </div>
      </el-form>
      <template #footer>
        <el-button v-if="canClearCell" link type="danger" @click="clearCell">清除此格</el-button>
        <span class="flex-1" />
        <el-button @click="cellDialog.visible = false">取消</el-button>
        <el-button type="primary" :loading="cellSaving" @click="saveCell">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import type { ClassItem, TimetableEntry, TimetableOverride, TimetableSyncResult } from '~/types/api'
import { getClasses } from '~/api/classes'
import {
  getTimetableClassOverrides,
  getTimetableSetting,
  getTimetableTemplate,
  saveTimetableClassOverrides,
  saveTimetableTemplate,
  syncTimetable,
  updateTimetableSetting,
} from '~/api/timetable'

// 课表管理（ui-design-notes §4）：开关 + 同步 → 年级模板网格 → 班级微调网格。
// 模板/微调均为全量提交；开启课表时服务端保存后自动重算未来 4 周。

const GRADES = ['小班', '中班', '大班']
const WEEKDAYS = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 7, label: '周日' },
]
const PERIODS = [1, 2, 3, 4, 5, 6, 7, 8]

const enabled = ref(false)
const switching = ref(false)
const syncing = ref(false)
const syncResult = ref<TimetableSyncResult | null>(null)

const activeGrade = ref('小班')
const templates = reactive<Record<string, TimetableEntry[]>>({ 小班: [], 中班: [], 大班: [] })

const classOptions = ref<ClassItem[]>([])
const selectedClassId = ref<number | null>(null)
const overrides = ref<TimetableOverride[]>([])

const cellSaving = ref(false)

function slotKey(weekday: number, period: number) {
  return `${weekday}-${period}`
}

function entriesToMap(entries: TimetableEntry[]) {
  const map = new Map<string, TimetableEntry>()
  for (const e of entries) map.set(slotKey(e.weekday, e.period), e)
  return map
}

const templateEntries = computed(() => templates[activeGrade.value] || [])
const templateMap = computed(() => entriesToMap(templateEntries.value))

const selectedClass = computed(() => classOptions.value.find((c) => c.id === selectedClassId.value))
const classGrade = computed(() => selectedClass.value?.grade || '小班')
const classTemplateMap = computed(() => entriesToMap(templates[classGrade.value] || []))
const overrideMap = computed(() => {
  const map = new Map<string, TimetableOverride>()
  for (const o of overrides.value) map.set(slotKey(o.weekday, o.period), o)
  return map
})

interface EffectiveCell {
  entry?: TimetableEntry
  action: 'template' | 'replace' | 'remove' | 'add'
}

const effectiveMap = computed(() => {
  const map = new Map<string, EffectiveCell>()
  for (const [k, e] of classTemplateMap.value) map.set(k, { entry: e, action: 'template' })
  for (const [k, o] of overrideMap.value) {
    if (o.action === 'remove') map.set(k, { action: 'remove' })
    else map.set(k, { entry: o, action: o.action })
  }
  return map
})

function templateEntryAt(weekday: number, period: number) {
  return templateMap.value.get(slotKey(weekday, period))
}

function effectiveCell(weekday: number, period: number) {
  return effectiveMap.value.get(slotKey(weekday, period))
}

function effectiveEntry(weekday: number, period: number) {
  return effectiveMap.value.get(slotKey(weekday, period))?.entry
}

function isOverridden(weekday: number, period: number) {
  return overrideMap.value.has(slotKey(weekday, period))
}

function cellSub(entry?: TimetableEntry) {
  if (!entry) return ''
  const range = entry.start_time
    ? entry.end_time
      ? `${entry.start_time}-${entry.end_time}`
      : entry.start_time
    : ''
  return [range, entry.location].filter(Boolean).join(' · ')
}

// ── 加载 ──────────────────────────────────────────────────────────────

async function loadSetting() {
  try {
    const res = await getTimetableSetting()
    enabled.value = res.timetable_enabled
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载课表开关失败')
  }
}

async function loadTemplate(grade: string) {
  try {
    const res = await getTimetableTemplate(grade)
    templates[grade] = res.entries || []
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : `加载${grade}模板失败`)
  }
}

async function loadClasses() {
  try {
    const res = await getClasses({ page: 1, page_size: 100 })
    classOptions.value = res.list.filter((c) => c.status === 1)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载班级失败')
  }
}

async function loadOverrides() {
  if (!selectedClassId.value) {
    overrides.value = []
    return
  }
  try {
    const res = await getTimetableClassOverrides(selectedClassId.value)
    overrides.value = res.overrides || []
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载班级微调失败')
  }
}

async function onClassChange() {
  await loadOverrides()
}

// ── 开关与同步 ────────────────────────────────────────────────────────

async function onToggle(value: boolean) {
  switching.value = true
  try {
    const res = await updateTimetableSetting(value)
    enabled.value = res.timetable_enabled
    ElMessage.success(value ? '已开启：课表已同步到日历' : '已关闭：课表事件不再呈现（数据保留）')
  } catch (e: unknown) {
    enabled.value = !value
    ElMessage.error(e instanceof Error ? e.message : '更新开关失败')
  } finally {
    switching.value = false
  }
}

async function syncNow() {
  syncing.value = true
  try {
    syncResult.value = await syncTimetable()
    ElMessage.success('同步完成')
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '同步失败')
  } finally {
    syncing.value = false
  }
}

// ── 格子编辑 ──────────────────────────────────────────────────────────

const cellDialog = reactive({
  visible: false,
  mode: 'template' as 'template' | 'override',
  weekday: 1,
  period: 1,
  action: 'replace' as 'replace' | 'remove' | 'add',
  title: '',
  location: '',
  start_time: '',
  end_time: '',
})

const cellDialogTitle = computed(() => {
  const wd = WEEKDAYS.find((w) => w.value === cellDialog.weekday)?.label || ''
  return `${wd} 第 ${cellDialog.period} 节 · ${cellDialog.mode === 'template' ? '模板' : '班级微调'}`
})

const hasTemplateAtCell = computed(
  () => !!classTemplateMap.value.get(slotKey(cellDialog.weekday, cellDialog.period)),
)

const canClearCell = computed(() => {
  if (cellDialog.mode === 'template') {
    return !!templateMap.value.get(slotKey(cellDialog.weekday, cellDialog.period))
  }
  return overrideMap.value.has(slotKey(cellDialog.weekday, cellDialog.period))
})

function fillCellFields(entry?: Partial<TimetableEntry>) {
  cellDialog.title = entry?.title || ''
  cellDialog.location = entry?.location || ''
  cellDialog.start_time = entry?.start_time || ''
  cellDialog.end_time = entry?.end_time || ''
}

function openTemplateCell(weekday: number, period: number) {
  cellDialog.mode = 'template'
  cellDialog.weekday = weekday
  cellDialog.period = period
  cellDialog.action = 'replace'
  fillCellFields(templateMap.value.get(slotKey(weekday, period)))
  cellDialog.visible = true
}

function openOverrideCell(weekday: number, period: number) {
  cellDialog.mode = 'override'
  cellDialog.weekday = weekday
  cellDialog.period = period
  const existing = overrideMap.value.get(slotKey(weekday, period))
  const templateEntry = classTemplateMap.value.get(slotKey(weekday, period))
  if (existing) {
    cellDialog.action = existing.action
    fillCellFields(existing)
  } else {
    // 无模板：新增；有模板：默认替换（比「不上此课」更常用）
    cellDialog.action = templateEntry ? 'replace' : 'add'
    fillCellFields(templateEntry)
  }
  cellDialog.visible = true
}

async function saveCell() {
  if (cellDialog.mode === 'template') {
    if (!cellDialog.title.trim()) {
      ElMessage.warning('请填写课程名称')
      return
    }
    const entry: TimetableEntry = {
      weekday: cellDialog.weekday,
      period: cellDialog.period,
      title: cellDialog.title.trim(),
      location: cellDialog.location.trim(),
      start_time: cellDialog.start_time || '',
      end_time: cellDialog.end_time || '',
    }
    const key = slotKey(cellDialog.weekday, cellDialog.period)
    const next = templateEntries.value.filter((e) => slotKey(e.weekday, e.period) !== key)
    next.push(entry)
    cellSaving.value = true
    try {
      const res = await saveTimetableTemplate(activeGrade.value, next)
      templates[activeGrade.value] = res.entries || next
      cellDialog.visible = false
      ElMessage.success(enabled.value ? '已保存，未来 4 周已同步' : '已保存（课表未开启，日历不呈现）')
    } catch (e: unknown) {
      ElMessage.error(e instanceof Error ? e.message : '保存失败')
    } finally {
      cellSaving.value = false
    }
    return
  }

  if (cellDialog.action !== 'remove' && !cellDialog.title.trim()) {
    ElMessage.warning('请填写课程名称')
    return
  }
  const override: TimetableOverride = {
    weekday: cellDialog.weekday,
    period: cellDialog.period,
    action: cellDialog.action,
    title: cellDialog.action === 'remove' ? '' : cellDialog.title.trim(),
    location: cellDialog.action === 'remove' ? '' : cellDialog.location.trim(),
    start_time: cellDialog.action === 'remove' ? '' : cellDialog.start_time || '',
    end_time: cellDialog.action === 'remove' ? '' : cellDialog.end_time || '',
  }
  await persistOverrides(override, '已保存微调')
}

async function persistOverrides(entry: TimetableOverride | null, successText: string) {
  if (!selectedClassId.value) return
  const key = entry ? slotKey(entry.weekday, entry.period) : null
  const next = overrides.value.filter((o) => slotKey(o.weekday, o.period) !== key)
  if (entry) next.push(entry)
  cellSaving.value = true
  try {
    const res = await saveTimetableClassOverrides(selectedClassId.value, next)
    overrides.value = res.overrides || next
    cellDialog.visible = false
    ElMessage.success(
      enabled.value ? `${successText}，未来 4 周已同步` : `${successText}（课表未开启，日历不呈现）`,
    )
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    cellSaving.value = false
  }
}

async function clearCell() {
  if (cellDialog.mode === 'template') {
    const key = slotKey(cellDialog.weekday, cellDialog.period)
    const next = templateEntries.value.filter((e) => slotKey(e.weekday, e.period) !== key)
    cellSaving.value = true
    try {
      const res = await saveTimetableTemplate(activeGrade.value, next)
      templates[activeGrade.value] = res.entries || next
      cellDialog.visible = false
      ElMessage.success(enabled.value ? '已移除，未来 4 周已同步' : '已移除（课表未开启）')
    } catch (e: unknown) {
      ElMessage.error(e instanceof Error ? e.message : '移除失败')
    } finally {
      cellSaving.value = false
    }
    return
  }
  await persistOverrides(null, '已清除该格微调')
}

onMounted(async () => {
  await Promise.all([loadSetting(), loadClasses(), ...GRADES.map((g) => loadTemplate(g))])
})
</script>

<style scoped>
.hint {
  @apply mb-3 text-xs leading-5 text-text-muted;
}
.empty {
  @apply rounded-lg border border-dashed border-border-hairline py-10 text-center text-sm text-text-muted;
}
.grid-wrap {
  @apply grid overflow-hidden rounded-lg border border-border-hairline;
  grid-template-columns: 72px repeat(7, minmax(0, 1fr));
}
.grid-head {
  @apply border-b border-r border-border-hairline bg-bg-page px-2 py-2 text-center text-xs font-medium text-text-muted;
}
.grid-wrap > :nth-child(8n) {
  border-right-width: 0;
}
.grid-period {
  @apply flex items-center justify-center border-b border-r border-border-hairline bg-bg-page px-1 text-xs text-text-muted;
}
.grid-cell {
  @apply relative flex min-h-[56px] cursor-pointer flex-col items-start justify-center gap-0.5 border-b border-r border-border-hairline bg-bg-card px-2 py-1.5 text-left transition-colors;
}
.grid-cell:hover {
  @apply bg-brand-primaryBg;
}
.grid-cell.is-overridden {
  @apply bg-brand-primaryBg;
}
.cell-title {
  @apply w-full truncate text-xs font-semibold text-text-primary;
}
.cell-sub {
  @apply w-full truncate text-xs text-text-muted;
}
.cell-empty {
  @apply w-full text-center text-sm text-text-hint;
}
.cell-removed {
  @apply text-xs text-text-hint line-through;
}
.cell-badge {
  @apply absolute right-1 top-1 rounded-full bg-brand-primary px-1.5 text-xs font-medium leading-4 text-text-onBrand;
}
</style>
