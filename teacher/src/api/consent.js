import { requestWithAuth } from "@/api/http"

export function getConsentStatus() {
  return requestWithAuth({
    url: "/common/consent",
    method: "GET",
    silent: true
  })
}

export function recordConsent(policyVersion) {
  return requestWithAuth({
    url: "/common/consent",
    method: "POST",
    data: { policy_version: policyVersion },
    silent: true
  })
}
