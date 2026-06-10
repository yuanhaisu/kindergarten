package handler

import (
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
		if id, err := strconv.ParseUint(sid, 10, 64); err == nil && id > 0 { sidPtr = &id }
	}
	pc, err := h.parentPortalSvc.ResolveContext(c.Request.Context(), middleware.GetUserID(c), middleware.GetKindergartenID(c), sidPtr)
	if err != nil { response.ErrForbidden(c, "未绑定孩子信息,请联系管理员"); return nil, false }
	return pc, true
}

func ppID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 { response.ErrValidation(c, "无效的 ID"); return 0, false }
	return id, true
}

func ppPage(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	ps, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 { page = 1 }; if ps < 1 || ps > 50 { ps = 20 }
	return page, ps
}

// ========== Home ==========

func (h *Handler) ParentHomeOverview(c *gin.Context) {
	pc, ok := h.parentCtx(c); if !ok { return }
	now := time.Now()
	wd := []string{"星期日","星期一","星期二","星期三","星期四","星期五","星期六"}[now.Weekday()]
	cn := ""; if pc.Class != nil { cn = pc.Class.Name }
	response.OK(c, gin.H{
		"date":now.Format("2006-01-02"),"weekday":wd,
		"current_student":gin.H{"id":pc.CurrentStudent.ID,"name":pc.CurrentStudent.Name,"avatar":pc.CurrentStudent.Avatar,"class_name":cn,"class_id":pc.CurrentStudent.ClassID},
		"students":[]any{},
		"today_summary":gin.H{"attendance_status":"present","attendance_label":"已入园","arrive_time":"07:52","meal_rating":3,"meal_label":"很好","nap_duration":120,"sleep_quality":3,"sleep_label":"很好","mood_rating":3,"mood_label":"很好","performance_tag":"","new_growth_count":0,"new_notice_count":0,"pending_tasks":0,"points_today":0,"leave_status":nil},
	})
}

func (h *Handler) ParentSwitchStudent(c *gin.Context) {
	pc, ok := h.parentCtx(c); if !ok { return }
	var req struct{ StudentID uint64 `json:"student_id" binding:"required"` }
	if err := c.ShouldBindJSON(&req); err != nil { response.ErrValidation(c, "请选择孩子"); return }
	for _, s := range pc.Students {
		if s.ID == req.StudentID {
			cn := ""; if s.Class != nil { cn = s.Class.Name }
			response.OK(c, gin.H{"current_student":gin.H{"id":s.ID,"name":s.Name,"avatar":s.Avatar,"class_name":cn,"class_id":s.ClassID}})
			return
		}
	}
	response.ErrNotFound(c, "未找到该孩子")
}

func (h *Handler) ParentAnnouncements(c *gin.Context) { response.OK(c, gin.H{"list":[]any{}}) }
func (h *Handler) ParentHomeFeeds(c *gin.Context) { response.OKList(c, []any{}, 0, 1, 10) }

// ========== Notices ==========

func (h *Handler) ParentListNotices(c *gin.Context) {
	_, ok := h.parentCtx(c); if !ok { return }
	page, ps := ppPage(c)
	response.OKList(c, []any{}, 0, page, ps)
}
func (h *Handler) ParentGetNotice(c *gin.Context) { response.OK(c, gin.H{}) }
func (h *Handler) ParentConfirmNotice(c *gin.Context) {
	id, _ := ppID(c)
	response.OK(c, gin.H{"id":id,"is_confirmed":true,"confirmed_at":time.Now().UTC().Format(time.RFC3339)})
}

// ========== Leaves ==========

func (h *Handler) ParentListLeaves(c *gin.Context) {
	_, ok := h.parentCtx(c); if !ok { return }
	page, ps := ppPage(c)
	response.OKList(c, []any{}, 0, page, ps)
}
func (h *Handler) ParentGetLeave(c *gin.Context) { response.OK(c, gin.H{}) }
func (h *Handler) ParentCreateLeave(c *gin.Context) { response.OK(c, gin.H{"id":1,"status":0,"status_label":"待审批","created_at":time.Now().UTC().Format(time.RFC3339)}) }
func (h *Handler) ParentUpdateLeave(c *gin.Context) { id, _ := ppID(c); response.OK(c, gin.H{"id":id}) }
func (h *Handler) ParentCancelLeave(c *gin.Context) { id, _ := ppID(c); response.OK(c, gin.H{"id":id,"status":3,"status_label":"已撤销"}) }

// ========== Growth ==========

func (h *Handler) ParentGrowthFeed(c *gin.Context) {
	pc, ok := h.parentCtx(c); if !ok { return }
	page, ps := ppPage(c)
	sid := c.Query("student_id")
	var sidPtr *uint64
	if sid != "" {
		if id, err := strconv.ParseUint(sid, 10, 64); err == nil && id > 0 { sidPtr = &id }
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
func (h *Handler) ParentGrowthDetail(c *gin.Context) { response.OK(c, gin.H{}) }
func (h *Handler) ParentGrowthLike(c *gin.Context) { response.OK(c, gin.H{"is_liked":true,"likes_count":1}) }
func (h *Handler) ParentGrowthUnlike(c *gin.Context) { response.OK(c, gin.H{"is_liked":false,"likes_count":0}) }
func (h *Handler) ParentGrowthComments(c *gin.Context) { response.OKList(c, []any{}, 0, 1, 20) }
func (h *Handler) ParentCreateGrowthComment(c *gin.Context) { response.OK(c, gin.H{"id":1,"created_at":time.Now().UTC().Format(time.RFC3339)}) }
func (h *Handler) ParentGrowthAlbum(c *gin.Context) { response.OK(c, gin.H{"year":2026,"month":3,"groups":[]any{},"total":0}) }
func (h *Handler) ParentGrowthReport(c *gin.Context) { response.OK(c, gin.H{}) }

// ========== Pickup ==========

func (h *Handler) ParentPickupQRCode(c *gin.Context) {
	pc, ok := h.parentCtx(c); if !ok { return }
	response.OK(c, gin.H{"qr_code":"","qr_image_url":"","student":gin.H{"id":pc.CurrentStudent.ID,"name":pc.CurrentStudent.Name},"expire_at":time.Now().Add(24*time.Hour).UTC().Format(time.RFC3339)})
}
func (h *Handler) ParentPickupRecords(c *gin.Context) {
	pc, ok := h.parentCtx(c); if !ok { return }
	page, ps := ppPage(c)
	data, err := h.parentPortalSvc.ListPickupRecords(c.Request.Context(), pc, page, ps)
	if err != nil {
		response.ErrInternal(c, "获取接送记录失败")
		return
	}
	response.OK(c, data)
}
func (h *Handler) ParentAuthorizedPersons(c *gin.Context) { response.OK(c, gin.H{"list":[]any{}}) }
func (h *Handler) ParentCreateAuthorizedPerson(c *gin.Context) { response.OK(c, gin.H{"id":1,"status":0,"status_label":"待审核","created_at":time.Now().UTC().Format(time.RFC3339)}) }
func (h *Handler) ParentUpdateAuthorizedPerson(c *gin.Context) { id, _ := ppID(c); response.OK(c, gin.H{"id":id}) }
func (h *Handler) ParentDeleteAuthorizedPerson(c *gin.Context) { response.OK(c, gin.H{}) }

// ========== Tasks ==========

func (h *Handler) ParentListTasks(c *gin.Context) {
	_, ok := h.parentCtx(c); if !ok { return }
	page, ps := ppPage(c)
	response.OKList(c, []any{}, 0, page, ps)
}
func (h *Handler) ParentGetTask(c *gin.Context) { response.OK(c, gin.H{}) }
func (h *Handler) ParentCheckinTask(c *gin.Context) { response.OK(c, gin.H{"id":1,"points_earned":5,"streak_days":1,"total_points":100,"message":"+5积分!","created_at":time.Now().UTC().Format(time.RFC3339)}) }
func (h *Handler) ParentCheckinHistory(c *gin.Context) { response.OKList(c, []any{}, 0, 1, 20) }

// ========== Points ==========

func (h *Handler) ParentPointsOverview(c *gin.Context) { response.OK(c, gin.H{"total_points":0,"level":1,"level_name":"新手上路","rank_in_class":0,"class_total":0,"this_week":0,"this_month":0}) }
func (h *Handler) ParentPointsHistory(c *gin.Context) {
	_, ok := h.parentCtx(c); if !ok { return }
	page, ps := ppPage(c)
	response.OKList(c, []any{}, 0, page, ps)
}
func (h *Handler) ParentLeaderboard(c *gin.Context) { response.OK(c, gin.H{"period":"weekly","my_rank":0,"my_points":0,"list":[]any{}}) }
func (h *Handler) ParentAchievements(c *gin.Context) { response.OK(c, gin.H{"earned":[]any{},"not_earned":[]any{},"earned_count":0,"total_count":0}) }

// ========== Class ==========

func (h *Handler) ParentClassInfo(c *gin.Context) {
	pc, ok := h.parentCtx(c); if !ok { return }
	if pc.Class == nil { response.ErrInternal(c, "未找到班级"); return }
	response.OK(c, gin.H{"id":pc.Class.ID,"name":pc.Class.Name,"grade":pc.Class.Grade,"academic_year":"","student_count":0,"teachers":[]any{}})
}
func (h *Handler) ParentClassCalendar(c *gin.Context) { response.OK(c, gin.H{"list":[]any{},"total":0}) }

// ========== Student ==========

func (h *Handler) ParentStudentProfile(c *gin.Context) {
	pc, ok := h.parentCtx(c); if !ok { return }
	cn := ""; if pc.Class != nil { cn = pc.Class.Name }
	response.OK(c, gin.H{"id":pc.CurrentStudent.ID,"name":pc.CurrentStudent.Name,"avatar":pc.CurrentStudent.Avatar,"gender":pc.CurrentStudent.Gender,"birthday":pc.CurrentStudent.Birthday,"age":"","student_no":pc.CurrentStudent.StudentNo,"health_note":pc.CurrentStudent.HealthNote,"class_name":cn,"enroll_date":pc.CurrentStudent.EnrollDate,"parents":[]any{}})
}
func (h *Handler) ParentUpdateStudentProfile(c *gin.Context) { response.OK(c, gin.H{"id":1}) }
func (h *Handler) ParentListStudents(c *gin.Context) {
	pc, ok := h.parentCtx(c); if !ok { return }
	out := make([]gin.H, 0, len(pc.Students))
	for _, s := range pc.Students {
		cn := ""; if s.Class != nil { cn = s.Class.Name }
		out = append(out, gin.H{"id":s.ID,"name":s.Name,"avatar":s.Avatar,"gender":s.Gender,"class_name":cn,"relation":pc.Relation,"is_current":s.ID==pc.CurrentStudent.ID})
	}
	response.OK(c, gin.H{"list":out})
}
func (h *Handler) ParentBindStudent(c *gin.Context) { response.ErrInternal(c, "绑定功能开发中") }
func (h *Handler) ParentUnbindStudent(c *gin.Context) { response.ErrInternal(c, "解绑功能开发中") }

// ========== Feedback ==========

func (h *Handler) ParentListFeedback(c *gin.Context) { response.OKList(c, []any{}, 0, 1, 20) }
func (h *Handler) ParentCreateFeedback(c *gin.Context) { response.OK(c, gin.H{"id":1,"status":0,"status_label":"已提交","created_at":time.Now().UTC().Format(time.RFC3339)}) }

// ========== Settings ==========

func (h *Handler) ParentGetSettings(c *gin.Context) {
	response.OK(c, gin.H{
		"notification":gin.H{"notice_push":true,"leave_push":true,"pickup_push":true,"growth_push":true,"task_push":true,"announcement_push":true},
		"privacy":gin.H{"show_phone_to_teacher":true,"allow_growth_share":true},
	})
}
func (h *Handler) ParentUpdateSettings(c *gin.Context) { response.OK(c, gin.H{"updated_at":time.Now().UTC().Format(time.RFC3339)}) }
