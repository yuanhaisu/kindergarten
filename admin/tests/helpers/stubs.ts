import { vi } from 'vitest'

/**
 * 页面组件测试的公共全局桩。
 *
 * Element Plus 在 jsdom/happy-dom 下依赖 ResizeObserver；Nuxt 的宏
 * definePageMeta 在 Vitest 里不存在。两段桩在 8 个页面测试里逐字重复，
 * 收敛到这里，避免新增页面测试时漏掉其中一个而报出与业务无关的错。
 */

export function stubResizeObserver() {
  class ResizeObserverStub {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  vi.stubGlobal('ResizeObserver', ResizeObserverStub)
}

export function stubNuxtPageMeta() {
  vi.stubGlobal('definePageMeta', vi.fn())
}
