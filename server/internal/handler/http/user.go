package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/pkg/response"
)

func (h *Handler) GetProfile(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	profile, err := h.authSvc.GetProfile(c.Request.Context(), uid)
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

	result, err := h.authSvc.UpdateProfile(c.Request.Context(), uid, req.Nickname, req.Gender)
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
