<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchFeedbackList, submitFeedback } from "@/api/parent-feedback"

const feedbacks = ref([])
const form = ref({ category: "suggestion", content: "" })
const showForm = ref(false)
const submitting = ref(false)

async function loadData() {
  try {
    await ensureLogin()
    const res = await fetchFeedbackList()
    feedbacks.value = res.data?.list || []
  } catch (err) { /* silent */ }
}

async function handleSubmit() {
  if (!form.value.content.trim()) {
    uni.showToast({ title: "请填写反馈内容", icon: "none" })
    return
  }
  submitting.value = true
  try {
    await submitFeedback(form.value)
    uni.showToast({ title: "提交成功", icon: "success" })
    form.value.content = ""
    showForm.value = false
    loadData()
  } catch (err) {
    uni.showToast({ title: err?.message || "提交失败", icon: "none" })
  } finally { submitting.value = false }
}

onLoad(loadData)
</script>

<template>
  <view class="page">
    <view class="header">
      <text class="page-title">意见反馈</text>
      <button class="add-btn" @click="showForm = !showForm">{{ showForm ? "取消" : "+ 反馈" }}</button>
    </view>

    <view v-if="showForm" class="card">
      <view class="cats">
        <button
          v-for="c in [{ k: 'suggestion', l: '建议' }, { k: 'complaint', l: '投诉' }, { k: 'bug', l: '问题' }]"
          :key="c.k"
          class="cat-btn"
          :class="{ active: form.category === c.k }"
          @click="form.category = c.k"
        >{{ c.l }}</button>
      </view>
      <textarea v-model="form.content" class="textarea" placeholder="请描述您的意见或问题..." :maxlength="500" />
      <button class="submit-btn" :disabled="submitting" @click="handleSubmit">
        {{ submitting ? "提交中..." : "提交" }}
      </button>
    </view>

    <view v-if="feedbacks.length === 0" class="state">暂无反馈记录</view>
    <view v-else class="list">
      <view v-for="item in feedbacks" :key="item.id" class="f-card">
        <view class="f-head">
          <text class="f-cat">{{ item.category_label }}</text>
          <text class="f-status">{{ item.status_label }}</text>
        </view>
        <text class="f-content">{{ item.content }}</text>
        <text v-if="item.reply" class="f-reply">回复: {{ item.reply }}</text>
        <text class="f-date">{{ item.created_at?.slice(0, 16) }}</text>
      </view>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page { min-height: 100vh; padding: 24rpx 28rpx; background: $color-bg-page; }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 24rpx; }
.page-title { color: $color-text-primary; font-size: 36rpx; font-weight: 800; }
.add-btn { padding: 14rpx 28rpx; border-radius: 999rpx; background: $color-brand-primary; color: #fff; font-size: 24rpx; }
.card { padding: 28rpx; margin-bottom: 24rpx; border-radius: 24rpx; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.cats { display: flex; gap: 16rpx; margin-bottom: 20rpx; }
.cat-btn { padding: 12rpx 28rpx; border-radius: 999rpx; background: #f0f0f0; color: $color-text-muted; font-size: 24rpx; }
.cat-btn.active { background: $parent-blue; color: #fff; }
.textarea { width: 100%; height: 180rpx; padding: 20rpx; border: 2rpx solid rgba(106,77,42,0.12); border-radius: 16rpx; font-size: 26rpx; box-sizing: border-box; }
.submit-btn { width: 100%; height: 80rpx; margin-top: 20rpx; border-radius: 999rpx; background: $parent-blue; color: #fff; font-size: 28rpx; font-weight: 700; line-height: 80rpx; }
.state { display: flex; align-items: center; justify-content: center; min-height: 60vh; color: $color-text-muted; font-size: 28rpx; }
.list { display: flex; flex-direction: column; gap: 18rpx; }
.f-card { padding: 26rpx; border-radius: 20rpx; background: $color-bg-card; box-shadow: 0 8rpx 24rpx rgba(106,77,42,0.06); }
.f-head { display: flex; justify-content: space-between; margin-bottom: 12rpx; }
.f-cat { padding: 6rpx 16rpx; border-radius: 999rpx; background: #cfe1ff; color: #2f5a8c; font-size: 22rpx; }
.f-status { color: $color-text-muted; font-size: 22rpx; }
.f-content { display: block; color: $color-text-primary; font-size: 26rpx; line-height: 1.5; }
.f-reply { display: block; margin-top: 12rpx; padding: 16rpx; border-radius: 12rpx; background: #f0f4ff; color: $color-text-primary; font-size: 24rpx; }
.f-date { display: block; margin-top: 12rpx; color: $color-text-muted; font-size: 20rpx; }
</style>
