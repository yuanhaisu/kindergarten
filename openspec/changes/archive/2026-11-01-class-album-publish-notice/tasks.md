# Tasks

## 1. 后端

- [x] 1.1 `domain/message.go` 增加 `MessageTypeAlbum`；`parent/src/utils/messages.js` 增加 `album` 映射（B2 一起改）
- [x] 1.2 新增 `service/album_notify.go`：参与者→监护人→逐人 `allow_class_share` 过滤；每家长一条；内部 action_path 带 `student_id`
- [x] 1.3 `PublishAlbum` 在状态更新/审计/指标后调用通知；`msgRepo=nil` 跳过；失败只记日志
- [x] 1.4 单测：发布发消息、重复发布不重复、opt-out 家长不收/共同监护人照收、非参与者不收、action_path 合法、nil msg 不影响发布
- [x] 1.5 契约：`api-common.md` 类型枚举、`api-teacher/10-album.md` 发布副作用；登记 `TC-NEW-ALBMSG-0001..0006`

## 2. 家长端

- [x] 2.1 `utils/messages.js` 识别 `album`（标签「相册」）；消息页不改跳转逻辑
- [x] 2.2 `build:mp-weixin` exit 0；DevTools 真实点击消息 → 相册详情（真实后端）

## 3. 集成验证

- [x] 3.1 真实后端 E2E：创建/发布相册 → 家长消息列表出现 album 消息 → 点击已读 → 进入详情 200；重复发布不新增
- [x] 3.2 清理测试相册/照片/消息/审计，残留 0；三端构建、token 基线、secret 0
- [x] 3.3 对照 specs 勾选，更新 PROGRESS，archive `class-album-publish-notice`
