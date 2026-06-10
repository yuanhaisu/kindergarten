package handler

import (
	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/pkg/response"
)

func (h *Handler) GetDashboardOverview(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	data, err := h.dashSvc.Overview(c.Request.Context(), kgID)
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
		case "KINDERGARTEN_NOT_FOUND":
			response.ErrNotFound(c, "园所不存在")
		default:
			response.ErrInternal(c, "加载看板概览失败")
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetDashboardAttendanceTrend(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	period := c.DefaultQuery("period", "30d")
	data, err := h.dashSvc.AttendanceTrend(c.Request.Context(), kgID, period)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "加载出勤趋势失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetDashboardClassRanking(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	data, err := h.dashSvc.ClassRanking(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "加载班级排行失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetDashboardGrowthStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	period := c.DefaultQuery("period", "30d")
	data, err := h.dashSvc.GrowthStats(c.Request.Context(), kgID, period)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "加载成长统计失败")
		return
	}
	response.OK(c, data)
}
