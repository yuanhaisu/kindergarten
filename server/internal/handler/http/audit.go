package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listAuditQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Type     string `form:"type"`
	Status   string `form:"status"`
}

func (h *Handler) GetAuditList(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listAuditQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.auditSvc.ListAudits(
		c.Request.Context(), kgID, pq, q.Type, q.Status,
	)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取审核列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) AuditContent(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的审核 ID")
		return
	}
	var req service.AuditContentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数错误")
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.auditSvc.AuditContent(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsAuditNotFound(err) {
			response.ErrNotFound(c, "审核记录不存在")
			return
		}
		if err.Error() == "BAD_AUDIT_ACTION" {
			response.ErrValidation(c, "无效的审核动作")
			return
		}
		response.ErrInternal(c, "审核失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetAuditStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.auditSvc.GetAuditStats(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取审核统计失败")
		return
	}
	response.OK(c, data)
}
