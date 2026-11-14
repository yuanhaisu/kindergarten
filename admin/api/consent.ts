export interface ConsentStatus {
  policy_version: string
  consented: boolean
  consented_at?: string | null
}

export function getConsentStatus() {
  return useHttp().get<ConsentStatus>('/common/consent')
}

export function recordConsent(policyVersion: string) {
  return useHttp().post<ConsentStatus>('/common/consent', { policy_version: policyVersion })
}
