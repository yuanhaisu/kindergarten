package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

func (h *Handler) GetPointsOverview(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.pointSvc.GetOverview(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取积分概览失败")
		return
	}
	response.OK(c, data)
}

type listPointRecordsQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	StudentID  uint64 `form:"student_id"`
	ClassID    uint64 `form:"class_id"`
	Type       string `form:"type"`
}

func (h *Handler) ListPointRecords(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listPointRecordsQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	var sid *uint64
	if q.StudentID > 0 {
		sid = &q.StudentID
	}
	var cid *uint64
	if q.ClassID > 0 {
		cid = &q.ClassID
	}
	list, total, page, pageSize, err := h.pointSvc.ListRecords(c.Request.Context(), kgID, pq, sid, cid, q.Type)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取积分流水失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) AdjustPoints(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.AdjustPointsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.pointSvc.AdjustPoints(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch {
		case err.Error() == "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case err.Error() == "STUDENT_NOT_FOUND":
			response.ErrNotFound(c, "学生不存在")
			return
		case err.Error() == "BAD_ADJUST_TYPE":
			response.ErrValidation(c, "type 须为 earn 或 spend")
			return
		case err.Error() == "BAD_REASON":
			response.ErrValidation(c, "请填写原因")
			return
		case errors.Is(err, repository.ErrInsufficientPointBalance):
			response.ErrBadRequest(c, 40041, "可用积分不足")
			return
		case err.Error() == "BAD_POINT_AMOUNT":
			response.ErrValidation(c, "数量须为正整数")
			return
		default:
			response.ErrInternal(c, "调整积分失败")
			return
		}
	}
	response.OK(c, data)
}

func (h *Handler) GetPointRules(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.pointSvc.GetPointRules(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取积分规则失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) UpdatePointRules(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.UpdatePointRulesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.pointSvc.UpdatePointRules(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "BAD_RULE_ID":
			response.ErrValidation(c, "规则 id 无效")
			return
		case "BAD_POINTS":
			response.ErrValidation(c, "积分不能为负数")
			return
		default:
			response.ErrInternal(c, "更新积分规则失败")
			return
		}
	}
	response.OK(c, data)
}
