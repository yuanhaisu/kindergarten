package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// platformOperatorID 当前登录的平台账号 ID。停用自己的守卫以 claims 为准，
// 不信任请求体里的任何身份字段。
func (h *Handler) platformOperatorID(c *gin.Context) uint64 {
	id, err := strconv.ParseUint(middleware.GetPlatformAdminID(c), 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// PlatformListAccounts 平台账号列表。status 为 enabled/disabled，不含口令与 role。
func (h *Handler) PlatformListAccounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	list, total, err := h.platformSvc.ListAccounts(c.Request.Context(), page, pageSize, c.Query("keyword"))
	if err != nil {
		writeAccountError(c, err)
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

// PlatformCreateAccount 新建平台账号。
func (h *Handler) PlatformCreateAccount(c *gin.Context) {
	var req service.CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	row, err := h.platformSvc.CreateAccount(c.Request.Context(), req, h.operatorOf(c), c.ClientIP())
	if err != nil {
		writeAccountError(c, err)
		return
	}
	response.Created(c, row)
}

// PlatformSetAccountStatus 启用或停用平台账号。
func (h *Handler) PlatformSetAccountStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "账号 ID 不合法")
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	if err := h.platformSvc.SetAccountStatus(
		c.Request.Context(), id, req.Status, h.platformOperatorID(c), h.operatorOf(c), c.ClientIP(),
	); err != nil {
		writeAccountError(c, err)
		return
	}
	response.OK(c, gin.H{"id": id, "status": req.Status, "updated_at": time.Now().Format(time.RFC3339)})
}

// PlatformResetAccountPassword 重置指定账号口令。口令由操作者提供，响应不回显。
func (h *Handler) PlatformResetAccountPassword(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "账号 ID 不合法")
		return
	}
	var req struct {
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	if err := h.platformSvc.ResetAccountPassword(
		c.Request.Context(), id, req.Password, h.operatorOf(c), c.ClientIP(),
	); err != nil {
		writeAccountError(c, err)
		return
	}
	response.OK(c, gin.H{"id": id, "updated_at": time.Now().Format(time.RFC3339)})
}

// PlatformChangeOwnPassword 平台账号修改本人密码。成功后当前凭证立即失效，
// 客户端应回到登录页。
func (h *Handler) PlatformChangeOwnPassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请求参数不合法")
		return
	}
	operatorID := h.platformOperatorID(c)
	if operatorID == 0 {
		response.ErrUnauthorized(c, "登录状态无效，请重新登录")
		return
	}
	if err := h.platformSvc.ChangeOwnPassword(c.Request.Context(), operatorID, req.OldPassword, req.NewPassword); err != nil {
		writeAccountError(c, err)
		return
	}
	response.OK(c, gin.H{"changed": true})
}

// writeAccountError 是平台账号相关错误的唯一映射出口，保证前后端对
// 守卫拒绝与口令策略的提示一致。
func writeAccountError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrAccountNotFound):
		response.ErrNotFound(c, "平台账号不存在")
	case errors.Is(err, service.ErrUsernameExists):
		response.ErrConflict(c, "用户名已存在")
	case errors.Is(err, service.ErrInvalidAccountStatus):
		response.ErrValidation(c, "状态取值不合法：只能是 enabled 或 disabled")
	case errors.Is(err, service.ErrCannotDisableSelf):
		response.ErrConflict(c, "不能停用当前登录账号")
	case errors.Is(err, service.ErrLastAvailableAccount):
		response.ErrConflict(c, "这是最后一个可用账号，停用后将无法登录平台")
	case errors.Is(err, service.ErrInvalidOldPassword):
		response.ErrValidation(c, "原密码不正确")
	case errors.Is(err, service.ErrWeakPassword):
		response.ErrValidation(c, "密码强度不足：至少 8 位，且同时包含大写字母、小写字母和数字")
	case errors.Is(err, service.ErrPlatformAuthUnavailable):
		response.ErrUnavailable(c, "认证服务暂不可用，请稍后重试")
	case errors.Is(err, service.ErrInvalidUsername):
		response.ErrValidation(c, "用户名不能为空")
	default:
		response.ErrInternal(c, "平台账号操作失败")
	}
}
