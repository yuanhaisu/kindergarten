<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex flex-wrap items-center justify-between gap-2">
        <span>亲子任务</span>
        <el-button type="primary" @click="openCreate">新建任务</el-button>
      </div>
    </template>
    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="rows as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      row-key="key"
      :tree-props="{ children: 'children' }"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column label="任务" min-width="240" show-overflow-tooltip>
        <template #default="{ row }">
          <span class="text-text-primary">{{ row.title }}</span>
          <el-tag v-if="row.kind === 'activity'" size="small" type="warning" class="ml-2">多班活动</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="班级" min-width="120">
        <template #default="{ row }">
          <el-tooltip
            v-if="row.kind === 'activity'"
            :content="row.classNames.join('、') || '—'"
            placement="top"
          >
            <span class="text-text-body">{{ row.classLabel }}</span>
          </el-tooltip>
          <span v-else class="text-text-body">{{ row.classLabel }}</span>
        </template>
      </el-table-column>
      <el-table-column prop="frequencyLabel" label="频率" width="110" />
      <el-table-column label="起止日期" width="200">
        <template #default="{ row }">
          <span class="text-sm text-text-muted">{{ row.startDate }} ~ {{ row.endDate }}</span>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="taskStatusTagType(row.status)" size="small">
            {{ taskStatusLabel(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="公开" width="90">
        <template #default="{ row }">
          <el-tag :type="row.publicState === 'none' ? 'info' : 'warning'" size="small" effect="plain">
            {{ row.publicState === 'all' ? '公开' : row.publicState === 'some' ? '部分公开' : '不公开' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="星星" width="120">
        <template #default="{ row }">
          <span class="text-sm text-text-muted">{{ row.starLabel }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <template v-if="row.kind === 'activity'">
            <el-button link type="primary" @click="openEditActivity(row as TaskListRow)">编辑活动</el-button>
            <el-button link type="danger" @click="removeActivity(row as TaskListRow)">删除整个活动</el-button>
          </template>
          <template v-else>
            <el-button link type="primary" @click="openDetail(row as TaskListRow)">详情</el-button>
            <el-button
              v-if="isStandaloneTask(row as TaskListRow)"
              link
              type="primary"
              @click="openEditTask(row as TaskListRow)"
            >
              编辑
            </el-button>
            <el-button
              v-if="isStandaloneTask(row as TaskListRow)"
              link
              type="danger"
              @click="removeTask(row as TaskListRow)"
            >
              删除
            </el-button>
            <span v-else class="text-xs text-text-hint">随活动编辑</span>
          </template>
        </template>
      </el-table-column>
    </DataTable>

    <el-drawer v-model="detailVisible" title="任务详情" size="520px" destroy-on-close>
      <div v-if="detail" class="space-y-4">
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="标题">{{ detail.title }}</el-descriptions-item>
          <el-descriptions-item label="班级">{{ detail.class_name || '—' }}</el-descriptions-item>
          <el-descriptions-item label="学习关联">{{ detail.learning_link || '—' }}</el-descriptions-item>
          <el-descriptions-item label="说明">{{ detail.description || '—' }}</el-descriptions-item>
          <el-descriptions-item label="频率">{{ detail.frequency_label }}</el-descriptions-item>
          <el-descriptions-item label="起止日期">{{ detail.start_date }} ~ {{ detail.end_date }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="taskStatusTagType(detail.status)" size="small">
              {{ taskStatusLabel(detail.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="认可仪式">{{ taskRitualLabel(detail.recognition_ritual) }}</el-descriptions-item>
          <el-descriptions-item label="星星口径">
            {{ taskStarScopeLabel(detail.star_scope) }}
            <span v-if="detail.star_scope !== 'class'" class="text-text-muted">
              （目标 {{ detail.star_goal }} 颗）
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="公开内容">{{ detail.allow_public ? '允许公开' : '不公开' }}</el-descriptions-item>
          <el-descriptions-item label="创建者">{{ detail.creator?.nickname || '—' }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions title="回应指标（累计）" :column="2" border size="small">
          <el-descriptions-item label="提交数">{{ detail.submission_count }}</el-descriptions-item>
          <el-descriptions-item label="已回应">{{ detail.responded_count }}</el-descriptions-item>
          <el-descriptions-item label="待回应">{{ detail.pending_count }}</el-descriptions-item>
          <el-descriptions-item label="回应率">{{ formatRate(detail.response_rate) }}</el-descriptions-item>
          <el-descriptions-item label="参与率">{{ formatRate(detail.participation_rate) }}</el-descriptions-item>
          <el-descriptions-item label="通知家庭数">{{ formatMetric(detail.notify_target_count) }}</el-descriptions-item>
        </el-descriptions>
        <p class="text-xs text-text-hint">
          「—」表示没有可计算的分母（例如尚无提交），与 0% 不是同一回事；通知家庭数为「—」表示该任务未记录通知义务集。
        </p>
        <el-button type="primary" plain @click="openPublicContent(detail)">查看公开内容</el-button>
      </div>
      <div v-else-if="detailLoading" class="text-sm text-text-muted">加载中…</div>
    </el-drawer>

    <el-dialog v-model="publicVisible" :title="`公开内容 · ${publicTaskTitle}`" width="760px" destroy-on-close>
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <el-radio-group v-model="publicStatusFilter" @change="reloadPublic">
          <el-radio-button value="public">公开 / 待审核</el-radio-button>
          <el-radio-button value="revoked">已撤回</el-radio-button>
          <el-radio-button value="all">全部</el-radio-button>
        </el-radio-group>
        <span class="text-xs text-text-hint">「已不可见」= 已批准但此刻不在公开墙上（任务结束/关闭公开/孩子转班）</span>
      </div>
      <DataTable
        v-model:page="publicPage"
        v-model:page-size="publicPageSize"
        :data="publicList as unknown as Record<string, unknown>[]"
        :loading="publicLoading"
        :total="publicTotal"
        @page-change="loadPublic"
        @size-change="loadPublic"
      >
        <el-table-column prop="student_name" label="孩子" width="90" />
        <el-table-column prop="content" label="内容" min-width="160" show-overflow-tooltip />
        <el-table-column label="提交时间" width="150">
          <template #default="{ row }">
            <span class="text-sm text-text-muted">{{ formatDateTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="公开状态" width="110">
          <template #default="{ row }">
            <el-tag :type="publicStatusTagType(row.public_status)" size="small">
              {{ publicStatusLabel(row.public_status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="墙上可见" width="110">
          <template #default="{ row }">
            <el-tag v-if="row.public_visible" type="success" size="small" effect="plain">在墙上</el-tag>
            <el-tag v-else type="danger" size="small" effect="plain">已不可见</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button
              v-if="row.public_status === 'pending' || row.public_status === 'approved'"
              link
              type="danger"
              @click="revokePublic(row as PublicCheckin)"
            >
              撤回
            </el-button>
            <span v-else class="text-xs text-text-hint">—</span>
          </template>
        </el-table-column>
      </DataTable>
    </el-dialog>

    <el-dialog
      v-model="dialogVisible"
      :title="dialogTitle"
      width="640px"
      destroy-on-close
      :close-on-click-modal="false"
    >
      <el-alert
        v-if="structureLocked && mode !== 'create'"
        :title="lockReason"
        type="warning"
        :closable="false"
        show-icon
        class="mb-4"
      />
      <el-form :model="form" label-width="104px">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" maxlength="200" show-word-limit placeholder="例如：每日阅读 10 分钟" />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="给家长的任务说明" />
        </el-form-item>
        <el-form-item label="学习关联" required>
          <el-input v-model="form.learning_link" maxlength="200" show-word-limit placeholder="例如：延续本周班级的春天主题" />
        </el-form-item>
        <el-form-item label="班级" required>
          <div v-if="mode === 'edit-activity'" class="w-full text-sm text-text-body">
            {{ editingActivityClasses.join('、') || '—' }}
            <span class="ml-2 text-xs text-text-hint">
              活动覆盖 {{ editingActivityClasses.length }} 个班；增减班级需重建活动
            </span>
          </div>
          <el-select
            v-else-if="mode === 'create'"
            v-model="form.class_ids"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="选择一个班 = 独立任务；多个班 = 多班活动"
            class="w-full"
            @change="onClassChange"
          >
            <el-option v-for="c in classOptions" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
          <!-- 编辑单条任务时班级只能「换到另一个班」：多选在这里没有对应语义
               （多班 = 活动，需重建），选两个会静默丢掉第二个。 -->
          <el-select
            v-else
            v-model="editClassId"
            filterable
            :disabled="structureLocked"
            placeholder="所属班级"
            class="w-full"
            @change="onClassChange"
          >
            <el-option v-for="c in classOptions" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
          <p v-if="mode === 'create'" class="mt-1 text-xs text-text-hint">
            选 1 个班创建独立任务；选多个班创建多班活动（每班一条任务，共享同一活动分组）。
          </p>
        </el-form-item>
        <el-form-item label="频率" required>
          <el-select v-model="form.frequency" :disabled="structureLocked" class="w-full">
            <el-option v-for="o in TASK_FREQUENCY_OPTIONS" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="起止日期" required>
          <el-date-picker
            v-model="form.date_range"
            type="daterange"
            value-format="YYYY-MM-DD"
            start-placeholder="开始日期"
            end-placeholder="结束日期"
            :disabled="structureLocked"
            class="!w-full"
          />
        </el-form-item>
        <el-form-item label="认可仪式">
          <el-select v-model="form.recognition_ritual" :disabled="structureLocked" class="w-full">
            <el-option v-for="o in TASK_RITUAL_OPTIONS" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="星星范围">
          <el-select v-model="form.star_scope" :disabled="structureLocked" class="w-full" @change="onStarScopeChange">
            <el-option v-for="o in TASK_STAR_SCOPE_OPTIONS" :key="o.value" :label="o.label" :value="o.value" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.star_scope !== 'class'" label="星星目标">
          <el-input-number v-model="form.star_goal" :min="0" :disabled="structureLocked" />
          <span class="ml-2 text-xs text-text-hint">仪式为「班级星星」且按任务统计时必须大于 0</span>
        </el-form-item>
        <!-- 活动编辑不接受结构字段（含星星范围），也没有单个班级可选：
             这里只说明口径，不去查某个班的罐，否则会显示一条假的「查询失败」。 -->
        <el-form-item v-else-if="mode === 'edit-activity'" label="班级星星罐">
          <span class="text-sm text-text-body">按班级星星罐统计，目标在「班级管理 → 星星罐」按班设置</span>
        </el-form-item>
        <el-form-item v-else label="班级星星罐">
          <div class="w-full space-y-1">
            <div v-for="id in form.class_ids" :key="id" class="text-sm">
              <span class="text-text-body">{{ className(id) }}：</span>
              <span v-if="starJarState(id)?.status === 'loading'" class="text-text-muted">查询中…</span>
              <span v-else-if="starJarState(id)?.status === 'ok' && starJarState(id)?.jar?.configured" class="text-semantic-success">
                目标 {{ starJarState(id)?.jar?.goal }} 颗（{{ starJarState(id)?.jar?.term }}）
              </span>
              <span v-else-if="starJarState(id)?.status === 'ok'" class="text-semantic-danger">
                当前学期未配置，请先在「班级管理 → 星星罐」设置
              </span>
              <span v-else class="text-semantic-danger">
                查询失败：{{ starJarError(id) }}
              </span>
            </div>
            <p class="text-xs text-text-hint">按班级星星罐统计时，忽略任务自身的星星目标。</p>
          </div>
        </el-form-item>
        <el-form-item label="允许公开">
          <el-switch v-model="form.allow_public" />
          <span class="ml-2 text-xs text-text-hint">开启后老师可把提交推送到班级公开墙</span>
        </el-form-item>
        <el-form-item v-if="mode === 'edit-task'" label="任务状态">
          <el-radio-group v-model="taskStatus">
            <el-radio :value="1">进行中</el-radio>
            <el-radio :value="0">已结束</el-radio>
          </el-radio-group>
          <p class="mt-1 text-xs text-text-hint">
            结束任务后家长端不再展示；状态不属于结构字段，有提交后仍可改。
          </p>
        </el-form-item>
        <el-form-item v-if="mode === 'edit-activity'" label="活动状态">
          <el-radio-group v-model="activityStatus">
            <el-radio :value="1">进行中</el-radio>
            <el-radio :value="0">已结束</el-radio>
          </el-radio-group>
          <p v-if="activityStatusMixed" class="mt-1 text-xs text-semantic-warning">
            本页可见的班级之间状态不一致（列表按「任一进行中即进行中」显示）；不改这里就保持原样，改了对所有班级生效。
          </p>
          <p v-else class="mt-1 text-xs text-text-hint">活动只有整组结束这一条路径，单条任务不接受 status 修改。</p>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" :disabled="submitDisabled" @click="submitForm">
          {{ mode === 'create' ? '创建' : '保存' }}
        </el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type {
  ClassItem,
  ClassStarJar,
  PublicCheckin,
  TaskDetail,
  TaskFrequency,
  TaskItem,
  TaskRitual,
  TaskStarScope,
} from '~/types/api'
import {
  createActivity,
  createTask,
  deleteActivity,
  deleteTask,
  getClassStarJar,
  getTask,
  getTasks,
  listTaskPublicCheckins,
  revokeTaskPublicCheckin,
  updateActivity,
  updateTask,
  type ActivityUpdateBody,
  type PublicCheckinStatusFilter,
  type TaskUpdateBody,
} from '~/api/tasks'
import { getClasses } from '~/api/classes'
import {
  buildTaskListRows,
  buildTaskUpdateBody,
  isStandaloneTask,
  lockedFieldsForTask,
  publicStatusLabel,
  publicStatusTagType,
  TASK_FREQUENCY_OPTIONS,
  TASK_RITUAL_OPTIONS,
  TASK_STAR_SCOPE_OPTIONS,
  taskFormSnapshotFromDetail,
  taskRitualLabel,
  taskStarScopeLabel,
  taskStatusLabel,
  taskStatusTagType,
  type TaskFormSnapshot,
  type TaskListRow,
} from '~/utils/tasks'
import { formatDateTime, formatMetric, formatRate } from '~/utils/format'

const { page, pageSize, total } = usePagination()
const loading = ref(false)
const list = ref<TaskItem[]>([])
const rows = computed(() => buildTaskListRows(list.value))
/** 翻页/改页长时会并发请求：只接受最后一次的结果，否则慢响应会覆盖新页 */
let loadSeq = 0

async function load() {
  loading.value = true
  const seq = ++loadSeq
  try {
    const res = await getTasks({ page: page.value, page_size: pageSize.value })
    if (seq !== loadSeq) return
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    if (seq !== loadSeq) return
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    if (seq === loadSeq) loading.value = false
  }
}

// ── 班级选项 ──
const classOptions = ref<ClassItem[]>([])

/** 后端分页上限 100（pkg/pagination），page_size 传更大也会被截断 */
const CLASS_PAGE_SIZE = 100

async function loadClassOptions() {
  try {
    const all: ClassItem[] = []
    for (let p = 1; ; p++) {
      const res = await getClasses({ page: p, page_size: CLASS_PAGE_SIZE })
      all.push(...res.list)
      // 不靠「本页不足一页」判断，避免某一页恰好整页时提前结束
      if (all.length >= res.total || res.list.length === 0) break
    }
    classOptions.value = all
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载班级列表失败')
  }
}

function className(id: number) {
  return classOptions.value.find((c) => c.id === id)?.name ?? `#${id}`
}

// ── 详情抽屉 ──
// 列表接口不返回每任务比率（7.4 下线），指标必须现取详情。
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<TaskDetail | null>(null)
/** 详情/编辑弹窗的请求序号：只接受最后一次点击的结果 */
let detailSeq = 0
let editSeq = 0

async function openDetail(row: TaskListRow) {
  // 连点两条任务时，慢的那条后返回会盖掉新选中的：用自增序号丢弃过期响应
  const seq = ++detailSeq
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const data = await getTask(row.taskId)
    if (seq !== detailSeq) return
    detail.value = data
  } catch (e: unknown) {
    if (seq !== detailSeq) return
    ElMessage.error(e instanceof Error ? e.message : '加载详情失败')
  } finally {
    if (seq === detailSeq) detailLoading.value = false
  }
}

// ── 公开内容（查看 + 撤回，7.3）──
// 公开状态与可见性都是服务端算好的（public_status / public_visible），前端不得自己拼条件，
// 否则会出现「界面显示已公开、墙上却没有」。已批准但不可见的行要显式标出，不能隐藏。
const publicVisible = ref(false)
const publicTaskId = ref(0)
const publicTaskTitle = ref('')
const publicStatusFilter = ref<PublicCheckinStatusFilter>('public')
const publicList = ref<PublicCheckin[]>([])
const publicLoading = ref(false)
/** 筛选/翻页是并发请求：只接受最后一次的结果 */
let publicSeq = 0
const {
  page: publicPage,
  pageSize: publicPageSize,
  total: publicTotal,
} = usePagination()

function openPublicContent(task: TaskDetail) {
  publicTaskId.value = task.id
  publicTaskTitle.value = task.title
  publicStatusFilter.value = 'public'
  publicPage.value = 1
  // 换任务/重开弹窗时必须先清空：否则会先渲染上一个任务的行，标题却已是新任务
  publicSeq++
  publicList.value = []
  publicTotal.value = 0
  publicVisible.value = true
  void loadPublic()
}

async function loadPublic() {
  if (!publicTaskId.value) return
  publicLoading.value = true
  const seq = ++publicSeq
  try {
    const res = await listTaskPublicCheckins(publicTaskId.value, {
      status: publicStatusFilter.value,
      page: publicPage.value,
      page_size: publicPageSize.value,
    })
    if (seq !== publicSeq) return
    publicList.value = res.list
    publicTotal.value = res.total
  } catch (e: unknown) {
    if (seq !== publicSeq) return
    // 失败时不能留着上一次筛选的行：管理员会对着不属于当前筛选的记录点「撤回」
    publicList.value = []
    publicTotal.value = 0
    ElMessage.error(e instanceof Error ? e.message : '加载公开内容失败')
  } finally {
    if (seq === publicSeq) publicLoading.value = false
  }
}

function reloadPublic() {
  publicPage.value = 1
  void loadPublic()
}

async function revokePublic(row: PublicCheckin) {
  try {
    await ElMessageBox.confirm(
      `撤回「${row.student_name}」的这条内容？撤回后公开墙上立即不可见，且无法恢复为已公开。`,
      '撤回公开内容',
      { type: 'warning', confirmButtonText: '撤回' },
    )
  } catch {
    return
  }
  try {
    // 管理端只能撤回：请求体固定 {"revoke": true}，不能代替家长申请或批准。
    const res = await revokeTaskPublicCheckin(row.checkin_id)
    ElMessage.success(res.public_visible ? '已撤回（仍可见，请刷新）' : '已撤回')
    // 撤回后该行不再属于「公开/待审核」，留在原筛选里会直接消失，
    // 管理员就看不到撤回结果了；切到「全部」让 public_visible=false 可核对。
    if (publicStatusFilter.value === 'public') publicStatusFilter.value = 'all'
    await loadPublic()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '撤回失败')
  }
}

// ── 建 / 改表单 ──
type DialogMode = 'create' | 'edit-task' | 'edit-activity'

const dialogVisible = ref(false)
const mode = ref<DialogMode>('create')
const saving = ref(false)
const editingTaskId = ref(0)
const editingActivityId = ref('')
const activityStatus = ref(1)
/** 打开弹窗时的活动状态：只有真的改过才回传（避免把混合状态整组归一） */
const originalActivityStatus = ref(1)
/** 打开弹窗时「整组是否全公开」：关闭公开的确认与回传都以它为基线 */
const activityAllowPublicBaseline = ref(false)
/** 组内各任务状态是否不一致（只可能来自历史数据；列表按「任一进行中」聚合） */
const activityStatusMixed = ref(false)
/** 打开弹窗时的任务详情：只用于展示与星罐口径判断，差异判定见 formBaseline */
const originalDetail = ref<TaskDetail | null>(null)
/** 打开弹窗时**归一后**的表单快照：编辑提交体按它做差异判定 */
type FormBaseline = {
  title: string
  description: string
  learning_link: string
  class_id: number
  frequency: TaskFrequency
  start_date: string
  end_date: string
  recognition_ritual: TaskRitual
  star_scope: TaskStarScope
  star_goal: number
  allow_public: boolean
  status: number
}
const formBaseline = ref<FormBaseline | null>(null)
/** 独立任务的状态（活动任务的状态走活动端点整组改） */
const taskStatus = ref(1)
const structureLocked = ref(false)
const lockReason = ref('')
/**
 * 编辑前的公开状态。关闭公开会让后端在同一事务里撤下该任务（活动为全部任务）的
 * 已公开内容，且不可恢复，所以只在「true → false」时弹二次确认。
 */
const previouslyPublic = ref(false)

const form = reactive({
  title: '',
  description: '',
  learning_link: '',
  class_ids: [] as number[],
  frequency: 'daily' as TaskFrequency,
  date_range: [] as string[],
  recognition_ritual: 'sticker' as TaskRitual,
  star_scope: 'task' as TaskStarScope,
  star_goal: 10,
  allow_public: false,
})

const dialogTitle = computed(() => {
  if (mode.value === 'create') return '新建任务'
  if (mode.value === 'edit-activity') return '编辑多班活动'
  return '编辑任务'
})

/**
 * 活动编辑弹窗要显示「活动覆盖哪些班」。活动批量编辑不接受 class_ids，
 * 所以这里只展示、不提供增减（增减班级只能重建活动）。
 */
const editingActivityClasses = ref<string[]>([])

/**
 * 编辑单条任务时的班级：单选控件与 `form.class_ids` 之间的桥。
 * 只有一份真相（class_ids 数组），单选写回单元素数组。
 */
const editClassId = computed<number | null>({
  get: () => form.class_ids[0] ?? null,
  set: (v) => {
    form.class_ids = v ? [v] : []
  },
})

// 星星罐前置校验：star_scope=class 时每个班都必须有当前学期目标，否则提交必被 422。
type StarJarEntry = { status: 'loading' | 'ok' | 'error'; jar?: ClassStarJar; error?: string }
const starJars = ref<Record<number, StarJarEntry>>({})

function starJarState(id: number): StarJarEntry | undefined {
  return starJars.value[id]
}

function starJarError(id: number) {
  return starJars.value[id]?.error ?? '未知错误'
}

async function refreshStarJars() {
  if (form.star_scope !== 'class' || form.class_ids.length === 0) {
    starJars.value = {}
    return
  }
  const ids = [...new Set(form.class_ids)]
  const prev = starJars.value
  starJars.value = Object.fromEntries(ids.map((id) => [id, prev[id] ?? { status: 'loading' }]))
  await Promise.all(
    ids.map(async (id) => {
      try {
        const jar = await getClassStarJar(id)
        starJars.value = { ...starJars.value, [id]: { status: 'ok', jar } }
      } catch (e: unknown) {
        // 查询失败与「未配置」必须分开：前者是故障，不该被当成可以继续提交的状态。
        starJars.value = {
          ...starJars.value,
          [id]: { status: 'error', error: e instanceof Error ? e.message : '请求失败' },
        }
      }
    }),
  )
}

const starJarMissing = computed(() => {
  if (form.star_scope !== 'class') return false
  if (form.class_ids.length === 0) return false
  return form.class_ids.some((id) => {
    const entry = starJars.value[id]
    return !entry || entry.status !== 'ok' || !entry.jar?.configured
  })
})

/**
 * 后端只在「范围是 class 且本次改动了范围或班级」时才校验班级罐
 * （task_service.go: starScope == class && (scope != cur.scope || class_id != cur.class_id)）。
 * 前端若一律拦，班级罐后来被删掉的任务连改标题都做不了——比后端更严也是 bug。
 */
const starJarBlocksSubmit = computed(() => {
  if (form.star_scope !== 'class') return false
  if (mode.value === 'create') return true
  const origin = formBaseline.value
  if (!origin) return false
  return origin.star_scope !== 'class' || origin.class_id !== form.class_ids[0]
})

const submitDisabled = computed(() => {
  if (mode.value === 'edit-activity') return false
  if (structureLocked.value) return false
  return starJarBlocksSubmit.value && starJarMissing.value
})

function onClassChange() {
  void refreshStarJars()
}

function onStarScopeChange() {
  void refreshStarJars()
}

function resetForm() {
  Object.assign(form, {
    title: '',
    description: '',
    learning_link: '',
    class_ids: classOptions.value[0] ? [classOptions.value[0].id] : [],
    frequency: 'daily',
    date_range: [],
    recognition_ritual: 'sticker',
    star_scope: 'task',
    star_goal: 10,
    allow_public: false,
  })
  starJars.value = {}
  structureLocked.value = false
  lockReason.value = ''
  activityStatus.value = 1
  originalActivityStatus.value = 1
  activityStatusMixed.value = false
  activityAllowPublicBaseline.value = false
  editingActivityClasses.value = []
  originalDetail.value = null
  formBaseline.value = null
  taskStatus.value = 1
  previouslyPublic.value = false
}

function openCreate() {
  // 作废可能在途的编辑详情请求：否则它回来时会把「新建」弹窗改成编辑并清空输入
  editSeq++
  mode.value = 'create'
  editingTaskId.value = 0
  editingActivityId.value = ''
  resetForm()
  dialogVisible.value = true
}

function fillFromDetail(detailData: TaskDetail) {
  originalDetail.value = detailData
  const snap = taskFormSnapshotFromDetail(detailData)
  formBaseline.value = snap
  form.title = snap.title
  form.description = snap.description
  form.learning_link = snap.learning_link
  form.class_ids = [snap.class_id]
  form.frequency = snap.frequency
  form.date_range = [snap.start_date, snap.end_date]
  // 历史行的仪式/范围可能是空串（后端明令「不能强行归一化」）：
  // 这里给下拉一个可显示的值，差异判定用上面这份**归一后**的基线。
  form.recognition_ritual = snap.recognition_ritual
  form.star_scope = snap.star_scope
  form.star_goal = snap.star_goal
  form.allow_public = snap.allow_public
  taskStatus.value = snap.status
  previouslyPublic.value = snap.allow_public
}

/**
 * 编辑提交体：只有真的改过的字段才发。
 *
 * 后端对指针字段是「缺省 = 保留原值」，`star_goal` / `target_student_id` 又用
 * 「出现非 null 就当作改动」判定（0 是合法业务值）。一律回传会让「只改标题」
 * 撞上历史脏数据校验；有提交时更严重——`taskSafeEditableFields` 只放行
 * title/description/learning_link/allow_public，回传任何结构键都会 422。
 * 因此基准是**打开弹窗时归一后的表单快照**，不是后端原值。
 */
function buildUpdateBody(): TaskUpdateBody {
  const origin = formBaseline.value
  const current: TaskFormSnapshot = {
    title: form.title,
    description: form.description,
    learning_link: form.learning_link,
    class_id: form.class_ids[0],
    frequency: form.frequency,
    start_date: form.date_range[0],
    end_date: form.date_range[1],
    recognition_ritual: form.recognition_ritual,
    star_scope: form.star_scope,
    star_goal: form.star_goal,
    allow_public: form.allow_public,
    status: taskStatus.value,
  }
  if (!origin) {
    // 不可达：打开编辑弹窗必经 fillFromDetail，那里一定写基线。
    // 留「半套 body」比报错更糟——只发 required 会静默丢掉说明/公开开关的修改，
    // 发结构键又会在有提交时 422。宁可显式失败让管理员重开弹窗。
    throw new Error('内部状态异常：缺少编辑基线，请重新打开弹窗')
  }
  return buildTaskUpdateBody(current, origin)
}

async function openEditTask(row: TaskListRow) {
  try {
    const seq = ++editSeq
    const detailData = await getTask(row.taskId)
    // 连点两条任务时，先点那条的响应可能后到：丢弃过期结果，否则会打开并保存错的任务
    if (seq !== editSeq) return
    mode.value = 'edit-task'
    editingTaskId.value = row.taskId
    editingActivityId.value = ''
    resetForm()
    fillFromDetail(detailData)
    const lock = lockedFieldsForTask({
      activityId: detailData.activity_id,
      submissionCount: detailData.submission_count,
    })
    structureLocked.value = lock.structureLocked
    lockReason.value = lock.lockReason
    dialogVisible.value = true
    void refreshStarJars()
  } catch (e: unknown) {
    // 请求失败时用户可能已经点了别的行：陈旧错误提示不该冒出来
    if (seq === editSeq) ElMessage.error(e instanceof Error ? e.message : '加载任务失败')
  }
}

async function openEditActivity(row: TaskListRow) {
  const first = row.children?.[0]
  if (!first) {
    ElMessage.error('活动下没有可编辑的任务')
    return
  }
  try {
    const seq = ++editSeq
    const detailData = await getTask(first.taskId)
    if (seq !== editSeq) return
    mode.value = 'edit-activity'
    editingActivityId.value = row.activityId
    editingTaskId.value = 0
    resetForm()
    editingActivityClasses.value = row.classNames
    fillFromDetail(detailData)
    // 活动只能整组编辑安全字段：结构位一律锁定（后端对任何非空结构字段都 422）。
    structureLocked.value = true
    lockReason.value = '活动只能整组修改标题、说明、学习关联、公开开关与结束状态'
    // 取本班任务自身状态，而不是组行的聚合状态：混合状态时组行会显示「进行中」，
    // 拿它当初始值再回传就会把已结束的班重新打开。
    activityStatus.value = detailData.status
    originalActivityStatus.value = detailData.status
    activityStatusMixed.value = new Set((row.children ?? []).map((t) => t.status)).size > 1
    // 公开开关是组级效果：关闭会撤回**全部**班级任务的公开内容。
    // 单个任务允许单独改 allow_public（契约允许），所以组内可能不一致——
    // 以「整组是否全公开」为基线，混合组不会因为编辑标题就被整组撤回。
    form.allow_public = row.allowPublic
    activityAllowPublicBaseline.value = row.allowPublic
    dialogVisible.value = true
  } catch (e: unknown) {
    // 同 openEditTask：陈旧的失败提示不应覆盖用户当前操作
    if (seq === editSeq) ElMessage.error(e instanceof Error ? e.message : '加载活动失败')
  }
}

async function submitForm() {
  if (!form.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  if (!form.learning_link.trim()) {
    ElMessage.warning('学习关联必填')
    return
  }
  if (mode.value !== 'edit-activity' && form.class_ids.length === 0) {
    ElMessage.warning('请选择班级')
    return
  }

  if (mode.value === 'edit-activity') {
    await submitActivityEdit()
    return
  }

  if (!form.date_range || form.date_range.length !== 2) {
    ElMessage.warning('请选择起止日期')
    return
  }
  if (starJarBlocksSubmit.value && starJarMissing.value) {
    ElMessage.warning('所选班级当前学期尚未配置星星罐目标，请先在班级管理中设置')
    return
  }
  // 后端规则：仪式为「班级星星」且按任务统计时目标必须 > 0。
  // 提前拦一道，否则只改标题/说明的历史脏数据任务也会被这条规则挡成 422。
  const starGoalTouched =
    mode.value === 'create' ||
    !formBaseline.value ||
    form.star_goal !== formBaseline.value.star_goal ||
    form.recognition_ritual !== formBaseline.value.recognition_ritual ||
    form.star_scope !== formBaseline.value.star_scope
  if (
    starGoalTouched &&
    form.recognition_ritual === 'class_star' &&
    form.star_scope === 'task' &&
    form.star_goal <= 0
  ) {
    ElMessage.warning('仪式为「班级星星」且按任务统计时，星星目标必须大于 0')
    return
  }

  const allowPublic = form.allow_public
  if (!allowPublic && previouslyPublic.value) {
    const ok = await confirmRevokePublic()
    if (!ok) return
  }

  // tasks 2.10：结束任务是破坏性状态变更（家长端立即不再展示），必须二次确认。
  // 只在「进行中 → 已结束」时询问，重复保存已结束任务不打扰管理员。
  if (
    mode.value === 'edit-task' &&
    formBaseline.value &&
    taskStatus.value === 0 &&
    formBaseline.value.status !== 0
  ) {
    const ok = await confirmEndTask('该任务')
    if (!ok) return
  }

  const [startDate, endDate] = form.date_range
  // 基础字段里除 description / allow_public 外都是后端 required，必须回传（值未参与编辑）
  const base = {
    title: form.title.trim(),
    learning_link: form.learning_link.trim(),
    frequency: form.frequency,
    start_date: startDate,
    end_date: endDate,
  }

  saving.value = true
  try {
    if (mode.value === 'create') {
      const starFields = {
        recognition_ritual: form.recognition_ritual,
        star_scope: form.star_scope,
        star_goal: form.star_scope === 'class' ? 0 : form.star_goal,
      }
      if (form.class_ids.length === 1) {
        const res = await createTask({
          ...base,
          ...starFields,
          description: form.description,
          allow_public: allowPublic,
          class_id: form.class_ids[0],
        })
        // 通知失败不回滚任务（D9），但必须让管理员看到，否则「家长没收到」无从追查
        ElMessage.success(
          notifyWarnSuffix(
            `已创建（通知 ${res.notified_count} 位家长）`,
            res.notify_failed_count,
          ),
        )
      } else {
        const res = await createActivity({
          ...base,
          ...starFields,
          description: form.description,
          allow_public: allowPublic,
          class_ids: form.class_ids,
        })
        ElMessage.success(
          notifyWarnSuffix(`已创建多班活动，共 ${res.task_ids.length} 条任务`, res.notify_failed_count),
        )
      }
    } else {
      await updateTask(editingTaskId.value, buildUpdateBody())
      ElMessage.success('已保存')
    }
    dialogVisible.value = false
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function submitActivityEdit() {
  const nextAllowPublic = form.allow_public
  const publicChanged = nextAllowPublic !== activityAllowPublicBaseline.value
  // 只有「本来整组全公开 → 现在关掉」才真的会撤下内容，其余情形不打扰管理员
  if (publicChanged && !nextAllowPublic) {
    const ok = await confirmRevokePublic()
    if (!ok) return
  }
  // 活动整组结束同样是破坏性变更，需要一次显式确认（tasks 2.10）。
  if (activityStatus.value === 0 && originalActivityStatus.value !== 0) {
    const ok = await confirmEndTask('该活动下所有班级')
    if (!ok) return
  }
  saving.value = true
  try {
    // 活动端点只接受安全字段 + status；结构位一律不发（后端对非空结构字段 422）。
    // status 只在真的改过时发：活动组若存在「部分已结束」的混合状态，
    // 无脑回传单选值会把已结束的班重新打开。
    const body: ActivityUpdateBody = {
      title: form.title.trim(),
      description: form.description,
      learning_link: form.learning_link.trim(),
    }
    // 与 status 同理：只有真的改过才回传，避免把混合组的公开状态整组归一
    if (publicChanged) body.allow_public = nextAllowPublic
    if (activityStatus.value !== originalActivityStatus.value) {
      body.status = activityStatus.value
    }
    const res = await updateActivity(editingActivityId.value, body)
    ElMessage.success(`已更新 ${res.updated_tasks} 条任务`)
    dialogVisible.value = false
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

/** 通知失败不回滚任务，但必须出现在成功提示里（否则家长没收到也看不出来） */
function notifyWarnSuffix(text: string, failed: number) {
  return failed > 0 ? `${text}，其中 ${failed} 条通知发送失败` : text
}

async function confirmRevokePublic(): Promise<boolean> {  try {
    await ElMessageBox.confirm(
      '关闭公开后，该任务已公开的内容将全部撤下，公开墙上不再可见。确定继续？',
      '关闭公开',
      { type: 'warning', confirmButtonText: '关闭并撤下', cancelButtonText: '取消' },
    )
    return true
  } catch {
    return false
  }
}

/** 结束任务（status=0）的二次确认：取消时整个保存中止，不写任何字段。 */
async function confirmEndTask(scope: string): Promise<boolean> {
  try {
    await ElMessageBox.confirm(
      `${scope}结束后家长端不再展示，已公开内容也会随任务下线。确定结束？`,
      '结束任务',
      { type: 'warning', confirmButtonText: '结束任务', cancelButtonText: '取消' },
    )
    return true
  } catch {
    return false
  }
}

// ── 删除 ──
async function removeTask(row: TaskListRow) {
  if (!isStandaloneTask(row)) {
    // 活动任务的 DELETE 会被后端 422；这里拦一道，避免把可修的提示变成报错。
    ElMessage.warning('活动任务不能单独删除，请删除整个活动')
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除任务「${row.title}」？该任务的提交与回应会一并删除。`, '删除任务', {
      type: 'warning',
    })
  } catch {
    return // 用户取消
  }
  // confirm 与请求分开 catch：把 422/503 吞成「取消」会让管理员以为删成功了
  try {
    await deleteTask(row.taskId)
    ElMessage.success('已删除')
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

async function removeActivity(row: TaskListRow) {
  try {
    await ElMessageBox.confirm(
      // 列表按任务分页，row.taskCount 只是本页可见的班级数，别拿它当删除条数承诺
      `确定删除活动「${row.title}」？将删除该活动下所有班级的任务及其提交（本页可见 ${row.taskCount} 条）。`,
      '删除整个活动',
      { type: 'warning' },
    )
  } catch {
    return
  }
  try {
    const res = await deleteActivity(row.activityId)
    ElMessage.success(`已删除 ${res.deleted_tasks} 条任务`)
    await load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '删除失败')
  }
}

// 弹窗关闭后，在途的编辑详情请求不能再把弹窗打开/改写表单
watch(dialogVisible, (open) => {
  if (!open) editSeq++
})

onMounted(async () => {
  await loadClassOptions()
  await load()
})
</script>
