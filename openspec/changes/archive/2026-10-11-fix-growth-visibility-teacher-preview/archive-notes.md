# 归档说明（实际触点 vs 设计阶段）

> 对应 config.yaml 的 archive 要求：记录实际碰到的触点与设计阶段的差异，便于回溯。

## 实际触点（设计阶段未预见的两处）

| 差异 | 说明 |
|---|---|
| **handler 错误映射修复** | 设计只规划了 service 层校验（`GetGrowthReport` 白名单）。实现后的对抗验证发现：权限拒绝被 handler 打成 `500 内部错误`（`parent_portal.go` 的 `ParentGrowthReport` 对 `ErrParentForbidden` 直接走 `ErrInternal`）。已补：越权 → `404 成长报告不存在`、参数缺失 → `400`（bizCode 40046）。设计阶段的「契约变更」清单未含此改动 |
| **`errBadRequest` 业务码** | 补错误映射时需要业务码，取当前未占用的 `40046`（已核对 40001~40062 现有分配） |

## 实际触点（与设计一致的部分）

- `parent_portal_service.go`：可见性构造器 + feed/首页/相册/报告/未读计数五处（设计写四处，实现时发现 `new_growth_count` 与首页同源需一并收敛）
- `teacher_portal_service.go`：列表/详情响应补齐（批量查询，无 N+1）、`performance_tags` 与契约对齐（逗号串 → 数组）
- `backend/api/api-teacher/06-growth.md`：字段对齐 + 删除 `shares_count`/`media_count`（无残留消费，已全仓核对）
- `teacher/src/pages/growth/`：三页改造 + API 层死代码清理（`deleteTeacherGrowthLog` 等 4 个未调用导出移除）

## 验证中的发现（超出本变更范围、未处理）

| 发现 | 处置 |
|---|---|
| `UpdateGrowthLog` 不更新 `growth_log_students`（编辑个体记录改学生名单静默丢失） | 纳入后续变更 `teacher-growth-student-picker`（规划中） |
| 学生选择器选中态不亮（`s.id`/`student_id` 错位）、30 人班无搜索/计数、候选含非在读学生 | 同上，纳入 `teacher-growth-student-picker` |

## 验证工具与残留

- `cmd/mktoken`（签发本地 JWT，验收用）——**保留待裁决**
- `cmd/dbcheck --query`（只读 SQL）——本次新增，建议保留
- `cmd/devsql`（临时写库工具）——已删除
- 测试种子数据 `seedgrowthfeed`（5 条，class 4 / creator 13）——仍在 dev 库，标记可识别，`--reset` 可清

## 验证覆盖摘要

- 后端：`go test -count=1 ./...` 全绿（含 6 个新增纯函数单测）；`make build` 通过；lint 无本次新增
- API 级：真实数据 + 真 JWT 的方向性验证（定向记录仅目标家长可见、首页/相册隔离、报告越权 404、编辑更新不新增）
- UI 级（老师端）：CDP 真实点击完成 列表→详情→编辑入口→保存 全链路；**限制**：编辑表单的改文字因 DevTools 对小程序 textarea 的合成输入限制未能注入（语义已由 API 级覆盖）；目视截图未完成（双实例窗口焦点问题，结论均来自 DOM 断言）；家长端 UI 未验（实例需扫码）
