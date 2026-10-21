<template>
  <div class="min-h-[60vh] flex items-center justify-center">
    <div class="max-w-md w-full text-center bg-bg-card rounded-2xl shadow-md p-8">
      <el-icon :size="48" class="text-text-hint"><Lock /></el-icon>
      <h1 class="mt-4 text-xl font-bold text-text-primary">暂无可访问的功能</h1>
      <p class="mt-2 text-sm text-text-body">
        当前账号未配置任何管理端权限，请联系园长在「角色管理」中调整后重新登录。
      </p>
      <el-button class="mt-6" type="primary" @click="logout">退出登录</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Lock } from '@element-plus/icons-vue'

const auth = useAuthStore()
const router = useRouter()

function logout() {
  auth.clearSession()
  router.push('/login')
}
</script>
