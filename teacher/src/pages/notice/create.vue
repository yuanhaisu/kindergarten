<script setup lang="ts">
import { ref, computed, onMounted } from "vue"
import { createTeacherNotice, fetchTeacherNoticeTemplates } from "@/api/teacher-notices"
import { requestWithAuth } from "@/api/http"

const categories = [
  { key: "homework", label: "作业" },
  { key: "bring", label: "带物" },
  { key: "activity", label: "活动" },
  { key: "health", label: "健康" },
  { key: "other", label: "其他" },
]

const submitting = ref(false)
const form = ref({
  category: "homework",
  title: "",
  content: "",
  need_confirm: true,
  scheduled_enabled: false,
  scheduled_at: "",
  attachments: [] as string[],
  class_id: 0,
})

const classList = ref<any[]>([])
const currentClassLabel = computed(() => {
  const c = classList.value.find((c) => c.class_id === form.value.class_id)
  return c ? `${c.class_name}全体家长` : "选择班级"
})

onMounted(async () => {
  try {
    const res: any = await requestWithAuth({ url: "/teacher/class/list", method: "GET" })
    const data = res.data || res
    classList.value = Array.isArray(data) ? data : (data.list || [])
    if (classList.value.length > 0 && form.value.class_id === 0) {
      form.value.class_id = classList.value[0].class_id
    }
  } catch { /* */ }
})

function selectCategory(key) {
  form.value.category = key
}

function addAttachment() {
  uni.chooseImage({
    count: 9 - form.value.attachments.length,
    success(res) { form.value.attachments.push(...res.tempFilePaths) }
  })
}
function removeAttachment(i) { form.value.attachments.splice(i, 1) }

function onScheduleDateChange(e) {
  form.value.scheduled_at = e.detail.value + " " + ((form.value.scheduled_at || "").split(" ")[1] || "09:00")
}
function onScheduleTimeChange(e) {
  form.value.scheduled_at = ((form.value.scheduled_at || "").split(" ")[0] || "") + " " + e.detail.value
}

// 模板
const templates = ref<any[]>([])
onMounted(async () => {
  try { const res = await fetchTeacherNoticeTemplates(); templates.value = res.data?.list || [] } catch { /* */ }
})
function useTemplate(t: Record<string, any>) {
  form.value.title = t.title || ""
  form.value.content = t.content || ""
  if (t.category) form.value.category = t.category
}

function validate() {
  if (!form.value.title.trim()) return "请输入通知标题"
  if (!form.value.content.trim()) return "请输入通知内容"
  if (form.value.scheduled_enabled && !form.value.scheduled_at) return "请选择发送时间"
  return ""
}

async function submit() {
  const msg = validate()
  if (msg) { uni.showToast({ title: msg, icon: "none" }); return }
  submitting.value = true
  try {
    await createTeacherNotice(form.value)
    uni.showToast({ title: "已发布", icon: "success" })
    setTimeout(() => {
        const pages = getCurrentPages()
        pages.length > 1 ? uni.navigateBack() : uni.reLaunch({ url: "/pages/index/index" })
      }, 600)
  } catch (e) {
    uni.showToast({ title: e?.message || "发布失败", icon: "none" })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <view class="page">
    <scroll-view class="body" scroll-y>
      <!-- 通知类型 -->
      <view class="section">
        <view class="card">
          <text class="card-title">通知类型</text>
          <view class="category-row">
            <view
              v-for="c in categories" :key="c.key"
              class="category-pill"
              :class="{ active: form.category === c.key }"
              @click="selectCategory(c.key)"
            >{{ c.label }}</view>
          </view>
        </view>
      </view>

      <!-- 标题 -->
      <view class="section">
        <view class="card">
          <text class="card-title">标题</text>
          <input v-model="form.title" class="input" placeholder="请输入通知标题" maxlength="80" />
        </view>
      </view>

      <!-- 内容 -->
      <view class="section">
        <view class="card">
          <text class="card-title">内容</text>
          <textarea v-model="form.content" class="textarea" placeholder="请输入通知内容..." />
        </view>
      </view>

      <!-- 添加图片/视频 -->
      <view class="section">
        <view class="card">
          <text class="card-title">添加图片/视频</text>
          <view class="media-row">
            <view class="media-add" @click="addAttachment">
              <text class="add-icon">＋</text>
              <text class="add-text">添加</text>
            </view>
            <view v-for="(img, i) in form.attachments" :key="i" class="media-item">
              <image :src="img" mode="aspectFill" class="media-img" />
              <view class="media-del" @click="removeAttachment(i)">✕</view>
            </view>
          </view>
        </view>
      </view>

      <!-- 接收范围 -->
      <view class="section">
        <picker v-if="classList.length > 1" mode="selector" :range="classList" range-key="class_name" @change="form.class_id = classList[$event.detail.value].class_id">
          <view class="card row-card">
            <view>
              <text class="card-title" style="margin-bottom:4rpx">接收范围</text>
              <text class="card-sub">{{ currentClassLabel }}</text>
            </view>
            <text class="row-arrow">▶</text>
          </view>
        </picker>
        <view v-else class="card row-card">
          <view>
            <text class="card-title" style="margin-bottom:4rpx">接收范围</text>
            <text class="card-sub">{{ currentClassLabel }}</text>
          </view>
        </view>
      </view>

      <!-- 需要家长确认收到 -->
      <view class="section">
        <view class="card row-card">
          <view>
            <text class="card-title" style="margin-bottom:4rpx">需要家长确认收到</text>
            <text class="card-sub">开启后可查看已读未读状态</text>
          </view>
          <view
            class="toggle"
            :class="{ on: form.need_confirm }"
            @click="form.need_confirm = !form.need_confirm"
          >
            <view class="toggle-knob" />
          </view>
        </view>
      </view>

      <!-- 定时发送 -->
      <view class="section">
        <view class="card row-card">
          <view>
            <text class="card-title" style="margin-bottom:4rpx">定时发送</text>
            <text class="card-sub">{{ form.scheduled_enabled && form.scheduled_at ? form.scheduled_at : '立即发送' }}</text>
          </view>
          <view
            class="toggle"
            :class="{ on: form.scheduled_enabled }"
            @click="form.scheduled_enabled = !form.scheduled_enabled"
          >
            <view class="toggle-knob" />
          </view>
        </view>
        <view v-if="form.scheduled_enabled" style="margin-top:16rpx">
          <picker
            mode="date"
            :value="(form.scheduled_at || '').split(' ')[0]"
            @change="onScheduleDateChange"
          >
            <view class="schedule-field">
              <text class="field-label">发送日期</text>
              <text class="field-value">{{ (form.scheduled_at || '').split(' ')[0] || '请选择日期' }}</text>
            </view>
          </picker>
          <picker
            mode="time"
            :value="(form.scheduled_at || '').split(' ')[1] || ''"
            @change="onScheduleTimeChange"
          >
            <view class="schedule-field">
              <text class="field-label">发送时间</text>
              <text class="field-value">{{ (form.scheduled_at || '').split(' ')[1] || '请选择时间' }}</text>
            </view>
          </picker>
        </view>
      </view>

      <!-- 常用模板 -->
      <view v-if="templates.length" class="section">
        <view class="template-header">
          <text class="section-label">常用模板</text>
          <text class="section-link">查看更多</text>
        </view>
        <view v-for="t in templates" :key="t.id" class="card template-card">
          <view class="template-top">
            <text class="template-name">{{ t.title }}</text>
            <view class="template-use" @click="useTemplate(t)">使用</view>
          </view>
          <text class="template-preview">{{ t.content }}</text>
        </view>
      </view>

      <view style="height:16rpx"></view>
    </scroll-view>

    <!-- 底部发布 -->
    <view class="bottom-bar">
      <button class="publish-btn" :loading="submitting" @click="submit">
        {{ submitting ? "发布中..." : "发布" }}
      </button>
    </view>
  </view>
</template>

<style scoped lang="scss">
.page {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: $color-bg-card;
}
.body { flex: 1; }

.section { padding: 0 48rpx 32rpx 48rpx; }
.section:first-child { padding-top: $spacing-xl; }

/* 卡片 */
.card {
  padding: $spacing-xl;
  background: #ffffff;
  border-radius: $border-radius-xl;
  box-shadow: 0 2rpx 4rpx rgba(0,0,0,0.05);
}
.card-title {
  display: block;
  font-size: $typography-font-size-lg;
  font-weight: 700;
  color: $color-text-primary;
  margin-bottom: $spacing-lg;
}
.card-sub { font-size: $typography-font-size-base; color: $color-text-muted; }

/* 分类胶囊 */
.category-row {
  display: flex;
  flex-wrap: wrap;
  gap: $spacing-lg;
}
.category-pill {
  padding: $spacing-md $spacing-xl;
  border: 1rpx solid #E2E8F0;
  border-radius: $border-radius-full;
  background: #f1f5f9;
  font-size: $typography-font-size-md;
  color: $color-text-muted;
  &.active {
    background: $color-semantic-info-bg;
    border-color: #bfdbfe;
    color: $color-semantic-info;
    font-weight: 600;
  }
}

/* 输入 */
.input {
  width: 100%;
  height: 88rpx;
  padding: 0 $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  font-size: $typography-font-size-lg;
  color: $color-text-primary;
  box-sizing: border-box;
}
.textarea {
  width: 100%;
  height: 240rpx;
  padding: $spacing-lg;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
  font-size: $typography-font-size-lg;
  color: $color-text-primary;
  box-sizing: border-box;
}

/* 附件上传 */
.media-row { display: flex; gap: $spacing-lg; flex-wrap: wrap; }
.media-add {
  width: 160rpx; height: 160rpx;
  border: 1rpx dashed #cbd5e1;
  border-radius: $border-radius-xl;
  background: $color-bg-card;
  display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  gap: $spacing-xs;
}
.add-icon { font-size: 48rpx; color: $color-text-hint; }
.add-text { font-size: $typography-font-size-sm; color: $color-text-hint; }
.media-item {
  width: 160rpx; height: 160rpx;
  border-radius: $border-radius-xl; overflow: hidden;
  position: relative;
}
.media-img { width: 100%; height: 100%; }
.media-del {
  position: absolute; top: 8rpx; right: 8rpx;
  width: 40rpx; height: 40rpx;
  background: rgba(0,0,0,0.5); color: #fff;
  border-radius: 50%;
  display: flex; align-items: center; justify-content: center;
  font-size: $typography-font-size-sm;
}

/* 行卡片 */
.row-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.row-card .card-title { margin-bottom: 0; }
.row-arrow { font-size: $typography-font-size-md; color: $color-text-hint; }

/* 定时发送 */
.schedule-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 80rpx;
  padding: 0 $spacing-lg;
  margin-bottom: $spacing-md;
  border-radius: $border-radius-lg;
  background: $color-bg-card;
}
.field-label { font-size: $typography-font-size-md; color: $color-text-primary; }
.field-value { font-size: $typography-font-size-base; color: $color-text-muted; }

/* 自定义开关 */
.toggle {
  width: 96rpx; height: 48rpx;
  border-radius: $border-radius-full;
  background: #E2E8F0;
  position: relative;
  transition: background 0.2s;
  &.on { background: $color-semantic-info; }
}
.toggle-knob {
  width: 42rpx; height: 42rpx;
  border-radius: 50%;
  background: #fff;
  position: absolute; top: 3rpx; left: 3rpx;
  transition: left 0.2s;
  .toggle.on & { left: 51rpx; }
}

/* 模板 */
.template-header {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: $spacing-lg;
}
.section-label { font-size: $typography-font-size-xl; font-weight: 700; color: $color-text-primary; }
.section-link { font-size: $typography-font-size-md; color: $color-semantic-info; }
.template-card { margin-bottom: 24rpx; }
.template-top {
  display: flex; justify-content: space-between; align-items: center;
  margin-bottom: $spacing-md;
}
.template-name { font-size: $typography-font-size-md; font-weight: 700; color: $color-text-primary; }
.template-use {
  padding: $spacing-xs $spacing-md;
  background: $color-semantic-info-bg; color: $color-semantic-info;
  border-radius: $border-radius-sm;
  font-size: $typography-font-size-sm; font-weight: 600;
}
.template-preview {
  font-size: $typography-font-size-base; color: $color-text-muted;
  display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 底部 */
.bottom-bar { padding: $spacing-lg 48rpx 48rpx 48rpx; background: #ffffff; }
.publish-btn {
  width: 100%; height: 96rpx;
  background: $color-semantic-info;
  color: #fff; font-size: $typography-font-size-xl; font-weight: 700;
  border-radius: $border-radius-xl; border: 0; line-height: 96rpx;
}
</style>
