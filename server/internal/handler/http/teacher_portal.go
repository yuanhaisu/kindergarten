package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/log"
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
	case errors.Is(err, service.ErrTeacherNoticeState):
		response.ErrBadRequest(c, 40012, "当前通知状态不允许此操作")
	// 与管理端 writeTaskError 同一句：三个创建入口（teacher/admin/activity）
	// 未配置班级罐时必须给同一条引导，不能一端一句。
	case errors.Is(err, service.ErrTaskStarJarMissing):
		response.ErrValidation(c, "请先配置班级星星罐")
	case errors.Is(err, service.ErrTeacherForbidden):
		response.ErrForbidden(c, "无权限访问该数据")
	case errors.Is(err, service.ErrTeacherDuplicate):
		response.ErrBadRequest(c, 40010, "该学生今日已有同类型接送记录")
	case errors.Is(err, service.ErrTeacherPickupOrder):
		response.ErrValidation(c, "请先登记入园再登记接走")
	case errors.Is(err, service.ErrTeacherInvalidQRCode):
		response.ErrValidation(c, "二维码无效或已过期")
	case errors.Is(err, service.ErrTeacherConsecutiveDirection):
		response.ErrValidation(c, "托管序列必须交替：不可连续送入或连续接走")
	case errors.Is(err, service.ErrTeacherRemarkRequired):
		response.ErrValidation(c, "重复接走或提前接走需填写理由")
	case errors.Is(err, service.ErrTeacherAuthorizationInvalid):
		response.ErrValidation(c, "接走人不在有效授权范围内")
	case errors.Is(err, service.ErrTeacherOverrideNameRequired):
		response.ErrValidation(c, "标记非授权接送时须填写来接者姓名")
	case errors.Is(err, service.ErrTaskBatchNotAllowed):
		response.ErrValidation(c, "一次性任务与个体任务需逐条回应")
	case errors.Is(err, service.ErrTaskBatchAllFailed):
		response.ErrInternal(c, "批量回应失败，提交仍待回应，请重试")
	case errors.Is(err, service.ErrTaskRespondConflict):
		response.ErrConflict(c, "该提交已被其他回应关闭，请刷新后重试")
	case errors.Is(err, service.ErrTaskSupplementForbidden):
		response.ErrValidation(c, "该提交尚未回应，请先完成主回应")
	case errors.Is(err, service.ErrTaskSupplementRitual):
		response.ErrValidation(c, "补充回应仅支持贴纸或语音")
	case errors.Is(err, service.ErrTaskRitualPayload):
		response.ErrValidation(c, "贴纸需要选择具体贴纸，语音需要先录制并上传")
	case errors.Is(err, service.ErrTaskPublicNotAllowed):
		response.ErrValidation(c, "该任务未开启公开、是个体任务，或提交者已不在本班，不能批准公开")
	case errors.Is(err, service.ErrTaskPublicScopeUnconfirmed):
		response.ErrValidation(c, "批准公开前必须确认内容中只有提交者孩子")
	case errors.Is(err, service.ErrTaskTitleTooLong):
		response.ErrValidation(c, "标题不能超过 200 个字符")
	case errors.Is(err, service.ErrTaskActivityDeleteForbidden):
		response.ErrValidation(c, "活动任务不能单独删除，请让管理员删除整个活动")
	case errors.Is(err, service.ErrTaskActivityEditForbidden):
		response.ErrValidation(c, "活动任务只能修改标题、说明、学习关联与公开开关，请让管理员整组编辑")
	case errors.Is(err, service.ErrTaskPublicStateConflict):
		response.ErrConflict(c, "该提交当前不在待审核状态，请让家长重新申请")
	case errors.Is(err, service.ErrTeacherRitualMediaType):
		response.ErrValidation(c, "语音回应必须使用音频文件")
	case errors.Is(err, service.ErrTaskStructuralLocked):
		response.ErrValidation(c, "任务已有提交，只能修改标题、说明、学习关联与公开开关")
	case errors.Is(err, service.ErrTaskBusy):
		response.ErrUnavailable(c, "操作冲突，请稍后重试")
	default:
		response.ErrInternal(c, fallback)
	}
}

// teacherNoticeErr 通知模块的错误出口。契约 api-teacher/04-notices.md 明确要求
// 「定时时间非法 / 发布不完整 / 附件超限 / 删除非草稿 / 撤回非已发布」返回 400，
// 而成长、任务等其它老师端契约写 422——所以只在这里把参数错误改成 400，
// 共用的 teacherPortalErr 保持 422，避免把别的接口带崩。
func teacherNoticeErr(c *gin.Context, err error, fallback string) {
	if errors.Is(err, service.ErrTeacherInvalidParams) {
		response.ErrBadRequest(c, 40014, "请求参数无效")
		return
	}
	teacherPortalErr(c, err, fallback)
}

func (h *Handler) TeacherSupplementResponse(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	taskID, ok := teacherPortalID(c)
	if !ok {
		return
	}
	checkinID, err := strconv.ParseUint(c.Param("checkin_id"), 10, 64)
	if err != nil || checkinID == 0 {
		response.ErrValidation(c, "无效的提交 ID")
		return
	}
	var req struct {
		Ritual      string `json:"ritual"`
		StickerKey  string `json:"sticker_key"`
		Content     string `json:"content"`
		MediaFileID uint64 `json:"media_file_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.SupplementResponse(c.Request.Context(), tc, taskID, checkinID, req.Ritual, req.StickerKey, req.Content, req.MediaFileID)
	if err != nil {
		teacherPortalErr(c, err, "补充回应失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherReviewTaskPublic(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	taskID, ok := teacherPortalID(c)
	if !ok {
		return
	}
	checkinID, err := strconv.ParseUint(c.Param("checkin_id"), 10, 64)
	if err != nil || checkinID == 0 {
		response.ErrValidation(c, "无效的提交 ID")
		return
	}
	var req struct {
		Approve        bool `json:"approve"`
		ScopeConfirmed bool `json:"scope_confirmed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, svcErr := h.teacherPortalSvc.ReviewPublicCheckin(c.Request.Context(), tc, taskID, checkinID, req.Approve, req.ScopeConfirmed)
	if svcErr != nil {
		teacherPortalErr(c, svcErr, "审核公开失败")
		return
	}
	response.OK(c, data)
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
		teacherNoticeErr(c, err, "获取通知列表失败")
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
		teacherNoticeErr(c, err, "发布通知失败")
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
		teacherNoticeErr(c, err, "获取通知详情失败")
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
	data, err := h.teacherPortalSvc.UpdateNotice(c.Request.Context(), tc, id, req)
	if err != nil {
		teacherNoticeErr(c, err, "更新通知失败")
		return
	}
	response.OK(c, data)
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
		teacherNoticeErr(c, err, "删除通知失败")
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
		teacherNoticeErr(c, err, "撤回通知失败")
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
		teacherNoticeErr(c, err, "获取通知已读状态失败")
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

// TeacherReviewMediaSuggestion 照片点名的建议确认/拒绝（0→1 确认 / 0→2 拒绝）。
func (h *Handler) TeacherReviewMediaSuggestion(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	mediaID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || mediaID == 0 {
		response.ErrValidation(c, "无效的媒体 ID")
		return
	}
	studentID, err := strconv.ParseUint(c.Param("sid"), 10, 64)
	if err != nil || studentID == 0 {
		response.ErrValidation(c, "无效的学生 ID")
		return
	}
	var req struct {
		Action string `json:"action"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	if req.Action != "confirm" && req.Action != "reject" {
		response.ErrValidation(c, "action 仅支持 confirm / reject")
		return
	}
	data, err := h.teacherPortalSvc.ReviewMediaSuggestion(c.Request.Context(), tc, mediaID, studentID, req.Action == "confirm")
	if err != nil {
		teacherPortalErr(c, err, "处理识别建议失败")
		return
	}
	response.OK(c, data)
}

// TeacherConfirmHighSuggestions 一键确认该记录全部高置信建议（D10 省力路径）。
func (h *Handler) TeacherConfirmHighSuggestions(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.ConfirmHighConfidenceSuggestions(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "批量确认失败")
		return
	}
	response.OK(c, data)
}

// TeacherGrowthTaggingStatus 记录照片的识别状态与建议（老师端轮询刷新用）。
func (h *Handler) TeacherGrowthTaggingStatus(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GrowthTaggingStatus(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取识别状态失败")
		return
	}
	response.OK(c, data)
}

// TeacherApplyClassRoster 大合影「一键全班」（D11）：把记录当日到场学生写为该照片的确认点名。
func (h *Handler) TeacherApplyClassRoster(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.ApplyClassRoster(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "全班点名失败")
		return
	}
	response.OK(c, data)
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
	Content  string  `json:"content"`
	ParentID *uint64 `json:"parent_id"`
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
	data, err := h.teacherPortalSvc.CreateGrowthComment(c.Request.Context(), tc, id, req.Content, req.ParentID)
	if err != nil {
		teacherPortalErr(c, err, "评论失败")
		return
	}
	response.Created(c, data)
}

// TeacherLikeGrowthComment 点赞评论
func (h *Handler) TeacherLikeGrowthComment(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.LikeGrowthComment(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "点赞失败")
		return
	}
	response.OK(c, data)
}

// TeacherUnlikeGrowthComment 取消点赞评论
func (h *Handler) TeacherUnlikeGrowthComment(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.UnlikeGrowthComment(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "取消点赞失败")
		return
	}
	response.OK(c, data)
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
	if studentID > 0 {
		sidPtr = &studentID
	}
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
	if req.IsAbnormal != nil {
		isAbnormal = *req.IsAbnormal
	}
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

// TeacherPresentStudents 当日到场学生（D11）：全班点名/大合影「全班」一键的前置名单。
// 口径：本班在读 ∧ 无已批准请假覆盖当日 ∧ 无考勤缺席；无请假/考勤数据时保守全员。
func (h *Handler) TeacherPresentStudents(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.PresentStudents(c.Request.Context(), tc, c.Query("date"))
	if err != nil {
		teacherPortalErr(c, err, "获取当日到场学生失败")
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
	// status=active|ended；缺省或其它值 = 全部（服务层收敛，Handler 不做 422）。
	data, err := h.teacherPortalSvc.ListTasksFiltered(c.Request.Context(), tc, page, pageSize, c.Query("status"))
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

// TeacherGetClassStarJar GET /teacher/class/star-jar
// 班级来自登录态（tc.ClassID），不接受 query 里的班级，避免老师读别的班配置。
// 老师端响应比管理端多 stars / progress / over_goal（C-4）：目标与星数必须来自
// 同一次 term 解析，否则学期边界会读到两套 term。
func (h *Handler) TeacherGetClassStarJar(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.taskSvc.GetClassStarJarWithProgress(c.Request.Context(), tc.KindergartenID, tc.ClassID)
	if err != nil {
		writeTaskError(c, err, "获取班级星星罐失败")
		return
	}
	response.OK(c, data)
}

// TeacherSetClassStarJar PUT /teacher/class/star-jar
func (h *Handler) TeacherSetClassStarJar(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req struct {
		Goal int `json:"goal"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.taskSvc.SetClassStarJar(c.Request.Context(), tc.KindergartenID, tc.ClassID, tc.UserID,
		req.Goal, c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		writeTaskError(c, err, "设置班级星星罐失败")
		return
	}
	response.OK(c, data)
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

func (h *Handler) TeacherGetTask(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GetTask(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取任务详情失败")
		return
	}
	response.OK(c, data)
}

// TeacherGetTaskShareCard 生成家长端任务小程序码（5.4）。
// 必须用家长端凭据生成，否则家长扫码会进老师端。
func (h *Handler) TeacherGetTaskShareCard(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherSvc.GenerateTaskShareCard(c.Request.Context(), tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		if service.IsNotFound(err) {
			response.ErrNotFound(c, "任务不存在")
			return
		}
		// 不能静默：凭据缺失、版本配置非法、微信接口失败都要在服务端留痕，否则线上只能看到 500。
		log.L(c.Request.Context()).Error("teacher task share-card failed", "task", id, "err", err)
		if errors.Is(err, auth.ErrInvalidCodeEnvVersion) {
			response.ErrInternal(c, "小程序码版本配置非法：wechat.code_env_version 只能是 release/trial/develop")
			return
		}
		response.ErrInternal(c, "生成家长端小程序码失败，请检查家长端微信配置")
		return
	}
	response.OK(c, data)
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

func (h *Handler) TeacherTaskFeedbackSummary(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.TaskFeedbackSummary(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取反馈汇总失败")
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
	data, err := h.teacherPortalSvc.TaskCheckins(c.Request.Context(), tc, id, c.Query("status"))
	if err != nil {
		teacherPortalErr(c, err, "获取任务打卡失败")
		return
	}
	response.OK(c, data)
}

func (h *Handler) TeacherRespondTask(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	taskID, ok := teacherPortalID(c)
	if !ok {
		return
	}
	checkinID, err := strconv.ParseUint(c.Param("checkin_id"), 10, 64)
	if err != nil || checkinID == 0 {
		response.ErrValidation(c, "无效的提交 ID")
		return
	}
	var req struct {
		Ritual      string `json:"ritual"`
		StickerKey  string `json:"sticker_key"`
		Content     string `json:"content"`
		MediaFileID uint64 `json:"media_file_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.RespondTask(c.Request.Context(), tc, taskID, checkinID, req.Ritual, req.StickerKey, req.Content, req.MediaFileID)
	if err != nil {
		teacherPortalErr(c, err, "回应任务失败")
		return
	}
	response.Created(c, data)
}

func (h *Handler) TeacherBatchRespondTask(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	taskID, ok := teacherPortalID(c)
	if !ok {
		return
	}
	var req struct {
		Ritual      string `json:"ritual"`
		StickerKey  string `json:"sticker_key"`
		Content     string `json:"content"`
		MediaFileID uint64 `json:"media_file_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.BatchRespondTask(c.Request.Context(), tc, taskID, req.Ritual, req.StickerKey, req.Content, req.MediaFileID)
	if err != nil {
		teacherPortalErr(c, err, "批量回应任务失败")
		return
	}
	// id=0 表示本周期没有待回应、未创建任何回应：按空操作返回 200，而不是 201 Created。
	if id, _ := data["id"].(uint64); id == 0 {
		response.OK(c, data)
		return
	}
	response.Created(c, data)
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
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok2 := teacherPortalID(c)
	if !ok2 {
		return
	}
	data, err := h.teacherPortalSvc.LikeGrowth(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "点赞失败")
		return
	}
	response.OK(c, data)
}

// TeacherGrowthUnlike 取消点赞
func (h *Handler) TeacherGrowthUnlike(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok2 := teacherPortalID(c)
	if !ok2 {
		return
	}
	data, err := h.teacherPortalSvc.UnlikeGrowth(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "取消点赞失败")
		return
	}
	response.OK(c, data)
}

// TeacherPickupAuthApprove 审核通过授权接送人。
func (h *Handler) TeacherPickupAuthApprove(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok2 := teacherPortalID(c)
	if !ok2 {
		return
	}
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
	if !ok {
		return
	}
	id, ok2 := teacherPortalID(c)
	if !ok2 {
		return
	}
	data, err := h.teacherPortalSvc.AuditPickupAuth(c.Request.Context(), tc, id, false, "")
	if err != nil {
		teacherPortalErr(c, err, "审核授权失败")
		return
	}
	response.OK(c, data)
}

// TeacherPickupNotifyOverdue 一键通知滞留儿童家长（本次契约变更新增）。
//
// 限本班学生；非滞留状态拒绝。重复触发每次留痕。
type teacherNotifyOverdueReq struct {
	StudentID uint64 `json:"student_id"`
}

func (h *Handler) TeacherPickupNotifyOverdue(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req teacherNotifyOverdueReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.NotifyOverdueParents(c.Request.Context(), tc, req.StudentID)
	if err != nil {
		teacherPortalErr(c, err, "一键通知失败")
		return
	}
	response.OK(c, data)
}

// TeacherPickupStudentAuthPersons 按学生查有效授权接送人（手动登记选人用）
func (h *Handler) TeacherPickupStudentAuthPersons(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	studentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || studentID == 0 {
		response.ErrValidation(c, "无效的学生 ID")
		return
	}
	data, err := h.teacherPortalSvc.StudentAuthorizedPersons(c.Request.Context(), tc, studentID)
	if err != nil {
		teacherPortalErr(c, err, "获取授权接送人失败")
		return
	}
	response.OK(c, data)
}

// TeacherAlbumStats GET /teacher/albums/stats 相册统计卡
func (h *Handler) TeacherAlbumStats(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.AlbumStats(c.Request.Context(), tc)
	if err != nil {
		teacherPortalErr(c, err, "获取相册统计失败")
		return
	}
	response.OK(c, data)
}

// TeacherListAlbums GET /teacher/albums?category&page&page_size
func (h *Handler) TeacherListAlbums(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	page, pageSize := teacherPortalPage(c)
	data, err := h.teacherPortalSvc.ListAlbums(c.Request.Context(), tc, c.Query("category"), page, pageSize)
	if err != nil {
		teacherPortalErr(c, err, "获取相册列表失败")
		return
	}
	response.OK(c, data)
}

// TeacherGetAlbum GET /teacher/albums/:id 相册详情（含照片列表）
func (h *Handler) TeacherGetAlbum(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	id, ok := teacherPortalID(c)
	if !ok {
		return
	}
	data, err := h.teacherPortalSvc.GetAlbum(c.Request.Context(), tc, id)
	if err != nil {
		teacherPortalErr(c, err, "获取相册详情失败")
		return
	}
	response.OK(c, data)
}

// TeacherCreateAlbum POST /teacher/albums 创建相册（关联已上传照片）
func (h *Handler) TeacherCreateAlbum(c *gin.Context) {
	tc, ok := h.teacherPortalContext(c)
	if !ok {
		return
	}
	var req service.TeacherAlbumInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrValidation(c, err.Error())
		return
	}
	data, err := h.teacherPortalSvc.CreateAlbum(c.Request.Context(), tc, req)
	if err != nil {
		teacherPortalErr(c, err, "创建相册失败")
		return
	}
	response.Created(c, data)
}
