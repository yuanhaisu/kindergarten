package handler

import (
	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listOperationLogsQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	Module     string `form:"module"`
	OperatorID uint64 `form:"operator_id"`
	StartDate  string `form:"start_date"`
	EndDate    string `form:"end_date"`
}

func (h *Handler) ListOperationLogs(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listOperationLogsQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.oplogSvc.ListOperationLogs(
		c.Request.Context(), kgID, pq, q.Module, q.OperatorID, q.StartDate, q.EndDate,
	)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "BAD_DATE_RANGE" {
			response.ErrValidation(c, "日期格式应为 YYYY-MM-DD，且结束日期不能早于开始日期")
			return
		}
		response.ErrInternal(c, "获取操作日志失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}
