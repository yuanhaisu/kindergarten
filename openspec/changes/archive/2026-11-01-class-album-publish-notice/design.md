# Design

## Context

Phase 1/2-A 已有班级相册发布状态机（`PublishAlbum`）与站内消息基础设施（`sendMessageToRecipients`，唯一写入入口，失败不回滚主业务）。Phase 2-B 只需在发布成功后投递消息，并在家长端消息中心识别新类型。

## Goals / Non-Goals

**Goals**：发布成功即通知参与者家长；收件人按班内共享逐人过滤；同一相册幂等；消息可直达相册详情；失败不影响发布。

**Non-Goals**：微信推送、周报、全班广播、可见性规则变更。

## Decisions

### D1. 新增 `album` 消息类型，不复用 `growth`

`domain.MessageType` 增加 `MessageTypeAlbum = "album"`。理由：成长记录与班级相册是两个业务对象（ADR 0002），复用 `growth` 会让消息中心文案与后续统计混淆；类型是字符串列，无迁移成本。家长端 `utils/messages.js` 增加映射（标签「相册」，复用成长色调/相机图标），未知类型的兜底不变。

### D2. 收件人：参与者监护人 + 逐人班内共享过滤

- 从 `album_students` 取参与者 `student_id`；
- 经 `student_parents` 取监护人 `user_id`；
- 查 `parent_settings`（同园所）取每位监护人的 `allow_class_share`；无设置行按默认 `true`；
- 只跳过**本人**关闭班内共享的监护人。同一孩子另一位未关闭的监护人仍会收到（通知发给家庭自身，不涉及向其他家庭暴露）。
- 同一家长若关联多个参与者孩子，只发一条，`student_id` 取第一个命中的孩子，避免一跳详情落到错误孩子。

### D3. 幂等键固定为 `(user_id, type=album, reference_id=album_id, title)`

复用 `MessageRepo.Exists` 的既有判重语义，标题固定为 `"班级相册已更新"`，不拼接相册标题——否则老师改标题后重新发布会产生第二条。重复发布/下架后再发布均不重复打扰。

### D4. 发布事务外、失败只记日志

在 `UpdateStatus` 成功、审计与指标之后调用 `notifyAlbumPublished`；投递函数自行吞掉单条失败并记 slog，返回成功/失败计数（仅日志用）。发布响应不因通知失败而失败。`msgRepo == nil`（测试/未接线）时直接跳过。

### D5. 家长端只加类型映射，不改消息页逻辑

`openMessage` 已做内部路径校验、tabBar 分流、失败 toast 与已读；`/pages/album/detail?id=..&student_id=..` 不是 tabBar 页，现有 `navigateTo` 即可。只补 `TYPE_META.album`，不改分页/已读/错误态。

## 跨端触点清单

| 层 | 是否修改 | 内容 |
| --- | --- | --- |
| Go domain | 会改 | `message.go` 新类型常量 |
| repository | 不改 | 复用 `MessageRepo.Create/Exists/List` |
| service | 会改 | `PublishAlbum` 调用新 `album_notify.go` |
| handler | 不改 | 发布响应字段不变 |
| Casbin policy | 不改 | 无新路由 |
| router.go | 不改 | 无新路由 |
| backend/api/ | 会改 | `api-common.md` 类型枚举；`api-teacher/10-album.md` 发布副作用 |
| 老师端 | 不改 | 发布交互不变 |
| 家长端 | 会改 | `parent/src/utils/messages.js` 类型映射 |
| design token | 不改 | 复用 growth 色调 |
| 测试 | 会改 | Go 真实 DB 单测 + DevTools 消息→详情 |
| 指标 | 不改 | 不新增事件；复用 `album_publish` |

## Risks / Trade-offs

- **逐人过滤 vs 学生级 opt-out**：Phase 1 的可见性对学生取「任一监护人关闭即关闭」；通知面向家庭自身，逐人过滤更符合个人偏好。若产品要求家庭级一致，可改为学生级（一处查询条件）。
- **重复发布不发**：家长不会因老师修图重发而再次收到提醒；符合「同一相册不重复」的验收。
- **消息量**：一班一次发布每个监护人一条；相册参与者通常 20–30 人，可接受。

## Migration Plan

无数据库迁移（`messages.type` 为字符串）。上线顺序：后端 → 家长端；旧家长端把 `album` 当未知类型展示（仍有标题/内容），点击仍能跳转。

## Open Questions

无。
