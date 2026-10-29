package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/log"
	"kindergarten-server/pkg/pagination"
)

type TeacherPortalService struct {
	db       *gorm.DB
	calendar *repository.CalendarRepo
	pickup   *repository.PickupRepo
	msg      *repository.MessageRepo
	notices  *repository.NoticeRepo
	queue    *RecognitionQueue // 照片识别入队（可为 nil：队列未配置时静默跳过）
	faceCfg  config.FaceConfig // 建议分级阈值（读取端标注高/低置信，前端不硬编码阈值）
}

func NewTeacherPortalService(
	db *gorm.DB,
	calendar *repository.CalendarRepo,
	pickup *repository.PickupRepo,
	msg *repository.MessageRepo,
	notices *repository.NoticeRepo,
	queue *RecognitionQueue,
	faceCfg config.FaceConfig,
) *TeacherPortalService {
	return &TeacherPortalService{db: db, calendar: calendar, pickup: pickup, msg: msg, notices: notices, queue: queue, faceCfg: faceCfg}
}

// enqueueRecognition 为一批照片建识别任务（发布路径零等待：仅插入任务行）。
func (s *TeacherPortalService) enqueueRecognition(ctx context.Context, tc *TeacherPortalContext, mediaIDs []uint64) {
	if s.queue == nil {
		return
	}
	for _, mid := range mediaIDs {
		if err := s.queue.Enqueue(ctx, tc.KindergartenID, mid); err != nil {
			// 入队失败不阻断发布（识别是增值路径）
			log.L(ctx).Error("enqueue recognition failed", "media", mid, "err", err)
		}
	}
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
	ErrTeacherContext       = errors.New("TEACHER_CONTEXT_NOT_FOUND")
	ErrTeacherForbidden     = errors.New("TEACHER_FORBIDDEN")
	ErrTeacherNotFound      = errors.New("TEACHER_RESOURCE_NOT_FOUND")
	ErrTeacherInvalidParams = errors.New("TEACHER_INVALID_PARAMS")
	// ErrTeacherRitualMediaType 语音回应必须挂音频文件，图片不能当语音发。
	ErrTeacherRitualMediaType = errors.New("TEACHER_RITUAL_MEDIA_TYPE")
	ErrTeacherDuplicate       = errors.New("PICKUP_DUPLICATE")
	ErrTeacherPickupOrder     = errors.New("PICKUP_ORDER_INVALID")
	ErrTeacherInvalidQRCode   = errors.New("INVALID_QR_CODE")
	// ErrTeacherConsecutiveDirection 连续同向登记：托管序列必须「送入—接走」交替
	ErrTeacherConsecutiveDirection = errors.New("PICKUP_CONSECUTIVE_DIRECTION")
	// ErrTeacherRemarkRequired 重复接走 / 提前接走时理由必填
	ErrTeacherRemarkRequired = errors.New("PICKUP_REMARK_REQUIRED")
	// ErrTeacherAuthorizationInvalid 接走人不在有效授权集合内（未审核/已撤销/已过期）
	ErrTeacherAuthorizationInvalid = errors.New("PICKUP_AUTHORIZATION_INVALID")
	// ErrTeacherOverrideNameRequired 标记授权例外时必须填写来接者姓名
	ErrTeacherOverrideNameRequired = errors.New("PICKUP_OVERRIDE_NAME_REQUIRED")
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
	loc := shanghaiLoc()
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
	ClassIDs        []uint64 `json:"class_ids"`
}

// taughtClassIDs 老师的任教班级集（ClassTeacher 多行并存）。无任教行时退回上下文当前班。
func (s *TeacherPortalService) taughtClassIDs(ctx context.Context, tc *TeacherPortalContext) ([]uint64, error) {
	var ids []uint64
	err := s.db.WithContext(ctx).Model(&domain.ClassTeacher{}).
		Joins("INNER JOIN classes c ON c.id = class_teachers.class_id AND c.kindergarten_id = ?", tc.KindergartenID).
		Where("class_teachers.user_id = ? AND class_teachers.status = ?", tc.UserID, 1).
		Order("class_teachers.id ASC").
		Pluck("class_teachers.class_id", &ids).Error
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 && tc.ClassID > 0 {
		ids = []uint64{tc.ClassID}
	}
	return ids, nil
}

// validateNoticeTargets 校验目标集非空且全部为任教班级。
// 目标为空 → 参数错；选到非任教班 → 越权（spec: 目标为空被拒 / 非任教班被拒）。
func (s *TeacherPortalService) validateNoticeTargets(ctx context.Context, tc *TeacherPortalContext, classIDs []uint64) error {
	if len(classIDs) == 0 {
		return ErrTeacherInvalidParams
	}
	taught, err := s.taughtClassIDs(ctx, tc)
	if err != nil {
		return err
	}
	taughtSet := make(map[uint64]struct{}, len(taught))
	for _, id := range taught {
		taughtSet[id] = struct{}{}
	}
	for _, id := range classIDs {
		if id == 0 {
			return ErrTeacherInvalidParams
		}
		if _, ok := taughtSet[id]; !ok {
			return ErrTeacherForbidden
		}
	}
	return nil
}

func (s *TeacherPortalService) ListNotices(ctx context.Context, tc *TeacherPortalContext, status *int8, category string, page, pageSize int) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	taught, err := s.taughtClassIDs(ctx, tc)
	if err != nil {
		return nil, err
	}
	// 可见性 = 目标集命中任一任教班级（EXISTS），多班通知不再碎在「当前班」过滤里
	db := s.notices.FilterByTargetHit(s.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ?", tc.KindergartenID), taught)
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
	ids := make([]uint64, 0, len(rows))
	for _, n := range rows {
		ids = append(ids, n.ID)
	}
	namesByNotice, err := s.notices.ClassNamesByNoticeIDs(ctx, tc.KindergartenID, ids)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	for _, n := range rows {
		readUsers, _, parentTotal, err := s.notices.NoticeReadStats(ctx, tc.KindergartenID, n.ID)
		if err != nil {
			return nil, err
		}
		readRate := "0%"
		if parentTotal > 0 {
			readRate = fmt.Sprintf("%d%%", readUsers*100/parentTotal)
		}
		list = append(list, map[string]any{
			"id": n.ID, "category": n.Category, "category_label": teacherPortalNoticeCategoryLabel(n.Category),
			"title": n.Title, "content": n.Content, "class_names": namesByNotice[n.ID], "need_confirm": n.NeedConfirm,
			"status": n.Status, "status_label": teacherPortalNoticeStatusLabel(n.Status),
			"read_count": readUsers, "total_count": parentTotal, "read_rate": readRate,
			"attachments_count": 0, "published_at": formatOptionalTime(n.PublishedAt), "created_at": n.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return map[string]any{"list": list, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

// classStudentCount 当前班在读学生数。⚠️ 仅用于出勤/任务等按班统计；
// 通知已读率的分母是「目标范围去重家长数」，走 NoticeRepo.NoticeReadStats。
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
	if err := s.validateNoticeTargets(ctx, tc, req.ClassIDs); err != nil {
		return nil, err
	}
	status := domain.NoticeStatusPublished
	now := time.Now()
	var scheduledAt *time.Time
	publishedAt := &now
	if req.ScheduleEnabled {
		t, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil || !t.After(now) {
			return nil, ErrTeacherInvalidParams
		}
		status = domain.NoticeStatusScheduled
		scheduledAt = &t
		publishedAt = nil
	}
	notice := &domain.Notice{
		KindergartenID: tc.KindergartenID, ClassID: req.ClassIDs[0], CreatorID: tc.UserID,
		Category: domain.NoticeCategory(req.Category), Title: strings.TrimSpace(req.Title), Content: strings.TrimSpace(req.Content),
		NeedConfirm: req.NeedConfirm, ScheduledAt: scheduledAt, PublishedAt: publishedAt, Status: status,
	}
	if notice.Category == "" {
		notice.Category = domain.NoticeCategoryOther
	}
	if err := s.db.WithContext(ctx).Create(notice).Error; err != nil {
		return nil, err
	}
	if err := s.notices.ReplaceNoticeTargets(ctx, tc.KindergartenID, notice.ID, req.ClassIDs); err != nil {
		return nil, err
	}
	return map[string]any{"id": notice.ID, "status": notice.Status, "status_label": teacherPortalNoticeStatusLabel(notice.Status), "created_at": notice.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

// getVisibleNotice 取通知，可见性 = 目标集命中任一任教班级。
func (s *TeacherPortalService) getVisibleNotice(ctx context.Context, tc *TeacherPortalContext, id uint64) (*domain.Notice, error) {
	taught, err := s.taughtClassIDs(ctx, tc)
	if err != nil {
		return nil, err
	}
	var n domain.Notice
	q := s.notices.FilterByTargetHit(s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ?", id, tc.KindergartenID), taught)
	if err := q.First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	return &n, nil
}

func (s *TeacherPortalService) GetNotice(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	n, err := s.getVisibleNotice(ctx, tc, id)
	if err != nil {
		return nil, err
	}
	readUsers, confirmed, parentTotal, err := s.notices.NoticeReadStats(ctx, tc.KindergartenID, n.ID)
	if err != nil {
		return nil, err
	}
	unread := parentTotal - readUsers
	if unread < 0 {
		unread = 0
	}
	namesByNotice, err := s.notices.ClassNamesByNoticeIDs(ctx, tc.KindergartenID, []uint64{n.ID})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id": n.ID, "category": n.Category, "category_label": teacherPortalNoticeCategoryLabel(n.Category),
		"title": n.Title, "content": n.Content, "class_names": namesByNotice[n.ID], "need_confirm": n.NeedConfirm,
		"status": n.Status, "status_label": teacherPortalNoticeStatusLabel(n.Status),
		"read_stats": map[string]any{
			"read_count": readUsers, "unread_count": unread, "confirmed_count": confirmed,
			"total_count": parentTotal, "read_rate": noticePercent(readUsers, parentTotal),
		},
		"read_count": readUsers, "total_count": parentTotal, "unread_count": unread,
		"published_at": formatOptionalTime(n.PublishedAt), "created_at": n.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func noticePercent(readUsers, parentTotal int64) string {
	if parentTotal <= 0 {
		return "0%"
	}
	return fmt.Sprintf("%d%%", readUsers*100/parentTotal)
}

func (s *TeacherPortalService) UpdateNotice(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherNoticeInput) error {
	if _, err := s.getVisibleNotice(ctx, tc, id); err != nil {
		return err
	}
	if err := s.validateNoticeTargets(ctx, tc, req.ClassIDs); err != nil {
		return err
	}
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
	res := s.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("id = ? AND kindergarten_id = ?", id, tc.KindergartenID).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	// 编辑目标集是显式重选（整体替换，不做增量）
	return s.notices.ReplaceNoticeTargets(ctx, tc.KindergartenID, id, req.ClassIDs)
}

func (s *TeacherPortalService) DeleteNotice(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	if _, err := s.getVisibleNotice(ctx, tc, id); err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ?", id, tc.KindergartenID).Delete(&domain.Notice{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) RevokeNotice(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	if _, err := s.getVisibleNotice(ctx, tc, id); err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("id = ? AND kindergarten_id = ?", id, tc.KindergartenID).Update("status", domain.NoticeStatusRevoked)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (s *TeacherPortalService) NoticeReadStatus(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.getVisibleNotice(ctx, tc, id); err != nil {
		return nil, err
	}
	readUsers, confirmed, parentTotal, err := s.notices.NoticeReadStats(ctx, tc.KindergartenID, id)
	if err != nil {
		return nil, err
	}
	unread := parentTotal - readUsers
	if unread < 0 {
		unread = 0
	}
	return map[string]any{
		"notice_id": id, "read_count": readUsers, "unread_count": unread,
		"total_count": parentTotal, "confirmed_count": confirmed,
		"unread_students": []any{},
	}, nil
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
	Date            string       `json:"date"`
	Content         string       `json:"content"`
	PerformanceTags []string     `json:"performance_tags"`
	SyncToFeed      bool         `json:"sync_to_feed"`
	TargetType      int8         `json:"target_type"`
	StudentIDs      []uint64     `json:"student_ids"`
	BreakfastRating int8         `json:"breakfast_rating"`
	LunchRating     int8         `json:"lunch_rating"`
	DinnerRating    int8         `json:"dinner_rating"`
	NapDuration     int          `json:"nap_duration"`
	SleepQuality    int8         `json:"sleep_quality"`
	MorningMood     int8         `json:"morning_mood"`
	AfternoonMood   int8         `json:"afternoon_mood"`
	Media           []MediaInput `json:"media"`
}

// MediaInput 成长记录媒体入参（素材管理基线 docs/media-management-design.md 的 L2 引用形态）。
// 客户端只传引用：FileID 指向 files 表的素材（URL 等物理属性由服务端反查，杜绝伪造）；
// ID 为编辑态保留的已有媒体行；RefDesc 是引用层描述（这次为什么用）。
//
// StudentIDs 是照片点名（design D8）：nil = 未提交该字段，不触碰既有点名（老客户端兼容）；
// 非 nil（含空数组）= 按提交重建该照片的点名（人工点名直接落 status=1）。
type MediaInput struct {
	ID         uint64   `json:"id"`
	FileID     uint64   `json:"file_id"`
	RefDesc    string   `json:"ref_desc"`
	StudentIDs []uint64 `json:"student_ids"`
}

// classStudentIDSet 本班在读学生 ID 集合（点名白名单与目标学生校验的公共来源）。
func classStudentIDSet(ctx context.Context, db *gorm.DB, kgID, classID uint64) (map[uint64]bool, error) {
	var ids []uint64
	if err := db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ? AND class_id = ? AND status = 1 AND deleted_at IS NULL", kgID, classID).
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	set := make(map[uint64]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

// growthStudentScope 本记录允许被点名的学生集合（design D9）：
// 全班记录 → 本班在读学生；个体记录 → 该记录关联学生（且必须都属于本班在读，否则视为越界）。
// 目标学生名单（req.StudentIDs）与照片点名共用此校验，避免两处各写一套范围判定。
func (s *TeacherPortalService) growthStudentScope(ctx context.Context, db *gorm.DB, tc *TeacherPortalContext, targetType int8, related []uint64) (map[uint64]bool, error) {
	classSet, err := classStudentIDSet(ctx, db, tc.KindergartenID, tc.ClassID)
	if err != nil {
		return nil, err
	}
	if targetType != 2 {
		return classSet, nil
	}
	scope := make(map[uint64]bool, len(related))
	for _, sid := range related {
		if sid == 0 || !classSet[sid] {
			return nil, ErrTeacherInvalidParams // 目标学生越界（非本班在读）
		}
		scope[sid] = true
	}
	return scope, nil
}

// validateMediaStudentIDs 纯判定：每张照片的点名对象必须落在记录范围内（D9 白名单），越界即拒绝。
// 0 与重复项被忽略（重复点名不叠加是幂等语义，不是错误）。
func validateMediaStudentIDs(items []MediaInput, scope map[uint64]bool) error {
	for _, it := range items {
		seen := make(map[uint64]bool, len(it.StudentIDs))
		for _, sid := range it.StudentIDs {
			if sid == 0 || seen[sid] {
				continue
			}
			if !scope[sid] {
				return ErrTeacherInvalidParams
			}
			seen[sid] = true
		}
	}
	return nil
}

// rebuildMediaStudents 按提交名单重建一张照片的点名关联（design D8：先删后插，随媒体同事务）。
// 未提交的既有关联（含 AI 建议）被移除；名单内关联以「人工确认」身份落库（is_auto=0/status=1）。
// 软删行占唯一索引，复活必须走 INSERT ... ON DUPLICATE KEY UPDATE（同 writeGrowthLogStudents）。
func rebuildMediaStudents(ctx context.Context, db *gorm.DB, mediaID uint64, studentIDs []uint64) error {
	ids := make([]uint64, 0, len(studentIDs))
	seen := make(map[uint64]bool, len(studentIDs))
	for _, sid := range studentIDs {
		if sid == 0 || seen[sid] {
			continue
		}
		seen[sid] = true
		ids = append(ids, sid)
	}
	del := db.WithContext(ctx).Where("media_id = ?", mediaID)
	if len(ids) > 0 {
		del = del.Where("student_id NOT IN ?", ids)
	}
	if err := del.Delete(&domain.GrowthMediaStudent{}).Error; err != nil {
		return err
	}
	for _, sid := range ids {
		if err := db.WithContext(ctx).Exec(
			"INSERT INTO growth_media_students (media_id, student_id, is_auto, status, confidence, created_at, updated_at, deleted_at) VALUES (?, ?, 0, ?, NULL, NOW(3), NOW(3), NULL) ON DUPLICATE KEY UPDATE is_auto = 0, status = VALUES(status), confidence = NULL, deleted_at = NULL, updated_at = NOW(3)",
			mediaID, sid, domain.MediaStudentStatusConfirmed).Error; err != nil {
			return err
		}
	}
	return nil
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
	if err := s.enrichGrowthLogsBatch(ctx, rows, list); err != nil {
		return nil, err
	}
	return map[string]any{"list": list, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

func teacherPortalGrowthMap(r domain.GrowthDailyLog) map[string]any {
	return map[string]any{"id": r.ID, "date": teacherPortalDateString(r.Date), "content": r.Content, "performance_tags": splitPerformanceTags(r.PerformanceTags), "sync_to_feed": r.SyncToFeed, "target_type": r.TargetType, "breakfast_rating": r.BreakfastRating, "lunch_rating": r.LunchRating, "dinner_rating": r.DinnerRating, "nap_duration": r.NapDuration, "sleep_quality": r.SleepQuality, "morning_mood": r.MorningMood, "afternoon_mood": r.AfternoonMood, "created_at": r.CreatedAt.UTC().Format(time.RFC3339)}
}

// enrichGrowthLogsBatch 为列表页记录批量补齐卡片渲染所需字段（发布者/照片预览/互动统计/相关孩子数）。
// 先收集本页 logIDs 再分批查询，避免逐条 N+1。
func (s *TeacherPortalService) enrichGrowthLogsBatch(ctx context.Context, rows []domain.GrowthDailyLog, list []map[string]any) error {
	if len(rows) == 0 {
		return nil
	}
	logIDs := make([]uint64, 0, len(rows))
	creatorIDs := make([]uint64, 0, len(rows))
	targetLogIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		logIDs = append(logIDs, r.ID)
		creatorIDs = append(creatorIDs, r.CreatorID)
		if r.TargetType == 2 {
			targetLogIDs = append(targetLogIDs, r.ID)
		}
	}
	creatorByID, err := s.growthCreatorMap(ctx, creatorIDs)
	if err != nil {
		return err
	}
	// 照片预览：只取缩略图，按记录聚合
	var medias []domain.GrowthMedia
	if err := s.db.WithContext(ctx).Where("log_id IN ?", logIDs).Order("sort_num ASC, id ASC").Find(&medias).Error; err != nil {
		return err
	}
	previewByLog := make(map[uint64][]map[string]any)
	for _, m := range medias {
		if m.LogID == nil {
			continue
		}
		previewByLog[*m.LogID] = append(previewByLog[*m.LogID], map[string]any{"thumb_url": m.ThumbURL, "type": m.FileType})
	}
	// 相关孩子数（仅个体记录）
	relatedCountByLog := make(map[uint64]int64)
	if len(targetLogIDs) > 0 {
		type countRow struct {
			LogID uint64
			Cnt   int64
		}
		var relRows []countRow
		if err := s.db.WithContext(ctx).Model(&domain.GrowthLogStudent{}).
			Where("log_id IN ?", targetLogIDs).
			Select("log_id, count(*) as cnt").Group("log_id").Scan(&relRows).Error; err != nil {
			return err
		}
		for _, c := range relRows {
			relatedCountByLog[c.LogID] = c.Cnt
		}
	}
	// 点赞数 / 评论数（一次聚合出 map）
	likeCounts, err := s.growthCountByLog(ctx, &domain.GrowthLike{}, logIDs)
	if err != nil {
		return err
	}
	commentCounts, err := s.growthCountByLog(ctx, &domain.GrowthComment{}, logIDs)
	if err != nil {
		return err
	}
	// 待确认建议数（D10 入口）：老师不必打开详情即可看到哪条记录有待处理建议
	pendingSuggByLog := make(map[uint64]int64)
	{
		type suggCountRow struct {
			LogID uint64
			Cnt   int64
		}
		var srows []suggCountRow
		if err := s.db.WithContext(ctx).Model(&domain.GrowthMediaStudent{}).
			Joins("INNER JOIN growth_media gm ON gm.id = growth_media_students.media_id AND gm.deleted_at IS NULL").
			Where("gm.log_id IN ? AND growth_media_students.status = ?", logIDs, domain.MediaStudentStatusSuggested).
			Select("gm.log_id AS log_id, COUNT(*) AS cnt").Group("gm.log_id").Scan(&srows).Error; err != nil {
			return err
		}
		for _, r := range srows {
			pendingSuggByLog[r.LogID] = r.Cnt
		}
	}
	for i := range list {
		r := rows[i]
		list[i]["creator"] = creatorByID[r.CreatorID]
		preview := previewByLog[r.ID]
		if preview == nil {
			preview = []map[string]any{}
		}
		list[i]["media_preview"] = preview
		list[i]["likes_count"] = likeCounts[r.ID]
		list[i]["comments_count"] = commentCounts[r.ID]
		list[i]["related_students_count"] = relatedCountByLog[r.ID]
		list[i]["pending_suggestions_count"] = pendingSuggByLog[r.ID]
	}
	return nil
}

// growthCreatorMap 批量取发布者展示信息。
func (s *TeacherPortalService) growthCreatorMap(ctx context.Context, creatorIDs []uint64) (map[uint64]map[string]any, error) {
	out := make(map[uint64]map[string]any)
	if len(creatorIDs) == 0 {
		return out, nil
	}
	var users []domain.User
	if err := s.db.WithContext(ctx).Where("id IN ?", creatorIDs).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = map[string]any{"id": u.ID, "nickname": u.Nickname, "avatar": u.Avatar}
	}
	return out, nil
}

// growthCountByLog 批量统计某关联表按 log_id 分组的条数。
func (s *TeacherPortalService) growthCountByLog(ctx context.Context, model any, logIDs []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64)
	type countRow struct {
		LogID uint64
		Cnt   int64
	}
	var rows []countRow
	if err := s.db.WithContext(ctx).Model(model).
		Where("log_id IN ?", logIDs).
		Select("log_id, count(*) as cnt").Group("log_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, c := range rows {
		out[c.LogID] = c.Cnt
	}
	return out, nil
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
	// 点名范围校验（D9）先于任何写入：越界直接拒绝，不留半成品记录
	scope, err := s.growthStudentScope(ctx, s.db.WithContext(ctx), tc, log.TargetType, req.StudentIDs)
	if err != nil {
		return nil, err
	}
	if err := validateMediaStudentIDs(req.Media, scope); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		return nil, err
	}
	if err := s.writeGrowthLogStudents(ctx, s.db.WithContext(ctx), log.ID, req.StudentIDs); err != nil {
		return nil, err
	}
	// 媒体入参：校验后按提交顺序建行绑定（创建路径本记录尚无既有媒体，"我的媒体"为空集）
	if len(req.Media) > 0 {
		if err := s.validateGrowthMediaInput(ctx, s.db.WithContext(ctx), log.ID, nil, req.Media); err != nil {
			return nil, err
		}
		filesByID, err := s.loadFilesByIDs(ctx, s.db.WithContext(ctx), mediaFileIDs(req.Media))
		if err != nil {
			return nil, err
		}
		if err := s.createGrowthMediaForFiles(ctx, s.db.WithContext(ctx), tc, log.ID, filesByID, req.Media); err != nil {
			return nil, err
		}
	}
	// 识别入队（发布路径零等待：仅插入任务行；识别在后台异步进行，见 design D2/D-异步）
	s.enqueueRecognitionForLog(ctx, tc, log.ID)
	return map[string]any{"id": log.ID, "created_at": log.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

// enqueueRecognitionForLog 把该记录当前绑定的照片全部入识别队列（media_id 唯一，幂等）。
func (s *TeacherPortalService) enqueueRecognitionForLog(ctx context.Context, tc *TeacherPortalContext, logID uint64) {
	if s.queue == nil {
		return
	}
	var ids []uint64
	if err := s.db.WithContext(ctx).Model(&domain.GrowthMedia{}).
		Where("log_id = ? AND file_type = ?", logID, "image").Pluck("id", &ids).Error; err != nil {
		log.L(ctx).Error("collect media for recognition failed", "log", logID, "err", err)
		return
	}
	s.enqueueRecognition(ctx, tc, ids)
}

// writeGrowthLogStudents 写入目标学生关联，语义为「确保这批 (log, student) 关联生效」：
// 已软删的同键行会被复活（deleted_at 置空），全新键插入。
// 唯一索引 idx_log_student 覆盖软删行，普通 Create 遇到复活场景会撞唯一键静默失败，
// 因此必须走 INSERT ... ON DUPLICATE KEY UPDATE。重复 sid 在此去重。
func (s *TeacherPortalService) writeGrowthLogStudents(ctx context.Context, db *gorm.DB, logID uint64, studentIDs []uint64) error {
	seen := make(map[uint64]bool, len(studentIDs))
	for _, sid := range studentIDs {
		if sid == 0 || seen[sid] {
			continue
		}
		seen[sid] = true
		if err := db.WithContext(ctx).Exec(
			"INSERT INTO growth_log_students (log_id, student_id, created_at, updated_at, deleted_at) VALUES (?, ?, NOW(3), NOW(3), NULL) ON DUPLICATE KEY UPDATE deleted_at = NULL, updated_at = NOW(3)",
			logID, sid).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *TeacherPortalService) GetGrowthLog(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var r domain.GrowthDailyLog
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	m := teacherPortalGrowthMap(r)
	// 相关孩子（个体记录）——计数与数组来自同一关联，保证一致。
	// 必须排除关联表的软删行：编辑改名单后旧关联是软删态，漏掉 deleted_at 过滤会读出历史学生。
	related := make([]map[string]any, 0)
	if r.TargetType == 2 {
		var students []domain.Student
		if err := s.db.WithContext(ctx).
			Joins("INNER JOIN growth_log_students gls ON gls.student_id = students.id AND gls.log_id = ? AND gls.deleted_at IS NULL", id).
			Where("students.kindergarten_id = ? AND students.class_id = ?", tc.KindergartenID, tc.ClassID).
			Find(&students).Error; err != nil {
			return nil, err
		}
		related = growthRelatedStudentItems(students)
	}
	m["related_students"] = related
	m["related_students_count"] = len(related)
	// 发布者
	creators, err := s.growthCreatorMap(ctx, []uint64{r.CreatorID})
	if err != nil {
		return nil, err
	}
	if c, ok := creators[r.CreatorID]; ok {
		m["creator"] = c
	}
	// 完整照片（详情页渲染）
	var medias []domain.GrowthMedia
	if err := s.db.WithContext(ctx).Where("log_id = ?", id).Order("sort_num ASC, id ASC").Find(&medias).Error; err != nil {
		return nil, err
	}
	mediaIDs := make([]uint64, 0, len(medias))
	for _, md := range medias {
		mediaIDs = append(mediaIDs, md.ID)
	}
	tags, err := s.loadMediaTagView(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	m["media"] = toGrowthMediaItems(medias, tags)
	// 互动统计
	likesCounts, err := s.growthCountByLog(ctx, &domain.GrowthLike{}, []uint64{id})
	if err != nil {
		return nil, err
	}
	commentCounts, err := s.growthCountByLog(ctx, &domain.GrowthComment{}, []uint64{id})
	if err != nil {
		return nil, err
	}
	m["likes_count"] = likesCounts[id]
	m["comments_count"] = commentCounts[id]
	return m, nil
}

// ReviewMediaSuggestion 建议确认/拒绝（design D8 的轻量端点）。
// 仅允许 0→1（确认，随即对家长端生效）/0→2（拒绝，不再提示）；重复同一动作幂等；
// 其他迁移（如已确认再拒绝）一律拒绝——那是编辑态改点名的职责，不走本端点。
func (s *TeacherPortalService) ReviewMediaSuggestion(ctx context.Context, tc *TeacherPortalContext, mediaID, studentID uint64, confirm bool) (map[string]any, error) {
	// 所有权：媒体必须属于本教师所在班级的某条已绑定记录
	var media domain.GrowthMedia
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ? AND log_id IS NOT NULL", mediaID, tc.KindergartenID, tc.ClassID).
		First(&media).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	var row domain.GrowthMediaStudent
	if err := s.db.WithContext(ctx).Where("media_id = ? AND student_id = ?", mediaID, studentID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	target := domain.MediaStudentStatusConfirmed
	if !confirm {
		target = domain.MediaStudentStatusRejected
	}
	current := domain.MediaStudentStatusConfirmed // NULL 为存量确认行（D6 兼容口径）
	if row.Status != nil {
		current = *row.Status
	}
	switch current {
	case target:
		// 幂等：已是目标状态，直接返回
	case domain.MediaStudentStatusSuggested:
		res := s.db.WithContext(ctx).Model(&domain.GrowthMediaStudent{}).
			Where("id = ? AND status = ?", row.ID, domain.MediaStudentStatusSuggested).
			Update("status", target)
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, ErrTeacherInvalidParams // 并发下已被处理，状态不再匹配
		}
	default:
		return nil, ErrTeacherInvalidParams // 已确认/已拒绝的行不在此端点流转
	}
	return map[string]any{"media_id": mediaID, "student_id": studentID, "status": target}, nil
}

// ConfirmHighConfidenceSuggestions 一键确认该记录的全部高置信建议（design D10）。
// 仅 confidence ≥ suggest_high 的建议行 0→1；低置信与已处理行不动。
// 存在意义：一次活动的合影会产出数十条建议，逐条确认把「AI 加速」变成负担。
func (s *TeacherPortalService) ConfirmHighConfidenceSuggestions(ctx context.Context, tc *TeacherPortalContext, logID uint64) (map[string]any, error) {
	var log domain.GrowthDailyLog
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", logID, tc.KindergartenID, tc.ClassID).
		First(&log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	var mediaIDs []uint64
	if err := s.db.WithContext(ctx).Model(&domain.GrowthMedia{}).Where("log_id = ?", logID).Pluck("id", &mediaIDs).Error; err != nil {
		return nil, err
	}
	if len(mediaIDs) == 0 {
		return map[string]any{"confirmed": 0}, nil
	}
	high := s.faceCfg.SuggestHigh
	if high <= 0 {
		high = 80
	}
	res := s.db.WithContext(ctx).Model(&domain.GrowthMediaStudent{}).
		Where("media_id IN ? AND status = ? AND confidence >= ?", mediaIDs, domain.MediaStudentStatusSuggested, float64(high)).
		Update("status", domain.MediaStudentStatusConfirmed)
	if res.Error != nil {
		return nil, res.Error
	}
	return map[string]any{"confirmed": res.RowsAffected}, nil
}

// presentStudentIDs 当日到场名单（D11 一键全班的名单来源）：
// 本班在读 ∧ **未被明确排除**——有已批准请假覆盖当日，或当日考勤被标记缺勤/请假。
// 无考勤数据时保守视为到场：家长端"漏标"（看不到孩子的照片）的成本远高于"误标"，系统向误标倾斜。
func (s *TeacherPortalService) presentStudentIDs(ctx context.Context, tc *TeacherPortalContext, date string) ([]uint64, error) {
	day := teacherPortalDay(time.Now())
	if strings.TrimSpace(date) != "" {
		if t, err := time.ParseInLocation("2006-01-02", date, time.Local); err == nil {
			day = teacherPortalDay(t)
		}
	}
	var ids []uint64
	if err := s.db.WithContext(ctx).Model(&domain.Student{}).
		Where("kindergarten_id = ? AND class_id = ? AND status = 1", tc.KindergartenID, tc.ClassID).
		Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	excluded := make(map[uint64]bool)
	// 明确不在（1）：当日已批准的请假
	var leaveIDs []uint64
	if err := s.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Where("student_id IN ? AND status = ? AND start_date <= ? AND end_date >= ?", ids, domain.LeaveStatusApproved, day, day).
		Pluck("student_id", &leaveIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range leaveIDs {
		excluded[id] = true
	}
	// 明确不在（2）：当日考勤被标记缺勤 / 请假
	var attIDs []uint64
	if err := s.db.WithContext(ctx).Model(&domain.Attendance{}).
		Where("student_id IN ? AND date = ? AND status IN ?", ids, day, []domain.AttendanceStatus{domain.AttendanceAbsent, domain.AttendanceLeave}).
		Pluck("student_id", &attIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range attIDs {
		excluded[id] = true
	}
	out := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if !excluded[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

// PresentStudents 当日到场名单（老师端一键全班的名单来源）。
func (s *TeacherPortalService) PresentStudents(ctx context.Context, tc *TeacherPortalContext, date string) (map[string]any, error) {
	ids, err := s.presentStudentIDs(ctx, tc, date)
	if err != nil {
		return nil, err
	}
	return map[string]any{"list": ids, "total": len(ids)}, nil
}

// ApplyClassRoster 一键「全班（当日到场）」（D11）：把该照片所属记录当日的到场学生
// 写为**确认点名**（并集语义：既有确认不受影响；**已拒绝的行不复活**——那是老师此前的明确表态）。
// 仅对全班记录开放（个体记录的照片本就属于那几位孩子）。
func (s *TeacherPortalService) ApplyClassRoster(ctx context.Context, tc *TeacherPortalContext, mediaID uint64) (map[string]any, error) {
	var media domain.GrowthMedia
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ? AND log_id IS NOT NULL", mediaID, tc.KindergartenID, tc.ClassID).
		First(&media).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	var log domain.GrowthDailyLog
	if err := s.db.WithContext(ctx).First(&log, *media.LogID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	if log.TargetType == 2 {
		return nil, ErrTeacherInvalidParams // 个体记录不做全班
	}
	ids, err := s.presentStudentIDs(ctx, tc, log.Date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	// 状态保护：已拒绝（2）的行保持拒绝——老师此前的表态优先于批量默认。
	// MySQL 的 ON DUPLICATE 赋值按书写顺序求值，后两行 IF 里的 status 已是本句更新后的值，
	// 故 status=2 时三个字段全部保持不变（含软删态）；其余情况转确认并降级为人工（is_auto=0）。
	for _, sid := range ids {
		if err := s.db.WithContext(ctx).Exec(
			"INSERT INTO growth_media_students (media_id, student_id, is_auto, status, confidence, created_at, updated_at, deleted_at) VALUES (?, ?, 0, ?, NULL, NOW(3), NOW(3), NULL) "+
				"ON DUPLICATE KEY UPDATE status = IF(status = ?, status, VALUES(status)), is_auto = IF(status = ?, is_auto, 0), deleted_at = IF(status = ?, deleted_at, NULL), updated_at = NOW(3)",
			mediaID, sid, domain.MediaStudentStatusConfirmed,
			domain.MediaStudentStatusRejected, domain.MediaStudentStatusRejected, domain.MediaStudentStatusRejected).Error; err != nil {
			return nil, err
		}
	}
	// 回读最终名单（确认态 = NULL 或 1）下发：客户端据此直接渲染，不自行猜测并集——
	// 客户端算不出「已拒绝不复活」，也看不见点名与到场名单之间的 TOCTOU 变化。
	var named []uint64
	if err := s.db.WithContext(ctx).Model(&domain.GrowthMediaStudent{}).
		Where("media_id = ? AND (status IS NULL OR status = ?)", mediaID, domain.MediaStudentStatusConfirmed).
		Order("student_id ASC").Pluck("student_id", &named).Error; err != nil {
		return nil, err
	}
	return map[string]any{"present": len(ids), "named": named}, nil
}

// GrowthTaggingStatus 识别状态查询（老师端轮询用）：按照片返回识别任务状态与当前建议。
// pending=true 表示仍有未完成任务，客户端据此决定是否继续轮询。
func (s *TeacherPortalService) GrowthTaggingStatus(ctx context.Context, tc *TeacherPortalContext, logID uint64) (map[string]any, error) {
	var log domain.GrowthDailyLog
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", logID, tc.KindergartenID, tc.ClassID).
		First(&log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	var medias []domain.GrowthMedia
	if err := s.db.WithContext(ctx).Where("log_id = ?", logID).Order("sort_num ASC, id ASC").Find(&medias).Error; err != nil {
		return nil, err
	}
	mediaIDs := make([]uint64, 0, len(medias))
	for _, md := range medias {
		mediaIDs = append(mediaIDs, md.ID)
	}
	type jobRow struct {
		MediaID   uint64
		Status    string
		LastError string
		FaceCount int
	}
	jobByMedia := make(map[uint64]jobRow)
	if len(mediaIDs) > 0 {
		var jobs []jobRow
		if err := s.db.WithContext(ctx).Model(&domain.RecognitionJob{}).
			Select("media_id, status, last_error, face_count").
			Where("media_id IN ?", mediaIDs).Scan(&jobs).Error; err != nil {
			return nil, err
		}
		for _, j := range jobs {
			jobByMedia[j.MediaID] = j
		}
	}
	tags, err := s.loadMediaTagView(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(medias))
	pending := false
	for _, md := range medias {
		recStatus := "none"
		lastErr := ""
		faceCount := 0
		if j, ok := jobByMedia[md.ID]; ok {
			recStatus = j.Status
			lastErr = j.LastError
			faceCount = j.FaceCount
		}
		if recStatus == domain.JobStatusPending || recStatus == domain.JobStatusProcessing {
			pending = true
		}
		sug := tags.suggestions[md.ID]
		if sug == nil {
			sug = []map[string]any{}
		}
		// face_count / is_group_photo（D11）：老师端据此把大合影照片呈现为「全班」一键而非逐人建议。
		// 分型在这里判定（阈值同源于队列配置），前端不硬编码阈值。
		items = append(items, map[string]any{
			"id": md.ID, "recognition_status": recStatus, "last_error": lastErr,
			"face_count": faceCount, "is_group_photo": faceCount >= s.groupPhotoThreshold(), "suggestions": sug,
		})
	}
	return map[string]any{"pending": pending, "media": items}, nil
}

// groupPhotoThreshold 大合影阈值（与识别队列同源配置，默认 6）。
func (s *TeacherPortalService) groupPhotoThreshold() int {
	if s.faceCfg.GroupPhotoMinFaces > 0 {
		return s.faceCfg.GroupPhotoMinFaces
	}
	return 6
}

// growthRelatedStudentItems 把相关孩子行转成契约展示形状。
// 详情响应里的 related_students 与 related_students_count 都直接由本函数的返回值派生，
// 保证「计数 = 数组长度」的一致性由构造保证，而非两处查询各算一次。
func growthRelatedStudentItems(students []domain.Student) []map[string]any {
	items := make([]map[string]any, 0, len(students))
	for _, st := range students {
		items = append(items, map[string]any{"id": st.ID, "name": st.Name, "avatar": st.Avatar})
	}
	return items
}

// toGrowthMediaItems 把媒体行转成老师端契约的展示形状。
// 点名与建议都随媒体项下发：student_ids 为已确认（含历史 NULL）点名；student_suggestions
// 为待老师处理的 AI 建议（带分级，老师端据此预勾高置信项）——建议绝不出现在家长端出参。
func toGrowthMediaItems(rows []domain.GrowthMedia, tags *mediaTagView) []map[string]any {
	items := make([]map[string]any, 0, len(rows))
	for _, m := range rows {
		confirmed := tags.confirmed[m.ID]
		if confirmed == nil {
			confirmed = []uint64{}
		}
		suggestions := tags.suggestions[m.ID]
		if suggestions == nil {
			suggestions = []map[string]any{}
		}
		items = append(items, map[string]any{
			"id": m.ID, "file_url": m.FileURL, "thumb_url": m.ThumbURL,
			"file_type": m.FileType, "file_name": m.FileName, "file_size": m.FileSize,
			"duration": m.Duration, "ref_desc": m.RefDesc,
			"student_ids": confirmed, "student_suggestions": suggestions,
		})
	}
	return items
}

// mediaTagView 一组照片的点名视图：确认项（student_ids）与待处理建议（student_suggestions）。
type mediaTagView struct {
	confirmed   map[uint64][]uint64
	suggestions map[uint64][]map[string]any
}

// loadMediaTagView 批量取照片点名（一次 JOIN 查询，避免逐张 N+1）。
// 状态口径（design D6）：NULL 或 1 = 确认；0 = 建议（带 confidence 与分级）；2 = 拒绝（不输出）。
func (s *TeacherPortalService) loadMediaTagView(ctx context.Context, mediaIDs []uint64) (*mediaTagView, error) {
	view := &mediaTagView{confirmed: map[uint64][]uint64{}, suggestions: map[uint64][]map[string]any{}}
	if len(mediaIDs) == 0 {
		return view, nil
	}
	type tagRow struct {
		MediaID    uint64
		StudentID  uint64
		Status     *int8
		Confidence *float64
		Name       string
		Avatar     string
	}
	var rows []tagRow
	if err := s.db.WithContext(ctx).Table("growth_media_students AS gms").
		Select("gms.media_id, gms.student_id, gms.status, gms.confidence, st.name, st.avatar").
		Joins("INNER JOIN students st ON st.id = gms.student_id AND st.deleted_at IS NULL").
		Where("gms.media_id IN ? AND gms.deleted_at IS NULL", mediaIDs).
		Order("gms.id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		switch {
		case r.Status == nil || *r.Status == domain.MediaStudentStatusConfirmed:
			view.confirmed[r.MediaID] = append(view.confirmed[r.MediaID], r.StudentID)
		case *r.Status == domain.MediaStudentStatusSuggested:
			conf := 0.0
			if r.Confidence != nil {
				conf = *r.Confidence
			}
			view.suggestions[r.MediaID] = append(view.suggestions[r.MediaID], map[string]any{
				"student_id": r.StudentID, "name": r.Name, "avatar": r.Avatar,
				"confidence": conf, "tier": s.suggestionTier(conf),
			})
		}
	}
	return view, nil
}

// suggestionTier 建议分级（design D6）：≥ suggest_high 为高置信（前端预勾），其余为低置信（仅提示）。
// 阈值来自 [face] 配置——分级判定在服务端，前端不硬编码阈值，校准时无需改客户端。
func (s *TeacherPortalService) suggestionTier(conf float64) string {
	high := s.faceCfg.SuggestHigh
	if high <= 0 {
		high = 80
	}
	if conf >= float64(high) {
		return "high"
	}
	return "low"
}

// ── 成长记录媒体入参（素材管理基线 L2，见 docs/media-management-design.md）──

// maxGrowthMediaPerLog 单条记录媒体数上限，与 /common/upload 批量上限同口径。
const maxGrowthMediaPerLog = 20

// loadFilesByIDs 批量取素材（L1 物理属性）；缺失的 id 不出现在返回值中，由调用方判缺。
func (s *TeacherPortalService) loadFilesByIDs(ctx context.Context, db *gorm.DB, ids []uint64) (map[uint64]domain.File, error) {
	out := make(map[uint64]domain.File, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.File
	if err := db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, f := range rows {
		out[f.ID] = f
	}
	return out, nil
}

// createGrowthMediaForFiles 以 files 行为准写入引用行（L2，含 L1 物理属性的快照），
// 支撑「先传后挂」：上传产生 files 行，发布时在此绑定 log_id。
// 必须按 **items（提交顺序）** 遍历而非 files（由 map 产出、顺序随机），否则入库顺序不可控。
func (s *TeacherPortalService) createGrowthMediaForFiles(ctx context.Context, db *gorm.DB, tc *TeacherPortalContext, logID uint64, filesByID map[uint64]domain.File, items []MediaInput) error {
	for i, it := range items {
		f, ok := filesByID[it.FileID]
		if !ok {
			continue // 该校验已在前置完成；防御性跳过
		}
		row := &domain.GrowthMedia{
			KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, LogID: &logID,
			UploaderID: tc.UserID, FileURL: f.FileURL, ThumbURL: f.ThumbURL,
			FileType: f.FileType, FileName: f.FileName, FileSize: f.FileSize,
			RefDesc: it.RefDesc, SortNum: i,
		}
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			return err
		}
		if it.StudentIDs != nil {
			if err := rebuildMediaStudents(ctx, db, row.ID, it.StudentIDs); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateGrowthMediaPart 纯判定部分（可单测）：空数组合法；上限 20；
// 编辑态 media.id 必须属于本记录当前媒体（mineIDs）。
// 返回待校验的 file_id 集合；DB 侧的存在性与占用校验由调用方完成。
func validateGrowthMediaPart(items []MediaInput, mineIDs map[uint64]bool) (map[uint64]bool, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if len(items) > maxGrowthMediaPerLog {
		return nil, ErrTeacherInvalidParams
	}
	fileIDSet := make(map[uint64]bool, len(items))
	for _, it := range items {
		if it.ID != 0 && !mineIDs[it.ID] {
			return nil, ErrTeacherInvalidParams // 编辑态引用了不属于本记录的媒体
		}
		if it.FileID != 0 {
			fileIDSet[it.FileID] = true
		}
	}
	return fileIDSet, nil
}

// validateGrowthMediaInput 校验提交的媒体列表。
// 规则（specs/growth-media-pipeline 逐条）：纯判定见 validateGrowthMediaPart；
// 另需 file_id 存在于 files 且未被其他记录占用。
func (s *TeacherPortalService) validateGrowthMediaInput(ctx context.Context, db *gorm.DB, logID uint64, mine []domain.GrowthMedia, items []MediaInput) error {
	mineByID := make(map[uint64]bool, len(mine))
	for _, m := range mine {
		mineByID[m.ID] = true
	}
	fileIDSet, err := validateGrowthMediaPart(items, mineByID)
	if err != nil || len(fileIDSet) == 0 {
		return err
	}
	for fid := range fileIDSet {
		var exists int64
		if err := db.WithContext(ctx).Model(&domain.File{}).Where("id = ?", fid).Count(&exists).Error; err != nil {
			return err
		}
		if exists == 0 {
			return ErrTeacherInvalidParams // 素材不存在
		}
		var used int64
		if err := db.WithContext(ctx).Model(&domain.GrowthMedia{}).
			Where("file_url = (?) AND log_id IS NOT NULL AND log_id <> ?", db.Model(&domain.File{}).Select("file_url").Where("id = ?", fid), logID).
			Count(&used).Error; err != nil {
			return err
		}
		if used > 0 {
			return ErrTeacherInvalidParams // 素材已被其他记录引用
		}
	}
	return nil
}

// updateGrowthMediaForLog 编辑态「全量替换」：未提交的既有媒体解除绑定（log_id=NULL，
// 不删文件行——基线 §4 解除绑定与删除分离）；保留项更新 ref_desc/sort_num；新素材建行。
func (s *TeacherPortalService) updateGrowthMediaForLog(ctx context.Context, db *gorm.DB, tc *TeacherPortalContext, logID uint64, items []MediaInput, mine []domain.GrowthMedia, filesByID map[uint64]domain.File) error {
	keep := make(map[uint64]bool, len(items))
	for _, it := range items {
		if it.ID != 0 {
			keep[it.ID] = true
		}
	}
	// 不在提交清单中的既有媒体：解除绑定（保留文件行）
	for _, m := range mine {
		if !keep[m.ID] {
			if err := db.WithContext(ctx).Model(&domain.GrowthMedia{}).Where("id = ?", m.ID).Update("log_id", nil).Error; err != nil {
				return err
			}
		}
	}
	// 保留项：更新引用层描述；提交了点名的（非 nil）按名单重建点名关联
	for i, it := range items {
		if it.ID == 0 {
			continue
		}
		if err := db.WithContext(ctx).Model(&domain.GrowthMedia{}).Where("id = ? AND log_id = ?", it.ID, logID).
			Updates(map[string]any{"ref_desc": it.RefDesc, "sort_num": i}).Error; err != nil {
			return err
		}
		if it.StudentIDs != nil {
			if err := rebuildMediaStudents(ctx, db, it.ID, it.StudentIDs); err != nil {
				return err
			}
		}
	}
	// 新素材：按提交顺序建行绑定
	for i, it := range items {
		if it.ID != 0 || it.FileID == 0 {
			continue
		}
		f, ok := filesByID[it.FileID]
		if !ok {
			continue // 前置校验已保证存在；防御性跳过
		}
		row := &domain.GrowthMedia{
			KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, LogID: &logID,
			UploaderID: tc.UserID, FileURL: f.FileURL, ThumbURL: f.ThumbURL,
			FileType: f.FileType, FileName: f.FileName, FileSize: f.FileSize,
			RefDesc: it.RefDesc, SortNum: i,
		}
		if err := db.WithContext(ctx).Create(row).Error; err != nil {
			return err
		}
		if it.StudentIDs != nil {
			if err := rebuildMediaStudents(ctx, db, row.ID, it.StudentIDs); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *TeacherPortalService) UpdateGrowthLog(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherGrowthInput) error {
	// 个体记录必须至少指定一位学生（与创建路径的校验一致）
	if req.TargetType == 2 && len(req.StudentIDs) == 0 {
		return ErrTeacherInvalidParams
	}
	// 主表更新与目标学生关联重建必须同成败：否则会出现「记录已改为个体但关联为空」或反之。
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&domain.GrowthDailyLog{}).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Updates(map[string]any{"content": req.Content, "performance_tags": strings.Join(req.PerformanceTags, ","), "sync_to_feed": req.SyncToFeed, "target_type": req.TargetType, "breakfast_rating": req.BreakfastRating, "lunch_rating": req.LunchRating, "dinner_rating": req.DinnerRating, "nap_duration": req.NapDuration, "sleep_quality": req.SleepQuality, "morning_mood": req.MorningMood, "afternoon_mood": req.AfternoonMood})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrTeacherNotFound
		}
		// 目标学生关联按提交名单重建（先删后插保证幂等；全班记录清空既有个体关联）
		if err := tx.Where("log_id = ?", id).Delete(&domain.GrowthLogStudent{}).Error; err != nil {
			return err
		}
		if req.TargetType == 2 {
			if err := s.writeGrowthLogStudents(ctx, tx, id, req.StudentIDs); err != nil {
				return err
			}
		}
		// 点名范围校验（D9）：目标学生与照片点名都必须落在本记录涉及的学生范围内
		scope, err := s.growthStudentScope(ctx, tx, tc, req.TargetType, req.StudentIDs)
		if err != nil {
			return err
		}
		if err := validateMediaStudentIDs(req.Media, scope); err != nil {
			return err
		}
		// 媒体「全量替换」：与记录主表更新同事务成败（基线 §3/§4）
		var mine []domain.GrowthMedia
		if err := tx.WithContext(ctx).Where("log_id = ?", id).Find(&mine).Error; err != nil {
			return err
		}
		if err := s.validateGrowthMediaInput(ctx, tx, id, mine, req.Media); err != nil {
			return err
		}
		filesByID, err := s.loadFilesByIDs(ctx, tx, mediaFileIDs(req.Media))
		if err != nil {
			return err
		}
		if err := s.updateGrowthMediaForLog(ctx, tx, tc, id, req.Media, mine, filesByID); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 事务成功后触发识别入队（media_id 唯一，已入队的照片不会重复建任务）
	s.enqueueRecognitionForLog(ctx, tc, id)
	return nil
}

// mediaFileIDs 收集提交清单中所有非零 file_id（去重）。
func mediaFileIDs(items []MediaInput) []uint64 {
	seen := make(map[uint64]bool, len(items))
	var ids []uint64
	for _, it := range items {
		if it.FileID != 0 && !seen[it.FileID] {
			seen[it.FileID] = true
			ids = append(ids, it.FileID)
		}
	}
	return ids
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

// growthLogRow 取记录行，施加与 GetGrowthLog 相同的可见范围（园所 + 班级）。
// 评论列表只需要记录的归属与 target_type 来解析家长称呼，不必拉全量媒体与点名。
func (s *TeacherPortalService) growthLogRow(ctx context.Context, tc *TeacherPortalContext, id uint64) (*domain.GrowthDailyLog, error) {
	var r domain.GrowthDailyLog
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).
		First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	return &r, nil
}

// growthCommentRow 取评论，并沿 log 校验其所属记录在本教师班级内。
func (s *TeacherPortalService) growthCommentRow(ctx context.Context, tc *TeacherPortalContext, id uint64) (*domain.GrowthComment, error) {
	var c domain.GrowthComment
	if err := s.db.WithContext(ctx).
		Joins("JOIN growth_daily_logs ON growth_daily_logs.id = growth_comments.log_id").
		Where("growth_comments.id = ? AND growth_daily_logs.kindergarten_id = ? AND growth_daily_logs.class_id = ?", id, tc.KindergartenID, tc.ClassID).
		Where("growth_daily_logs.deleted_at IS NULL").
		Take(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	return &c, nil
}

// growthCommentLikeStats 统计一批评论的点赞数，并标出当前用户赞过哪些。
func (s *TeacherPortalService) growthCommentLikeStats(
	ctx context.Context, commentIDs []uint64, userID uint64,
) (map[uint64]int64, map[uint64]bool, error) {
	counts := make(map[uint64]int64, len(commentIDs))
	liked := make(map[uint64]bool, len(commentIDs))
	if len(commentIDs) == 0 {
		return counts, liked, nil
	}
	var rows []struct {
		CommentID uint64
		Cnt       int64
	}
	if err := s.db.WithContext(ctx).Model(&domain.GrowthCommentLike{}).
		Select("comment_id, COUNT(*) AS cnt").Where("comment_id IN ?", commentIDs).
		Group("comment_id").Scan(&rows).Error; err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		counts[r.CommentID] = r.Cnt
	}
	var mine []uint64
	if err := s.db.WithContext(ctx).Model(&domain.GrowthCommentLike{}).
		Where("comment_id IN ? AND user_id = ?", commentIDs, userID).
		Pluck("comment_id", &mine).Error; err != nil {
		return nil, nil, err
	}
	for _, cid := range mine {
		liked[cid] = true
	}
	return counts, liked, nil
}

func (s *TeacherPortalService) GrowthComments(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	log, err := s.growthLogRow(ctx, tc, id)
	if err != nil {
		return nil, err
	}
	var rows []domain.GrowthComment
	if err := s.db.WithContext(ctx).Preload("User").Where("log_id = ?", id).
		Order("created_at ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uint64, 0, len(rows))
	commentIDs := make([]uint64, 0, len(rows))
	seenUser := make(map[uint64]bool, len(rows))
	for _, r := range rows {
		commentIDs = append(commentIDs, r.ID)
		if !seenUser[r.UserID] {
			seenUser[r.UserID] = true
			userIDs = append(userIDs, r.UserID)
		}
	}
	extras, err := commentAuthorExtras(ctx, s.db, tc.KindergartenID, log, userIDs)
	if err != nil {
		return nil, err
	}
	likeCounts, likedByMe, err := s.growthCommentLikeStats(ctx, commentIDs, tc.UserID)
	if err != nil {
		return nil, err
	}

	build := func(r domain.GrowthComment) map[string]any {
		u := map[string]any{}
		if r.User != nil {
			u = map[string]any{"id": r.User.ID, "nickname": r.User.Nickname, "avatar": r.User.Avatar}
		}
		// 老师/员工没有 StudentParent 行：按契约省略 relation / student_name，不报错。
		if ex, ok := extras[r.UserID]; ok && ex.Relation != "" {
			u["relation"] = ex.Relation
			if ex.StudentName != "" {
				u["student_name"] = ex.StudentName
			}
		}
		return map[string]any{
			"id": r.ID, "user": u, "content": r.Content, "parent_id": r.ParentID,
			"likes_count": likeCounts[r.ID], "liked": likedByMe[r.ID],
			"created_at": r.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	// total 含回复（契约）；list 只放顶层，回复在各自的 replies[] 里。
	return map[string]any{"list": buildCommentTree(rows, build), "total": len(rows)}, nil
}

func (s *TeacherPortalService) CreateGrowthComment(ctx context.Context, tc *TeacherPortalContext, id uint64, content string, parentID *uint64) (map[string]any, error) {
	if strings.TrimSpace(content) == "" {
		return nil, ErrTeacherInvalidParams
	}
	if _, err := s.growthLogRow(ctx, tc, id); err != nil {
		return nil, err
	}
	row := &domain.GrowthComment{LogID: id, UserID: tc.UserID, Content: strings.TrimSpace(content)}
	if parentID != nil {
		// 只允许回复本记录下的顶层评论：契约承诺只有一层回复，多级回复在渲染端会被摊平，
		// 与其让家长看到错位，不如在写入时拒绝。
		var parent domain.GrowthComment
		if err := s.db.WithContext(ctx).Where("id = ?", *parentID).Take(&parent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrTeacherInvalidParams
			}
			return nil, err
		}
		if parent.LogID != id || parent.ParentID != nil {
			return nil, ErrTeacherInvalidParams
		}
		row.ParentID = parentID
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

// LikeGrowthComment 点赞一条评论。重复赞幂等：不报错、不重复计数。
func (s *TeacherPortalService) LikeGrowthComment(ctx context.Context, tc *TeacherPortalContext, commentID uint64) (map[string]any, error) {
	if _, err := s.growthCommentRow(ctx, tc, commentID); err != nil {
		return nil, err
	}
	// 与成长记录点赞同因同解（B04）：先 Unscoped 查，软删行复活，查不到才插入。
	// 并发双击时两个请求都会走插入分支，唯一索引 uk_comment_like_user 只允许一个成功——
	// 点赞本身幂等，撞键说明对方已写入，不能把错误抛给客户端。
	var existing domain.GrowthCommentLike
	err := s.db.WithContext(ctx).Unscoped().
		Where("comment_id = ? AND user_id = ?", commentID, tc.UserID).Take(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		if e := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
			Create(&domain.GrowthCommentLike{CommentID: commentID, UserID: tc.UserID}).Error; e != nil {
			return nil, e
		}
	case err != nil:
		return nil, err
	case existing.DeletedAt.Valid:
		if e := s.db.WithContext(ctx).Unscoped().Model(&existing).
			Update("deleted_at", nil).Error; e != nil {
			return nil, e
		}
	}
	return s.growthCommentLikeResult(ctx, commentID, tc.UserID)
}

// UnlikeGrowthComment 取消点赞。未点过赞时同样成功（幂等）。
func (s *TeacherPortalService) UnlikeGrowthComment(ctx context.Context, tc *TeacherPortalContext, commentID uint64) (map[string]any, error) {
	if _, err := s.growthCommentRow(ctx, tc, commentID); err != nil {
		return nil, err
	}
	// 硬删：软删行会占住 uk_comment_like_user，导致再次点赞撞键（见 LikeGrowthComment）。
	if err := s.db.WithContext(ctx).Unscoped().
		Where("comment_id = ? AND user_id = ?", commentID, tc.UserID).
		Delete(&domain.GrowthCommentLike{}).Error; err != nil {
		return nil, err
	}
	return s.growthCommentLikeResult(ctx, commentID, tc.UserID)
}

func (s *TeacherPortalService) growthCommentLikeResult(ctx context.Context, commentID, userID uint64) (map[string]any, error) {
	var count, mine int64
	if err := s.db.WithContext(ctx).Model(&domain.GrowthCommentLike{}).
		Where("comment_id = ?", commentID).Count(&count).Error; err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&domain.GrowthCommentLike{}).
		Where("comment_id = ? AND user_id = ?", commentID, userID).Count(&mine).Error; err != nil {
		return nil, err
	}
	return map[string]any{"liked": mine > 0, "likes_count": count}, nil
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
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil {
		return nil, err
	}
	// 与家长端同因同解：growth_likes 软删除，而唯一索引 idx_log_user 不含
	// deleted_at —— 软删行会占住唯一键，直接 Create 撞键。先 Unscoped 查，
	// 软删的复活，查不到才插入；写入错误必须向上抛（不能静默返回 is_liked:true）。
	var existing domain.GrowthLike
	err := s.db.WithContext(ctx).Unscoped().
		Where("log_id = ? AND user_id = ?", id, tc.UserID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		if e := s.db.WithContext(ctx).Create(&domain.GrowthLike{LogID: id, UserID: tc.UserID}).Error; e != nil {
			return nil, e
		}
	case err != nil:
		return nil, err
	case existing.DeletedAt.Valid:
		if e := s.db.WithContext(ctx).Unscoped().Model(&existing).
			Update("deleted_at", nil).Error; e != nil {
			return nil, e
		}
	}
	var total int64
	if e := s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", id).Count(&total).Error; e != nil {
		return nil, e
	}
	return map[string]any{"id": id, "is_liked": true, "likes_count": total}, nil
}

func (s *TeacherPortalService) UnlikeGrowth(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil {
		return nil, err
	}
	// 硬删：点赞没有审计价值；留软删行只会占住 idx_log_user 唯一键（见 LikeGrowth）。
	if e := s.db.WithContext(ctx).Unscoped().
		Where("log_id = ? AND user_id = ?", id, tc.UserID).
		Delete(&domain.GrowthLike{}).Error; e != nil {
		return nil, e
	}
	var total int64
	if e := s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", id).Count(&total).Error; e != nil {
		return nil, e
	}
	return map[string]any{"id": id, "is_liked": false, "likes_count": total}, nil
}

func (s *TeacherPortalService) GrowthRelatedStudents(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	if _, err := s.GetGrowthLog(ctx, tc, id); err != nil {
		return nil, err
	}
	var rows []domain.Student
	err := s.db.WithContext(ctx).Joins("INNER JOIN growth_log_students gls ON gls.student_id = students.id AND gls.log_id = ? AND gls.deleted_at IS NULL", id).Where("students.kindergarten_id = ? AND students.class_id = ?", tc.KindergartenID, tc.ClassID).Find(&rows).Error
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

// PickupOverview 今日接送统计。
//
// 口径按「最后一次转移方向」派生（B1），而非「当日是否有某类记录」：
//   - not_arrived  当日无任何记录
//   - picked_up    最后一次是接走（已离园）
//   - not_picked_up 最后一次是送入（仍在园）
//   - arrived      = picked_up + not_picked_up（当日到过校）
//   - overdue_not_picked 仍在园且已过班级放学时刻，按学生计
func (s *TeacherPortalService) PickupOverview(ctx context.Context, tc *TeacherPortalContext) (map[string]any, error) {
	today := teacherPortalDay(time.Now())
	total := s.classStudentCount(ctx, tc)

	var abnormal int64
	_ = s.db.WithContext(ctx).Model(&domain.PickupRecord{}).
		Where("kindergarten_id = ? AND class_id = ? AND date = ? AND is_abnormal = ?",
			tc.KindergartenID, tc.ClassID, today, true).
		Count(&abnormal).Error

	// 全班在读学生 → 当日最后一条
	studentIDs := s.classStudentIDs(ctx, tc)
	lastMap, err := s.pickup.LastRecordsOfDay(ctx, studentIDs, today)
	if err != nil {
		return nil, err
	}

	deadline := s.classPickupDeadline(ctx, tc.ClassID)
	resolvedDeadline := ResolvePickupDeadline(deadline)
	var pickedUp, notPickedUp int64
	for _, id := range studentIDs {
		lr, ok := lastMap[id]
		if !ok {
			continue // 未到园
		}
		if lr.Type == int8(domain.PickupTypePickUp) {
			pickedUp++
			continue
		}
		notPickedUp++
	}

	// overdue_not_picked 是累计口径：放学时刻点上仍在园即计一次，
	// 之后被接走也不清零（spec TC-NEW-PKU-111）。与 is_overdue 的实时高亮不同。
	everOverdue, err := s.pickup.EverOverdueStudentIDs(ctx, studentIDs, today, resolvedDeadline, time.Now())
	if err != nil {
		return nil, err
	}

	arrived := pickedUp + notPickedUp
	return map[string]any{
		"date":               teacherPortalDateString(today),
		"total":              total,
		"arrived":            arrived,
		"not_arrived":        total - arrived,
		"picked_up":          pickedUp,
		"not_picked_up":      notPickedUp,
		"overdue_not_picked": int64(len(everOverdue)),
		"abnormal":           abnormal,
	}, nil
}

// classStudentIDs 取本班全部在读学生 ID（升序，供展示/遍历）。
func (s *TeacherPortalService) classStudentIDs(ctx context.Context, tc *TeacherPortalContext) []uint64 {
	set, err := classStudentIDSet(ctx, s.db, tc.KindergartenID, tc.ClassID)
	if err != nil {
		return nil
	}
	ids := make([]uint64, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// PickupStudents 本班学生接送状态列表。
//
// 状态口径与 PickupOverview 一致（看最后一条方向）：
//   - not_arrived   当日无记录
//   - picked_up     最后一次是接走
//   - not_picked_up 最后一次是送入（仍在园）
//   - arrived       当日到过校（有过送入）
//
// 每人额外输出 is_overdue / pickup_deadline / last_remark / is_authorized_override，
// 供前端滞留高亮与理由/例外可见（接口粒度：仅列表渲染所需）。
func (s *TeacherPortalService) PickupStudents(ctx context.Context, tc *TeacherPortalContext, status string) (map[string]any, error) {
	today := teacherPortalDay(time.Now())
	now := time.Now()
	deadline := s.classPickupDeadline(ctx, tc.ClassID)

	allIDs := s.classStudentIDs(ctx, tc)
	if len(allIDs) == 0 {
		return map[string]any{"list": []any{}, "total": 0}, nil
	}

	lastMap, err := s.pickup.LastRecordsOfDay(ctx, allIDs, today)
	if err != nil {
		return nil, err
	}

	// 依 status 过滤（口径与 Overview 一致）
	filtered := make([]uint64, 0, len(allIDs))
	for _, id := range allIDs {
		lr, has := lastMap[id]
		switch status {
		case "arrived":
			if has && lr.HasDropOff {
				filtered = append(filtered, id)
			}
		case "picked_up":
			if has && lr.Type == int8(domain.PickupTypePickUp) {
				filtered = append(filtered, id)
			}
		case "not_picked_up":
			if has && lr.Type == int8(domain.PickupTypeDropOff) {
				filtered = append(filtered, id)
			}
		case "not_arrived":
			if !has {
				filtered = append(filtered, id)
			}
		default:
			filtered = append(filtered, id) // 不传 status = 全部
		}
	}

	// 基本信息
	type studentRow struct {
		ID     uint64 `gorm:"column:id"`
		Name   string `gorm:"column:name"`
		Avatar string `gorm:"column:avatar"`
	}
	var rows []studentRow
	if len(filtered) > 0 {
		if err := s.db.WithContext(ctx).Table("students").
			Select("id, name, COALESCE(avatar,'') AS avatar").
			Where("id IN ?", filtered).Order("id ASC").Scan(&rows).Error; err != nil {
			return nil, err
		}
	}

	// 当日首次送入时刻（到达时间）
	arriveMap := make(map[uint64]string)
	var arriveRows []struct {
		StudentID uint64    `gorm:"column:student_id"`
		FirstDrop time.Time `gorm:"column:first_drop"`
	}
	if len(filtered) > 0 {
		_ = s.db.WithContext(ctx).Raw(
			`SELECT student_id, MIN(recorded_at) AS first_drop FROM pickup_records
			 WHERE student_id IN ? AND deleted_at IS NULL AND date = ? AND type = 1
			 GROUP BY student_id`, filtered, today).Scan(&arriveRows).Error
	}
	for _, a := range arriveRows {
		arriveMap[a.StudentID] = a.FirstDrop.Format("15:04")
	}

	// 有效授权接送人姓名
	authMap := make(map[uint64]string)
	var authRows []struct {
		StudentID uint64 `gorm:"column:student_id"`
		Names     string `gorm:"column:names"`
	}
	if len(filtered) > 0 {
		_ = s.db.WithContext(ctx).Raw(
			`SELECT student_id, GROUP_CONCAT(name SEPARATOR '、') AS names
			 FROM pickup_authorizations
			 WHERE student_id IN ? AND deleted_at IS NULL AND status = 1 AND review_status = 1
			 GROUP BY student_id`, filtered).Scan(&authRows).Error
	}
	for _, a := range authRows {
		authMap[a.StudentID] = a.Names
	}

	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		lr, has := lastMap[r.ID]

		ps := "not_arrived"
		if has {
			if lr.Type == int8(domain.PickupTypePickUp) {
				ps = "picked_up"
			} else {
				ps = "not_picked_up"
			}
		}

		lastRemark := ""
		override := false
		if has {
			lastRemark = lr.Remark
			override = lr.IsAuthorizedOverride
		}

		overdue := false
		if has && lr.Type == int8(domain.PickupTypeDropOff) {
			overdue = IsOverdueNotPicked(
				&domain.PickupRecord{Type: domain.PickupType(lr.Type)}, now, deadline)
		}

		ap := authMap[r.ID]
		if ap == "" {
			ap = "暂无"
		}

		list = append(list, map[string]any{
			"student_id":             r.ID,
			"name":                   r.Name,
			"avatar":                 r.Avatar,
			"authorized_persons":     ap,
			"arrive_time":            arriveMap[r.ID],
			"pickup_status":          ps,
			"is_overdue":             overdue,
			"pickup_deadline":        ResolvePickupDeadline(deadline),
			"last_remark":            lastRemark,
			"is_authorized_override": override,
		})
	}

	return map[string]any{"list": list, "total": len(list)}, nil
}

type TeacherPickupInput struct {
	// StudentID 仅手动登记需要；扫码路径由凭证自解释，忽略此字段
	StudentID      uint64 `json:"student_id"`
	Type           int8   `json:"type"`
	PickupName     string `json:"pickup_name"`
	PickupRelation string `json:"pickup_relation"`
	Remark         string `json:"remark"`
	Code           string `json:"code"`
	QRCode         string `json:"qr_code"`
	// AuthorizationID 手动登记时从授权人列表选人；标例外时为 nil
	AuthorizationID *uint64 `json:"authorization_id"`
	// IsAuthorizedOverride 标记本次为授权例外（来接者不在有效授权集合内）。
	// 例外不阻断登记，但必须显式标记，使例外可观测统计
	IsAuthorizedOverride bool `json:"is_authorized_override"`
	// OverrideName 例外情形下的来接者姓名
	OverrideName string `json:"override_name"`
}

// CreatePickupRecord 登记一次托管转移。
//
// 校验链顺序（design D1/D5）：
//  1. 凭证自解释（扫码）或显式人指定（手动）
//  2. 学生归属本班本园
//  3. 连续同向拒绝 —— 托管序列必须「送入—接走」交替
//  4. 接走须先有送入
//  5. 理由必填判定（重复接走 / 提前接走）
//  6. 授权二次校验 —— 接走人仍在有效授权集合内
//  7. 落库（PickupPersonID / 例外标志）
//  8. 凭证即焚
//  9. 通知家长（失败仅记日志，不回滚登记）
func (s *TeacherPortalService) CreatePickupRecord(ctx context.Context, tc *TeacherPortalContext, req TeacherPickupInput, method domain.PickupMethod) (map[string]any, error) {
	// 兼容 code 和 qr_code 两种 key
	if req.Code == "" {
		req.Code = req.QRCode
	}

	now := time.Now()
	today := teacherPortalDay(now)
	pickupType := domain.PickupType(req.Type)
	if req.Type == 0 {
		// 扫码只用于接走（送入不走码，见丙）；手动未指定时默认送入
		if method == domain.PickupMethodScan {
			pickupType = domain.PickupTypePickUp
		} else {
			pickupType = domain.PickupTypeDropOff
		}
		req.Type = int8(pickupType)
	}

	// ---- 1. 识别「谁」：凭证自解释 或 手动指定 ----
	var (
		studentID       uint64
		authorizationID *uint64
		pickupName      = req.PickupName
		pickupRelation  = req.PickupRelation
		pickupAvatar    string
		isOverride      bool
		overrideName    string
		qrToBurn        *domain.PickupQRCode
	)

	switch method {
	case domain.PickupMethodScan:
		// 扫码：前端只传凭证，学生与接走人由后端解出
		if req.Code == "" {
			return nil, ErrTeacherInvalidParams
		}
		qr, err := s.pickup.FindQRCodeByCode(ctx, req.Code)
		if err != nil {
			return nil, err
		}
		if qr == nil || !qr.IsActive {
			return nil, ErrTeacherInvalidQRCode
		}
		if qr.ExpireAt.Before(now) {
			return nil, ErrTeacherInvalidQRCode
		}
		studentID = qr.StudentID
		authID := qr.AuthorizationID
		authorizationID = &authID
		qrToBurn = qr
		if qr.Authorization != nil {
			pickupName = qr.Authorization.Name
			pickupRelation = qr.Authorization.Relation
			pickupAvatar = qr.Authorization.Avatar
		}

	default: // 手动登记
		if req.StudentID == 0 {
			return nil, ErrTeacherInvalidParams
		}
		studentID = req.StudentID
		// 送入只作到场标记：不记送托人、不要理由（丙，见 proposal 分歧清单）。
		// 因此只有接走才要求「归因到人」——要么列表选人，要么标例外。
		if pickupType == domain.PickupTypePickUp {
			if req.IsAuthorizedOverride {
				// 例外通道：来接者不在授权集合，必须留姓名以便事后复核
				if strings.TrimSpace(req.OverrideName) == "" {
					return nil, ErrTeacherOverrideNameRequired
				}
				isOverride = true
				overrideName = strings.TrimSpace(req.OverrideName)
				if pickupName == "" {
					pickupName = overrideName
				}
			} else {
				// 列表选人：必须给出授权记录
				if req.AuthorizationID == nil || *req.AuthorizationID == 0 {
					return nil, ErrTeacherInvalidParams
				}
				authorizationID = req.AuthorizationID
			}
		}
	}

	// ---- 2. 学生归属 ----
	var st domain.Student
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", studentID, tc.KindergartenID, tc.ClassID).
		First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}

	// ---- 3. 连续同向拒绝（托管序列必须交替）----
	last, err := s.pickup.FindLastRecordOfDay(ctx, studentID, today)
	if err != nil {
		return nil, err
	}
	if IsConsecutiveSameDirection(last, pickupType) {
		return nil, ErrTeacherConsecutiveDirection
	}

	// ---- 4. 接走须先有送入 ----
	if pickupType == domain.PickupTypePickUp && last == nil {
		return nil, ErrTeacherPickupOrder
	}

	// ---- 5. 理由必填判定（重复接走 / 提前接走）----
	deadline := s.classPickupDeadline(ctx, tc.ClassID)
	priorPickups := int64(0)
	if pickupType == domain.PickupTypePickUp {
		priorPickups, err = s.pickup.CountRecordsOfTypeAfter(ctx, studentID, today, domain.PickupTypePickUp)
		if err != nil {
			return nil, err
		}
	}
	remarkRequired, remarkReason := NeedsRemarkForPickup(pickupType, priorPickups, now, deadline)
	if remarkRequired && strings.TrimSpace(req.Remark) == "" {
		return nil, fmt.Errorf("%w: %s", ErrTeacherRemarkRequired, remarkReason)
	}

	// ---- 6. 授权二次校验（I1：接走人仍在有效授权集合内）----
	var isAuthorized bool
	if authorizationID != nil {
		auth, err := s.pickup.FindAuthorizationByStudent(ctx, studentID, *authorizationID)
		if err != nil {
			return nil, err
		}
		if auth == nil || !authorizationValid(auth, now) {
			return nil, ErrTeacherAuthorizationInvalid
		}
		isAuthorized = true
		if auth.Name != "" {
			pickupName = auth.Name
			pickupRelation = auth.Relation
		}
		pickupAvatar = auth.Avatar
	}

	// ---- 7. 落库 ----
	row := &domain.PickupRecord{
		KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, StudentID: studentID,
		Date: today, Type: pickupType, Method: method,
		PickupPersonID: authorizationID,
		PickupName:     pickupName, PickupRelation: pickupRelation,
		OperatorID: tc.UserID, RecordedAt: now, Remark: req.Remark,
		IsAuthorizedOverride: isOverride, OverrideName: overrideName,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}

	// ---- 8. 凭证即焚（一次有效）----
	if qrToBurn != nil {
		if err := s.pickup.DeactivateStudentQRCodes(ctx, studentID); err != nil {
			return nil, err
		}
	}

	// ---- 9. 通知家长（失败仅记日志，不回滚登记）----
	// 通知接收方是该孩家长，不是接走人本人 —— NotifyOnPickup 由家长在建立授权时设定
	if pickupType == domain.PickupTypePickUp {
		s.notifyPickupParents(ctx, st, row, isOverride, overrideName, authorizationID)
	}

	// 入园(type=1) → 自动生成待晨检记录
	if pickupType == domain.PickupTypeDropOff {
		hc := &domain.MorningHealthCheck{
			KindergartenID: tc.KindergartenID,
			ClassID:        tc.ClassID,
			StudentID:      studentID,
			PickupRecordID: row.ID,
			Date:           today.Format("2006-01-02"),
			CheckStatus:    domain.HealthCheckPending,
		}
		if err := s.db.WithContext(ctx).Create(hc).Error; err != nil {
			slog.Default().Error("auto-create health check failed", "error", err)
		}
	}

	resp := map[string]any{
		"record_id": row.ID,
		"student": map[string]any{
			// Student 域无独立 photo 字段，复用 avatar（产品决策 2026-10-07）
			"id": st.ID, "name": st.Name, "avatar": st.Avatar, "photo": st.Avatar,
		},
		"person": map[string]any{
			"name": pickupName, "relation": pickupRelation, "photo": pickupAvatar,
		},
		"type":                   int(row.Type),
		"type_label":             pickupTypeLabel(pickupType),
		"is_authorized":          isAuthorized,
		"is_authorized_override": isOverride,
		"recorded_at":            row.RecordedAt.UTC().Format(time.RFC3339),
	}
	return resp, nil
}

// StudentAuthorizedPersons 返回指定学生的有效授权接送人列表（老师端手动登记选人用）。
//
// 仅返回有效授权人（status=1, review_status=1, 未过期），
// 与后端 authorizationValid 校验口径一致，确保列表中的人能通过二次校验。
func (s *TeacherPortalService) StudentAuthorizedPersons(
	ctx context.Context,
	tc *TeacherPortalContext,
	studentID uint64,
) (map[string]any, error) {
	// 校验学生归属本班
	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.Student{}).
		Where("id = ? AND class_id = ? AND deleted_at IS NULL", studentID, tc.ClassID).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errors.New("学生不属于本班")
	}

	rows, _, err := s.pickup.ListAuthorizationsByStudentIDs(ctx, tc.KindergartenID, []uint64{studentID}, 0, 50)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		a := &domain.PickupAuthorization{
			Status:       r.Status,
			ReviewStatus: r.ReviewStatus,
			ExpireAt:     r.ExpireAt,
		}
		if !authorizationValid(a, now) {
			continue
		}
		list = append(list, map[string]any{
			"id":        r.ID,
			"name":      r.Name,
			"relation":  r.Relation,
			"phone":     r.Phone,
			"avatar":    r.Avatar,
			"auth_type": r.AuthType,
		})
	}
	return map[string]any{"list": list, "total": len(list)}, nil
}

// authorizationValid 判定授权是否仍在有效集合内（I1）。
// 口径：status=1 有效、review_status=1 已通过、且未过期。
func authorizationValid(a *domain.PickupAuthorization, now time.Time) bool {
	if a == nil {
		return false
	}
	if a.Status != 1 {
		return false
	}
	if a.ReviewStatus != 1 {
		return false
	}
	if a.ExpireAt != nil && a.ExpireAt.Before(now) {
		return false
	}
	return true
}

// classPickupDeadline 取班级放学时刻；未配置时返回空串，由 NeedsRemarkForPickup 回退缺省。
func (s *TeacherPortalService) classPickupDeadline(ctx context.Context, classID uint64) string {
	var c domain.Class
	if err := s.db.WithContext(ctx).Select("pickup_deadline").
		Where("id = ?", classID).First(&c).Error; err != nil {
		return ""
	}
	return c.PickupDeadline
}

// notifyPickupParents 孩子被接走时通知其家长。
//
// 接收方是该孩家长（由 StudentParent 关联），不是接走人本人：
// NotifyOnPickup 配置由家长在建立授权时设定，受益方即家长。
// 通知是附带信息，写失败只记日志，绝不回滚接走登记。
// 写入走 pickup_notify.go 的共享路径（与滞留一键通知同一入口）。
func (s *TeacherPortalService) notifyPickupParents(
	ctx context.Context,
	st domain.Student,
	row *domain.PickupRecord,
	isOverride bool,
	overrideName string,
	authorizationID *uint64,
) {
	// 例外接走一律通知（风险信号），不看 NotifyOnPickup
	notify := true
	if !isOverride && authorizationID != nil {
		if auth, err := s.pickup.FindAuthorizationByStudent(ctx, st.ID, *authorizationID); err == nil && auth != nil {
			notify = auth.NotifyOnPickup
		}
	}
	if !notify {
		return
	}

	who := row.PickupName
	if isOverride && overrideName != "" {
		who = overrideName
	}
	suffix := ""
	if isOverride {
		suffix = "（授权例外，请留意）"
	}
	title := "孩子已被接走"
	content := fmt.Sprintf("%s 于 %s 被 %s 接走%s",
		st.Name,
		row.RecordedAt.Format("2006-01-02 15:04"),
		who,
		suffix,
	)

	notifyStudentParents(ctx, s.db, s.msg, st.ID, title, content, row.ID)
}

func (s *TeacherPortalService) PickupRecords(ctx context.Context, tc *TeacherPortalContext, page, pageSize int, date string, studentID *uint64) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	db := s.db.WithContext(ctx).Model(&domain.PickupRecord{}).Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID)
	if date != "" {
		d, err := time.Parse("2006-01-02", date)
		if err == nil {
			db = db.Where("date = ?", d)
		}
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
	Status      string `json:"status"` // draft / published（缺省 published）
	Notify      bool   `json:"notify"` // 保存后是否通知家长（草稿忽略）
}

// CalendarEvents 老师视角事件列表：本班事件（含草稿）+ 园级已发布；标签/颜色按园所字典解析。
func (s *TeacherPortalService) CalendarEvents(ctx context.Context, tc *TeacherPortalContext, start, end string) (map[string]any, error) {
	enabled, err := loadTimetableEnabled(ctx, s.db, tc.KindergartenID)
	if err != nil {
		return nil, err
	}
	rows, err := s.calendar.FindTeacherEvents(ctx, tc.KindergartenID, tc.ClassID, start, end, enabled)
	if err != nil {
		return nil, err
	}
	meta, err := loadCalendarTypeMeta(ctx, s.db, tc.KindergartenID)
	if err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, teacherCalendarEventMap(&rows[i], meta))
	}
	return map[string]any{"list": list, "total": len(list)}, nil
}

func (s *TeacherPortalService) CalendarEventsByDate(ctx context.Context, tc *TeacherPortalContext, date string) (map[string]any, error) {
	day, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return nil, ErrTeacherInvalidParams
	}
	res, err := s.CalendarEvents(ctx, tc, date, date)
	if err != nil {
		return nil, err
	}
	weekdays := [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	return map[string]any{"date": date, "weekday": weekdays[int(day.Weekday())], "events": res["list"]}, nil
}

// GetCalendarEvent 查询单条日历事件详情（园级事件可读不可改）。
func (s *TeacherPortalService) GetCalendarEvent(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	row, err := s.calendar.FindTeacherEventByID(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, ErrTeacherNotFound
	}
	meta, err := loadCalendarTypeMeta(ctx, s.db, tc.KindergartenID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"event": teacherCalendarEventMap(row, meta)}, nil
}

func (s *TeacherPortalService) CreateCalendarEvent(ctx context.Context, tc *TeacherPortalContext, req TeacherCalendarInput) (map[string]any, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" || strings.TrimSpace(req.EventDate) == "" {
		return nil, ErrTeacherInvalidParams
	}
	day, err := time.ParseInLocation("2006-01-02", req.EventDate, time.Local)
	if err != nil {
		return nil, ErrTeacherInvalidParams
	}
	status, ok := calendarNormalizeStatus(req.Status)
	if !ok {
		return nil, ErrTeacherInvalidParams
	}
	row := &domain.CalendarEvent{
		KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, CreatorID: tc.UserID,
		Title: title, Description: req.Description,
		EventType: calendarNormalizeEventType(req.EventType),
		Scope:     domain.CalendarScopeClass, Status: status, Source: domain.CalendarSourceManual,
		EventDate: teacherPortalDay(day), StartTime: req.StartTime, EndTime: req.EndTime,
		Location: req.Location, IsAllDay: req.IsAllDay,
	}
	if err := s.calendar.CreateEvent(ctx, row); err != nil {
		return nil, err
	}
	if status == domain.CalendarStatusPublished && req.Notify {
		if err := s.pushCalendarNotice(ctx, tc, row, ""); err != nil {
			return nil, err
		}
	}
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

// UpdateCalendarEvent 编辑本班事件。园级事件只读；已发布事件的实质变更会记录变更要旨（家长端提示的数据源）。
func (s *TeacherPortalService) UpdateCalendarEvent(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherCalendarInput) error {
	existing, err := s.calendar.FindTeacherEventByID(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrTeacherNotFound
	}
	if existing.Scope == domain.CalendarScopeKindergarten {
		return ErrTeacherForbidden // 园级事件对老师只读（spec R2/R6）
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return ErrTeacherInvalidParams
	}
	status, ok := calendarNormalizeStatus(req.Status)
	if !ok {
		return ErrTeacherInvalidParams
	}
	next := *existing
	next.Title = title
	next.Description = req.Description
	next.EventType = calendarNormalizeEventType(req.EventType)
	next.StartTime = req.StartTime
	next.EndTime = req.EndTime
	next.Location = req.Location
	next.IsAllDay = req.IsAllDay
	next.Status = status
	if req.EventDate != "" {
		day, err := time.ParseInLocation("2006-01-02", req.EventDate, time.Local)
		if err != nil {
			return ErrTeacherInvalidParams
		}
		next.EventDate = teacherPortalDay(day)
	}

	updates := map[string]any{
		"title": next.Title, "description": next.Description, "event_type": next.EventType,
		"start_time": next.StartTime, "end_time": next.EndTime, "location": next.Location,
		"is_all_day": next.IsAllDay, "status": next.Status, "event_date": next.EventDate,
	}
	// 课表物化实例被老师调整 → 记为覆盖，后续同步不再触碰该槽位（D5）
	if existing.Source == domain.CalendarSourceTimetable {
		updates["overridden"] = true
	}
	// 变更要旨：仅针对已发布事件（草稿无人知晓）；有实质差异才落库。
	summary := ""
	if existing.Status == domain.CalendarStatusPublished {
		summary = calendarChangeSummary(existing, &next)
		if summary != "" {
			updates["revised_at"] = time.Now()
			updates["change_summary"] = summary
		}
	}

	affected, err := s.calendar.UpdateEvent(ctx, tc.KindergartenID, tc.ClassID, id, updates)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrTeacherNotFound
	}
	if next.Status == domain.CalendarStatusPublished && req.Notify {
		return s.pushCalendarNotice(ctx, tc, &next, summary)
	}
	return nil
}

func (s *TeacherPortalService) DeleteCalendarEvent(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	existing, err := s.calendar.FindTeacherEventByID(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrTeacherNotFound
	}
	if existing.Scope == domain.CalendarScopeKindergarten {
		return ErrTeacherForbidden // 园级事件对老师只读
	}
	// 课表物化实例被取消：软删行保持占位并标记覆盖，同步不复活（D5/R4）
	if existing.Source == domain.CalendarSourceTimetable {
		if _, err := s.calendar.UpdateEvent(ctx, tc.KindergartenID, tc.ClassID, id, map[string]any{"overridden": true}); err != nil {
			return err
		}
	}
	affected, err := s.calendar.DeleteEvent(ctx, tc.KindergartenID, tc.ClassID, id)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

// CalendarDots 月视图标记点：日期 → 当天事件类型集合（与列表同口径；客户端按字典色渲染）。
func (s *TeacherPortalService) CalendarDots(ctx context.Context, tc *TeacherPortalContext, start, end string) (map[string]any, error) {
	enabled, err := loadTimetableEnabled(ctx, s.db, tc.KindergartenID)
	if err != nil {
		return nil, err
	}
	rows, err := s.calendar.DotsForTeacher(ctx, tc.KindergartenID, tc.ClassID, start, end, enabled)
	if err != nil {
		return nil, err
	}
	dots := make(map[string][]string)
	for _, r := range rows {
		day := teacherPortalDateString(r.Date)
		dots[day] = append(dots[day], r.EventType)
	}
	return map[string]any{"dots": dots}, nil
}

// pushCalendarNotice 经通知模块生成一条面向本班家长的通知，并回写事件的推送记录位（is_published/notice_id）。
// 通知失败不撤销事件：error 上抛让调用方感知（客户端可改后重试通知）。
func (s *TeacherPortalService) pushCalendarNotice(ctx context.Context, tc *TeacherPortalContext, event *domain.CalendarEvent, changeSummary string) error {
	res, err := s.CreateNotice(ctx, tc, TeacherNoticeInput{
		Title:    calendarNoticeTitle(event, changeSummary),
		Content:  calendarNoticeContent(event, changeSummary),
		Category: calendarNoticeCategory(event.EventType),
		ClassIDs: []uint64{event.ClassID},
	})
	if err != nil {
		return err
	}
	updates := map[string]any{"is_published": true}
	if id, ok := res["id"].(uint64); ok && id > 0 {
		updates["notice_id"] = id
	}
	_, err = s.calendar.UpdateEvent(ctx, tc.KindergartenID, event.ClassID, event.ID, updates)
	return err
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

// TeacherTaskInput 老师端创建/编辑任务入参。
//
// 可选项用指针表示「未提供」：省略 `star_goal`/`allow_public`/`description` 时
// 编辑必须保留原值。用零值会让一次「只改标题」的请求静默把星星目标清零、
// 关掉公开并批量撤回已公开内容（管理端 C-1 已改，这里必须同口径）。
type TeacherTaskInput struct {
	Title             string           `json:"title"`
	Description       *string          `json:"description"`
	Intro             string           `json:"intro"`
	Requirements      []string         `json:"requirements"`
	TargetStudentID   uint64           `json:"target_student_id"`
	LearningLink      string           `json:"learning_link"`
	Frequency         string           `json:"frequency"`
	StartDate         string           `json:"start_date"`
	EndDate           string           `json:"end_date"`
	AllowPublic       *bool            `json:"allow_public"`
	RecognitionRitual string           `json:"recognition_ritual"`
	StarScope         string           `json:"star_scope"`
	StarGoal          *int             `json:"star_goal"`
	Status            int8             `json:"status"`
}

func (s *TeacherPortalService) ListTasks(ctx context.Context, tc *TeacherPortalContext, page, pageSize int) (map[string]any, error) {
	q := parseTeacherPortalPage(page, pageSize)
	db := s.db.WithContext(ctx).Model(&domain.ParentTask{}).Where("kindergarten_id = ? AND class_id = ?", tc.KindergartenID, tc.ClassID)
	var total int64
	// 计数失败必须上报：静默返回 0 会让老师看到「没有待回应」，而实际上只是查询失败。
	// 上报前先落日志——handler 层只回「获取亲子任务失败」，ctx 里的原始错误若不在这里打，
	// 服务端就一行痕迹都不留，排障只能靠猜（超时？连接池耗尽？哪个任务？）。
	if err := db.Count(&total).Error; err != nil {
		log.L(ctx).Error("list tasks count failed", "scope", "total", "err", err)
		return nil, err
	}
	var rows []domain.ParentTask
	if err := db.Order("created_at DESC").Limit(q.PageSize).Offset(q.Offset()).Find(&rows).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	// 星星进度按列表一次批量算，避免每任务一条 COUNT。
	// 失败必须上报：静默归零会让老师看到「0 星」，而实际只是查询失败。
	taskIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		taskIDs = append(taskIDs, r.ID)
	}
	starCounts, err := taskStarCounts(ctx, s.db, taskIDs)
	if err != nil {
		// 星数是装饰性聚合：读失败不应让整个任务列表 500（老师还需要看任务/待回应），
		// 但也不能给 0（假的「没有星星」）。整页字段置 null，客户端隐藏进度。
		log.L(ctx).Error("list tasks star count failed", "err", err)
		starCounts = nil
	}
	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		var pending int64
		if err := s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
			Where("task_id = ? AND response_id = 0", r.ID).Count(&pending).Error; err != nil {
			log.L(ctx).Error("list tasks count failed", "scope", "pending", "task", r.ID, "err", err)
			return nil, err
		}
		// 汇总回应只关当前周期，列表同时给出本周期数量与区间：
		// 否则「待回应 N」与点进详情后「一键送星」实际关闭的条数对不上。
		periodStart, periodEnd := taskPeriodRange(&r, now)
		periodPending := pending
		if pending > 0 && batchRespondAllowed(&r) {
			periodPending = 0
			// 同上报错，不回退成全量：回退值看起来和「没有旧周期遗留」完全一样，
			// 会把「查询失败」伪装成「本周期全都关得掉」。
			if err := s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
				Where("task_id = ? AND response_id = 0 AND date >= ? AND date <= ?", r.ID,
					periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02")).Count(&periodPending).Error; err != nil {
				log.L(ctx).Error("list tasks count failed", "scope", "period_pending", "task", r.ID, "err", err)
				return nil, err
			}
		}
		var starCount any
		if starCounts != nil {
			starCount = starCounts[r.ID]
		}
		list = append(list, map[string]any{
			"id": r.ID, "title": r.Title, "description": r.Description, "intro": r.Intro,
			"learning_link": r.LearningLink, "target_student_id": r.TargetStudentID,
			"frequency": r.Frequency, "start_date": r.StartDate.Format("2006-01-02"),
			"end_date": r.EndDate.Format("2006-01-02"), "status": r.Status,
			"allow_public": r.AllowPublic, "recognition_ritual": r.RecognitionRitual,
			"star_scope": r.StarScope, "star_goal": r.StarGoal, "star_count": starCount,
			"pending_responses": pending, "pending_period": periodPending,
			"period_start": periodStart.Format("2006-01-02"),
			"period_end":   periodEnd.Format("2006-01-02"),
			"created_at":   r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return map[string]any{"list": list, "total": total, "page": q.Page, "page_size": q.PageSize}, nil
}

func (s *TeacherPortalService) GetTask(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var t domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&t).Error; err != nil {
		return nil, ErrTeacherNotFound
	}
	// 汇总回应只关当前周期：把区间一起返回，前端才能区分「本周期待回应」与旧周期遗留。
	periodStart, periodEnd := taskPeriodRange(&t, time.Now())
	// 详情页的星数读失败不阻塞编辑/回应，但也不能给 0 —— 0 会伪装成「没有星星」。
	// 失败时字段为 null，客户端据此隐藏「已集 N 星」，而不是显示错误的 0。
	var starCount any
	if counts, err := taskStarCounts(ctx, s.db, []uint64{t.ID}); err != nil {
		log.L(ctx).Error("get task star count failed", "task", t.ID, "err", err)
	} else {
		starCount = counts[t.ID]
	}
	return map[string]any{
		"id": t.ID, "title": t.Title, "description": t.Description, "intro": t.Intro,
		"learning_link": t.LearningLink, "target_student_id": t.TargetStudentID,
		"frequency": t.Frequency, "start_date": t.StartDate.Format("2006-01-02"),
		"end_date": t.EndDate.Format("2006-01-02"), "allow_public": t.AllowPublic,
		"recognition_ritual": t.RecognitionRitual, "star_scope": t.StarScope,
		"star_goal": t.StarGoal, "star_count": starCount, "status": t.Status,
		"period_start": periodStart.Format("2006-01-02"),
		"period_end":   periodEnd.Format("2006-01-02"),
		"created_at":   t.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *TeacherPortalService) CreateTask(ctx context.Context, tc *TeacherPortalContext, req TeacherTaskInput) (map[string]any, error) {
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.LearningLink) == "" {
		return nil, ErrTeacherInvalidParams
	}
	// 列宽校验必须在 service：TeacherTaskInput 没有 binding，放过去就是 MySQL 1406 → 500。
	if err := validateTaskTitleLength(req.Title); err != nil {
		return nil, err
	}
	start := teacherPortalDay(time.Now())
	end := start.AddDate(0, 0, 7)
	// 请求里的日期是业务日期（Asia/Shanghai），必须与 taskPeriodRange 同一时区解析，
	// 否则服务器时区偏东时会被 teacherPortalDay 换算成前一天。
	loc := shanghaiLoc()
	if t, err := time.ParseInLocation("2006-01-02", req.StartDate, loc); err == nil {
		start = teacherPortalDay(t)
	}
	if t, err := time.ParseInLocation("2006-01-02", req.EndDate, loc); err == nil {
		end = teacherPortalDay(t)
	}
	if end.Before(start) {
		return nil, ErrTeacherInvalidParams
	}
	freq := domain.TaskFrequency(strings.TrimSpace(req.Frequency))
	if freq == "" {
		freq = domain.TaskFrequencyOnce
	}
	if !validTaskFrequency(freq) {
		return nil, ErrTeacherInvalidParams
	}
	reqsJSON, _ := json.Marshal(req.Requirements)
	// 未提供的字段按创建默认值展开；枚举与下界走 task_rules.go 的共享判定，
	// 与管理端同一套，避免老师端能落库「CLASS」「class 」这类非规范值。
	starGoal := 0
	if req.StarGoal != nil {
		starGoal = *req.StarGoal
	}
	description := ""
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}
	allowPublicReq := false
	if req.AllowPublic != nil {
		allowPublicReq = *req.AllowPublic
	}
	starScope := strings.TrimSpace(req.StarScope)
	if starScope == "" {
		starScope = domain.TaskStarScopeTask
	}
	ritual := strings.TrimSpace(req.RecognitionRitual)
	if ritual == "" {
		ritual = domain.TaskRitualSticker
	}
	allowPublic := allowPublicReq && req.TargetStudentID == 0
	// 结构规则（目标孩子归属 / 仪式 / 星星范围 / 目标 / 班级罐）与管理端独立任务、
	// 多班活动共用同一个 helper（D12 / tasks 3.8）。
	if err := validateTaskStructureForCreate(ctx, s.db, tc.KindergartenID, tc.ClassID,
		req.TargetStudentID,
		taskStructure{Ritual: ritual, StarScope: starScope, StarGoal: starGoal}); err != nil {
		// 班级罐缺失必须原样上抛：spec 要求 teacher/admin/activity 三个入口
		// 返回同一句「请先配置班级星星罐」，折叠成笼统的「参数无效」会让老师
		// 不知道去哪配。其余结构错误仍是 ErrTeacherInvalidParams，不扩大改动面。
		if errors.Is(err, ErrTaskStarJarMissing) {
			return nil, err
		}
		return nil, ErrTeacherInvalidParams
	}
	row := &domain.ParentTask{
		KindergartenID: tc.KindergartenID, ClassID: tc.ClassID, CreatorID: tc.UserID,
		TargetStudentID: req.TargetStudentID, Title: req.Title, Description: description,
		Intro: req.Intro, Requirements: string(reqsJSON),
		LearningLink: req.LearningLink, Frequency: freq, StartDate: start, EndDate: end,
		AllowPublic: allowPublic, RecognitionRitual: ritual, StarScope: starScope,
		StarGoal: starGoal, Status: 1,
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	// 通知在任务落库之后做：失败不回滚任务（5.3），只把计数返回给老师。
	notified, notifyFailed := s.notifyTaskPublished(ctx, row)
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339),
		"notified_count": notified, "notify_failed_count": notifyFailed}, nil
}

func (s *TeacherPortalService) UpdateTask(ctx context.Context, tc *TeacherPortalContext, id uint64, req TeacherTaskInput) error {
	var cur domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&cur).Error; err != nil {
		return ErrTeacherNotFound
	}
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.LearningLink) == "" {
		return ErrTeacherInvalidParams
	}
	if err := validateTaskTitleLength(req.Title); err != nil {
		return err
	}
	if req.Frequency != "" && !validTaskFrequency(domain.TaskFrequency(req.Frequency)) {
		return ErrTeacherInvalidParams
	}
	// 未提供的可选字段一律沿用原值：用零值会让「只改标题」静默清零星星目标、
	// 关掉公开并批量撤回已公开内容（与管理端同一口径，D12）。
	starGoal := cur.StarGoal
	if req.StarGoal != nil {
		starGoal = *req.StarGoal
	}
	description := cur.Description
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}
	allowPublic := cur.AllowPublic
	if req.AllowPublic != nil {
		allowPublic = *req.AllowPublic
	}
	if !validTaskStarGoal(starGoal) {
		return ErrTeacherInvalidParams
	}
	if err := validateTargetStudent(ctx, s.db, tc.KindergartenID, tc.ClassID, req.TargetStudentID); err != nil {
		return ErrTeacherInvalidParams
	}
	starScope := strings.TrimSpace(req.StarScope)
	if starScope == "" {
		starScope = cur.StarScope
	}
	if starScope != domain.TaskStarScopeTask && starScope != domain.TaskStarScopeClass {
		return ErrTeacherInvalidParams
	}
	ritual := strings.TrimSpace(req.RecognitionRitual)
	if ritual == "" {
		ritual = cur.RecognitionRitual
	}
	if !validTaskRitual(ritual) {
		return ErrTeacherInvalidParams
	}
	// 只在范围**真的变化**时校验班级罐：罐被删后，改标题不该连带 422（与管理端同口径）。
	// 校验本身放到结构锁之后（见下方）：已有提交的任务连结构都改不了，
	// 先报「请先配置班级星星罐」会让老师配完再撞 422，属于误导。
	scopeChanged := starScope != cur.StarScope
	// 目标规则只在本次动到目标/仪式/范围时生效：历史脏数据（goal=0 的 class_star 任务）
	// 不该把无关编辑一起挡住。判定用**有效值**，否则省略 star_goal 的请求永远看到 0。
	goalTouched := req.StarGoal != nil || ritual != cur.RecognitionRitual || scopeChanged
	if ritual == domain.TaskRitualClassStar && starScope == domain.TaskStarScopeTask && starGoal <= 0 && goalTouched {
		return ErrTeacherInvalidParams
	}
	loc := shanghaiLoc()
	start, _ := time.ParseInLocation("2006-01-02", req.StartDate, loc)
	end, _ := time.ParseInLocation("2006-01-02", req.EndDate, loc)
	updates := map[string]any{}
	if req.Title != cur.Title {
		updates["title"] = req.Title
	}
	if description != cur.Description {
		updates["description"] = description
	}
	if req.LearningLink != cur.LearningLink {
		updates["learning_link"] = req.LearningLink
	}
	if req.TargetStudentID != cur.TargetStudentID {
		updates["target_student_id"] = req.TargetStudentID
	}
	if req.Frequency != "" && req.Frequency != string(cur.Frequency) {
		updates["frequency"] = req.Frequency
	}
	if !start.IsZero() && !sameTaskDay(start, cur.StartDate) {
		updates["start_date"] = start
	}
	if !end.IsZero() && !sameTaskDay(end, cur.EndDate) {
		updates["end_date"] = end
	}
	if allowPublic != cur.AllowPublic {
		updates["allow_public"] = allowPublic
	}
	if ritual != cur.RecognitionRitual {
		updates["recognition_ritual"] = ritual
	}
	if starScope != cur.StarScope {
		updates["star_scope"] = starScope
	}
	if starGoal != cur.StarGoal {
		updates["star_goal"] = starGoal
	}
	if req.Status != 0 && req.Status != cur.Status {
		updates["status"] = req.Status
	}
	if req.TargetStudentID > 0 && allowPublic {
		updates["allow_public"] = false
	}
	if len(updates) == 0 {
		return nil
	}
	// 活动任务只能改安全字段：老师端同样不能单条改频率/日期/个体/星星/状态，
	// 否则一个 2 班活动会被改成「1 班 + 1 个个体任务」，activity_id 还留着。
	if cur.ActivityID != "" {
		for k := range updates {
			if !taskSafeEditableField(k) {
				return ErrTaskActivityEditForbidden
			}
		}
	}
	if err := assertTaskEditable(ctx, s.db, id, updates); err != nil {
		// 结构锁要单独上报，handler 才能给出「有提交后只能改…」的明确文案。
		if errors.Is(err, ErrTaskStructuralLocked) {
			return ErrTaskStructuralLocked
		}
		return ErrTeacherInvalidParams
	}
	// 班级罐校验放在结构锁之后：已有提交的任务改了 scope 也落不了库，
	// 先报「请先配置」会把老师引到一条配完仍然失败的路径。
	if starScope == domain.TaskStarScopeClass && scopeChanged {
		if err := validateClassStarJar(ctx, s.db, tc.KindergartenID, tc.ClassID); err != nil {
			// 与创建路径同口径：班级罐缺失要给引导文案，不折叠成「参数无效」。
			if errors.Is(err, ErrTaskStarJarMissing) {
				return err
			}
			return ErrTeacherInvalidParams
		}
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&domain.ParentTask{}).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).Updates(updates).Error; err != nil {
			return err
		}
		// 只在公开状态真的发生收缩时撤销：任务本来就不公开时也调一次，
		// 会给提交反复写 public_revoked_at，把「从未申请」洗成「已撤回」，
		// 并让撤回时间随每次无关编辑漂移（B2 的 public_status 会把这种脏数据暴露出来）。
		//
		// 判定要用**落库后的有效值**：个体任务会被强制 allow_public=false，
		// 用原始 req.AllowPublic 判定会在 clamp 逻辑变化时漏掉撤销。
		effectiveAllowPublic := allowPublic && req.TargetStudentID == 0
		becameIndividual := req.TargetStudentID > 0 && cur.TargetStudentID == 0
		if becameIndividual || (cur.AllowPublic && !effectiveAllowPublic) {
			reason := "allow_public_off"
			if becameIndividual {
				reason = "became_individual"
			}
			return revokePublicForTask(ctx, tx, tc.KindergartenID, tc.UserID, id, reason)
		}
		return nil
	})
}

func (s *TeacherPortalService) DeleteTask(ctx context.Context, tc *TeacherPortalContext, id uint64) error {
	var title string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var task domain.ParentTask
		// 先锁任务行再做级联：与 SupplementResponse / ReactToResponse 的「锁任务行」对齐，
		// 让删除与并发追加补充/反馈串行，级联才不会漏掉对方刚写入的关系行。
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).
			First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeacherNotFound
			}
			log.L(ctx).Error("task delete load task failed", "task", id, "err", err)
			return err
		}
		title = task.Title
		if task.ActivityID != "" {
			// 活动任务必须整组删：老师单删一条会让活动分组与统计聚合悬空
			// （与 TaskService.DeleteTask 同一道禁令，spec：活动任务 MUST NOT 走单条删除）。
			return ErrTaskActivityDeleteForbidden
		}
		if err := deleteTaskCascade(ctx, tx, id); err != nil {
			if errors.Is(err, errTaskNotFound) {
				return ErrTeacherNotFound
			}
			// 级联失败不是「任务不存在」：吞成 404 会让客户端以为已经删掉了。
			log.L(ctx).Error("task delete cascade failed", "task", id, "err", err)
			return err
		}
		return nil
	})
	if isTaskLockConflict(err) {
		return ErrTaskBusy
	}
	if err != nil {
		return err
	}
	_ = s.db.WithContext(ctx).Create(&domain.OperationLog{
		KindergartenID: tc.KindergartenID,
		UserID:         tc.UserID,
		Module:         "tasks",
		Action:         "delete",
		TargetType:     "parent_task",
		TargetID:       id,
		Detail:         fmt.Sprintf("title=%s", title),
	}).Error
	return nil
}

func (s *TeacherPortalService) TaskStats(ctx context.Context, tc *TeacherPortalContext, id uint64) (map[string]any, error) {
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).First(&task).Error; err != nil {
		return nil, ErrTeacherNotFound
	}
	var total, responded, pending int64
	// 三个计数失败必须上报：静默归零会让统计显示「0 条提交」，而实际只是查询失败；
	// pending 是算出来的差值，responded 失败时还会把待回应数算大。
	if err := s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).Where("task_id = ?", id).Count(&total).Error; err != nil {
		log.L(ctx).Error("task stats count failed", "scope", "total", "task", id, "err", err)
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).Where("task_id = ? AND response_id <> 0", id).Count(&responded).Error; err != nil {
		log.L(ctx).Error("task stats count failed", "scope", "responded", "task", id, "err", err)
		return nil, err
	}
	pending = total - responded
	return map[string]any{"task_id": id, "total_students": s.classStudentCount(ctx, tc),
		"submission_count": total, "responded_count": responded, "pending_count": pending}, nil
}

// TaskFeedbackSummary 老师端任务反馈汇总（tasks 3.11）。
//
// 口径与管理端 7.2 完全一致：分母是主回应承诺的义务家长数（notify_target_count），
// 分子是去重后的 (回应, 家长) 反馈对；补充回应不被任何提交引用，天然不计入。
// 公式直接复用 taskFeedbackTotalsOf，老师端不另写一套统计。
//
// 「按孩子」归属规则：
//   - 反馈归属到**同一条回应覆盖的孩子**，满足其一即可：该孩子那条提交的提交人就是反馈人，
//     或反馈人与该孩子有 student_parents 绑定（co-parent）；
//   - 一条反馈可能同时命中同一家长的多个孩子（同班兄弟姐妹），按孩子各计一次；
//     总量仍按 (回应, 家长) 对只计一次，不受影响；
//   - 两者都不命中的反馈计入 unattributed_feedback，不静默丢弃。
func (s *TeacherPortalService) TaskFeedbackSummary(
	ctx context.Context, tc *TeacherPortalContext, id uint64,
) (map[string]any, error) {
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}

	var checkinRows []struct {
		StudentID  uint64 `gorm:"column:student_id"`
		UserID     uint64 `gorm:"column:user_id"`
		ResponseID uint64 `gorm:"column:response_id"`
	}
	if err := s.db.WithContext(ctx).Table("task_check_ins").
		Select("student_id, user_id, response_id").
		Where("task_id = ? AND deleted_at IS NULL", id).
		Scan(&checkinRows).Error; err != nil {
		return nil, err
	}

	responses := map[uint64]taskStatsResponse{}
	reactions := map[uint64][]taskStatsReaction{}
	remaining := map[uint64]struct{}{}
	covered := map[uint64]map[uint64]struct{}{}  // response_id -> 覆盖的学生（去重）
	checkinUser := map[[2]uint64]uint64{}        // (response_id, student_id) -> 提交人
	studentSet := map[uint64]struct{}{}
	for _, c := range checkinRows {
		studentSet[c.StudentID] = struct{}{}
		if c.ResponseID == 0 {
			continue
		}
		remaining[c.ResponseID] = struct{}{}
		if covered[c.ResponseID] == nil {
			covered[c.ResponseID] = map[uint64]struct{}{}
		}
		covered[c.ResponseID][c.StudentID] = struct{}{}
		checkinUser[[2]uint64{c.ResponseID, c.StudentID}] = c.UserID
	}
	studentIDs := make([]uint64, 0, len(studentSet))
	for sid := range studentSet {
		studentIDs = append(studentIDs, sid)
	}
	sort.Slice(studentIDs, func(i, j int) bool { return studentIDs[i] < studentIDs[j] })

	referenced := make([]uint64, 0, len(remaining))
	for rid := range remaining {
		referenced = append(referenced, rid)
	}
	sort.Slice(referenced, func(i, j int) bool { return referenced[i] < referenced[j] })

	if len(referenced) > 0 {
		var respRows []struct {
			ID                uint64    `gorm:"column:id"`
			ResponseType      string    `gorm:"column:response_type"`
			CreatedAt         time.Time `gorm:"column:created_at"`
			NotifyTargetCount *int      `gorm:"column:notify_target_count"`
		}
		if err := s.db.WithContext(ctx).Table("task_responses").
			Select("id, response_type, created_at, notify_target_count").
			Where("id IN ? AND deleted_at IS NULL", referenced).
			Scan(&respRows).Error; err != nil {
			return nil, err
		}
		for _, r := range respRows {
			responses[r.ID] = taskStatsResponse{
				ResponseType: r.ResponseType, CreatedAt: r.CreatedAt, NotifyTargetCount: r.NotifyTargetCount,
			}
		}

		var reactRows []struct {
			ResponseID uint64 `gorm:"column:response_id"`
			UserID     uint64 `gorm:"column:user_id"`
			Reaction   string `gorm:"column:reaction"`
		}
		if err := s.db.WithContext(ctx).Table("task_response_reactions").
			Select("response_id, user_id, reaction").
			Where("response_id IN ? AND deleted_at IS NULL", referenced).
			Scan(&reactRows).Error; err != nil {
			return nil, err
		}
		// 与 7.2 同样的按对去重：唯一索引之外再防一层软删复活产生的两行。
		seen := make(map[[2]uint64]struct{}, len(reactRows))
		for _, r := range reactRows {
			key := [2]uint64{r.ResponseID, r.UserID}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			reactions[r.ResponseID] = append(reactions[r.ResponseID],
				taskStatsReaction{UserID: r.UserID, Reaction: r.Reaction})
		}
	}

	totals := taskFeedbackTotalsOf(referenced, responses, reactions)

	names := map[uint64]string{}
	if len(studentIDs) > 0 {
		var rows []struct {
			ID   uint64 `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if err := s.db.WithContext(ctx).Table("students").Select("id, name").
			Where("id IN ? AND kindergarten_id = ? AND deleted_at IS NULL",
				studentIDs, tc.KindergartenID).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			names[r.ID] = r.Name
		}
	}

	// 反馈归属：先看提交人本人，再看同一条回应覆盖范围内的绑定家长。
	bound := map[uint64]map[uint64]struct{}{}
	if len(studentIDs) > 0 {
		var rows []struct {
			StudentID uint64 `gorm:"column:student_id"`
			UserID    uint64 `gorm:"column:user_id"`
		}
		if err := s.db.WithContext(ctx).Table("student_parents").
			Select("student_id, user_id").
			Where("student_id IN ? AND deleted_at IS NULL", studentIDs).
			Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			if bound[r.StudentID] == nil {
				bound[r.StudentID] = map[uint64]struct{}{}
			}
			bound[r.StudentID][r.UserID] = struct{}{}
		}
	}

	type childFeedback struct{ happy, received, thanks int }
	perChild := make(map[uint64]*childFeedback, len(studentIDs))
	for _, sid := range studentIDs {
		perChild[sid] = &childFeedback{}
	}
	unattributed := 0
	for _, rid := range referenced {
		for _, rc := range reactions[rid] {
			targets := make([]uint64, 0, 1)
			for sid := range covered[rid] {
				if checkinUser[[2]uint64{rid, sid}] == rc.UserID {
					targets = append(targets, sid)
					continue
				}
				if _, ok := bound[sid][rc.UserID]; ok {
					targets = append(targets, sid)
				}
			}
			if len(targets) == 0 {
				unattributed++
				continue
			}
			sort.Slice(targets, func(i, j int) bool { return targets[i] < targets[j] })
			for _, sid := range targets {
				switch rc.Reaction {
				case domain.TaskReactionChildHappy:
					perChild[sid].happy++
				case domain.TaskReactionReceived:
					perChild[sid].received++
				case domain.TaskReactionThanks:
					perChild[sid].thanks++
				}
			}
		}
	}

	children := make([]map[string]any, 0, len(studentIDs))
	for _, sid := range studentIDs {
		c := perChild[sid]
		children = append(children, map[string]any{
			"student_id": sid, "student_name": names[sid],
			"child_happy": c.happy, "received": c.received, "thanks": c.thanks,
		})
	}

	return map[string]any{
		"task_id":           id,
		"feedback_coverage": taskStatRate(totals.Pairs, totals.Denominator),
		"child_happy_rate":  taskStatRate(totals.Happy, totals.Denominator),
		"ack_rate":          taskStatRate(totals.Ack(), totals.Denominator),
		"feedback_counts": map[string]any{
			"pairs": totals.Pairs, "child_happy": totals.Happy,
			"received": totals.Received, "thanks": totals.Thanks,
		},
		"notify_baseline_missing": totals.BaselineMissing,
		"unattributed_feedback":   unattributed,
		"children":                children,
	}, nil
}

// taskStarCounts 统计每个任务已集星数（D6）：
// 主回应为 class_star 且未删除的提交数。period 回应覆盖 N 条提交就计 N 星；
// 补充回应不改变 checkin.response_id，天然不计入。
//
// 这里只统计**本任务**的贡献。StarScope=class 的班级罐总进度需要 class_star_jars
// 的学期目标（C 切片），不在这个只读字段里冒充。
func taskStarCounts(ctx context.Context, db *gorm.DB, taskIDs []uint64) (map[uint64]int64, error) {
	out := make(map[uint64]int64, len(taskIDs))
	if len(taskIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		TaskID uint64 `gorm:"column:task_id"`
		Stars  int64  `gorm:"column:stars"`
	}
	// 谓词由 starCountBase 提供：班级罐的进度用的是同一份定义，不各写一套。
	if err := starCountBase(ctx, db, taskIDs).
		Select("c.task_id AS task_id, COUNT(*) AS stars").
		Group("c.task_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.TaskID] = r.Stars
	}
	return out, nil
}

// taskSupplementsForCheckins 批量取一组提交上的补充回应，按提交 id 分组。
// 老师端要看到「某条提交已补过几条」，否则重新进入页面会以为没补过而重复追加。
func (s *TeacherPortalService) taskSupplementsForCheckins(ctx context.Context, checkinIDs []uint64) (map[uint64][]taskSupplementRow, error) {
	out := make(map[uint64][]taskSupplementRow)
	if len(checkinIDs) == 0 {
		return out, nil
	}
	var rows []taskSupplementRow
	// 与家长端同口径：关系、提交、回应三层都过滤软删。
	if err := s.db.WithContext(ctx).Table("task_response_supplements AS s").
		Select("s.response_id, s.check_in_id, c.response_id AS main_id, r.ritual, r.sticker_key, r.content, r.media_url, r.created_at").
		Joins("INNER JOIN task_check_ins AS c ON c.id = s.check_in_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN task_responses AS r ON r.id = s.response_id AND r.deleted_at IS NULL").
		Where("s.check_in_id IN ? AND s.deleted_at IS NULL", checkinIDs).
		Order("s.created_at ASC, s.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.CheckInID] = append(out[r.CheckInID], r)
	}
	return out, nil
}

func (s *TeacherPortalService) TaskCheckins(ctx context.Context, tc *TeacherPortalContext, id uint64, status string) (map[string]any, error) {
	// 只为校验任务存在，不借用 TaskStats：那会顺带算两个统计 COUNT，
	// 计数一失败就把整张提交列表拖成 500，而那两个数在这个接口里根本用不到。
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", id, tc.KindergartenID, tc.ClassID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		log.L(ctx).Error("task checkins load task failed", "task", id, "err", err)
		return nil, err
	}
	q := s.db.WithContext(ctx).Preload("Student").Where("task_id = ?", id)
	switch status {
	case "pending":
		q = q.Where("response_id = 0").Order("created_at ASC")
	case "responded":
		q = q.Where("response_id <> 0").Order("created_at DESC")
	default:
		q = q.Order("created_at DESC")
	}
	var rows []domain.TaskCheckIn
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	checkinIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		checkinIDs = append(checkinIDs, r.ID)
	}
	// 一次取全页的补充回应，避免每行一条查询。
	supMap, err := s.taskSupplementsForCheckins(ctx, checkinIDs)
	if err != nil {
		// 不能降级成空数组：那对老师是「这条提交没补过」的假事实，
		// 会诱发重复补充，而客户端无法区分「没补」与「没读到」。
		// 与家长端同一故障的处理一致：整请求失败，客户端可重试。
		log.L(ctx).Error("task checkins load supplements failed", "task", id, "err", err)
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	now := time.Now().In(shanghaiLoc())
	for i, r := range rows {
		var approvedAt, revokedAt any
		if r.PublicApprovedAt != nil {
			approvedAt = r.PublicApprovedAt.UTC().Format(time.RFC3339)
		}
		if r.PublicRevokedAt != nil {
			revokedAt = r.PublicRevokedAt.UTC().Format(time.RFC3339)
		}
		// public_visible 要和公开墙同口径：提交者当前班级必须等于任务班级，
		// 否则孩子转班后老师仍看到「已公开」，而墙上其实已经隐藏。
		var submitterClass uint64
		if r.Student != nil {
			submitterClass = r.Student.ClassID
		}
		item := map[string]any{
			"id": r.ID, "student_id": r.StudentID, "content": r.Content,
			"media_file_id": r.MediaFileID, "media_url": r.MediaURL,
			"is_public_requested": r.IsPublicRequested, "public_scope_confirmed": r.PublicScopeConfirmed,
			"public_approved_at": approvedAt, "public_revoked_at": revokedAt,
			"public_status": publicCheckinStatus(&rows[i]), "public_visible": publicCheckinVisible(&task, &rows[i], submitterClass, now),
			"response_id": r.ResponseID, "created_at": r.CreatedAt.UTC().Format(time.RFC3339),
			"date": r.Date.Format("2006-01-02"),
		}
		// 补充回应：始终给数组（空数组而不是 null），客户端少一层判空。
		spOut := make([]map[string]any, 0, len(supMap[r.ID]))
		for _, sp := range supMap[r.ID] {
			spOut = append(spOut, map[string]any{
				"id": sp.ResponseID, "checkin_id": sp.CheckInID,
				"ritual": sp.Ritual, "sticker_key": sp.StickerKey,
				"content": sp.Content, "media_url": sp.MediaURL,
				"created_at": sp.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
		item["supplements"] = spOut
		if r.Student != nil {
			item["student"] = map[string]any{"id": r.Student.ID, "name": r.Student.Name, "avatar": r.Student.Avatar}
		}
		if r.ResponseID > 0 {
			var resp domain.TaskResponse
			if err := s.db.WithContext(ctx).Where("id = ?", r.ResponseID).First(&resp).Error; err == nil {
				item["response"] = map[string]any{
					"id": resp.ID, "ritual": resp.Ritual, "sticker_key": resp.StickerKey,
					"content": resp.Content, "media_url": resp.MediaURL,
					"response_type": resp.ResponseType, "created_at": resp.CreatedAt.UTC().Format(time.RFC3339),
				}
			}
		}
		list = append(list, item)
	}
	return map[string]any{"list": list, "total": len(list)}, nil
}

func (s *TeacherPortalService) RespondTask(ctx context.Context, tc *TeacherPortalContext, taskID, checkinID uint64, ritual, stickerKey, content string, mediaFileID uint64) (map[string]any, error) {
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", taskID, tc.KindergartenID, tc.ClassID).First(&task).Error; err != nil {
		return nil, ErrTeacherNotFound
	}
	if ritual == "" {
		ritual = domain.TaskRitualSticker
	}
	if ritual != domain.TaskRitualSticker && ritual != domain.TaskRitualVoice && ritual != domain.TaskRitualClassStar {
		return nil, ErrTeacherInvalidParams
	}
	if err := validateRitualPayload(ritual, stickerKey, mediaFileID); err != nil {
		return nil, err
	}
	var mediaURL string
	if mediaFileID > 0 {
		var f domain.File
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ? AND biz_type = ? AND deleted_at IS NULL", mediaFileID, tc.UserID, "task").First(&f).Error; err != nil {
			return nil, ErrTeacherInvalidParams
		}
		if f.FileType != "image" && f.FileType != "audio" {
			return nil, ErrTeacherInvalidParams
		}
		// 语音只能挂音频：挂图片能上传成功，但家长端播放不出来（规格：语音 MUST 能被播放）。
		if ritual == domain.TaskRitualVoice && f.FileType != "audio" {
			return nil, ErrTeacherRitualMediaType
		}
		mediaURL = f.FileURL
	}
	var response domain.TaskResponse
	var checkin domain.TaskCheckIn
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先锁任务行，与 DeleteTask 的级联同序：否则「锁 checkin → 插 response」会与
		// 「锁 response → 软删 checkin」形成反序死锁（实测 1213）。
		var lockedTask domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", taskID).First(&lockedTask).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeacherNotFound
			}
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND task_id = ? AND response_id = 0", checkinID, taskID).
			First(&checkin).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 区分「提交不存在」与「已被别的回应关闭」：后者是可重试冲突（409），
				// 不能报成参数无效（422）。
				var exists domain.TaskCheckIn
				if e2 := tx.Where("id = ? AND task_id = ?", checkinID, taskID).First(&exists).Error; e2 == nil && exists.ResponseID != 0 {
					return ErrTaskRespondConflict
				}
				return ErrTeacherInvalidParams
			}
			return err
		}
		response = domain.TaskResponse{
			KindergartenID: tc.KindergartenID, TaskID: taskID, TeacherID: tc.UserID,
			Ritual: ritual, StickerKey: stickerKey, Content: content,
			MediaFileID: mediaFileID, MediaURL: mediaURL, ResponseType: domain.TaskResponseSingle,
		}
		if err := tx.Create(&response).Error; err != nil {
			return err
		}
		res := tx.Model(&domain.TaskCheckIn{}).Where("id = ? AND response_id = 0", checkinID).Update("response_id", response.ID)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 并发下被抢先关闭：回滚，避免留下没有指向任何提交的回应。
			return ErrTaskRespondConflict
		}
		return nil
	})
	if isTaskLockConflict(err) {
		return nil, ErrTaskBusy
	}
	if err != nil {
		return nil, err
	}
	// 只通知本次回应覆盖到的家庭；提交人与其同孩子的另一位家长都会收到（按家长去重）。
	notified, notifyFailed := s.notifyTaskResponse(ctx, &task, response.ID, []domain.TaskCheckIn{checkin})
	return map[string]any{
		"id":                  response.ID,
		"created_at":          response.CreatedAt.UTC().Format(time.RFC3339),
		"notified_count":      notified,
		"notify_failed_count": notifyFailed,
	}, nil
}

func (s *TeacherPortalService) BatchRespondTask(ctx context.Context, tc *TeacherPortalContext, taskID uint64, ritual, stickerKey, content string, mediaFileID uint64) (map[string]any, error) {
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", taskID, tc.KindergartenID, tc.ClassID).First(&task).Error; err != nil {
		return nil, ErrTeacherNotFound
	}
	if ritual == "" {
		ritual = domain.TaskRitualSticker
	}
	if ritual != domain.TaskRitualSticker && ritual != domain.TaskRitualVoice && ritual != domain.TaskRitualClassStar {
		return nil, ErrTeacherInvalidParams
	}
	if err := validateRitualPayload(ritual, stickerKey, mediaFileID); err != nil {
		return nil, err
	}
	// 规格：个体任务与一次性活动 MUST 逐条回应，只有每日/每周班级任务可汇总。
	if !batchRespondAllowed(&task) {
		return nil, ErrTaskBatchNotAllowed
	}
	var mediaURL string
	if mediaFileID > 0 {
		var f domain.File
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ? AND biz_type = ? AND deleted_at IS NULL", mediaFileID, tc.UserID, "task").First(&f).Error; err != nil {
			return nil, ErrTeacherInvalidParams
		}
		if f.FileType != "image" && f.FileType != "audio" {
			return nil, ErrTeacherInvalidParams
		}
		// 语音只能挂音频：挂图片能上传成功，但家长端播放不出来（规格：语音 MUST 能被播放）。
		if ritual == domain.TaskRitualVoice && f.FileType != "audio" {
			return nil, ErrTeacherRitualMediaType
		}
		mediaURL = f.FileURL
	}
	// 汇总回应只关闭当前周期内的待回应提交；区间在事务内锁定读取，
	// 并发/双击时后到的事务拿不到行，不会产生孤儿回应或虚高计数。
	periodStart, periodEnd := taskPeriodRange(&task, time.Now())
	var response domain.TaskResponse
	failed := []uint64{}
	// 只收集真正写成功的提交：批量部分失败时，失败的家庭不能收到
	// 「老师回应了」——他们的提交仍是待回应。
	respondedCheckins := []domain.TaskCheckIn{}
	responded := 0
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 同 RespondTask：先锁任务行，保证与 DeleteTask 的级联加锁顺序一致。
		var lockedTask domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", taskID).First(&lockedTask).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeacherNotFound
			}
			return err
		}
		var rows []domain.TaskCheckIn
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("task_id = ? AND response_id = 0 AND date >= ? AND date <= ?",
				taskID, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02")).
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return errTaskBatchNothingPending
		}
		response = domain.TaskResponse{
			KindergartenID: tc.KindergartenID, TaskID: taskID, TeacherID: tc.UserID,
			Ritual: ritual, StickerKey: stickerKey, Content: content,
			MediaFileID: mediaFileID, MediaURL: mediaURL, ResponseType: domain.TaskResponsePeriod,
			PeriodStart: &periodStart, PeriodEnd: &periodEnd,
		}
		if err := tx.Create(&response).Error; err != nil {
			return err
		}
		for _, r := range rows {
			res := tx.Model(&domain.TaskCheckIn{}).Where("id = ? AND response_id = 0", r.ID).Update("response_id", response.ID)
			if res.Error != nil || res.RowsAffected == 0 {
				// 0 行更新说明已被其他事务关闭：计入失败项，不能静默算成功。
				failed = append(failed, r.ID)
				continue
			}
			respondedCheckins = append(respondedCheckins, r)
			responded++
		}
		if responded == 0 {
			// 有候选却一条都没写成功：回滚，不留孤儿回应，并按失败上报（不能伪装成「没有待回应」）。
			return ErrTaskBatchAllFailed
		}
		return nil
	})
	if errors.Is(err, errTaskBatchNothingPending) {
		// 读取阶段就没有待回应行：这是真正的空操作。
		return map[string]any{"id": uint64(0), "responded_count": 0, "failed_ids": []uint64{},
			"notified_count": 0, "notify_failed_count": 0}, nil
	}
	if isTaskLockConflict(err) {
		return nil, ErrTaskBusy
	}
	if err != nil {
		return nil, err
	}
	// 一次汇总回应覆盖同一家长多条提交时只发一条（5.2）。
	notified, notifyFailed := s.notifyTaskResponse(ctx, &task, response.ID, respondedCheckins)
	return map[string]any{"id": response.ID, "responded_count": responded, "failed_ids": failed,
		"notified_count": notified, "notify_failed_count": notifyFailed}, nil
}

// SupplementResponse 在汇总/单条回应之后，给某一条已回应的提交追加个别补充回应。
//
// 补充回应是一条独立的 TaskResponse(response_type=single) + 一行
// task_response_supplements 关系；**不改** TaskCheckIn.ResponseID，因此提交的
// 「已回应」状态与主回应保持不变，家长端按「主回应 + 补充回应」展示。
// 待回应的提交不能补：它还没有主回应，追加会让家长看到一条孤立的补充。
func (s *TeacherPortalService) SupplementResponse(ctx context.Context, tc *TeacherPortalContext, taskID, checkinID uint64, ritual, stickerKey, content string, mediaFileID uint64) (map[string]any, error) {
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND class_id = ?", taskID, tc.KindergartenID, tc.ClassID).First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		// DB 故障不能伪装成 404：那会让老师以为任务没了，重试也没意义。
		log.L(ctx).Error("task supplement load task failed", "task", taskID, "err", err)
		return nil, err
	}
	ritual, err := supplementRitual(ritual)
	if err != nil {
		return nil, err
	}
	if err := validateRitualPayload(ritual, stickerKey, mediaFileID); err != nil {
		return nil, err
	}
	var mediaURL string
	if mediaFileID > 0 {
		var f domain.File
		if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ? AND biz_type = ? AND deleted_at IS NULL", mediaFileID, tc.UserID, "task").First(&f).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrTeacherInvalidParams
			}
			// 文件表查询失败不是「参数无效」，不能吞成 422。
			log.L(ctx).Error("task supplement load media failed", "file", mediaFileID, "err", err)
			return nil, err
		}
		if f.FileType != "image" && f.FileType != "audio" {
			return nil, ErrTeacherInvalidParams
		}
		if ritual == domain.TaskRitualVoice && f.FileType != "audio" {
			return nil, ErrTeacherRitualMediaType
		}
		mediaURL = f.FileURL
	}
	var response domain.TaskResponse
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 先锁任务行：与 DeleteTask 的级联删除串行，避免「任务正在删」与「追加补充」
		// 交错，留下指向已软删回应的孤儿关系行（SQL 审查 F1）。
		var lockedTask domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", taskID).First(&lockedTask).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeacherNotFound
			}
			return err
		}
		// 锁住提交行：校验「已有主回应」与写补充关系必须在同一把锁内完成，
		// 否则并发的撤回/重复点击会让状态在校验与写入之间变化。
		var checkin domain.TaskCheckIn
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND task_id = ?", checkinID, taskID).
			First(&checkin).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeacherInvalidParams
			}
			return err
		}
		if checkin.ResponseID == 0 {
			return ErrTaskSupplementForbidden
		}
		response = domain.TaskResponse{
			KindergartenID: tc.KindergartenID, TaskID: taskID, TeacherID: tc.UserID,
			Ritual: ritual, StickerKey: stickerKey, Content: content,
			MediaFileID: mediaFileID, MediaURL: mediaURL, ResponseType: domain.TaskResponseSingle,
		}
		if err := tx.Create(&response).Error; err != nil {
			return err
		}
		return tx.Create(&domain.TaskResponseSupplement{
			KindergartenID: tc.KindergartenID, ResponseID: response.ID, CheckInID: checkinID,
		}).Error
	})
	if isTaskLockConflict(err) {
		return nil, ErrTaskBusy
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":         response.ID,
		"created_at": response.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// ReviewPublicCheckin 老师审核家长提交的公开申请（批准 / 拒绝或撤回）。
//
// 批准时必须 scopeConfirmed=true：规格要求老师确认「内容中只有提交者孩子」，
// 服务端无法自动判断内容边界，只能用这个显式确认承载责任（并留痕 approved_by）。
func (s *TeacherPortalService) ReviewPublicCheckin(ctx context.Context, tc *TeacherPortalContext, taskID, checkinID uint64, approve, scopeConfirmed bool) (map[string]any, error) {
	// 先做一次非锁定校验，把 404（任务不在本班）与 422（参数/状态）分开。
	// 真正的判定在锁内重做——老师从打开页面到点确认之间，任务可能被改成个体任务。
	var task domain.ParentTask
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ?", taskID, tc.KindergartenID, tc.ClassID).
		First(&task).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		log.L(ctx).Error("task public review load task failed", "task", taskID, "err", err)
		return nil, err
	}
	if approve && !scopeConfirmed {
		return nil, ErrTaskPublicScopeUnconfirmed
	}
	now := time.Now().In(shanghaiLoc())
	var out map[string]any
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 第一把锁：parent_tasks 行（锁序见 handoff §12.2）。
		// 条件必须和预检一致地带上园所/班级：否则老师打开审核页之后任务被改派到
		// 别的班，这把锁仍能拿到行，旧班老师就操作了已不属于自己的任务。
		var locked domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND kindergarten_id = ? AND class_id = ? AND deleted_at IS NULL",
				taskID, tc.KindergartenID, tc.ClassID).First(&locked).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeacherNotFound
			}
			return err
		}
		// 只有「批准」需要任务本身还支持公开；「拒绝/撤回」必须永远可用，
		// 否则任务被关掉公开开关后，老师连撤下都做不到（与家长取消同理）。
		if approve && (!locked.AllowPublic || locked.TargetStudentID > 0) {
			return ErrTaskPublicNotAllowed
		}
		var c domain.TaskCheckIn
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND task_id = ?", checkinID, taskID).First(&c).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTeacherInvalidParams
			}
			return err
		}
		// 提交者现在必须还在本班：公开墙要求「提交者班级 = 任务班级」，
		// 孩子转班后批准只会得到一个永远不上墙的「已公开」状态。
		//
		// 学生记录被删除时 ClassID 为 0，与任务班级必然不等——批准被拒；
		// 但**撤回必须照常可用**（契约：撤回永远可用），所以只有查库出错才中断。
		var submitter domain.Student
		submitterErr := tx.Select("class_id").Where("id = ?", c.StudentID).Take(&submitter).Error
		if submitterErr != nil && !errors.Is(submitterErr, gorm.ErrRecordNotFound) {
			return submitterErr
		}
		if approve && submitter.ClassID != locked.ClassID {
			return ErrTaskPublicNotAllowed
		}
		// 审计要记下变更**前**的状态与审核人：更新后 PublicApprovedBy 会被清空。
		priorStatus := publicCheckinStatus(&c)
		priorApprover := c.PublicApprovedBy
		var updates map[string]any
		if approve {
			switch publicCheckinStatus(&c) {
			case TaskPublicApproved:
				// 幂等：重复批准不刷新审核时间，避免审计记录被反复覆盖。
			case TaskPublicPending:
				updates = publicApproveUpdates(now, tc.UserID)
			default:
				// 家长已取消或撤回：必须等家长重新申请再批准，不能凭老师记忆直接放行。
				return ErrTaskPublicStateConflict
			}
		} else {
			switch publicCheckinStatus(&c) {
			case TaskPublicNone, TaskPublicRevoked:
				// 幂等：没有可撤回的东西，不刷新撤回时间。
			default:
				updates = publicRevokeUpdates(now)
			}
		}
		if updates != nil {
			if err := tx.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).Updates(updates).Error; err != nil {
				return err
			}
			var fresh domain.TaskCheckIn
			if err := tx.Where("id = ?", c.ID).First(&fresh).Error; err != nil {
				return err
			}
			c = fresh

			action := PublicAuditRevoke
			if approve {
				action = PublicAuditApprove
			}
			detail := fmt.Sprintf("task_id=%d prior_status=%s", taskID, priorStatus)
			if priorApprover > 0 {
				detail += fmt.Sprintf(" approved_by=%d", priorApprover)
			}
			if err := tx.Create(publicAuditEntry(
				tc.KindergartenID, tc.UserID, action, "task_check_in", c.ID, detail,
			)).Error; err != nil {
				return err
			}
		}
		out = publicCheckinView(&locked, &c, submitter.ClassID, now)
		return nil
	})
	if isTaskLockConflict(err) {
		return nil, ErrTaskBusy
	}
	if err != nil {
		return nil, err
	}
	return out, nil
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

// NotifyOverdueParents 老师对本班滞留儿童一键通知家长。
//
// 与管理端 PickupService.NotifyOverdueParents 走同一条 Message 写入路径，
// 但额外限定**本班**——老师不得越权通知别班学生。
// 重复触发每次各写一条，不覆盖历史。
func (s *TeacherPortalService) NotifyOverdueParents(
	ctx context.Context,
	tc *TeacherPortalContext,
	studentID uint64,
) (map[string]any, error) {
	now := time.Now()
	day := teacherPortalDay(now)

	// 归属校验：必须是本班学生
	var st domain.Student
	if err := s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND class_id = ? AND status = 1 AND deleted_at IS NULL",
			studentID, tc.KindergartenID, tc.ClassID).
		First(&st).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}

	// 校验该生当前确属滞留（在园且已过放学时刻）
	lastMap, err := s.pickup.LastRecordsOfDay(ctx, []uint64{studentID}, day)
	if err != nil {
		return nil, err
	}
	lr, ok := lastMap[studentID]
	if !ok || lr.Type != int8(domain.PickupTypeDropOff) {
		return nil, ErrTeacherInvalidParams
	}
	deadline := s.classPickupDeadline(ctx, tc.ClassID)
	if !IsOverdueNotPicked(&domain.PickupRecord{Type: domain.PickupTypeDropOff}, now, deadline) {
		return nil, ErrTeacherInvalidParams
	}

	title := "孩子尚未被接走"
	content := fmt.Sprintf("%s 已超过放学时间（%s）仍未被接走，请尽快安排接回",
		st.Name, ResolvePickupDeadline(deadline))

	sent := notifyStudentParents(ctx, s.db, s.msg, studentID, title, content, studentID)
	return map[string]any{
		"student_id":  studentID,
		"notified":    sent > 0,
		"sent_count":  sent,
		"message_id":  uint64(0),
		"notified_at": now.UTC().Format(time.RFC3339),
	}, nil
}
