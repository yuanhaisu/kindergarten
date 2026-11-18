<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-text-primary">平台账号</h1>
        <p class="mt-1 text-xs text-text-muted">
          平台账号与园所账号完全隔离。停用或改密后，该账号已登录的会话立即失效。
        </p>
      </div>
      <div class="flex items-center gap-2">
        <el-button @click="openChangeOwnPassword">修改我的密码</el-button>
        <el-button type="primary" @click="openCreate">新建账号</el-button>
      </div>
    </div>

    <SearchBar
      v-model:keyword="keyword"
      keyword-placeholder="搜索用户名 / 姓名"
      @search="search"
      @reset="search"
    />

    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="rows as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      @page-change="load"
      @size-change="onSizeChange"
    >
      <el-table-column prop="username" label="用户名" min-width="140" />
      <el-table-column prop="name" label="姓名" min-width="120">
        <template #default="{ row }">{{ row.name || '—' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === 'enabled' ? 'success' : 'info'" size="small">
            {{ row.status === 'enabled' ? '启用' : '停用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="创建时间" width="140">
        <template #default="{ row }">{{ formatDate(row.created_at as string | null) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openReset(row as { id: number; username: string })">
            重置密码
          </el-button>
          <el-button
            link
            :type="row.status === 'enabled' ? 'danger' : 'success'"
            @click="toggleStatus(row as { id: number; status: string; username: string })"
          >
            {{ row.status === 'enabled' ? '停用' : '启用' }}
          </el-button>
        </template>
      </el-table-column>
    </DataTable>

    <el-dialog v-model="createVisible" title="新建平台账号" width="460px">
      <el-form ref="createFormRef" :model="createForm" :rules="createRules" label-width="88px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="createForm.username" maxlength="64" placeholder="登录用户名" />
        </el-form-item>
        <el-form-item label="姓名">
          <el-input v-model="createForm.name" maxlength="64" placeholder="显示名（可选）" />
        </el-form-item>
        <el-form-item label="初始密码" prop="password">
          <el-input
            v-model="createForm.password"
            type="password"
            show-password
            placeholder="含大写、小写与数字，至少 8 位"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitCreate">创建</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="resetVisible" :title="`重置密码：${resetTarget?.username || ''}`" width="460px">
      <el-form ref="resetFormRef" :model="resetForm" :rules="resetRules" label-width="88px">
        <el-form-item label="新密码" prop="password">
          <el-input
            v-model="resetForm.password"
            type="password"
            show-password
            placeholder="含大写、小写与数字，至少 8 位"
          />
        </el-form-item>
      </el-form>
      <p class="text-xs text-text-muted">重置成功后该账号已登录的会话立即失效，请把新口令告知对方。</p>
      <template #footer>
        <el-button @click="resetVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitReset">重置</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="ownVisible" title="修改我的密码" width="460px">
      <el-form ref="ownFormRef" :model="ownForm" :rules="ownRules" label-width="88px">
        <el-form-item label="原密码" prop="old_password">
          <el-input v-model="ownForm.old_password" type="password" show-password />
        </el-form-item>
        <el-form-item label="新密码" prop="new_password">
          <el-input
            v-model="ownForm.new_password"
            type="password"
            show-password
            placeholder="含大写、小写与数字，至少 8 位"
          />
        </el-form-item>
      </el-form>
      <p class="text-xs text-text-muted">修改成功后需要重新登录。</p>
      <template #footer>
        <el-button @click="ownVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitOwnPassword">确认修改</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import type { FormInstance, FormRules } from 'element-plus'
import {
  changeOwnPassword,
  createAccount,
  listAccounts,
  resetAccountPassword,
  setAccountStatus,
  type PlatformAccountRow,
} from '~/api/platform/accounts'
import { buildAccountListQuery, validatePlatformPassword } from '~/utils/platformAccount'
import { PLATFORM_TOKEN_COOKIE, buildAuthCookieOptions } from '~/utils/authCookie'
import { formatDate } from '~/utils/format'

definePageMeta({ layout: 'platform' })

const keyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rows = ref<PlatformAccountRow[]>([])
const loading = ref(false)
const submitting = ref(false)

const createVisible = ref(false)
const createFormRef = ref<FormInstance>()
const createForm = reactive({ username: '', name: '', password: '' })
const createRules: FormRules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [
    { required: true, message: '请输入初始密码', trigger: 'blur' },
    { validator: passwordValidator, trigger: 'blur' },
  ],
}

const resetVisible = ref(false)
const resetFormRef = ref<FormInstance>()
const resetTarget = ref<{ id: number; username: string } | null>(null)
const resetForm = reactive({ password: '' })
const resetRules: FormRules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { validator: passwordValidator, trigger: 'blur' },
  ],
}

const ownVisible = ref(false)
const ownFormRef = ref<FormInstance>()
const ownForm = reactive({ old_password: '', new_password: '' })
const ownRules: FormRules = {
  old_password: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  new_password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { validator: passwordValidator, trigger: 'blur' },
  ],
}

function passwordValidator(_rule: unknown, value: unknown, callback: (error?: Error) => void) {
  const error = validatePlatformPassword(String(value ?? ''))
  if (error) callback(new Error(error))
  else callback()
}

async function load() {
  loading.value = true
  try {
    const data = await listAccounts(
      buildAccountListQuery({ page: page.value, pageSize: pageSize.value, keyword: keyword.value }),
    )
    rows.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  void load()
}

function onSizeChange() {
  page.value = 1
  void load()
}

function openCreate() {
  Object.assign(createForm, { username: '', name: '', password: '' })
  createVisible.value = true
}

async function submitCreate() {
  const valid = await createFormRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    await createAccount({
      username: createForm.username.trim(),
      name: createForm.name.trim() || undefined,
      password: createForm.password,
    })
    ElMessage.success('账号已创建')
    createVisible.value = false
    await load()
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '创建失败')
  } finally {
    submitting.value = false
  }
}

function openReset(row: { id: number; username: string }) {
  resetTarget.value = row
  resetForm.password = ''
  resetVisible.value = true
}

async function submitReset() {
  const valid = await resetFormRef.value?.validate().catch(() => false)
  if (!valid || !resetTarget.value) return
  submitting.value = true
  try {
    await resetAccountPassword(resetTarget.value.id, resetForm.password)
    ElMessage.success('密码已重置，该账号的登录会话已失效')
    resetVisible.value = false
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '重置失败')
  } finally {
    submitting.value = false
  }
}

async function toggleStatus(row: { id: number; status: string; username: string }) {
  const next = row.status === 'enabled' ? 'disabled' : 'enabled'
  if (next === 'disabled') {
    try {
      await ElMessageBox.confirm(
        `停用「${row.username}」后，该账号已登录的会话立即失效且无法再登录平台。确定停用？`,
        '停用平台账号',
        { type: 'warning', confirmButtonText: '停用', cancelButtonText: '取消' },
      )
    } catch {
      return
    }
  }
  try {
    await setAccountStatus(row.id, next)
    ElMessage.success(next === 'enabled' ? '已启用' : '已停用')
    await load()
  } catch (e) {
    // 停用自己 / 最后一个可用账号的守卫提示由后端给出，原样展示。
    ElMessage.error(e instanceof Error ? e.message : '操作失败')
  }
}

function openChangeOwnPassword() {
  Object.assign(ownForm, { old_password: '', new_password: '' })
  ownVisible.value = true
}

async function submitOwnPassword() {
  const valid = await ownFormRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    await changeOwnPassword(ownForm.old_password, ownForm.new_password)
    ownVisible.value = false
    // 改密后当前令牌已被服务端作废：清本地登录态并回登录页。
    useCookie<string | null>(PLATFORM_TOKEN_COOKIE, buildAuthCookieOptions()).value = null
    ElMessage.success('密码已修改，请重新登录')
    await navigateTo('/platform/login')
  } catch (e) {
    ElMessage.error(e instanceof Error ? e.message : '修改失败')
  } finally {
    submitting.value = false
  }
}

onMounted(load)
</script>
