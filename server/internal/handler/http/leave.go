package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listLeavesQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	ClassID    uint64 `form:"class_id"`
	Status     *int8  `form:"status"`
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
}

func (h *Handler) ListLeaves(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listLeavesQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	var classID *uint64
	if q.ClassID > 0 {
		classID = &q.ClassID
	}
	list, total, page, pageSize, err := h.leaveSvc.ListLeaves(
		c.Request.Context(), kgID, pq, classID, q.Status, q.StartDate, q.EndDate,
	)
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
		case "BAD_DATE", "BAD_DATE_RANGE":
			response.ErrValidation(c, "日期参数无效")
		default:
			response.ErrInternal(c, "获取请假列表失败")
		}
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

type leaveStatsQuery struct {
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

func (h *Handler) GetLeaveStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q leaveStatsQuery
	_ = c.ShouldBindQuery(&q)
	data, err := h.leaveSvc.GetLeaveStats(c.Request.Context(), kgID, q.StartDate, q.EndDate)
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
		case "BAD_DATE", "BAD_DATE_RANGE":
			response.ErrValidation(c, "日期参数无效")
		default:
			response.ErrInternal(c, "获取请假统计失败")
		}
		return
	}
	response.OK(c, data)
}

func (h *Handler) AuditLeave(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的请假 ID")
		return
	}
	var req service.AuditLeaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	err = h.leaveSvc.AuditLeave(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
		default:
			if service.IsLeaveNotFound(err) {
				response.ErrNotFound(c, "请假记录不存在")
				return
			}
			if service.IsLeaveClientErr(err) {
				response.ErrBadRequest(c, 40040, "仅待审批的请假可审核，且 status 须为 1（同意）或 2（驳回）")
				return
			}
			response.ErrInternal(c, "审核失败")
		}
		return
	}
	response.OK(c, map[string]interface{}{})
}
