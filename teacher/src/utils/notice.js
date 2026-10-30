// 通知分类：列表筛选与发布表单共用一份，避免两页各维护一份映射后漂移。
// 契约见 backend/api/api-teacher/04-notices.md：分类只有 homework / bring /
// activity / health / other 五个码，中文由端上映射。
export const NOTICE_CATEGORIES = Object.freeze([
  { key: "homework", label: "作业" },
  { key: "bring", label: "带物" },
  { key: "activity", label: "活动" },
  { key: "health", label: "健康" },
  { key: "other", label: "其他" }
])

/** 分类码 → 中文；未知/空返回空串，由调用方决定兜底文案（列表「通知」、表单「其他」）。 */
export function noticeCategoryLabel(code) {
  const item = NOTICE_CATEGORIES.find(category => category.key === code)
  return item ? item.label : ""
}
