package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listAchievementsQuery struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

func (h *Handler) ListAchievements(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listAchievementsQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	list, total, page, pageSize, err := h.achievementSvc.ListAchievements(c.Request.Context(), kgID, pq)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取成就列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) CreateAchievement(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateAchievementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.achievementSvc.CreateAchievement(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "BAD_NAME":
			response.ErrValidation(c, "名称不能为空")
			return
		default:
			response.ErrInternal(c, "创建成就失败")
			return
		}
	}
	response.Created(c, data)
}

func (h *Handler) UpdateAchievement(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的成就 ID")
		return
	}
	var req service.UpdateAchievementReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.achievementSvc.UpdateAchievement(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "BAD_NAME":
			response.ErrValidation(c, "名称不能为空")
			return
		case "BAD_TARGET_VALUE":
			response.ErrValidation(c, "target_value 不能为负数")
			return
		case "EMPTY_UPDATE":
			response.ErrValidation(c, "请至少提交一个可更新字段")
			return
		default:
			if service.IsAchievementNotFound(err) {
				response.ErrNotFound(c, "成就不存在")
				return
			}
			response.ErrInternal(c, "更新成就失败")
			return
		}
	}
	response.OK(c, data)
}

func (h *Handler) DeleteAchievement(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的成就 ID")
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.achievementSvc.DeleteAchievement(c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsAchievementNotFound(err) {
			response.ErrNotFound(c, "成就不存在")
			return
		}
		response.ErrInternal(c, "删除成就失败")
		return
	}
	response.OK(c, gin.H{})
}
