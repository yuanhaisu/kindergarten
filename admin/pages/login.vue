<template>
  <el-card class="!w-full max-w-md shadow-lg border-0">
    <template #header>
      <div class="text-center text-lg font-semibold text-slate-800">管理后台登录</div>
    </template>
    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="onSubmit">
      <el-form-item label="手机号" prop="phone">
        <el-input v-model="form.phone" placeholder="手机号" maxlength="11" />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input v-model="form.password" type="password" placeholder="密码" show-password />
      </el-form-item>
      <el-button type="primary" class="w-full" :loading="loading" native-type="submit">
        登录
      </el-button>
    </el-form>
  </el-card>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { changePassword, loginPassword } from '~/api/auth'

definePageMeta({
  layout: 'auth',
  middleware: ['guest'],
})

const formRef = ref<FormInstance>()
const loading = ref(false)
const route = useRoute()
const auth = useAuthStore()

const form = reactive({
  phone: '',
  password: '',
})

const rules: FormRules = {
  phone: [{ required: true, message: '请输入手机号', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

/** 平台方开通园所时下发的是初始密码；首次登录提示改密（软要求，取消也能继续）。 */
async function promptInitialPasswordChange(auth: ReturnType<typeof useAuthStore>) {
  try {
    const { value } = await ElMessageBox.prompt('这是平台方发放的初始密码，请设置新密码', '首次登录', {
      inputType: 'password',
      inputPlaceholder: '至少 8 位，含大写、小写与数字',
      inputValidator: (v: string) => (v && v.length >= 8 ? true : '至少 8 位'),
      confirmButtonText: '修改',
      cancelButtonText: '稍后',
    })
    await changePassword({ old_password: form.password, new_password: value })
    if (auth.user) auth.user.must_change_password = false
    ElMessage.success('密码已更新')
  } catch {
    // 用户选择稍后处理
  }
}

async function onSubmit() {
  await formRef.value?.validate().catch(() => Promise.reject())
  loading.value = true
  try {
    const data = await loginPassword({ phone: form.phone, password: form.password })
    auth.setSession(data)
    // 未同意当前隐私政策：先完成同意，不进入业务页（首次改密也等同意之后）。
    if (data.consent_required) {
      const redirect = (route.query.redirect as string) || '/'
      await navigateTo({ path: '/consent', query: { redirect } })
      return
    }
    if (data.user.must_change_password) {
      await promptInitialPasswordChange(auth)
    }
    const redirect = (route.query.redirect as string) || '/'
    await navigateTo(redirect)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
