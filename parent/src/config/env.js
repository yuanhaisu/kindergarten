const ENV_CONFIG = {
  development: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "http://127.0.0.1:8080/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "http://127.0.0.1:8080/api/v1/common/upload",
    siteURL: import.meta.env.VITE_SITE_URL || "http://127.0.0.1:3000",
    env: "development",
    envName: "开发环境"
  },
  test: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "https://test-api.xxx.com/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "https://test-api.xxx.com/api/v1/common/upload",
    siteURL: import.meta.env.VITE_SITE_URL || "https://test-www.xxx.com",
    env: "test",
    envName: "测试环境"
  },
  production: {
    baseURL: import.meta.env.VITE_API_BASE_URL || "https://api.xxx.com/api/v1",
    uploadURL: import.meta.env.VITE_UPLOAD_URL || "https://api.xxx.com/api/v1/common/upload",
    siteURL: import.meta.env.VITE_SITE_URL || "https://www.xxx.com",
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
