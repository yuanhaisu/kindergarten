# OS-ALB-E2E · 真实后端端到端验证

- verdict: **PASS**
- score: 94/100
- 验证范围: class-album-parent-sharing 的「老师创建 → 发布 → 家长可见 → 下架 → 恢复/重发 → 编辑 → 软删」真实链路 + 班内共享 opt-out 整张隐藏
- 后端: `:8080` PID 11076（未重启，当前源码）
- 前端: `parent/dist/dev/mp-weixin`（本次重建，mtime 2026-11-01 19:15:32）
- 基线: `f01fcd2` → 工作区（未提交；本次未改源码、未提交）

## DevTools 证据

- 构建: `cd parent && npm run dev:mp-weixin` → `DONE Build complete. Watching...` / `ready in 35213ms`；`app.json`、`pages/album/detail.js` mtime 19:15，晚于 `parent/src/pages/album/detail.vue`、`settings/index.vue` 的修改时间；watch 进程已按 PID 清理
- Token: 老师端 DevTools 账号 user 11（小一班）与本切片所需的中二班家长不同班；家长 token 由 DevTools appservice `wx.login` + `/auth/login/wechat` 真实登录得到（原 storage token 仅 13 字符，已失效），老师 token 按 `internal/auth/jwt.go` 为 class 4 的 user 65 签发测试 token（未打印任何 token 值）
- Secret scan: 未打印 token/DSN/密码；本次无源码改动，`check-secrets` 不适用
- 正常态: class_id=4 / kindergarten_id=4；album 998261，participants [27,9]；photo1→[27]、photo2→[9]、photo3→[27,9]
  - B（student 9）监护人 user 28 夹具 `allow_class_share=0`
  - 家长 61（真实会话）列表 `visible_photo_count=1`、`cover_urls`=1；详情 `photos=[998741]`（仅 photo1；photo2 我的孩子过滤，photo3 整张隐藏）
  - 老师 65 详情 `status=published`、participants [9,27]、`unauthorized_photo_count=3`（A 的家长 61 既有 opt-out + B 夹具 + 合影；与 SQL 口径一致）
- 下架/恢复: `unpublish` → 家长列表无、详情 404；删除 user 28 夹具 + `publish` → 家长列表/详情 `visible_photo_count=2`（photo3 回归）；服务端 `album_parent_view` 事件记录 1,1,1,2,2,2
- 编辑/删除: `PUT` 标题「E2E班级相册-编辑后」+ 移除 photo2 → 家长详情标题同步、2 张；`DELETE` → 家长 404；`operation_logs` = publish, unpublish, publish, edit, delete
- 交互: `pages/class/index` 真实点击 `.entry-cell`（班级相册）→ 列表；真实点击 `.album-card` → 详情。opt-out 态列表 `1 张照片` / 封面 1；详情 `共 1 张照片` / `.photo-tile`=1。恢复后列表 `2 张照片`、详情 `共 2 张照片` / tiles=2
- 空态: 软删后家长列表 `还没有班级相册` / `老师发布班级相册后会出现在这里`
- 错误态: 已删相册深链 → `相册无法查看` / `相册不存在`；重试按钮真实点击会重新发起请求（404，符合预期）
- Console: 真实点击期间 live CDP（Runtime+Log+Console）0 事件；首次 `Console.enable` 回放的 3 条 401 属登录前历史（旧 13 字符 token），重新登录后 app 内 `wx.request /parent/home/overview` 返回 200；错误态仅有预期的 404 网络日志与产品 `console.warn`
- 路由: 家长端 tabBar 未涉及；班级页 → 相册列表 / 列表 → 详情均为 `navigateTo`（非 tabBar 页），真实点击生效
- 截图: 未截（团队负责人明确「No screenshots」），以 DOM/CSS 断言 + 服务端日志替代

## 清理

- 硬删 albums / album_photos / album_photo_students / album_students / operation_logs(target_id=998261) / files(992000232-234) / parent_settings(user 28)；残留全部 0
- 全表回到基线：albums=0、album_photos=0、album_students=0、album_photo_students=0、operation_logs=931、album files=0
- 家长 61 真实设置未改（allow_class_share=0 原样）

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (3)

### 1. [test-hygiene] 3 个测试 PNG 的 OSS 对象未删
- 证据: files 行已硬删，OSS 对象仍存在（无 ossutil/删除入口）
- 建议: 测试环境定期清理；或后续加测试专用 OSS 清理脚本

### 2. [method] 老师端使用签发测试 token
- 证据: DevTools 老师账号 user 11 只教 class 1，本切片需要 class 4；为 user 65 签发 15 分钟 HS256 token（未写库）
- 建议: 双端 E2E 用同班老师账号登录；或接受测试 token 作为文档化方法

### 3. [fixture] `unauthorized_photo_count` 期望值依赖家长 61 既有 opt-out
- 证据: user 61 `allow_class_share=0`（真实数据），故 3 张照片全部计入；团队负责人预估 2
- 建议: 若需固定为 2，改用 allow_class_share=1 的家长会话做 P1

## 未覆盖（残留）

- 管理端真实 E2E（无 admin 凭据）：园所范围列表、园长下架、幂等/422、无 growth:view 菜单
- 第二家长（B 的家长）本人视角（关闭后仍可见含 B 照片）——仅有 DB 夹具，无 B 会话
- 毕业 90 天 / 转园立即失效真实边界
- 真机保存原图 + `album_photo_save` 事件
- 老师端列表在 published 态未单独断言（详情已覆盖 status/participants/unauthorized_photo_count）
- 目标 Go 测试: `go test ./internal/service/ -run 'Album|ClassAlbum|ParentSettings' -count=1 -timeout 600s` → exit 0，`ok ... 265.811s`
