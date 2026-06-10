# 幼儿园老师端小程序

基于 `uni-app + Vue 3 + Vite` 初始化，目标平台为微信小程序。

## 开发命令

```bash
npm install
npm run dev:mp-weixin
```

微信开发者工具导入目录：`dist/dev/mp-weixin`。

## 环境配置

- 开发环境：`.env.development`
- 测试环境：`.env.test`
- 生产环境：`.env.production`

默认 API 基础路径为 `/api/v1`，老师端接口统一走 `/teacher/*`，登录接口为 `POST /auth/login/wechat`，请求会携带 `role=teacher` 与 `terminal_from=teacher`。

## 待替换

- `src/manifest.json` 中的真实微信小程序 AppID。
- `.env.test` 与 `.env.production` 中的后端域名。
- 首页快捷入口对应的业务页面路由。
