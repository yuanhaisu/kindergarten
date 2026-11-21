package handler

import (
	"github.com/gin-gonic/gin"

	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// GetPublicKindergartenBrand 是未登录可访问的园所公开品牌接口（#170）。
//
// 只按 public_code 定位（不暴露连续 id），只下发名称/logo/主题色/公开码；
// 软删或不存在的园所统一 404，避免通过响应差异探测园所是否存在。
func (h *Handler) GetPublicKindergartenBrand(c *gin.Context) {
	code := c.Param("public_code")
	brand, err := h.publicBrandSvc.GetByPublicCode(c.Request.Context(), code)
	if err != nil {
		log.L(c.Request.Context()).Error("load public kindergarten brand failed",
			"public_code", code, "err", err)
		response.ErrInternal(c, "获取园所品牌失败")
		return
	}
	if brand == nil {
		response.ErrNotFound(c, "园所不存在")
		return
	}
	// 签名 logo 有有效期，缓存时间必须显著短于 TTL。
	c.Header("Cache-Control", "public, max-age=60")
	response.OK(c, brand)
}
