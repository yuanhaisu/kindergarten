package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

func (h *Handler) ListRoles(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	data, err := h.roleSvc.ListRoles(c.Request.Context(), kgID)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取角色列表失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) CreateRole(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.roleSvc.CreateRole(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if mapRoleErr(c, err) {
			return
		}
		response.ErrInternal(c, "创建角色失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的角色 ID")
		return
	}
	var req service.UpdateRoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.roleSvc.UpdateRole(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "NOT_FOUND" {
			response.ErrNotFound(c, "角色不存在")
			return
		}
		if mapRoleErr(c, err) {
			return
		}
		response.ErrInternal(c, "更新角色失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteRole(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的角色 ID")
		return
	}
	uid := middleware.GetUserID(c)
	err = h.roleSvc.DeleteRole(c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if err.Error() == "NOT_FOUND" {
			response.ErrNotFound(c, "角色不存在")
			return
		}
		switch err.Error() {
		case "SYSTEM_ROLE_PROTECTED":
			response.ErrBadRequest(c, 40051, "系统内置角色不可删除")
		case "ROLE_IN_USE":
			response.ErrBadRequest(c, 40052, "仍有管理员使用该角色，无法删除")
		default:
			response.ErrInternal(c, "删除角色失败")
		}
		return
	}
	response.OK(c, gin.H{})
}

func (h *Handler) GetPermissionTree(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	tree, err := h.roleSvc.PermissionTree(c.Request.Context())
	if err != nil {
		response.ErrInternal(c, "获取权限树失败")
		return
	}
	response.OK(c, gin.H{"permissions": tree})
}

func mapRoleErr(c *gin.Context, err error) bool {
	if !service.IsRoleClientErr(err) {
		return false
	}
	msg := err.Error()
	switch msg {
	case "EMPTY_NAME":
		response.ErrValidation(c, "角色名称不能为空")
	case "EMPTY_PERMISSIONS":
		response.ErrValidation(c, "请选择权限")
	case "NAME_TOO_LONG":
		response.ErrValidation(c, "角色名称过长")
	case "DESCRIPTION_TOO_LONG":
		response.ErrValidation(c, "描述过长")
	case "DUPLICATE_ROLE_NAME":
		response.ErrBadRequest(c, 40053, "角色名称已存在")
	case "INVALID_PERMISSION", "INVALID_WILDCARD_MIX":
		response.ErrValidation(c, "权限代码无效")
	case "SYSTEM_ROLE_NAME_LOCKED":
		response.ErrBadRequest(c, 40054, "系统内置角色不可改名")
	default:
		return false
	}
	return true
}
