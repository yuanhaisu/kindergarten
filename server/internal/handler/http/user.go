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

// GetMyKindergartens 我的园所（#169）：按当前客户端角色族与园所通道过滤。
func (h *Handler) GetMyKindergartens(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}

	items, err := h.authSvc.ListMyKindergartens(c.Request.Context(), middleware.GetUserID(c), claims)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientScopeRequired):
			response.ErrForbidden(c, "该功能仅限小程序客户端")
		case errors.Is(err, service.ErrClientUnavailable):
			response.ErrForbidden(c, "小程序客户端未注册或已停用，请更新小程序后重试")
		default:
			log.L(c.Request.Context()).Error("list my kindergartens failed", "err", err)
			response.ErrInternal(c, "获取园所列表失败")
		}
		return
	}

	response.OK(c, items)
}

type switchKindergartenReq struct {
	KindergartenID uint64 `json:"kindergarten_id" binding:"required"`
	RefreshToken   string `json:"refresh_token" binding:"required"`
}

// SwitchKindergarten 切换当前客户端的单园会话（#169）。
// 管理端 token 没有 app_id，统一 403；refresh 无效/被消费统一 401。
func (h *Handler) SwitchKindergarten(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}

	var req switchKindergartenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	result, err := h.authSvc.SwitchKindergarten(c.Request.Context(), middleware.GetUserID(c), claims, req.KindergartenID, req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientScopeRequired):
			response.ErrForbidden(c, "该功能仅限小程序客户端")
		case errors.Is(err, service.ErrClientUnavailable):
			response.ErrForbidden(c, "小程序客户端未注册或已停用，请更新小程序后重试")
		case errors.Is(err, service.ErrKindergartenNotMember):
			response.ErrForbidden(c, "目标园所不在当前客户端的可切换范围内")
		case errors.Is(err, service.ErrTokenRevoked):
			response.ErrUnauthorized(c, "登录状态已失效，请重新登录")
		case errors.Is(err, service.ErrAuthStorageUnavailable):
			log.L(c.Request.Context()).Error("switch kindergarten storage failed", "err", err)
			response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
		default:
			response.ErrUnauthorized(c, "刷新令牌无效或已过期")
		}
		return
	}

	h.recordAuthAudit(c, "switch_kindergarten", claims.AppID, result)
	response.OK(c, result)
}

func (h *Handler) GetProfile(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	// 小程序凭证只返回当前客户端角色族的角色（#175）；管理端为空作用域，行为不变。
	profile, err := h.authSvc.GetProfile(c.Request.Context(), uid, service.RoleScopeFromClaims(claims))
	if err != nil {
		if err.Error() == "user not found" {
			response.ErrNotFound(c, "用户不存在")
			return
		}
		response.ErrInternal(c, "获取用户信息失败")
		return
	}

	response.OK(c, profile)
}

type updateProfileReq struct {
	Nickname string `json:"nickname"`
	Gender   int    `json:"gender"`
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	var req updateProfileReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	result, err := h.authSvc.UpdateProfile(c.Request.Context(), uid, req.Nickname, req.Gender, service.RoleScopeFromClaims(claims))
	if err != nil {
		response.ErrInternal(c, "更新个人信息失败")
		return
	}

	response.OK(c, result)
}

type updateAvatarReq struct {
	Avatar string `json:"avatar" binding:"required"`
}

func (h *Handler) UpdateAvatar(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	var req updateAvatarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	avatar, err := h.authSvc.UpdateAvatar(c.Request.Context(), uid, req.Avatar)
	if err != nil {
		response.ErrInternal(c, "更新头像失败")
		return
	}

	response.OK(c, gin.H{"avatar": avatar})
}
