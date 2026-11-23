package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/response"
)

// SMSDisabled 是 /auth/sms/send 与 /auth/login/sms 在 v1 的统一响应。
//
// 短信通道在 v1 关闭（#162）：SMSLogin 不看 role、没有频控，验证码只写 Redis
// 从不真正下发——留在公网就是一条绕过密码登录的后门。真实短信 + 频控 +
// 角色校验落地前不得恢复这两个端点。这里不做请求体解析，避免把校验细节
// 当成「还活着」的信号。
func (h *Handler) SMSDisabled(c *gin.Context) {
	log.L(c.Request.Context()).Warn("sms endpoint rejected: disabled in v1", "path", c.FullPath())
	response.ErrForbidden(c, "短信功能在 v1 未开放")
}

type wechatLoginReq struct {
	AppID string `json:"app_id" binding:"required"`
	Code  string `json:"code" binding:"required"`
	// InviteCode 只对老师端有效；家长端携带会被拒绝（见 #175 / #168）。
	InviteCode string `json:"invite_code"`
	// Role 是客户端自报角色，仅用于一致性校验；授权角色以 app_id 对应客户端为准。
	Role string `json:"role"`
}

// WeChatLogin 小程序登录（#175）：角色与微信凭据都取自客户端注册表。
func (h *Handler) WeChatLogin(c *gin.Context) {
	var req wechatLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	result, err := h.authSvc.WeChatLogin(c.Request.Context(), service.WeChatLoginRequest{
		AppID:      req.AppID,
		Code:       req.Code,
		InviteCode: req.InviteCode,
		Role:       req.Role,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientUnavailable):
			log.L(c.Request.Context()).Warn("wechat login rejected: client unavailable", "app_id", req.AppID)
			response.ErrForbidden(c, "小程序客户端未注册或已停用，请更新小程序后重试")
		case errors.Is(err, service.ErrClientRoleMismatch):
			log.L(c.Request.Context()).Warn("wechat login rejected: body role mismatch", "app_id", req.AppID, "role", req.Role)
			response.ErrValidation(c, "请求角色与小程序客户端不一致")
		case errors.Is(err, service.ErrInviteCodeNotAllowed):
			log.L(c.Request.Context()).Warn("wechat login rejected: invite_code not allowed", "app_id", req.AppID)
			response.ErrValidation(c, "当前小程序不支持邀请码")
		case errors.Is(err, service.ErrWeChatLoginFailed):
			log.L(c.Request.Context()).Warn("wechat login code2session failed", "app_id", req.AppID, "err", err)
			response.ErrBadRequest(c, 40002, "微信登录凭证无效，请重试")
		default:
			log.L(c.Request.Context()).Error("wechat login failed", "app_id", req.AppID, "err", err)
			response.ErrInternal(c, "登录失败")
		}
		return
	}

	h.recordAuthAudit(c, "login", req.AppID, result)
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
		case "ADMIN_KINDERGARTEN_REQUIRED":
			response.ErrForbidden(c, "该账号未绑定园所，请联系平台方")
		case "ADMIN_KINDERGARTEN_CONFLICT":
			response.ErrForbidden(c, "该账号是多个园所的管理员，请联系平台方处理")
		case "KINDERGARTEN_SUSPENDED":
			response.ErrForbidden(c, "园所已停用，请联系平台方")
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
		switch {
		case errors.Is(err, service.ErrTokenRevoked):
			response.ErrUnauthorized(c, "登录状态已失效，请重新登录")
		case errors.Is(err, service.ErrAuthStorageUnavailable):
			log.L(c.Request.Context()).Error("refresh token blacklist check failed", "err", err)
			response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
		default:
			response.ErrUnauthorized(c, "刷新令牌无效或已过期")
		}
		return
	}

	response.OK(c, result)
}

type logoutReq struct {
	// 可选：客户端带上 refresh token 时一并吊销，否则登出后仍能拿它换新 access。
	RefreshToken string `json:"refresh_token"`
}

// Logout 吊销当前 access token；请求体带 refresh_token 时连同 refresh 会话一起吊销。
// 吊销存储写失败必须报错——返回 200 会让客户端以为服务端已经失效了 token。
func (h *Handler) Logout(c *gin.Context) {
	const prefix = "Bearer "
	header := c.GetHeader("Authorization")
	if len(header) <= len(prefix) || !strings.HasPrefix(header, prefix) {
		response.ErrUnauthorized(c, "缺少认证凭证")
		return
	}

	var req logoutReq
	// 允许空 body：客户端只发 `{}`，但也不该因为没带 refresh token 就失败。
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.ErrValidation(c, err.Error())
		return
	}

	if err := h.authSvc.Logout(c.Request.Context(), header[len(prefix):], req.RefreshToken); err != nil {
		log.L(c.Request.Context()).Error("logout revoke failed", "err", err)
		response.ErrUnavailable(c, "认证服务暂不可用，请稍后再试")
		return
	}

	response.JSON(c, http.StatusOK, response.Envelope{Code: 0, Message: "success", Data: gin.H{}})
}

// TokenRevoker 暴露吊销存储给路由装配。走 Handler 而不是改 NewRouter 签名，
// 是为了不触发 wire_gen.go 重新生成（本次改动不拥有 DI 生成物）。
func (h *Handler) TokenRevoker() auth.TokenRevoker {
	if h == nil || h.authSvc == nil {
		return nil
	}
	return h.authSvc.TokenRevoker()
}

// AuthService 暴露认证服务给路由装配的 consent 门禁使用；nil 时门禁直接放行。
func (h *Handler) AuthService() *service.AuthService {
	if h == nil {
		return nil
	}
	return h.authSvc
}

// TenantEpochStore 暴露园所账号级凭证失效存储给 AuthJWT 装配（#182）。
// 与 TokenRevoker 同理，走 Handler 以避免改 NewRouter 签名。
func (h *Handler) TenantEpochStore() auth.TenantEpochStore {
	if h == nil || h.authSvc == nil {
		return nil
	}
	return h.authSvc.TenantEpochStore()
}

type bindParentPhoneReq struct {
	WxPhoneCode string `json:"wx_phone_code" binding:"required"`
	// PublicCode 是园所入口码（#177）。绑定页从 scene/入口带入；没有时走无 scene 绑定。
	PublicCode string `json:"public_code"`
	// KindergartenID 是 public_code 之外的显式目标，二者只用于定位校验。
	KindergartenID uint64 `json:"kindergarten_id"`
}

// BindParentPhone 家长绑定手机号（#167）。
//
// 只服务家长端客户端：手机号来自家长小程序凭据的 GetPhoneNumber，openid 来自
// 当前 token（不接受请求体自称）。目标园所只做校验与初始落点，账号在其他
// shared 园所的既有成员关系不受影响。
func (h *Handler) BindParentPhone(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "登录状态已失效，请重新登录")
		return
	}

	var req bindParentPhoneReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	result, err := h.authSvc.BindParentPhone(c.Request.Context(), service.ParentBindPhoneInput{
		AppID:          claims.AppID,
		OpenID:         claims.OpenID,
		ClientIP:       c.ClientIP(),
		WxPhoneCode:    req.WxPhoneCode,
		PublicCode:     req.PublicCode,
		KindergartenID: req.KindergartenID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrClientUnavailable):
			log.L(c.Request.Context()).Warn("bind parent phone rejected: client unavailable", "app_id", claims.AppID)
			response.ErrForbidden(c, "小程序客户端未注册或已停用，请更新小程序后重试")
		case errors.Is(err, service.ErrBindClientNotParent):
			log.L(c.Request.Context()).Warn("bind parent phone rejected: not a parent client", "app_id", claims.AppID)
			response.ErrForbidden(c, "当前小程序不是家长端")
		case errors.Is(err, service.ErrBindPhoneIdentityMissing):
			response.ErrUnauthorized(c, "登录状态已失效，请重新登录")
		case errors.Is(err, service.ErrBindPhoneCodeInvalid):
			log.L(c.Request.Context()).Warn("bind parent phone: wechat phone code invalid", "app_id", claims.AppID, "err", err)
			response.ErrBadRequest(c, 40030, "微信手机号授权失败，请重试")
		case errors.Is(err, service.ErrBindKindergartenNotFound):
			response.ErrNotFound(c, "园所不存在或已停用")
		case errors.Is(err, service.ErrBindPhoneChannelMismatch):
			log.L(c.Request.Context()).Warn("bind parent phone rejected: channel mismatch", "app_id", claims.AppID, "err", err)
			response.ErrBadRequest(c, 40033, "当前小程序不支持该园所，请更新小程序或联系园所")
		case errors.Is(err, service.ErrBindPhoneNotEnrolled):
			// 无 scene 绑定的提示要引导家长「让园所录入 / 扫园所码」，两种情况文案一致可用。
			response.ErrBadRequest(c, 40031, "园所尚未录入该家长手机号，请先让园所录入或扫描园所二维码")
		case errors.Is(err, service.ErrBindPhoneAlreadyBound):
			log.L(c.Request.Context()).Warn("bind parent phone rejected: phone already bound", "app_id", claims.AppID, "err", err)
			response.ErrBadRequest(c, 40032, "该手机号已绑定其他微信")
		case errors.Is(err, service.ErrBindPhoneRateLimited):
			response.ErrRateLimited(c)
		case errors.Is(err, service.ErrKindergartenSuspended):
			response.ErrForbidden(c, "园所已停用，请联系平台方")
		default:
			log.L(c.Request.Context()).Error("bind parent phone failed", "app_id", claims.AppID, "err", err)
			response.ErrInternal(c, "绑定失败，请稍后再试")
		}
		return
	}

	h.recordAuthAudit(c, "bind_parent_phone", claims.AppID, result.LoginResult)
	response.OK(c, result)
}

func (h *Handler) HealthCheck(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}
