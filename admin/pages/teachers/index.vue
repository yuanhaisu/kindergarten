<template>
  <el-card shadow="never">
    <template #header>
      <div class="flex justify-between">
        <span>老师管理</span>
        <el-button type="primary" @click="openCreate">添加老师</el-button>
      </div>
    </template>
    <SearchBar v-model:keyword="keyword" @search="() => { resetPage(); load() }" @reset="() => { resetPage(); load() }" />
    <DataTable
      v-model:page="page"
      v-model:page-size="pageSize"
      :data="list as unknown as Record<string, unknown>[]"
      :loading="loading"
      :total="total"
      @page-change="load"
      @size-change="load"
    >
      <el-table-column prop="nickname" label="昵称" />
      <el-table-column prop="phone" label="手机" width="130" />
      <el-table-column label="带班" min-width="200">
        <template #default="{ row }">
          {{
            (row as { classes?: { class_name: string; title_label: string }[] }).classes
              ?.map((c) => `${c.class_name}(${c.title_label})`)
              .join('、') || '-'
          }}
        </template>
      </el-table-column>
      <el-table-column prop="status_label" label="状态" width="90" />
      <el-table-column label="操作" width="320" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="view((row as { id: number }).id)">详情</el-button>
          <el-button link @click="openEdit(row as TeacherItem)">编辑</el-button>
          <el-button link type="success" @click="openInviteManage((row as { id: number }).id)">邀请</el-button>
          <el-button link @click="changeStatus(row as TeacherItem)">
            {{ (row as TeacherItem).status === 1 ? '休假' : '在职' }}
          </el-button>
          <el-button link type="danger" @click="remove((row as { id: number }).id)">删除</el-button>
        </template>
      </el-table-column>
    </DataTable>
    <el-dialog v-model="visible" :title="editingId ? '编辑老师' : '添加老师'" width="480px">
      <el-form :model="form" label-width="100px">
        <el-form-item label="真实姓名"><el-input v-model="form.real_name" /></el-form-item>
        <el-form-item label="昵称"><el-input v-model="form.nickname" /></el-form-item>
        <el-form-item label="手机"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item label="性别">
          <el-radio-group v-model="form.gender">
            <el-radio :value="1">男</el-radio>
            <el-radio :value="2">女</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="带班">
          <div style="width:100%">
            <div v-for="(item, i) in form.class_assignments" :key="i" style="display:flex;gap:8px;margin-bottom:8px">
              <el-select v-model="item.class_id" placeholder="选择班级" style="flex:1">
                <el-option v-for="c in classList" :key="c.id" :label="c.name" :value="c.id" />
              </el-select>
              <el-select v-model="item.title" placeholder="职称" style="width:110px">
                <el-option v-for="t in titleOptions" :key="t.value" :label="t.label" :value="t.value" />
              </el-select>
              <el-button
                :disabled="form.class_assignments.length <= 1"
                @click="removeClassAssignment(i)"
              >删除</el-button>
            </div>
            <el-button link type="primary" @click="addClassAssignment">+ 添加班级</el-button>
          </div>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="visible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submit">保存</el-button>
      </template>
    </el-dialog>
    <el-drawer v-model="drawer" title="老师详情" size="400px">
      <el-descriptions v-if="current" :column="1" border size="small">
        <el-descriptions-item label="昵称">{{ current.nickname }}</el-descriptions-item>
        <el-descriptions-item label="手机">{{ current.phone }}</el-descriptions-item>
        <el-descriptions-item label="学历">{{ current.education }}</el-descriptions-item>
      </el-descriptions>
    </el-drawer>

    <!-- 邀请码弹窗 - 创建成功后自动弹出 -->
    <el-dialog v-model="inviteDialog" title="邀请老师加入小程序" width="480px">
      <div style="text-align: center">
        <p style="margin-bottom: 12px; color: #666">
          将此路径发送给老师，老师通过微信小程序打开即可自动绑定账号
        </p>
        <img
          v-if="qrImageUrl"
          :src="qrImageUrl"
          alt="邀请小程序码"
          style="width: 200px; height: 200px; margin-bottom: 12px; border: 1px solid #eee"
        />
        <el-input
          :model-value="inviteLink"
          readonly
          size="large"
          style="margin-bottom: 12px"
        >
          <template #append>
            <el-button @click="copyInviteLink">
              {{ copied ? '已复制' : '复制' }}
            </el-button>
          </template>
        </el-input>
        <p style="font-size: 12px; color: #999">
          邀请码：{{ inviteToken }} &nbsp;|&nbsp;
          有效期至：{{ inviteExpiry }}
        </p>
      </div>
      <template #footer>
        <el-button @click="inviteDialog = false">关闭</el-button>
        <el-button type="primary" @click="doRegenerateInvite" :loading="regenerating">
          重新生成
        </el-button>
      </template>
    </el-dialog>

    <!-- 邀请码管理弹窗 - 从列表操作进入 -->
    <el-dialog v-model="inviteManageDialog" title="邀请码管理" width="480px">
      <div v-if="!inviteManageToken" style="text-align: center; padding: 24px 0">
        <p style="color: #999; margin-bottom: 16px">该老师暂无有效邀请码</p>
        <el-button type="primary" :loading="regenerating" @click="doRegenerateForManage">
          生成邀请码
        </el-button>
      </div>
      <div v-else style="text-align: center">
        <img
          v-if="manageQrImageUrl"
          :src="manageQrImageUrl"
          alt="邀请小程序码"
          style="width: 200px; height: 200px; margin-bottom: 12px; border: 1px solid #eee"
        />
        <el-input
          :model-value="inviteManageLink"
          readonly
          size="large"
          style="margin-bottom: 12px"
        >
          <template #append>
            <el-button @click="copyInviteManageLink">
              {{ manageCopied ? '已复制' : '复制' }}
            </el-button>
          </template>
        </el-input>
        <p style="font-size: 12px; color: #999">
          邀请码：{{ inviteManageToken }} &nbsp;|&nbsp;
          状态：<el-tag :type="inviteManageStatus === 1 ? 'success' : 'info'" size="small">
            {{ inviteManageStatus === 1 ? '待使用' : '已使用' }}
          </el-tag>
          &nbsp;|&nbsp; 有效期至：{{ inviteManageExpiry }}
        </p>
      </div>
      <template #footer>
        <el-button @click="inviteManageDialog = false">关闭</el-button>
        <el-button type="primary" :loading="regenerating" @click="doRegenerateForManage">
          重新生成
        </el-button>
      </template>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import type { ClassItem, TeacherDetail, TeacherInvitation, TeacherItem } from '~/types/api'
import { getClasses } from '~/api/classes'
import {
  createTeacher,
  deleteTeacher,
  getTeacher,
  getTeacherInvitation,
  getTeachers,
  regenerateTeacherInvitation,
  updateTeacher,
  updateTeacherStatus,
} from '~/api/teachers'

const { page, pageSize, total, resetPage } = usePagination()
const loading = ref(false)
const list = ref<TeacherItem[]>([])
const keyword = ref('')

async function load() {
  loading.value = true
  try {
    const res = await getTeachers({
      page: page.value,
      page_size: pageSize.value,
      keyword: keyword.value || undefined,
    })
    list.value = res.list
    total.value = res.total
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

const visible = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const form = reactive({
  real_name: '',
  nickname: '',
  phone: '',
  gender: 2,
  education: '本科',
  major: '学前教育',
  certificate_no: '',
  join_date: '',
  class_assignments: [] as { class_id: number; title: string }[],
})

const classList = ref<ClassItem[]>([])
const titleOptions = [
  { value: 'head', label: '班主任' },
  { value: 'assistant', label: '配班老师' },
  { value: 'nursery', label: '保育员' },
]

async function openCreate() {
  editingId.value = null
  Object.assign(form, {
    real_name: '',
    nickname: '',
    phone: '',
    gender: 2,
    education: '本科',
    major: '学前教育',
    certificate_no: '',
    join_date: new Date().toISOString().slice(0, 10),
    class_assignments: [{ class_id: 0, title: 'assistant' }],
  })
  if (classList.value.length === 0) {
    try { const res = await getClasses({ page_size: 200 }); classList.value = res.list } catch { /* */ }
  }
  visible.value = true
}

function addClassAssignment() {
  form.class_assignments.push({ class_id: 0, title: 'assistant' })
}
function removeClassAssignment(i: number) {
  form.class_assignments.splice(i, 1)
}

async function openEdit(row: TeacherItem) {
  if (classList.value.length === 0) {
    try { const res = await getClasses({ page_size: 200 }); classList.value = res.list } catch { /* */ }
  }
  try {
    const detail = await getTeacher(row.id)
    editingId.value = row.id
    Object.assign(form, {
      real_name: detail.real_name || '',
      nickname: detail.nickname || '',
      phone: detail.phone || '',
      gender: detail.gender || 2,
      education: detail.education || '本科',
      major: detail.major || '学前教育',
      certificate_no: detail.certificate_no || '',
      join_date: detail.join_date || new Date().toISOString().slice(0, 10),
      class_assignments: detail.classes.map((item) => ({
        class_id: item.class_id,
        title: item.title,
      })),
    })
    visible.value = true
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载老师详情失败')
  }
}

async function submit() {
  saving.value = true
  try {
    const payload = {
      ...form,
      class_assignments: form.class_assignments.filter((a) => a.class_id > 0),
    }
    if (editingId.value) {
      await updateTeacher(editingId.value, payload)
      ElMessage.success('已更新')
    } else {
      const res = await createTeacher(payload)
      ElMessage.success('已添加')
      // 创建成功后弹出邀请码
      inviteToken.value = res.invite_code || ''
      inviteTeacherId.value = res.id
      inviteExpiry.value = (res.expires_at || '').slice(0, 16).replace('T', ' ')
      formatInviteDisplay()
      revokeQrUrl(qrImageUrl.value)
      qrImageUrl.value = await loadQrImage(res.id)
      inviteDialog.value = true
    }
    visible.value = false
    editingId.value = null
    load()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '失败')
  } finally {
    saving.value = false
  }
}

const drawer = ref(false)
const current = ref<TeacherDetail | null>(null)
async function view(id: number) {
  try {
    current.value = await getTeacher(id)
    drawer.value = true
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  }
}

async function changeStatus(row: TeacherItem) {
  const nextStatus = row.status === 1 ? 2 : 1
  const nextLabel = nextStatus === 1 ? '在职' : '休假'
  try {
    await ElMessageBox.confirm(`确定将「${row.nickname}」状态调整为${nextLabel}吗？`, '提示', { type: 'warning' })
    await updateTeacherStatus(row.id, { status: nextStatus })
    ElMessage.success('状态已更新')
    load()
  } catch {
    /* */
  }
}

async function remove(id: number) {
  try {
    await ElMessageBox.confirm('确定删除该老师？', '提示', { type: 'warning' })
    await deleteTeacher(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* */
  }
}

// --------------- 邀请码管理 ---------------
const inviteDialog = ref(false)
const inviteToken = ref('')
const inviteTeacherId = ref<number | null>(null)
const inviteLink = ref('')
const inviteExpiry = ref('')
const copied = ref(false)
const qrImageUrl = ref('')

function formatInviteDisplay() {
  inviteLink.value = `pages/index/index?invite_code=${inviteToken.value}`
}

function copyInviteLink() {
  navigator.clipboard.writeText(inviteLink.value).then(() => {
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  }).catch(() => {
    ElMessage.warning('复制失败，请手动复制')
  })
}

async function loadQrImage(id: number): Promise<string> {
  try {
    const res = await useHttp().getBlob(`/admin/teachers/${id}/invitation/qrcode`)
    const url = URL.createObjectURL(res.data as Blob)
    return url
  } catch {
    return ''
  }
}

function revokeQrUrl(url: string) {
  if (url) URL.revokeObjectURL(url)
}

const regenerating = ref(false)
async function doRegenerateInvite() {
  if (!inviteTeacherId.value) return
  regenerating.value = true
  try {
    const inv = await regenerateTeacherInvitation(inviteTeacherId.value)
    inviteToken.value = inv.token
    inviteExpiry.value = inv.expires_at.slice(0, 16).replace('T', ' ')
    formatInviteDisplay()
    revokeQrUrl(qrImageUrl.value)
    qrImageUrl.value = await loadQrImage(inviteTeacherId.value)
    ElMessage.success('邀请码已重新生成')
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '重新生成失败')
  } finally {
    regenerating.value = false
  }
}

// --------------- 邀请码管理弹窗（列表操作入口） ---------------
const inviteManageDialog = ref(false)
const inviteManageToken = ref('')
const inviteManageLink = ref('')
const inviteManageStatus = ref(0)
const inviteManageExpiry = ref('')
const inviteManageTeacherId = ref<number | null>(null)
const manageCopied = ref(false)
const manageQrImageUrl = ref('')

async function openInviteManage(id: number) {
  inviteManageTeacherId.value = id
  revokeQrUrl(manageQrImageUrl.value)
  manageQrImageUrl.value = ''
  try {
    const inv = await getTeacherInvitation(id)
    if (inv) {
      inviteManageToken.value = inv.token
      inviteManageStatus.value = inv.status
      inviteManageExpiry.value = inv.expires_at.slice(0, 16).replace('T', ' ')
      inviteManageLink.value = `pages/index/index?invite_code=${inv.token}`
      if (inv.status === 1) {
        manageQrImageUrl.value = await loadQrImage(id)
      }
    } else {
      inviteManageToken.value = ''
      inviteManageStatus.value = 0
      inviteManageExpiry.value = ''
      inviteManageLink.value = ''
    }
  } catch {
    inviteManageToken.value = ''
  }
  inviteManageDialog.value = true
}

function copyInviteManageLink() {
  navigator.clipboard.writeText(inviteManageLink.value).then(() => {
    manageCopied.value = true
    setTimeout(() => {
      manageCopied.value = false
    }, 2000)
  }).catch(() => {
    ElMessage.warning('复制失败，请手动复制')
  })
}

async function doRegenerateForManage() {
  if (!inviteManageTeacherId.value) return
  regenerating.value = true
  try {
    const inv = await regenerateTeacherInvitation(inviteManageTeacherId.value)
    inviteManageToken.value = inv.token
    inviteManageStatus.value = inv.status
    inviteManageExpiry.value = inv.expires_at.slice(0, 16).replace('T', ' ')
    inviteManageLink.value = `pages/index/index?invite_code=${inv.token}`
    revokeQrUrl(manageQrImageUrl.value)
    manageQrImageUrl.value = await loadQrImage(inviteManageTeacherId.value)
    ElMessage.success('邀请码已重新生成')
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '重新生成失败')
  } finally {
    regenerating.value = false
  }
}

onMounted(load)
</script>
