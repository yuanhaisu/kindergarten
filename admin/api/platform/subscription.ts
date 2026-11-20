import type { PlatformEntitlementRow } from './kindergartens'

export interface SubscriptionView {
  plan_name: string
  modules: PlatformEntitlementRow[]
  expires_at: string | null
  grace_until: string | null
  state: 'active' | 'grace' | 'expired'
  student_count: number
  tier_max_students: number
  over_limit: boolean
}

export function getSubscription() {
  return useHttp().get<SubscriptionView>('/admin/billing/subscription')
}
