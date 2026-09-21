const CURRENT_STUDENT_KEY = "parent_current_student_id"

/**
 * 当前选中的孩子。
 *
 * 只存 id 数字，不存整个对象 —— 对象里的姓名/班级会随 overview 一起过期，
 * 存下来反而会出现「切了孩子但名字还是旧的」。
 *
 * 注意服务端的行为：`PUT /parent/home/switch-student` 只校验归属后原样返回，
 * **不写任何存储**，所以「当前孩子」完全由本地维护。后端各接口是通过
 * query 参数 `student_id` 接受这个选择的（见 server/internal/handler/http/parent_portal.go
 * 的 parentCtx）。
 */
export function setCurrentStudentId(id) {
  const n = Number(id)
  if (!n || n <= 0) {
    uni.removeStorageSync(CURRENT_STUDENT_KEY)
    return
  }
  uni.setStorageSync(CURRENT_STUDENT_KEY, n)
}

/**
 * 返回当前孩子 id；未选择时返回 null（不是 undefined）。
 *
 * `uni.getStorageSync` 在 key 不存在时返回空字符串，而 `Number("") === 0`，
 * 所以判定必须用 `n > 0`。调用点统一写 `getCurrentStudentId() || undefined`，
 * 让 GET 请求省略该参数 —— 服务端在 student_id 缺失或不属于该家长时会回退到
 * 第一个孩子，这正是「没选过」时的期望行为。
 */
export function getCurrentStudentId() {
  const n = Number(uni.getStorageSync(CURRENT_STUDENT_KEY))
  return n > 0 ? n : null
}

export function clearCurrentStudentId() {
  uni.removeStorageSync(CURRENT_STUDENT_KEY)
}

/**
 * 把接口返回的「当前孩子」落盘。
 *
 * 首次登录、或存储被清空时，这是唯一的自愈入口：overview 返回的
 * current_student 就是服务端认定的当前孩子，把它记下来，
 * 后续页面就有 id 可传了。
 */
export function syncCurrentStudentFromOverview(data) {
  const id = data && data.current_student && data.current_student.id
  if (id) setCurrentStudentId(id)
}
