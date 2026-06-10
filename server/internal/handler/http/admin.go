package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listAdminsQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	Keyword  string `form:"keyword"`
}

func (h *Handler) ListAdmins(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listAdminsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize, Keyword: q.Keyword}
	list, total, page, pageSize, err := h.adminSvc.ListAdmins(c.Request.Context(), kgID, pq)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取管理员列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) CreateAdmin(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateAdminReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.adminSvc.CreateAdmin(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsAdminClientErr(err) {
			if err.Error() == "WEAK_PASSWORD" || err.Error() == "INVALID_PHONE" {
				response.ErrValidation(c, err.Error())
				return
			}
			if err.Error() == "DUPLICATE_ADMIN_PHONE" {
				response.ErrBadRequest(c, 40042, "该手机号已是本园管理员")
				return
			}
			if err.Error() == "ROLE_NOT_FOUND" || err.Error() == "KINDERGARTEN_NOT_FOUND" {
				response.ErrBadRequest(c, 40043, err.Error())
				return
			}
		}
		response.ErrInternal(c, "添加管理员失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) UpdateAdmin(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的管理员 ID")
		return
	}
	var req service.UpdateAdminReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.adminSvc.UpdateAdmin(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "管理员不存在")
			return
		}
		if service.IsAdminClientErr(err) {
			response.ErrBadRequest(c, 40043, err.Error())
			return
		}
		response.ErrInternal(c, "更新管理员失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteAdmin(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的管理员 ID")
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.adminSvc.DeleteAdmin(c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "管理员不存在")
			return
		}
		if err.Error() == "CANNOT_DELETE_SELF" {
			response.ErrBadRequest(c, 40044, "不能删除当前登录账号")
			return
		}
		if err.Error() == "LAST_ADMIN" {
			response.ErrBadRequest(c, 40045, "至少保留一名管理员")
			return
		}
		response.ErrInternal(c, "删除管理员失败")
		return
	}
	response.OK(c, gin.H{})
}

func (h *Handler) ResetAdminPassword(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的管理员 ID")
		return
	}
	var req service.ResetAdminPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.adminSvc.ResetAdminPassword(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "管理员不存在")
			return
		}
		if err.Error() == "WEAK_PASSWORD" {
			response.ErrValidation(c, err.Error())
			return
		}
		response.ErrInternal(c, "重置密码失败")
		return
	}
	response.OK(c, data)
}
