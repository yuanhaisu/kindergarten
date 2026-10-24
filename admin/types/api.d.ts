/** 统一 API 包装（实际响应外层） */
export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data: T
}

export interface PageParams {
  page?: number
  page_size?: number
  keyword?: string
}

export interface PageResult<T> {
  list: T[]
  total: number
  page: number
  page_size: number
}

export interface LoginPasswordBody {
  phone: string
  password: string
}

export interface UserRoleItem {
  role: string
  kindergarten_id: number
  kindergarten_name?: string
  class_id?: number
  class_name?: string
  title?: string
  admin_role?: string
  /** 管理端角色条目才有：该账号权限码集合（通配为 ["*"]） */
  permissions?: string[]
}

export interface AuthUser {
  id: number
  phone: string
  nickname: string
  avatar?: string
  gender: number
  roles: UserRoleItem[]
}

export interface LoginResult {
  token: string
  refresh_token: string
  expires_at: string
  user: AuthUser
}

// ===== Dashboard 扩展类型 =====

export interface DashboardStatItem {
  label: string
  value: number | string
  trend?: string
  trendDirection?: 'up' | 'down' | 'flat'
}

export interface LeaveReasonItem {
  name: string
  value: number
  color?: string
}

export interface ClassActivityItem {
  class_id: number
  class_name: string
  activity_rate: number
  trend: 'up' | 'down' | 'flat'
  activity_score?: number
}

export interface ClassActivityRes {
  list: ClassActivityItem[]
}

export interface TodoItem {
  id: number
  title: string
  description: string
  time_ago: string
  action_type: string
  /** 落地页路径（可带 query）；缺省时不渲染操作按钮，避免死控件 */
  target?: string
}

export interface LeaveReason {
  name: string
  value: number
}

export interface LeaveReasonRes {
  total: number
  reasons: LeaveReason[]
}

export interface DashboardOverview {
  kindergarten: { id: number; name: string }
  realtime: {
    present_count: number
    leave_count: number
    total_student_count: number
    attendance_rate: string
    picked_up_count: number
    not_picked_up_count: number
  }
  today: {
    new_notices: number
    pending_leaves: number
    growth_logs: number
    new_tasks: number
  }
  this_month: {
    avg_attendance_rate: string
    total_growth_logs: number
    total_points_awarded: number
    new_students: number
    new_parents: number
  }
}

export interface AttendanceTrendPoint {
  date: string
  present: number
  leave: number
  absent: number
  rate: string
}

export interface AttendanceTrendRes {
  period: string
  data: AttendanceTrendPoint[]
}

export interface ClassRankingItem {
  class_id: number
  class_name: string
  present: number
  total: number
  rate: string
}

export interface ClassRankingRes {
  list: ClassRankingItem[]
}

export interface GrowthStatsRes {
  period: string
  total_logs: number
  total_media: number
  total_likes: number
  total_comments: number
  daily_avg_logs: number
  top_classes: { class_name: string; logs_count: number; teacher_name: string }[]
}

export interface KindergartenInfo {
  id: number
  name: string
  address: string
  phone: string
  principal: string
  logo_url: string
  license_no: string
  capacity: number
  current_count: number
  class_count: number
  teacher_count: number
  established_at: string
  description: string
}

export interface ClassTeacherBrief {
  id: number
  nickname: string
  title: string
  title_label: string
}

export interface ClassItem {
  id: number
  name: string
  grade: string
  academic_year: string
  capacity: number
  student_count: number
  teachers: ClassTeacherBrief[]
  status: number
  created_at: string
}

export interface ClassDetail extends Omit<ClassItem, 'teachers'> {
  teachers: (ClassTeacherBrief & { avatar?: string; phone?: string })[]
  students_preview: { id: number; name: string; avatar: string }[]
}

export interface StudentItem {
  id: number
  name: string
  avatar: string
  gender: number
  birthday: string
  age: string
  student_no: string
  class_name: string
  class_id: number
  status: number
  status_label: string
  enroll_date: string
  parents_count: number
  created_at: string
  /** 人脸底库聚合状态：0未录入 1已注册 2注册失败 3注册中 */
  face_status: number
  face_status_label: string
  face_count: number
}

/** 统一上传通道（/common/upload）的返回项 */
export interface UploadedFile {
  id: number
  file_url: string
  thumb_url?: string
  file_name: string
  file_size: number
  file_type: string
  duration: number
}

/** 学生人脸底库照片行 */
export interface StudentFace {
  id: number
  face_url: string
  /** 0待注册 1已注册 2注册失败 */
  status: number
  status_label: string
  last_error: string
  created_at: string
}

/** 学生人脸底库（照片列表 + 聚合状态） */
export interface StudentFacesResult {
  list: StudentFace[]
  total: number
  status: number
  status_label: string
  enabled: boolean
  provider: string
  max_count: number
}

export interface StudentParentDetail {
  user_id: number
  name: string
  phone: string
  relation: string
  relation_label: string
  is_primary: boolean
  avatar?: string
  id_card_number?: string
}

export interface StudentDetail {
  id: number
  name: string
  avatar: string
  gender: number
  birthday: string
  age: string
  student_no: string
  id_card_number?: string
  health_note?: string
  address?: string
  class_id: number
  class_name: string
  enroll_date: string
  status: number
  status_label: string
  parents: StudentParentDetail[]
  pickup_authorizations: {
    id: number
    name: string
    relation: string
    phone: string
    status: number
  }[]
  created_at: string
}

export interface TeacherItem {
  id: number
  nickname: string
  avatar: string
  phone: string
  gender: number
  classes: { class_id: number; class_name: string; title: string; title_label: string }[]
  status: number
  status_label: string
  join_date: string
  created_at: string
}

export interface TeacherDetail {
  id: number
  nickname: string
  real_name: string
  avatar: string
  phone: string
  gender: number
  id_card_number?: string
  education?: string
  major?: string
  certificate_no?: string
  classes: { class_id: number; class_name: string; title: string; title_label: string }[]
  status: number
  status_label: string
  join_date: string
  stats?: {
    total_growth_logs: number
    total_notices: number
    avg_parent_satisfaction: number
  }
  created_at: string
}

export interface TeacherInvitation {
  token: string
  status: number // 1=待使用 2=已使用
  expires_at: string
  created_at: string
}

export interface ParentItem {
  id: number
  nickname: string
  avatar: string
  phone: string
  gender: number
  children: {
    student_id: number
    student_name: string
    class_name: string
    relation: string
    relation_label: string
  }[]
  status: number
  status_label: string
  created_at: string
}

export interface NoticeItem {
  id: number
  category: string
  category_label: string
  title: string
  class_names: string[]
  creator: { id: number; nickname: string }
  status: number
  status_label: string
  read_rate: string
  published_at?: string
  created_at: string
}

export interface NoticeReadStats {
  read_count: number
  unread_count: number
  confirmed_count: number
  total_count: number
  read_rate: string
}

export interface NoticeDetail {
  id: number
  category: string
  category_label: string
  title: string
  content: string
  class_names: string[]
  need_confirm: boolean
  status: number
  published_at?: string
  attachments: { id: number; file_url: string; file_type: string; file_name: string }[]
  read_stats: NoticeReadStats
  creator: { id: number; nickname: string; avatar?: string }
  audit_status: string
  auditor?: { id: number; nickname: string; avatar?: string }
  audited_at?: string
  created_at: string
}

export interface NoticeAttachmentInput {
  file_url: string
  file_type?: string
  file_name?: string
}

export interface CreateNoticeBody {
  class_ids: number[]
  /** 年级圈选（小班/中班/大班），发布时展开为班级快照，与 class_ids 并集去重 */
  grades?: string[]
  category: string
  title: string
  content?: string
  need_confirm?: boolean
  scheduled_at?: string
  attachments?: NoticeAttachmentInput[]
}

export interface UpdateNoticeBody {
  /** 显式重选目标集（整体替换） */
  class_ids?: number[]
  grades?: string[]
  category?: string
  title?: string
  content?: string
  need_confirm?: boolean
  scheduled_at?: string | null
  status?: number
  attachments?: NoticeAttachmentInput[] | null
}

export interface AttendanceOverview {
  date: string
  total_students: number
  present: number
  leave: number
  absent: number
  attendance_rate: string
  by_class: {
    class_id: number
    class_name: string
    total: number
    present: number
    leave: number
    absent: number
    rate: string
  }[]
}

export interface LeaveItem {
  id: number
  student: { id: number; name: string; class_name: string }
  applicant: { nickname: string; relation: string }
  reason: string
  start_date: string
  end_date: string
  days: number
  status: number
  status_label: string
  reviewer?: { nickname: string }
  reviewed_at?: string
  created_at: string
}

export interface PickupOverview {
  date: string
  total_students: number
  arrived: number
  picked_up: number
  not_picked_up: number
  overdue_not_picked?: number
  abnormal_count: number
  by_class: {
    class_id: number
    class_name: string
    arrived: number
    picked_up: number
    not_picked_up: number
  }[]
}

export interface PickupRecordItem {
  id: number
  student: { id: number; name: string; class_name: string }
  type: number
  type_label: string
  method: number
  method_label: string
  pickup_name: string
  operator?: { nickname: string }
  is_abnormal: boolean
  recorded_at: string
}

export interface GrowthLogItem {
  id: number
  date: string
  class_name: string
  content_preview: string
  media_count: number
  likes_count: number
  comments_count: number
  creator: { id: number; nickname: string }
  audit_status: string
  created_at: string
}

/** 任务频率：与 server/internal/domain 的 TaskFrequency 一致 */
export type TaskFrequency = 'daily' | 'weekly' | 'once'
/** 星星统计范围：task=按任务目标，class=按班级罐目标（忽略星任务目标） */
export type TaskStarScope = 'task' | 'class'
/** 认可仪式 */
export type TaskRitual = 'sticker' | 'voice' | 'class_star'

/**
 * 公开状态的四种取值（服务端 task_rules.go 计算，前端不得自拼）。
 * none=从未申请，pending=待老师审核，approved=已批准且未撤回，revoked=已撤回。
 */
export type TaskPublicStatus = 'none' | 'pending' | 'approved' | 'revoked'

export interface TaskItem {
  id: number
  title: string
  class_name: string
  class_id: number
  learning_link: string
  target_student_id: number
  frequency: TaskFrequency
  frequency_label: string
  start_date: string
  end_date: string
  status: number
  allow_public: boolean
  /** 历史行可能为空串（早于仪式/范围字段的版本），保留 '' 以免调用方误以为一定有值 */
  recognition_ritual: TaskRitual | ''
  star_scope: TaskStarScope | ''
  star_goal: number
  /** 空串 = 独立任务；非空 = 多班活动，同活动下每班一条任务 */
  activity_id: string
  creator: { nickname: string }
  created_at: string
}

/**
 * 任务详情：列表字段 + 说明 + 累计口径指标。
 * 打开详情抽屉前必须取详情——列表接口不再返回每任务比率（7.4 下线）。
 */
export interface TaskDetail extends TaskItem {
  description: string
  /** 全部未删除提交数（不按时间窗过滤） */
  submission_count: number
  responded_count: number
  /** 主回应待回应 = submission_count - responded_count */
  pending_count: number
  /** "NN%" 或 null（无提交时无法计算，显示「—」而不是 0%） */
  response_rate: string | null
  participation_rate: string | null
  /** 发布时按家长去重的通知义务集；null = 未知（历史任务或收件人查询失败） */
  notify_target_count: number | null
  creator: { id: number; nickname: string; avatar: string }
}

/** 任务统计的一个聚合桶（全园 / 班级 / 活动 / 独立任务共用这套键名） */
export interface TaskStatMetrics {
  total_checkins: number
  /** 比率一律是 "NN%" 字符串或 null（分母为 0 时不可计算），不要 Number(null) */
  response_rate: string | null
  participation_rate: string | null
  pending_count: number
  pending_over_48h: number
  /** 仅 single 主回应的提交→回应时长中位数；无样本为 null */
  median_response_hours: number | null
  median_response_negative_excluded: number
  feedback_coverage: string | null
  child_happy_rate: string | null
  ack_rate: string | null
  notify_baseline_missing: number
}

export interface TaskStatsByClass extends TaskStatMetrics {
  class_id: number
  class_name: string
  tasks: number
}

/**
 * by_activity 的一行：activity_id 非空按活动聚合（task_id=0）；
 * 独立任务各占一行（activity_id 为空串，task_id 为任务 id），不合并成「空活动」。
 */
export interface TaskStatsByActivity extends TaskStatMetrics {
  activity_id: string
  task_id: number
  title: string
  tasks: number
}

export interface TaskStats extends TaskStatMetrics {
  period: { start: string; end: string }
  total_tasks: number
  active_tasks: number
  completed_tasks: number
  by_class: TaskStatsByClass[]
  by_activity: TaskStatsByActivity[]
}

/** 列表里的多班活动分组视图：同 activity_id 的各班任务折叠成一条 */
export interface ActivityTask {
  activity_id: string
  title: string
  /** 组内各班任务（展开后逐条显示） */
  tasks: TaskItem[]
}

/** 公开内容列表项（GET /admin/tasks/:id/checkins） */
export interface PublicCheckin {
  checkin_id: number
  task_id: number
  student_id: number
  student_name: string
  content: string
  media_url: string
  created_at: string
  is_public_requested: boolean
  public_scope_confirmed: boolean
  public_approved_at: string | null
  public_approved_by: number
  public_revoked_at: string | null
  public_status: TaskPublicStatus
  /**
   * 此刻是否真的在班级公开墙上可见。已批准也可能为 false
   * （任务结束/关闭公开/孩子转班），所以行里要显式标注而不是隐藏。
   */
  public_visible: boolean
}

/** 班级星星罐（当前学期） */
export interface ClassStarJar {
  class_id: number
  /** `<学年>-<学期>`，如 2031-2032-spring */
  term: string
  term_start: string
  term_end: string
  /** 未配置时为 0 */
  goal: number
  configured: boolean
  updated_at: string | null
}

export interface PointsOverview {
  total_points_in_circulation: number
  total_points_awarded_this_month: number
  total_points_spent_this_month: number
  top_earners: {
    student: { id: number; name: string; class_name: string }
    points: number
  }[]
}

export interface PointRecordItem {
  id: number
  student: { id: number; name: string; class_name: string }
  type: string
  source: string
  source_label: string
  amount: number
  description: string
  operator: unknown
  created_at: string
}

export interface AchievementItem {
  id: number
  name: string
  description: string
  icon_url: string
  category: string
  category_label: string
  condition: string
  earned_count: number
  status: number
  created_at: string
}

export interface AnnouncementItem {
  id: number
  title: string
  content_preview: string
  cover_url: string
  is_important: boolean
  is_top: boolean
  target_roles: string[]
  status: number
  status_label: string
  view_count: number
  creator: { nickname: string }
  published_at?: string
  created_at: string
}

export interface AnnouncementAttachmentItem {
  file_url: string
  file_name: string
  file_size: number
}

export interface AnnouncementDetail {
  id: number
  title: string
  content: string
  cover_url: string
  is_important: boolean
  is_top: boolean
  target_roles: string[]
  status: number
  published_at?: string
  attachments: AnnouncementAttachmentItem[]
  view_count: number
  creator: { id: number; nickname: string; avatar?: string }
  created_at: string
}

export interface AuditListItem {
  id: number
  type: string
  type_label: string
  reference_id: number
  title: string
  content_preview: string
  submitter: { id: number; nickname: string; class_name?: string }
  status: string
  status_label: string
  submitted_at: string
}

export interface AdminItem {
  id: number
  nickname: string
  avatar: string
  phone: string
  admin_role: { id: number; name: string; permissions_count: number }
  status: number
  status_label: string
  last_login_at?: string
  created_at: string
}

export interface RoleItem {
  id: number
  name: string
  description: string
  is_system: boolean
  admin_count: number
  permissions: string[]
  created_at: string
}

export interface PermissionModule {
  module: string
  module_label: string
  items: { key: string; label: string }[]
}

export interface OperationLogItem {
  id: number
  module: string
  module_label: string
  action: string
  action_label: string
  target: string
  operator: { id: number; nickname: string; ip: string }
  detail: Record<string, unknown>
  created_at: string
}

export interface SystemSettings {
  basic: {
    academic_year: string
    semester: string
    semester_start: string
    semester_end: string
  }
  attendance: {
    arrive_start_time: string
    arrive_end_time: string
    pickup_start_time: string
    late_threshold_minutes: number
  }
  notice: { need_audit: boolean; auto_remind_hours: number }
  growth: {
    need_audit: boolean
    allow_parent_comment: boolean
    allow_parent_like: boolean
  }
  pickup: { qr_code_expire_hours: number; need_authorization_audit: boolean }
  points: {
    task_checkin_points: number
    weekly_attendance_bonus: number
    monthly_attendance_bonus: number
  }
  health: {
    temperature_required: boolean
    fever_threshold: number
  }
}

// 字典管理
export interface DictTypeDetail {
  id: number
  type_key: string
  type_name: string
  description: string
  is_system: boolean
  sort_order: number
  item_count: number
  created_at: string
}

// 晨检统计
export interface HealthStatsClassRow {
  class_id: number; class_name: string
  total: number; pending: number; completed: number
  healthy: number; observe: number; reject: number
}
export interface HealthSymptomCount { tag: string; count: number }
export interface HealthStats {
  date: string; by_class: HealthStatsClassRow[]
  symptoms: HealthSymptomCount[]; total_students: number
}

export interface DictItemDetail {
  id: number
  type_id: number
  item_key: string
  item_label: string
  extra: Record<string, unknown> | null
  sort_order: number
  is_enabled: boolean
}

export interface ExportTaskStatus {
  task_id: string
  status: string
  progress?: number
  download_url?: string
  file_name?: string
  file_size?: number
  expire_at?: string
  created_at?: string
  completed_at?: string
  message?: string
}

// 园级日历（align-class-calendar）
export interface GardenCalendarEventItem {
  id: number
  title: string
  description: string
  event_type: string
  event_type_label: string
  scope: 'class' | 'kindergarten'
  status: 'draft' | 'published'
  event_date: string
  start_time: string | null
  end_time: string | null
  location: string
  is_all_day: boolean
  /** 数据驱动色（来源：字典 calendar_event_type 的 Extra） */
  color: string
}

export interface GardenCalendarListResult {
  list: GardenCalendarEventItem[]
  total: number
}

export interface GardenCalendarInput {
  title: string
  description: string
  event_type: string
  event_date: string
  start_time: string
  end_time: string
  location: string
  is_all_day: boolean
  status: 'draft' | 'published'
  notify: boolean
}

// 课表（align-class-calendar）
export interface TimetableEntry {
  weekday: number
  period: number
  title: string
  location: string
  start_time: string
  end_time: string
}

export interface TimetableOverride extends TimetableEntry {
  action: 'replace' | 'remove' | 'add'
}

export interface TimetableTemplateResult {
  grade: string
  entries: TimetableEntry[]
}

export interface TimetableClassOverridesResult {
  class_id: number
  overrides: TimetableOverride[]
}

export interface TimetableSyncResult {
  synced: boolean
  created: number
  updated: number
  deleted: number
  skipped_overridden: number
}
