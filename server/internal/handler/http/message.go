package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/pkg/response"
)

func (h *Handler) ListMessages(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	msgType := c.Query("type")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	var isRead *bool
	if v := c.Query("is_read"); v != "" {
		b := v == "true"
		isRead = &b
	}

	items, total, err := h.msgSvc.List(c.Request.Context(), uid, msgType, isRead, page, pageSize)
	if err != nil {
		response.ErrInternal(c, "获取消息列表失败")
		return
	}

	response.OKList(c, items, total, page, pageSize)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	result, err := h.msgSvc.UnreadCount(c.Request.Context(), uid)
	if err != nil {
		response.ErrInternal(c, "获取未读消息数失败")
		return
	}

	response.OK(c, result)
}

func (h *Handler) MarkMessageRead(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	msgID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.ErrValidation(c, "无效的消息ID")
		return
	}

	result, err := h.msgSvc.MarkRead(c.Request.Context(), msgID, uid)
	if err != nil {
		response.ErrInternal(c, "标记已读失败")
		return
	}
	if result == nil {
		response.ErrNotFound(c, "消息不存在")
		return
	}

	response.OK(c, result)
}

type markAllReadReq struct {
	Type string `json:"type"`
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	claims := middleware.GetClaims(c)
	if claims == nil {
		response.ErrUnauthorized(c, "未认证")
		return
	}
	uid, _ := strconv.ParseUint(claims.UserID, 10, 64)

	var req markAllReadReq
	_ = c.ShouldBindJSON(&req)

	result, err := h.msgSvc.MarkAllRead(c.Request.Context(), uid, req.Type)
	if err != nil {
		response.ErrInternal(c, "标记全部已读失败")
		return
	}

	response.OK(c, result)
}
