// 官网是纯静态站：`nuxt generate` 产出 .output/public，由现有 nginx 直接托管。
// 视觉沿用 shared-design-tokens（Ink / Amber / Cream），不引入 Tailwind —— 单页营销站
// 的版式是一次性的，语义化 CSS 比工具类更好审阅。
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: false },
  ssr: true,
  css: ['kindergarten-design-tokens/tokens-admin.css', '~/assets/css/site.css'],
  app: {
    head: {
      htmlAttrs: { lang: 'zh-CN' },
      title: '养成星球 · 幼儿园家园共育与园务管理平台',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'theme-color', content: '#F7F3E8' },
        {
          name: 'description',
          content:
            '养成星球把通知、成长记录、请假、接送与园务管理放在同一条记录线上：老师的每个动作都留下姓名、时间和结果，家长看得见，园长查得到。',
        },
        { property: 'og:type', content: 'website' },
        { property: 'og:title', content: '养成星球 · 幼儿园家园共育与园务管理平台' },
        {
          property: 'og:description',
          content: '孩子在园的每一天，都有据可查。两个微信小程序 + 一个园所后台。',
        },
        { property: 'og:image', content: '/og.png' },
        { property: 'og:locale', content: 'zh_CN' },
      ],
      link: [
        { rel: 'icon', href: '/favicon.ico', sizes: '48x48' },
        { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' },
        { rel: 'apple-touch-icon', href: '/apple-touch-icon.png' },
        { rel: 'manifest', href: '/site.webmanifest' },
      ],
    },
  },
})
