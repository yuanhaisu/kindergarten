// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: true },
  ssr: false,
  /**
   * 浏览器请求同源 `/api/v1`（默认 apiBase），避免直连 :8080 触发 CORS。
   * - `vite.server.proxy`：dev 下页面由 Vite 提供，XHR 走 Vite 转发最可靠。
   * - `nitro.devProxy`：保留与 Nitro 侧一致（部分场景/工具链会走 Nitro）。
   * 生产部署请用 Nginx 反代 `/api/v1` 或设置 `NUXT_PUBLIC_API_BASE` 为完整后端地址。
   */
  vite: {
    server: {
      proxy: {
        '/api/v1': {
          target: 'http://127.0.0.1:8080',
          changeOrigin: true,
        },
      },
    },
  },
  nitro: {
    devProxy: {
      '/api/v1': {
        target: 'http://127.0.0.1:8080/api/v1',
        changeOrigin: true,
      },
    },
  },
  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss', '@element-plus/nuxt'],
  components: [
    { path: '~/components', pathPrefix: false },
  ],
  css: ['kindergarten-design-tokens/tokens-admin.css', '~/assets/css/main.scss'],
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || '/api/v1',
      // 官网地址：同意页的隐私政策/用户协议链接；生产由部署环境注入。
      siteUrl: process.env.NUXT_PUBLIC_SITE_URL || 'http://localhost:3000',
    },
  },
  elementPlus: {
    importStyle: 'css',
    // 管理端界面为中文：日历/日期选择器/分页等内置文案统一走 zh-cn
    defaultLocale: 'zh-cn',
  },
  app: {
    head: {
      title: '养成星球 · 管理后台',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'theme-color', content: '#F7F3E8' },
      ],
      link: [
        { rel: 'icon', href: '/favicon.ico', sizes: '48x48' },
        { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' },
        { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' },
      ],
    },
  },
})
