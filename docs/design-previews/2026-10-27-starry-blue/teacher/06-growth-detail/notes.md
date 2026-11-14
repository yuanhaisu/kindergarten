# 成长记录详情（老师端）

- 源码：`teacher/src/pages/growth/detail.vue`
- 设计系统：`user:nurture-planet-local`，`data-surface="teacher"`
- 外壳：`_shared/shell.css` + `shell.js`（`mp-navbar` / `mp-capsule` / `mp-back` / `mp-demo`）
- 方向：一条记录的「阅读页」——媒体优先，评论收口到底部固定输入栏。

## 信息层级

1. 作者行：头像（昵称首字）+ 昵称 + 发布时间
2. 标签标题：`performance_tags` 用「 · 」连接，无则不显示
3. 正文：多段，行长受控
4. 照片九宫格：单图可点开预览；关联孩子以小胶囊标在照片下；大合影带「一键全班」；AI 建议以行内「确认 / 拒绝」呈现
5. 相关孩子：头像 + 姓名，最多 6 个，超出 `+N`
6. 互动与操作：点赞数、评论数；编辑 / 分享 / 授徽章（徽章面板为展开态，可多选孩子与「作为获奖见证」）
7. 家长评论：头像、昵称、角色、时间、正文、赞 / 回复；老师回复以缩进卡片挂在评论下
8. 底部固定评论输入栏：空输入时发送按钮禁用

## 状态覆盖

- 正常：完整记录 + 5 条评论
- 空：`data-state="empty"`，评论列表清空、评论计数归零（`comment-empty`）；发首条评论后回到正常态并从 1 起算，切回其它状态时演示评论原样放回
- 加载：`data-state="loading"`，整页骨架（`skeleton-view`）
- 失败：`data-state="error"`，整页错误 + 重试（`error-view`）
- 局部失败：单张照片加载失败显示带「重试」的占位（`photo-fail__retry`）

## token 映射

- 卡片：`var(--surface)` / `var(--radius-card)` / `var(--shadow-md)`
- 主按钮：`var(--accent-gradient)`；次级：`var(--accent-wash)` + `var(--accent-deep)`
- 文字：`var(--fg)` / `var(--fg-2)` / `var(--muted)`
- 语义：`var(--success)`、`var(--warning)`、`var(--danger)` 及各自 `-wash`
- 分隔：`var(--border)`；输入框圆角 `var(--radius-xl)`

## 与现状的差异

- 照片预览改为底部弹层（原型模拟，不是真实大图查看器）
- AI 识别建议从跳转确认改为行内确认 / 拒绝，减少一次页面往返
- 授徽章原来在列表里也有入口，这里只保留详情页入口

## 遗留

- 源码没有删除 / 置顶 / 更多菜单，原型未画
- 视频未涉及（源码本页只处理图片）
- 评论回复为静态展示，未做真实回复交互
- `styles.css` 25KB，超出 20KB 建议值
