package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type attendanceOverviewQuery struct {
	Date string `form:"date"`
}

func (h *Handler) GetAttendanceOverview(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q attendanceOverviewQuery
	_ = c.ShouldBindQuery(&q)
	data, err := h.attendanceSvc.Overview(c.Request.Context(), kgID, q.Date)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "BAD_DATE" {
			response.ErrValidation(c, "日期格式无效")
			return
		}
		response.ErrInternal(c, "获取出勤概览失败")
		return
	}
	response.OK(c, data)
}

type attendanceRecordsQuery struct {
	ClassID   uint64 `form:"class_id"`
	Date      string `form:"date"`
	Status    string `form:"status"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

func (h *Handler) GetAttendanceRecords(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q attendanceRecordsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	var classID *uint64
	if q.ClassID > 0 {
		if err := h.attendanceSvc.ValidateClassInKindergarten(c.Request.Context(), kgID, q.ClassID); err != nil {
			if err.Error() == "CLASS_NOT_FOUND" {
				response.ErrNotFound(c, "班级不存在")
				return
			}
			response.ErrInternal(c, "校验班级失败")
			return
		}
		classID = &q.ClassID
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize, Keyword: ""}
	list, total, page, pageSize, err := h.attendanceSvc.Records(c.Request.Context(), kgID, classID, q.Date, q.Status, pq)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "BAD_DATE" {
			response.ErrValidation(c, "日期格式无效")
			return
		}
		if err.Error() == "BAD_STATUS" {
			response.ErrValidation(c, "status 须为 present / leave / absent")
			return
		}
		response.ErrInternal(c, "获取出勤明细失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

type attendanceReportQuery struct {
	ClassID   uint64 `form:"class_id"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
	Format    string `form:"format"`
}

func (h *Handler) ExportAttendanceReport(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q attendanceReportQuery
	_ = c.ShouldBindQuery(&q)
	if strings.TrimSpace(q.StartDate) == "" || strings.TrimSpace(q.EndDate) == "" {
		response.ErrValidation(c, "请填写 start_date 与 end_date（YYYY-MM-DD）")
		return
	}
	var classID *uint64
	if q.ClassID > 0 {
		if err := h.attendanceSvc.ValidateClassInKindergarten(c.Request.Context(), kgID, q.ClassID); err != nil {
			if err.Error() == "CLASS_NOT_FOUND" {
				response.ErrNotFound(c, "班级不存在")
				return
			}
			response.ErrInternal(c, "校验班级失败")
			return
		}
		classID = &q.ClassID
	}
	data, fn, err := h.attendanceSvc.ExportReport(c.Request.Context(), kgID, classID, q.StartDate, q.EndDate, q.Format)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if mapAttendanceExportErr(c, err) {
			return
		}
		response.ErrInternal(c, "导出失败")
		return
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fn))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func mapAttendanceExportErr(c *gin.Context, err error) bool {
	if !service.IsAttendanceClientErr(err) {
		return false
	}
	switch err.Error() {
	case "BAD_DATE":
		response.ErrValidation(c, "start_date / end_date 格式须为 YYYY-MM-DD")
	case "BAD_DATE_RANGE":
		response.ErrValidation(c, "结束日期不能早于开始日期")
	case "BAD_FORMAT":
		response.ErrBadRequest(c, 40060, "仅支持 format=xlsx")
	default:
		response.ErrValidation(c, err.Error())
	}
	return true
}
