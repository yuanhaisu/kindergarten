import type { ActivityTask, TaskDetail, TaskFrequency, TaskItem, TaskRitual, TaskStarScope } from '~/types/api'
import type { TaskUpdateBody } from '~/api/tasks'

/**
 * 任务模块的前端枚举与视图辅助。
 *
 * 取值必须与后端常量保持一致：
 * - 频率：server/internal/domain/task.go `TaskFrequency*`
 * - 仪式：`TaskRitual*`（sticker / voice / class_star）
 * - 星星范围：`TaskStarScope*`（task / class）
 * 文案与 `taskFrequencyLabel`（task_service.go）同口径，避免两处叫法不同。
 */
export const TASK_FREQUENCY_OPTIONS: { value: TaskFrequency; label: string }[] = [
  { value: 'daily', label: '每日任务' },
  { value: 'weekly', label: '每周任务' },
  { value: 'once', label: '一次性任务' },
]

export const TASK_RITUAL_OPTIONS: { value: TaskRitual; label: string }[] = [
  { value: 'sticker', label: '贴纸' },
  { value: 'voice', label: '语音回应' },
  { value: 'class_star', label: '班级星星' },
]

export const TASK_STAR_SCOPE_OPTIONS: { value: TaskStarScope; label: string }[] = [
  { value: 'task', label: '按任务目标' },
  { value: 'class', label: '按班级星星罐' },
]

export function taskRitualLabel(value: string) {
  return TASK_RITUAL_OPTIONS.find((o) => o.value === value)?.label ?? value
}

export function taskStarScopeLabel(value: string) {
  return TASK_STAR_SCOPE_OPTIONS.find((o) => o.value === value)?.label ?? value
}

/** 任务状态：后端只有 0 已结束 / 1 进行中两种 */
export function taskStatusLabel(status: number) {
  return status === 1 ? '进行中' : '已结束'
}

export function taskStatusTagType(status: number): 'success' | 'info' {
  return status === 1 ? 'success' : 'info'
}

/** 公开状态四态的文案与标签色（取值来自服务端 `public_status`，前端不拼条件） */
const PUBLIC_STATUS_LABELS: Record<string, string> = {
  pending: '待审核',
  approved: '已批准',
  revoked: '已撤回',
  none: '未申请',
}

export function publicStatusLabel(status: string) {
  return PUBLIC_STATUS_LABELS[status] ?? status
}

export function publicStatusTagType(status: string): 'success' | 'warning' | 'info' | 'danger' {
  if (status === 'approved') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'revoked') return 'danger'
  return 'info'
}

/**
 * 列表分组：同一活动折叠成一条，独立任务各自成组。
 *
 * 后端列表按任务分页返回（活动 N 个班就是 N 行），这里只做展示层折叠，
 * 不改变分页口径——所以「活动显示几个班」只反映本页可见的行数。
 */
export function groupTasksByActivity(items: TaskItem[]): (TaskItem | ActivityTask)[] {
  const groups: (TaskItem | ActivityTask)[] = []
  const index = new Map<string, ActivityTask>()
  for (const item of items) {
    if (!item.activity_id) {
      groups.push(item)
      continue
    }
    const existing = index.get(item.activity_id)
    if (existing) {
      existing.tasks.push(item)
      continue
    }
    const group: ActivityTask = {
      activity_id: item.activity_id,
      title: item.title,
      tasks: [item],
    }
    index.set(item.activity_id, group)
    groups.push(group)
  }
  return groups
}

export function isActivityGroup(row: unknown): row is ActivityTask {
  return !!row && typeof row === 'object' && Array.isArray((row as ActivityTask).tasks)
}

/**
 * 表格行模型：活动组行与任务行共用同一套列，避免模板里到处判断 row.tasks。
 *
 * - `kind=activity`：折叠后的一条活动，`taskId=0`，`children` 是各班任务；
 * - `kind=task`：独立任务，或活动组展开后的某个班级任务（此时 `activityId` 非空）。
 *
 * 后端列表按**任务**分页返回，所以活动组的「N 个班」只反映本页可见的班级数。
 */
export interface TaskListRow {
  key: string
  kind: 'task' | 'activity'
  activityId: string
  taskId: number
  title: string
  classLabel: string
  classNames: string[]
  taskCount: number
  frequencyLabel: string
  startDate: string
  endDate: string
  status: number
  allowPublic: boolean
  /**
   * 活动组的公开状态：`all` 全公开 / `some` 组内不一致 / `none` 全不公开。
   * 单个任务允许单独改 allow_public，所以混合状态是真实存在的，
   * 用 `every()` 压成「不公开」会让管理员误判整组的公开情况。
   */
  publicState: 'all' | 'some' | 'none'
  starLabel: string
  children?: TaskListRow[]
}

function starLabelOf(t: TaskItem) {
  if (t.star_scope === 'class') return '班级星星罐'
  return t.star_goal > 0 ? `${t.star_goal} 颗` : '未设目标'
}

function taskRow(t: TaskItem): TaskListRow {
  return {
    key: `task:${t.id}`,
    kind: 'task',
    activityId: t.activity_id,
    taskId: t.id,
    title: t.title,
    classLabel: t.class_name || '—',
    classNames: t.class_name ? [t.class_name] : [],
    taskCount: 1,
    frequencyLabel: t.frequency_label,
    startDate: t.start_date,
    endDate: t.end_date,
    status: t.status,
    allowPublic: t.allow_public,
    publicState: t.allow_public ? 'all' : 'none',
    starLabel: starLabelOf(t),
  }
}

/**
 * 把列表接口返回的任务行折成表格行：同 activity_id 折叠成一条，独立任务各自一条。
 * 不改变分页口径，也不在前端重算任何比率（7.4 已下线每任务比率）。
 */
export function buildTaskListRows(items: TaskItem[]): TaskListRow[] {
  return groupTasksByActivity(items).map((entry) => {
    if (!isActivityGroup(entry)) return taskRow(entry)
    const tasks = entry.tasks
    return {
      key: `activity:${entry.activity_id}`,
      kind: 'activity',
      activityId: entry.activity_id,
      taskId: 0,
      title: entry.title,
      classLabel: `${tasks.length} 个班`,
      classNames: tasks.map((t) => t.class_name).filter(Boolean),
      taskCount: tasks.length,
      frequencyLabel: tasks[0].frequency_label,
      startDate: tasks.reduce((min, t) => (t.start_date < min ? t.start_date : min), tasks[0].start_date),
      endDate: tasks.reduce((max, t) => (t.end_date > max ? t.end_date : max), tasks[0].end_date),
      // 组状态取「任一进行中即进行中」：部分班级已结束不该让整组看起来已结束。
      status: tasks.some((t) => t.status === 1) ? 1 : 0,
      // 公开开关是活动级字段：只有整组都开着才算「已公开」，
      // 组内不一致时单独标「部分公开」，不要压成「不公开」。
      allowPublic: tasks.every((t) => t.allow_public),
      publicState: tasks.every((t) => t.allow_public)
        ? 'all'
        : tasks.some((t) => t.allow_public)
          ? 'some'
          : 'none',
      starLabel: starLabelOf(tasks[0]),
      children: tasks.map(taskRow),
    }
  })
}

/**
 * 独立任务（`activity_id` 为空）才有单条删除/单条编辑语义：
 * 活动任务的 DELETE 与 unsafe 单条编辑都会被后端 422，整组操作走 /admin/task-activities。
 */
export function isStandaloneTask(row: Pick<TaskListRow, 'activityId'>) {
  return row.activityId === ''
}

/**
 * 活动任务只允许改标题/说明/学习关联/公开开关（后端 422 口径）：
 * 即使没有提交也不能改结构，所以表单要按此锁定。
 */
export function lockedFieldsForTask(opts: {
  activityId: string
  submissionCount: number
}): {
  structureLocked: boolean
  /** 频率、日期、班级、个体目标、仪式、星星范围/目标 */
  lockReason: string
} {
  if (opts.activityId) {
    return {
      structureLocked: true,
      lockReason: '活动任务只能修改标题、说明、学习关联与公开开关，请使用活动编辑整组修改',
    }
  }
  if (opts.submissionCount > 0) {
    return {
      structureLocked: true,
      lockReason: '任务已有提交，只能修改标题、说明、学习关联与公开开关',
    }
  }
  return { structureLocked: false, lockReason: '' }
}

/** 编辑弹窗的字段快照（`current` 与 `baseline` 用同一形状，便于做差异判定） */
export interface TaskFormSnapshot {
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

/**
 * 详情 → 编辑表单快照（同时是差异基线）。
 *
 * 历史行可能缺字段：后端 `resolveTaskRitual` / `resolveTaskStarScope` 明令空值不归一，
 * 所以库里存在 `recognition_ritual=''` 的行（kg4 就有，且带提交）。下拉需要可显示的值，
 * 这里补默认值——**归一后的快照才是基线**，否则「没动过」会被判成改动并回传结构键，
 * 有提交的任务立刻 422（taskSafeEditableFields 只放行四个字段）。
 */
export function taskFormSnapshotFromDetail(detail: TaskDetail): TaskFormSnapshot {
  return {
    title: detail.title,
    description: detail.description ?? '',
    learning_link: detail.learning_link ?? '',
    class_id: detail.class_id,
    frequency: detail.frequency,
    start_date: detail.start_date,
    end_date: detail.end_date,
    recognition_ritual: detail.recognition_ritual || 'sticker',
    star_scope: detail.star_scope || 'task',
    star_goal: detail.star_goal,
    allow_public: detail.allow_public,
    status: detail.status,
  }
}

/**
 * 构造 `PUT /admin/tasks/:id` 的请求体：只有真的改过的字段才发。
 *
 * 为什么必须做差异而不是整表单回传：
 * - 后端对指针字段是「缺省 = 保留原值」，`star_goal`/`target_student_id` 却是
 *   「出现非 null 就当作改动」（0 是合法业务值，分不清「回传」与「改成 0」）；
 * - **有提交的任务**由 `taskSafeEditableFields` 只放行 title/description/learning_link/allow_public，
 *   回传任何一个结构键都会 422——包括「UI 把历史空仪式归一成 sticker」这种并非用户改动的差异。
 *
 * 因此 `baseline` 必须是**打开弹窗时归一后的表单快照**，而不是后端返回的原值。
 */
export function buildTaskUpdateBody(
  current: TaskFormSnapshot,
  baseline: TaskFormSnapshot,
): TaskUpdateBody {
  // required 字段必须给（值未变也算），后端按 binding:required 校验
  const body: TaskUpdateBody = {
    title: current.title.trim(),
    learning_link: current.learning_link.trim(),
    class_id: current.class_id,
    frequency: current.frequency,
    start_date: current.start_date,
    end_date: current.end_date,
  }
  if (current.description !== baseline.description) body.description = current.description
  if (current.allow_public !== baseline.allow_public) body.allow_public = current.allow_public
  if (current.recognition_ritual !== baseline.recognition_ritual) {
    body.recognition_ritual = current.recognition_ritual
  }
  if (current.star_scope !== baseline.star_scope) body.star_scope = current.star_scope
  // 按班级罐统计时任务目标无意义，不发以免被当成「改成 0」
  if (current.star_scope !== 'class' && current.star_goal !== baseline.star_goal) {
    body.star_goal = current.star_goal
  }
  // status 不受结构锁限制（后端把「结束任务」排除在结构字段之外），但活动任务必须整组改
  if (current.status !== baseline.status) body.status = current.status
  return body
}
