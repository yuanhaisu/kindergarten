package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// 客户端身份支持操作（#172）。
//
// 园所管理端与平台控制台是两个信任域：前者从 JWT 取园所与操作人，
// 后者从 scope=platform 凭证取平台操作者。两边都调同一个 service，
// 审计分别落 operation_logs / platform_operation_logs。

func adminSupportActor(c *gin.Context) service.SupportActor {
	return service.SupportActor{
		Kind:           service.SupportActorAdmin,
		AdminUserID:    middleware.GetUserID(c),
		KindergartenID: middleware.GetKindergartenID(c),
		IP:             c.ClientIP(),
		UserAgent:      c.GetHeader("User-Agent"),
	}
}

func platformSupportActor(c *gin.Context) service.SupportActor {
	return service.SupportActor{
		Kind:             service.SupportActorPlatform,
		PlatformOperator: middleware.GetPlatformAdminID(c),
		IP:               c.ClientIP(),
		UserAgent:        c.GetHeader("User-Agent"),
	}
}

func supportErr(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrSupportKindergartenRequired):
		response.ErrForbidden(c, "未绑定园所，请重新登录")
	case errors.Is(err, service.ErrSupportUserNotFound):
		response.ErrNotFound(c, "账号不存在")
	case errors.Is(err, service.ErrSupportIdentityNotFound):
		response.ErrNotFound(c, "客户端身份不存在")
	case errors.Is(err, service.ErrSupportInvalidPhone):
		response.ErrValidation(c, "手机号格式不正确")
	case errors.Is(err, service.ErrSupportIdentityRemains):
		response.ErrConflict(c, "该账号仍有微信身份，请先解绑错误身份再更正手机号")
	case errors.Is(err, service.ErrSupportPhoneConflict):
		response.ErrConflict(c, "新手机号已被其他账号占用")
	default:
		log.L(c.Request.Context()).Error(fallback, "err", err)
		response.ErrInternal(c, fallback)
	}
}

func parseUserIDParam(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的账号 ID")
		return 0, false
	}
	return id, true
}

// AdminUnbindParentIdentity 园所管理员把某个微信从家长账号上解绑。
// 只释放身份，账号与孩子关系保持不变；错误手机号更正后家长可重新绑定。
func (h *Handler) AdminUnbindParentIdentity(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	identityID, err := strconv.ParseUint(c.Param("identity_id"), 10, 64)
	if err != nil || identityID == 0 {
		response.ErrValidation(c, "无效的客户端身份 ID")
		return
	}
	result, err := h.supportSvc.UnbindIdentity(c.Request.Context(), adminSupportActor(c), userID, identityID)
	if err != nil {
		supportErr(c, err, "解绑客户端身份失败")
		return
	}
	response.OK(c, result)
}

// AdminGetParentSupport 园所管理员查看家长账号的支持视图：
// 手机号、客户端身份（掩码）与本园成员关系。
func (h *Handler) AdminGetParentSupport(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	view, err := h.supportSvc.ViewUser(c.Request.Context(), adminSupportActor(c), userID)
	if err != nil {
		supportErr(c, err, "获取账号支持信息失败")
		return
	}
	response.OK(c, view)
}

// AdminCorrectParentPhone 园所管理员更正家长账号手机号。
// 账号上仍有微信身份时拒绝；先解绑，再改号，由正确家长重新绑定。
func (h *Handler) AdminCorrectParentPhone(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请输入手机号")
		return
	}
	view, err := h.supportSvc.CorrectPhone(c.Request.Context(), adminSupportActor(c), userID, req.Phone)
	if err != nil {
		supportErr(c, err, "更正手机号失败")
		return
	}
	response.OK(c, view)
}

// PlatformGetUserSupport 平台支持查看目标用户的全部园所成员关系与客户端身份。
// 平台信任域不受园所限制，审计落 platform_operation_logs。
func (h *Handler) PlatformGetUserSupport(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	view, err := h.supportSvc.ViewUser(c.Request.Context(), platformSupportActor(c), userID)
	if err != nil {
		supportErr(c, err, "获取账号支持信息失败")
		return
	}
	response.OK(c, view)
}

// PlatformUnbindUserIdentity 平台支持解绑客户端身份。
func (h *Handler) PlatformUnbindUserIdentity(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	identityID, err := strconv.ParseUint(c.Param("identity_id"), 10, 64)
	if err != nil || identityID == 0 {
		response.ErrValidation(c, "无效的客户端身份 ID")
		return
	}
	result, err := h.supportSvc.UnbindIdentity(c.Request.Context(), platformSupportActor(c), userID, identityID)
	if err != nil {
		supportErr(c, err, "解绑客户端身份失败")
		return
	}
	response.OK(c, result)
}

// PlatformCorrectUserPhone 平台支持更正账号手机号。
func (h *Handler) PlatformCorrectUserPhone(c *gin.Context) {
	userID, ok := parseUserIDParam(c)
	if !ok {
		return
	}
	var req struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请输入手机号")
		return
	}
	view, err := h.supportSvc.CorrectPhone(c.Request.Context(), platformSupportActor(c), userID, req.Phone)
	if err != nil {
		supportErr(c, err, "更正手机号失败")
		return
	}
	response.OK(c, view)
}
