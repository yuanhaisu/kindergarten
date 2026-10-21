package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// ==================== Doctor Handlers ====================

func (h *Handler) DoctorGetQueue(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var classID *uint64
	if cid, err := strconv.ParseUint(c.Query("class_id"), 10, 64); err == nil && cid > 0 {
		classID = &cid
	}
	date := c.DefaultQuery("date", "")
	data, err := h.healthCheckSvc.GetQueueWithPolicy(c.Request.Context(), kgID, classID, date)
	if err != nil {
		response.ErrInternal(c, "获取队列失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DoctorGetCheckDetail(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	checkID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || checkID == 0 {
		response.ErrValidation(c, "无效的 ID")
		return
	}
	data, err := h.healthCheckSvc.GetDetail(c.Request.Context(), kgID, checkID)
	if err != nil {
		if errors.Is(err, service.ErrHealthCheckNotFound) {
			response.ErrNotFound(c, "晨检记录不存在")
			return
		}
		response.ErrInternal(c, "获取详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DoctorSubmitCheck(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	checkID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || checkID == 0 {
		response.ErrValidation(c, "无效的 ID")
		return
	}
	var req service.SubmitHealthCheckReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数不合法")
		return
	}
	userID := middleware.GetUserID(c)
	data, err := h.healthCheckSvc.SubmitCheck(c.Request.Context(), kgID, userID, checkID, req)
	if err != nil {
		if errors.Is(err, service.ErrHealthCheckNotFound) {
			response.ErrNotFound(c, "晨检记录不存在")
			return
		}
		if errors.Is(err, service.ErrHealthCheckAlreadyDone) {
			response.ErrValidation(c, "该学生已完成晨检")
			return
		}
		if errors.Is(err, service.ErrHealthCheckInvalidItem) {
			response.ErrValidation(c, err.Error())
			return
		}
		if errors.Is(err, service.ErrHealthCheckTemperatureRequired) {
			response.ErrValidation(c, "本园要求全员测温，请先录入体温")
			return
		}
		response.ErrInternal(c, "提交失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DoctorAddMedia(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	checkID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || checkID == 0 {
		response.ErrValidation(c, "无效的 ID")
		return
	}
	var req struct {
		FileID   uint64 `json:"file_id" binding:"required"`
		Category string `json:"category" binding:"required"`
		RefDesc  string `json:"ref_desc"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数不合法")
		return
	}
	media, err := h.healthCheckSvc.AddMedia(
		c.Request.Context(), kgID, checkID, req.FileID, middleware.GetUserID(c), req.Category, req.RefDesc)
	if err != nil {
		if errors.Is(err, service.ErrHealthCheckNotFound) {
			response.ErrNotFound(c, "晨检记录不存在")
			return
		}
		if errors.Is(err, service.ErrHealthCheckMediaInvalid) {
			response.ErrValidation(c, err.Error())
			return
		}
		response.ErrInternal(c, "添加照片失败")
		return
	}
	response.OK(c, media)
}

func (h *Handler) DoctorTodayStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	date := c.DefaultQuery("date", "")
	data, err := h.healthCheckSvc.GetStats(c.Request.Context(), kgID, date)
	if err != nil {
		response.ErrInternal(c, "获取统计失败")
		return
	}
	response.OK(c, data)
}

// ==================== Admin Health Handler ====================

func (h *Handler) AdminHealthStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	date := c.DefaultQuery("date", "")
	data, err := h.healthCheckSvc.GetStats(c.Request.Context(), kgID, date)
	if err != nil {
		response.ErrInternal(c, "获取统计失败")
		return
	}
	response.OK(c, data)
}
