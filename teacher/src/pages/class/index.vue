<script setup>
import { ref, computed } from "vue"
import { onLoad, onPullDownRefresh } from "@dcloudio/uni-app"
import { fetchTeacherClassInfo, fetchTeacherClassStudents } from "@/api/teacher-class"
import { fetchTeacherPickupOverview } from "@/api/teacher-pickup"
import { fetchTeacherNotices } from "@/api/teacher-notices"
import BottomNav from "@/components/BottomNav.vue"

const info = ref({})
const students = ref([])
const pickupStat = ref({})
const loading = ref(false)
const loadError = ref("")
const searchKeyword = ref("")
const activeFilter = ref("全部")
const recentNotices = ref([])

// 后端 /teacher/class/students 复用接送查询（ClassStudents → PickupStudents），
// 返回的是真实接送状态，**没有出勤字段**。这里只展示有数据支撑的字段，
// 不再按下标编造出勤。
const PICKUP_STATUS_LABEL = { not_arrived: "未入园", not_picked_up: "未接走", picked_up: "已接走" }
const PICKUP_STATUS_CLASS = { not_arrived: "status-absent", not_picked_up: "status-late", picked_up: "status-present" }

const filters = ["全部", "未入园", "未接走", "已接走"]

const filteredStudents = computed(() => {
  let list = students.value
  if (searchKeyword.value) {
    const kw = searchKeyword.value.toLowerCase()
    // 只按姓名搜：后端不返回 student_no，搜它恒为空
    list = list.filter((s) => (s.name || "").toLowerCase().includes(kw))
  }
  if (activeFilter.value !== "全部") {
    list = list.filter((s) => (PICKUP_STATUS_LABEL[s.pickup_status] || "") === activeFilter.value)
  }
  return list
})

function onSearchInput(e) { searchKeyword.value = e.detail.value }
function switchFilter(f) { activeFilter.value = f }
function openNotices() { uni.navigateTo({ url: "/pages/messages/index" }) }

function quickAction(type) {
  const routes = { notice: "/pages/notice/create", growth: "/pages/growth/create", pickup: "/pages/pickup/index" }
  uni.navigateTo({ url: routes[type] })
}

function formatDate() { const d = new Date(); return `${d.getMonth() + 1}月${d.getDate()}日` }

async function loadClass() {
  loading.value = true
  loadError.value = ""
  try {
    const [infoRes, studentsRes, statRes] = await Promise.all([
      fetchTeacherClassInfo(),
      fetchTeacherClassStudents(),
      fetchTeacherPickupOverview(),
    ])
    info.value = infoRes.data || {}
    students.value = studentsRes.data?.list || []
    pickupStat.value = statRes.data || {}
  } catch (err) {
    // 之前这里是静默 catch + 硬编码兜底，接口挂了会渲染出一份看起来很真的假班级
    loadError.value = err?.message || "班级数据加载失败"
    students.value = []
  } finally {
    loading.value = false
    uni.stopPullDownRefresh()
  }
  // 通知是非关键数据，单独 catch —— 它失败不该让整个花名册变空
  try {
    const noticeRes = await fetchTeacherNotices({ page: 1, page_size: 2 })
    recentNotices.value = (noticeRes.data?.list || []).map((n) => ({
      id: n.id,
      title: n.title,
      // 后端给的是 "85%" 这样的字符串；totalCount 为 0 时会是空串
      readRate: parseInt(n.read_rate, 10) || 0,
      time: (n.published_at || n.created_at || "").slice(5, 10),
    }))
  } catch {
    recentNotices.value = []
  }
}

const className = computed(() => info.value.name || "")
const studentCount = computed(() => info.value.student_count || students.value.length)
const teacherName = computed(() => (info.value.teacher && info.value.teacher.teacher_name) || "")
const kindergartenName = computed(() => (info.value.teacher && info.value.teacher.kindergarten_name) || "")
const academicInfo = computed(() => info.value.academic_year || "")
const todaySnapshot = computed(() => ({
  arrived: pickupStat.value.arrived || 0,
  not_arrived: pickupStat.value.not_arrived || 0,
  picked_up: pickupStat.value.picked_up || 0,
  not_picked_up: pickupStat.value.not_picked_up || 0,
}))

onLoad(loadClass)
onPullDownRefresh(loadClass)
</script>

<template>
  <view class="page">
    <view class="header">
      <view class="header-top">
        <view class="class-title-row">
          <view class="class-icon-box"><text class="class-icon">&#xf1ae;</text></view>
          <view class="class-name-col">
            <view class="class-name-row"><text class="class-name">{{ className }}</text><text class="edit-icon">&#xea6e;</text></view>
            <text class="class-subtitle">{{ academicInfo }}</text>
          </view>
        </view>
        <view class="header-actions">
          <view class="hdr-btn" @click="openNotices"><text class="hdr-btn-icon">&#xef94;</text><view class="hdr-badge" /></view>
        </view>
      </view>
    </view>

    <scroll-view scroll-y class="content-scroll" :refresher-enabled="true" @refresherrefresh="loadClass">
      <view class="info-card">
        <view class="info-head">
          <view class="info-school-icon"><text class="info-school-icon-text">&#xf036;</text></view>
          <view class="info-head-text"><text class="info-head-title">{{ kindergartenName }} · {{ className }}</text><text class="info-head-sub">{{ academicInfo }}</text></view>
        </view>
        <view class="info-divider" />
        <view class="info-row">
          <view class="info-item">
            <view class="info-item-avatar"><text class="info-item-avatar-text">师</text></view>
            <view class="info-item-text"><text class="info-item-label">班主任</text><text class="info-item-value">{{ teacherName }}</text></view>
          </view>
          <view class="info-v-divider" />
          <view class="info-item">
            <view class="info-item-icon-box"><text class="info-item-icon-text">&#xf1ae;</text></view>
            <view class="info-item-text"><text class="info-item-label">学生总数</text><text class="info-item-value">{{ studentCount }}人</text></view>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-head"><text class="section-title">今日快照</text><text class="section-date">{{ formatDate() }}</text></view>
        <view class="snapshot-row">
          <view class="snap-item"><text class="snap-num green">{{ todaySnapshot.arrived }}</text><text class="snap-unit">人</text><text class="snap-label">已入园</text></view>
          <view class="snap-item"><text class="snap-num yellow">{{ todaySnapshot.not_arrived }}</text><text class="snap-unit">人</text><text class="snap-label">未入园</text></view>
          <view class="snap-item"><text class="snap-num orange">{{ todaySnapshot.picked_up }}</text><text class="snap-unit">人</text><text class="snap-label">已接走</text></view>
          <view class="snap-item"><text class="snap-num blue">{{ todaySnapshot.not_picked_up }}</text><text class="snap-unit">人</text><text class="snap-label">未接走</text></view>
        </view>
      </view>

      <view class="section">
        <view class="section-head"><text class="section-title">班级动态</text></view>
        <view class="quick-row">
          <view class="quick-btn" @click="quickAction('notice')"><text class="quick-icon">&#xef94;</text><text class="quick-label">发布通知</text></view>
          <view class="quick-btn" @click="quickAction('growth')"><text class="quick-icon">&#xec29;</text><text class="quick-label">记成长</text></view>
          <view class="quick-btn" @click="quickAction('pickup')"><text class="quick-icon">&#xeb39;</text><text class="quick-label">接送</text></view>
        </view>
        <view class="sub-title-row"><text class="sub-title">最新通知</text></view>
        <view v-for="notice in recentNotices" :key="notice.id" class="notice-card">
          <view class="notice-icon-box"><text class="notice-icon">&#xef94;</text></view>
          <view class="notice-body"><text class="notice-title">{{ notice.title }}</text><text class="notice-time">{{ notice.time }}</text></view>
          <view class="notice-rate">
            <view class="rate-bar-bg"><view class="rate-bar-fill" :style="{ width: notice.readRate + '%' }" /></view>
            <text class="rate-text">已读 {{ notice.readRate }}%</text>
          </view>
        </view>
      </view>

      <view class="section">
        <view class="section-head">
          <text class="section-title">学生花名册</text>
        </view>
        <view class="search-filter-row">
          <view class="search-box">
            <text class="search-icon">&#xea76;</text>
            <input class="search-input" placeholder="搜索学生姓名" placeholder-style="color:#94A3B8;font-size: $typography-font-size-base" :value="searchKeyword" @input="onSearchInput" />
          </view>
          <scroll-view scroll-x class="filter-scroll" :show-scrollbar="false">
            <view class="filter-row">
              <view v-for="f in filters" :key="f" class="filter-pill" :class="{ active: activeFilter === f }" @click="switchFilter(f)"><text>{{ f }}</text></view>
            </view>
          </scroll-view>
        </view>
        <view v-if="loading" class="state-wrap"><text class="state-text">加载中...</text></view>
        <view v-else-if="loadError" class="state-wrap">
          <text class="state-text">{{ loadError }}</text>
          <view class="retry-btn" @click="loadClass">重试</view>
        </view>
        <view v-else-if="filteredStudents.length === 0" class="state-wrap"><text class="state-text">暂无匹配学生</text></view>
        <view v-else class="student-list">
          <view v-for="student in filteredStudents" :key="student.student_id" class="student-card">
            <image v-if="student.avatar" class="stu-avatar" :src="student.avatar" mode="aspectFill" />
            <view v-else class="stu-avatar fallback"><text>{{ (student.name || "学").slice(0, 1) }}</text></view>
            <view class="stu-info"><view class="stu-name-row"><text class="stu-name">{{ student.name }}</text><text class="stu-no">{{ student.arrive_time ? student.arrive_time + " 入园" : "" }}</text></view></view>
            <view class="stu-status" :class="PICKUP_STATUS_CLASS[student.pickup_status]"><view class="status-dot" /><text class="status-text">{{ PICKUP_STATUS_LABEL[student.pickup_status] || "—" }}</text></view>
          </view>
        </view>
      </view>
      <view class="bottom-safe" />
    </scroll-view>
    <BottomNav :active="1" />
  </view>
</template>

<style scoped lang="scss">
// 本页图标是 Remix Icon 的 PUA 码位（&#xef94; 等）。不指定 font-family 时
// 小程序没有可渲染该码位的字体，整页图标会显示成方块。
// 注：U+F1AE 经实测不是有效字形（落成横杠），仅「接送」已改用家长端验证过的
// U+EB39；班级头像与学生总数两处仍待定，见视觉审计的图标体系决定。
.class-icon, .edit-icon, .hdr-btn-icon, .info-school-icon-text,
.info-item-icon-text, .quick-icon, .notice-icon, .search-icon {
  font-family: remixicon;
}

.page { min-height: 100vh; background: $color-bg-page; display: flex; flex-direction: column; box-sizing: border-box; overflow-x: hidden; }

.header { background: linear-gradient(135deg, $color-semantic-info 0%, $color-semantic-info 100%); padding: 96rpx 48rpx 48rpx; box-sizing: border-box; width: 100%; }
.header-top { display: flex; align-items: center; justify-content: space-between; }
.class-title-row { display: flex; align-items: center; gap: $spacing-lg; }
.class-icon-box { width: 96rpx; height: 96rpx; border-radius: $border-radius-full; background: rgba(255,255,255,0.2); display: flex; align-items: center; justify-content: center; }
.class-icon { font-size: 48rpx; color: #fff; }
.class-name-col { display: flex; flex-direction: column; gap: $spacing-xs; }
.class-name-row { display: flex; align-items: center; gap: $spacing-sm; }
.class-name { font-size: $typography-font-size-2xl; font-weight: 800; color: #fff; }
.edit-icon { font-size: $typography-font-size-3xl; color: rgba(255,255,255,0.7); }
.class-subtitle { font-size: $typography-font-size-sm; color: rgba(255,255,255,0.7); }
.header-actions { display: flex; align-items: center; gap: $spacing-md; }
.hdr-btn { width: 80rpx; height: 80rpx; border-radius: $border-radius-full; background: rgba(255,255,255,0.2); display: flex; align-items: center; justify-content: center; position: relative; }
.hdr-btn-icon { font-size: 44rpx; color: #fff; }
.hdr-badge { position: absolute; top: 14rpx; right: 16rpx; width: 18rpx; height: 16rpx; background: $color-semantic-danger; border-radius: $border-radius-full; border: 4rpx solid rgba(255,255,255,0.2); }

.content-scroll { flex: 1; padding: $spacing-lg 28rpx; height: 0; box-sizing: border-box; width: 100%; }

.info-card { background: #fff; border-radius: $border-radius-xl; padding: $spacing-xl; box-shadow: 0 4rpx 16rpx rgba(120, 53, 15, 0.05); margin-bottom: $spacing-lg; box-sizing: border-box; width: 100%; }
.info-head { display: flex; align-items: center; gap: $spacing-lg; }
.info-school-icon { width: 88rpx; height: 88rpx; border-radius: $border-radius-xl; background: #F1F5F9; display: flex; align-items: center; justify-content: center; }
.info-school-icon-text { font-size: 48rpx; color: $color-text-muted; }
.info-head-text { display: flex; flex-direction: column; gap: $spacing-xs; }
.info-head-title { font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-primary; }
.info-head-sub { font-size: $typography-font-size-base; color: $color-text-hint; }
.info-divider { width: 100%; height: 2rpx; background: #F1F5F9; margin: 28rpx 0; }
.info-row { display: flex; align-items: center; }
.info-item { flex: 1; display: flex; align-items: center; gap: $spacing-md; }
.info-v-divider { width: 4rpx; height: 60rpx; border-radius: 2rpx; background: #F1F5F9; margin: 0 $spacing-lg; }
.info-item-avatar { width: 64rpx; height: 64rpx; border-radius: $border-radius-full; background: #F1F5F9; display: flex; align-items: center; justify-content: center; }
.info-item-avatar-text { font-size: $typography-font-size-xl; color: $color-text-muted; }
.info-item-icon-box { width: 64rpx; height: 64rpx; border-radius: 18rpx; background: #F1F5F9; display: flex; align-items: center; justify-content: center; }
.info-item-icon-text { font-size: $typography-font-size-2xl; color: $color-text-muted; }
.info-item-text { display: flex; flex-direction: column; gap: 4rpx; }
.info-item-label { font-size: $typography-font-size-xs; color: $color-text-hint; }
.info-item-value { font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; }

.section { margin-bottom: $spacing-lg; }
.section-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: $spacing-md; }
.section-title { font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-primary; }
.section-date { font-size: $typography-font-size-sm; color: $color-text-hint; }

.snapshot-row { display: flex; gap: 20rpx; }
.snap-item { flex: 1; background: #fff; border-radius: $border-radius-xl; padding: $spacing-lg 0; display: flex; flex-direction: column; align-items: center; gap: 4rpx; box-shadow: 0 2rpx 8rpx rgba(120, 53, 15, 0.04); }
.snap-num { font-size: 52rpx; font-weight: 800; }
.snap-num.green  { color: $color-semantic-success; }
.snap-num.yellow { color: #EAB308; }
.snap-num.orange { color: #F97316; }
.snap-num.blue   { color: $color-semantic-info; }
.snap-unit { font-size: $typography-font-size-xs; color: $color-text-hint; margin-top: -4rpx; }
.snap-label { font-size: $typography-font-size-sm; color: $color-text-muted; font-weight: 500; }

.quick-row { display: flex; gap: 20rpx; margin-bottom: $spacing-lg; }
.quick-btn { flex: 1; background: #fff; border-radius: $border-radius-xl; padding: 28rpx 0; display: flex; flex-direction: column; align-items: center; gap: $spacing-xs; box-shadow: 0 2rpx 8rpx rgba(120, 53, 15, 0.04); }
.quick-icon { font-size: 44rpx; color: $color-semantic-info; }
.quick-label { font-size: $typography-font-size-base; color: $color-text-primary; font-weight: 500; }

.sub-title-row { margin-bottom: $spacing-md; }
.sub-title { font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; }

.notice-card { background: #fff; border-radius: $border-radius-xl; padding: $spacing-lg 28rpx; display: flex; align-items: center; gap: 20rpx; box-shadow: 0 2rpx 8rpx rgba(120, 53, 15, 0.04); margin-bottom: $spacing-md; }
.notice-icon-box { width: 80rpx; height: 80rpx; border-radius: 20rpx; background: #DBEAFE; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.notice-icon { font-size: 44rpx; color: $color-semantic-info; }
.notice-body { flex: 1; overflow: hidden; display: flex; flex-direction: column; gap: $spacing-xs; }
.notice-title { font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.notice-time { font-size: $typography-font-size-xs; color: $color-text-hint; }
.notice-rate { display: flex; align-items: center; gap: $spacing-xs; flex-shrink: 0; }
.rate-bar-bg { width: 96rpx; height: 12rpx; border-radius: $border-radius-full; background: #E2E8F0; overflow: hidden; }
.rate-bar-fill { height: 100%; border-radius: $border-radius-full; background: $color-semantic-info; }
.rate-text { font-size: 20rpx; color: $color-text-hint; white-space: nowrap; }

.search-filter-row { display: flex; align-items: center; gap: $spacing-md; margin-bottom: 20rpx; }
.search-box { flex: 1; height: 80rpx; background: #fff; border-radius: 20rpx; border: 2rpx solid #E2E8F0; display: flex; align-items: center; padding: 0 $spacing-lg; gap: $spacing-md; box-sizing: border-box; }
.search-icon { font-size: $typography-font-size-3xl; color: $color-text-hint; flex-shrink: 0; }
.search-input { flex: 1; font-size: $typography-font-size-base; color: $color-text-primary; }
.filter-scroll { white-space: nowrap; flex-shrink: 0; }
.filter-row { display: flex; gap: $spacing-sm; }
.filter-pill { flex-shrink: 0; padding: $spacing-md 28rpx; border-radius: $border-radius-full; background: #fff; border: 2rpx solid #E2E8F0; text { font-size: $typography-font-size-base; color: $color-text-muted; } }
.filter-pill.active { background: $color-semantic-info; border-color: $color-semantic-info; text { color: #fff; font-weight: 600; } }

.state-wrap { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: $spacing-lg; padding: 120rpx 0; }
.state-text { font-size: $typography-font-size-md; color: $color-text-hint; }
.retry-btn { padding: $spacing-sm $spacing-xl; border-radius: $border-radius-full; background: $color-semantic-info; font-size: $typography-font-size-base; color: #fff; }

.student-list { display: flex; flex-direction: column; gap: $spacing-md; }
.student-card { background: #fff; border-radius: $border-radius-xl; padding: $spacing-lg 28rpx; display: flex; align-items: center; gap: 20rpx; box-shadow: 0 2rpx 8rpx rgba(120, 53, 15, 0.04); }
.stu-avatar { width: 80rpx; height: 80rpx; border-radius: $border-radius-full; flex-shrink: 0; }
.stu-avatar.fallback { background: #F1F5F9; display: flex; align-items: center; justify-content: center; text { font-size: $typography-font-size-2xl; font-weight: 700; color: $color-text-muted; } }
.stu-info { flex: 1; overflow: hidden; }
.stu-name-row { display: flex; align-items: center; gap: $spacing-md; }
.stu-name { font-size: $typography-font-size-lg; font-weight: 600; color: $color-text-primary; }
.stu-no { font-size: $typography-font-size-sm; color: $color-text-hint; }
.stu-status { display: flex; align-items: center; gap: $spacing-xs; flex-shrink: 0; }
.status-dot { width: 16rpx; height: 16rpx; border-radius: $border-radius-full; }
.status-text { font-size: $typography-font-size-sm; font-weight: 500; }
.status-present { .status-dot { background: $color-semantic-success; } .status-text { color: $color-semantic-success; } }
.status-leave  { .status-dot { background: $color-semantic-info; } .status-text { color: $color-semantic-info; } }
.status-absent { .status-dot { background: $color-semantic-danger; } .status-text { color: $color-semantic-danger; } }
.status-late   { .status-dot { background: #F97316; } .status-text { color: #F97316; } }

.arrow-icon { font-size: $typography-font-size-3xl; color: $color-text-hint; flex-shrink: 0; &.sm { font-size: $typography-font-size-2xl; } }
.view-all { display: flex; align-items: center; gap: $spacing-xs; }
.view-all-text { font-size: $typography-font-size-base; color: $color-semantic-info; }
.bottom-safe { height: calc(80rpx + env(safe-area-inset-bottom)); }
</style>
