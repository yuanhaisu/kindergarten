// 相册分类：取值以 backend/api/api-teacher/10-album.md 为准。
// 「全部」只是列表筛选值，不传 category、不入库，因此不在 ALBUM_CATEGORIES 里。
export const ALBUM_CATEGORIES = [
  { value: "activity", label: "活动" },
  { value: "daily", label: "日常" },
  { value: "work", label: "作品" },
  { value: "festival", label: "节日" }
]

export const ALBUM_FILTERS = [{ value: "", label: "全部" }, ...ALBUM_CATEGORIES]

// 服务端会在 category_label 返回中文标签，这里只作兜底与筛选/上传胶囊的文案来源。
export function albumCategoryLabel(value) {
  const hit = ALBUM_CATEGORIES.find(item => item.value === value)
  return hit ? hit.label : ""
}

const TAG_CLASS = {
  activity: "tag--activity",
  daily: "tag--daily",
  work: "tag--work",
  festival: "tag--festival"
}

export function albumTagClass(value) {
  return TAG_CLASS[value] || "tag--daily"
}
