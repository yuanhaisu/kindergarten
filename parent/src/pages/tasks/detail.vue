<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { getCurrentStudentId } from "@/utils/currentStudent"
import { ensureLogin } from "@/utils/login"
import { getToken } from "@/utils/auth"
import config from "@/config/env"
import { fetchTaskDetail, checkinTask, fetchCheckinHistory, withStudentScope } from "@/api/parent-task"

const detail = ref({})
const history = ref([])
// 后端分页上限 50。超出时页面要说明「只显示最近的」，
// 否则每日任务跑满一个月后，更早的提交会悄无声息地不见。
const historyTotal = ref(0)
const checkinContent = ref("")
const mediaFileID = ref(0)
const mediaName = ref("")
const loading = ref(false)
const loadError = ref("")

async function loadData(id) {
  loading.value = true
  loadError.value = ""
  try {
    await ensureLogin()
    // 当前孩子解析与 403 重试统一走 withStudentScope（与列表页同一条路径）。
    const scoped = await withStudentScope((sid) => Promise.all([
      fetchTaskDetail(id, { student_id: sid }),
      // 后端已分页（默认 20 条）；这里一次取到上限 50，并记下 total，
      // 超出一页时在页面上说明「只显示最近 N 条」，而不是静默少显示。
      fetchCheckinHistory(id, { student_id: sid, page: 1, page_size: 50 }),
    ]))
    if (!scoped) {
      loadError.value = "未绑定孩子信息"
      return
    }
    const [d, h] = scoped.result
    detail.value = d.data || {}
    history.value = h.data?.list || []
    historyTotal.value = Number(h.data?.total || history.value.length)
  } catch (err) {
    detail.value = {}
    history.value = []
    loadError.value = err?.message || "加载任务失败"
  } finally { loading.value = false }
}

async function handleCheckin() {
  if (!detail.value.id) return
  const content = checkinContent.value.trim()
  if (!content && !mediaFileID.value) {
    uni.showToast({ title: "请填写内容或上传照片", icon: "none" })
    return
  }
  try {
    // student_id 走 query（第三参）：body 里的会被后端丢弃
    const res = await checkinTask(detail.value.id, {
      content,
      media_file_id: mediaFileID.value || undefined
    }, getCurrentStudentId())
    checkinContent.value = ""
    mediaFileID.value = 0
    mediaName.value = ""
    detail.value.submitted_current_period = true
    uni.showToast({ title: res.data?.message || "已提交", icon: "success" })
    loadData(detail.value.id)
  } catch (err) {
    uni.showToast({ title: err?.message || "打卡失败", icon: "none" })
  }
}

async function quickCheckin() {
  if (!detail.value.id) return
  try {
    const res = await checkinTask(detail.value.id, { content: "" }, getCurrentStudentId())
    mediaFileID.value = 0
    mediaName.value = ""
    detail.value.submitted_current_period = true
    uni.showToast({ title: res.data?.message || "已提交", icon: "success" })
    loadData(detail.value.id)
  } catch (e) {
    uni.showToast({ title: e?.message || "提交失败", icon: "none" })
  }
}

async function chooseMedia() {
  try {
    await ensureLogin()
  } catch (e) {
    return
  }
  uni.chooseImage({
    count: 1,
    success: (res) => {
      const path = res.tempFilePaths?.[0]
      if (!path) return
      uni.showLoading({ title: "上传中" })
      uni.uploadFile({
        url: config.uploadURL,
        filePath: path,
        name: "file",
        formData: { biz_type: "task" },
        header: { Authorization: `Bearer ${getToken()}` },
        success: (up) => {
          try {
            const body = JSON.parse(up.data || "{}")
            if (body.code === 0 && body.data?.id) {
              mediaFileID.value = body.data.id
              mediaName.value = body.data.file_name || "已上传"
              uni.showToast({ title: "上传成功", icon: "success" })
            } else {
              uni.showToast({ title: body.message || "上传失败", icon: "none" })
            }
          } catch (e) {
            uni.showToast({ title: "上传失败", icon: "none" })
          }
        },
        fail: () => uni.showToast({ title: "上传失败", icon: "none" }),
        complete: () => uni.hideLoading()
      })
    }
  })
}

onLoad((opts) => { if (opts?.id) loadData(opts.id) })
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="loadError" class="state">{{ loadError }}</view>
    <template v-else>
      <view class="card">
        <text class="title">{{ detail.title }}</text>
        <text class="desc">{{ detail.description }}</text>
        <text class="intro">{{ detail.intro }}</text>
        <view v-if="detail.requirements?.length" class="reqs">
          <text v-for="(r, i) in detail.requirements" :key="i" class="req">{{ i + 1 }}. {{ r }}</text>
        </view>
        <view class="stats">
          <text>{{ detail.frequency === "daily" ? "每日任务" : detail.frequency === "weekly" ? "每周任务" : "一次性任务" }}</text>
          <text v-if="detail.learning_link">和幼儿园学习相关</text>
        </view>
        <text v-if="detail.learning_link" class="intro">{{ detail.learning_link }}</text>
      </view>

      <view v-if="detail.responses?.length" class="history-section">
        <text class="section-title">老师回应</text>
        <view v-for="r in detail.responses" :key="r.id" class="history-item">
          <text class="h-date">{{ r.ritual === "voice" ? "语音" : r.ritual === "class_star" ? "班级星星" : "贴纸" }}</text>
          <text class="h-content">{{ r.content || "老师回应了你" }}</text>
        </view>
      </view>

      <view v-if="detail.id && !detail.submitted_current_period" class="checkin-card">
        <text class="section-title">今日提交</text>
        <view class="media-btn" @click="chooseMedia">
          <text>{{ mediaName || "＋ 添加照片（可选）" }}</text>
        </view>
        <view class="media-btn" @click="quickCheckin">
          <text>今天完成了（一键提交）</text>
        </view>
        <textarea v-model="checkinContent" class="textarea" placeholder="记录今天的内容..." :maxlength="500" />
        <button class="checkin-btn" @click="handleCheckin">打卡</button>
      </view>

      <view class="history-section">
        <text class="section-title">打卡历史</text>
        <view v-for="h in history" :key="h.id" class="history-item">
          <text class="h-date">{{ h.date }}</text>
          <text class="h-content">{{ h.content || "已提交" }}</text>
        </view>
        <text v-if="historyTotal > history.length" class="history-hint">
          仅显示最近 {{ history.length }} 条提交（共 {{ historyTotal }} 条）
        </text>
      </view>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg 28rpx 48rpx; background: $color-bg-page; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: $typography-font-size-md; }
.card { padding: $spacing-xl; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.title { display: block; color: $color-text-primary; font-size: $typography-font-size-2xl; font-weight: 800; }
.desc { display: block; margin-top: 12rpx; color: $color-text-primary; font-size: $typography-font-size-base; }
.intro { display: block; margin-top: 16rpx; color: $color-text-muted; font-size: $typography-font-size-sm; line-height: 1.6; }
.reqs { margin-top: 16rpx; }
.req { display: block; margin-top: 6rpx; color: $color-text-muted; font-size: $typography-font-size-sm; }
.stats { display: flex; gap: $spacing-lg; margin-top: 20rpx; color: $color-semantic-info; font-size: $typography-font-size-sm; }
.checkin-card { margin-top: 24rpx; padding: 28rpx; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.section-title { display: block; margin-bottom: 16rpx; color: $color-text-primary; font-size: $typography-font-size-md; font-weight: 700; }
.media-btn { display: flex; align-items: center; justify-content: center; height: 96rpx; margin-bottom: 16rpx; border: 2rpx dashed rgba(106,77,42,0.2); border-radius: $border-radius-lg; color: $color-text-muted; font-size: $typography-font-size-sm; }
.textarea { width: 100%; height: 160rpx; padding: 20rpx; border: 2rpx solid rgba(106,77,42,0.12); border-radius: $border-radius-lg; font-size: $typography-font-size-base; box-sizing: border-box; }
.checkin-btn { width: 100%; height: 80rpx; margin-top: 20rpx; border-radius: 999rpx; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-md; font-weight: 700; line-height: 80rpx; }
.history-section { margin-top: 24rpx; }
.history-item { display: flex; align-items: center; gap: $spacing-md; padding: 20rpx; margin-bottom: 12rpx; border-radius: $border-radius-lg; background: $color-bg-card; }
.h-date { flex: 0 0 140rpx; color: $color-text-muted; font-size: $typography-font-size-xs; }
.h-content { flex: 1; color: $color-text-primary; font-size: $typography-font-size-sm; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.history-hint { display: block; margin-top: $spacing-xs; color: $color-text-muted; font-size: $typography-font-size-xs; }
.h-points { color: $color-brand-primary; font-size: $typography-font-size-sm; font-weight: 700; }
</style>
