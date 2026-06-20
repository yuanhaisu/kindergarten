package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/pagination"
)

type TeacherPortalService struct {
	db       *gorm.DB
	calendar *repository.CalendarRepo
}

func NewTeacherPortalService(db *gorm.DB, calendar *repository.CalendarRepo) *TeacherPortalService {
	return &TeacherPortalService{db: db, calendar: calendar}
}

type TeacherPortalContext struct {
	UserID           uint64 `json:"user_id"`
	KindergartenID   uint64 `json:"kindergarten_id"`
	ClassID          uint64 `json:"class_id"`
	ClassName        string `json:"class_name"`
	Title            string `json:"title"`
	TitleLabel       string `json:"title_label"`
	TeacherName      string `json:"teacher_name"`
	TeacherAvatar    string `json:"teacher_avatar"`
	KindergartenName string `json:"kindergarten_name"`
}

var (
	ErrTeacherContext        = errors.New("TEACHER_CONTEXT_NOT_FOUND")
	ErrTeacherForbidden      = errors.New("TEACHER_FORBIDDEN")
	ErrTeacherNotFound       = errors.New("TEACHER_RESOURCE_NOT_FOUND")
	ErrTeacherInvalidParams  = errors.New("TEACHER_INVALID_PARAMS")
	ErrTeacherDuplicate      = errors.New("PICKUP_DUPLICATE")
	ErrTeacherPickupOrder    = errors.New("PICKUP_ORDER_INVALID")
	ErrTeacherInvalidQRCode  = errors.New("INVALID_QR_CODE")
)

func (s *TeacherPortalService) ResolveContext(ctx context.Context, userID, kindergartenID uint64) (*TeacherPortalContext, error) {
	if userID == 0 {
		return nil, ErrTeacherContext
	}
	q := s.db.WithContext(ctx).Where("user_id = ? AND role = ?", userID, "teacher")
	if kindergartenID > 0 {
		q = q.Where("kindergarten_id = ?", kindergartenID)
	}
	var role domain.UserRole
	if err := q.Order("id ASC").First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherContext
		}
		return nil, err
	}
	if role.KindergartenID == nil || *role.KindergartenID == 0 {
		return nil, ErrTeacherContext
	}

	var user domain.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}

	out := &TeacherPortalContext{
		UserID: userID, KindergartenID: *role.KindergartenID,
		TeacherName: strings.TrimSpace(user.Nickname), TeacherAvatar: user.Avatar,
		Title: role.Title, TitleLabel: role.TitleLabel,
	}
	if out.TeacherName == "" {
		out.TeacherName = "老师"
	}

	var profile domain.TeacherProfile
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND kindergarten_id = ?", userID, out.KindergartenID).
		First(&profile).Error; err == nil && strings.TrimSpace(profile.RealName) != "" {
		out.TeacherName = profile.RealName
	}

	var kg domain.Kindergarten
	if err := s.db.WithContext(ctx).First(&kg, out.KindergartenID).Error; err == nil {
		out.KindergartenName = kg.Name
	}

	var ct domain.ClassTeacher
	if err := s.db.WithContext(ctx).
		Preload("Class").
		Joins("INNER JOIN classes c ON c.id = class_teachers.class_id AND c.kindergarten_id = ?", out.KindergartenID).
		Where("class_teachers.user_id = ? AND class_teachers.status = ?", userID, 1).
		Order("class_teachers.id ASC").
		First(&ct).Error; err == nil {
		out.ClassID = ct.ClassID
		out.Title = string(ct.Title)
		out.TitleLabel = teacherPortalTitleLabel(ct.Title)
		if ct.Class != nil {
			out.ClassName = ct.Class.Name
		}
	}
	if out.ClassID == 0 && role.ClassID != nil {
		out.ClassID = *role.ClassID
		out.ClassName = role.ClassName
	}
	if out.ClassID == 0 {
		return nil, ErrTeacherContext
	}
	if out.ClassName == "" {
		var class domain.Class
		if err := s.db.WithContext(ctx).First(&class, "id = ? AND kindergarten_id = ?", out.ClassID, out.KindergartenID).Error; err == nil {
			out.ClassName = class.Name
		}
	}
	return out, nil
}

func teacherPortalTitleLabel(t domain.TeacherTitle) string {
	switch t {
	case domain.TeacherTitleHead:
		return "主班老师"
	case domain.TeacherTitleAssistant:
		return "配班老师"
	case domain.TeacherTitleNursery:
		return "保育老师"
	default:
		return "老师"
	}
}

func teacherPortalDay(t time.Time) time.Time {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	if loc == nil {
		loc = time.Local
	}
	lt := t.In(loc)
	return time.Date(lt.Year(), lt.Month(), lt.Day(), 0, 0, 0, 0, loc)
}

func teacherPortalDateString(t time.Time) string {
	return teacherPortalDay(t).Format("2006-01-02")
}

func teacherPortalStatusLabel(status int8) string {
	switch status {
	case 0:
		return "待审批"
	case 1:
		return "已同意"
	case 2:
		return "已拒绝"
	case 3:
		return "已取消"
	default:
		return "未知"
	}
}

func teacherPortalNoticeStatusLabel(status domain.NoticeStatus) string {
	switch status {
	case domain.NoticeStatusDraft:
		return "草稿"
	case domain.NoticeStatusScheduled:
		return "定时待发"
	case domain.NoticeStatusPublished:
		return "已发布"
	case domain.NoticeStatusRevoked:
		return "已撤回"
	default:
		return "未知"
	}
}

func teacherPortalNoticeCategoryLabel(category domain.NoticeCategory) string {
	switch category {
	case domain.NoticeCategoryHomework:
		return "作业"
	case domain.NoticeCategoryBring:
		return "物品"
	case domain.NoticeCategoryActivity:
		return "活动"
	case domain.NoticeCategoryHealth:
		return "健康"
	default:
		return "其他"
	}
}

func parseTeacherPortalPage(page, pageSize int) pagination.Query {
	q := pagination.Query{Page: page, PageSize: pageSize}
	q.Normalize()
	return q
}

func (s *TeacherPortalService) HomeOverview(ctx context.Context, tc *TeacherPortalContext) (map[string]any, error) {
	today := teacherPortalDay(time.Now())
	var totalStudents int64
	var present int64
	var leaveCount int64
	var pendingApprovals int64
	var unreadNotices int64

	db := s.db.WithContext(ctx)
	if err := db.Model(&domain.Student{}).Where("kindergarten_id = ? AND class_id = ? AND status = ?", tc.KindergartenID, tc.ClassID, 1).Count(&totalStudents).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&domain.Attendance{}).Where("kindergarten_id = ? AND class_id = ? AND date = ? AND status = ?", tc.KindergartenID, tc.ClassID, today, domain.AttendancePresent).Count(&present).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&domain.LeaveRequest{}).Where("kindergarten_id = ? AND class_id = ? AND start_date <= ? AND end_date >= ? AND status = ?", tc.KindergartenID, tc.ClassID, today, today, domain.LeaveStatusApproved).Count(&leaveCount).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&domain.LeaveRequest{}).Where("kindergarten_id = ? AND class_id = ? AND status = ?", tc.KindergartenID, tc.ClassID, domain.LeaveStatusPending).Count(&pendingApprovals).Error; err != nil {
		return nil, err
	}
	if err := db.Model(&domain.Notice{}).Where("kindergarten_id = ? AND class_id = ? AND status = ? AND need_confirm = ?", tc.KindergartenID, tc.ClassID, domain.NoticeStatusPublished, true).Count(&unreadNotices).Error; err != nil {
		return nil, err
	}
	rate := "--"
	if totalStudents > 0 {
		rate = fmt.Sprintf("%d%%", present*100/totalStudents)
	}
	return map[string]any{
		"date": teacherPortalDateString(today), "weekday": today.Format("Monday"),
		"teacher_name": tc.TeacherName, "teacher_avatar": tc.TeacherAvatar,
		"class_name": tc.ClassName, "student_count": totalStudents, "attendance_rate": rate,
		"class_info":        map[string]any{"id": tc.ClassID, "name": tc.ClassName, "title": tc.Title, "title_label": tc.TitleLabel},
		"attendance":        map[string]any{"present": present, "leave": leaveCount},
		"pending_approvals": pendingApprovals, "unread_notices": unreadNotices,
		"todo_count": pendingApprovals + unreadNotices,
	}, nil
}

func (s *TeacherPortalService) HomeTodos(ctx context.Context, tc *TeacherPortalContext, page, pageSize int) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	list := make([]map[string]any, 0)
	var leaves []domain.LeaveRequest
	err := s.db.WithContext(ctx).Preload("Student").Preload("Applicant").
		Where("kindergarten_id = ? AND class_id = ? AND status = ?", tc.KindergartenID, tc.ClassID, domain.LeaveStatusPending).
		Order("created_at DESC").Limit(q.PageSize).Offset(q.Offset()).Find(&leaves).Error
	if err != nil {
		return nil, err
	}
	for _, lr := range leaves {
		student := "学生"
		if lr.Student != nil {
			student = lr.Student.Name
		}
		applicant := "家长"
		if lr.Applicant != nil && lr.Applicant.Nickname != "" {
			applicant = lr.Applicant.Nickname
		}
		list = append(list, map[string]any{
			"id": lr.ID, "type": "leave_approval", "type_name": "请假审批",
			"title":        fmt.Sprintf("%s提交的请假申请", applicant),
			"description":  fmt.Sprintf("%s：%s（%s-%s）", student, lr.Reason, teacherPortalDateString(lr.StartDate), teacherPortalDateString(lr.EndDate)),
			"reference_id": lr.ID, "actions": []string{"approve", "reject"}, "priority": "high", "created_at": lr.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	var unreadNotices int64
	if err := s.db.WithContext(ctx).Model(&domain.Notice{}).Where("kindergarten_id = ? AND class_id = ? AND status = ? AND need_confirm = ?", tc.KindergartenID, tc.ClassID, domain.NoticeStatusPublished, true).Count(&unreadNotices).Error; err != nil {
		return nil, err
	}
	if unreadNotices > 0 && len(list) < q.PageSize {
		list = append(list, map[string]any{
			"id": "notice-unread-remind", "type": "notice_unread_remind", "type_name": "通知提醒",
			"title": "班级通知未读提醒", "description": fmt.Sprintf("还有 %d 条通知需要关注，可转发到微信群提醒家长查看", unreadNotices),
			"reference_id": nil, "actions": []string{"share"}, "priority": "normal", "created_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
	today := teacherPortalDay(time.Now())
	var growthToday int64
	if err := s.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).Where("kindergarten_id = ? AND class_id = ? AND date = ?", tc.KindergartenID, tc.ClassID, today).Count(&growthToday).Error; err != nil {
		return nil, err
	}
	if growthToday == 0 && len(list) < q.PageSize {
		list = append(list, map[string]any{
			"id": "growth-pending", "type": "growth_pending", "type_name": "成长记录",
			"title": "今日成长记录待提交", "description": "还未上传今日班级活动照片和记录",
			"reference_id": nil, "actions": []string{"go_record"}, "priority": "normal", "created_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
	return map[string]any{"list": list, "total": len(list), "page": q.Page, "page_size": q.PageSize}, nil
}

func (s *TeacherPortalService) TodoAction(ctx context.Context, tc *TeacherPortalContext, id uint64, action, remark string) (map[string]any, error) {
	switch action {
	case "approve", "reject":
		status := domain.LeaveStatusApproved
		msg := "已同意请假申请"
		if action == "reject" {
			status = domain.LeaveStatusRejected
			msg = "已拒绝请假申请"
			if strings.TrimSpace(remark) == "" {
				return nil, ErrTeacherInvalidParams
			}
		}
		now := time.Now()
		res := s.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
			Where("id = ? AND kindergarten_id = ? AND class_id = ? AND status = ?", id, tc.KindergartenID, tc.ClassID, domain.LeaveStatusPending).
			Updates(map[string]any{"status": status, "reviewer_id": tc.UserID, "reviewed_at": &now, "review_remark": remark})
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, ErrTeacherNotFound
		}
		return map[string]any{"id": id, "status": "done", "message": msg}, nil
	case "go_record":
		return map[string]any{"id": id, "status": "pending", "message": "请前往成长记录发布"}, nil
	default:
		return nil, ErrTeacherInvalidParams
	}
}

type TeacherNoticeInput struct {
	Category        string   `json:"category"`
	Title           string   `json:"title"`
	Content         string   `json:"content"`
	NeedConfirm     bool     `json:"need_confirm"`
	ScheduleEnabled bool     `json:"schedule_enabled"`
	ScheduledAt     string   `json:"scheduled_at"`
	Attachments     []string `json:"attachments"`
	ClassID         uint64   `json:"class_id"`
}

func (s *TeacherPortalService) ListNotices(ctx context.Context, tc *TeacherPortalContext, status *int8, category string, page, pageSize int) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	db := s.db.WithContext(ctx).Model(&domain.Notice{}).Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID)
	if status != nil {
		db = db.Where("status = ?", *status)
	}
	if strings.TrimSpace(category) != "" {
		db = db.Where("category = ?", category)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []domain.Notice
	if err := db.Order("created_at DESC").Limit(q.PageSize).Offset(q.Offset()).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	for _, n := range rows {
		var readCount int64
		_ = s.db.WithContext(ctx).Model(&domain.NoticeReadLog{}).Where("notice_id = ?", n.ID).Count(&readCount).Error
		totalCount := s.classStudentCount(ctx, tc)
		readRate := "0%"
		if totalCount > 0 {
			readRate = fmt.Sprintf("%d%%", readCount*100/totalCount)
		}
		list = append(list, map[string]any{
			"id": n.ID, "category": n.Category, "category_label": teacherPortalNoticeCategoryLabel(n.Category),
			"title": n.Title, "content": n.Content, "need_confirm": n.NeedConfirm,
			"status": n.Status, "status_label": teacherPortalNoticeStatusLabel(n.Status),
			"read_count": readCount, "total_count": totalCount, "read_rate": readRate,
			"attachments_count": 0, "published_at": formatOptionalTime(n.PublishedAt), "created_at": n.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return map[string]any{"list": list, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

func (s *TeacherPortalService) classStudentCount(ctx context.Context, tc *TeacherPortalContext) int64 {
	var n int64
	_ = s.db.WithContext(ctx).Model(&domain.Student{}).Where("kindergarten_id = ? AND class_id = ? AND status = ?", tc.KindergartenID, tc.ClassID, 1).Count(&n).Error
	return n
}

func formatOptionalTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func (s *TeacherPortalService) CreateNotice(ctx context.Context, tc *TeacherPortalContext, req TeacherNoticeInput) (map[string]any, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Content) == "" {
		return nil, ErrTeacherInvalidParams
	}
	status := domain.NoticeStatusPublished
	now := time.Now()
	var scheduledAt *time.Time
	var publishedAt *time.Time = &now
	if req.ScheduleEnabled {
		t, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil || !t.After(now) {
			return nil, ErrTeacherInvalidParams
		}
		status = domain.NoticeStatusScheduled
		scheduledAt = &t
		publishedAt = nil
	}
	classID := tc.ClassID
	if req.ClassID > 0 && req.ClassID != tc.ClassID {
		// 验证这个班级属于当前老师
		var ct domain.ClassTeacher
		err := s.db.WithContext(ctx).Joins("INNER JOIN classes c ON c.id = class_teachers.class_id AND c.kindergarten_id = ?", tc.KindergartenID).
			Where("class_teachers.user_id = ? AND class_teachers.class_id = ?", tc.UserID, req.ClassID).First(&ct).Error
		if err != nil {
			return nil, ErrTeacherForbidden
		}
		classID = req.ClassID
	}
	notice := &domain.Notice{
		KindergartenID: tc.KindergartenID, ClassID: classID, CreatorID: tc.UserID,
		Category: domain.NoticeCategory(req.Category), Title: strings.TrimSpace(req.Title), Content: strings.TrimSpace(req.Content),
		NeedConfirm: req.NeedConfirm, ScheduledAt: scheduledAt, PublishedAt: publishedAt, Status: status,
	}
	if notice.Category == "" {
		notice.Category = domain.NoticeCategoryOther
	}
	if err := s.db.WithContext(ctx).Create(notice).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": notice.ID, "status": notice.Status, "status_label": teacherPortalNoticeStatusLabel(notice.Status), "created_at": notice.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *TeacherPortalService) GetNotice(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var n domain.Notice
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	list, err := s.ListNotices(ctx, tc, nil, "", 1, 1)
	if err != nil {
		return nil, err
	}
	_ = list
	var readCount int64
	_ = s.db.WithContext(ctx).Model(&domain.NoticeReadLog{}).Where("notice_id = ?", n.ID).Count(&readCount).Error
	totalCount := s.classStudentCount(ctx, tc)
	return map[string]any{
		"id": n.ID, "category": n.Category, "category_label": teacherPortalNoticeCategoryLabel(n.Category),
		"title": n.Title, "content": n.Content, "need_confirm": n.NeedConfirm,
		"status": n.Status, "status_label": teacherPortalNoticeStatusLabel(n.Status),
		"read_count": readCount, "total_count": totalCount, "unread_count": totalCount - readCount,
		"published_at": formatOptionalTime(n.PublishedAt), "created_at": n.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *TeacherPortalService) UpdateNotice(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherNoticeInput) error {
	updates := map[string]any{}
	if strings.TrimSpace(req.Title) != "" {
		updates["title"] = strings.TrimSpace(req.Title)
	}
	if strings.TrimSpace(req.Content) != "" {
		updates["content"] = strings.TrimSpace(req.Content)
	}
	if strings.TrimSpace(req.Category) != "" {
		updates["category"] = req.Category
	}
	updates["need_confirm"] = req.NeedConfirm
	res := s.db.WithContext(ctx).Model(&domain.Notice{}).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) DeleteNotice(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Delete(&domain.Notice{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) RevokeNotice(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Model(&domain.Notice{}).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Update("status", domain.NoticeStatusRevoked)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) NoticeReadStatus(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.GetNotice(ctx, tc, id); err != nil {
		return nil, err
	}
	var readCount int64
	_ = s.db.WithContext(ctx).Model(&domain.NoticeReadLog{}).Where("notice_id = ?", id).Count(&readCount).Error
	totalCount := s.classStudentCount(ctx, tc)
	return map[string]any{"notice_id": id, "read_count": readCount, "unread_count": totalCount - readCount, "total_count": totalCount, "unread_students": []any{}}, nil
}

func (s *TeacherPortalService) ListNoticeTemplates(ctx context.Context, tc *TeacherPortalContext) (map[string]any, error) {
	var rows []domain.NoticeTemplate
	if err := s.db.WithContext(ctx).Where("kindergarten_id = ? AND creator_id = ?", tc.KindergartenID, tc.UserID).Order("usage_count DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": len(rows)}, nil
}

func (s *TeacherPortalService) CreateNoticeTemplate(ctx context.Context, tc *TeacherPortalContext, req TeacherNoticeInput) (map[string]any, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrTeacherInvalidParams
	}
	tpl := &domain.NoticeTemplate{KindergartenID: tc.KindergartenID, CreatorID: tc.UserID, Category: domain.NoticeCategory(req.Category), Title: req.Title, Content: req.Content}
	if err := s.db.WithContext(ctx).Create(tpl).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": tpl.ID, "created_at": tpl.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *TeacherPortalService) UpdateNoticeTemplate(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherNoticeInput) error {
	res := s.db.WithContext(ctx).Model(&domain.NoticeTemplate{}).Where("id = ? AND kindergarten_id = ? AND creator_id = ?", id, tc.KindergartenID, tc.UserID).Updates(map[string]any{"category": req.Category, "title": req.Title, "content": req.Content})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) DeleteNoticeTemplate(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND creator_id = ?", id, tc.KindergartenID, tc.UserID).Delete(&domain.NoticeTemplate{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) ListLeaves(ctx context.Context, tc *TeacherPortalContext, status string, page, pageSize int) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	db := s.db.WithContext(ctx).Model(&domain.LeaveRequest{}).Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID)
	switch status {
	case "pending":
		db = db.Where("status = ?", domain.LeaveStatusPending)
	case "approved":
		db = db.Where("status = ?", domain.LeaveStatusApproved)
	case "rejected":
		db = db.Where("status = ?", domain.LeaveStatusRejected)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []domain.LeaveRequest
	if err := db.Preload("Student").Preload("Applicant").Order("created_at DESC").Limit(q.PageSize).Offset(q.Offset()).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	for _, lr := range rows {
		list = append(list, teacherPortalLeaveMap(lr))
	}
	return map[string]any{"list": list, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

func teacherPortalLeaveMap(lr domain.LeaveRequest) map[string]any {
	student := map[string]any{"id": lr.StudentID, "name": "学生", "avatar": ""}
	if lr.Student != nil {
		student["name"] = lr.Student.Name
		student["avatar"] = lr.Student.Avatar
	}
	applicant := map[string]any{"id": lr.ApplicantID, "nickname": "家长", "relation": ""}
	if lr.Applicant != nil {
		applicant["nickname"] = lr.Applicant.Nickname
	}
	return map[string]any{
		"id": lr.ID, "student": student, "applicant": applicant, "reason": lr.Reason,
		"start_date": teacherPortalDateString(lr.StartDate), "end_date": teacherPortalDateString(lr.EndDate), "days": lr.Days,
		"status": lr.Status, "status_label": teacherPortalStatusLabel(int8(lr.Status)), "review_remark": lr.ReviewRemark,
		"created_at": lr.CreatedAt.UTC().Format(time.RFC3339), "reviewed_at": formatOptionalTime(lr.ReviewedAt),
	}
}

func (s *TeacherPortalService) LeaveStats(ctx context.Context, tc *TeacherPortalContext) (map[string]any, error) {
	var pending, approved, rejected, monthly int64
	monthStart := teacherPortalDay(time.Now()).AddDate(0, 0, -time.Now().Day()+1)
	db := s.db.WithContext(ctx).Model(&domain.LeaveRequest{}).Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID)
	if err := db.Where("status = ?", domain.LeaveStatusPending).Count(&pending).Error; err != nil {
		return nil, err
	}
	_ = db.Where("status = ?", domain.LeaveStatusApproved).Count(&approved).Error
	_ = db.Where("status = ?", domain.LeaveStatusRejected).Count(&rejected).Error
	_ = db.Where("created_at >= ?", monthStart).Count(&monthly).Error
	return map[string]any{"pending": pending, "approved": approved, "rejected": rejected, "monthly_total": monthly}, nil
}

func (s *TeacherPortalService) GetLeave(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var lr domain.LeaveRequest
	if err := s.db.WithContext(ctx).Preload("Student").Preload("Applicant").Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&lr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	return teacherPortalLeaveMap(lr), nil
}

func (s *TeacherPortalService) AuditLeave(ctx context.Context, tc *TeacherPortalContext, id uint64, approve bool, remark string) error {
	if !approve && strings.TrimSpace(remark) == "" {
		return ErrTeacherInvalidParams
	}
	action := "approve"
	if !approve {
		action = "reject"
	}
	_, err := s.TodoAction(ctx, tc, id, action, remark)
	return err
}

type TeacherGrowthInput struct {
	Date            string   `json:"date"`
	Content         string   `json:"content"`
	PerformanceTags []string `json:"performance_tags"`
	SyncToFeed      bool     `json:"sync_to_feed"`
	TargetType      int8     `json:"target_type"`
	StudentIDs      []uint64 `json:"student_ids"`
	BreakfastRating int8     `json:"breakfast_rating"`
	LunchRating     int8     `json:"lunch_rating"`
	DinnerRating    int8     `json:"dinner_rating"`
	NapDuration     int      `json:"nap_duration"`
	SleepQuality    int8     `json:"sleep_quality"`
	MorningMood     int8     `json:"morning_mood"`
	AfternoonMood   int8     `json:"afternoon_mood"`
}

func (s *TeacherPortalService) ListGrowthLogs(ctx context.Context, tc *TeacherPortalContext, page, pageSize int) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	db := s.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []domain.GrowthDailyLog
	if err := db.Order("date DESC, id DESC").Limit(q.PageSize).Offset(q.Offset()).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		list = append(list, teacherPortalGrowthMap(r))
	}
	return map[string]any{"list": list, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

func teacherPortalGrowthMap(r domain.GrowthDailyLog) map[string]any {
	return map[string]any{"id": r.ID, "date": teacherPortalDateString(r.Date), "content": r.Content, "performance_tags": r.PerformanceTags, "sync_to_feed": r.SyncToFeed, "target_type": r.TargetType, "breakfast_rating": r.BreakfastRating, "lunch_rating": r.LunchRating, "dinner_rating": r.DinnerRating, "nap_duration": r.NapDuration, "sleep_quality": r.SleepQuality, "morning_mood": r.MorningMood, "afternoon_mood": r.AfternoonMood, "created_at": r.CreatedAt.UTC().Format(time.RFC3339)}
}

func (s *TeacherPortalService) CreateGrowthLog(ctx context.Context, tc *TeacherPortalContext, req TeacherGrowthInput) (map[string]any, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, ErrTeacherInvalidParams
	}
	day := teacherPortalDay(time.Now())
	if strings.TrimSpace(req.Date) != "" {
		if t, err := time.ParseInLocation("2006-01-02", req.Date, time.Local); err == nil {
			day = teacherPortalDay(t)
		}
	}
	log := &domain.GrowthDailyLog{KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, CreatorID: tc.UserID, Date: day, Content: req.Content, PerformanceTags: strings.Join(req.PerformanceTags, ","), SyncToFeed: req.SyncToFeed, TargetType: req.TargetType, BreakfastRating: domain.MealRating(req.BreakfastRating), LunchRating: domain.MealRating(req.LunchRating), DinnerRating: domain.MealRating(req.DinnerRating), NapDuration: req.NapDuration, SleepQuality: domain.SleepQuality(req.SleepQuality), MorningMood: domain.MoodRating(req.MorningMood), AfternoonMood: domain.MoodRating(req.AfternoonMood)}
	if log.TargetType == 0 {
		log.TargetType = 1
	}
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		return nil, err
	}
	for _, sid := range req.StudentIDs {
		_ = s.db.WithContext(ctx).Create(&domain.GrowthLogStudent{LogID: log.ID, StudentID: sid}).Error
	}
	return map[string]any{"id": log.ID, "created_at": log.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *TeacherPortalService) GetGrowthLog(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var r domain.GrowthDailyLog
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	return teacherPortalGrowthMap(r), nil
}

func (s *TeacherPortalService) UpdateGrowthLog(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherGrowthInput) error {
	res := s.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Updates(map[string]any{"content": req.Content, "performance_tags": strings.Join(req.PerformanceTags, ","), "sync_to_feed": req.SyncToFeed, "target_type": req.TargetType, "breakfast_rating": req.BreakfastRating, "lunch_rating": req.LunchRating, "dinner_rating": req.DinnerRating, "nap_duration": req.NapDuration, "sleep_quality": req.SleepQuality, "morning_mood": req.MorningMood, "afternoon_mood": req.AfternoonMood})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) DeleteGrowthLog(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Delete(&domain.GrowthDailyLog{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) GrowthComments(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil {
		return nil, err
	}
	var rows []domain.GrowthComment
	if err := s.db.WithContext(ctx).Preload("User").Where("log_id = ?", id).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": len(rows)}, nil
}

func (s *TeacherPortalService) CreateGrowthComment(ctx context.Context, tc *TeacherPortalContext, id uint64, content string) (map[string]any, error) {
	if strings.TrimSpace(content) == "" {
		return nil, ErrTeacherInvalidParams
	}
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil {
		return nil, err
	}
	row := &domain.GrowthComment{LogID: id, UserID: tc.UserID, Content: strings.TrimSpace(content)}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *TeacherPortalService) GrowthLikes(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil {
		return nil, err
	}
	var total int64
	_ = s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", id).Count(&total).Error
	return map[string]any{"list": []any{}, "total": total}, nil
}

func (s *TeacherPortalService) LikeGrowth(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil { return nil, err }
	var existing int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ? AND user_id = ?", id, tc.UserID).Count(&existing)
	if existing == 0 {
		s.db.WithContext(ctx).Create(&domain.GrowthLike{LogID: id, UserID: tc.UserID})
	}
	var total int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", id).Count(&total)
	return map[string]any{"id": id, "is_liked": true, "likes_count": total}, nil
}

func (s *TeacherPortalService) UnlikeGrowth(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil { return nil, err }
	s.db.WithContext(ctx).Where("log_id = ? AND user_id = ?", id, tc.UserID).Delete(&domain.GrowthLike{})
	var total int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", id).Count(&total)
	return map[string]any{"id": id, "is_liked": false, "likes_count": total}, nil
}

func (s *TeacherPortalService) GrowthRelatedStudents(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil {
		return nil, err
	}
	var rows []domain.Student
	err := s.db.WithContext(ctx).Joins("INNER JOIN growth_log_students gls ON gls.student_id = students.id AND gls.log_id = ?", id).Where("students.kindergarten_id = ? AND students.class_id = ?", tc.KindergartenID, tc.ClassID).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": len(rows)}, nil
}

type TeacherMediaInput struct {
	FileURL  string `json:"file_url"`
	ThumbURL string `json:"thumb_url"`
	FileType string `json:"file_type"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

func (s *TeacherPortalService) CreateGrowthMedia(ctx context.Context, tc *TeacherPortalContext, req TeacherMediaInput) (map[string]any, error) {
	if strings.TrimSpace(req.FileURL) == "" {
		return nil, ErrTeacherInvalidParams
	}
	m := &domain.GrowthMedia{KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, UploaderID: tc.UserID, FileURL: req.FileURL, ThumbURL: req.ThumbURL, FileType: req.FileType, FileName: req.FileName, FileSize: req.FileSize}
	if m.FileType == "" {
		m.FileType = "image"
	}
	if err := s.db.WithContext(ctx).Create(m).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": m.ID, "file_url": m.FileURL}, nil
}

func (s *TeacherPortalService) PickupOverview(ctx context.Context, tc *TeacherPortalContext) (map[string]any, error) {
	today := teacherPortalDay(time.Now())
	total := s.classStudentCount(ctx, tc)
	var arrived, picked, abnormal int64
	_ = s.db.WithContext(ctx).Model(&domain.PickupRecord{}).Where("kindergarten_id = ? AND class_id = ? AND date = ? AND type = ?", tc.KindergartenID, tc.ClassID, today, domain.PickupTypeDropOff).Distinct("student_id").Count(&arrived).Error
	_ = s.db.WithContext(ctx).Model(&domain.PickupRecord{}).Where("kindergarten_id = ? AND class_id = ? AND date = ? AND type = ?", tc.KindergartenID, tc.ClassID, today, domain.PickupTypePickUp).Distinct("student_id").Count(&picked).Error
	_ = s.db.WithContext(ctx).Model(&domain.PickupRecord{}).Where("kindergarten_id = ? AND class_id = ? AND date = ? AND is_abnormal = ?", tc.KindergartenID, tc.ClassID, today, true).Count(&abnormal).Error
	return map[string]any{"date": teacherPortalDateString(today), "total": total, "arrived": arrived, "not_arrived": total - arrived, "picked_up": picked, "not_picked_up": total - picked, "abnormal": abnormal}, nil
}

func (s *TeacherPortalService) PickupStudents(ctx context.Context, tc *TeacherPortalContext, status string) (map[string]any, error) {
		today := teacherPortalDay(time.Now())
		base := s.db.WithContext(ctx).Table("students AS s").
			Where("s.kindergarten_id = ? AND s.class_id = ? AND s.status = 1 AND s.deleted_at IS NULL", tc.KindergartenID, tc.ClassID)

		switch status {
		case "arrived":
			// 当日有送入记录
			base = base.Where(`EXISTS (SELECT 1 FROM pickup_records pr WHERE pr.student_id = s.id AND pr.deleted_at IS NULL AND pr.date = ? AND pr.type = ?)`, today, domain.PickupTypeDropOff)
		case "picked_up":
			// 当日有接走记录
			base = base.Where(`EXISTS (SELECT 1 FROM pickup_records pr WHERE pr.student_id = s.id AND pr.deleted_at IS NULL AND pr.date = ? AND pr.type = ?)`, today, domain.PickupTypePickUp)
		case "not_picked_up":
			// 当日有送入但无接走
			base = base.Where(`EXISTS (SELECT 1 FROM pickup_records pr WHERE pr.student_id = s.id AND pr.deleted_at IS NULL AND pr.date = ? AND pr.type = ?)`, today, domain.PickupTypeDropOff).
				Where(`NOT EXISTS (SELECT 1 FROM pickup_records pr WHERE pr.student_id = s.id AND pr.deleted_at IS NULL AND pr.date = ? AND pr.type = ?)`, today, domain.PickupTypePickUp)
		default:
			// 不传 status 返回全部在读学生
		}

		type studentRow struct {
			ID     uint64 `gorm:"column:id"`
			Name   string `gorm:"column:name"`
			Avatar string `gorm:"column:avatar"`
		}
		var rows []studentRow
		if err := base.Select("s.id, s.name, COALESCE(s.avatar,'') AS avatar").Order("s.id ASC").Scan(&rows).Error; err != nil {
			return nil, err
		}

		// 取当日到达时间 & 接送状态
		studentIDs := make([]uint64, len(rows))
		for i, r := range rows { studentIDs[i] = r.ID }

		type pickupInfo struct {
			StudentID uint64    `gorm:"column:student_id"`
			PickType  int8      `gorm:"column:pick_type"`
			PickTime  time.Time `gorm:"column:pick_time"`
		}
		var pickups []pickupInfo
		_ = s.db.WithContext(ctx).Raw(`SELECT student_id, type AS pick_type, MIN(recorded_at) AS pick_time FROM pickup_records WHERE student_id IN ? AND deleted_at IS NULL AND date = ? GROUP BY student_id, type`, studentIDs, today).Scan(&pickups).Error

		arriveMap := make(map[uint64]string)
		pickupMap := make(map[uint64]bool)
		timeLayout := "15:04"
		for _, p := range pickups {
			if p.PickType == int8(domain.PickupTypeDropOff) {
				arriveMap[p.StudentID] = p.PickTime.Format(timeLayout)
			} else if p.PickType == int8(domain.PickupTypePickUp) {
				pickupMap[p.StudentID] = true
			}
		}

		// 取授权接送人姓名
		authMap := make(map[uint64]string)
		var authRows []struct {
			StudentID uint64 `gorm:"column:student_id"`
			Names     string `gorm:"column:names"`
		}
		_ = s.db.WithContext(ctx).Raw(`SELECT student_id, GROUP_CONCAT(name SEPARATOR '、') AS names FROM pickup_authorizations WHERE student_id IN ? AND deleted_at IS NULL AND status = 1 AND review_status = 1 GROUP BY student_id`, studentIDs).Scan(&authRows).Error
		for _, a := range authRows { authMap[a.StudentID] = a.Names }

		list := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			ps := "not_picked_up"
			if pickupMap[r.ID] { ps = "picked_up" }
			if _, ok := arriveMap[r.ID]; !ok && !pickupMap[r.ID] { ps = "not_arrived" }
			if arriveMap[r.ID] != "" && !pickupMap[r.ID] { ps = "not_picked_up" }
			at := arriveMap[r.ID]
			ap := authMap[r.ID]
			if ap == "" { ap = "暂无" }
			list = append(list, map[string]any{
				"student_id":        r.ID,
				"name":              r.Name,
				"avatar":            r.Avatar,
				"authorized_persons": ap,
				"arrive_time":       at,
				"pickup_status":     ps,
			})
		}

		return map[string]any{"list": list, "total": len(list)}, nil
	}

type TeacherPickupInput struct {
	StudentID      uint64 `json:"student_id"`
	Type           int8   `json:"type"`
	PickupName     string `json:"pickup_name"`
	PickupRelation string `json:"pickup_relation"`
	Remark         string `json:"remark"`
	Code           string `json:"code"`
		QRCode         string `json:"qr_code"`
}

func (s *TeacherPortalService) CreatePickupRecord(ctx context.Context, tc *TeacherPortalContext, req TeacherPickupInput, method domain.PickupMethod) (map[string]any, error) {
		if req.StudentID == 0 {
			return nil, ErrTeacherInvalidParams
		}
		// 兼容 code 和 qr_code 两种 key
		if req.Code == "" { req.Code = req.QRCode }

		var st domain.Student
		if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", req.StudentID, tc.KindergartenID, tc.ClassID).First(&st).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrTeacherNotFound
			}
			return nil, err
		}
		if req.Type == 0 {
			req.Type = int8(domain.PickupTypeDropOff)
		}
		now := time.Now()
		today := teacherPortalDay(now)

		// P1-1: 防重 — 同学生同日同类型不可重复登记
		var duplicate int64
		if err := s.db.WithContext(ctx).Model(&domain.PickupRecord{}).
			Where("student_id = ? AND date = ? AND type = ? AND deleted_at IS NULL", req.StudentID, today, req.Type).
			Count(&duplicate).Error; err != nil {
			return nil, err
		}
		if duplicate > 0 {
			return nil, ErrTeacherDuplicate
		}

		// P1-2: 接走前必须先有送入
		isPickup := domain.PickupType(req.Type) == domain.PickupTypePickUp
		if isPickup {
			var dropOff int64
			if err := s.db.WithContext(ctx).Model(&domain.PickupRecord{}).
				Where("student_id = ? AND date = ? AND type = ? AND deleted_at IS NULL", req.StudentID, today, domain.PickupTypeDropOff).
				Count(&dropOff).Error; err != nil {
				return nil, err
			}
			if dropOff == 0 {
				return nil, ErrTeacherPickupOrder
			}
		}

		// P1-3: 扫码模式校验二维码有效性
		if method == domain.PickupMethodScan {
			if req.Code == "" {
				return nil, ErrTeacherInvalidParams
			}
			var qr domain.PickupQRCode
			err := s.db.WithContext(ctx).Where("code = ? AND is_active = ?", req.Code, true).First(&qr).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, ErrTeacherInvalidQRCode
				}
				return nil, err
			}
			if qr.ExpireAt.Before(now) {
				return nil, ErrTeacherInvalidQRCode
			}
			if qr.StudentID != req.StudentID {
				return nil, ErrTeacherInvalidQRCode
			}
		}

		// 检查是否为授权接送人
		var isAuthorized bool
		var pickupPersonID *uint64

		row := &domain.PickupRecord{
			KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, StudentID: req.StudentID,
			Date: today, Type: domain.PickupType(req.Type), Method: method,
			PickupName: req.PickupName, PickupRelation: req.PickupRelation,
			OperatorID: tc.UserID, RecordedAt: now, Remark: req.Remark,
			PickupPersonID: pickupPersonID,
		}
		if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
			return nil, err
		}

		// P1-3: 扫码成功后失效二维码
		if method == domain.PickupMethodScan && req.Code != "" {
			_ = s.db.WithContext(ctx).Model(&domain.PickupQRCode{}).
				Where("code = ?", req.Code).Update("is_active", false).Error
		}

		// 入园(type=1) → 自动生成待晨检记录
		if row.Type == domain.PickupTypeDropOff {
			hc := &domain.MorningHealthCheck{
				KindergartenID: tc.KindergartenID,
				ClassID:        tc.ClassID,
				StudentID:      req.StudentID,
				PickupRecordID: row.ID,
				Date:           row.Date.Format("2006-01-02"),
				CheckStatus:    domain.HealthCheckPending,
			}
			if err := s.db.WithContext(ctx).Create(hc).Error; err != nil {
				println("auto-create health check failed:", err.Error())
			}
		}

		resp := map[string]any{
			"record_id":     row.ID,
			"student":       map[string]any{"id": st.ID, "name": st.Name, "avatar": st.Avatar},
			"type":          int(row.Type),
			"type_label":    pickupTypeLabel(domain.PickupType(req.Type)),
			"is_authorized": isAuthorized,
			"recorded_at":   row.RecordedAt.UTC().Format(time.RFC3339),
		}

		return resp, nil
	}

func (s *TeacherPortalService) PickupRecords(ctx context.Context, tc *TeacherPortalContext, page, pageSize int, date string, studentID *uint64) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	db := s.db.WithContext(ctx).Model(&domain.PickupRecord{}).Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID)
	if date != "" {
		d, err := time.Parse("2006-01-02", date)
		if err == nil { db = db.Where("date = ?", d) }
	}
	if studentID != nil && *studentID > 0 {
		db = db.Where("student_id = ?", *studentID)
	}
	var total int64
	_ = db.Count(&total).Error
	var rows []domain.PickupRecord
	if err := db.Preload("Student").Order("recorded_at DESC").Limit(q.PageSize).Offset(q.Offset()).Find(&rows).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

func (s *TeacherPortalService) PickupRecord(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var row domain.PickupRecord
	if err := s.db.WithContext(ctx).Preload("Student").Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	return map[string]any{"record": row}, nil
}

func (s *TeacherPortalService) MarkPickupAbnormal(ctx context.Context, tc *TeacherPortalContext, id uint64, isAbnormal bool, reason string) (map[string]any, error) {
	res := s.db.WithContext(ctx).Model(&domain.PickupRecord{}).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Updates(map[string]any{"is_abnormal": isAbnormal, "abnormal_reason": reason})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrTeacherNotFound
	}
	now := time.Now().UTC().Format(time.RFC3339)
	return map[string]any{"id": id, "is_abnormal": isAbnormal, "abnormal_reason": reason, "updated_at": now}, nil
}

	// AuditPickupAuth 老师审核授权接送人（通过/拒绝）。
	func (s *TeacherPortalService) AuditPickupAuth(ctx context.Context, tc *TeacherPortalContext, authID uint64, approve bool, remark string) (map[string]any, error) {
		// 查询授权记录，通过 student 做 class_id 范围校验
		var pa domain.PickupAuthorization
		err := s.db.WithContext(ctx).Model(&domain.PickupAuthorization{}).
			Joins("INNER JOIN students AS st ON st.id = pickup_authorizations.student_id AND st.deleted_at IS NULL").
			Where("pickup_authorizations.id = ? AND st.class_id = ? AND st.kindergarten_id = ?", authID, tc.ClassID, tc.KindergartenID).
			First(&pa).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrTeacherNotFound
			}
			return nil, err
		}

		// 只能审核待审核状态
		if pa.ReviewStatus != 0 {
			return nil, ErrTeacherInvalidParams
		}

		var reviewStatus int8
		var label string
		if approve {
			reviewStatus = 1
			label = "已通过"
		} else {
			reviewStatus = 2
			label = "已拒绝"
		}

		now := time.Now()
		if err := s.db.WithContext(ctx).Model(&domain.PickupAuthorization{}).
			Where("id = ?", authID).
			Updates(map[string]any{
				"review_status": reviewStatus,
				"audited_at":    now,
				"audit_remark":  remark,
			}).Error; err != nil {
			return nil, err
		}

		return map[string]any{
			"id":           authID,
			"status":       reviewStatus,
			"status_label": label,
			"audited_at":   now.UTC().Format(time.RFC3339),
		}, nil
	}

	type TeacherCalendarInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	EventType   string `json:"event_type"`
	EventDate   string `json:"event_date"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Location    string `json:"location"`
	IsAllDay    bool   `json:"is_all_day"`
	Color       string `json:"color"`
	Importance  string `json:"importance"`
	Cover       string `json:"cover"`
	IsPublished bool   `json:"is_published"`
}

func (s *TeacherPortalService) CalendarEvents(ctx context.Context, tc *TeacherPortalContext, start, end string) (map[string]any, error) {
	rows, err := s.calendar.FindEvents(ctx, tc.KindergartenID, tc.ClassID, start, end)
	if err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": len(rows)}, nil
}

func (s *TeacherPortalService) CalendarEventsByDate(ctx context.Context, tc *TeacherPortalContext, date string) (map[string]any, error) {
	return s.CalendarEvents(ctx, tc, date, date)
}

// GetCalendarEvent 查询单条日历事件详情。
func (s *TeacherPortalService) GetCalendarEvent(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	row, err := s.calendar.FindEventByID(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrTeacherNotFound
	}
	return map[string]any{"event": row}, nil
}

func (s *TeacherPortalService) CreateCalendarEvent(ctx context.Context, tc *TeacherPortalContext, req TeacherCalendarInput) (map[string]any, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.EventDate) == "" {
		return nil, ErrTeacherInvalidParams
	}
	day, err := time.ParseInLocation("2006-01-02", req.EventDate, time.Local)
	if err != nil {
		return nil, ErrTeacherInvalidParams
	}
	row := &domain.CalendarEvent{KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, CreatorID: tc.UserID, Title: req.Title, Description: req.Description, EventType: domain.EventType(req.EventType), EventDate: teacherPortalDay(day), StartTime: req.StartTime, EndTime: req.EndTime, Location: req.Location, IsAllDay: req.IsAllDay, Color: req.Color, Importance: req.Importance, Cover: req.Cover, IsPublished: req.IsPublished}
	if row.EventType == "" {
		row.EventType = domain.EventTypeActivity
	}
	if err := s.calendar.CreateEvent(ctx, row); err != nil {
		return nil, err
	}
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *TeacherPortalService) UpdateCalendarEvent(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherCalendarInput) error {
	updates := map[string]any{"title": req.Title, "description": req.Description, "event_type": req.EventType, "start_time": req.StartTime, "end_time": req.EndTime, "location": req.Location, "is_all_day": req.IsAllDay, "color": req.Color, "importance": req.Importance, "cover": req.Cover, "is_published": req.IsPublished}
	if req.EventDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", req.EventDate, time.Local); err == nil {
			updates["event_date"] = teacherPortalDay(t)
		}
	}
	affected, err := s.calendar.UpdateEvent(ctx, tc.KindergartenID, tc.ClassID, id, updates)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) DeleteCalendarEvent(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	affected, err := s.calendar.DeleteEvent(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) CalendarDots(ctx context.Context, tc *TeacherPortalContext, start, end string) (map[string]any, error) {
	rows, err := s.calendar.AggregateDots(ctx, tc.KindergartenID, tc.ClassID, start, end)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		list = append(list, map[string]any{"date": teacherPortalDateString(r.Date), "count": r.Count})
	}
	return map[string]any{"list": list}, nil
}

func (s *TeacherPortalService) ClassInfo(ctx context.Context, tc *TeacherPortalContext) (map[string]any, error) {
	var class domain.Class
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ?", tc.ClassID, tc.KindergartenID).First(&class).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": class.ID, "name": class.Name, "grade": class.Grade, "academic_year": class.AcademicYear, "teacher": tc, "student_count": s.classStudentCount(ctx, tc)}, nil
}

func (s *TeacherPortalService) ClassStudents(ctx context.Context, tc *TeacherPortalContext) (map[string]any, error) {
	return s.PickupStudents(ctx, tc, "")
}

func (s *TeacherPortalService) ClassStudent(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var st domain.Student
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	return map[string]any{"student": st}, nil
}

// ListTeacherClasses 返回当前老师在该园所的所有关联班级。
func (s *TeacherPortalService) ListTeacherClasses(ctx context.Context, tc *TeacherPortalContext) ([]map[string]any, error) {
	var cts []domain.ClassTeacher
	if err := s.db.WithContext(ctx).Preload("Class").
		Joins("INNER JOIN classes c ON c.id = class_teachers.class_id AND c.kindergarten_id = ?", tc.KindergartenID).
		Where("class_teachers.user_id = ? AND class_teachers.status = ?", tc.UserID, 1).
		Order("class_teachers.id ASC").Find(&cts).Error; err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(cts))
	for _, ct := range cts {
		cn := ""
		if ct.Class != nil {
			cn = ct.Class.Name
		}
		out = append(out, map[string]any{
			"class_id":    ct.ClassID,
			"class_name":  cn,
			"title":       string(ct.Title),
			"title_label": teacherPortalTitleLabel(ct.Title),
		})
	}
	return out, nil
}

type TeacherTaskInput struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Intro        string   `json:"intro"`
	Requirements []string `json:"requirements"`
	Rewards      []map[string]any `json:"rewards"`
	Frequency    string   `json:"frequency"`
	StartDate    string   `json:"start_date"`
	EndDate      string   `json:"end_date"`
	Points       int      `json:"points"`
	Status       int8     `json:"status"`
}

func (s *TeacherPortalService) ListTasks(ctx context.Context, tc *TeacherPortalContext, page, pageSize int) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	db := s.db.WithContext(ctx).Model(&domain.ParentTask{}).Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID)
	var total int64
	_ = db.Count(&total).Error
	var rows []domain.ParentTask
	if err := db.Order("created_at DESC").Limit(q.PageSize).Offset(q.Offset()).Find(&rows).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

func (s *TeacherPortalService) CreateTask(ctx context.Context, tc *TeacherPortalContext, req TeacherTaskInput) (map[string]any, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrTeacherInvalidParams
	}
	start := teacherPortalDay(time.Now())
	end := start.AddDate(0, 0, 7)
	if t, err := time.ParseInLocation("2006-01-02", req.StartDate, time.Local); err == nil {
		start = teacherPortalDay(t)
	}
	if t, err := time.ParseInLocation("2006-01-02", req.EndDate, time.Local); err == nil {
		end = teacherPortalDay(t)
	}
		reqsJSON, _ := json.Marshal(req.Requirements)
		rewardsJSON, _ := json.Marshal(req.Rewards)
		row := &domain.ParentTask{KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, CreatorID: tc.UserID, Title: req.Title, Description: req.Description, Intro: req.Intro, Requirements: string(reqsJSON), Rewards: string(rewardsJSON), Frequency: domain.TaskFrequency(req.Frequency), StartDate: start, EndDate: end, Points: req.Points, Status: 1}
	if row.Frequency == "" {
		row.Frequency = domain.TaskFrequencyOnce
	}
	if row.Points == 0 {
		row.Points = 1
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *TeacherPortalService) UpdateTask(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherTaskInput) error {
	res := s.db.WithContext(ctx).Model(&domain.ParentTask{}).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Updates(map[string]any{"title": req.Title, "description": req.Description, "frequency": req.Frequency, "points": req.Points, "status": req.Status})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) DeleteTask(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Delete(&domain.ParentTask{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) TaskStats(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&task).Error; err != nil {
		return nil, ErrTeacherNotFound
	}
	var checkins int64
	_ = s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).Where("task_id = ?", id).Count(&checkins).Error
	total := s.classStudentCount(ctx, tc)
	return map[string]any{"task_id": id, "total_students": total, "checkin_count": checkins, "not_checkin_count": total - checkins}, nil
}

func (s *TeacherPortalService) TaskCheckins(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.TaskStats(ctx, tc, id); err != nil {
		return nil, err
	}
	var rows []domain.TaskCheckIn
	if err := s.db.WithContext(ctx).Preload("Student").Where("task_id = ?", id).Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": len(rows)}, nil
}

// ListTeacherMessages 聚合老师端消息列表（通知发布记录 + 请假审批 + 系统消息）。
func (s *TeacherPortalService) ListTeacherMessages(ctx context.Context, tc *TeacherPortalContext, page, pageSize int) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	db := s.db.WithContext(ctx)
	var items []domain.TeacherMessageItem

	// 1) 已发布/定时发送的通知（本班级、当前老师发布）
	totalStudents := s.classStudentCount(ctx, tc)
	{
		var notices []domain.Notice
		if err := db.Where("kindergarten_id = ? AND class_id = ? AND creator_id = ? AND status IN ?",
			tc.KindergartenID, tc.ClassID, tc.UserID,
			[]domain.NoticeStatus{domain.NoticeStatusScheduled, domain.NoticeStatusPublished},
		).Find(&notices).Error; err != nil {
			return nil, err
		}
		for _, n := range notices {
			var readCount int64
			_ = db.Model(&domain.NoticeReadLog{}).Where("notice_id = ?", n.ID).Count(&readCount).Error
			readRate := "0%"
			if totalStudents > 0 {
				readRate = fmt.Sprintf("%d%%", readCount*100/totalStudents)
			}
			content := n.Content
			if len([]rune(content)) > 200 {
				content = string([]rune(content)[:200]) + "..."
			}
			items = append(items, domain.TeacherMessageItem{
				Type:        domain.TeacherMsgTypeNotice,
				Title:       n.Title,
				Content:     content,
				ReferenceID: n.ID,
				Extra: map[string]any{
					"category":       n.Category,
					"category_label": teacherPortalNoticeCategoryLabel(n.Category),
					"status":         n.Status,
					"status_label":   teacherPortalNoticeStatusLabel(n.Status),
					"read_count":     readCount,
					"total_count":    totalStudents,
					"read_rate":      readRate,
				},
				CreatedAt: n.CreatedAt,
			})
		}
	}

	// 2) 请假审批（本班级）
	{
		var leaves []domain.LeaveRequest
		if err := db.Preload("Student").Preload("Applicant").
			Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID).
			Order("created_at DESC").Find(&leaves).Error; err != nil {
			return nil, err
		}
		for _, lr := range leaves {
			applicantName := "家长"
			if lr.Applicant != nil && strings.TrimSpace(lr.Applicant.Nickname) != "" {
				applicantName = lr.Applicant.Nickname
			}
			studentName := "学生"
			if lr.Student != nil {
				studentName = lr.Student.Name
			}
			title := fmt.Sprintf("%s提交的请假申请", applicantName)
			content := fmt.Sprintf("%s：%s（%s 至 %s）", studentName, lr.Reason,
				teacherPortalDateString(lr.StartDate), teacherPortalDateString(lr.EndDate))
			if len([]rune(content)) > 200 {
				content = string([]rune(content)[:200]) + "..."
			}
			items = append(items, domain.TeacherMessageItem{
				Type:        domain.TeacherMsgTypeLeave,
				Title:       title,
				Content:     content,
				ReferenceID: lr.ID,
				Extra: map[string]any{
					"student_name":   studentName,
					"applicant_name": applicantName,
					"leave_dates": fmt.Sprintf("%s 至 %s",
						teacherPortalDateString(lr.StartDate), teacherPortalDateString(lr.EndDate)),
					"status":       lr.Status,
					"status_label": teacherPortalStatusLabel(int8(lr.Status)),
				},
				CreatedAt: lr.CreatedAt,
			})
		}
	}

	// 3) 系统消息（messages 表 type=system，发给当前老师）
	{
		var msgs []domain.Message
		if err := db.Where("user_id = ? AND type = ?", tc.UserID, string(domain.MessageTypeSystem)).
			Order("created_at DESC").Find(&msgs).Error; err != nil {
			return nil, err
		}
		for _, m := range msgs {
			content := m.Content
			if len([]rune(content)) > 200 {
				content = string([]rune(content)[:200]) + "..."
			}
			items = append(items, domain.TeacherMessageItem{
				Type:        domain.TeacherMsgTypeSystem,
				Title:       m.Title,
				Content:     content,
				ReferenceID: m.ReferenceID,
				Extra:       map[string]any{},
				CreatedAt:   m.CreatedAt,
			})
		}
	}

	// 4) 合并且按 CreatedAt 倒序排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	total := int64(len(items))
	offset := q.Offset()
	if int64(offset) >= total {
		return map[string]any{"list": []domain.TeacherMessageItem{}, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
	}
	end := int64(offset) + int64(q.PageSize)
	if end > total {
		end = total
	}
	paged := items[offset:end]

	return map[string]any{"list": paged, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}
