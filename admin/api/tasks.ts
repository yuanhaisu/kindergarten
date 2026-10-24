import type {
  ClassStarJar,
  PageParams,
  PageResult,
  PublicCheckin,
  TaskDetail,
  TaskFrequency,
  TaskItem,
  TaskRitual,
  TaskStarScope,
  TaskStats,
} from '~/types/api'

export function getTasks(params?: PageParams & { class_id?: number; status?: number }) {
  return useHttp().get<PageResult<TaskItem>>('/admin/tasks', { params })
}

export function getTask(id: number) {
  return useHttp().get<TaskDetail>(`/admin/tasks/${id}`)
}

/** 独立任务创建：activity_id 必须缺省，多班活动走 createActivity */
export interface TaskCreateBody {
  title: string
  description?: string
  learning_link: string
  class_id: number
  target_student_id?: number
  frequency: TaskFrequency
  start_date: string
  end_date: string
  allow_public?: boolean
  recognition_ritual?: TaskRitual
  star_scope?: TaskStarScope
  star_goal?: number
}

/**
 * 编辑独立任务。`title` / `class_id` / `frequency` / `start_date` / `end_date` 是后端
 * `binding:"required"`（必须回传，值未变也算）；其余字段用指针语义：缺省 = 保留原值。
 * `learning_link` 在后端是「缺省或空 = 保留原值」，所以这里不能沿用创建体的必填约束。
 * 因此「没动过」的量（`star_goal` / `target_student_id`）必须省略键而不是传 0。
 */
export interface TaskUpdateBody {
  title: string
  description?: string
  learning_link?: string
  class_id: number
  target_student_id?: number
  frequency: TaskFrequency
  start_date: string
  end_date: string
  status?: number
  allow_public?: boolean
  recognition_ritual?: TaskRitual
  star_scope?: TaskStarScope
  star_goal?: number
}

export function createTask(body: TaskCreateBody) {
  return useHttp().post<{
    id: number
    created_at: string
    notified_count: number
    notify_failed_count: number
  }>('/admin/tasks', body)
}

export function updateTask(id: number, body: TaskUpdateBody) {
  return useHttp().put<{ id: number }>(`/admin/tasks/${id}`, body)
}

/** 活动任务的单条删除会被后端 422 拒绝，整组删除走 deleteActivity */
export function deleteTask(id: number) {
  return useHttp().delete(`/admin/tasks/${id}`)
}

export interface ActivityCreateBody {
  title: string
  description?: string
  learning_link: string
  frequency: TaskFrequency
  start_date: string
  end_date: string
  allow_public?: boolean
  recognition_ritual?: TaskRitual
  star_scope?: TaskStarScope
  star_goal?: number
  /** 非空、不得重复（重复是 422，不是静默去重）、≤50 */
  class_ids: number[]
}

/**
 * 活动批量编辑。后端只接受 title/description/learning_link/allow_public/status，
 * 显式传任何非空结构字段（frequency/class_ids/star_goal/...）都会 422，
 * 所以整表单回传时必须把结构位剔掉。
 */
export interface ActivityUpdateBody {
  title?: string
  description?: string
  learning_link?: string
  allow_public?: boolean
  /** 0 已结束 / 1 进行中；这是活动唯一的结束路径 */
  status?: number
}

export function createActivity(body: ActivityCreateBody) {
  return useHttp().post<{
    activity_id: string
    task_ids: number[]
    notified_count: number
    notify_failed_count: number
  }>('/admin/task-activities', body)
}

export function updateActivity(activityId: string, body: ActivityUpdateBody) {
  return useHttp().put<{ activity_id: string; updated_tasks: number }>(
    `/admin/task-activities/${activityId}`,
    body,
  )
}

export function deleteActivity(activityId: string) {
  return useHttp().delete<{ activity_id: string; deleted_tasks: number }>(
    `/admin/task-activities/${activityId}`,
  )
}

export type PublicCheckinStatusFilter = 'public' | 'revoked' | 'all'

/** 公开内容列表：后端缺省 20 / 上限 100，必须分页 */
export function listTaskPublicCheckins(
  taskId: number,
  params?: { status?: PublicCheckinStatusFilter } & PageParams,
) {
  return useHttp().get<PageResult<PublicCheckin>>(`/admin/tasks/${taskId}/checkins`, { params })
}

/** 管理员只能撤回；请求体只能是 {"revoke": true} */
export function revokeTaskPublicCheckin(checkinId: number) {
  return useHttp().put<{
    checkin_id: number
    task_id: number
    is_public_requested: boolean
    public_scope_confirmed: boolean
    public_approved_at: string | null
    public_approved_by: number
    public_revoked_at: string | null
    public_status: string
    public_visible: boolean
  }>(`/admin/tasks/checkins/${checkinId}/public`, { revoke: true })
}

export function getTaskStats(params?: { start_date?: string; end_date?: string }) {
  return useHttp().get<TaskStats>('/admin/tasks/stats', { params })
}

export function getClassStarJar(classId: number) {
  return useHttp().get<ClassStarJar>(`/admin/classes/${classId}/star-jar`)
}

/** 目标必须 > 0，0 / 负数 → 422 */
export function setClassStarJar(classId: number, goal: number) {
  return useHttp().put<ClassStarJar>(`/admin/classes/${classId}/star-jar`, { goal })
}
