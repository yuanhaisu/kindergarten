package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listNoticesQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Keyword    string `form:"keyword"`
	ClassID    uint64 `form:"class_id"`
	Category   string `form:"category"`
	Status     *int8  `form:"status"`
}

func (h *Handler) ListNotices(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listNoticesQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize, Keyword: q.Keyword}
	var classID *uint64
	if q.ClassID > 0 {
		classID = &q.ClassID
	}
	list, total, page, pageSize, err := h.noticeSvc.ListNotices(
		c.Request.Context(), kgID, pq, classID, q.Category, q.Status,
	)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取通知列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) GetNotice(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的通知 ID")
		return
	}
	data, err := h.noticeSvc.GetNoticeDetail(c.Request.Context(), kgID, id)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNoticeNotFound(err) {
			response.ErrNotFound(c, "通知不存在")
			return
		}
		response.ErrInternal(c, "获取通知详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateNotice(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.noticeSvc.CreateNotice(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "CLASS_NOT_FOUND":
			response.ErrNotFound(c, "班级不存在")
			return
		case "BAD_GRADE":
			response.ErrValidation(c, "年级取值无效")
			return
		case "EMPTY_NOTICE_TARGETS":
			// 旧入参只带单 class_id 也走这里：明确报参数错，不静默按单班处理
			response.ErrValidation(c, "接收班级不能为空，请传 class_ids 或 grades")
			return
		case "BAD_CATEGORY", "BAD_TITLE", "BAD_SCHEDULED_AT":
			response.ErrValidation(c, "请求参数无效")
			return
		default:
			response.ErrInternal(c, "发布通知失败")
			return
		}
	}
	response.Created(c, data)
}

func (h *Handler) UpdateNotice(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的通知 ID")
		return
	}
	var req service.UpdateNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.noticeSvc.UpdateNotice(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "CLASS_NOT_FOUND":
			response.ErrNotFound(c, "班级不存在")
			return
		case "BAD_GRADE":
			response.ErrValidation(c, "年级取值无效")
			return
		case "EMPTY_NOTICE_TARGETS":
			response.ErrValidation(c, "接收班级不能为空，请传 class_ids 或 grades")
			return
		case "BAD_CATEGORY", "BAD_TITLE", "BAD_SCHEDULED_AT", "BAD_STATUS", "EMPTY_UPDATE":
			response.ErrValidation(c, "请求参数无效")
			return
		default:
			if service.IsNoticeNotFound(err) {
				response.ErrNotFound(c, "通知不存在")
				return
			}
			response.ErrInternal(c, "更新通知失败")
			return
		}
	}
	response.OK(c, gin.H{"id": id})
}

func (h *Handler) DeleteNotice(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的通知 ID")
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.noticeSvc.DeleteNotice(c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNoticeNotFound(err) {
			response.ErrNotFound(c, "通知不存在")
			return
		}
		response.ErrInternal(c, "删除通知失败")
		return
	}
	response.OK(c, gin.H{})
}

func (h *Handler) AuditNotice(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的通知 ID")
		return
	}
	var req service.AuditNoticeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.noticeSvc.AuditNotice(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "BAD_AUDIT_ACTION" {
			response.ErrValidation(c, "action 须为 pass 或 reject")
			return
		}
		if service.IsNoticeNotFound(err) {
			response.ErrNotFound(c, "通知不存在")
			return
		}
		response.ErrInternal(c, "审核通知失败")
		return
	}
	response.OK(c, data)
}
