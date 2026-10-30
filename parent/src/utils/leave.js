// 请假类型 / 状态的前端展示映射。
// 标签以 backend/api/api-parent/03-leave.md 为唯一来源：
// leave_type sick 病假 / personal 事假 / other 其他；
// status_label 全端统一 0 待审批 / 1 已同意 / 2 已拒绝 / 3 已取消。

export const LEAVE_STATUS = {
  PENDING: 0,
  APPROVED: 1,
  REJECTED: 2,
  CANCELLED: 3
}

export const LEAVE_TYPES = [
  { key: "sick", label: "病假", desc: "因病需要休息" },
  { key: "personal", label: "事假", desc: "个人或家庭事务" },
  { key: "other", label: "其他", desc: "其他原因" }
]

const TYPE_LABELS = {
  sick: "病假",
  personal: "事假",
  other: "其他"
}

const STATUS_LABELS = {
  0: "待审批",
  1: "已同意",
  2: "已拒绝",
  3: "已取消"
}

/** 列表类型标签：优先接口透传的 leave_type_label，旧后端缺字段时按代码兜底 */
export function leaveTypeLabel(item) {
  return item?.leave_type_label || TYPE_LABELS[item?.leave_type] || "其他"
}

/** 状态标签；已取消在列表仍叫「已取消」，详情页由 leaveStatusTitle 改写为「已撤销」 */
export function leaveStatusLabel(status, label) {
  return label || STATUS_LABELS[Number(status)] || "未知"
}

/** 详情页状态名：已取消（3）显示为「已撤销」 */
export function leaveStatusTitle(status, label) {
  if (Number(status) === LEAVE_STATUS.CANCELLED) return "已撤销"
  return leaveStatusLabel(status, label)
}

/** 状态语义色键，用于徽标 / 时间线 / 状态球 */
export function leaveStatusTone(status) {
  switch (Number(status)) {
    case LEAVE_STATUS.PENDING:
      return "pending"
    case LEAVE_STATUS.APPROVED:
      return "approved"
    case LEAVE_STATUS.REJECTED:
      return "rejected"
    case LEAVE_STATUS.CANCELLED:
      return "cancelled"
    default:
      return "cancelled"
  }
}
