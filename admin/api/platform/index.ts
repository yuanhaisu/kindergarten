// 平台 API 的公共入口：`~/api/platform` 仍可解析。
// 布局 `layouts/platform.vue` 由 T9 统一改线，T0 不改该文件，故保留此转发层。
export * from './auth'
export * from './kindergartens'
export * from './plans'
export * from './orders'
export * from './distributors'
export * from './commissions'
export * from './subscription'
export * from './accounts'
export * from './dashboard'
export * from './logs'
export * from './types'
