package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/response"
)

func (h *Handler) teacherPortalContext(c *gin.Context) (*service.TeacherPortalContext, bool) {
	tc, err := h.teacherPortalSvc.ResolveContext(c.Request.Context(), middleware.GetUserID(c), middleware.GetKindergartenID(c))
	if err != nil {
		if errors.Is(err, service.ErrTeacherContext) {
			response.ErrForbidden(c, "未绑定老师班级，请联系管理员")
			return nil, false
		}
		response.ErrInternal(c, "获取老师身份失败")
		return nil, false
	}
	return tc, true
}

func teacherPortalID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		response.ErrValidation(c, "无效的 ID")
		return 0, false
	}
	return id, true
}

func teacherPortalPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	return page, pageSize
}

func teacherPortalErr(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, service.ErrTeacherNotFound):
		response.ErrNotFound(c, "数据不存在或无权访问")
	case errors.Is(err, service.ErrTeacherInvalidParams):
		response.ErrValidation(c, "请求参数无效")
	case errors.Is(err, service.ErrTeacherForbidden):
		response.ErrForbidden(c, "无权限访问该数据")
	case errors.Is(err, service.ErrTeacherDuplicate):
		response.ErrBadRequest(c, 40010, "该学生今日已有同类型接送记录")
	case errors.Is(err, service.ErrTeacherPickupOrder):
		response.ErrValidation(c, "请先登记入园再登记接走")
	case errors.Is(err, service.ErrTeacherInvalidQRCode):
		response.ErrValidation(c, "二维码无效或已过期")
	default:
		response.ErrInternal(c, fallback)
	}
}

func (h *Handler) TeacherHomeOverview(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.HomeOverview(c.Request.Context(), tc)
	if err != nil {
		teacherPortalErr(c, err, "获取首页概览失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherHomeTodos(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	page, pageSize := teacherPortalPage(c)
	data, err := h.teacherPortalSvc.HomeTodos(c.Request.Context(), tc, page, pageSize)
	if err != nil {
		teacherPortalErr(c, err, "获取待办失败")
		return
	}
	response.OK(c, data)
}

type teacherTodoActionReq struct {
	Action string `json:"action" binding:"required"`
	Remark string `json:"remark"`
}

func (h *Handler) TeacherTodoAction(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req teacherTodoActionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.TodoAction(c.Request.Context(), tc, id, req.Action, req.Remark)
	if err != nil {
		teacherPortalErr(c, err, "处理待办失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherListNotices(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var status *int8
	if raw := c.Query("status"); raw != "" {
		v, err := strconv.ParseInt(raw, 10, 8)
		if err != nil {
			response.ErrValidation(c, "通知状态无效")
			return
		}
		sv := int8(v)
		status = &sv
	}
	page, pageSize := teacherPortalPage(c)
	data, err := h.teacherPortalSvc.ListNotices(c.Request.Context(), tc, status, c.Query("category"), page, pageSize)
	if err != nil {
		teacherPortalErr(c, err, "获取通知列表失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherCreateNotice(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req service.TeacherNoticeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreateNotice(c.Request.Context(), tc, req)
	if err != nil {
		teacherPortalErr(c, err, "发布通知失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherGetNotice(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GetNotice(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取通知详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherUpdateNotice(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req service.TeacherNoticeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	if err := h.teacherPortalSvc.UpdateNotice(c.Request.Context(), tc, id, req); err != nil {
		teacherPortalErr(c, err, "更新通知失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherDeleteNotice(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	if err := h.teacherPortalSvc.DeleteNotice(c.Request.Context(), tc, id); err != nil {
		teacherPortalErr(c, err, "删除通知失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherRevokeNotice(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	if err := h.teacherPortalSvc.RevokeNotice(c.Request.Context(), tc, id); err != nil {
		teacherPortalErr(c, err, "撤回通知失败")
		return
	}
	response.OK(c, map[string]any{"id": id, "status": domain.NoticeStatusRevoked})
}


func (h *Handler) TeacherNoticeReadStatus(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.NoticeReadStatus(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取通知已读状态失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherListNoticeTemplates(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.ListNoticeTemplates(c.Request.Context(), tc)
	if err != nil {
		teacherPortalErr(c, err, "获取通知模板失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherCreateNoticeTemplate(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req service.TeacherNoticeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreateNoticeTemplate(c.Request.Context(), tc, req)
	if err != nil {
		teacherPortalErr(c, err, "创建通知模板失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherUpdateNoticeTemplate(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req service.TeacherNoticeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	if err := h.teacherPortalSvc.UpdateNoticeTemplate(c.Request.Context(), tc, id, req); err != nil {
		teacherPortalErr(c, err, "更新通知模板失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherDeleteNoticeTemplate(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	if err := h.teacherPortalSvc.DeleteNoticeTemplate(c.Request.Context(), tc, id); err != nil {
		teacherPortalErr(c, err, "删除通知模板失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherListLeaves(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	page, pageSize := teacherPortalPage(c)
	data, err := h.teacherPortalSvc.ListLeaves(c.Request.Context(), tc, c.Query("status"), page, pageSize)
	if err != nil {
		teacherPortalErr(c, err, "获取请假列表失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherLeaveStats(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.LeaveStats(c.Request.Context(), tc)
	if err != nil {
		teacherPortalErr(c, err, "获取请假统计失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherGetLeave(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GetLeave(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取请假详情失败")
		return
	}
	response.OK(c, data)
}

type teacherAuditLeaveReq struct {
	Remark string `json:"remark"`
	Reason string `json:"reason"`
}

func (h *Handler) TeacherApproveLeave(c *gin.Context) {
	h.teacherAuditLeave(c, true)
}

func (h *Handler) TeacherRejectLeave(c *gin.Context) {
	h.teacherAuditLeave(c, false)
}

func (h *Handler) teacherAuditLeave(c *gin.Context, approve bool) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req teacherAuditLeaveReq
	_ = c.ShouldBindJSON(&req)
	remark := req.Remark
	if remark == "" {
		remark = req.Reason
	}
	if err := h.teacherPortalSvc.AuditLeave(c.Request.Context(), tc, id, approve, remark); err != nil {
		teacherPortalErr(c, err, "审批请假失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherListGrowthLogs(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	page, pageSize := teacherPortalPage(c)
	data, err := h.teacherPortalSvc.ListGrowthLogs(c.Request.Context(), tc, page, pageSize)
	if err != nil {
		teacherPortalErr(c, err, "获取成长记录失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherCreateGrowthLog(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req service.TeacherGrowthInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreateGrowthLog(c.Request.Context(), tc, req)
	if err != nil {
		teacherPortalErr(c, err, "创建成长记录失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherGetGrowthLog(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GetGrowthLog(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取成长记录详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherUpdateGrowthLog(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req service.TeacherGrowthInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	if err := h.teacherPortalSvc.UpdateGrowthLog(c.Request.Context(), tc, id, req); err != nil {
		teacherPortalErr(c, err, "更新成长记录失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherDeleteGrowthLog(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	if err := h.teacherPortalSvc.DeleteGrowthLog(c.Request.Context(), tc, id); err != nil {
		teacherPortalErr(c, err, "删除成长记录失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherGrowthComments(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GrowthComments(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取评论失败")
		return
	}
	response.OK(c, data)
}

type teacherCommentReq struct {
	Content string `json:"content"`
}

func (h *Handler) TeacherCreateGrowthComment(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req teacherCommentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreateGrowthComment(c.Request.Context(), tc, id, req.Content)
	if err != nil {
		teacherPortalErr(c, err, "评论失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherGrowthLikes(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GrowthLikes(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取点赞失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherGrowthRelatedStudents(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GrowthRelatedStudents(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取关联学生失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherGrowthMediaUpload(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req service.TeacherMediaInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreateGrowthMedia(c.Request.Context(), tc, req)
	if err != nil {
		teacherPortalErr(c, err, "保存媒体失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherPickupOverview(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.PickupOverview(c.Request.Context(), tc)
	if err != nil {
		teacherPortalErr(c, err, "获取接送概览失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherPickupStudents(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	status := c.Query("status")
	data, err := h.teacherPortalSvc.PickupStudents(c.Request.Context(), tc, status)
	if err != nil {
		teacherPortalErr(c, err, "获取接送学生失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherPickupScan(c *gin.Context) {
	h.teacherPickupCreate(c, domain.PickupMethodScan)
}

func (h *Handler) TeacherPickupManual(c *gin.Context) {
	h.teacherPickupCreate(c, domain.PickupMethodManual)
}

func (h *Handler) teacherPickupCreate(c *gin.Context, method domain.PickupMethod) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req service.TeacherPickupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreatePickupRecord(c.Request.Context(), tc, req, method)
	if err != nil {
		teacherPortalErr(c, err, "登记接送失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherPickupRecords(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	page, pageSize := teacherPortalPage(c)
	date := c.Query("date")
	studentID, _ := strconv.ParseUint(c.Query("student_id"), 10, 64)
	var sidPtr *uint64
	if studentID > 0 { sidPtr = &studentID }
	data, err := h.teacherPortalSvc.PickupRecords(c.Request.Context(), tc, page, pageSize, date, sidPtr)
	if err != nil {
		teacherPortalErr(c, err, "获取接送记录失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherPickupRecord(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.PickupRecord(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取接送记录详情失败")
		return
	}
	response.OK(c, data)
}

type teacherAbnormalReq struct {
	IsAbnormal     *bool  `json:"is_abnormal"`
	AbnormalReason string `json:"abnormal_reason"`
}

func (h *Handler) TeacherPickupAbnormal(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req teacherAbnormalReq
	_ = c.ShouldBindJSON(&req)
	isAbnormal := true
	if req.IsAbnormal != nil { isAbnormal = *req.IsAbnormal }
	data, err := h.teacherPortalSvc.MarkPickupAbnormal(c.Request.Context(), tc, id, isAbnormal, req.AbnormalReason)
	if err != nil {
		teacherPortalErr(c, err, "标记异常失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherCalendarEvents(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.CalendarEvents(c.Request.Context(), tc, c.Query("start"), c.Query("end"))
	if err != nil {
		teacherPortalErr(c, err, "获取日历事件失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherGetCalendarEvent(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GetCalendarEvent(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取日历事件详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherCalendarEventsByDate(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.CalendarEventsByDate(c.Request.Context(), tc, c.Param("date"))
	if err != nil {
		teacherPortalErr(c, err, "获取当日事件失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherCreateCalendarEvent(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req service.TeacherCalendarInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreateCalendarEvent(c.Request.Context(), tc, req)
	if err != nil {
		teacherPortalErr(c, err, "创建日历事件失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherUpdateCalendarEvent(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req service.TeacherCalendarInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	if err := h.teacherPortalSvc.UpdateCalendarEvent(c.Request.Context(), tc, id, req); err != nil {
		teacherPortalErr(c, err, "更新日历事件失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherDeleteCalendarEvent(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	if err := h.teacherPortalSvc.DeleteCalendarEvent(c.Request.Context(), tc, id); err != nil {
		teacherPortalErr(c, err, "删除日历事件失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherCalendarDots(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.CalendarDots(c.Request.Context(), tc, c.Query("start"), c.Query("end"))
	if err != nil {
		teacherPortalErr(c, err, "获取日历标记失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherClassInfo(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.ClassInfo(c.Request.Context(), tc)
	if err != nil {
		teacherPortalErr(c, err, "获取班级信息失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherClassList(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.ListTeacherClasses(c.Request.Context(), tc)
	if err != nil {
		teacherPortalErr(c, err, "获取班级列表失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherClassStudents(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.ClassStudents(c.Request.Context(), tc)
	if err != nil {
		teacherPortalErr(c, err, "获取班级学生失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherClassStudent(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.ClassStudent(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取学生详情失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherListTasks(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	page, pageSize := teacherPortalPage(c)
	data, err := h.teacherPortalSvc.ListTasks(c.Request.Context(), tc, page, pageSize)
	if err != nil {
		teacherPortalErr(c, err, "获取亲子任务失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherCreateTask(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req service.TeacherTaskInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreateTask(c.Request.Context(), tc, req)
	if err != nil {
		teacherPortalErr(c, err, "创建亲子任务失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherUpdateTask(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req service.TeacherTaskInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	if err := h.teacherPortalSvc.UpdateTask(c.Request.Context(), tc, id, req); err != nil {
		teacherPortalErr(c, err, "更新亲子任务失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherDeleteTask(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	if err := h.teacherPortalSvc.DeleteTask(c.Request.Context(), tc, id); err != nil {
		teacherPortalErr(c, err, "删除亲子任务失败")
		return
	}
	response.OK(c, map[string]any{"id": id})
}

func (h *Handler) TeacherTaskStats(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.TaskStats(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取任务统计失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherTaskCheckins(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.TaskCheckins(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取任务打卡失败")
		return
	}
	response.OK(c, data)
}

// TeacherListMessages 老师端消息聚合列表（通知 + 审批 + 系统消息）
func (h *Handler) TeacherListMessages(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	page, pageSize := teacherPortalPage(c)
	data, err := h.teacherPortalSvc.ListTeacherMessages(c.Request.Context(), tc, page, pageSize)
	if err != nil {
		teacherPortalErr(c, err, "获取消息列表失败")
		return
	}
	response.OK(c, data)
}

// TeacherGrowthLike 点赞成长记录
func (h *Handler) TeacherGrowthLike(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c); if !ok { return }
	id, ok2 := teacherPortalID(c); if !ok2 { return }
	data, err := h.teacherPortalSvc.LikeGrowth(c.Request.Context(), tc, id)
	if err != nil { teacherPortalErr(c, err, "点赞失败"); return }
	response.OK(c, data)
}

// TeacherGrowthUnlike 取消点赞
func (h *Handler) TeacherGrowthUnlike(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c); if !ok { return }
	id, ok2 := teacherPortalID(c); if !ok2 { return }
	data, err := h.teacherPortalSvc.UnlikeGrowth(c.Request.Context(), tc, id)
	if err != nil { teacherPortalErr(c, err, "取消点赞失败"); return }
	response.OK(c, data)
}

// TeacherPickupAuthApprove 审核通过授权接送人。
func (h *Handler) TeacherPickupAuthApprove(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok { return }
	id, ok2 := teacherPortalID(c); if !ok2 { return }
	data, err := h.teacherPortalSvc.AuditPickupAuth(c.Request.Context(), tc, id, true, "")
	if err != nil {
		teacherPortalErr(c, err, "审核授权失败")
		return
	}
	response.OK(c, data)
}

// TeacherPickupAuthReject 拒绝授权接送人。
func (h *Handler) TeacherPickupAuthReject(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok { return }
	id, ok2 := teacherPortalID(c); if !ok2 { return }
	data, err := h.teacherPortalSvc.AuditPickupAuth(c.Request.Context(), tc, id, false, "")
	if err != nil {
		teacherPortalErr(c, err, "审核授权失败")
		return
	}
	response.OK(c, data)
}
