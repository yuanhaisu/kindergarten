import { defineStore } from 'pinia'
import { getPickupOverdue } from '~/api/pickup'

/** 接送滞留是时间敏感状态，30s 足够跟上放学时段的变化 */
const POLL_MS = 30_000

/**
 * 壳层接送滞留信号的唯一状态源（侧栏徽标 + 顶部横幅共用）。
 * 只回答一个问题：现在有没有孩子超时未接、能不能信这个答案。
 *
 * 数据源用列表端点的 overdue_total（实时口径），不用 /overdue/count（累计口径，
 * 被接走也不清零）——横幅要求「请立即处理」，累计数会在孩子被接走后继续报警。
 */
export const usePickupAlertStore = defineStore('pickupAlert', () => {
  const count = ref(0)
  /**
   * idle=未取过 / ok=最近一次成功 / error=最近一次失败（禁止静默当 0 处理）
   * / forbidden=该账号无接送查看权限（403 不是故障，是本就不该看的信号）
   */
  const status = ref<'idle' | 'ok' | 'error' | 'forbidden'>('idle')
  const bannerDismissed = ref(false)
  const errorDismissed = ref(false)

  const hasOverdue = computed(() => count.value > 0)
  const showUrgencyBanner = computed(() => hasOverdue.value && !bannerDismissed.value)
  const showErrorBanner = computed(() => status.value === 'error' && !errorDismissed.value)

  let timer: ReturnType<typeof setInterval> | null = null
  let started = false

  async function fetchCount() {
    try {
      const res = await getPickupOverdue()
      count.value = res.overdue_total ?? 0
      status.value = 'ok'
      // 数据恢复可信后，允许下一次失败重新出声
      errorDismissed.value = false
    } catch (e: unknown) {
      if ((e as { status?: number }).status === 403) {
        // 权限不足 ≠ 数据故障：静默降级为「无此信号」，只留日志
        count.value = 0
        status.value = 'forbidden'
        console.warn('[pickupAlert] 无接送查看权限，滞留信号关闭', e)
        return
      }
      // 明确留痕：状态降级为 error 由横幅呈现，绝不吞掉异常显示 0
      status.value = 'error'
      console.error('[pickupAlert] 滞留计数获取失败', e)
    }
  }

  function onVisible() {
    if (typeof document === 'undefined' || document.hidden) return
    fetchCount()
  }

  function start() {
    if (started) return
    // 无接送查看权限的账号不轮询壳层滞留状态（403 请求都不发）
    if (!usePermissionStore().can('pickup:view')) {
      status.value = 'forbidden'
      return
    }
    started = true
    fetchCount()
    timer = setInterval(() => {
      if (typeof document !== 'undefined' && document.hidden) return
      fetchCount()
    }, POLL_MS)
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', onVisible)
    }
  }

  function stop() {
    if (timer) clearInterval(timer)
    timer = null
    started = false
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', onVisible)
    }
  }

  /** 本次会话内关掉横幅；徽标继续显示真实计数，刷新后恢复。错误横幅单独关、恢复后可再响 */
  function dismissBanner() {
    if (status.value === 'error') {
      errorDismissed.value = true
    } else {
      bannerDismissed.value = true
    }
  }

  function retry() {
    return fetchCount()
  }

  return {
    count,
    status,
    hasOverdue,
    showUrgencyBanner,
    showErrorBanner,
    start,
    stop,
    dismissBanner,
    retry,
  }
})
