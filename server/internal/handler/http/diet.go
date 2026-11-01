package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// GetDietConfig 返回园所供餐配置。
//
// AQR-S1b 只落路由与权限骨架，service 尚未实现，统一返回 501；
// 真实读取/默认值属于 S2（tasks 2.1/2.2）。
func (h *Handler) GetDietConfig(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.dietConfigSvc.GetConfig(c.Request.Context(), kgID)
	if err != nil {
		respondDietConfigError(c, err)
		return
	}
	response.OK(c, data)
}

// PutDietConfig 保存园所供餐配置（整组替换餐次）。
func (h *Handler) PutDietConfig(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	if err := h.dietConfigSvc.SaveConfig(c.Request.Context(), kgID); err != nil {
		respondDietConfigError(c, err)
		return
	}
	response.OK(c, nil)
}

// respondDietConfigError 把 service 错误映射到项目统一错误包络。
// TODO(AQR-S2): 501 是 S1b 的占位分支；S2 实现真实读写后删除。
func respondDietConfigError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrDietNotImplemented) {
		response.Err(c, http.StatusNotImplemented, 50100, "膳食模块尚未实现")
		return
	}
	response.ErrInternal(c, "膳食配置服务异常")
}
