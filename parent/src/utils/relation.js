/**
 * 接送关系枚举（9 值）。
 *
 * 契约见 backend/api/api-parent/05-pickup.md「关系枚举」：服务端只返回 code，
 * 中文标签由前端映射。接送首页、授权人列表、二维码页共用这一份，避免各页
 * 各自维护 map 后漂移。
 *
 * 存量历史数据可能是 9 个中文标签之一（旧版家长端直接提交中文），读取时
 * 必须原样展示；老师代录等未知值（如 teacher）与空值统一兜底，不把脏值
 * 直接渲染到界面。
 */
export const RELATION_OPTIONS = [
  { code: "father", label: "爸爸" },
  { code: "mother", label: "妈妈" },
  { code: "grandfather", label: "爷爷" },
  { code: "grandmother", label: "奶奶" },
  { code: "maternal_grandfather", label: "外公" },
  { code: "maternal_grandmother", label: "外婆" },
  { code: "uncle", label: "叔叔" },
  { code: "aunt", label: "阿姨" },
  { code: "other", label: "其他" }
]

const LABEL_BY_CODE = RELATION_OPTIONS.reduce((map, item) => {
  map[item.code] = item.label
  return map
}, {})

const CODE_BY_LABEL = RELATION_OPTIONS.reduce((map, item) => {
  map[item.label] = item.code
  return map
}, {})

/** 关系值（code 或存量中文）→ 中文标签；空值/未知值返回 fallback。 */
export function relationLabel(raw, fallback = "其他") {
  const value = String(raw ?? "").trim()
  if (!value) return fallback
  if (LABEL_BY_CODE[value]) return LABEL_BY_CODE[value]
  if (CODE_BY_LABEL[value]) return value
  return fallback
}

/** 关系值 → 9 个稳定 code；未知值返回空串，由调用方决定兜底。 */
export function relationCode(raw) {
  const value = String(raw ?? "").trim()
  if (LABEL_BY_CODE[value]) return value
  if (CODE_BY_LABEL[value]) return CODE_BY_LABEL[value]
  return ""
}
