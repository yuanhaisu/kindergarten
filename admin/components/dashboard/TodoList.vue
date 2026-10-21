<template>
  <div>
    <div v-if="items.length" class="space-y-0 divide-y divide-slate-50">
      <div
        v-for="item in items"
        :key="item.id"
        class="py-3 first:pt-0 last:pb-0"
      >
        <div class="flex items-start gap-3">
          <!-- 红点 -->
          <span class="w-2 h-2 mt-1.5 rounded-full bg-red-500 shrink-0" />

          <div class="flex-1 min-w-0">
            <!-- 标题 -->
            <div class="text-sm font-semibold text-slate-800 truncate">
              {{ item.title }}
            </div>
            <!-- 描述 -->
            <div class="text-xs text-slate-500 mt-0.5 line-clamp-2">
              {{ item.description }}
            </div>
            <!-- 时间 + 操作：无落地页的待办不渲染按钮，杜绝死控件 -->
            <div class="flex items-center justify-between mt-2">
              <span class="text-xs text-slate-400">{{ item.time_ago }}</span>
              <el-button
                v-if="item.target"
                size="small"
                type="primary"
                link
                @click="go(item.target)"
              >
                {{ item.action_type === 'audit' ? '审核' : '处理' }}
              </el-button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <el-empty v-else description="暂无待办事项" :image-size="80" />
  </div>
</template>

<script setup lang="ts">
import type { TodoItem } from '~/types/api'

defineProps<{
  items: TodoItem[]
}>()

const router = useRouter()

function go(target: string) {
  router.push(target)
}
</script>
