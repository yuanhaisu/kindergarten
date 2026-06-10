<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { fetchLeaveDetail, cancelLeave } from "@/api/parent-leave"

const detail = ref({})
const loading = ref(false)

async function loadData(id) {
  loading.value = true
  try {
    const res = await fetchLeaveDetail(id)
    detail.value = res.data || {}
  } finally { loading.value = false }
}

async function handleCancel() {
  const r = await new Promise(r => uni.showModal({ title: "确认撤销", content: "确定要撤销此请假申请吗？", success: s => r(s.confirm) }))
  if (!r) return
  try {
    await cancelLeave(detail.value.id)
    detail.value.status = 3
    detail.value.status_label = "已撤销"
    uni.showToast({ title: "已撤销", icon: "success" })
  } catch (err) {
    uni.showToast({ title: err?.message || "操作失败", icon: "none" })
  }
}

onLoad((opts) => { if (opts?.id) loadData(opts.id) })
</script>

<template>
  <view class="page">
    <view v-if="loading" class="state">加载中...</view>
    <template v-else>
      <!-- 状态头部 -->
      <view class="status-card" :class="'status-' + detail.status">
        <view class="status-icon-box">
          <text class="status-icon">{{ detail.status === 1 ? '' : detail.status === 2 ? '' : detail.status === 3 ? '' : '' }}</text>
        </view>
        <text class="status-text">{{ detail.status_label }}</text>
        <text v-if="detail.review_remark" class="status-remark">{{ detail.review_remark }}</text>
      </view>

      <!-- 信息卡片 -->
      <view class="card">
        <view class="info-row">
          <text class="info-key">学生</text>
          <text class="info-val">{{ detail.student?.name || "" }} · {{ detail.student?.class_name || "" }}</text>
        </view>
        <view class="info-row">
          <text class="info-key">日期</text>
          <text class="info-val">{{ detail.start_date }} ~ {{ detail.end_date }}</text>
        </view>
        <view class="info-row">
          <text class="info-key">天数</text>
          <text class="info-val">{{ detail.days }}天</text>
        </view>
        <view v-if="detail.emergency_contact?.name" class="info-row">
          <text class="info-key">接送人</text>
          <text class="info-val">{{ detail.emergency_contact.name }} {{ detail.emergency_contact.phone }}</text>
        </view>
        <view class="info-row" style="border-bottom:none">
          <text class="info-key">原因</text>
          <text class="info-val reason">{{ detail.reason }}</text>
        </view>
      </view>

      <!-- 审批信息 -->
      <view v-if="detail.reviewer" class="card" style="margin-top:24rpx">
        <text class="card-title">审批信息</text>
        <view class="spacer-h16" />
        <view class="reviewer-row">
          <view class="reviewer-avatar">
            <text class="ra-icon"></text>
          </view>
          <view class="spacer-w12" />
          <view>
            <text class="reviewer-name">{{ detail.reviewer?.nickname || "" }}</text>
            <text class="reviewer-time">{{ detail.reviewed_at?.slice(0, 16) || "" }}</text>
          </view>
        </view>
      </view>

      <!-- 撤销按钮 -->
      <button v-if="detail.status === 0" class="cancel-btn" @click="handleCancel">撤销申请</button>
    </template>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx 160rpx; background: $parent-bg; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }
.spacer-h16 { height: 32rpx; }

.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $parent-hint; font-size: 28rpx; }

.status-card { text-align: center; padding: 48rpx 32rpx; border-radius: $parent-radius-md; margin-bottom: 24rpx; }
.status-card.status-0 { background: #FFF7ED; }
.status-card.status-1 { background: #DCFCE7; }
.status-card.status-2 { background: #FEE2E2; }
.status-card.status-3 { background: #F1F5F9; }
.status-icon-box { width: 96rpx; height: 96rpx; margin: 0 auto 16rpx; border-radius: 9999rpx; background: rgba(255,255,255,0.8); display: flex; align-items: center; justify-content: center; }
.status-icon { font-family: remixicon; font-size: 48rpx; color: $parent-primary; }
.status-card.status-1 .status-icon { color: #16A34A; }
.status-card.status-2 .status-icon { color: #EF4444; }
.status-card.status-3 .status-icon { color: #94A3B8; }
.status-text { font-size: 36rpx; font-weight: 700; color: $parent-text; }
.status-remark { display: block; margin-top: 16rpx; font-size: 26rpx; color: $parent-muted; }

.card { padding: 32rpx; border-radius: $parent-radius-md; background: $parent-card; box-shadow: $parent-shadow; }
.card-title { font-size: 28rpx; font-weight: 600; color: $parent-text; }

.info-row { display: flex; justify-content: space-between; padding: 20rpx 0; border-bottom: 1rpx solid #F1F5F9; }
.info-key { font-size: 26rpx; color: $parent-muted; flex-shrink: 0; }
.info-val { font-size: 26rpx; color: $parent-text; text-align: right; max-width: 65%; }
.info-val.reason { line-height: 1.5; }

.reviewer-row { display: flex; align-items: center; }
.reviewer-avatar { width: 80rpx; height: 80rpx; border-radius: 9999rpx; background: #DBEAFE; display: flex; align-items: center; justify-content: center; }
.ra-icon { font-family: remixicon; font-size: 36rpx; color: #2563EB; }
.reviewer-name { display: block; font-size: 28rpx; font-weight: 500; color: $parent-text; }
.reviewer-time { display: block; margin-top: 4rpx; font-size: 24rpx; color: $parent-hint; }

.cancel-btn { width: calc(100% - 56rpx); height: 88rpx; margin: 48rpx 28rpx; border-radius: 9999rpx; background: #EF4444; color: #fff; font-size: 28rpx; font-weight: 700; line-height: 88rpx; }
</style>
