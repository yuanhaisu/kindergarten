package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listGrowthLogsQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	ClassID    uint64 `form:"class_id"`
	TeacherID  uint64 `form:"teacher_id"`
	Date       string `form:"date"`
}

func (h *Handler) ListGrowthLogs(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listGrowthLogsQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	var classID *uint64
	if q.ClassID > 0 {
		classID = &q.ClassID
	}
	var teacherID *uint64
	if q.TeacherID > 0 {
		teacherID = &q.TeacherID
	}
	list, total, page, pageSize, err := h.growthSvc.ListGrowthLogs(
		c.Request.Context(), kgID, pq, classID, teacherID, q.Date,
	)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取成长记录列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) GetGrowthLog(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的日志 ID")
		return
	}
	data, err := h.growthSvc.GetGrowthLogDetail(c.Request.Context(), kgID, id)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsGrowthNotFound(err) {
			response.ErrNotFound(c, "成长记录不存在")
			return
		}
		response.ErrInternal(c, "获取成长记录详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteGrowthLog(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的日志 ID")
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.growthSvc.DeleteGrowthLog(c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsGrowthNotFound(err) {
			response.ErrNotFound(c, "成长记录不存在")
			return
		}
		response.ErrInternal(c, "删除成长记录失败")
		return
	}
	response.OK(c, gin.H{})
}

func (h *Handler) AuditGrowthLog(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的日志 ID")
		return
	}
	var req service.AuditGrowthLogReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.growthSvc.AuditGrowthLog(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "BAD_AUDIT_ACTION" {
			response.ErrValidation(c, "action 须为 pass 或 reject")
			return
		}
		if service.IsGrowthNotFound(err) {
			response.ErrNotFound(c, "成长记录不存在")
			return
		}
		response.ErrInternal(c, "审核失败")
		return
	}
	response.OK(c, data)
}

type growthStatsQuery struct {
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

func (h *Handler) GetGrowthStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q growthStatsQuery
	_ = c.ShouldBindQuery(&q)
	data, err := h.growthSvc.GetGrowthStats(c.Request.Context(), kgID, q.StartDate, q.EndDate)
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "BAD_DATE", "BAD_DATE_RANGE":
			response.ErrValidation(c, "日期参数无效")
			return
		default:
			response.ErrInternal(c, "获取成长统计失败")
			return
		}
	}
	response.OK(c, data)
}
