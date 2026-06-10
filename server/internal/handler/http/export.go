package handler

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

func (h *Handler) PostExportAttendance(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		response.ErrValidation(c, "参数错误")
		return
	}
	data, err := h.exportSvc.CreateAttendanceExport(c.Request.Context(), kgID, raw)
	if err != nil {
		mapExportErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) PostExportStudents(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		response.ErrValidation(c, "参数错误")
		return
	}
	data, err := h.exportSvc.CreateStudentsExport(c.Request.Context(), kgID, raw)
	if err != nil {
		mapExportErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) PostExportGrowth(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		response.ErrValidation(c, "参数错误")
		return
	}
	data, err := h.exportSvc.CreateGrowthExport(c.Request.Context(), kgID, raw)
	if err != nil {
		mapExportErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) GetExportTask(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	tid := strings.TrimSpace(c.Param("taskId"))
	if tid == "" {
		response.ErrValidation(c, "无效的任务 ID")
		return
	}
	data, err := h.exportSvc.GetExportTaskStatus(c.Request.Context(), kgID, tid)
	if err != nil {
		mapExportErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) DownloadExportFile(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	tid := strings.TrimSpace(c.Param("taskId"))
	if tid == "" {
		response.ErrValidation(c, "无效的任务 ID")
		return
	}
	path, fn, err := h.exportSvc.GetExportFileForDownload(c.Request.Context(), kgID, tid)
	if err != nil {
		mapExportErr(c, err)
		return
	}
	if fn == "" {
		fn = filepath.Base(path)
	}
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fn))
	c.File(path)
}

func mapExportErr(c *gin.Context, err error) {
	if err == nil {
		return
	}
	switch err.Error() {
	case "NO_KINDERGARTEN":
		response.ErrForbidden(c, "未绑定园所，请重新登录")
	case "NOT_FOUND":
		response.ErrNotFound(c, "导出任务不存在")
	case "NOT_READY":
		response.ErrBadRequest(c, 40061, "文件尚未就绪")
	case "EXPIRED":
		response.ErrBadRequest(c, 40062, "导出文件已过期，请重新导出")
	case "BAD_JSON":
		response.ErrValidation(c, "参数格式错误")
	case "BAD_DATE":
		response.ErrValidation(c, "请填写正确的日期（YYYY-MM-DD）")
	case "CLASS_NOT_FOUND":
		response.ErrNotFound(c, "班级不存在")
	default:
		if service.IsAttendanceClientErr(err) {
			if mapAttendanceExportErr(c, err) {
				return
			}
		}
		if err.Error() == "BAD_FORMAT" || err.Error() == "BAD_DATE_RANGE" {
			if err.Error() == "BAD_FORMAT" {
				response.ErrBadRequest(c, 40060, "仅支持 format=xlsx")
				return
			}
			response.ErrValidation(c, "结束日期不能早于开始日期")
			return
		}
		response.ErrInternal(c, "导出失败")
	}
}
