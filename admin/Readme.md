# 养成星球 · 管理后台（Nuxt 3）

Vue 3 + Nuxt 3 + Element Plus + Tailwind CSS + Pinia + ECharts。

## 开发

```bash
cp .env.example .env
# 编辑 NUXT_PUBLIC_API_BASE 指向后端，例如 http://localhost:8080/api/v1

npm install
npm run dev
```

## 登录

使用公共接口 `POST /api/v1/auth/login/password`（见 `backend/api/api-common.md`），管理端账号密码登录。

## 权限说明

- 默认 `stores/permission.ts` 使用 `['*']`，便于联调；后端若在登录/资料接口返回权限码列表，可在登录成功后调用 `usePermissionStore().setCodes(...)` 收紧菜单与 `v-permission`。
- 路由级校验：在页面 `definePageMeta({ middleware: ['permission'], permission: 'xxx:view' })`（需自行按需添加）。

## 目录说明

- `api/` — 按模块封装的接口（基于 `useHttp`）
- `components/layout` — 侧栏、顶栏、面包屑
- `components/common` — 统计卡片、表格、搜索条等
- `constants/menus.ts` — 侧边栏菜单与权限码
- `middleware/auth.global.ts` — 未登录跳转 `/login`

## 构建

```bash
npm run build
npm run preview
```
