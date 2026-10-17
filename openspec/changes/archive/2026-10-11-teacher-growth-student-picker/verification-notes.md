# 验证说明（实现期发现与覆盖，供归档回溯）

> 对应 config.yaml 的 archive 要求。本文件记录实现/验证过程中**超出规划范围**的发现与修复。

## 实现期发现的两个深层缺陷（均已在实现中修复）

| # | 缺陷 | 根因 | 修复 |
|---|---|---|---|
| 1 | 编辑改名单后 `related_students` 出现**历史学生**（`[1]→[4]` 读出 `[1,4]`） | `GetGrowthLog` 与 `GrowthRelatedStudents` 的 JOIN **未过滤 `growth_log_students.deleted_at`**。该表按表结构是软删（列存在），软删行在重新插回前不会被清；关联此前从无更新路径，故该缺陷从未暴露 | JOIN 条件补 `AND gls.deleted_at IS NULL`（两处同类查询） |
| 2 | 移除后又加回同一学生**静默失败**（`[1,4]→[4]→[1]` 加不回） | 唯一索引 `idx_log_student` 覆盖软删行，普通 `Create` 撞唯一键报错，而创建路径 `_ = ...Error` 静默吞错 | 引入 `writeGrowthLogStudents`（`INSERT ... ON DUPLICATE KEY UPDATE deleted_at = NULL` 的 upsert，插入即复活），创建与更新两条路径共用；不再吞错 |

缺陷 1 的修复同时消除了「改为全班后又改回个体、历史学生幽灵复现」的隐患。

## 规划与实际的差异

| 项 | 规划 | 实际 |
|---|---|---|
| 学生接口契约位置 | design 写 `api-teacher/06-growth.md` | 实为 **`09-class-and-tasks.md`**；且该处文档与实现漂移（画的 `id`/全字段/分页 vs 实返 `student_id`/接送向字段/全量），已按「契约=实际返回」校正 |
| 任务 1.1（补 status=1） | 视为待办改动 | **核实为 HEAD 既有**（`classStudentIDs` 已含 `status=1` 与在读口径注释），两调用方共用 |
| 任务 4.2（搜索过滤纯函数单测） | 计划抽 Go 纯函数 | **判定不适用**：过滤生产实现在前端 `filteredStudents`（本地 computed）；Go 侧造测试专用函数属死代码。过滤语义由 CDP 真实输入验证覆盖 |

## 验证覆盖（超出 TC 清单的额外场景）

- 「复活软删行」全链路：`[1]→[4]→[1,4]→[4]→[1]` 每一步断言 PASS
- 「空名单拒绝状态不变」：`PUT target_type=2 + []` → `422/42200`，既有名单不受影响
- 编辑态回显末位核对：`create?id=40` → 计数「已选 2 人」，王乐乐/张朵朵高亮、其余不亮

## 环境备注

- 老师端 IDE 实例（9223）验证期间需手动登录；小程序内 storage 需注入由 `cmd/mktoken` 签发的教师 JWT（`teacher_token`）
- 页面内含滚动容器为 `body`（不是 `.body` class）；CDP 点击前需 `scrollIntoView` 并重取坐标
- 搜索框为微信自定义 `WX-INPUT`，v-model 只认 `Input.insertText` 路径，原生 setter 无效
