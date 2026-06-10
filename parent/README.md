# 幼儿园家长端小程序

基于 uni-app + Vue 3 + Vite 的微信小程序，服务于幼儿园家长用户。

## 快速开始

```bash
npm install
npm run dev:mp-weixin
```

微信开发者工具导入 `dist/dev/mp-weixin` 目录。

## 环境配置

| 文件 | 用途 |
|------|------|
| `.env.development` | 本地开发，默认连接 `127.0.0.1:8080` |
| `.env.test` | 测试环境 |
| `.env.production` | 生产环境 |

## 项目结构

```
src/
├── api/                  # 业务 API（按家长端模块拆分）
│   ├── http.js           # 请求封装（raw + auth）
│   ├── auth.js           # 登录相关接口
│   ├── parent-home.js    # 首页
│   ├── parent-notice.js  # 通知公告
│   ├── parent-leave.js   # 请假
│   ├── parent-growth.js  # 成长时光
│   ├── parent-pickup.js  # 接送管理
│   ├── parent-task.js    # 亲子任务
│   ├── parent-points.js  # 积分与成就
│   ├── parent-class.js   # 班级
│   ├── parent-student.js # 学生管理
│   ├── parent-feedback.js # 意见反馈
│   ├── parent-settings.js # 设置
│   └── parent-messages.js # 消息（公共接口）
├── config/
│   └── env.js            # 环境映射
├── utils/
│   ├── auth.js           # token/user 存取
│   └── login.js          # 静默登录 + ensureLogin
├── pages/                # 页面（26个页面）
├── components/           # 复用组件
├── App.vue
├── main.js
├── manifest.json
├── pages.json
└── uni.scss
```

## 页面路由

| 页面 | 路径 | 说明 |
|------|------|------|
| 首页 | `pages/index/index` | 概览、快捷功能、动态流 |
| 登录 | `pages/auth/phone` | 微信一键登录 |
| 绑定孩子 | `pages/auth/bind` | 邀请码/扫码绑定 |
| 通知列表 | `pages/notice/index` | 通知公告列表 |
| 通知详情 | `pages/notice/detail` | 通知详情+确认 |
| 请假列表 | `pages/leave/index` | 请假记录 |
| 请假申请 | `pages/leave/create` | 提交请假 |
| 请假详情 | `pages/leave/detail` | 请假详情+撤销 |
| 成长时光 | `pages/growth/index` | 时光流 |
| 成长详情 | `pages/growth/detail` | 日志详情+评论+点赞 |
| 成长相册 | `pages/growth/album` | 按日期分组的照片 |
| 成长报告 | `pages/growth/report` | 周报/月报 |
| 接送记录 | `pages/pickup/index` | 接送列表 |
| 接送二维码 | `pages/pickup/qrcode` | 扫码接送 |
| 授权接送人 | `pages/pickup/authorized` | 接送人管理 |
| 亲子任务 | `pages/tasks/index` | 任务列表 |
| 任务详情 | `pages/tasks/detail` | 任务详情+打卡 |
| 积分成就 | `pages/points/index` | 积分/排行榜/勋章 |
| 班级信息 | `pages/class/index` | 班级及老师信息 |
| 班级日历 | `pages/calendar/index` | 活动安排 |
| 孩子信息 | `pages/student/index` | 孩子详情+多孩管理 |
| 意见反馈 | `pages/feedback/index` | 提交+查看反馈 |
| 设置 | `pages/settings/index` | 通知开关+退出 |
| 消息 | `pages/messages/index` | 消息列表 |
| 我的 | `pages/profile/index` | 个人中心 |
| WebView | `pages/webview/index` | H5容器 |

## 认证

- 登录接口：`POST /api/v1/auth/login/wechat`（`role=parent`）
- Token 存储 key：`parent_token`
- `App.vue` 启动时自动预热 `ensureLogin()`

## 构建发布

```bash
npm run build:mp-weixin:test    # 测试包
npm run build:mp-weixin:prod    # 生产包
```
