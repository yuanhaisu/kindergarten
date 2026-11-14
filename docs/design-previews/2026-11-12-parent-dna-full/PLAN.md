# 家长端全量设计稿 · 计划

- 日期：2026-11-12
- 状态：**设计稿已出齐（31 页 + 10 个状态变体），机械审计 0 error，待用户确认**
- 前提：**DNA v1 已冻结**（`docs/design-dna/2026-11-11-kinderly/`，commit `7f859db`）
- 目标：先出齐家长端全部页面的设计稿并确认，再进入移植开发；本阶段**不改产品代码**

## 页面清单（以 `parent/src/pages.json` 为准，共 31 页）

| # | 路由 | 页面 | 模块 |
|---|---|---|---|
| 1 | pages/index/index | 首页（tab） | 首页 |
| 2 | pages/auth/login | 首页登录 | 登录 |
| 3 | pages/auth/bind | 绑定孩子 | 登录 |
| 4 | pages/auth/consent | 隐私政策与用户协议 | 登录 |
| 5 | pages/auth/policy | 政策全文 | 登录 |
| 6 | pages/notice/index | 通知列表 | 通知 |
| 7 | pages/notice/detail | 通知详情 | 通知 |
| 8 | pages/leave/index | 请假记录 | 请假 |
| 9 | pages/leave/create | 新建请假 | 请假 |
| 10 | pages/leave/detail | 请假详情 | 请假 |
| 11 | pages/growth/index | 成长时光（tab） | 成长 |
| 12 | pages/growth/detail | 成长详情 | 成长 |
| 13 | pages/growth/album | 成长相册 | 成长 |
| 14 | pages/growth/report | 成长报告 | 成长 |
| 15 | pages/album/index | 班级相册 | 相册 |
| 16 | pages/album/detail | 相册详情 | 相册 |
| 17 | pages/pickup/index | 接送记录（tab） | 接送 |
| 18 | pages/pickup/qrcode | 接送二维码 | 接送 |
| 19 | pages/pickup/authorized | 授权人管理 | 接送 |
| 20 | pages/pickup/authorized-create | 新增授权人 | 接送 |
| 21 | pages/tasks/index | 亲子任务 | 任务 |
| 22 | pages/tasks/detail | 任务详情 | 任务 |
| 23 | pages/tasks/wall | 班级墙 | 任务 |
| 24 | pages/points/index | 积分与成就 | 积分 |
| 25 | pages/class/index | 班级信息 | 班级 |
| 26 | pages/calendar/index | 班级日历 | 日历 |
| 27 | pages/student/index | 幼儿信息 | 档案 |
| 28 | pages/feedback/index | 意见反馈 | 反馈 |
| 29 | pages/settings/index | 设置 | 设置 |
| 30 | pages/messages/index | 消息 | 消息 |
| 31 | pages/profile/index | 我的（tab） | 我的 |

> `CLAUDE.md` 写的是 29 页，实际 `pages.json` 是 31 页（多了 `auth/consent`、`auth/policy`）。
> **以 `pages.json` 为准**，CLAUDE.md 的计数后续单独修正。

## 设计基线

- 规则：`docs/design-dna/2026-11-11-kinderly/DESIGN.md`（冻结版）
- 视觉参考：`docs/design-previews/2026-11-11-dna-validation/`（5 页验证集：首页 / 接送 / 成长 / 请假新建 / 成长详情）
- 结构脚手架：`docs/design-previews/2026-10-25-full-redesign/parent/`（29 个旧页面，可复制后重绘，不沿用旧视觉）

## 产物结构

```
docs/design-previews/2026-11-12-parent-dna-full/
  PLAN.md
  _shared/dna.css          # 冻结 DNA 的 token + 组件样式
  01-home/index.html ... 31-profile/index.html
  index.html               # 画廊（用 preview.png 缩略图）
  preview.png              # 每页 390×844 截图
```

- 每页 390×844；tab 页带自绘悬浮胶囊底栏；非 tab 页带返回式页头
- 列表页至少覆盖：正常态 + 空态；接送页覆盖三态（未入园 / 在园中 / 已接走）
- 表单页覆盖：默认 + 校验错误；危险操作按 DNA 用独立确认页或底部抽屉
- 文案用真实业务语，不写 lorem ipsum

## 工作流

1. ✅ 建立 `_shared/dna.css`（从 DNA v1 直接翻译，颜色/字号/圆角/阴影/动效全用 token 值）
2. ✅ 先出 5 页代表页（首页、成长、接送、我的、登录）作为批次基准，自检后继续
3. ✅ 按模块补齐其余页面；旧页面只借结构，视觉全部按 DNA 重绘
4. ⏳ 生成画廊与 390 宽截图，逐页自查（渲染成功≠可用）
5. ⏳ 交用户确认；按反馈修订；确认后冻结本批次设计稿
6. ⏳ **设计稿确认后才进入移植开发**（第一步仍是 `tokens.json` 迁移）

## Compact 说明

DNA 规则、冻结状态、页面清单都已落盘到本文件与 `docs/design-dna/`；
会话压缩只会丢对话细节，不影响本计划。压缩后从本文件 + `DESIGN.md` 继续。
