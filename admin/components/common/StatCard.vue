<template>
  <div
    class="rounded-xl border border-slate-200 bg-white p-4 shadow-sm hover:shadow transition-shadow flex flex-col"
  >
    <!-- 顶部：左侧信息 + 右侧图标 -->
    <div class="flex items-start justify-between">
      <div class="flex-1 min-w-0">
        <div class="text-sm text-slate-500 mb-1">{{ title }}</div>
        <div class="text-2xl font-bold text-slate-800 tabular-nums">{{ displayValue }}</div>
      </div>
      <div
        v-if="icon"
        class="w-12 h-12 shrink-0 rounded-xl flex items-center justify-center ml-3"
        :class="iconBgClass"
      >
        <el-icon :size="22" :color="iconColor || '#3B82F6'">
          <component :is="icon" />
        </el-icon>
      </div>
    </div>

    <!-- 底部：趋势指示 -->
    <div v-if="trend" class="flex items-center gap-1 mt-3 pt-2 border-t border-slate-100">
      <el-icon :size="12">
        <CaretTop v-if="trendDirection === 'up'" color="#22C55E" />
        <CaretBottom v-else-if="trendDirection === 'down'" color="#EF4444" />
        <Minus v-else color="#94A3B8" />
      </el-icon>
      <span
        class="text-xs font-medium"
        :class="[
          trendDirection === 'up' ? 'text-green-600' : '',
          trendDirection === 'down' ? 'text-red-500' : '',
          trendDirection === 'flat' ? 'text-slate-500' : '',
        ]"
      >
        {{ trend }}
      </span>
      <span class="text-xs text-slate-400">{{ trendLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { CaretBottom, CaretTop, Minus } from '@element-plus/icons-vue'
import { computed } from 'vue'

const props = defineProps<{
  title: string
  value: string | number
  icon?: any
  iconBg?: string
  iconColor?: string
  trend?: string
  trendDirection?: 'up' | 'down' | 'flat'
  trendLabel?: string
  format?: 'number' | 'percent' | 'raw'
}>()

const displayValue = computed(() => {
  if (props.format === 'raw') return props.value
  if (typeof props.value === 'number') return props.value.toLocaleString()
  return props.value
})

const iconBgClass = computed(() => {
  if (props.iconBg) return props.iconBg
  return 'bg-blue-50'
})
</script>
