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
		if errors.Is(err, service.ErrTaskBusy) {
			response.ErrUnavailable(c, "操作冲突，请稍后重试")
			return
		}
		response.ErrInternal(c, "删除任务失败")
		return
	}
	response.OK(c, gin.H{})
}

// RevokeTaskPublicCheckin PUT /admin/tasks/checkins/:checkin_id/public
// 管理端只能撤回：不接受 approve=true，避免绕过老师的内容边界确认。
func (h *Handler) RevokeTaskPublicCheckin(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	checkinID, err := strconv.ParseUint(c.Param("checkin_id"), 10, 64)
	if err != nil || checkinID == 0 {
		response.ErrValidation(c, "无效的提交 ID")
		return
	}
	var req struct {
		Revoke bool `json:"revoke"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	if !req.Revoke {
		response.ErrValidation(c, "仅支持撤回操作")
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.taskSvc.RevokePublicCheckin(c.Request.Context(), kgID, uid, checkinID, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		if err.Error() == "NO_KINDERGARTEN" {
			response.ErrForbidden(c, "未绑定园所，请重新登录")
			return
		}
		if service.IsTaskNotFound(err) {
			response.ErrNotFound(c, "提交不存在")
			return
		}
		if errors.Is(err, service.ErrTaskBusy) {
			response.ErrUnavailable(c, "操作冲突，请稍后重试")
			return
		}
		response.ErrInternal(c, "撤回公开内容失败")
		return
	}
	response.OK(c, data)
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

// writeTaskError 把 TaskService 的哨兵/字符串错误映射成稳定的 HTTP 语义。
//
// 结构锁、班级罐缺失、目标孩子越权都必须给出「能照着修」的文案：
// 全部折成 500 或笼统的「参数错误」时，调用方只能猜是哪个字段。
func writeTaskError(c *gin.Context, err error, fallback string) {
	switch {
	case err.Error() == "NO_KINDERGARTEN":
		response.ErrForbidden(c, "未绑定园所，请重新登录")
	case errors.Is(err, service.ErrClassNotFound):
		response.ErrNotFound(c, "班级不存在")
	case service.IsTaskNotFound(err):
		response.ErrNotFound(c, "任务不存在")
	case errors.Is(err, service.ErrTaskStructuralLocked):
		response.ErrValidation(c, "任务已有提交，只能修改标题、说明、学习关联与公开开关")
	case errors.Is(err, service.ErrTaskTargetInvalid):
		response.ErrValidation(c, "目标孩子不属于该班级")
	case errors.Is(err, service.ErrTaskStarJarMissing):
		response.ErrValidation(c, "请先配置班级星星罐")
	case errors.Is(err, service.ErrTaskStarGoalNegative):
		response.ErrValidation(c, "星星目标不能为负数")
	case errors.Is(err, service.ErrTaskStarGoalInvalid):
		response.ErrValidation(c, "选择班级星星仪式且按任务统计时，星星目标必须大于 0")
	case errors.Is(err, service.ErrTaskRitualInvalid):
		response.ErrValidation(c, "recognition_ritual 须为 sticker / voice / class_star")
	case errors.Is(err, service.ErrTaskStarScopeInvalid):
		response.ErrValidation(c, "star_scope 须为 task / class")
	case errors.Is(err, service.ErrTaskActivityIDForbidden):
		response.ErrValidation(c, "独立任务不接受 activity_id，请使用 /admin/task-activities 创建多班活动")
	case errors.Is(err, service.ErrStarJarGoalInvalid):
		response.ErrValidation(c, "班级星星罐目标必须大于 0")
	case errors.Is(err, service.ErrTaskBusy):
		response.ErrUnavailable(c, "操作冲突，请稍后重试")
	case err.Error() == "BAD_FREQUENCY":
		response.ErrValidation(c, "frequency 须为 daily / weekly / once")
	case err.Error() == "BAD_DATE", err.Error() == "BAD_DATE_RANGE":
		response.ErrValidation(c, "日期参数无效")
	case err.Error() == "BAD_STATUS":
		response.ErrValidation(c, "status 须为 0（已结束）或 1（进行中）")
	case err.Error() == "BAD_TITLE":
		response.ErrValidation(c, "标题不能为空")
	case err.Error() == "BAD_LEARNING_LINK":
		response.ErrValidation(c, "学习关联必填")
	default:
		response.ErrInternal(c, fallback)
	}
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
		writeTaskError(c, err, "创建任务失败")
		return
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
		writeTaskError(c, err, "更新任务失败")
		return
	}
	response.OK(c, gin.H{"id": id})
}

// GetClassStarJar GET /admin/classes/:id/star-jar
func (h *Handler) GetClassStarJar(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	classID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || classID == 0 {
		response.ErrValidation(c, "无效的班级 ID")
		return
	}
	data, err := h.taskSvc.GetClassStarJar(c.Request.Context(), kgID, classID)
	if err != nil {
		writeTaskError(c, err, "获取班级星星罐失败")
		return
	}
	response.OK(c, data)
}

// SetClassStarJar PUT /admin/classes/:id/star-jar
func (h *Handler) SetClassStarJar(c *gin.Context) {
	kgID := middleware.GetKindergartenID(c)
	if kgID == 0 {
		response.ErrForbidden(c, "未绑定园所，请重新登录")
		return
	}
	classID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || classID == 0 {
		response.ErrValidation(c, "无效的班级 ID")
		return
	}
	var req struct {
		Goal int `json:"goal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	uid := middleware.GetUserID(c)
	data, err := h.taskSvc.SetClassStarJar(c.Request.Context(), kgID, classID, uid, req.Goal,
		c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		writeTaskError(c, err, "设置班级星星罐失败")
		return
	}
	response.OK(c, data)
}
