package handler

// timetable.go —— 课表维护的 HTTP 入口（管理端）：
// 年级模板 / 班级微调 / 园所开关 / 立即同步。
// 路径与权限码见 backend/api/api-admin/21-calendar.md 与权限登记表（timetable:view / timetable:manage）。

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

type saveTimetableTemplateReq struct {
	Entries []service.TimetableEntryInput `json:"entries"`
}

type saveTimetableOverridesReq struct {
	Overrides []service.TimetableOverrideInput `json:"overrides"`
}

func (h *Handler) AdminGetTimetableTemplate(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	data, err := h.timetableSvc.GetTemplate(c.Request.Context(), kgID, c.Param("grade"))
	if err != nil {
		timetableErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) AdminSaveTimetableTemplate(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	var req saveTimetableTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.timetableSvc.SaveTemplate(c.Request.Context(), kgID, c.Param("grade"), req.Entries)
	if err != nil {
		timetableErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) AdminGetTimetableClassOverrides(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	classID, ok := timetableClassID(c)
	if !ok {
		return
	}
	data, err := h.timetableSvc.GetClassOverrides(c.Request.Context(), kgID, classID)
	if err != nil {
		timetableErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) AdminSaveTimetableClassOverrides(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	classID, ok := timetableClassID(c)
	if !ok {
		return
	}
	var req saveTimetableOverridesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.timetableSvc.SaveClassOverrides(c.Request.Context(), kgID, classID, req.Overrides)
	if err != nil {
		timetableErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) AdminGetTimetableSetting(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	data, err := h.timetableSvc.GetSetting(c.Request.Context(), kgID)
	if err != nil {
		timetableErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) AdminUpdateTimetableSetting(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	var req struct {
		TimetableEnabled *bool `json:"timetable_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TimetableEnabled == nil {
		response.ErrValidation(c, "参数不合法：timetable_enabled 必填")
		return
	}
	data, err := h.timetableSvc.UpdateSetting(c.Request.Context(), kgID, *req.TimetableEnabled)
	if err != nil {
		timetableErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) AdminSyncTimetable(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	data, err := h.timetableSvc.SyncKindergarten(c.Request.Context(), kgID)
	if err != nil {
		timetableErr(c, err)
		return
	}
	response.OK(c, data)
}

func timetableClassID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("class_id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的班级 ID")
		return 0, false
	}
	return id, true
}

func timetableErr(c *gin.Context, err error) {
	if errors.Is(err, service.ErrTimetableInvalidParams) {
		response.ErrValidation(c, "课表参数不合法")
		return
	}
	response.ErrInternal(c, "课表操作失败")
}
