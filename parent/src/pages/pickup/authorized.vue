<script setup>
import { ref } from "vue"
import { onShow } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { getCurrentStudentId } from "@/utils/currentStudent"
import {
  fetchAuthorizedPersons,
  deleteAuthorizedPerson,
  designatePickupPerson,
  fetchDesignatedPerson
} from "@/api/parent-pickup"

const persons = ref([])
const loading = ref(false)
// 当前指定的授权人 ID（null = 今日尚未指定）
const designatedId = ref(null)

async function loadData() {
  loading.value = true
  try {
    await ensureLogin()
    const sid = getCurrentStudentId()
    // 并行拉列表与当前指定；指定查询无副作用，失败不阻断列表
    const [listRes, desigRes] = await Promise.all([
      fetchAuthorizedPersons(sid),
      fetchDesignatedPerson(sid).catch(() => ({ data: null }))
    ])
    persons.value = listRes.data?.list || []
    designatedId.value = desigRes.data?.designated ? desigRes.data.authorization_id : null
  } catch (err) {
    uni.showToast({ title: err?.message || "加载失败", icon: "none" })
  } finally {
    loading.value = false
  }
}

async function handleDesignate(person) {
  // 只有已审核通过且有效的授权人才能被指定
  if (person.review_status !== 1) {
    uni.showToast({ title: "该接送人尚未通过园所审核", icon: "none" })
    return
  }
  if (person.status !== 1) {
    uni.showToast({ title: "该接送人授权已撤销", icon: "none" })
    return
  }

  // 换人会作废今日已生成的凭证 —— 必须讲清楚
  const tip = designatedId.value
    ? `换为 ${person.name} 后，今日已生成的接送二维码将全部失效，需要重新获取。确定继续？`
    : `确定指定 ${person.name} 为今日接走人吗？`
  const ok = await new Promise(r =>
    uni.showModal({ title: "设为今日接走人", content: tip, success: s => r(s.confirm) })
  )
  if (!ok) return

  try {
    await designatePickupPerson(person.id, getCurrentStudentId())
    designatedId.value = person.id
    uni.showToast({ title: `今日接走人：${person.name}`, icon: "success" })
  } catch (err) {
    uni.showToast({ title: err?.message || "指定失败", icon: "none" })
  }
}

async function handleDelete(person) {
  const r = await new Promise(r => uni.showModal({ title: "确认删除", content: `确定删除 ${person.name} 吗?`, success: s => r(s.confirm) }))
  if (!r) return
  try {
    await deleteAuthorizedPerson(person.id)
    if (designatedId.value === person.id) designatedId.value = null
    uni.showToast({ title: "已删除", icon: "success" })
    loadData()
  } catch (err) {
    uni.showToast({ title: err?.message || "删除失败", icon: "none" })
  }
}

function goAdd() { uni.navigateTo({ url: "/pages/pickup/authorized-create" }) }

// 用 onShow 而非 onLoad：从新增页返回时要重新拉列表，否则刚添加的人看不见。
onShow(loadData)
</script>

<template>
  <view class="page">
    <!-- 安全提示 -->
    <view class="safety-card">
      <text class="safety-icon"></text>
      <view class="spacer-w12" />
      <view>
        <text class="safety-title">安全提示</text>
        <text class="safety-text">授权接送人信息仅供门卫核验身份使用，请确保信息真实有效。</text>
      </view>
    </view>

    <view class="header">
      <text class="page-title">授权接送人</text>
      <button class="add-btn" @click="goAdd">+ 添加</button>
    </view>

    <!-- 当前指定：跨页可见，指定后才能取接送二维码 -->
    <view v-if="designatedId" class="designated-banner">
      <text class="designated-text">
        今日接走人：{{ (persons.find(p => p.id === designatedId) || {}).name || '已指定' }}
      </text>
    </view>
    <view v-else-if="!loading && persons.length > 0" class="designated-banner banner-empty">
      <text class="designated-text">今日尚未指定接走人，指定后才能获取接送二维码</text>
    </view>

    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="persons.length === 0" class="state">暂无授权接送人</view>
    <view v-else class="list">
      <view
        v-for="p in persons"
        :key="p.id"
        :class="['card', p.id === designatedId ? 'card-designated' : '']"
      >
        <view class="person-avatar">
          <image v-if="p.avatar" :src="p.avatar" class="avatar-img" mode="aspectFill" />
          <text v-else class="avatar-placeholder"></text>
        </view>
        <view class="spacer-w12" />
        <view class="info">
          <view class="name-row">
            <text class="name">{{ p.name }}</text>
            <view :class="['auth-badge', p.auth_type === 'permanent' ? 'badge-perm' : 'badge-temp']">
              <text>{{ p.auth_type_label || '长期授权' }}</text>
            </view>
            <view v-if="p.id === designatedId" class="today-badge">
              <text>今日接走人</text>
            </view>
          </view>
          <text class="relation">{{ p.relation }} · {{ p.phone }}</text>
          <view class="perm-tags">
            <text v-if="p.allow_pickup" class="perm-tag green">可接送</text>
            <text v-if="p.allow_view_record" class="perm-tag blue">看记录</text>
            <text v-if="p.allow_growth_view" class="perm-tag purple">看成长</text>
          </view>
          <view class="actions">
            <button
              v-if="p.id !== designatedId"
              :class="['designate-btn', (p.review_status === 1 && p.status === 1) ? '' : 'btn-disabled']"
              @click="handleDesignate(p)"
            >设为今日接走人</button>
            <button class="delete-btn" @click="handleDelete(p)">删除</button>
          </view>
        </view>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: $spacing-lg $component-page-padding-x calc(150rpx + env(safe-area-inset-bottom)); background: $color-bg-page; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }

.safety-card { display: flex; padding: $spacing-lg; border-radius: $border-radius-xl; background: $color-brand-primary-bg; margin-bottom: 24rpx; }
.safety-icon { font-family: remixicon; font-size: $typography-font-size-3xl; color: $color-brand-primary; flex-shrink: 0; }
.safety-title { display: block; font-size: $typography-font-size-base; font-weight: 600; color: $color-semantic-warning; }
.safety-text { display: block; margin-top: 4rpx; font-size: $typography-font-size-sm; color: $color-semantic-warning; opacity: 0.8; }

.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24rpx; }
.page-title { font-size: $typography-font-size-2xl; font-weight: 700; color: $color-text-primary; }
.add-btn { padding: $spacing-md 28rpx; border-radius: $border-radius-full; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-sm; }

.state { display: flex; align-items: center; justify-content: center; min-height: 50vh; color: $color-text-hint; font-size: $typography-font-size-md; }
.list { display: flex; flex-direction: column; gap: $spacing-md; }

.card { display: flex; align-items: center; padding: $spacing-lg; border-radius: $border-radius-xl; background: $color-bg-card; box-shadow: $shadow-sm; }
.person-avatar { width: $component-avatar-md; height: $component-avatar-md; border-radius: $border-radius-full; overflow: hidden; background: $color-semantic-info-bg; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.avatar-img { width: 100%; height: 100%; }
.avatar-placeholder { font-family: remixicon; font-size: $typography-font-size-3xl; color: $color-semantic-info; }

.info { flex: 1; overflow: hidden; }
.name-row { display: flex; align-items: center; gap: $spacing-sm; }
.name { font-size: $typography-font-size-md; font-weight: 600; color: $color-text-primary; }
.auth-badge { padding: 4rpx 16rpx; border-radius: $border-radius-full; font-size: $typography-font-size-xs; }
.badge-perm { background: $color-semantic-success-bg; color: $color-semantic-success; }
.badge-temp { background: $color-brand-primary-bg; color: $color-brand-primary; }
.relation { display: block; margin-top: 8rpx; font-size: $typography-font-size-sm; color: $color-text-muted; }
.perm-tags { display: flex; gap: $spacing-xs; margin-top: 8rpx; }
.perm-tag { padding: 4rpx 12rpx; border-radius: $border-radius-full; font-size: $typography-font-size-xs; }
.perm-tag.green { background: $color-semantic-success-bg; color: $color-semantic-success; }
.perm-tag.blue { background: $color-semantic-info-bg; color: $color-semantic-info; }
.perm-tag.purple { background: $parent-purple-bg; color: $parent-purple; }

.delete-btn { padding: $spacing-md 24rpx; border-radius: $border-radius-full; background: $color-semantic-danger-bg; color: $color-semantic-danger; font-size: $typography-font-size-xs; flex-shrink: 0; }

/* 当前指定横幅 */
.designated-banner { display: flex; align-items: center; padding: $spacing-md $spacing-lg; border-radius: $border-radius-xl; background: $color-semantic-success-bg; margin-bottom: 24rpx; }
.designated-banner.banner-empty { background: $color-semantic-warning-bg; }
.designated-text { font-size: $typography-font-size-sm; color: $color-semantic-success; }
.banner-empty .designated-text { color: $color-semantic-warning; }

/* 指定中的卡片高亮 */
.card-designated { border: 2rpx solid $color-brand-primary; }
.today-badge { padding: 4rpx 16rpx; border-radius: $border-radius-full; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-xs; }

/* 操作行：指定 + 删除 */
.actions { display: flex; gap: $spacing-sm; margin-top: 12rpx; }
.designate-btn { padding: $spacing-md 24rpx; border-radius: $border-radius-full; background: $color-brand-primary; color: #fff; font-size: $typography-font-size-xs; }
.designate-btn.btn-disabled { background: $color-text-hint; opacity: 0.4; color: #fff; }
</style>
