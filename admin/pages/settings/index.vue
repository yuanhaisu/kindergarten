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
        <el-form-item label="园所 Logo">
          <div class="flex items-center gap-4">
            <img
              v-if="kgForm.logo_url"
              :src="kgForm.logo_url"
              alt="园所 Logo"
              class="h-16 w-16 rounded-lg border border-border-subtle object-contain"
              @error="onLogoError"
            />
            <div
              v-else
              class="flex h-16 w-16 items-center justify-center rounded-lg border border-dashed border-border-subtle text-xs text-text-muted"
            >
              未上传
            </div>
            <el-upload :auto-upload="false" :show-file-list="false" accept="image/*" :on-change="onLogoPicked">
              <el-button :loading="logoUploading">上传 Logo</el-button>
            </el-upload>
            <el-button v-if="kgForm.logo_url" link type="danger" @click="kgForm.logo_url = ''">移除</el-button>
          </div>
        </el-form-item>
        <el-form-item label="品牌主题色">
          <div class="flex flex-wrap items-center gap-3">
            <el-color-picker v-model="brandColor" :predefine="[...BRAND_COLOR_PRESETS]" />
            <el-input v-model="brandColor" placeholder="留空使用默认主题，如 #F97316" class="!w-64" />
            <span
              class="h-6 w-6 rounded-full border border-border-subtle"
              :style="{ backgroundColor: brandColor || BRAND_COLOR_PRESETS[0] }"
            />
          </div>
          <div class="mt-1 text-xs text-text-muted">
            家长端/老师端登录后按此颜色展示园所品牌；留空使用默认主题
          </div>
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

    <el-card shadow="never">
      <template #header><span>家长入口二维码</span></template>
      <div class="max-w-3xl space-y-4">
        <p class="text-sm text-text-muted">
          家长扫描二维码进入本园家长端首页。二维码只代表本园入口，可下载后打印或用于入园物料。
        </p>
        <div class="flex flex-wrap items-center gap-3">
          <el-select v-model="qrEnvVersion" class="!w-48" :disabled="qrLoading">
            <el-option label="跟随服务端配置" value="" />
            <el-option label="正式版" value="release" />
            <el-option label="体验版" value="trial" />
            <el-option label="开发版" value="develop" />
          </el-select>
          <el-button type="primary" :loading="qrLoading" @click="generateQr">生成二维码</el-button>
          <el-button :disabled="!qrDownloadable" @click="downloadQr">下载二维码</el-button>
        </div>

        <el-alert
          v-if="qrState === 'error'"
          type="error"
          :closable="false"
          show-icon
          :title="qrError"
        />
        <el-alert
          v-else-if="qrState === 'missing'"
          type="warning"
          :closable="false"
          show-icon
          title="未生成"
          description="后端未返回完整的家长入口码或二维码图片，暂时无法查看或下载。请稍后重试；若持续出现请联系平台。"
        />

        <div v-if="qrImageSrc" class="flex flex-wrap items-start gap-5">
          <div class="rounded-lg border border-border-subtle bg-bg-card p-3">
            <img
              :src="qrImageSrc"
              alt="家长端园所入口二维码"
              class="block h-56 w-56 max-w-full"
            />
          </div>
          <div class="space-y-1 text-sm">
            <div class="text-text-primary">
              入口码：<span class="font-mono select-all">{{ qrData?.public_code }}</span>
            </div>
            <div class="text-text-muted">扫码进入：{{ qrData?.page }}</div>
            <div class="text-text-muted">小程序版本：{{ qrEnvLabel }}</div>
          </div>
        </div>
        <div v-else-if="qrState === 'idle'" class="text-sm text-text-muted">
          尚未生成，点击「生成二维码」获取家长入口物料。
        </div>
      </div>
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
import {
  getKindergartenInfo,
  getParentEntryQRCode,
  updateKindergartenInfo,
  type ParentEntryQRCode,
} from '~/api/kindergarten'
import { uploadFile } from '~/api/common'
import { BRAND_COLOR_PRESETS, normalizeBrandColorInput } from '~/utils/brandColor'
import { downloadBlob } from '~/utils/export'
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
const logoUploading = ref(false)

// el-color-picker 清空时回传 null；表单里统一用空串表示「回落默认主题」
const brandColor = computed<string>({
  get: () => kgForm.value?.brand_color || '',
  set: (value) => {
    if (kgForm.value) kgForm.value.brand_color = value || ''
  },
})

// Logo 预览是 1 小时签名 URL，过期会加载失败；沿用列表页的刷新策略
const onLogoError = useMediaRefresh(loadKg)

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

/** 上传后先回填签名 URL 做预览；保存时后端会归一化回 object key，避免签名过期 */
async function onLogoPicked(file: { raw?: File }) {
  const picked = file.raw
  if (!picked || !kgForm.value) return
  logoUploading.value = true
  try {
    const uploaded = await uploadFile(picked, 'logo')
    kgForm.value.logo_url = uploaded.file_url
    ElMessage.success('Logo 已上传，保存园所信息后生效')
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : 'Logo 上传失败')
  } finally {
    logoUploading.value = false
  }
}

async function saveKg() {
  if (!kgForm.value) return
  const { name, address, phone, principal, description, logo_url, brand_color } = kgForm.value
  const normalizedColor = normalizeBrandColorInput(brand_color)
  if (normalizedColor === null) {
    // 服务端会以 422 拒绝；提前提示，避免一次失败的保存请求
    ElMessage.warning('主题色需为 #RRGGBB 格式，或留空使用默认主题')
    return
  }
  kgSaving.value = true
  try {
    await updateKindergartenInfo({
      name,
      address,
      phone,
      principal,
      description,
      logo_url,
      brand_color: normalizedColor,
    })
    ElMessage.success('园所信息已保存')
    await loadKg()
  } catch (e: unknown) {
    ElMessage.error(e instanceof Error ? e.message : '保存失败')
  } finally {
    kgSaving.value = false
  }
}

// ---- 家长入口二维码（#205） ----
// 后端在首次生成时会补齐 public_code 并落库；前端只负责展示/下载，
// 任何失败都不得画占位码，避免园所把假物料发出去。
type QrState = 'idle' | 'loading' | 'ready' | 'missing' | 'error'

const QR_ENV_LABELS: Record<string, string> = {
  release: '正式版',
  trial: '体验版',
  develop: '开发版',
}

const qrState = ref<QrState>('idle')
const qrData = ref<ParentEntryQRCode | null>(null)
const qrError = ref('')
const qrEnvVersion = ref('')

const qrLoading = computed(() => qrState.value === 'loading')
const qrImageSrc = computed(() => {
  if (qrState.value !== 'ready' || !qrData.value?.image_base64) return ''
  return `data:image/png;base64,${qrData.value.image_base64}`
})
const qrDownloadable = computed(
  () => qrState.value === 'ready' && !!qrData.value?.public_code && !!qrData.value?.image_base64,
)
const qrEnvLabel = computed(() => {
  const version = qrData.value?.env_version || qrEnvVersion.value
  return version ? (QR_ENV_LABELS[version] ?? version) : '服务端配置'
})

/** 端点错误 -> 可操作提示。409/503 是园所通道、客户端部署问题，必须指向平台处理。 */
function qrFailureMessage(e: unknown): string {
  const err = e as { status?: number; message?: string }
  switch (err?.status) {
    case 409:
      return '当前园所通道与家长端小程序不匹配，无法生成二维码。请联系平台处理后再试。'
    case 503:
      return '家长端小程序未就绪，暂时无法生成二维码。请稍后重试；若持续失败请联系平台。'
    case 422:
      return '小程序版本参数不合法，请重新选择版本后再试。'
    case 403:
      return '当前账号无权生成园所二维码，请使用园长账号或联系平台。'
    default: {
      const detail = err?.message?.trim()
      return detail
        ? `生成家长入口二维码失败：${detail}。请稍后重试；若持续失败请联系平台。`
        : '生成家长入口二维码失败，请稍后重试；若持续失败请联系平台。'
    }
  }
}

async function generateQr() {
  qrState.value = 'loading'
  qrError.value = ''
  try {
    const data = await getParentEntryQRCode(qrEnvVersion.value || undefined)
    qrData.value = data
    if (!data?.public_code || !data?.image_base64) {
      // 后端理论上会自动补码；这里兜底契约违约，绝不渲染半成品或假码
      qrState.value = 'missing'
      return
    }
    qrState.value = 'ready'
  } catch (e: unknown) {
    qrData.value = null
    qrError.value = qrFailureMessage(e)
    qrState.value = 'error'
  }
}

/** 微信返回的是裸 base64；先解码成 PNG Blob 再交给浏览器下载。 */
function base64ToBlob(base64: string, mime: string): Blob {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return new Blob([bytes], { type: mime })
}

function downloadQr() {
  if (!qrDownloadable.value || !qrData.value) return
  try {
    const blob = base64ToBlob(qrData.value.image_base64, 'image/png')
    downloadBlob(blob, `家长入口二维码-${qrData.value.public_code}.png`)
  } catch (e: unknown) {
    // base64 损坏时明确提示并留痕，不静默失败
    console.error('[settings] decode parent entry qr failed', e)
    ElMessage.error('二维码图片解析失败，请重新生成')
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
