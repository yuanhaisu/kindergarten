import { requestWithAuth } from "@/api/http"

export function fetchEntitlements() {
  return requestWithAuth({
    url: "/common/entitlements",
    method: "GET"
  })
}
