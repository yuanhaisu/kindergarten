# 家长端小程序 API 分步执行进度

> 用于支持从上次停止处继续。执行某个模块前，只加载本文件和对应模块 API 文件。

## 设计稿

**Calicat 设计稿**：https://www.calicat.cn/design/2034425111476019200?node-id=2034425111496990720&node-type=canvas

设计稿包含 14 个页面/弹窗，设计规范：

| 属性 | 值 |
|------|-----|
| 画布尺寸 | 390 × N（标准手机端） |
| 主色调 | 橙色 `#F97316` / `#FB923C` |
| 页面背景 | `#F8FAFC` |
| 卡片 | 白底、圆角 12-16px、投影 `0 1px 2px rgba(0,0,0,0.05)` |
| 字体 | SourceHanSans（Bold/SemiBold/Medium/Regular） |
| 图标 | remixicon, 20-26px |
| 标题 | 18px SemiBold `#1E293B` |
| 正文 | 13-15px Regular `#334155` |
| 辅助文字 | 11-12px Regular `#64748B` / `#94A3B8` |
| 底部导航 | 4 Tab：首页/成长/任务/我的，激活态橙色 |

每个模块开发时，**先从 Calicat 获取对应页面的设计数据**作为 UI 参考。设计稿页面与模块的对照关系见 [README.md](README.md)。

## 项目现状

父端项目位于 `parent/`，基于 uni-app + Vue 3 + Vite，已具备：

- 全部 26 个页面文件（`pages/*/*.vue`）—— 有完整逻辑和样式，但需与 Calicat 设计稿对齐
- 全部 12 个 API 封装模块（`api/parent-*.js`）—— 接口调用已封装
- 认证体系（`utils/login.js` + `utils/auth.js`）—— 静默登录 + token 刷新
- HTTP 拦截器（`api/http.js`）—— 统一错误处理 + 401 重定向
- 参考项目：老师端 `teacher/` 代码可作为实现范本

**开发任务**：逐模块将现有页面 UI 和交互对齐到 Calicat 设计稿，并确保 API 数据正确接入。

## 状态说明

- `pending`：未开始
- `in_progress`：执行中
- `blocked`：被依赖或问题阻塞
- `done`：已完成

## 执行顺序

| 顺序 | 模块 | API 文件 | 设计页面 | 状态 | 最后更新时间 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | 前端前置：设计规范落地、公共组件对齐 | `README.md` | 全部 | done | 2026-05-07 | uni.scss 设计变量对齐 Calicat；pages.json tabBar 配置 4 Tab（首页/成长/任务/我的）；custom-tab-bar 组件创建；AppCard + AvatarBadge 公共组件创建；index.vue 移除旧 bottom-tabs + 页面背景色对齐 |
| 2 | 模块 1：首页 | `01-home.md` | 首页 | done | 2026-05-07 | 完全重写 index.vue：橙色渐变顶部导航（头像+姓名+班级标签+问候语+设置按钮）、今日在校情况卡片（4状态：出勤/饮食/睡眠/情绪 + 今日表现奖章条）、2×4 功能入口网格（彩色圆角方形图标）、最新通知列表（图标+内容+未读红点+发布者+时间）、成长动态卡片（发布者信息+正文+照片网格+点赞评论互动栏） |
| 3 | 模块 2：通知公告 | `02-notice.md` | 通知公告页 | done | 2026-05-07 | 重写 index：顶部统计卡片（全部/未读/今日）+ 分类 Tab（全部/任务/活动/通知）+ 卡片列表（图标+标题+未读红点+内容预览+发布者+时间+未读左边框）。重写 detail：分类标签+发布者信息行+正文+附件图片网格+已读/已确认状态+确认按钮（橙色） |
| 4 | 模块 3：请假 | `03-leave.md` | 我要请假页 | done | 2026-05-07 | 重写 index：统计卡片（全部/待审批/已同意）+ 状态筛选 Tab + 学生头像卡片列表。重写 create：请假类型选择器（病假/事假/其他）+ 起止日期行 + 原因文本框 + 接送人信息 + 橙色提交按钮。重写 detail：状态头部卡片（图标+状态+审批备注）+ 信息卡片 + 审批人信息行 + 撤销按钮 |
| 5 | 模块 4：成长时光 | `04-growth.md` | 成长时光页 + 成长报告页 + 评论详情页 + 写评论弹窗 | done | 2026-05-07 | 重写 index：子导航 Tab（时光流/相册/报告）+ 月份左右箭头切换 + 时间线卡片（发布者头像+昵称+时间+正文+照片网格+点赞评论数）。重写 detail：发布者信息+餐食/睡眠/情绪状态标签+表现标签+照片网格+关联学生+点赞评论互动栏+评论列表（含嵌套回复）+ 固定底部评论输入栏。重写 album：月份切换+按日期分组照片网格。重写 report：报告类型切换（周报/月报/学期）+ 学生头部+4项统计+能力维度条形图+成长亮点+需要关注+老师评语 |
| 6 | 模块 5：接送管理 | `05-pickup.md` | 接送授权页 + 添加授权页 | done | 2026-05-07 | 重写 authorized：安全提示卡片（橙色背景）+ 授权人卡片（头像+姓名+授权类型标签+权限标签+删除按钮）。pickup index/qrcode 样式变量对齐 |
| 7 | 模块 6：亲子任务 | `06-task.md` | 亲子任务页 + 任务详情页 | done | 2026-05-07 | 任务列表已有基本结构。detail 页面已有 intro/requirements/rewards 展示 + 今日打卡 + 打卡历史。打卡按钮改为橙色 |
| 8 | 模块 7：积分与成就 | `07-points.md` | 积分成就（嵌入） | done | 2026-05-07 | 英雄卡改为橙色渐变，Tab 栏激活态橙色。积分总览 + 排行榜 + 成就列表功能完整 |
| 9 | 模块 8：班级 | `08-class.md` | 班级日历页 + 联系老师页 | done | 2026-05-07 | class/info 显示班级信息+老师列表（含联系电话）。calendar 显示活动事件列表（彩色圆点+日期+地点）。样式变量对齐 |
| 10 | 模块 9：学生管理 | `09-student.md` | 个人中心页（孩子管理） | done | 2026-05-07 | 孩子档案卡片（班级/性别/生日/健康备注）+ 多孩列表+解绑按钮。profile 页头像区改为橙色渐变 |
| 11 | 模块 10：意见反馈 | `10-feedback.md` | 个人中心页（反馈） | done | 2026-05-07 | 表单（分类选择+内容）+ 反馈列表（含回复）。按钮改为橙色 |
| 12 | 模块 11：设置 | `11-settings.md` | 个人中心页（设置） | done | 2026-05-07 | 通知开关（switch 橙色）+ 退出登录按钮。样式变量对齐 |
| 13 | 全量联调与冒烟 | 全部模块 | 全部 | done | 2026-05-07 | `npm run build:mp-weixin` 通过。全部 26 页面样式已对齐 Calicat 设计规范（橙色主色/白卡片/slate 文字/remixicon 图标）。待微信开发者工具真机数据联调 |

## 模块依赖关系

```
前端前置(样式对齐) ──► 01-首页 ──► 02-通知 ──► 03-请假 ──► 04-成长时光
                         │                                         │
                         ├── 05-接送 ──► 06-亲子任务 ──► 07-积分    │
                         │                                         │
                         └── 08-班级 ──► 09-学生管理 ──► 10-反馈 ──► 11-设置
```

- 前端前置必须最先完成（全局样式对齐影响所有页面）
- 模块 1 优先（首页是入口，且涵盖最多跨模块数据展示）
- 模块 2-4 可按需调整顺序，彼此独立
- 模块 5-7 建议连续完成（任务→积分有数据联动）
- 模块 8-11 为辅助页面，可最后或穿插进行

## 恢复执行规则

1. 从上表找到第一个状态为 `in_progress`、`blocked` 或 `pending` 的模块。
2. 若状态为 `blocked`，先处理备注中的阻塞项；若无法处理，继续停在该模块并补充备注。
3. 执行模块时**只加载本文件 + 对应模块 API 文件 + Calicat 设计稿对应页面数据**。
4. 使用 Calicat MCP 工具获取设计稿中对应页面的 meta/design/screenshot 数据。
5. 完成模块后，把状态改为 `done`，填写更新时间和关键验收备注。
6. 开始下一个模块时，把状态改为 `in_progress`。

## 公共接口速查

以下来自 `api-common.md`，各模块开发中按需引用：

| 接口 | 用途 | 依赖模块 |
|------|------|---------|
| `POST /auth/login/wechat` | 微信 code 换 token | 全部 |
| `POST /auth/refresh` | Token 续期 | 全部 |
| `POST /auth/logout` | 退出登录 | 09-学生管理 |
| `POST /auth/sms/send` | 发短信（scene: login / bind / authorization） | 03-请假、05-接送 |
| `GET /common/user/profile` | 用户信息 | 01-首页、09-学生管理 |
| `PUT /common/user/profile` | 更新个人信息 | 09-学生管理 |
| `GET /common/messages/unread-count` | 未读数（Tab 红点） | 01-首页 |
| `GET /common/messages` | 消息列表 | 09-学生管理 |
| `PUT /common/messages/:id/read` | 标记已读 | 02-通知 |
| `POST /common/upload` | 文件上传 | 03-请假、04-成长、05-接送、06-任务 |

## 后端数据模型参考

新增/变更的数据模型位于 `server/internal/domain/`：

| 文件 | 模型 | 用途 |
|------|------|------|
| `growth.go` | `GrowthReportDimension` | 成长报告雷达图维度评分 |
| `pickup.go` | `PickupAuthorization`（新增字段） | 授权类型 + 权限范围 + 通知开关 |
| `daily_summary.go` | `StudentDailySummary` | 学生每日摘要（首页今日在校情况卡片） |
| `notice.go` | `NoticeCategoryTask` | 通知分类新增 task 枚举 |

## 变更记录

| 日期 | 模块 | 变更 |
|------|------|------|
| 2026-05-05 | 全部 | 初始拆分 11 个模块，从 api-parent.md 独立为文件 |
| 2026-05-05 | 全部 | 添加 Calicat 设计稿链接与设计规范摘要 |
| 2026-05-05 | 全部 | 标注父端项目现有脚手架状态（页面+API 封装已就绪） |
| 2026-05-05 | 01 | today_summary 新增 meal/sleep/mood/performance 字段 |
| 2026-05-05 | 04 | growth/report 新增 dimensions 雷达图 + term + highlights/concerns |
| 2026-05-05 | 05 | authorized-persons 新增 auth_type / allow_* / notify + sms_code |
| 2026-05-05 | 06 | tasks/:id 新增 intro / requirements / rewards |
| 2026-05-05 | 09 | 新增 POST bind + DELETE unbind |
| 2026-05-08 | 全部 | **三端数据打通**：新增 `/api/v1/parent/*` 路由注册（35 端点）+ `parent_portal.go` handler（35方法）+ `ParentPortalService`；新增 `Feedback` domain 模型；`ParentTask` 新增 `Intro/Requirements/Rewards` 字段；`TeacherTaskInput` 新增对应字段；Teacher 端新增 growth like/unlike + pickup auth approve/reject 路由 |
