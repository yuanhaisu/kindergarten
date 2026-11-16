import { describe, expect, it } from 'vitest'
import { expiringSoon } from '../../utils/entitlement'

/**
 * 「到期前 30 天转付费提示」只处理仍在期、且已进入提示窗口的授权。
 * 已过期（含宽限期）由页面上既有的到期/降级提示承担，这里必须返回 null，
 * 否则同一条到期事实会在页面上叠加两条提示。
 *
 * 时间边界按毫秒差判定：diff <= 0 即已过期；daysLeft = ceil(diff / 24h)，
 * 因此「还剩不足一天」计为 1 天，而不是 0。
 */

const NOW = new Date('2027-01-01T23:59:59+08:00')

describe('expiringSoon', () => {
  it('无到期时间返回 null', () => {
    expect(expiringSoon(null, NOW)).toBeNull()
    expect(expiringSoon(undefined, NOW)).toBeNull()
    expect(expiringSoon('', NOW)).toBeNull()
  })

  it('无法解析的日期返回 null', () => {
    expect(expiringSoon('not-a-date', NOW)).toBeNull()
  })

  it('恰好 30 天到期时返回提示', () => {
    expect(expiringSoon('2027-01-31T23:59:59+08:00', NOW)).toEqual({
      daysLeft: 30,
      expiresOn: '2027-01-31',
    })
  })

  it('超过 30 天不提示', () => {
    expect(expiringSoon('2027-02-01T00:00:00+08:00', NOW)).toBeNull()
  })

  it('还剩不足一天计为 1 天', () => {
    expect(
      expiringSoon('2027-01-31T23:59:59+08:00', new Date('2027-01-31T23:00:00+08:00')),
    ).toEqual({ daysLeft: 1, expiresOn: '2027-01-31' })
  })

  it('已过期一分钟也返回 null（避免 -0 天被当成在期）', () => {
    expect(
      expiringSoon('2027-01-31T23:59:59+08:00', new Date('2027-02-01T00:00:59+08:00')),
    ).toBeNull()
  })

  it('已过期一天返回 null', () => {
    expect(
      expiringSoon('2027-01-31T23:59:59+08:00', new Date('2027-02-01T23:59:59+08:00')),
    ).toBeNull()
  })

  it('提示窗口可配置', () => {
    expect(expiringSoon('2027-01-08T00:00:00+08:00', NOW, 7)).toEqual({
      daysLeft: 7,
      expiresOn: '2027-01-08',
    })
    expect(expiringSoon('2027-01-09T00:00:00+08:00', NOW, 7)).toBeNull()
  })
})
