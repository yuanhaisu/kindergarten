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
import { loginPassword } from '~/api/auth'
import { resolveAuthGate } from '~/utils/authFlow'
import { safeRedirectPath } from '~/utils/authPaths'

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

async function onSubmit() {
  await formRef.value?.validate().catch(() => Promise.reject())
  loading.value = true
  try {
    const data = await loginPassword({ phone: form.phone, password: form.password })
    auth.setSession(data)
    // 契约顺序：先同意隐私政策，再完成首登改密，最后才进业务页。
    // 改密是硬门槛（服务端 40303），登录页不再提供可取消的软提示。
    await navigateTo(
      resolveAuthGate(
        {
          consentRequired: data.consent_required,
          mustChangePassword: data.user?.must_change_password,
        },
        safeRedirectPath(route.query.redirect),
      ),
    )
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
