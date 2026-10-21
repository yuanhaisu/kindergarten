<template>
  <!-- 静默时零噪音：count=0 且无错误时不渲染任何东西 -->
  <div
    v-if="alert.showUrgencyBanner"
    class="flex flex-wrap items-center gap-x-3 gap-y-2 px-4 sm:px-6 py-2.5 bg-semantic-dangerBg"
    role="alert"
  >
    <el-icon :size="18" class="shrink-0 text-semantic-danger"><WarningFilled /></el-icon>
    <!-- 不用 truncate：小屏必须能读完告警文案，宁可换行也不许吞字 -->
    <div class="flex-1 min-w-[12rem] text-sm font-semibold text-text-primary">
      {{ alert.count }} 名幼儿超时未接
      <span class="sm:ml-2 font-normal text-text-body">放学后仍未被接走，请立即处理</span>
    </div>
    <button
      class="shrink-0 h-8 px-4 rounded-full bg-text-primary text-text-onBrand text-xs font-semibold hover:opacity-90 transition-opacity"
      @click="goOverdue"
    >
      查看滞留名单
    </button>
    <button
      class="shrink-0 w-8 h-8 flex items-center justify-center rounded-full text-text-body hover:bg-white/60"
      aria-label="关闭滞留提示"
      @click="alert.dismissBanner()"
    >
      <el-icon :size="16"><Close /></el-icon>
    </button>
  </div>

  <!-- 数据不可信时必须出声，不能装作 0 个滞留 -->
  <div
    v-else-if="alert.showErrorBanner"
    class="flex flex-wrap items-center gap-x-3 gap-y-2 px-4 sm:px-6 py-2.5 bg-semantic-warningBg"
    role="alert"
  >
    <el-icon :size="18" class="shrink-0 text-semantic-warning"><WarningFilled /></el-icon>
    <div class="flex-1 min-w-[12rem] text-sm font-semibold text-text-primary">
      无法获取滞留状态
      <span class="sm:ml-2 font-normal text-text-body">接送数据可能不是最新的</span>
    </div>
    <button
      class="shrink-0 h-8 px-4 rounded-full bg-text-primary text-text-onBrand text-xs font-semibold hover:opacity-90 transition-opacity"
      @click="alert.retry()"
    >
      重试
    </button>
    <button
      class="shrink-0 w-8 h-8 flex items-center justify-center rounded-full text-text-body hover:bg-white/60"
      aria-label="关闭滞留状态提示"
      @click="alert.dismissBanner()"
    >
      <el-icon :size="16"><Close /></el-icon>
    </button>
  </div>
</template>

<script setup lang="ts">
import { Close, WarningFilled } from '@element-plus/icons-vue'

const alert = usePickupAlertStore()
const router = useRouter()

function goOverdue() {
  router.push('/pickup?tab=overdue')
}
</script>
