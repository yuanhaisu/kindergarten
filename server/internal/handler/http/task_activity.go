package handler

import (
	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

// 管理端多班活动端点（design D12 / tasks 7.1）。
// activity_id 由服务端生成，客户端只能通过本组端点管理分组。

// CreateTaskActivity POST /admin/task-activities
func (h *Handler) CreateTaskActivity(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.taskSvc.CreateActivity(c.Request.Context(), kgID, uid, req,
		c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		writeTaskError(c, err, "创建活动失败")
		return
	}
	response.Created(c, data)
}

// UpdateTaskActivity PUT /admin/task-activities/:activity_id
func (h *Handler) UpdateTaskActivity(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	activityID := c.Param("activity_id")
	var req service.UpdateActivityReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.taskSvc.UpdateActivity(c.Request.Context(), kgID, uid, activityID, req,
		c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		writeTaskError(c, err, "更新活动失败")
		return
	}
	response.OK(c, data)
}

// DeleteTaskActivity DELETE /admin/task-activities/:activity_id
func (h *Handler) DeleteTaskActivity(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	activityID := c.Param("activity_id")
	uid := middleware.GetUserID(c)
	data, err := h.taskSvc.DeleteActivity(c.Request.Context(), kgID, uid, activityID,
		c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		writeTaskError(c, err, "删除活动失败")
		return
	}
	response.OK(c, data)
}
