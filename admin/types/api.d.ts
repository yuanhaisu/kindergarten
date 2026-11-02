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

// ===== 膳食管理（add-quantified-recipe；契约 backend/api/api-admin/23-diet.md） =====

/** 膳食接口错误：useHttp 把 HTTP 状态挂在 error.status 上，404/409/422 由契约定义业务含义 */
export interface DietApiError extends Error {
  status?: number
}

export type DietMealKey =
  | 'breakfast'
  | 'morning_snack'
  | 'lunch'
  | 'afternoon_snack'
  | 'dinner'
  | 'evening_snack'

export type DietCareType = 'day_care' | 'full_care'
export type DietRatioMode = 'standard' | 'custom'
/** 重量口径：本变更仅支持市品重量 */
export type DietFoodInputType = 'market_weight'
export type DietCreatedMethod = 'custom' | 'template' | 'copy'
export type DietFoodSource = 'system' | 'custom'
export type DietSourceFlag = 'animal' | 'plant' | 'milk' | 'meat'

export type DietGroup16 =
  | 'grain'
  | 'tuber'
  | 'vegetable'
  | 'fruit'
  | 'meat'
  | 'egg'
  | 'aquatic'
  | 'soybean'
  | 'dairy'
  | 'oil'
  | 'nut'
  | 'instant_food'
  | 'beverage'
  | 'confection'
  | 'condiment'
  | 'other'

/**
 * 27 项营养（每 100g 可食部浓度）。缺测为 null —— 展示「—」，
 * 任何合计都不得把 null 当 0（契约「营养字段」一节）。
 */
export interface DietNutrients {
  energy: number | null
  heat: number | null
  protein: number | null
  fats: number | null
  carb: number | null
  dietary_fiber: number | null
  ash: number | null
  water: number | null
  cholesterol: number | null
  retinol: number | null
  carotin: number | null
  vitamin_a: number | null
  vitamin_b1: number | null
  vitamin_b2: number | null
  nicotinic_acid: number | null
  vitamin_c: number | null
  vitamin_e: number | null
  calcium: number | null
  phosphorus: number | null
  potassium: number | null
  sodium: number | null
  magnesium: number | null
  iron: number | null
  zinc: number | null
  selenium: number | null
  copper: number | null
  manganese: number | null
}

/** 食物列表页展示的核心 15 项营养 */
export interface DietFoodCoreNutrients {
  energy: number | null
  protein: number | null
  fats: number | null
  carb: number | null
  retinol: number | null
  carotin: number | null
  calcium: number | null
  iron: number | null
  zinc: number | null
  phosphorus: number | null
  nicotinic_acid: number | null
  vitamin_a: number | null
  vitamin_b1: number | null
  vitamin_b2: number | null
  vitamin_c: number | null
}

/** 周食谱明细的营养快照：27 项浓度 + 食部 */
export interface DietNutritionSnapshot extends DietNutrients {
  edible_percent: number | null
}

// ----- 集体对象（基本参数） -----

export interface DietMealConfigItem {
  meal_key: DietMealKey
  name: string
  time: string
  energy_percent: number
  sort: number
  enabled: boolean
}

export interface DietMealConfigInput {
  meal_key: DietMealKey
  time: string
  energy_percent: number
  enabled: boolean
}

/** 就餐班级：班级是动态数据，对象只存 class_id，读取时回查当前班级表 */
export interface DietGroupObjectClass {
  id: number
  name: string
  grade: string
}

/** 列表行只含渲染所需字段；时间、占比、比例模式在详情才返回（契约 §一） */
export interface DietGroupObjectListItem {
  id: number
  name: string
  academic_year_id: number
  care_type: DietCareType
  care_type_label: string
  food_input_type: DietFoodInputType
  food_input_type_label: string
  policy_name: string
  /** 只含已启用餐次，按固定餐次顺序排列 */
  meal_names: string[]
  classes: DietGroupObjectClass[]
}

export interface DietGroupObjectListQuery {
  /** 缺省取当前学年 */
  academic_year_id?: number
  keyword?: string
  page?: number
  page_size?: number
}

export interface DietGroupObjectListResult extends PageResult<DietGroupObjectListItem> {}

export interface DietGroupObjectDetail extends DietGroupObjectListItem {
  ratio_mode: DietRatioMode
  ratio_mode_label: string
  /** 只读：已启用餐次 energy_percent 之和 ÷ 100 */
  coefficient: number
  /** 全部 6 个餐次（含未启用） */
  meals: DietMealConfigItem[]
}

export interface DietGroupObjectInput {
  academic_year_id: number
  name: string
  care_type: DietCareType
  food_input_type: DietFoodInputType
  ratio_mode: DietRatioMode
  class_ids: number[]
  meals: DietMealConfigInput[]
}

// ----- 食物分类 -----

export interface DietFoodCategory {
  id: number
  parent_id: number
  level: 1 | 2
  name: string
  children?: DietFoodCategory[]
}

// ----- 食物库 -----

export interface DietFoodListItem extends DietFoodCoreNutrients {
  id: number
  name: string
  spell_code: string
  category_id: number
  category_name: string
  source: DietFoodSource
  edible_percent: number | null
}

export interface DietFoodListQuery {
  keyword?: string
  /** 一级分类（含其全部二级）或二级分类 */
  category_id?: number
  page?: number
  page_size?: number
}

export interface DietFoodListResult extends PageResult<DietFoodListItem> {}

export interface DietFoodDetail {
  id: number
  name: string
  spell_code: string
  category_id: number
  category_name: string
  category_parent_id: number
  source: DietFoodSource
  edible_percent: number | null
  group12: number | null
  group16: DietGroup16 | null
  source_flags: DietSourceFlag[]
  nutrients: DietNutrients
}

export interface DietFoodCreateBody {
  name: string
  /** 叶子分类 */
  category_id: number
  edible_percent: number
  group12?: number | null
  group16?: DietGroup16 | null
  source_flags?: DietSourceFlag[]
  /** 只传有值的键；未传的键写 NULL（不写 0） */
  nutrients?: Partial<DietNutrients>
}

/** PUT 字段均可选；nutrients 一旦提交即整体替换 */
export type DietFoodUpdateBody = Partial<DietFoodCreateBody>

export interface DietFoodBatchDeleteBody {
  ids: number[]
}

export type DietFoodBatchDeleteSkipReason = 'referenced_by_recipe' | 'not_found'

export interface DietFoodBatchDeleteSkipped {
  id: number
  name: string
  reason: DietFoodBatchDeleteSkipReason
}

export interface DietFoodBatchDeleteResult {
  deleted: number
  skipped: DietFoodBatchDeleteSkipped[]
}

export interface DietReplaceCandidate {
  id: number
  name: string
  spell_code: string
  category_id: number
  category_name: string
}

export interface DietReplaceCandidateQuery {
  keyword?: string
  limit?: number
}

export interface DietReplaceCandidateResult {
  list: DietReplaceCandidate[]
}

// ----- 菜谱库 -----

export interface DietRecipeListItem {
  id: number
  name: string
  spell_code: string
  item_count: number
}

export interface DietRecipeListQuery {
  keyword?: string
  page?: number
  page_size?: number
}

export interface DietRecipeListResult extends PageResult<DietRecipeListItem> {}

export interface DietRecipeSaveItem {
  food_id: number
  grams: number
}

export interface DietRecipeCreateBody {
  name: string
  items: DietRecipeSaveItem[]
  /** 同名时 false → 409（前端弹「替换 / 仅本次」）；true → 覆盖原配方 */
  replace?: boolean
}

export interface DietRecipeDetailItem {
  food_id: number
  food_name: string
  grams: number
  sort: number
  /** 该食物当前的 27 项营养（每 100g 可食部） */
  nutrients: DietNutrients
}

export interface DietRecipeDetail {
  id: number
  name: string
  spell_code: string
  items: DietRecipeDetailItem[]
}

export interface DietRecipeUpdateBody {
  name?: string
  /** 提交即整体替换 */
  items?: DietRecipeSaveItem[]
}

export interface DietRecipeMergeBody {
  source_id: number
  target_id: number
}

export interface DietRecipeMergeResult {
  target_id: number
  merged_item_count: number
}

export interface DietRecipeSuggestQuery {
  keyword?: string
  limit?: number
}

export interface DietRecipeSuggestResult {
  list: DietRecipeListItem[]
}

// ----- 周食谱 -----

export interface DietPlanListItem {
  id: number
  name: string
  academic_year_id: number
  group_object_id: number
  /** 回查当前对象名称；旧行 group_object_id=0 时为空串 */
  group_object_name: string
  start_date: string
  end_date: string
  day_num: number
  week_of_year: number
  created_method: DietCreatedMethod
  created_method_label: string
}

export interface DietPlanListQuery {
  /** 缺省取当前学年 */
  academic_year_id?: number
  keyword?: string
  page?: number
  page_size?: number
}

export interface DietPlanListResult extends PageResult<DietPlanListItem> {}

export interface DietPlanCreateBase {
  name: string
  academic_year_id: number
  /** 必填；对象的学年必须与 academic_year_id 一致（前端从对象带出学年） */
  group_object_id: number
  /** 周一 */
  start_date: string
  /** = start_date + 6 天 */
  end_date: string
  /** 仅展示，不参与计算 */
  week_of_year?: number
}

export type DietPlanCreateBody = DietPlanCreateBase &
  (
    | { created_method: 'custom' }
    | { created_method: 'template'; template_id: number }
    | { created_method: 'copy'; source_plan_id: number }
  )

export interface DietPlanCreateResult {
  id: number
  name: string
  start_date: string
  end_date: string
  day_num: number
  week_of_year: number
  warnings: string[]
}

export interface DietDriTarget {
  nutrient_key: string
  label: string
  unit: string
  value: number
}

export interface DietDriWeight {
  age_band: number
  weight: number
}

export interface DietPlanConfigSnapshot {
  group_object_id: number
  group_object_name: string
  /** 创建时固化的就餐班级，供复核 DRIs 加权范围 */
  group_object_class_ids: number[]
  care_type: DietCareType
  food_input_type: DietFoodInputType
  ratio_mode: DietRatioMode
  coefficient: number
  meals: DietMealConfigItem[]
  dri_targets: DietDriTarget[]
  dri_weights: DietDriWeight[]
}

export interface DietPlanItemInput {
  date: string
  meal_key: DietMealKey
  recipe_name: string
  recipe_id?: number | null
  food_id: number
  /** 客户端传值仅作旧数据兜底，服务端以食物库名称为准 */
  food_name?: string
  grams: number
}

export interface DietPlanItem extends DietPlanItemInput {
  id: number
  /** 由 date 推导：1=周一 … 7=周日 */
  weekday: number
  sort: number
  nutrition_snapshot: DietNutritionSnapshot
}

export interface DietPlanDetail {
  id: number
  name: string
  academic_year_id: number
  /** 名称优先取创建快照：对象改名后旧食谱仍显示原名 */
  group_object_id: number
  group_object_name: string
  start_date: string
  end_date: string
  day_num: number
  week_of_year: number
  created_method: DietCreatedMethod
  created_method_label: string
  source_template_id: number | null
  source_plan_id: number | null
  config_snapshot: DietPlanConfigSnapshot
  items: DietPlanItem[]
}

export interface DietPlanSaveBody {
  name?: string
  items: DietPlanItemInput[]
}

export interface DietPlanSaveResult {
  id: number
  day_num: number
  item_count: number
}

export interface DietPlanClearResult {
  id: number
  day_num: number
}

// ----- 模板食谱 -----

export interface DietTemplateListItem {
  id: number
  name: string
  day_num: number
  meal_keys: DietMealKey[]
  item_count: number
}

export interface DietTemplateListResult {
  list: DietTemplateListItem[]
}

export interface DietTemplateItem {
  weekday: number
  meal_key: DietMealKey
  recipe_name: string
  food_id: number
  food_name: string
  grams: number
  sort: number
}

export interface DietTemplateDetail {
  id: number
  name: string
  day_num: number
  meal_keys: DietMealKey[]
  items: DietTemplateItem[]
}

// ----- 营养分析（7 组表 + 结论） -----

export interface DietAnalysisPlan {
  id: number
  name: string
  start_date: string
  end_date: string
  day_num: number
  coefficient: number
}

export type DietNutrientComment = '低' | '合格' | '高'

export interface DietNutritionRow {
  key: string
  label: string
  unit: string
  intake: number
  dri: number
  compare_percent: number
  comment: DietNutrientComment
}

export interface DietAnalysisNutrition {
  items: DietNutritionRow[]
}

export type DietMacroEnergyRowKey = 'kcal' | 'kj' | 'percent' | 'grams_ratio'

export interface DietMacroEnergyRow {
  key: DietMacroEnergyRowKey
  label: string
  protein: number
  fats: number
  carb: number
  recommended?: { protein: string; fats: string; carb: string }
}

export interface DietAnalysisMacroEnergy {
  denominator: number
  rows: DietMacroEnergyRow[]
}

export interface DietMealEnergyItem {
  meal_key: DietMealKey
  label: string
  energy: number
  percent: number
}

export interface DietAnalysisMealEnergy {
  meals: DietMealEnergyItem[]
  total_percent: number
}

export interface DietCompositionGroup {
  key: DietGroup16
  label: string
  grams: number
  /** 页面只渲染 true 的 10 类 */
  displayed: boolean
}

export interface DietAnalysisComposition {
  groups: DietCompositionGroup[]
}

export interface DietCalciumIronSource {
  key: string
  label: string
  value: number
  percent: number
}

export interface DietAnalysisCalciumIron {
  calcium: { total: number; sources: DietCalciumIronSource[] }
  iron: { total: number; sources: DietCalciumIronSource[] }
}

export interface DietProteinSourceItem {
  key: string
  label: string
  value: number
  percent: number
}

export interface DietAnalysisProteinSource {
  total: number
  items: DietProteinSourceItem[]
  animal_bean_percent: number
}

export interface DietFatSourceItem extends DietCalciumIronSource {
  displayed: boolean
}

export interface DietAnalysisFatSource {
  total: number
  items: DietFatSourceItem[]
}

export interface DietSuggestion {
  nutrient_key: string
  level: 'low' | 'high'
  text: string
}

export interface DietConclusion {
  food_kind_count: number
  condiment_kind_count: number
  non_condiment_kind_count: number
  dri_percent: Record<string, number>
  macro_percent: { protein: number; fats: number; carb: number }
  macro_ratio: string
  animal_bean_protein_percent: number
  suggestions: DietSuggestion[]
}

export interface DietAnalysis {
  plan: DietAnalysisPlan
  nutrition: DietAnalysisNutrition
  macro_energy: DietAnalysisMacroEnergy
  meal_energy: DietAnalysisMealEnergy
  composition: DietAnalysisComposition
  calcium_iron: DietAnalysisCalciumIron
  protein_source: DietAnalysisProteinSource
  fat_source: DietAnalysisFatSource
  conclusion: DietConclusion
}
