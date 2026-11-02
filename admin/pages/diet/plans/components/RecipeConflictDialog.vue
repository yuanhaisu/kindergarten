<template>
  <el-dialog v-model="visible" title="菜谱名称已存在" width="540px">
    <div class="space-y-2 text-sm leading-6 text-text-body">
      <p>菜谱库中已存在同名菜谱「{{ recipeName }}」。</p>
      <p>
        选择「替换原菜谱」会把本次配方写回菜谱库，其他周食谱不受影响；
        选择「仅本次使用」则只保留在当前周食谱中，不写入菜谱库。
      </p>
    </div>
    <template #footer>
      <el-button data-testid="recipe-conflict-cancel" @click="emit('cancel')">取消</el-button>
      <el-button data-testid="recipe-conflict-once" @click="emit('applyOnce')">仅本次使用</el-button>
      <el-button
        type="primary"
        data-testid="recipe-conflict-replace"
        :loading="saving"
        @click="emit('replace')"
      >
        替换原菜谱
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
defineProps<{
  recipeName: string
  saving: boolean
}>()

// 事件名不能是纯保留后缀：Vue 会把监听编译成 on<Event>，isEmitListener 剥掉
// 末尾的 Once 后若得到空串，读取首字符即抛异常，组件挂载失败导致整页白屏。
const emit = defineEmits<{
  replace: []
  applyOnce: []
  cancel: []
}>()

const visible = defineModel<boolean>({ required: true })
</script>
