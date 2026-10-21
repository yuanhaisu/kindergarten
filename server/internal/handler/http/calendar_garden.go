package handler

// calendar_garden.go —— 园级日历事件的 HTTP 入口（align-class-calendar）：
// 管理端（园长角色，权限码 calendar:view / calendar:manage）与医生端（/doctor/calendar/*）。
// 业务逻辑共用 service.GardenCalendarService。

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// ==================== Admin 园级日历 ====================

func (h *Handler) AdminListCalendarEvents(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	data, err := h.gardenCalendarSvc.ListGardenEvents(c.Request.Context(), kgID,
		c.Query("start"), c.Query("end"), c.Query("status"))
	if err != nil {
		gardenCalendarErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) AdminCreateCalendarEvent(c *gin.Context) {
	h.gardenCalendarCreate(c)
}

func (h *Handler) AdminUpdateCalendarEvent(c *gin.Context) {
	h.gardenCalendarUpdate(c)
}

func (h *Handler) AdminDeleteCalendarEvent(c *gin.Context) {
	h.gardenCalendarDelete(c)
}

// ==================== Doctor 园级日历（保健医发布，老师端小程序内） ====================

func (h *Handler) DoctorListCalendarEvents(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	data, err := h.gardenCalendarSvc.ListGardenEvents(c.Request.Context(), kgID,
		c.Query("start"), c.Query("end"), c.Query("status"))
	if err != nil {
		gardenCalendarErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) DoctorCreateCalendarEvent(c *gin.Context) {
	h.gardenCalendarCreate(c)
}

func (h *Handler) DoctorUpdateCalendarEvent(c *gin.Context) {
	h.gardenCalendarUpdate(c)
}

func (h *Handler) DoctorDeleteCalendarEvent(c *gin.Context) {
	h.gardenCalendarDelete(c)
}

// ── 共用实现 ──────────────────────────────────────────────────────────

func (h *Handler) gardenCalendarCreate(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	userID := middleware.GetUserID(c)
	var req service.GardenCalendarInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.gardenCalendarSvc.CreateGardenEvent(c.Request.Context(), kgID, userID, req)
	if err != nil {
		gardenCalendarErr(c, err)
		return
	}
	response.Created(c, data)
}

func (h *Handler) gardenCalendarUpdate(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	userID := middleware.GetUserID(c)
	id, ok := gardenCalendarID(c)
	if !ok {
		return
	}
	var req service.GardenCalendarInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.gardenCalendarSvc.UpdateGardenEvent(c.Request.Context(), kgID, userID, id, req)
	if err != nil {
		gardenCalendarErr(c, err)
		return
	}
	response.OK(c, data)
}

func (h *Handler) gardenCalendarDelete(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	id, ok := gardenCalendarID(c)
	if !ok {
		return
	}
	if err := h.gardenCalendarSvc.DeleteGardenEvent(c.Request.Context(), kgID, id); err != nil {
		gardenCalendarErr(c, err)
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func gardenCalendarID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的事件 ID")
		return 0, false
	}
	return id, true
}

func gardenCalendarErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrGardenNotFound):
		response.ErrNotFound(c, "园级事件不存在")
	case errors.Is(err, service.ErrGardenNoClasses):
		response.ErrValidation(c, "本园暂无在读班级，无法通知家长")
	case errors.Is(err, service.ErrGardenInvalidParams):
		response.ErrValidation(c, "参数不合法")
	default:
		response.ErrInternal(c, "园级日历操作失败")
	}
}
