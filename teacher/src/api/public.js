import { rawRequest } from "@/api/http"

// 园所公开品牌（#170）：未登录可访问，按不可枚举的 public_code 定位单园。
// 老师端只用它给「当前园所」取名称/logo/主色；不存在任何全量园所列表接口。
export function fetchPublicKindergarten(publicCode) {
  return rawRequest({
    url: `/public/kindergartens/${encodeURIComponent(String(publicCode || "").trim())}`,
    method: "GET",
    // 入口码可能已失效（404）：由首页决定丢弃入口或退回通用引导，不弹全局 toast。
    silent: true
  })
}
