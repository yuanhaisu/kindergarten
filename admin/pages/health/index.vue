<template>
  <div class="space-y-6">
    <!-- Date filter -->
    <el-card shadow="never">
      <div class="flex items-center gap-4">
        <span>日期：</span>
        <el-date-picker v-model="date" type="date" placeholder="选择日期" value-format="YYYY-MM-DD" @change="load" />
        <el-button @click="load" :loading="loading">刷新</el-button>
      </div>
    </el-card>

    <!-- Summary cards -->
    <div class="grid grid-cols-6 gap-4">
      <StatCard label="已入园" :value="stats.total_students" color="blue" />
      <StatCard label="待检查" :value="totalPending" color="amber" />
      <StatCard label="已完成" :value="totalCompleted" color="green" />
      <StatCard label="健康" :value="totalHealthy" color="emerald" />
      <StatCard label="需观察" :value="totalObserve" color="yellow" />
      <StatCard label="劝返" :value="totalReject" color="red" />
    </div>

    <!-- Class breakdown -->
    <el-card v-loading="loading" shadow="never">
      <template #header><span>分班统计</span></template>
      <el-table :data="stats.by_class" stripe size="small" empty-text="暂无数据">
        <el-table-column prop="class_name" label="班级" width="120" />
        <el-table-column prop="total" label="总入园" width="80" align="center" />
        <el-table-column prop="pending" label="待检查" width="80" align="center">
          <template #default="{ row }"><span class="text-amber-600 font-semibold">{{ row.pending }}</span></template>
        </el-table-column>
        <el-table-column prop="completed" label="已完成" width="80" align="center" />
        <el-table-column prop="healthy" label="健康" width="80" align="center">
          <template #default="{ row }"><span class="text-green-600">{{ row.healthy }}</span></template>
        </el-table-column>
        <el-table-column prop="observe" label="需观察" width="80" align="center">
          <template #default="{ row }"><span class="text-yellow-600" :class="{ 'font-semibold': row.observe > 0 }">{{ row.observe }}</span></template>
        </el-table-column>
        <el-table-column prop="reject" label="劝返" width="80" align="center">
          <template #default="{ row }"><span class="text-red-600" :class="{ 'font-semibold': row.reject > 0 }">{{ row.reject }}</span></template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Symptoms -->
    <el-card v-if="stats.symptoms.length > 0" shadow="never">
      <template #header><span>今日症状分布</span></template>
      <div class="flex flex-wrap gap-3">
        <el-tag
          v-for="s in stats.symptoms" :key="s.tag"
          size="large" :type="s.count >= 3 ? 'danger' : 'warning'"
        >{{ symptomLabel(s.tag) }} × {{ s.count }}</el-tag>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { onMounted } from 'vue'
import { getHealthStats } from '~/api/health'
import type { HealthStats } from '~/types/api'

const date = ref(new Date().toISOString().slice(0, 10))
const loading = ref(false)
const stats = ref<HealthStats>({ date: '', by_class: [], symptoms: [], total_students: 0 })

const totalPending = computed(() => stats.value.by_class.reduce((s, r) => s + r.pending, 0))
const totalCompleted = computed(() => stats.value.by_class.reduce((s, r) => s + r.completed, 0))
const totalHealthy = computed(() => stats.value.by_class.reduce((s, r) => s + r.healthy, 0))
const totalObserve = computed(() => stats.value.by_class.reduce((s, r) => s + r.observe, 0))
const totalReject = computed(() => stats.value.by_class.reduce((s, r) => s + r.reject, 0))

const symptomLabels: Record<string, string> = {
  fever: '发烧',
  red_throat: '喉咙发红',
  swollen_throat: '喉咙红肿',
  hand_rash: '手部红疹',
  hand_blister: '手部疱疹',
  mouth_ulcer: '口腔溃疡',
  mouth_blister: '口腔疱疹',
  red_eye: '眼睛发红',
  eye_discharge: '眼睛分泌物',
  skin_rash: '皮肤红疹',
  skin_blister: '皮肤水疱',
}

function symptomLabel(tag: string) { return symptomLabels[tag] || tag }

async function load() {
  loading.value = true
  try {
    const res = await getHealthStats({ date: date.value })
    stats.value = res as any
  } catch (e: any) {
    // ignore
  } finally { loading.value = false }
}

onMounted(load)
</script>
