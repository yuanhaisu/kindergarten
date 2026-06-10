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
	items, err := h.healthCheckSvc.GetQueue(c.Request.Context(), kgID, classID, date)
	if err != nil {
		response.ErrInternal(c, "获取队列失败")
		return
	}
	response.OK(c, gin.H{"list": items})
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
		response.ErrInternal(c, "提交失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DoctorAddMedia(c *gin.Context) {
	checkID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || checkID == 0 {
		response.ErrValidation(c, "无效的 ID")
		return
	}
	var req struct {
		FileURL  string `json:"file_url" binding:"required"`
		ThumbURL string `json:"thumb_url"`
		Category string `json:"category"`
		FileName string `json:"file_name"`
		FileSize int64  `json:"file_size"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数不合法")
		return
	}
	media, err := h.healthCheckSvc.AddMedia(c.Request.Context(), checkID, req.FileURL, req.ThumbURL, req.Category, req.FileName, req.FileSize)
	if err != nil {
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
