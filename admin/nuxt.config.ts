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
  css: ['~/assets/css/main.scss'],
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || '/api/v1',
    },
  },
  elementPlus: {
    importStyle: 'css',
  },
  app: {
    head: {
      title: '养成星球 · 管理后台',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
      link: [
        {
          rel: 'stylesheet',
          href: 'https://fonts.googleapis.com/css2?family=Noto+Sans+SC:wght@400;500;600;700&display=swap',
        },
      ],
    },
  },
})
