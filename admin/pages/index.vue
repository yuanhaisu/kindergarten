<template>
  <div class="space-y-6">
    <!-- 统计卡片行 -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <StatCard
        title="总学生数"
        :value="stats.totalStudents"
        :icon="UserFilled"
        icon-bg="bg-blue-50"
        icon-color="#3B82F6"
        :trend="stats.totalStudentsTrend"
        trend-direction="up"
        trend-label="较昨日"
      />
      <StatCard
        title="今日出勤"
        :value="stats.todayAttendance"
        :icon="Tickets"
        icon-bg="bg-emerald-50"
        icon-color="#10B981"
        :trend="stats.attendanceTrend"
        :trend-direction="stats.attendanceTrendDir"
        trend-label="较昨日"
      />
      <StatCard
        title="通知已读率"
        :value="stats.noticeReadRate"
        :icon="Document"
        icon-bg="bg-violet-50"
        icon-color="#8B5CF6"
        :trend="stats.noticeTrend"
        :trend-direction="stats.noticeTrendDir"
        trend-label="较上周"
        format="percent"
      />
      <StatCard
        title="任务完成率"
        :value="stats.taskCompletionRate"
        :icon="List"
        icon-bg="bg-amber-50"
        icon-color="#F59E0B"
        :trend="stats.taskTrend"
        :trend-direction="stats.taskTrendDir"
        trend-label="较昨日"
        format="percent"
      />
    </div>

    <!-- 图表行：出勤趋势 + 请假原因分布 -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
      <!-- 出勤趋势图 -->
      <el-card class="lg:col-span-2" shadow="never">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="text-base font-semibold text-slate-800">出勤趋势</span>
            <div class="flex gap-1">
              <el-button
                :type="trendPeriod === '7d' ? 'primary' : 'default'"
                size="small"
                @click="switchTrendPeriod('7d')"
              >
                周
              </el-button>
              <el-button
                :type="trendPeriod === '30d' ? 'primary' : 'default'"
                size="small"
                @click="switchTrendPeriod('30d')"
              >
                月
              </el-button>
            </div>
          </div>
        </template>
        <AttendanceTrendChart :data="trendData" />
      </el-card>

      <!-- 请假原因分布 -->
      <el-card shadow="never">
        <template #header>
          <span class="text-base font-semibold text-slate-800">请假原因分布</span>
        </template>
        <LeaveReasonPieChart :data="leaveReasons" :total="leaveTotal" />
      </el-card>
    </div>

    <!-- 底部行：班级活跃度 + 待办事项 -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <!-- 班级活跃度排名 -->
      <el-card shadow="never">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="text-base font-semibold text-slate-800">班级活跃度排名</span>
            <el-button size="small" text type="primary">查看全部</el-button>
          </div>
        </template>
        <ClassRankingTable :list="classActivity" />
      </el-card>

      <!-- 待办事项 -->
      <el-card shadow="never">
        <template #header>
          <div class="flex items-center justify-between">
            <span class="text-base font-semibold text-slate-800">待办事项</span>
            <el-tag size="small" type="danger" round>{{ todos.length }}条待处理</el-tag>
          </div>
        </template>
        <TodoList :items="todos" />
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Document, List, Tickets, UserFilled } from '@element-plus/icons-vue'
import type {
  AttendanceTrendPoint,
  ClassActivityItem,
  DashboardOverview,
  LeaveReasonItem,
  TodoItem,
} from '~/types/api'
import {
  getAttendanceTrend,
  getClassActivity,
  getClassRanking,
  getDashboardOverview,
  getLeaveReasons,
  getTodoList,
} from '~/api/dashboard'

// ---- 统计卡片数据 ----
const stats = reactive({
  totalStudents: 0,
  totalStudentsTrend: '+0%',
  todayAttendance: 0,
  attendanceTrend: '0%',
  attendanceTrendDir: 'flat' as 'up' | 'down' | 'flat',
  noticeReadRate: '0%',
  noticeTrend: '0%',
  noticeTrendDir: 'flat' as 'up' | 'down' | 'flat',
  taskCompletionRate: '0%',
  taskTrend: '0%',
  taskTrendDir: 'flat' as 'up' | 'down' | 'flat',
})

// ---- 出勤趋势 ----
const trendPeriod = ref<'7d' | '30d'>('30d')
const trendData = ref<AttendanceTrendPoint[]>([])

// ---- 请假原因 ----
const leaveReasons = ref<LeaveReasonItem[]>([])
const leaveTotal = ref(0)

// ---- 班级活跃度 ----
const classActivity = ref<ClassActivityItem[]>([])

// ---- 待办事项 ----
const todos = ref<TodoItem[]>([])

// ---- 数据加载 ----
function mapStats(overview: DashboardOverview) {
  const rt = overview.realtime
  stats.totalStudents = rt.total_student_count || 0
  // 如果后端的 total_student_count 可用则使用，否则使用出勤+请假数作为估算
  if (!stats.totalStudents) {
    stats.totalStudents = (rt.present_count || 0) + (rt.leave_count || 0)
  }
  stats.totalStudentsTrend = '+0%' // 需后端支持
  stats.todayAttendance = rt.present_count || 0
  stats.attendanceTrend = rt.attendance_rate || '0%'
  stats.attendanceTrendDir = 'flat'
  stats.noticeReadRate = '0%' // 需后端支持
  stats.noticeTrend = '0%'
  stats.noticeTrendDir = 'flat'
  stats.taskCompletionRate = '0%' // 需后端支持
  stats.taskTrend = '0%'
  stats.taskTrendDir = 'flat'
}

function mapTodos(overview: DashboardOverview): TodoItem[] {
  const t = overview.today
  if (!t) return []
  const items: TodoItem[] = []
  if (t.new_notices) {
    items.push({
      id: 1,
      title: `${t.new_notices} 条新通知待查看`,
      description: '有新的班级通知需要审核和发布',
      time_ago: '今天',
      action_type: 'view',
    })
  }
  if (t.pending_leaves) {
    items.push({
      id: 2,
      title: `${t.pending_leaves} 条请假申请待审核`,
      description: '家长提交的请假申请需要及时处理',
      time_ago: '今天',
      action_type: 'audit',
    })
  }
  if (t.growth_logs) {
    items.push({
      id: 3,
      title: `${t.growth_logs} 条成长记录待发布`,
      description: '老师提交的成长记录需要审核',
      time_ago: '今天',
      action_type: 'audit',
    })
  }
  if (t.new_tasks) {
    items.push({
      id: 4,
      title: `${t.new_tasks} 个新任务待分配`,
      description: '亲子任务需要分配给相应班级',
      time_ago: '今天',
      action_type: 'view',
    })
  }
  return items
}

async function loadTrend(period: '7d' | '30d') {
  try {
    const res = await getAttendanceTrend(period)
    trendData.value = res.data || []
  } catch {
    trendData.value = []
  }
}

async function loadLeaveReasons() {
  try {
    const res = await getLeaveReasons('30d')
    leaveReasons.value = res.reasons || []
    leaveTotal.value = res.total || 0
  } catch {
    // 后端可能暂不支持，使用空数据
    leaveReasons.value = []
    leaveTotal.value = 0
  }
}

async function loadClassActivity() {
  try {
    const res = await getClassActivity()
    classActivity.value = res.list || []
  } catch {
    // 回退：使用 getClassRanking 并映射数据
    try {
      const res = await getClassRanking()
      classActivity.value = (res.list || []).map((item) => ({
        class_id: item.class_id,
        class_name: item.class_name,
        activity_rate: parseFloat(item.rate) || 0,
        trend: 'flat' as const,
        activity_score: item.present,
      }))
    } catch {
      classActivity.value = []
    }
  }
}

function switchTrendPeriod(period: '7d' | '30d') {
  trendPeriod.value = period
  loadTrend(period)
}

onMounted(async () => {
  try {
    const overview = await getDashboardOverview()
    mapStats(overview)
    todos.value = mapTodos(overview)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载看板失败')
  }

  await Promise.all([
    loadTrend(trendPeriod.value),
    loadLeaveReasons(),
    loadClassActivity(),
  ])
})
</script>
