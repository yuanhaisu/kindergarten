<template>
  <div class="rounded-xl border border-platform-border bg-platform-card p-4 shadow-platformXs sm:p-5">
    <div class="flex h-11 w-11 items-center justify-center rounded-xl sm:h-12 sm:w-12" :class="tone.iconBg">
      <el-icon :size="22" :class="tone.iconText">
        <component :is="icon" />
      </el-icon>
    </div>
    <div class="mt-4">
      <div class="text-xl font-semibold tabular-nums text-platform-textPrimary sm:text-2xl">
        {{ displayValue }}<span v-if="unit" class="ml-1 text-sm font-medium text-platform-textMuted">{{ unit }}</span>
      </div>
      <div class="mt-1 text-sm text-platform-textMuted">{{ title }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'

const props = withDefaults(
  defineProps<{
    title: string
    value: string | number
    icon: Component
    tone?: 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'neutral'
    unit?: string
  }>(),
  {
    tone: 'neutral',
    unit: '',
  },
)

const tones = {
  brand: { iconBg: 'bg-platform-brandBg', iconText: 'text-platform-brand' },
  success: { iconBg: 'bg-platform-successBg', iconText: 'text-platform-successStrong' },
  warning: { iconBg: 'bg-platform-warningBg', iconText: 'text-platform-warningStrong' },
  danger: { iconBg: 'bg-platform-dangerBg', iconText: 'text-platform-dangerStrong' },
  info: { iconBg: 'bg-platform-infoBg', iconText: 'text-platform-infoStrong' },
  neutral: { iconBg: 'bg-platform-bg', iconText: 'text-platform-textMuted' },
}

const tone = computed(() => tones[props.tone])

const displayValue = computed(() =>
  typeof props.value === 'number' ? props.value.toLocaleString() : props.value,
)
</script>
