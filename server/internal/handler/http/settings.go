package handler

import (
	"errors"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

func (h *Handler) GetSettings(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.settingsSvc.GetSettings(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取系统设置失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.UpdateSettingsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.settingsSvc.UpdateSettings(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "设置不存在")
			return
		}
		response.ErrInternal(c, "更新系统设置失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) ListAcademicYears(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.settingsSvc.ListAcademicYears(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取学年列表失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateAcademicYear(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateAcademicYearReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.settingsSvc.CreateAcademicYear(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if errors.Is(err, service.ErrDuplicateAcademicYear) {
			response.ErrBadRequest(c, 40041, "学年已存在")
			return
		}
		if service.IsSettingsClientErr(err) {
			response.ErrValidation(c, err.Error())
			return
		}
		response.ErrInternal(c, "创建学年失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) GetKindergartenInfo(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.settingsSvc.GetKindergartenInfo(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "园所不存在")
			return
		}
		response.ErrInternal(c, "获取园所信息失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) UpdateKindergartenInfo(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.UpdateKindergartenInfoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.settingsSvc.UpdateKindergartenInfo(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "园所不存在")
			return
		}
		if err.Error() == "INVALID_DATE" || err.Error() == "EMPTY_DATE" {
			response.ErrValidation(c, err.Error())
			return
		}
		response.ErrInternal(c, "更新园所信息失败")
		return
	}
	response.OK(c, data)
}
