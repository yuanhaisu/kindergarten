package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/pkg/response"
)

type sendSMSReq struct {
	Phone string `json:"phone" binding:"required"`
	Scene string `json:"scene" binding:"required,oneof=login bind"`
}

func (h *Handler) SendSMS(c *gin.Context) {
	var req sendSMSReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	expire, err := h.authSvc.SendSMS(c.Request.Context(), req.Phone, req.Scene)
	if err != nil {
		if err.Error() == "SMS_TOO_FREQUENT" {
			response.ErrBadRequest(c, 40010, "验证码发送过于频繁，请稍后再试")
			return
		}
		response.ErrInternal(c, "验证码发送失败")
		return
	}

	response.OK(c, gin.H{"expire_seconds": expire})
}

type smsLoginReq struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
	Role  string `json:"role" binding:"required,oneof=teacher parent admin"`
}

func (h *Handler) SMSLogin(c *gin.Context) {
	var req smsLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	result, err := h.authSvc.SMSLogin(c.Request.Context(), req.Phone, req.Code, req.Role)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "SMS_CODE_EXPIRED":
			response.ErrBadRequest(c, 40001, "验证码错误或已过期")
		case "SMS_CODE_INVALID":
			response.ErrBadRequest(c, 40001, "验证码错误或已过期")
		default:
			response.ErrInternal(c, "登录失败")
		}
		return
	}

	response.OK(c, result)
}

type wechatLoginReq struct {
	Code          string `json:"code" binding:"required"`
	Role          string `json:"role" binding:"required,oneof=teacher parent"`
	EncryptedData string `json:"encrypted_data"`
	IV            string `json:"iv"`
	InviteCode    string `json:"invite_code"`
}

func (h *Handler) WeChatLogin(c *gin.Context) {
	var req wechatLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	result, err := h.authSvc.WeChatLogin(c.Request.Context(), req.Code, req.Role, req.InviteCode)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "INVITE_CODE_INVALID":
			response.ErrBadRequest(c, 40020, "邀请码无效或已过期")
		case "INVITE_USER_NOT_FOUND":
			response.ErrBadRequest(c, 40021, "邀请码关联的教师账号不存在")
		default:
			response.ErrBadRequest(c, 40002, "微信登录失败")
		}
		return
	}

	response.OK(c, result)
}

type passwordLoginReq struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) PasswordLogin(c *gin.Context) {
	var req passwordLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	result, err := h.authSvc.PasswordLogin(c.Request.Context(), req.Phone, req.Password)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "USER_NOT_FOUND", "PASSWORD_INCORRECT":
			response.ErrBadRequest(c, 40003, "账号或密码错误")
		case "NOT_ADMIN":
			response.ErrForbidden(c, "该账号无管理后台权限")
		default:
			response.ErrInternal(c, "登录失败")
		}
		return
	}

	response.OK(c, result)
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	result, err := h.authSvc.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.ErrUnauthorized(c, "刷新令牌无效或已过期")
		return
	}

	response.OK(c, result)
}

func (h *Handler) Logout(c *gin.Context) {
	header := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.HasPrefix(header, prefix) {
		tokenStr := header[len(prefix):]
		_ = h.authSvc.Logout(c.Request.Context(), tokenStr)
	}

	response.JSON(c, http.StatusOK, response.Envelope{Code: 0, Message: "success", Data: gin.H{}})
}

type bindInviteReq struct {
	InviteCode string `json:"invite_code" binding:"required"`
}

func (h *Handler) BindInvite(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == 0 {
		response.ErrUnauthorized(c, "请先登录")
		return
	}
	var req bindInviteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	result, err := h.authSvc.BindInvite(c.Request.Context(), uid, req.InviteCode)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "INVITE_CODE_INVALID":
			response.ErrBadRequest(c, 40020, "邀请码无效或已过期")
		case "INVITE_USER_NOT_FOUND":
			response.ErrBadRequest(c, 40021, "邀请码关联的教师账号不存在")
		case "INVITE_SELF_BIND":
			response.ErrBadRequest(c, 40022, "不能绑定自己")
		default:
			response.ErrInternal(c, "绑定失败")
		}
		return
	}
	response.OK(c, result)
}

type bindParentPhoneReq struct {
	WxPhoneCode string `json:"wx_phone_code" binding:"required"`
}

func (h *Handler) BindParentPhone(c *gin.Context) {
	uid := middleware.GetUserID(c)
	if uid == 0 {
		response.ErrUnauthorized(c, "请先登录")
		return
	}
	var req bindParentPhoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	result, err := h.authSvc.BindParentPhone(c.Request.Context(), uid, req.WxPhoneCode)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "PHONE_NOT_FOUND":
			response.ErrBadRequest(c, 40030, "未获取到手机号，请重新授权")
		case "PARENT_RECORD_NOT_FOUND":
			response.ErrBadRequest(c, 40031, "未找到匹配的家长信息，请联系管理员先创建家长账号")
		default:
			response.ErrInternal(c, "手机号绑定失败")
		}
		return
	}
	response.OK(c, result)
}

func (h *Handler) HealthCheck(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}
