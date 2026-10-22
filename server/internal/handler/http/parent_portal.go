package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

func (h *Handler) parentCtx(c *gin.Context) (*service.ParentPortalContext, bool) {
	sid := c.Query("student_id")
	var sidPtr *uint64
	if sid != "" {
		if id, err := strconv.ParseUint(sid, 10, 64); err == nil && id > 0 {
			sidPtr = &id
		}
	}
	pc, err := h.parentPortalSvc.ResolveContext(c.Request.Context(), middleware.GetUserID(c), middleware.GetKindergartenID(c), sidPtr)
	if err != nil {
		response.ErrForbidden(c, "未绑定孩子信息,请联系管理员")
		return nil, false
	}
	return pc, true
}

// parentTaskCtx 任务接口专用：student_id 必填且必须属于当前家长。
// ResolveContext 在 student_id 不属于该家长时会回退到第一个孩子，
// 任务链路不能接受这种回退（会把 A 孩子的任务显示成 B 孩子的）。
func (h *Handler) parentTaskCtx(c *gin.Context) (*service.ParentPortalContext, bool) {
	raw := c.Query("student_id")
	if raw == "" {
		response.ErrValidation(c, "缺少 student_id")
		return nil, false
	}
	sid, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || sid == 0 {
		response.ErrValidation(c, "无效的 student_id")
		return nil, false
	}
	pc, err := h.parentPortalSvc.ResolveContext(c.Request.Context(), middleware.GetUserID(c), middleware.GetKindergartenID(c), &sid)
	if err != nil {
		response.ErrForbidden(c, "未绑定孩子信息,请联系管理员")
		return nil, false
	}
	if pc.CurrentStudent == nil || pc.CurrentStudent.ID != sid {
		response.ErrForbidden(c, "无权查看该孩子的任务")
		return nil, false
	}
	return pc, true
}

func ppID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的 ID")
		return 0, false
	}
	return id, true
}

func ppPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if ps < 1 || ps > 50 {
		ps = 20
	}
	return page, ps
}

// ========== Home ==========

func (h *Handler) ParentHomeOverview(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	data, err := h.parentPortalSvc.HomeOverview(c.Request.Context(), pc)
	if err != nil {
		response.ErrInternal(c, "获取首页概览失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) ParentSwitchStudent(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	var req struct {
		StudentID uint64 `json:"student_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请选择孩子")
		return
	}
	for _, s := range pc.Students {
		if s.ID == req.StudentID {
			cn := ""
			if s.Class != nil {
				cn = s.Class.Name
			}
			response.OK(c, gin.H{"current_student": gin.H{"id": s.ID, "name": s.Name, "avatar": s.Avatar, "class_name": cn, "class_id": s.ClassID}})
			return
		}
	}
	response.ErrNotFound(c, "未找到该孩子")
}

func (h *Handler) ParentAnnouncements(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	result, err := h.parentPortalSvc.ListAnnouncements(c.Request.Context(), pc, page, ps)
	if err != nil {
		response.ErrInternal(c, "获取公告失败")
		return
	}
	response.OK(c, result)
}
func (h *Handler) ParentHomeFeeds(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	result, err := h.parentPortalSvc.ListHomeFeeds(c.Request.Context(), pc, page, ps)
	if err != nil {
		response.ErrInternal(c, "获取动态失败")
		return
	}
	response.OK(c, result)
}

// ========== Notices ==========

func (h *Handler) ParentMarkNoticeRead(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	if err := h.parentPortalSvc.MarkNoticeRead(c.Request.Context(), pc, id); err != nil {
		response.ErrInternal(c, "标记已读失败")
		return
	}
	response.OK(c, gin.H{"id": id})
}

func (h *Handler) ParentListNotices(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	result, err := h.parentPortalSvc.ListNotices(c.Request.Context(), pc, c.Query("category"), page, ps)
	if err != nil {
		response.ErrInternal(c, "获取通知列表失败")
		return
	}
	response.OK(c, result)
}
func (h *Handler) ParentGetNotice(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	data, err := h.parentPortalSvc.GetNotice(c.Request.Context(), pc, id)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "通知不存在")
			return
		}
		response.ErrInternal(c, "获取通知失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentConfirmNotice(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	data, err := h.parentPortalSvc.ConfirmNotice(c.Request.Context(), pc, id)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "通知不存在")
			return
		}
		response.ErrInternal(c, "确认通知失败")
		return
	}
	response.OK(c, data)
}

// ========== Leaves ==========

func (h *Handler) ParentListLeaves(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	result, err := h.parentPortalSvc.ListLeaves(c.Request.Context(), pc, c.Query("status"), page, ps)
	if err != nil {
		response.ErrInternal(c, "获取请假列表失败")
		return
	}
	response.OK(c, result)
}
func (h *Handler) ParentGetLeave(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	data, err := h.parentPortalSvc.GetLeave(c.Request.Context(), pc, id)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "请假记录不存在")
			return
		}
		response.ErrInternal(c, "获取请假详情失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentCreateLeave(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	var req service.ParentLeaveInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数无效")
		return
	}
	data, err := h.parentPortalSvc.CreateLeave(c.Request.Context(), pc, req)
	if err != nil {
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "参数无效")
			return
		}
		if errors.Is(err, service.ErrParentForbidden) {
			response.ErrForbidden(c, "无权为该学生请假")
			return
		}
		response.ErrInternal(c, "创建请假失败")
		return
	}
	response.Created(c, data)
}
func (h *Handler) ParentUpdateLeave(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	var req service.ParentLeaveInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数无效")
		return
	}
	if err := h.parentPortalSvc.UpdateLeave(c.Request.Context(), pc, id, req); err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "请假记录不存在")
			return
		}
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "仅可编辑待审批的请假")
			return
		}
		response.ErrInternal(c, "更新请假失败")
		return
	}
	response.OK(c, gin.H{"id": id})
}
func (h *Handler) ParentCancelLeave(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	if err := h.parentPortalSvc.CancelLeave(c.Request.Context(), pc, id); err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "请假记录不存在")
			return
		}
		response.ErrInternal(c, "取消请假失败")
		return
	}
	response.OK(c, gin.H{"id": id, "status": 3, "status_label": "已撤销"})
}

// ========== Growth ==========

func (h *Handler) ParentGrowthFeed(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	sid := c.Query("student_id")
	var sidPtr *uint64
	if sid != "" {
		if id, err := strconv.ParseUint(sid, 10, 64); err == nil && id > 0 {
			sidPtr = &id
		}
	}
	year, _ := strconv.Atoi(c.DefaultQuery("year", "0"))
	month, _ := strconv.Atoi(c.DefaultQuery("month", "0"))
	result, err := h.parentPortalSvc.ListGrowthFeed(c.Request.Context(), pc, sidPtr, year, month, page, ps)
	if err != nil {
		response.ErrInternal(c, "加载成长记录失败")
		return
	}
	list, _ := result["list"]
	total, _ := result["total"].(int64)
	response.OKList(c, list, total, page, ps)
}
func (h *Handler) ParentGrowthDetail(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	data, err := h.parentPortalSvc.GrowthFeedDetail(c.Request.Context(), pc, id)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "成长记录不存在")
			return
		}
		if errors.Is(err, service.ErrParentForbidden) {
			response.ErrForbidden(c, "无权查看")
			return
		}
		response.ErrInternal(c, "获取成长详情失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentGrowthLike(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	data, err := h.parentPortalSvc.LikeGrowth(c.Request.Context(), pc, id)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) || errors.Is(err, service.ErrParentForbidden) {
			response.ErrNotFound(c, "成长记录不存在")
			return
		}
		response.ErrInternal(c, "点赞失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentGrowthUnlike(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	data, err := h.parentPortalSvc.UnlikeGrowth(c.Request.Context(), pc, id)
	if err != nil {
		response.ErrInternal(c, "取消点赞失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentGrowthComments(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	data, err := h.parentPortalSvc.ListGrowthComments(c.Request.Context(), pc, id)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) || errors.Is(err, service.ErrParentForbidden) {
			response.ErrNotFound(c, "成长记录不存在")
			return
		}
		response.ErrInternal(c, "获取评论失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentCreateGrowthComment(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	var req struct {
		Content  string  `json:"content" binding:"required"`
		ParentID *uint64 `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "评论内容不能为空")
		return
	}
	data, err := h.parentPortalSvc.CreateGrowthComment(c.Request.Context(), pc, id, req.Content, req.ParentID)
	if err != nil {
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "评论内容不能为空")
			return
		}
		if errors.Is(err, service.ErrParentNotFound) || errors.Is(err, service.ErrParentForbidden) {
			response.ErrNotFound(c, "成长记录不存在")
			return
		}
		response.ErrInternal(c, "发表评论失败")
		return
	}
	response.Created(c, data)
}
func (h *Handler) ParentGrowthAlbum(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	year, _ := strconv.Atoi(c.DefaultQuery("year", "0"))
	month, _ := strconv.Atoi(c.DefaultQuery("month", "0"))
	data, err := h.parentPortalSvc.ListGrowthMedia(c.Request.Context(), pc, year, month)
	if err != nil {
		response.ErrInternal(c, "获取相册失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentGrowthReport(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	sidStr := c.Query("student_id")
	var sid uint64
	if sidStr != "" {
		sid, _ = strconv.ParseUint(sidStr, 10, 64)
	}
	data, err := h.parentPortalSvc.GetGrowthReport(c.Request.Context(), pc, sid, c.Query("type"))
	if err != nil {
		// 越权请求与参数缺失都是客户端可纠正的错误，不能打成「服务器错误」
		if errors.Is(err, service.ErrParentForbidden) {
			response.ErrNotFound(c, "成长报告不存在")
			return
		}
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrBadRequest(c, 40046, "请指定学生")
			return
		}
		response.ErrInternal(c, "获取成长报告失败")
		return
	}
	response.OK(c, data)
}

// ========== Pickup ==========

func (h *Handler) ParentPickupQRCode(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	data, err := h.parentPortalSvc.GetQRCode(c.Request.Context(), pc)
	if err != nil {
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "请先指定「今日接走人」")
			return
		}
		if errors.Is(err, service.ErrParentForbidden) {
			response.ErrForbidden(c, "所选接走人授权已失效，请重新指定")
			return
		}
		response.ErrInternal(c, "获取接送二维码失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentPickupRecords(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	data, err := h.parentPortalSvc.ListPickupRecords(c.Request.Context(), pc, page, ps)
	if err != nil {
		response.ErrInternal(c, "获取接送记录失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentAuthorizedPersons(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	data, err := h.parentPortalSvc.ListAuthorizedPersons(c.Request.Context(), pc, page, ps)
	if err != nil {
		response.ErrInternal(c, "获取授权接送人失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentCreateAuthorizedPerson(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	var req service.CreateAuthorizedPersonInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.parentPortalSvc.CreateAuthorizedPerson(c.Request.Context(), pc, req)
	if err != nil {
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "参数无效")
			return
		}
		if errors.Is(err, service.ErrParentForbidden) {
			response.ErrForbidden(c, "无权为该学生添加授权")
			return
		}
		response.ErrInternal(c, "添加授权接送人失败")
		return
	}
	response.Created(c, data)
}
func (h *Handler) ParentUpdateAuthorizedPerson(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	var req service.UpdateAuthorizedPersonInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.parentPortalSvc.UpdateAuthorizedPerson(c.Request.Context(), pc, id, req)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "授权记录不存在")
			return
		}
		response.ErrInternal(c, "编辑授权接送人失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentDeleteAuthorizedPerson(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	if err := h.parentPortalSvc.DeleteAuthorizedPerson(c.Request.Context(), pc, id); err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "授权记录不存在")
			return
		}
		response.ErrInternal(c, "删除授权接送人失败")
		return
	}
	response.OK(c, gin.H{})
}

// ========== Tasks ==========

func (h *Handler) ParentListTasks(c *gin.Context) {
	pc, ok := h.parentTaskCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	result, err := h.parentPortalSvc.ListTasks(c.Request.Context(), pc, page, ps)
	if err != nil {
		response.ErrInternal(c, "获取任务列表失败")
		return
	}
	response.OK(c, result)
}
func (h *Handler) ParentGetTask(c *gin.Context) {
	pc, ok := h.parentTaskCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	data, err := h.parentPortalSvc.GetTask(c.Request.Context(), pc, id)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "任务不存在")
			return
		}
		response.ErrInternal(c, "获取任务失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentCheckinTask(c *gin.Context) {
	pc, ok := h.parentTaskCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	var req struct {
		Content     string `json:"content"`
		MediaFileID uint64 `json:"media_file_id"`
	}
	_ = c.ShouldBindJSON(&req)
	data, err := h.parentPortalSvc.CheckinTask(c.Request.Context(), pc, id, req.Content, req.MediaFileID)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "任务不存在")
			return
		}
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "当前周期已提交或参数不合法")
			return
		}
		response.ErrInternal(c, "打卡失败")
		return
	}
	response.Created(c, data)
}
func (h *Handler) ParentCheckinHistory(c *gin.Context) {
	pc, ok := h.parentTaskCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	page, ps := ppPage(c)
	data, err := h.parentPortalSvc.CheckinHistory(c.Request.Context(), pc, id, page, ps)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "任务不存在")
			return
		}
		response.ErrInternal(c, "获取打卡历史失败")
		return
	}
	response.OK(c, data)
}

// ========== Points ==========

func (h *Handler) ParentPointsOverview(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	data, err := h.parentPortalSvc.PointsOverview(c.Request.Context(), pc)
	if err != nil {
		response.ErrInternal(c, "获取积分概览失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentPointsHistory(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	result, err := h.parentPortalSvc.PointsHistory(c.Request.Context(), pc, page, ps)
	if err != nil {
		response.ErrInternal(c, "获取积分历史失败")
		return
	}
	response.OK(c, result)
}
func (h *Handler) ParentLeaderboard(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	data, err := h.parentPortalSvc.Leaderboard(c.Request.Context(), pc)
	if err != nil {
		response.ErrInternal(c, "获取排行榜失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentAchievements(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	data, err := h.parentPortalSvc.ListAchievements(c.Request.Context(), pc)
	if err != nil {
		response.ErrInternal(c, "获取成就列表失败")
		return
	}
	response.OK(c, data)
}

// ========== Class ==========

func (h *Handler) ParentClassInfo(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	if pc.Class == nil {
		response.ErrInternal(c, "未找到班级")
		return
	}
	data, err := h.parentPortalSvc.ClassInfo(c.Request.Context(), pc)
	if err != nil {
		response.ErrInternal(c, "获取班级信息失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentClassCalendar(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	start := c.Query("start")
	end := c.Query("end")
	data, err := h.parentPortalSvc.ListClassCalendar(c.Request.Context(), pc, start, end)
	if err != nil {
		response.ErrInternal(c, "获取班级日历失败")
		return
	}
	response.OK(c, data)
}

// ========== Morning Health ==========

func (h *Handler) ParentHealthRecords(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	sid, err := strconv.ParseUint(c.Query("student_id"), 10, 64)
	if err != nil || sid == 0 {
		response.ErrValidation(c, "请选择孩子")
		return
	}
	// ResolveContext 在未命中时会回退到第一个孩子，这里必须再核对一次，
	// 否则「传别人孩子 id」会看到自己第一个孩子的记录。
	if pc.CurrentStudent == nil || pc.CurrentStudent.ID != sid {
		response.ErrForbidden(c, "无权查看该孩子的晨检记录")
		return
	}
	data, err := h.parentPortalSvc.ListHealthRecords(c.Request.Context(), pc, sid)
	if err != nil {
		if errors.Is(err, service.ErrParentForbidden) {
			response.ErrForbidden(c, "无权查看该孩子的晨检记录")
			return
		}
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "请选择孩子")
			return
		}
		response.ErrInternal(c, "获取晨检记录失败")
		return
	}
	// 列表接口统一包在 list 里（与 students / queue 等一致），
	// 契约见 backend/api/api-parent/12-health.md。
	response.OK(c, gin.H{"list": data})
}

// ========== Student ==========

func (h *Handler) ParentStudentProfile(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	data, err := h.parentPortalSvc.GetStudentProfile(c.Request.Context(), pc)
	if err != nil {
		response.ErrInternal(c, "获取学生资料失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentUpdateStudentProfile(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数无效")
		return
	}
	if err := h.parentPortalSvc.UpdateStudentProfile(c.Request.Context(), pc, req); err != nil {
		response.ErrInternal(c, "更新学生资料失败")
		return
	}
	response.OK(c, gin.H{"id": pc.CurrentStudent.ID})
}
func (h *Handler) ParentListStudents(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	out := make([]gin.H, 0, len(pc.Students))
	for _, s := range pc.Students {
		cn := ""
		if s.Class != nil {
			cn = s.Class.Name
		}
		out = append(out, gin.H{"id": s.ID, "name": s.Name, "avatar": s.Avatar, "gender": s.Gender, "class_name": cn, "relation": pc.Relation, "is_current": s.ID == pc.CurrentStudent.ID})
	}
	response.OK(c, gin.H{"list": out})
}
func (h *Handler) ParentBindStudent(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	var req struct {
		InviteCode string `json:"invite_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "请输入邀请码")
		return
	}
	data, err := h.parentPortalSvc.BindStudent(c.Request.Context(), pc, req.InviteCode)
	if err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "未找到该学生，请检查邀请码")
			return
		}
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "已绑定或邀请码无效")
			return
		}
		response.ErrInternal(c, "绑定学生失败")
		return
	}
	response.Created(c, data)
}
func (h *Handler) ParentUnbindStudent(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	id, ok2 := ppID(c)
	if !ok2 {
		return
	}
	if err := h.parentPortalSvc.UnbindStudent(c.Request.Context(), pc, id); err != nil {
		if errors.Is(err, service.ErrParentNotFound) {
			response.ErrNotFound(c, "绑定记录不存在")
			return
		}
		response.ErrInternal(c, "解绑失败")
		return
	}
	response.OK(c, gin.H{})
}

// ========== Feedback ==========

func (h *Handler) ParentListFeedback(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	page, ps := ppPage(c)
	result, err := h.parentPortalSvc.ListFeedback(c.Request.Context(), pc, page, ps)
	if err != nil {
		response.ErrInternal(c, "获取反馈列表失败")
		return
	}
	response.OK(c, result)
}
func (h *Handler) ParentCreateFeedback(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	var req service.ParentFeedbackInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数无效")
		return
	}
	data, err := h.parentPortalSvc.CreateFeedback(c.Request.Context(), pc, req)
	if err != nil {
		if errors.Is(err, service.ErrParentInvalidParams) {
			response.ErrValidation(c, "内容不能为空")
			return
		}
		response.ErrInternal(c, "提交反馈失败")
		return
	}
	response.Created(c, data)
}

// ========== Settings ==========

func (h *Handler) ParentGetSettings(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	data, err := h.parentPortalSvc.GetSettings(c.Request.Context(), pc)
	if err != nil {
		response.ErrInternal(c, "获取设置失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentUpdateSettings(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, "参数无效")
		return
	}
	if err := h.parentPortalSvc.UpdateSettings(c.Request.Context(), pc, req); err != nil {
		response.ErrInternal(c, "保存设置失败")
		return
	}
	response.OK(c, gin.H{"updated_at": time.Now().UTC().Format(time.RFC3339)})
}

// ParentPickupDesignate 指定今日接走人（本次契约变更新增）。
//
// 覆盖该孩当日此前的指定，并作废当日已生成的接送凭证。
type parentDesignateReq struct {
	AuthorizationID uint64 `json:"authorization_id"`
}

func (h *Handler) ParentPickupDesignate(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	var req parentDesignateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.parentPortalSvc.DesignatePickupPerson(
		c.Request.Context(), pc, service.DesignatePickupInput{AuthorizationID: req.AuthorizationID})
	if err != nil {
		if errors.Is(err, service.ErrParentForbidden) {
			response.ErrForbidden(c, "被指定人不在有效授权范围内")
			return
		}
		response.ErrValidation(c, "请求参数无效")
		return
	}
	response.OK(c, data)
}

// ParentPickupDesignation 只读查「今日接走人」指定（本次新增，供授权人列表展示）。
func (h *Handler) ParentPickupDesignation(c *gin.Context) {
	pc, ok := h.parentCtx(c)
	if !ok {
		return
	}
	data, err := h.parentPortalSvc.GetDailyDesignation(c.Request.Context(), pc)
	if err != nil {
		response.ErrInternal(c, "获取今日接走人失败")
		return
	}
	response.OK(c, data)
}
