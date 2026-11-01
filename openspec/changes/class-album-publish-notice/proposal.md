# Proposal: 班级相册发布通知（Phase 2-B）

## Why

班级相册发布后，家长目前只能自己打开成长/相册页才发现有新内容。Phase 2-B 在发布成功时向相册参与者的家长写一条站内消息，让家长从消息中心直接进入相册详情；不引入微信订阅消息推送。

## What Changes

- 新增站内消息类型 `album`（仅站内，不发微信推送）。
- 老师发布相册（草稿/下架 → 已发布）成功后，在发布事务外为相册参与者的家长写消息：
  - 收件人只取相册参与者（`album_students`）的监护人；
  - 监护人自己的 `allow_class_share=false` 时不收（未配置按默认 true）；
  - 同一相册对同一家长只发一条，重复发布不重复发；
  - 消息带内部跳转 `/pages/album/detail?id=<album_id>&student_id=<该家长参与的孩子>`。
- 写入失败只记日志，不影响发布结果。
- 家长端消息中心识别 `album` 类型并复用现有点击跳转/已读逻辑。

## Non-Goals

- 不做微信订阅消息/短信/Push；
- 不做周报/摘要；
- 不扩大收件人到全班（全班视图是 A 的能力）；
- 不改变 Phase 1/2-A 的照片可见性与 opt-out 规则。

## Impact

- 能力：`class-album-notify`（新增）
- 契约：`backend/api/api-common.md`（消息类型）、`backend/api/api-teacher/10-album.md`（发布副作用）
- 代码：`server/internal/domain/message.go`、`teacher_album_service.go`、新增 `album_notify.go`、`parent/src/utils/messages.js`
- 数据：仅写 `messages`，无新表/新列
