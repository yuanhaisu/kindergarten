<template>
  <el-card class="!w-full max-w-lg shadow-lg border-0">
    <template #header>
      <div class="flex items-center justify-between gap-3">
        <span class="text-lg font-semibold text-slate-800">隐私政策与用户协议</span>
        <el-tag v-if="status.policy_version" type="info" effect="plain">
          版本 {{ status.policy_version }}
        </el-tag>
      </div>
    </template>

    <p class="text-sm leading-relaxed text-slate-600">
      在使用养成星球管理后台前，请阅读并同意以下内容。
    </p>

    <ul class="mt-4 space-y-3 text-sm leading-relaxed text-slate-600">
      <li>
        <strong class="text-slate-800">我们处理哪些信息</strong><br />
        教师与家长账号、幼儿档案、照片、出勤、接送与园务记录。
      </li>
      <li>
        <strong class="text-slate-800">谁可以看到</strong><br />
        幼儿照片默认只给本班家长、老师与园所管理者；班内共享可随时撤回。
      </li>
      <li>
        <strong class="text-slate-800">你的权利</strong><br />
        可查询、更正、删除信息，或通过邮箱申请注销。
      </li>
    </ul>

    <div class="mt-4 flex flex-col gap-1 text-sm">
      <a :href="`${siteUrl}/privacy`" target="_blank" rel="noopener" class="text-brand-primaryDark underline">
        查看完整《隐私政策》
      </a>
      <a :href="`${siteUrl}/terms`" target="_blank" rel="noopener" class="text-brand-primaryDark underline">
        查看《用户协议》
      </a>
    </div>

    <el-alert v-if="error" :title="error" type="error" :closable="false" class="mt-4" />

    <el-checkbox v-model="agreed" class="mt-4">
      我已阅读并同意《用户协议》和《隐私政策》
    </el-checkbox>

    <el-button
      type="primary"
      class="mt-4 w-full"
      :loading="submitting"
      :disabled="!agreed || loading"
      @click="submit"
    >
      同意并继续
    </el-button>
    <el-button text class="mt-2 w-full" @click="decline">暂不同意</el-button>
  </el-card>
</template>

<script setup lang="ts">
import { getConsentStatus, recordConsent, type ConsentStatus } from '~/api/consent'

definePageMeta({ layout: 'auth' })

const route = useRoute()
const auth = useAuthStore()
const config = useRuntimeConfig()
const siteUrl = (config.public.siteUrl as string) || ''

const status = ref<ConsentStatus>({ policy_version: '', consented: false })
const loading = ref(true)
const submitting = ref(false)
const agreed = ref(false)
const error = ref('')

function goNext() {
  const redirect = (route.query.redirect as string) || '/'
  navigateTo(redirect)
}

async function loadStatus() {
  loading.value = true
  error.value = ''
  try {
    status.value = await getConsentStatus()
    if (status.value.consented) goNext()
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : '读取政策状态失败'
  } finally {
    loading.value = false
  }
}

onMounted(loadStatus)

async function submit() {
  if (!agreed.value || submitting.value) return
  submitting.value = true
  error.value = ''
  try {
    const data = await recordConsent(status.value.policy_version)
    auth.markConsented(data)
    ElMessage.success('已同意')
    goNext()
  } catch (e: unknown) {
    const err = e as Error & { code?: number }
    if (err.code === 40030) {
      await loadStatus()
      ElMessage.warning('政策已更新，请重新阅读')
    } else {
      error.value = err.message || '提交失败，请重试'
    }
  } finally {
    submitting.value = false
  }
}

function decline() {
  auth.clearSession()
  navigateTo('/login')
}
</script>
