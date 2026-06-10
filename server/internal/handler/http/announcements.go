package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listAnnouncementsQuery struct {
	Page     int   `form:"page"`
	PageSize int   `form:"page_size"`
	Status   *int8 `form:"status"`
}

func (h *Handler) ListAnnouncements(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listAnnouncementsQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.announcementSvc.ListAnnouncements(
		c.Request.Context(), kgID, pq, q.Status,
	)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取公告列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) GetAnnouncement(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的公告 ID")
		return
	}
	data, err := h.announcementSvc.GetAnnouncementDetail(c.Request.Context(), kgID, id)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsAnnouncementNotFound(err) {
			response.ErrNotFound(c, "公告不存在")
			return
		}
		response.ErrInternal(c, "获取公告详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateAnnouncement(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.announcementSvc.CreateAnnouncement(
		c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"),
	)
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "BAD_TITLE", "BAD_SCHEDULED_AT", "BAD_TARGET_ROLES":
			response.ErrValidation(c, "请求参数无效")
			return
		default:
			response.ErrInternal(c, "发布公告失败")
			return
		}
	}
	response.Created(c, data)
}

func (h *Handler) UpdateAnnouncement(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的公告 ID")
		return
	}
	var req service.UpdateAnnouncementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.announcementSvc.UpdateAnnouncement(
		c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"),
	)
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "BAD_TITLE", "BAD_SCHEDULED_AT", "BAD_TARGET_ROLES", "EMPTY_UPDATE":
			response.ErrValidation(c, "请求参数无效")
			return
		default:
			if service.IsAnnouncementNotFound(err) {
				response.ErrNotFound(c, "公告不存在")
				return
			}
			response.ErrInternal(c, "更新公告失败")
			return
		}
	}
	response.OK(c, data)
}

func (h *Handler) DeleteAnnouncement(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的公告 ID")
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.announcementSvc.DeleteAnnouncement(
		c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent"),
	); err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsAnnouncementNotFound(err) {
			response.ErrNotFound(c, "公告不存在")
			return
		}
		response.ErrInternal(c, "删除公告失败")
		return
	}
	response.OK(c, gin.H{})
}

func (h *Handler) SetAnnouncementTop(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的公告 ID")
		return
	}
	var req service.SetAnnouncementTopReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.announcementSvc.SetAnnouncementTop(
		c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"),
	)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsAnnouncementNotFound(err) {
			response.ErrNotFound(c, "公告不存在")
			return
		}
		response.ErrInternal(c, "置顶操作失败")
		return
	}
	response.OK(c, data)
}
