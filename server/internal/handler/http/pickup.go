package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type pickupOverviewQuery struct {
	Date string `form:"date"`
}

func (h *Handler) GetPickupOverview(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q pickupOverviewQuery
	_ = c.ShouldBindQuery(&q)
	data, err := h.pickupSvc.GetPickupOverview(c.Request.Context(), kgID, q.Date)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "BAD_DATE" {
			response.ErrValidation(c, "日期参数无效")
			return
		}
		response.ErrInternal(c, "获取接送概览失败")
		return
	}
	response.OK(c, data)
}

type pickupRecordsQuery struct {
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
	ClassID     uint64 `form:"class_id"`
	Date        string `form:"date"`
	IsAbnormal  *bool  `form:"is_abnormal"`
}

func (h *Handler) GetPickupRecords(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q pickupRecordsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	var classID *uint64
	if q.ClassID > 0 {
		classID = &q.ClassID
	}
	list, total, page, pageSize, err := h.pickupSvc.ListPickupRecords(
		c.Request.Context(), kgID, pq, classID, q.Date, q.IsAbnormal,
	)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "BAD_DATE" {
			response.ErrValidation(c, "日期参数无效")
			return
		}
		response.ErrInternal(c, "获取接送记录失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

type pickupAbnormalQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Date     string `form:"date"`
}

func (h *Handler) GetPickupAbnormal(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q pickupAbnormalQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.pickupSvc.ListPickupAbnormal(c.Request.Context(), kgID, pq, q.Date)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "BAD_DATE" {
			response.ErrValidation(c, "日期参数无效")
			return
		}
		response.ErrInternal(c, "获取异常接送失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) AuditPickupAuthorizedPerson(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的授权 ID")
		return
	}
	var req service.AuditPickupAuthReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.pickupSvc.AuditPickupAuthorization(
		c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"),
	)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsPickupAuthNotFound(err) {
			response.ErrNotFound(c, "授权记录不存在")
			return
		}
		if service.IsBadPickupAction(err) {
			response.ErrValidation(c, "action 须为 approve 或 reject")
			return
		}
		if service.IsPickupAuthNotPendingErr(err) {
			response.ErrBadRequest(c, 40040, "仅待审核的授权可申请")
			return
		}
		response.ErrInternal(c, "审核失败")
		return
	}
	response.OK(c, data)
}
