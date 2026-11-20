<template>
  <div class="rounded-xl border border-platform-border bg-platform-card p-6 shadow-platformSm sm:p-8">
    <div class="text-center">
      <img src="/brand-badge.png" alt="" class="mx-auto h-12 w-12 rounded-full">
      <h1 class="mt-4 text-2xl font-semibold tracking-tight text-platform-textPrimary">登录平台运营台</h1>
      <p class="mt-1.5 text-sm text-platform-textMuted">仅平台方账号可登录，与园所后台互不通用</p>
    </div>

    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      class="mt-7"
      @submit.prevent="onSubmit"
    >
      <el-form-item label="用户名" prop="username">
        <el-input v-model="form.username" placeholder="平台账号" autocomplete="username" />
      </el-form-item>
      <el-form-item label="密码" prop="password">
        <el-input
          v-model="form.password"
          type="password"
          placeholder="密码"
          show-password
          autocomplete="current-password"
        />
      </el-form-item>
      <el-button type="primary" class="!mt-2 !h-11 !w-full" :loading="loading" native-type="submit">
        登录
      </el-button>
    </el-form>

    <p class="mt-6 text-center text-xs text-platform-textMuted">账号由平台管理员统一开通</p>
  </div>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import { platformLogin } from '~/api/platform/auth'
import {
  PLATFORM_REFRESH_COOKIE,
  PLATFORM_REFRESH_MAX_AGE,
  PLATFORM_TOKEN_COOKIE,
  PLATFORM_TOKEN_MAX_AGE,
  buildAuthCookieOptions,
} from '~/utils/authCookie'

definePageMeta({
  layout: 'platform-auth',
  middleware: [
    // 已有登录态时不必再看登录页，直接回看板。
    () => {
      const token = useCookie<string | null>(PLATFORM_TOKEN_COOKIE, buildAuthCookieOptions())
      if (token.value) return navigateTo('/platform')
    },
  ],
})

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
    if (data.refresh_token) {
      useCookie<string | null>(
        PLATFORM_REFRESH_COOKIE,
        buildAuthCookieOptions({ maxAge: PLATFORM_REFRESH_MAX_AGE }),
      ).value = data.refresh_token
    }
    // 顶部账号菜单读取显示名；这只是展示信息，不参与鉴权。
    if (data.admin) {
      localStorage.setItem('platform_user_name', data.admin.name || data.admin.username)
    }
    const redirect = (route.query.redirect as string) || '/platform'
    await navigateTo(redirect)
  } catch (e: unknown) {
    const status = (e as { status?: number }).status
    ElMessage.error(
      status === 429
        ? '登录尝试过于频繁，请稍后再试'
        : e instanceof Error
          ? e.message
          : '登录失败',
    )
  } finally {
    loading.value = false
  }
}
</script>
