# 家长端接送记录列表

> **状态**: ✅已完成
> **创建**: 2026-06-09
> **关联 Skill**: 无（待建 pickup skill）

## 背景与动机

家长端接送记录接口 (`GET /parent/pickup/records`) 和后端 service 层均为 stub，返回空数组。家长无法查看孩子的接送历史。

## 数据模型

无新增表。利用已有的 `pickup_records` 表，在 `PickupRecordListRow` 中补充 `PickupRelation` 字段。

## API 设计

### 改动：`GET /parent/pickup/records`

复用 `parentCtx()` 解析家长关联的学生列表 → 按学生 ID 查询接送记录 → 分页返回。

### 新增：`PickupRepo.ListRecordsByStudentIDs`

根据学生 ID 列表查询，分页，按 `recorded_at DESC` 排序。

### 新增：`ParentPortalService.ListPickupRecords`

组装学生 ID → 调 repo → 格式化输出。ParentPortalService 新增 `pickup *PickupRepo` 依赖。

## 前端交互

家长端 `parent/src/pages/pickup/index.vue` 已有完整 UI（卡片列表、送入/接走颜色标签、接送人信息），接入后端数据即可，前端零改动。

## 边界与异常

- 家长未绑定孩子时返回空列表
- 分页默认 20 条/页，上限 50

## 验收标准

- [x] 后端编译通过（go build + make wire）
- [x] 父端编译通过
- [x] 响应字段与前端模板匹配（type / type_label / date / pickup_name / pickup_relation / recorded_at）
- [ ] 实际联调：家长登录 → 打开接送记录 → 看到历史记录
