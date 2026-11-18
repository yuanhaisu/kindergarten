<template>
  <el-card class="!w-full max-w-md shadow-lg border-0">
    <template #header>
      <div class="text-center">
        <div class="text-lg font-semibold text-text-primary">养成星球 · 平台运营</div>
        <div class="mt-1 text-xs text-text-muted">仅平台方账号可登录，与园所后台互不通用</div>
      </div>
    </template>
    <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="onSubmit">
      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" placeholder="平台账号" />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input v-model="form.password" type="password" placeholder="密码" show-password />
      </el-form-item>
      <el-button type="primary" class="w-full" :loading="loading" native-type="submit">登录</el-button>
    </el-form>
  </el-card>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { platformLogin } from '~/api/platform/auth'
import {
  PLATFORM_TOKEN_COOKIE,
  PLATFORM_TOKEN_MAX_AGE,
  buildAuthCookieOptions,
} from '~/utils/authCookie'

definePageMeta({ layout: 'auth' })

const formRef = ref<FormInstance>()
const loading = ref(false)
const route = useRoute()

const form = reactive({ username: '', password: '' })
const rules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function onSubmit() {
  await formRef.value?.validate().catch(() => Promise.reject())
  loading.value = true
  try {
    const data = await platformLogin(form.username, form.password)
    useCookie<string | null>(
      PLATFORM_TOKEN_COOKIE,
      buildAuthCookieOptions({ maxAge: PLATFORM_TOKEN_MAX_AGE }),
    ).value = data.token
    const redirect = (route.query.redirect as string) || '/platform/kindergartens'
    await navigateTo(redirect)
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '登录失败')
  } finally {
    loading.value = false
  }
}
</script>
