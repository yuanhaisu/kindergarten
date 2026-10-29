/**
 * 接送关系枚举（9 码）→ 中文。
 *
 * 契约见 backend/api/api-teacher/07-pickup.md：服务端只返回 code，中文由端上
 * 映射。存量库里还留着旧版直接提交的中文标签（B07 之前的事），读取时必须原样
 * 展示；老师代录等未知值也原样透出，不猜测、不吞成「其他」——把未知关系改写成
 * 「其他」会让老师在登记时误以为这是后端给的关系。
 */
export const RELATION_LABELS = Object.freeze({
  father: "爸爸",
  mother: "妈妈",
  grandfather: "爷爷",
  grandmother: "奶奶",
  maternal_grandfather: "外公",
  maternal_grandmother: "外婆",
  uncle: "叔叔",
  aunt: "阿姨",
  other: "其他"
})

/** 关系值（9 码或历史中文）→ 中文；空值返回空串，未知值原样返回。 */
export function relationLabel(raw) {
  const value = String(raw ?? "").trim()
  if (!value) return ""
  return RELATION_LABELS[value] || value
}

/** 结构化授权人：姓名（关系）；关系缺失时只显示姓名，不补空括号。 */
export function personLabel(name, relation) {
  const personName = String(name ?? "").trim()
  const label = relationLabel(relation)
  if (!personName) return label
  // 姓名里已经带了括号（历史数据把关系写进了姓名）就不再追加一次。
  if (personName.includes("（") || personName.includes("(")) return personName
  if (!label || label === personName) return personName
  return `${personName}（${label}）`
}

/**
 * 接送记录里的接送人：pickup_name 历史上可能已经带「（关系）」，
 * 与结构化授权人共用同一条「有括号就不追加」的规则。
 */
export function recordPersonLabel(name, relation) {
  const personName = String(name ?? "").trim()
  if (!personName) return ""
  return personLabel(personName, relation)
}
