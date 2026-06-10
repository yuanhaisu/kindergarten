package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
	"kindergarten-server/pkg/response"
)

type listTasksQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
	ClassID  uint64 `form:"class_id"`
	Status   *int8  `form:"status"`
}

func (h *Handler) ListTasks(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q listTasksQuery
	_ = c.ShouldBindQuery(&q)
	pq := pagination.Query{Page: q.Page, PageSize: q.PageSize}
	var classID *uint64
	if q.ClassID > 0 {
		classID = &q.ClassID
	}
	list, total, page, pageSize, err := h.taskSvc.ListTasks(c.Request.Context(), kgID, pq, classID, q.Status)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		response.ErrInternal(c, "获取任务列表失败")
		return
	}
	response.OKList(c, list, total, page, pageSize)
}

func (h *Handler) GetTask(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的任务 ID")
		return
	}
	data, err := h.taskSvc.GetTaskDetail(c.Request.Context(), kgID, id)
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsTaskNotFound(err) {
			response.ErrNotFound(c, "任务不存在")
			return
		}
		response.ErrInternal(c, "获取任务详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) DeleteTask(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的任务 ID")
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.taskSvc.DeleteTask(c.Request.Context(), kgID, uid, id, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsTaskNotFound(err) {
			response.ErrNotFound(c, "任务不存在")
			return
		}
		response.ErrInternal(c, "删除任务失败")
		return
	}
	response.OK(c, gin.H{})
}

type taskStatsQuery struct {
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

func (h *Handler) GetTaskStats(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var q taskStatsQuery
	_ = c.ShouldBindQuery(&q)
	data, err := h.taskSvc.GetTaskStats(c.Request.Context(), kgID, q.StartDate, q.EndDate)
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "BAD_DATE", "BAD_DATE_RANGE":
			response.ErrValidation(c, "日期参数无效")
			return
		default:
			response.ErrInternal(c, "获取任务统计失败")
			return
		}
	}
	response.OK(c, data)
}

func (h *Handler) CreateTask(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	var req service.CreateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.taskSvc.CreateTask(c.Request.Context(), kgID, uid, req, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "CLASS_NOT_FOUND":
			response.ErrNotFound(c, "班级不存在")
			return
		case "BAD_FREQUENCY":
			response.ErrValidation(c, "frequency 须为 daily / weekly / once")
			return
		case "BAD_DATE", "BAD_DATE_RANGE":
			response.ErrValidation(c, "日期参数无效")
			return
		default:
			response.ErrInternal(c, "创建任务失败")
			return
		}
	}
	response.Created(c, data)
}

func (h *Handler) UpdateTask(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的任务 ID")
		return
	}
	var req service.UpdateTaskReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	if err := h.taskSvc.UpdateTask(c.Request.Context(), kgID, uid, id, req, c.ClientIP(), c.GetHeader("User-Agent")); err != nil {
		switch err.Error() {
		case "NO_KINDERGARTEN":
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		case "CLASS_NOT_FOUND":
			response.ErrNotFound(c, "班级不存在")
			return
		case "BAD_FREQUENCY":
			response.ErrValidation(c, "frequency 须为 daily / weekly / once")
			return
		case "BAD_DATE", "BAD_DATE_RANGE":
			response.ErrValidation(c, "日期参数无效")
			return
		case "BAD_STATUS":
			response.ErrValidation(c, "status 须为 0（已结束）或 1（进行中）")
			return
		default:
			if service.IsTaskNotFound(err) {
				response.ErrNotFound(c, "任务不存在")
				return
			}
			response.ErrInternal(c, "更新任务失败")
			return
		}
	}
	response.OK(c, gin.H{"id": id})
}
