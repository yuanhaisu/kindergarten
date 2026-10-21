# 小程序项目初始化模板（基于 `24yes-mp`）

用于快速初始化一个新的微信小程序项目。复制本模板后，按“待替换”项逐步修改，即可完成从 0 到可联调的基础工程搭建。

本模板已按当前幼儿园后端 API 约定整理，默认可对接 `server` Go 服务：

- API 基础路径：`/api/v1`
- 老师端业务路径：`/api/v1/teacher/*`
- 认证方式：`Authorization: Bearer <token>`
- 登录接口：`POST /api/v1/auth/login/wechat`
- 上传接口：`POST /api/v1/common/upload`
- 统一响应：`{ "code": 0, "message": "success", "data": ... }`

> 注意：`server/Readme.md` 中的旧版登录路径 `/api/v1/auth/wechat/login` 与 `backend/api` 文档不完全一致。当前以 `backend/api/api-common.md` 和 `backend/api/api-admin/README.md` 中已注册路由为准；如果实际后端仍未同步，初始化后只需修改 `src/api/auth.js` 的登录路径。

---

## 1. 技术栈模板

- 框架：`uni-app` + `Vue 3`
- 构建：`Vite`
- 目标平台：微信小程序（`mp-weixin`）
- 语言风格：`Composition API` 优先，页面使用 `.vue` 单文件组件
- 样式：常规 `css/scss`（如项目接入 Tailwind，可统一改为原子类）

---

## 2. 初始化命令模板

```bash
# 1) 创建项目（示例命令，按团队实际脚手架执行）
npm create uni@latest

# 2) 安装依赖
npm install

# 3) 启动微信小程序开发编译
npm run dev:mp-weixin
```

---

## 3. `package.json` 脚本模板

至少保留以下脚本，便于开发/测试/生产分环境构建：

```json
{
  "scripts": {
    "dev:h5": "uni",
    "dev:mp-weixin": "uni -p mp-weixin",
    "build:mp-weixin": "uni build -p mp-weixin",
    "build:mp-weixin:test": "uni build -p mp-weixin --mode test",
    "build:mp-weixin:prod": "uni build -p mp-weixin --mode production"
  }
}
```

---

## 4. 目录结构模板

```text
project-root/
├─ src/
│  ├─ api/                  # 业务 API（按模块拆分）
│  │  ├─ http.js            # 请求封装（raw + auth）
│  │  ├─ auth.js            # 登录相关接口
│  │  └─ xxx.js             # 业务接口
│  ├─ config/
│  │  └─ env.js             # 环境映射（dev/test/prod）
│  ├─ utils/
│  │  ├─ auth.js            # token/user 存取
│  │  └─ login.js           # 静默登录 + ensureLogin
│  ├─ pages/
│  │  ├─ index/index.vue
│  │  ├─ auth/login.vue
│  │  └─ ...                # 业务页面
│  ├─ components/           # 复用组件
│  ├─ App.vue
│  ├─ main.js
│  ├─ manifest.json
│  └─ pages.json
├─ static/                  # 静态资源
├─ .env.development
├─ .env.test
├─ .env.production
└─ README.md
```

---

## 5. 环境配置模板（必须先做）

### 5.1 环境变量文件

` .env.development `

```bash
VITE_ENV=development
NODE_ENV=development
VITE_API_BASE_URL=http://127.0.0.1:8080/api/v1
VITE_UPLOAD_URL=http://127.0.0.1:8080/api/v1/common/upload
```

` .env.test `

```bash
VITE_ENV=test
NODE_ENV=production
VITE_API_BASE_URL=https://test-api.xxx.com/api/v1
VITE_UPLOAD_URL=https://test-api.xxx.com/api/v1/common/upload
```

` .env.production `

```bash
VITE_ENV=production
NODE_ENV=production
VITE_API_BASE_URL=https://api.xxx.com/api/v1
VITE_UPLOAD_URL=https://api.xxx.com/api/v1/common/upload
```

> 微信开发者工具本机预览可使用 `127.0.0.1`。真机联调通常需要改为局域网 IP 或内网穿透 HTTPS 域名，并在开发者工具中按需关闭域名校验。

### 5.2 `src/config/env.js` 模板

```js
const ENV_CONFIG = {
  development: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "http://127.0.0.1:8080/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "http://127.0.0.1:8080/api/v1/common/upload",
    env: "development",
    envName: "开发环境"
  },
  test: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "https://test-api.xxx.com/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "https://test-api.xxx.com/api/v1/common/upload",
    env: "test",
    envName: "测试环境"
  },
  production: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "https://api.xxx.com/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "https://api.xxx.com/api/v1/common/upload",
    env: "production",
    envName: "生产环境"
  }
}

function getCurrentEnv() {
  return import.meta.env.VITE_ENV || import.meta.env.MODE || "development"
}

const currentEnv = getCurrentEnv()
const config = ENV_CONFIG[currentEnv] || ENV_CONFIG.development

export default config
export { ENV_CONFIG, currentEnv }
```

---

## 6. 小程序关键配置模板

### 6.1 `src/manifest.json` 待替换项

- `name`：项目名
- `description`：项目描述
- `appid` 与 `mp-weixin.appid`：替换为真实小程序 AppID
- `mp-weixin.setting.urlCheck`：开发阶段可 `false`，上线前按规范处理

### 6.2 `src/pages.json` 约定

- `pages[0]` 必须是启动页
- 页面级分享开关：`enableShareAppMessage` / `enableShareTimeline`
- 全局导航配置统一放在 `globalStyle`

---

## 7. 请求与登录体系模板（推荐复用）

### 7.1 `src/api/http.js`

- `rawRequest`：不带 token（用于登录、公开接口）
- `requestWithAuth`：自动附带 `Authorization: Bearer <token>`
- 统一 header（如 `platform`、`terminal_from`）避免页面散写
- 默认返回完整响应信封 `{ code, message, data, request_id }`
- 当前后端标准成功码为 `code === 0`
- 兼容 `server/Readme.md` 中旧版 `{ success, data, error }` 响应，便于后端迁移期联调

```js
import config from "@/config/env"
import { getToken, clearAuth } from "@/utils/auth"

const SUCCESS_CODE = 0

function normalizeResponse(body) {
  if (typeof body?.code === "number") return body

  if (typeof body?.success === "boolean") {
    return body.success
      ? { code: SUCCESS_CODE, message: "success", data: body.data, request_id: body.request_id }
      : {
          code: body.error?.code || 50000,
          message: body.error?.message || "请求失败",
          data: null,
          request_id: body.request_id
        }
  }

  return { code: SUCCESS_CODE, message: "success", data: body }
}

function request({ url, method = "GET", data, header = {}, auth = false }) {
  return new Promise((resolve, reject) => {
    const token = auth ? getToken() : ""
    uni.request({
      url: `${config.baseURL}${url}`,
      method,
      data,
      header: {
        "Content-Type": "application/json",
        platform: "mp-weixin",
        terminal_from: "teacher",
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
        ...header
      },
      success(res) {
        const body = normalizeResponse(res.data)

        if (res.statusCode === 401) {
          clearAuth()
          uni.reLaunch({ url: "/pages/auth/login" })
          reject(body)
          return
        }

        if (res.statusCode >= 200 && res.statusCode < 300 && body.code === SUCCESS_CODE) {
          resolve(body)
          return
        }

        uni.showToast({ title: body.message || "请求失败", icon: "none" })
        reject(body)
      },
      fail(err) {
        uni.showToast({ title: "网络异常", icon: "none" })
        reject(err)
      }
    })
  })
}

export function rawRequest(options) {
  return request({ ...options, auth: false })
}

export function requestWithAuth(options) {
  return request({ ...options, auth: true })
}
```

### 7.2 `src/utils/auth.js`

- 统一封装 `setToken/getToken/clearToken`
- 统一封装 `setRefreshToken/getRefreshToken/clearRefreshToken`
- 统一封装 `setUser/getUser/clearUser`
- 持久化字段建议固定 key，便于全局复用

```js
const TOKEN_KEY = "teacher_token"
const REFRESH_TOKEN_KEY = "teacher_refresh_token"
const USER_KEY = "teacher_user"

export function setToken(token) {
  uni.setStorageSync(TOKEN_KEY, token || "")
}

export function getToken() {
  return uni.getStorageSync(TOKEN_KEY) || ""
}

export function setRefreshToken(token) {
  uni.setStorageSync(REFRESH_TOKEN_KEY, token || "")
}

export function getRefreshToken() {
  return uni.getStorageSync(REFRESH_TOKEN_KEY) || ""
}

export function setUser(user) {
  uni.setStorageSync(USER_KEY, user || null)
}

export function getUser() {
  return uni.getStorageSync(USER_KEY) || null
}

export function clearAuth() {
  uni.removeStorageSync(TOKEN_KEY)
  uni.removeStorageSync(REFRESH_TOKEN_KEY)
  uni.removeStorageSync(USER_KEY)
}
```

### 7.3 `src/utils/login.js`

- `silentLogin`：`uni.login` -> `wxCodeLogin`
- `bindPhoneLogin`：手机号绑定后重新调用 `wxCodeLogin`
- `ensureLogin`：业务入口统一调用，未登录时自动触发静默登录
- 使用 `loginPromise` 防抖，避免并发登录
- 后端登录入参必须带 `role: "teacher"`

```js
import { wxCodeLogin } from "@/api/auth"
import { getToken, setToken, setRefreshToken, setUser } from "@/utils/auth"

let loginPromise = null

function uniLogin() {
  return new Promise((resolve, reject) => {
    uni.login({
      provider: "weixin",
      success: resolve,
      fail: reject
    })
  })
}

export async function silentLogin(extra = {}) {
  const loginRes = await uniLogin()
  const res = await wxCodeLogin({
    code: loginRes.code,
    encryptedData: extra.encryptedData,
    iv: extra.iv
  })

  const data = res.data || {}
  setToken(data.token)
  setRefreshToken(data.refresh_token)
  setUser(data.user)
  return data
}

export async function ensureLogin() {
  if (getToken()) return true
  if (!loginPromise) {
    loginPromise = silentLogin().finally(() => {
      loginPromise = null
    })
  }
  await loginPromise
  return true
}
```

### 7.4 `src/App.vue`

- 在 `onLaunch` 预热 `ensureLogin()`，减少页面首次等待

---

## 8. API 分层模板

按业务模块拆分文件，示例：

- `src/api/auth.js`：登录、用户信息
- `src/api/configure.js`：全局配置
- `src/api/comment.js`：评论/留言
- `src/api/xxx.js`：具体业务域（如任务、订单、活动）

约定：

- 页面不直接写 `uni.request`
- 页面只调用 `api/*` 导出的函数
- GET/POST/PUT/DELETE 在 API 层显式声明
- 老师端业务接口统一以 `/teacher` 开头，例如 `/teacher/home/overview`
- 公共接口统一以 `/common` 开头，例如 `/common/upload`

### 8.1 `src/api/auth.js` 示例

```js
import { rawRequest, requestWithAuth } from "@/api/http"

export function wxCodeLogin({ code, encryptedData, iv }) {
  return rawRequest({
    url: "/auth/login/wechat",
    method: "POST",
    data: {
      code,
      role: "teacher",
      encrypted_data: encryptedData,
      iv
    }
  })
}

export function refreshToken(refreshToken) {
  return rawRequest({
    url: "/auth/refresh",
    method: "POST",
    data: {
      refresh_token: refreshToken
    }
  })
}

export function fetchCurrentUser() {
  return requestWithAuth({
    url: "/common/user/profile",
    method: "GET"
  })
}
```

### 8.2 `src/api/teacher-home.js` 示例

```js
import { requestWithAuth } from "@/api/http"

export function fetchTeacherHomeOverview() {
  return requestWithAuth({
    url: "/teacher/home/overview",
    method: "GET"
  })
}

export function fetchTeacherTodos(params = {}) {
  return requestWithAuth({
    url: "/teacher/home/todos",
    method: "GET",
    data: params
  })
}
```

---

## 9. 页面开发模板（可复制）

```vue
<script setup>
import { ref } from "vue"
import { onLoad } from "@dcloudio/uni-app"
import { ensureLogin } from "@/utils/login"
import { fetchTeacherHomeOverview } from "@/api/teacher-home"

const state = ref({})

onLoad(async () => {
  await ensureLogin()
  const res = await fetchTeacherHomeOverview()
  if (res.code === 0) state.value = res.data || {}
})
</script>

<template>
  <view class="page-wrap">
    <!-- 页面内容 -->
  </view>
</template>

<style scoped lang="scss">
.page-wrap {
  min-height: 100vh;
}
</style>
```

---

## 10. 联调与发布流程模板

### 10.1 本地联调

```bash
npm run dev:mp-weixin
```

- 微信开发者工具导入目录：`dist/dev/mp-weixin`
- 开发阶段可关闭域名校验（仅本地）

### 10.2 测试包

```bash
npm run build:mp-weixin:test
```

- 构建目录：`dist/build/mp-weixin`

### 10.3 生产包

```bash
npm run build:mp-weixin:prod
```

- 微信公众平台配置合法域名（request/upload/download）
- 确保均为 HTTPS 且域名已备案

---

## 11. 初始化检查清单（Checklist）

- [ ] 已替换 `manifest.json` 的 AppID 与项目信息
- [ ] 已创建 `.env.*` 三套环境变量文件
- [ ] 已配置 `src/config/env.js` 的 `/api/v1` 测试/生产地址
- [ ] 已接入 `http.js + auth.js + login.js` 三层封装
- [ ] 已确认登录接口为 `/auth/login/wechat`，且传入 `role: "teacher"`
- [ ] 已确认接口响应按 `{ code, message, data }` 处理，成功码为 `0`
- [ ] 已在 `App.vue` 的 `onLaunch` 预热登录
- [ ] 已建立首页、授权页、webview页等基础页面
- [ ] 已按模块拆分 API 文件，不在页面里直写请求
- [ ] 已在微信公众平台配置服务器域名
- [ ] 已完成 `dev/test/prod` 三环境构建验证

---

## 12. 新项目替换位（复制后先改）

- 项目名：`<your-mini-program-name>`
- 小程序 AppID：`<your-appid>`
- 开发环境 API：`<dev-api-domain>`
- 测试环境 API：`<test-api-domain>`
- 生产环境 API：`<prod-api-domain>`
- 登录角色/终端参数：`teacher` / `terminal_from=teacher`
- 业务首屏路径：`pages/<module>/index`

---

可将本文档直接复制为新项目的 `README.md` 初始版本，再按业务补充页面、接口和组件规范。
