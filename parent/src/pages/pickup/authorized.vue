<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchAuthorizedPersons, deleteAuthorizedPerson } from "@/api/parent-pickup"

const persons = ref([])
const loading = ref(false)

async function loadData() {
  loading.value = true
  try { await ensureLogin(); const res = await fetchAuthorizedPersons(1); persons.value = res.data?.list || [] }
  catch (err) { uni.showToast({ title: err?.message || "加载失败", icon: "none" }) }
  finally { loading.value = false }
}

async function handleDelete(person) {
  const r = await new Promise(r => uni.showModal({ title: "确认删除", content: `确定删除 ${person.name} 吗?`, success: s => r(s.confirm) }))
  if (!r) return
  try { await deleteAuthorizedPerson(person.id); uni.showToast({ title: "已删除", icon: "success" }); loadData() }
  catch (err) { uni.showToast({ title: err?.message || "删除失败", icon: "none" }) }
}

function goAdd() { uni.navigateTo({ url: "/pages/pickup/authorized" }) }

onLoad(loadData)
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

    <view v-if="loading" class="state">加载中...</view>
    <view v-else-if="persons.length === 0" class="state">暂无授权接送人</view>
    <view v-else class="list">
      <view v-for="p in persons" :key="p.id" class="card">
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
          </view>
          <text class="relation">{{ p.relation }} · {{ p.phone }}</text>
          <view class="perm-tags">
            <text v-if="p.allow_pickup" class="perm-tag green">可接送</text>
            <text v-if="p.allow_view_record" class="perm-tag blue">看记录</text>
            <text v-if="p.allow_growth_view" class="perm-tag purple">看成长</text>
          </view>
        </view>
        <button class="delete-btn" @click="handleDelete(p)">删除</button>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx calc(150rpx + env(safe-area-inset-bottom)); background: $parent-bg; }
.spacer-w12 { width: 24rpx; flex-shrink: 0; }

.safety-card { display: flex; padding: 24rpx; border-radius: $parent-radius; background: #FFF7ED; margin-bottom: 24rpx; }
.safety-icon { font-family: remixicon; font-size: 40rpx; color: #F97316; flex-shrink: 0; }
.safety-title { display: block; font-size: 26rpx; font-weight: 600; color: #92400E; }
.safety-text { display: block; margin-top: 4rpx; font-size: 24rpx; color: #A16207; }

.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24rpx; }
.page-title { font-size: 36rpx; font-weight: 700; color: $parent-text; }
.add-btn { padding: 14rpx 28rpx; border-radius: 9999rpx; background: $parent-primary; color: #fff; font-size: 24rpx; }

.state { display: flex; align-items: center; justify-content: center; min-height: 50vh; color: $parent-hint; font-size: 28rpx; }
.list { display: flex; flex-direction: column; gap: 16rpx; }

.card { display: flex; align-items: center; padding: 24rpx; border-radius: $parent-radius; background: $parent-card; box-shadow: $parent-shadow; }
.person-avatar { width: 96rpx; height: 96rpx; border-radius: 9999rpx; overflow: hidden; background: #DBEAFE; display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
.avatar-img { width: 100%; height: 100%; }
.avatar-placeholder { font-family: remixicon; font-size: 40rpx; color: #2563EB; }

.info { flex: 1; overflow: hidden; }
.name-row { display: flex; align-items: center; gap: 12rpx; }
.name { font-size: 28rpx; font-weight: 600; color: $parent-text; }
.auth-badge { padding: 4rpx 16rpx; border-radius: 9999rpx; font-size: 20rpx; }
.badge-perm { background: #DCFCE7; color: #166534; }
.badge-temp { background: #FFEDD5; color: #92400E; }
.relation { display: block; margin-top: 8rpx; font-size: 24rpx; color: $parent-muted; }
.perm-tags { display: flex; gap: 8rpx; margin-top: 8rpx; }
.perm-tag { padding: 4rpx 12rpx; border-radius: 9999rpx; font-size: 20rpx; }
.perm-tag.green { background: #DCFCE7; color: #166534; }
.perm-tag.blue { background: #DBEAFE; color: #1D4ED8; }
.perm-tag.purple { background: #F3E8FF; color: #7E22CE; }

.delete-btn { padding: 12rpx 24rpx; border-radius: 9999rpx; background: #FEE2E2; color: #DC2626; font-size: 22rpx; flex-shrink: 0; }
</style>
