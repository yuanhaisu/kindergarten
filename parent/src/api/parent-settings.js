import { requestWithAuth } from "@/api/http"

export function fetchSettings() {
  return requestWithAuth({
    url: "/parent/settings",
    method: "GET"
  })
}

export function updateSettings(data) {
  return requestWithAuth({
    url: "/parent/settings",
    method: "PUT",
    data
  })
}
