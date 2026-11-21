import { resolveClientScope } from "../utils/client-scope.mjs"

// 客户端注册表（#163）：app_id 由 VITE_APP_ID 注入，缺省回落到构建内置值。
// 与 src/manifest.json 的 mp-weixin.appid 保持一致（测试会校验）。
const CLIENT_SCOPE = resolveClientScope(import.meta.env, {
  fallbackAppId: "wxdf989efff8796c53",
  fallbackVersion: "0.1.0"
})

const ENV_CONFIG = {
  development: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "http://127.0.0.1:8080/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "http://127.0.0.1:8080/api/v1/common/upload",
    siteURL: import.meta.env.VITE_SITE_URL || "http://127.0.0.1:3000",
    appId: CLIENT_SCOPE.appId,
    clientVersion: CLIENT_SCOPE.clientVersion,
    env: "development",
    envName: "开发环境"
  },
  test: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "https://test-api.nurtureplanet.cn/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "https://test-api.nurtureplanet.cn/api/v1/common/upload",
    siteURL: import.meta.env.VITE_SITE_URL || "https://test-www.nurtureplanet.cn",
    appId: CLIENT_SCOPE.appId,
    clientVersion: CLIENT_SCOPE.clientVersion,
    env: "test",
    envName: "测试环境"
  },
  production: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "https://api.xxx.com/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "https://api.xxx.com/api/v1/common/upload",
    siteURL: import.meta.env.VITE_SITE_URL || "https://www.xxx.com",
    appId: CLIENT_SCOPE.appId,
    clientVersion: CLIENT_SCOPE.clientVersion,
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
