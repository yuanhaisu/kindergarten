<template>
  <div class="space-y-6">
    <el-card v-loading="loading" shadow="never">
      <template #header><span>系统设置</span></template>
      <el-form v-if="form" :model="form" label-width="160px" class="max-w-3xl">
        <el-divider content-position="left">基础</el-divider>
        <el-form-item label="当前学年"><el-input v-model="form.basic.academic_year" /></el-form-item>
        <el-form-item label="学期">
          <el-select v-model="form.basic.semester" class="w-full">
            <el-option label="春季" value="spring" />
            <el-option label="秋季" value="fall" />
          </el-select>
        </el-form-item>
        <el-form-item label="学期开始"><el-input v-model="form.basic.semester_start" placeholder="YYYY-MM-DD" /></el-form-item>
        <el-form-item label="学期结束"><el-input v-model="form.basic.semester_end" placeholder="YYYY-MM-DD" /></el-form-item>
        <el-divider content-position="left">出勤</el-divider>
        <el-form-item label="入园开始"><el-input v-model="form.attendance.arrive_start_time" /></el-form-item>
        <el-form-item label="入园结束"><el-input v-model="form.attendance.arrive_end_time" /></el-form-item>
        <el-form-item label="离园开始"><el-input v-model="form.attendance.pickup_start_time" /></el-form-item>
        <el-form-item label="迟到阈值(分)">
          <el-input-number v-model="form.attendance.late_threshold_minutes" :min="0" class="!w-full" />
        </el-form-item>
        <el-divider content-position="left">通知 / 成长 / 接送</el-divider>
        <el-form-item label="通知需审核"><el-switch v-model="form.notice.need_audit" /></el-form-item>
        <el-form-item label="成长需审核"><el-switch v-model="form.growth.need_audit" /></el-form-item>
        <el-form-item label="接送授权需审核"><el-switch v-model="form.pickup.need_authorization_audit" /></el-form-item>
        <template v-if="canEditAlbum">
          <el-divider content-position="left">班级相册</el-divider>
          <el-form-item label="允许家长浏览全班相册">
            <div class="flex flex-col gap-1">
              <el-switch v-model="form.album.classwide_enabled" />
              <span class="text-xs leading-5 text-slate-500">
                开启后家长可在班级相册切换查看全班照片；默认关闭，仍尊重每位家长的班内共享设置
              </span>
            </div>
          </el-form-item>
        </template>
        <el-divider content-position="left">晨检</el-divider>
        <el-form-item label="要求全员测温"><el-switch v-model="form.health.temperature_required" /></el-form-item>
        <el-form-item label="发烧阈值(°C)">
          <el-input-number
            v-model="form.health.fever_threshold"
            :min="35"
            :max="42"
            :step="0.1"
            :precision="1"
            class="!w-full"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card v-loading="kgLoading" shadow="never">
      <template #header><span>园所信息</span></template>
      <el-form v-if="kgForm" :model="kgForm" label-width="120px" class="max-w-3xl">
        <el-form-item label="园所名称"><el-input v-model="kgForm.name" /></el-form-item>
        <el-form-item label="地址"><el-input v-model="kgForm.address" /></el-form-item>
        <el-form-item label="电话"><el-input v-model="kgForm.phone" /></el-form-item>
        <el-form-item label="园长"><el-input v-model="kgForm.principal" /></el-form-item>
        <el-form-item label="简介">
          <el-input v-model="kgForm.description" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item v-if="kgForm.id" label="统计">
          <span class="text-slate-600 text-sm">
            在园 {{ kgForm.current_count }} / 容量 {{ kgForm.capacity }} · 班级 {{ kgForm.class_count }} · 教师
            {{ kgForm.teacher_count }}
          </span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="kgSaving" @click="saveKg">保存园所信息</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card v-loading="yearsLoading" shadow="never">
      <template #header>
        <div class="flex items-center justify-between gap-4">
          <span>学年管理</span>
          <el-button type="primary" @click="openYearDialog">新建学年</el-button>
        </div>
      </template>
      <el-table :data="yearList" border stripe class="w-full" empty-text="暂无学年">
        <el-table-column prop="year" label="学年" min-width="120" />
        <el-table-column label="当前学年" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.is_current" type="success" size="small">是</el-tag>
            <span v-else class="text-slate-400">否</span>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="yearDialogVisible" title="新建学年" width="480px" @close="resetYearForm">
      <el-form :model="yearForm" label-width="140px">
        <el-form-item label="学年" required>
          <el-input v-model="yearForm.year" placeholder="如 2027-2028" />
        </el-form-item>
        <el-form-item label="春季学期开始" required>
          <el-input v-model="yearForm.semester_spring_start" placeholder="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="春季学期结束" required>
          <el-input v-model="yearForm.semester_spring_end" placeholder="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="秋季学期开始" required>
          <el-input v-model="yearForm.semester_fall_start" placeholder="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="秋季学期结束" required>
          <el-input v-model="yearForm.semester_fall_end" placeholder="YYYY-MM-DD" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="yearDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="yearCreating" @click="submitYear">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import type { KindergartenInfo } from '~/types/api'
import { getSettings, updateSettings, getAcademicYears, createAcademicYear } from '~/api/settings'
import { getKindergartenInfo, updateKindergartenInfo } from '~/api/kindergarten'
import type { CreateAcademicYearBody, SystemSettingsWithAlbum } from '~/api/settings'

const loading = ref(true)
const saving = ref(false)
const form = ref<SystemSettingsWithAlbum | null>(null)

// 与菜单/路由守卫同源的既有权限码，不新增
const perm = usePermissionStore()
const canEditAlbum = computed(() => perm.can('system:kindergarten'))

const kgLoading = ref(true)
const kgSaving = ref(false)
const kgForm = ref<KindergartenInfo | null>(null)

const yearsLoading = ref(true)
const yearList = ref<{ year: string; is_current: boolean }[]>([])

const yearDialogVisible = ref(false)
const yearCreating = ref(false)
const yearForm = ref<CreateAcademicYearBody>({
  year: '',
  semester_spring_start: '',
  semester_spring_end: '',
  semester_fall_start: '',
  semester_fall_end: '',
})

onMounted(async () => {
  await Promise.all([loadSettings(), loadKg(), loadYears()])
})

async function loadSettings() {
  loading.value = true
  try {
    form.value = await getSettings()
    // 兼容尚未下发 health / album 块的旧后端，避免 v-model 绑到 undefined
    if (!form.value.health) {
      form.value.health = { temperature_required: false, fever_threshold: 37.3 }
    }
    if (!form.value.album) {
      form.value.album = { classwide_enabled: false }
    }
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '加载失败')
  } finally {
    loading.value = false
  }
}

async function loadKg() {
  kgLoading.value = true
  try {
    kgForm.value = await getKindergartenInfo()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '园所信息加载失败')
  } finally {
    kgLoading.value = false
  }
}

async function loadYears() {
  yearsLoading.value = true
  try {
    const res = await getAcademicYears()
    yearList.value = res.list || []
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '学年列表加载失败')
  } finally {
    yearsLoading.value = false
  }
}

async function save() {
  if (!form.value) return
  saving.value = true
  try {
    // PUT 按块整体替换：只下发契约内的块，其余块携带的是加载值，改 album 不会重置它们
    const { attendance, notice, growth, pickup, points, health, album } = form.value
    await updateSettings({ attendance, notice, growth, pickup, points, health, album })
    ElMessage.success('已保存')
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function saveKg() {
  if (!kgForm.value) return
  kgSaving.value = true
  try {
    const { name, address, phone, principal, description, logo_url } = kgForm.value
    await updateKindergartenInfo({ name, address, phone, principal, description, logo_url })
    ElMessage.success('园所信息已保存')
    await loadKg()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    kgSaving.value = false
  }
}

function openYearDialog() {
  resetYearForm()
  yearDialogVisible.value = true
}

function resetYearForm() {
  yearForm.value = {
    year: '',
    semester_spring_start: '',
    semester_spring_end: '',
    semester_fall_start: '',
    semester_fall_end: '',
  }
}

async function submitYear() {
  const b = yearForm.value
  if (!b.year.trim()) {
    ElMessage.warning('请填写学年')
    return
  }
  yearCreating.value = true
  try {
    await createAcademicYear(b)
    ElMessage.success('学年已创建')
    yearDialogVisible.value = false
    await loadYears()
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : '创建失败'
    ElMessage.error(msg)
  } finally {
    yearCreating.value = false
  }
}
</script>
