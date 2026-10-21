package handler

import (
	"errors"
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

// teacherGrantReq 老师授予徽章的请求体。
// log_id 可选：给出时把该成长记录标为获奖见证（design D1：见证可空）。
type teacherGrantReq struct {
	StudentIDs []uint64 `json:"student_ids" binding:"required,min=1"`
	LogID      *uint64  `json:"log_id"`
}

// TeacherGrantAchievement 老师把徽章授予学生，可选绑定成长记录作为获奖见证。
// 错误映射：400 徽章未启用/见证无效/参数错 · 403 跨园所或非本班 · 409 重复授予
func (h *Handler) TeacherGrantAchievement(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	achievementID, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req teacherGrantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}

	granted, err := h.achievementSvc.GrantToStudent(
		c.Request.Context(),
		tc.KindergartenID, tc.ClassID, tc.UserID, achievementID,
		req.StudentIDs, req.LogID,
	)
	if err != nil {
		// 语义化错误 → 4xx；其余走 500。不得把业务错误吞成 500。
		switch {
		case errors.Is(err, service.ErrAchievementDisabled):
			response.ErrBadRequest(c, 40010, "徽章不存在或未启用")
		case errors.Is(err, service.ErrGrantWitnessInvalid):
			response.ErrBadRequest(c, 40011, "获奖见证记录无效，或该学生对它不可见")
		case errors.Is(err, service.ErrGrantStudentNotFound):
			response.ErrBadRequest(c, 40012, "学生不存在或参数为空")
		case errors.Is(err, service.ErrGrantCrossKindergarten):
			response.ErrForbidden(c, "只能向自己班级的学生授予本园徽章")
		case errors.Is(err, service.ErrGrantDuplicate):
			response.Err(c, 409, 40910, "该学生已获得过这枚徽章")
		default:
			response.ErrInternal(c, "授予徽章失败")
		}
		return
	}
	response.Created(c, map[string]any{
		"achievement_id":      achievementID,
		"granted_student_ids": granted,
		"granted_count":       len(granted),
		"log_id":              req.LogID,
	})
}

// TeacherListAchievements 老师端授予入口的徽章选择器：只列本园已启用的徽章。
func (h *Handler) TeacherListAchievements(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	list, err := h.achievementSvc.ListEnabled(c.Request.Context(), tc.KindergartenID)
	if err != nil {
		response.ErrInternal(c, "获取徽章列表失败")
		return
	}
	response.OK(c, map[string]any{"list": list, "total": len(list)})
}
