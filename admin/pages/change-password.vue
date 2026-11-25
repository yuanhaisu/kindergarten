<template>
  <el-card class="!w-full max-w-md shadow-lg border-0">
    <template #header>
      <div class="text-center text-lg font-semibold text-text-primary">设置新密码</div>
    </template>

    <p class="text-sm leading-relaxed text-text-body">
      平台方发放的是初始密码。为保障园所数据安全，请先设置仅你知道的新密码；完成前无法进入业务功能。
    </p>

    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      class="mt-4"
      @submit.prevent="onSubmit"
    >
      <el-form-item label="当前密码" prop="old_password">
        <el-input
          v-model="form.old_password"
          type="password"
          show-password
          placeholder="请输入当前密码"
          autocomplete="current-password"
        />
      </el-form-item>
      <el-form-item label="新密码" prop="new_password">
        <el-input
          v-model="form.new_password"
          type="password"
          show-password
          placeholder="至少 8 位，含大写、小写与数字"
          autocomplete="new-password"
        />
      </el-form-item>
      <el-form-item label="确认新密码" prop="confirm">
        <el-input
          v-model="form.confirm"
          type="password"
          show-password
          placeholder="再次输入新密码"
          autocomplete="new-password"
        />
      </el-form-item>

      <el-alert v-if="error" :title="error" type="error" :closable="false" class="mb-4" />

      <el-button type="primary" class="w-full" :loading="submitting" native-type="submit">
        确认修改
      </el-button>
    </el-form>
    <!-- 换错账号 / 忘记旧密码时的出口：走中心化登出，吊销服务端会话。 -->
    <el-button text class="mt-2 w-full" @click="logout">退出登录</el-button>
  </el-card>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { changePassword } from '~/api/auth'

definePageMeta({ layout: 'auth' })

const formRef = ref<FormInstance>()
const submitting = ref(false)
const error = ref('')
const auth = useAuthStore()
const logout = useAdminLogout()

const form = reactive({
  old_password: '',
  new_password: '',
  confirm: '',
})

const rules: FormRules = {
  old_password: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
  new_password: [{ required: true, message: '请输入新密码', trigger: 'blur' }],
  confirm: [{ required: true, message: '请再次输入新密码', trigger: 'blur' }],
}

/** 与服务端 validatePassword 保持同一口径，先本地拦一道；服务端仍是最终裁判。 */
function validateNewPassword(password: string): string {
  if (password.length < 8) return '新密码至少 8 位'
  if (!/[A-Z]/.test(password) || !/[a-z]/.test(password) || !/[0-9]/.test(password)) {
    return '新密码须包含大写字母、小写字母和数字'
  }
  return ''
}

async function onSubmit() {
  // 校验失败由 Element Plus 在字段上展示；这里只负责中止提交，
  // 不再把 reject 抛成未处理异常（与订单页等表单页保持同一写法）。
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  error.value = ''

  const weak = validateNewPassword(form.new_password)
  if (weak) {
    error.value = weak
    return
  }
  if (form.new_password !== form.confirm) {
    error.value = '两次输入的新密码不一致'
    return
  }

  submitting.value = true
  try {
    await changePassword({
      old_password: form.old_password,
      new_password: form.new_password,
    })
    // 改密会推进账号级失效时间（#182），当前 access / refresh 都已失效。
    // 不能继续用旧会话回原页；清掉本地会话并让用户用新口令重新登录。
    ElMessage.success('密码已更新，请使用新密码重新登录')
    auth.clearSession()
    await navigateTo({ path: '/login' })
  } catch (e: unknown) {
    // 原密码错 / 新密码强度不足等由服务端判定，必须让用户看到原因。
    error.value = e instanceof Error ? e.message : '修改密码失败'
  } finally {
    submitting.value = false
  }
}
</script>
