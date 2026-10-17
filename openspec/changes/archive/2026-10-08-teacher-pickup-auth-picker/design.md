# Design

## Context

老师端手动登记弹窗已有「选授权人 / 标例外」mode-switch 和 `authorization_id` 参数（后端校验链就绪），但选人 UI 未开发。Repo 层 `ListAuthorizationsByStudentIDs` 已有完整查询能力。`GET /teacher/pickup/students` 返回的 `authorized_persons` 是逗号拼接的姓名字符串，无法用于选人。

## Goals / Non-Goals

**Goals:**
- 补全「选授权人」选人 UI（懒加载该生授权人列表）
- 送入（入园）一键登记入口
- 错误提示可操作化

**Non-Goals:**
- 不改 `POST /pickup/manual` 入参（`authorization_id` 已就绪）
- 不改 `domain/` 数据结构
- 不涉及家长端 / 管理端
- 不做授权人管理（增删改审）

## Decisions

### D1: 授权人列表 API 形态 — 独立端点 + 懒加载

**选择**：新增 `GET /teacher/pickup/students/:id/authorized-persons`

**理由**：
- 学生列表响应保持轻量（`authorized_persons` 字符串已够列表展示）
- 点「登记」时才拉取该生授权人（每个学生约 3-5 条记录，懒加载无性能压力）
- 独立端点便于 Casbin 按 `/teacher/*` 通配授权

**备选**：
- ~~扩展 `GET /pickup/students` 响应携带结构化列表~~：一次请求 16 人 × 多授权人，响应膨胀；且不是每次查看学生列表都需要授权人详情
- ~~复用 `GET /parent/pickup/authorized-persons`~~：角色不匹配，403

### D2: 响应字段与家长端对齐

返回字段（`id`/`name`/`relation`/`phone`/`avatar`/`auth_type`）与 `GET /parent/pickup/authorized-persons` 一致（`backend/api/api-parent/05-pickup.md` 已定义），数据源 `domain.PickupAuthorization`。**不新增字段**——选人只需要标识 + 展示信息。

过滤条件：`status = 1 AND review_status = 1 AND expire_at > now`（与后端 `authorizationValid` 校验一致），确保列表只出现能通过校验的授权人。

### D3: 送入一键登记 — 不走弹窗

**理由**：spec 明确「送入不记交接人、不要理由」——弹窗的整个意义是选人和填理由，送入不需要。一键直调 `POST /pickup/manual`（`type: 1`），成功后 toast + 刷新列表。

学生卡片按钮按状态派生：

```
not_arrived    →  [入园]  [接走]     (入园一键直登, 接走走弹窗)
not_picked_up  →  [接走]             (走弹窗)
picked_up      →  [再接走]           (走弹窗, 理由必填)
```

### D4: 弹窗选人交互 — radio 卡片

弹窗内「选授权人」tab 激活时渲染授权人卡片列表（头像 + 姓名 + 关系 + 脱敏手机号），点选高亮。切换模式时清空选人状态（`selectedAuthId` 置空）。未选人时提交按钮 toast 拦截。

### D5: 错误提示映射

后端错误 → 前端可操作提示的映射表：

| 后端错误 | 前端提示 |
|---|---|
| 「先登记入园再接走」 | 「该生今日尚未入园，请先登记入园（送入）后再登记接走」 |
| 「重复接走需填写理由」 | 「重复接走需填写理由」 |
| 「被指定人不在有效授权范围内」 | 「所选接送人授权已失效，请重新选择或标例外」 |

## Risks / Trade-offs

- [懒加载多一次请求] → 单次 < 50ms，且只在打开弹窗时发生；比列表膨胀好
- [送入一键误触] → toast 成功后可观察到状态变化；送入是低风险操作（可再次送入）
- [授权人过期与列表不一致] → 列表过滤条件与后端校验一致，但存在时间差（列表加载后授权过期）→ 后端校验兜底，报错走 D5 提示
