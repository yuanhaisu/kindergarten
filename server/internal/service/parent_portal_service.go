package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/pkg/log"
)

var (
	ErrParentContext       = errors.New("PARENT_CONTEXT")
	ErrParentNotFound      = errors.New("PARENT_NOT_FOUND")
	ErrParentInvalidParams = errors.New("PARENT_INVALID_PARAMS")
	ErrParentForbidden     = errors.New("PARENT_FORBIDDEN")
)

type ParentPortalContext struct {
	UserID         uint64
	KindergartenID uint64
	CurrentStudent *domain.Student
	Students       []domain.Student
	Class          *domain.Class
	Relation       string
}

type ParentPortalService struct {
	db       *gorm.DB
	parents  *repository.ParentRepo
	roles    *repository.UserRoleRepo
	students *repository.StudentRepo
	classes  *repository.ClassRepo
	pickup   *repository.PickupRepo
	calendar *repository.CalendarRepo
	notice   *repository.NoticeRepo
}

func NewParentPortalService(
	db *gorm.DB,
	parents *repository.ParentRepo,
	roles *repository.UserRoleRepo,
	students *repository.StudentRepo,
	classes *repository.ClassRepo,
	pickup *repository.PickupRepo,
	calendar *repository.CalendarRepo,
	notice *repository.NoticeRepo,
) *ParentPortalService {
	return &ParentPortalService{db: db, parents: parents, roles: roles, students: students, classes: classes, pickup: pickup, calendar: calendar, notice: notice}
}

func (s *ParentPortalService) ResolveContext(ctx context.Context, userID, kindergartenID uint64, studentID *uint64) (*ParentPortalContext, error) {
	ur, err := s.roles.FindParentInKindergarten(ctx, userID, kindergartenID)
	if err != nil || ur == nil {
		return nil, ErrParentContext
	}
	spRows, err := s.parents.ListStudentParentsForUsersInKindergarten(ctx, kindergartenID, []uint64{userID})
	if err != nil || len(spRows) == 0 {
		return nil, ErrParentContext
	}
	var students []domain.Student
	for _, sp := range spRows {
		if sp.Student != nil {
			students = append(students, *sp.Student)
		}
	}
	if len(students) == 0 {
		return nil, ErrParentContext
	}

	current := &students[0]
	relation := string(spRows[0].Relation)
	if studentID != nil && *studentID != 0 {
		for i := range spRows {
			if spRows[i].StudentID == *studentID {
				current = spRows[i].Student
				relation = string(spRows[i].Relation)
				break
			}
		}
	}

	pc := &ParentPortalContext{
		UserID:         userID,
		KindergartenID: kindergartenID,
		CurrentStudent: current,
		Students:       students,
		Relation:       relation,
	}
	if current != nil {
		class, _ := s.classes.FindByID(ctx, kindergartenID, current.ClassID)
		if class != nil {
			pc.Class = class
		}
	}
	return pc, nil
}

// ResolveTaskContext 是任务链路的家长上下文解析：student_id 必填且必须属于该家长。
//
// 与 ResolveContext 的唯一差别：**不回退到第一个孩子**。任务数据按孩子隔离，
// 回退会把 A 孩子的任务/提交显示成 B 孩子的（tasks 2.2）。把这条规则放在 service，
// 而不是靠每个 handler 自己比对 `CurrentStudent.ID`，避免新增任务端点时漏掉。
func (s *ParentPortalService) ResolveTaskContext(
	ctx context.Context, userID, kindergartenID, studentID uint64,
) (*ParentPortalContext, error) {
	if studentID == 0 {
		return nil, ErrParentInvalidParams
	}
	pc, err := s.ResolveContext(ctx, userID, kindergartenID, &studentID)
	if err != nil {
		return nil, err
	}
	if pc.CurrentStudent == nil || pc.CurrentStudent.ID != studentID {
		return nil, ErrParentForbidden
	}
	return pc, nil
}

// ListGrowthFeed 家长端统一成长记录流:
// - target_type=1 (全班记录): 孩子所在班级的所有班级记录都可见
// - target_type=2 (指定学生): 仅目标学生家长可见
// 支持 year/month 可选过滤
func (s *ParentPortalService) ListGrowthFeed(ctx context.Context, pc *ParentPortalContext, studentID *uint64, year, month int, page, pageSize int) (map[string]any, error) {
	if len(pc.Students) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	studentIDs := make([]uint64, len(pc.Students))
	classIDSet := make(map[uint64]bool)
	for i, st := range pc.Students {
		studentIDs[i] = st.ID
		classIDSet[st.ClassID] = true
	}
	classIDs := make([]uint64, 0, len(classIDSet))
	for cid := range classIDSet {
		classIDs = append(classIDs, cid)
	}

	db := s.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).
		Where("growth_daily_logs.kindergarten_id = ?", pc.KindergartenID).
		Where("growth_daily_logs.class_id IN ?", classIDs)

	if year > 0 && month > 0 {
		start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 1, 0)
		db = db.Where("growth_daily_logs.date >= ? AND growth_daily_logs.date < ?", start, end)
	}

	// 统一流: 全班记录 + 个体记录(仅目标学生)。可见性规则由 growthLogVisibilityWhere 统一产出。
	{
		var single *domain.Student
		if studentID != nil && *studentID != 0 {
			for i := range pc.Students {
				if pc.Students[i].ID == *studentID {
					single = &pc.Students[i]
					break
				}
			}
			if single == nil {
				return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
			}
		}
		visWhere, visArgs := growthLogVisibilityWhere(pc.KindergartenID, pc.Students, single)
		db = db.Where(visWhere, visArgs...)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	var rows []domain.GrowthDailyLog
	if err := db.Preload("Creator").Order("growth_daily_logs.date DESC, growth_daily_logs.id DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	// 批量取卡片渲染所需的关联数据，避免每条记录单独查（一页 20 条会退化成 60+ 次查询）。
	// 空页时跳过全部查询：GORM 对空切片会生成 IN (NULL)。
	logIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		logIDs = append(logIDs, r.ID)
	}
	mediaByLog := make(map[uint64][]domain.GrowthMedia)
	likedSet := make(map[uint64]bool)
	commentCounts := make(map[uint64]int64)
	if len(logIDs) > 0 {
		var medias []domain.GrowthMedia
		if err := s.db.WithContext(ctx).Where("log_id IN ?", logIDs).Order("sort_num ASC, id ASC").Find(&medias).Error; err != nil {
			return nil, err
		}
		for _, m := range medias {
			if m.LogID == nil {
				continue // 未挂载的素材，不属于任何记录
			}
			mediaByLog[*m.LogID] = append(mediaByLog[*m.LogID], m)
		}

		var myLikes []domain.GrowthLike
		if err := s.db.WithContext(ctx).Where("log_id IN ? AND user_id = ?", logIDs, pc.UserID).Find(&myLikes).Error; err != nil {
			return nil, err
		}
		for _, l := range myLikes {
			likedSet[l.LogID] = true
		}

		type commentCountRow struct {
			LogID uint64
			Cnt   int64
		}
		var ccRows []commentCountRow
		if err := s.db.WithContext(ctx).Model(&domain.GrowthComment{}).
			Where("log_id IN ?", logIDs).
			Select("log_id, count(*) as cnt").
			Group("log_id").
			Scan(&ccRows).Error; err != nil {
			return nil, err
		}
		for _, c := range ccRows {
			commentCounts[c.LogID] = c.Cnt
		}
	}

	// 徽章：只取「当前孩子」的授予记录，且见证记录落在本页里。
	// 别的孩子的徽章不该挂到本卡片上。
	badgeByLog := map[uint64]map[string]any{}
	if len(logIDs) > 0 && pc.CurrentStudent != nil {
		var grants []domain.StudentAchievement
		if err := s.db.WithContext(ctx).
			Where("student_id = ? AND log_id IN ?", pc.CurrentStudent.ID, logIDs).
			Find(&grants).Error; err != nil {
			return nil, err
		}
		if len(grants) > 0 {
			achIDs := make([]uint64, 0, len(grants))
			for _, g := range grants {
				achIDs = append(achIDs, g.AchievementID)
			}
			var achs []domain.Achievement
			if err := s.db.WithContext(ctx).Where("id IN ?", achIDs).Find(&achs).Error; err != nil {
				return nil, err
			}
			byID := make(map[uint64]domain.Achievement, len(achs))
			for _, a := range achs {
				byID[a.ID] = a
			}
			for _, g := range grants {
				if g.LogID == nil {
					continue
				}
				if a, ok := byID[g.AchievementID]; ok {
					badgeByLog[*g.LogID] = toBadge(a)
				}
			}
		}
	}

	// 照片点名（仅已确认项；建议对家长不存在）
	pageMediaIDs := make([]uint64, 0)
	for _, ms := range mediaByLog {
		for _, m := range ms {
			pageMediaIDs = append(pageMediaIDs, m.ID)
		}
	}
	confirmedByMedia, err := s.confirmedMediaStudents(ctx, pageMediaIDs)
	if err != nil {
		return nil, err
	}

	list := assembleFeedItems(rows, mediaByLog, confirmedByMedia, likedSet, commentCounts, badgeByLog)

	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

// feedScope 把内部发送范围映射为家长端展示层 scope。
// 1=全班，2=个体；其他值防御性归为 class（当前可见性查询只返回 1/2）。
// 不直接暴露 target_type，避免客户端耦合服务端数字枚举。
func feedScope(targetType int8) string {
	if targetType == 2 {
		return "individual"
	}
	return "class"
}

// assembleFeedItems 把批量查到的关联数据装配成成长时光列表条目。
// 抽成纯函数是为了可测：响应形状（含「不含冗余字段」）是 spec 的验收点，
// 而批量查询本身依赖真实库，由集成验证覆盖。
func assembleFeedItems(
	rows []domain.GrowthDailyLog,
	mediaByLog map[uint64][]domain.GrowthMedia,
	confirmedByMedia map[uint64][]uint64,
	likedSet map[uint64]bool,
	commentCounts map[uint64]int64,
	badgeByLog map[uint64]map[string]any,
) []map[string]any {
	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		// 只输出卡片实际渲染的内容，以及「只看孩子 / 看全部」两模式必需的 scope。
		// 列表不承载详情页才需要的信息（related_students），
		// 也不输出设计稿从不渲染的冗余字段（target_type/class_name/likes_count/media_count/performance_tags）。
		m := map[string]any{
			"id":             r.ID,
			"scope":          feedScope(r.TargetType),
			"date":           r.Date.Format("2006-01-02"),
			"content":        r.Content,
			"created_at":     r.CreatedAt.UTC().Format(time.RFC3339),
			"media":          toFeedMedia(mediaByLog[r.ID], confirmedByMedia),
			"is_liked":       likedSet[r.ID],
			"comments_count": commentCounts[r.ID],
		}
		if r.Creator != nil {
			m["creator"] = map[string]any{
				"id":       r.CreatorID,
				"nickname": r.Creator.Nickname,
				"avatar":   r.Creator.Avatar,
			}
		}
		// 徽章只在「本条记录是该学生某次获奖的见证」时出现（design D1：见证可空）。
		// 无见证的徽章只体现在已获数里，不落到任何卡片上。
		if b, ok := badgeByLog[r.ID]; ok {
			m["badge"] = b
		}
		list[i] = m
	}
	return list
}

// toBadge 徽章的对外形状。配色不落库——前端按 category 映射 design token 语义色（design D3）。
func toBadge(a domain.Achievement) map[string]any {
	return map[string]any{
		"name":       a.Name,
		"icon":       a.Icon,
		"color_kind": badgeColorKind(a.Category),
	}
}

// badgeColorKind 把徽章类别映射到前端可直接选 token 的色类。
// 只表达「哪套语义色」，不带具体色值——色值 SoT 是 tokens.json。
func badgeColorKind(category string) string {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "attendance", "task", "tasks", "points", "point":
		return "info"
	case "growth", "art", "arts":
		return "warning"
	default:
		return "info"
	}
}

// feedMediaItem 附件的对外形状。
// 只暴露卡片/详情真正用到的字段；GrowthMedia 其余字段（class_id/log_id/uploader_id/
// file_name/file_size/sort_num/时间戳）是服务端记账信息，序列化出去属于实现泄漏内部字段，
// 与本次砍掉的 target_type/class_name 同类。
// StudentIDs 为照片上「已确认点名」的孩子（家长端筛选与照片标识的依据，design D7）。
type feedMediaItem struct {
	ID         uint64   `json:"id"`
	FileURL    string   `json:"file_url"`
	ThumbURL   string   `json:"thumb_url"`
	FileType   string   `json:"file_type"`
	Duration   int      `json:"duration"`
	StudentIDs []uint64 `json:"student_ids"`
}

func toFeedMedia(rows []domain.GrowthMedia, confirmedByMedia map[uint64][]uint64) []feedMediaItem {
	// 恒返回非 nil 切片：无附件时是 [] 而非 null，前端按 length 判断是否渲染照片区。
	out := make([]feedMediaItem, 0, len(rows))
	for _, m := range rows {
		ids := confirmedByMedia[m.ID]
		if ids == nil {
			ids = []uint64{}
		}
		out = append(out, feedMediaItem{
			ID:         m.ID,
			FileURL:    m.FileURL,
			ThumbURL:   m.ThumbURL,
			FileType:   m.FileType,
			Duration:   m.Duration,
			StudentIDs: ids,
		})
	}
	return out
}

// confirmedMediaStudents 批量取照片的「已确认」点名（仅 status=1/NULL）。
// status=0 的 AI 建议对家长不存在——这是信任模型的落点（design D6/D7）。
func (s *ParentPortalService) confirmedMediaStudents(ctx context.Context, mediaIDs []uint64) (map[uint64][]uint64, error) {
	out := make(map[uint64][]uint64)
	if len(mediaIDs) == 0 {
		return out, nil
	}
	type mediaStudentRow struct {
		MediaID   uint64
		StudentID uint64
	}
	var rows []mediaStudentRow
	if err := s.db.WithContext(ctx).Model(&domain.GrowthMediaStudent{}).
		Select("media_id, student_id").
		Where("media_id IN ? AND (status IS NULL OR status = ?)", mediaIDs, domain.MediaStudentStatusConfirmed).
		Order("id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.MediaID] = append(out[r.MediaID], r.StudentID)
	}
	return out, nil
}

// splitPerformanceTags 把库里的逗号串解析成标签数组。
// 契约与 detail.vue 的「表现标签」区块都按数组消费；空串返回空数组而非 [""],
// 否则前端 v-for 会渲染出一个空标签。
func splitPerformanceTags(raw string) []string {
	tags := []string{}
	for _, t := range strings.Split(raw, ",") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// ==================== Pickup ====================

func pickupLabel(t domain.PickupType) string {
	switch t {
	case domain.PickupTypeDropOff:
		return "送入"
	case domain.PickupTypePickUp:
		return "接走"
	default:
		return "未知"
	}
}

// ListPickupRecords 家长查看孩子的接送记录（分页）。
func (s *ParentPortalService) ListPickupRecords(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if len(pc.Students) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	studentIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students {
		studentIDs[i] = st.ID
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, total, err := s.pickup.ListRecordsByStudentIDs(ctx, pc.KindergartenID, studentIDs, offset, pageSize)
	if err != nil {
		return nil, err
	}

	type item struct {
		ID             uint64 `json:"id"`
		Type           int8   `json:"type"`
		TypeLabel      string `json:"type_label"`
		Date           string `json:"date"`
		StudentName    string `json:"student_name"`
		ClassName      string `json:"class_name"`
		PickupName     string `json:"pickup_name"`
		PickupRelation string `json:"pickup_relation"`
		RecordedAt     string `json:"recorded_at"`
		// IsAbnormal 园所是否标记该次接送为异常（老师手动标记）。
		// 家长应能看到某次接送被园所标异常（对齐 admin-05 明细的「状态」列语义）。
		IsAbnormal bool `json:"is_abnormal"`
	}

	list := make([]item, 0, len(rows))
	for _, r := range rows {
		list = append(list, item{
			ID:             r.ID,
			Type:           int8(r.Type),
			TypeLabel:      pickupLabel(r.Type),
			Date:           r.RecordedAt.Format("2006-01-02"),
			StudentName:    r.StudentName,
			ClassName:      r.ClassName,
			PickupName:     r.PickupName,
			PickupRelation: r.PickupRelation,
			RecordedAt:     r.RecordedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
			IsAbnormal:     r.IsAbnormal,
		})
	}

	return map[string]any{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	}, nil
}

func RelationLabel(r domain.ParentRelation) string {
	m := map[domain.ParentRelation]string{
		domain.RelationFather:      "爸爸",
		domain.RelationMother:      "妈妈",
		domain.RelationGrandfather: "爷爷",
		domain.RelationGrandmother: "奶奶",
		domain.RelationOther:       "其他",
	}
	if v, ok := m[r]; ok {
		return v
	}
	return string(r)
}

// ==================== Authorized Persons ====================

// CreateAuthorizedPersonInput 家长创建授权接送人入参。
type CreateAuthorizedPersonInput struct {
	StudentID       uint64 `json:"student_id"`
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	SmsCode         string `json:"sms_code"`
	IDCardNumber    string `json:"id_card_number"`
	Avatar          string `json:"avatar"`
	Relation        string `json:"relation"`
	AuthType        string `json:"auth_type"` // permanent / temporary
	AllowPickup     *bool  `json:"allow_pickup"`
	AllowViewRecord *bool  `json:"allow_view_record"`
	AllowViewStatus *bool  `json:"allow_view_status"`
	AllowGrowthView *bool  `json:"allow_growth_view"`
	NotifyOnPickup  *bool  `json:"notify_on_pickup"`
}

// UpdateAuthorizedPersonInput 家长编辑授权接送人入参。
type UpdateAuthorizedPersonInput struct {
	Name            string `json:"name"`
	Phone           string `json:"phone"`
	Avatar          string `json:"avatar"`
	Relation        string `json:"relation"`
	AuthType        string `json:"auth_type"`
	AllowPickup     *bool  `json:"allow_pickup"`
	AllowViewRecord *bool  `json:"allow_view_record"`
	AllowViewStatus *bool  `json:"allow_view_status"`
	AllowGrowthView *bool  `json:"allow_growth_view"`
	NotifyOnPickup  *bool  `json:"notify_on_pickup"`
	ExpireAt        string `json:"expire_at"`
}

// ListAuthorizedPersons 家长查看授权接送人列表。
func (s *ParentPortalService) ListAuthorizedPersons(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if len(pc.Students) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	studentIDs := make([]uint64, len(pc.Students))
	studentIDSet := make(map[uint64]bool)
	for i, st := range pc.Students {
		studentIDs[i] = st.ID
		studentIDSet[st.ID] = true
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 50 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rows, total, err := s.pickup.ListAuthorizationsByStudentIDs(ctx, pc.KindergartenID, studentIDs, offset, pageSize)
	if err != nil {
		return nil, err
	}

	authTypeLabel := func(t string) string {
		if t == "temporary" {
			return "临时授权"
		}
		return "长期授权"
	}
	reviewStatusLabel := func(s int8) string {
		switch s {
		case 0:
			return "待审核"
		case 1:
			return "已通过"
		case 2:
			return "已拒绝"
		default:
			return "未知"
		}
	}

	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		item := map[string]any{
			"id":                  r.ID,
			"student_id":          r.StudentID,
			"student_name":        r.StudentName,
			"class_name":          r.ClassName,
			"name":                r.Name,
			"phone":               maskPhone(r.Phone),
			"relation":            r.Relation,
			"avatar":              r.Avatar,
			"auth_type":           r.AuthType,
			"auth_type_label":     authTypeLabel(r.AuthType),
			"allow_pickup":        r.AllowPickup,
			"allow_view_record":   r.AllowViewRecord,
			"allow_view_status":   r.AllowViewStatus,
			"allow_growth_view":   r.AllowGrowthView,
			"notify_on_pickup":    r.NotifyOnPickup,
			"status":              r.Status,
			"review_status":       r.ReviewStatus,
			"review_status_label": reviewStatusLabel(r.ReviewStatus),
		}
		if r.ExpireAt != nil {
			item["expire_at"] = r.ExpireAt.UTC().Format(time.RFC3339)
		}
		list = append(list, item)
	}

	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

// CreateAuthorizedPerson 家长添加授权接送人。
func (s *ParentPortalService) CreateAuthorizedPerson(ctx context.Context, pc *ParentPortalContext, req CreateAuthorizedPersonInput) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	// 校验必填: name, phone, relation
	if req.Name == "" || req.Phone == "" || req.Relation == "" {
		return nil, ErrParentInvalidParams
	}
	// 校验学生归属
	found := false
	for _, st := range pc.Students {
		if st.ID == pc.CurrentStudent.ID {
			found = true
			break
		}
	}
	if !found {
		return nil, ErrParentForbidden
	}

	now := time.Now()

	allowPickup := true
	if req.AllowPickup != nil {
		allowPickup = *req.AllowPickup
	}
	allowViewRecord := false
	if req.AllowViewRecord != nil {
		allowViewRecord = *req.AllowViewRecord
	}
	allowViewStatus := false
	if req.AllowViewStatus != nil {
		allowViewStatus = *req.AllowViewStatus
	}
	allowGrowthView := false
	if req.AllowGrowthView != nil {
		allowGrowthView = *req.AllowGrowthView
	}
	notifyOnPickup := true
	if req.NotifyOnPickup != nil {
		notifyOnPickup = *req.NotifyOnPickup
	}

	authType := req.AuthType
	if authType == "" {
		authType = "permanent"
	}

	auth := &domain.PickupAuthorization{
		StudentID:       pc.CurrentStudent.ID,
		Name:            req.Name,
		Phone:           req.Phone,
		Relation:        req.Relation,
		IDCard:          req.IDCardNumber,
		Avatar:          req.Avatar,
		Status:          1, // 1=有效
		CreatorID:       pc.UserID,
		AuthType:        authType,
		AllowPickup:     allowPickup,
		AllowViewRecord: allowViewRecord,
		AllowViewStatus: allowViewStatus,
		AllowGrowthView: allowGrowthView,
		NotifyOnPickup:  notifyOnPickup,
		ReviewStatus:    0, // 0=待审核
	}

	if err := s.pickup.CreateAuthorization(ctx, auth); err != nil {
		return nil, err
	}

	statusLabel := "待审核"
	if auth.ReviewStatus == 1 {
		statusLabel = "已通过"
	}

	return map[string]any{
		"id":           auth.ID,
		"status":       auth.ReviewStatus,
		"status_label": statusLabel,
		"created_at":   now.UTC().Format(time.RFC3339),
	}, nil
}

// UpdateAuthorizedPerson 家长编辑授权接送人。
func (s *ParentPortalService) UpdateAuthorizedPerson(ctx context.Context, pc *ParentPortalContext, authID uint64, req UpdateAuthorizedPersonInput) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	// 归属校验
	existing, err := s.pickup.FindAuthorizationByStudent(ctx, pc.CurrentStudent.ID, authID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrParentNotFound
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Relation != "" {
		updates["relation"] = req.Relation
	}
	if req.AuthType != "" {
		updates["auth_type"] = req.AuthType
	}
	if req.AllowPickup != nil {
		updates["allow_pickup"] = *req.AllowPickup
	}
	if req.AllowViewRecord != nil {
		updates["allow_view_record"] = *req.AllowViewRecord
	}
	if req.AllowViewStatus != nil {
		updates["allow_view_status"] = *req.AllowViewStatus
	}
	if req.AllowGrowthView != nil {
		updates["allow_growth_view"] = *req.AllowGrowthView
	}
	if req.NotifyOnPickup != nil {
		updates["notify_on_pickup"] = *req.NotifyOnPickup
	}
	if req.ExpireAt != "" {
		t, err := time.Parse(time.RFC3339, req.ExpireAt)
		if err == nil {
			updates["expire_at"] = t
		}
	}

	if len(updates) == 0 {
		return map[string]any{"id": authID, "updated_at": time.Now().UTC().Format(time.RFC3339)}, nil
	}
	updates["updated_at"] = time.Now()

	if err := s.pickup.UpdateAuthorizationFields(ctx, pc.CurrentStudent.ID, authID, updates); err != nil {
		return nil, err
	}

	return map[string]any{"id": authID, "updated_at": time.Now().UTC().Format(time.RFC3339)}, nil
}

// DeleteAuthorizedPerson 家长删除授权接送人（软删除）。
func (s *ParentPortalService) DeleteAuthorizedPerson(ctx context.Context, pc *ParentPortalContext, authID uint64) error {
	if pc.CurrentStudent == nil {
		return ErrParentInvalidParams
	}
	existing, err := s.pickup.FindAuthorizationByStudent(ctx, pc.CurrentStudent.ID, authID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrParentNotFound
	}
	return s.pickup.SoftDeleteAuthorization(ctx, pc.CurrentStudent.ID, authID)
}

// ==================== 今日接走人指定 ====================

// DesignatePickupInput 指定今日接走人入参。
type DesignatePickupInput struct {
	AuthorizationID uint64 `json:"authorization_id"`
}

// DesignatePickupPerson 家长指定「今日接走人」。
//
// 指定结果覆盖该孩当日此前的指定，并使该孩当日已生成的接送凭证全部失效
// （旧凭证绑的是前任接走人，换人即作废）。
// 被指定人必须是该孩已审核通过且有效（未撤销、未过期）的授权接送人（I1）。
//
// 与凭证分离的原因见 domain.PickupDailyDesignation 注释：
// 码一次性易失，指定须独立存续，才能回答「原计划谁来接」。
func (s *ParentPortalService) DesignatePickupPerson(ctx context.Context, pc *ParentPortalContext, req DesignatePickupInput) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	if req.AuthorizationID == 0 {
		return nil, ErrParentInvalidParams
	}

	// 归属校验：该授权人必须属于当前学生
	auth, err := s.pickup.FindAuthorizationByStudent(ctx, pc.CurrentStudent.ID, req.AuthorizationID)
	if err != nil {
		return nil, err
	}
	if auth == nil {
		return nil, ErrParentForbidden
	}

	// I1：被指定人须在有效授权集合内
	if !authorizationValid(auth, time.Now()) {
		return nil, ErrParentForbidden
	}

	// 覆盖式指定
	if err := s.pickup.UpsertDailyDesignation(
		ctx, pc.KindergartenID, pc.CurrentStudent.ID, req.AuthorizationID, time.Now(),
	); err != nil {
		return nil, err
	}

	// 换人即作废当日全部旧凭证
	if err := s.pickup.DeactivateStudentQRCodes(ctx, pc.CurrentStudent.ID); err != nil {
		return nil, err
	}

	return map[string]any{
		"student_id":       pc.CurrentStudent.ID,
		"authorization_id": req.AuthorizationID,
		"person": map[string]any{
			"name":     auth.Name,
			"relation": auth.Relation,
			"avatar":   auth.Avatar,
		},
		"designated_at": time.Now().UTC().Format(time.RFC3339),
	}, nil
}

// ==================== Pickup QR Code ====================

// GetQRCode 家长获取接送二维码（绑「学生 + 今日接走人」）。
//
// 前提：已通过 DesignatePickupPerson 指定今日接走人。
// 凭证当日过期、一次有效；生成时会再次校验被指定人仍在有效授权集合内。
func (s *ParentPortalService) GetQRCode(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}

	now := time.Now()

	// 必须先指定今日接走人
	desig, err := s.pickup.FindDailyDesignation(ctx, pc.CurrentStudent.ID, now)
	if err != nil {
		return nil, err
	}
	if desig == nil {
		return nil, ErrParentInvalidParams
	}

	// 再校验被指定人仍在有效授权集合内（I1 二次校验）
	auth, err := s.pickup.FindAuthorizationByStudent(ctx, pc.CurrentStudent.ID, desig.AuthorizationID)
	if err != nil {
		return nil, err
	}
	if auth == nil || !authorizationValid(auth, now) {
		return nil, ErrParentForbidden
	}

	person := map[string]any{
		"name":     auth.Name,
		"relation": auth.Relation,
		"photo":    auth.Avatar,
	}
	student := map[string]any{
		"id":    pc.CurrentStudent.ID,
		"name":  pc.CurrentStudent.Name,
		"photo": pc.CurrentStudent.Avatar, // Student 域无独立 photo 字段，复用 avatar
	}

	// 已有有效凭证且仍绑定同一接走人 → 直接复用
	if qr, err := s.pickup.FindActiveQRCodeByStudent(ctx, pc.CurrentStudent.ID); err != nil {
		return nil, err
	} else if qr != nil && qr.ExpireAt.After(now) && qr.AuthorizationID == desig.AuthorizationID {
		return map[string]any{
			"qr_code":   qr.Code,
			"student":   student,
			"person":    person,
			"expire_at": qr.ExpireAt.UTC().Format(time.RFC3339),
		}, nil
	}

	// 失效旧码后生成新码（绑指定的接走人）
	if err := s.pickup.DeactivateStudentQRCodes(ctx, pc.CurrentStudent.ID); err != nil {
		return nil, err
	}

	dayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	newQR := &domain.PickupQRCode{
		StudentID:       pc.CurrentStudent.ID,
		AuthorizationID: desig.AuthorizationID,
		Code:            generateQRCodeToken(),
		ExpireAt:        dayEnd,
		IsActive:        true,
	}
	if err := s.pickup.CreateQRCode(ctx, newQR); err != nil {
		return nil, err
	}

	return map[string]any{
		"qr_code":   newQR.Code,
		"student":   student,
		"person":    person,
		"expire_at": newQR.ExpireAt.UTC().Format(time.RFC3339),
	}, nil
}

// generateQRCodeToken 生成接送二维码 token。
//
// 纯随机串，不内嵌 student_id 等可枚举标识（E4）：
// 学生与接走人由后端按码查库解出，凭证本身不泄露任何 ID。
func generateQRCodeToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ==================== Leaves ====================

// ParentLeaveInput 家长请假输入
type ParentLeaveInput struct {
	StudentID uint64 `json:"student_id"`
	Reason    string `json:"reason"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Days      int    `json:"days"`
}

func parentLeaveMap(lr domain.LeaveRequest) map[string]any {
	student := map[string]any{"id": lr.StudentID, "name": "学生", "avatar": ""}
	if lr.Student != nil {
		student["name"] = lr.Student.Name
		student["avatar"] = lr.Student.Avatar
	}
	applicant := map[string]any{"id": lr.ApplicantID, "nickname": "家长"}
	if lr.Applicant != nil && lr.Applicant.Nickname != "" {
		applicant["nickname"] = lr.Applicant.Nickname
	}
	statusLabel := "待审批"
	switch lr.Status {
	case domain.LeaveStatusApproved:
		statusLabel = "已同意"
	case domain.LeaveStatusRejected:
		statusLabel = "已拒绝"
	case domain.LeaveStatusCanceled:
		statusLabel = "已取消"
	}
	return map[string]any{
		"id": lr.ID, "student": student, "applicant": applicant,
		"reason": lr.Reason, "start_date": lr.StartDate.Format("2006-01-02"),
		"end_date": lr.EndDate.Format("2006-01-02"), "days": lr.Days,
		"status": lr.Status, "status_label": statusLabel,
		"review_remark": lr.ReviewRemark,
		"created_at":    lr.CreatedAt.UTC().Format(time.RFC3339),
		"reviewed_at":   formatOptionalTime(lr.ReviewedAt),
	}
}

func (s *ParentPortalService) ListLeaves(ctx context.Context, pc *ParentPortalContext, status string, page, pageSize int) (map[string]any, error) {
	offset := (page - 1) * pageSize
	db := s.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Where("kindergarten_id = ? AND applicant_id = ?", pc.KindergartenID, pc.UserID)
	switch status {
	case "pending":
		db = db.Where("status = ?", domain.LeaveStatusPending)
	case "approved":
		db = db.Where("status = ?", domain.LeaveStatusApproved)
	case "rejected":
		db = db.Where("status = ?", domain.LeaveStatusRejected)
	case "canceled":
		db = db.Where("status = ?", domain.LeaveStatusCanceled)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []domain.LeaveRequest
	if err := db.Preload("Student").Preload("Applicant").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		list[i] = parentLeaveMap(r)
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) GetLeave(ctx context.Context, pc *ParentPortalContext, id uint64) (map[string]any, error) {
	var lr domain.LeaveRequest
	if err := s.db.WithContext(ctx).Preload("Student").Preload("Applicant").
		Where("id = ? AND kindergarten_id = ? AND applicant_id = ?", id, pc.KindergartenID, pc.UserID).
		First(&lr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	return parentLeaveMap(lr), nil
}

func (s *ParentPortalService) CreateLeave(ctx context.Context, pc *ParentPortalContext, req ParentLeaveInput) (map[string]any, error) {
	if req.StudentID == 0 || strings.TrimSpace(req.Reason) == "" || req.StartDate == "" || req.EndDate == "" {
		return nil, ErrParentInvalidParams
	}
	startDate, err := time.ParseInLocation("2006-01-02", req.StartDate, time.Local)
	if err != nil {
		return nil, ErrParentInvalidParams
	}
	endDate, err := time.ParseInLocation("2006-01-02", req.EndDate, time.Local)
	if err != nil {
		return nil, ErrParentInvalidParams
	}
	if endDate.Before(startDate) {
		return nil, ErrParentInvalidParams
	}

	// Verify student belongs to this parent
	studentFound := false
	var studentClassID uint64
	for _, s := range pc.Students {
		if s.ID == req.StudentID {
			studentFound = true
			studentClassID = s.ClassID
			break
		}
	}
	if !studentFound {
		return nil, ErrParentForbidden
	}

	days := req.Days
	if days <= 0 {
		days = int(endDate.Sub(startDate).Hours()/24) + 1
	}

	lr := &domain.LeaveRequest{
		KindergartenID: pc.KindergartenID, ClassID: studentClassID,
		StudentID: req.StudentID, ApplicantID: pc.UserID,
		Reason: strings.TrimSpace(req.Reason), StartDate: startDate, EndDate: endDate,
		Days: days, Status: domain.LeaveStatusPending,
	}
	if err := s.db.WithContext(ctx).Create(lr).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": lr.ID, "status": int8(lr.Status), "status_label": "待审批", "created_at": lr.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *ParentPortalService) UpdateLeave(ctx context.Context, pc *ParentPortalContext, id uint64, req ParentLeaveInput) error {
	var lr domain.LeaveRequest
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ? AND applicant_id = ?", id, pc.KindergartenID, pc.UserID).First(&lr).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrParentNotFound
		}
		return err
	}
	if lr.Status != domain.LeaveStatusPending {
		return ErrParentInvalidParams
	}
	updates := map[string]any{}
	if req.Reason != "" {
		updates["reason"] = strings.TrimSpace(req.Reason)
	}
	if req.StartDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", req.StartDate, time.Local); err == nil {
			updates["start_date"] = t
		}
	}
	if req.EndDate != "" {
		if t, err := time.ParseInLocation("2006-01-02", req.EndDate, time.Local); err == nil {
			updates["end_date"] = t
		}
	}
	if req.Days > 0 {
		updates["days"] = req.Days
	}
	if len(updates) == 0 {
		return nil
	}
	s.db.WithContext(ctx).Model(&lr).Updates(updates)
	return nil
}

func (s *ParentPortalService) CancelLeave(ctx context.Context, pc *ParentPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Model(&domain.LeaveRequest{}).
		Where("id = ? AND kindergarten_id = ? AND applicant_id = ?", id, pc.KindergartenID, pc.UserID).
		Update("status", domain.LeaveStatusCanceled)
	if res.RowsAffected == 0 {
		return ErrParentNotFound
	}
	return nil
}

// ==================== Notices ====================

// MarkNoticeRead 标记通知已读。
func (s *ParentPortalService) MarkNoticeRead(ctx context.Context, pc *ParentPortalContext, noticeID uint64) error {
	return s.notice.RecordReadLog(ctx, noticeID, pc.UserID)
}

func (s *ParentPortalService) ListNotices(ctx context.Context, pc *ParentPortalContext, category string, page, pageSize int) (map[string]any, error) {
	classIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students {
		classIDs[i] = st.ClassID
	}
	if len(classIDs) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	// EXISTS 命中判定：任一孩子班级命中目标集即可见，一条通知只出一行（多孩合并）
	db := s.notice.FilterByTargetHit(s.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ? AND status = ?", pc.KindergartenID, domain.NoticeStatusPublished), classIDs)
	if category != "" {
		db = db.Where("category = ?", category)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	var rows []domain.Notice
	if err := db.Order("published_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	list := make([]map[string]any, len(rows))
	for i, n := range rows {
		var readLog domain.NoticeReadLog
		_ = s.db.WithContext(ctx).Where("notice_id = ? AND user_id = ?", n.ID, pc.UserID).First(&readLog).Error
		isRead := !readLog.ReadAt.IsZero()
		isConfirmed := readLog.ConfirmedAt != nil
		catLabel := string(n.Category)
		list[i] = map[string]any{
			"id": n.ID, "category": n.Category, "category_label": catLabel,
			"title": n.Title, "content": n.Content, "need_confirm": n.NeedConfirm,
			"status": n.Status, "published_at": formatOptionalTime(n.PublishedAt),
			"created_at": n.CreatedAt.UTC().Format(time.RFC3339),
			"is_read":    isRead, "is_confirmed": isConfirmed,
		}
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) GetNotice(ctx context.Context, pc *ParentPortalContext, id uint64) (map[string]any, error) {
	classIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students {
		classIDs[i] = st.ClassID
	}
	var n domain.Notice
	q := s.notice.FilterByTargetHit(s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ? AND status = ?", id, pc.KindergartenID, domain.NoticeStatusPublished), classIDs)
	if err := q.First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	var readLog domain.NoticeReadLog
	_ = s.db.WithContext(ctx).Where("notice_id = ? AND user_id = ?", n.ID, pc.UserID).First(&readLog).Error
	isRead := !readLog.ReadAt.IsZero()
	isConfirmed := readLog.ConfirmedAt != nil
	return map[string]any{
		"id": n.ID, "category": n.Category, "title": n.Title, "content": n.Content,
		"need_confirm": n.NeedConfirm, "status": n.Status,
		"published_at": formatOptionalTime(n.PublishedAt),
		"created_at":   n.CreatedAt.UTC().Format(time.RFC3339),
		"is_read":      isRead, "is_confirmed": isConfirmed,
	}, nil
}

func (s *ParentPortalService) ConfirmNotice(ctx context.Context, pc *ParentPortalContext, id uint64) (map[string]any, error) {
	classIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students {
		classIDs[i] = st.ClassID
	}
	var n domain.Notice
	// 可见性以目标集判定：非目标家长不可确认（确认按用户一次，对全部孩子生效）
	q := s.notice.FilterByTargetHit(s.db.WithContext(ctx).
		Where("id = ? AND kindergarten_id = ?", id, pc.KindergartenID), classIDs)
	if err := q.First(&n).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	now := time.Now()
	var log domain.NoticeReadLog
	result := s.db.WithContext(ctx).Where("notice_id = ? AND user_id = ?", id, pc.UserID).First(&log)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		s.db.WithContext(ctx).Create(&domain.NoticeReadLog{NoticeID: id, UserID: pc.UserID, ReadAt: now, ConfirmedAt: &now})
	} else {
		s.db.WithContext(ctx).Model(&log).Updates(map[string]any{"read_at": now, "confirmed_at": now})
	}
	return map[string]any{"id": id, "is_confirmed": true, "confirmed_at": now.UTC().Format(time.RFC3339)}, nil
}

// ==================== Growth Detail / Comments / Likes ====================

func (s *ParentPortalService) verifyGrowthAccess(ctx context.Context, pc *ParentPortalContext, logID uint64) (*domain.GrowthDailyLog, error) {
	var log domain.GrowthDailyLog
	if err := s.db.WithContext(ctx).Where("id = ? AND kindergarten_id = ?", logID, pc.KindergartenID).First(&log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	if log.TargetType == 1 {
		// 全班记录：验证孩子所在班级
		for _, st := range pc.Students {
			if st.ClassID == log.ClassID {
				return &log, nil
			}
		}
		return nil, ErrParentForbidden
	}
	// 指定学生记录：验证目标学生在家长孩子中
	var count int64
	s.db.WithContext(ctx).Model(&domain.GrowthLogStudent{}).
		Where("log_id = ? AND student_id IN ?", logID, studentIDs(pc.Students)).Count(&count)
	if count == 0 {
		return nil, ErrParentForbidden
	}
	return &log, nil
}

func studentIDs(students []domain.Student) []uint64 {
	ids := make([]uint64, len(students))
	for i, s := range students {
		ids[i] = s.ID
	}
	return ids
}

// growthLogVisibilityWhere 是家长端成长记录可见性规则的唯一实现点：
// 「记录属于孩子所在班级（target_type=1）」或「记录明确发给了孩子本人（target_type=2）」。
// 输入学生为空时返回恒假条件；student 非 nil 时只按该学生判断。
// 纯函数形态是为了让规则本身可被单元测试直接断言。
func growthLogVisibilityWhere(kgID uint64, students []domain.Student, student *domain.Student) (string, []any) {
	if student != nil {
		return "growth_daily_logs.kindergarten_id = ? AND growth_daily_logs.class_id = ? AND (growth_daily_logs.target_type = 1 OR (growth_daily_logs.target_type = 2 AND growth_daily_logs.id IN (SELECT log_id FROM growth_log_students WHERE student_id = ?)))",
			[]any{kgID, student.ClassID, student.ID}
	}
	ids := studentIDs(students)
	if len(ids) == 0 {
		return "1 = 0", nil
	}
	classIDs := make([]uint64, 0, len(ids))
	seen := make(map[uint64]bool, len(ids))
	for _, st := range students {
		if !seen[st.ClassID] {
			seen[st.ClassID] = true
			classIDs = append(classIDs, st.ClassID)
		}
	}
	return "growth_daily_logs.kindergarten_id = ? AND growth_daily_logs.class_id IN ? AND (growth_daily_logs.target_type = 1 OR (growth_daily_logs.target_type = 2 AND growth_daily_logs.id IN (SELECT log_id FROM growth_log_students WHERE student_id IN ?)))",
		[]any{kgID, classIDs, ids}
}

func (s *ParentPortalService) GrowthFeedDetail(ctx context.Context, pc *ParentPortalContext, logID uint64) (map[string]any, error) {
	log, err := s.verifyGrowthAccess(ctx, pc, logID)
	if err != nil {
		return nil, err
	}

	m := map[string]any{
		"id": log.ID, "date": log.Date.Format("2006-01-02"), "target_type": log.TargetType,
		"content": log.Content, "performance_tags": splitPerformanceTags(log.PerformanceTags),
		"breakfast_rating": log.BreakfastRating, "lunch_rating": log.LunchRating,
		"dinner_rating": log.DinnerRating, "nap_duration": log.NapDuration,
		"sleep_quality": log.SleepQuality, "morning_mood": log.MorningMood,
		"afternoon_mood": log.AfternoonMood, "sync_to_feed": log.SyncToFeed,
		"created_at": log.CreatedAt.UTC().Format(time.RFC3339),
	}

	// 相关孩子是详情页语义（teacher-08 详情设计的「相关孩子」区块），只在发给指定学生的记录上有值。
	if log.TargetType == 2 {
		var linked []domain.Student
		if err := s.db.WithContext(ctx).
			Joins("INNER JOIN growth_log_students gls ON gls.student_id = students.id AND gls.log_id = ?", logID).
			Find(&linked).Error; err != nil {
			return nil, err
		}
		rs := make([]map[string]any, 0, len(linked))
		for _, st := range linked {
			rs = append(rs, map[string]any{"id": st.ID, "name": st.Name, "avatar": st.Avatar})
		}
		m["related_students"] = rs
	}

	// 徽章：本条记录是当前孩子某次获奖的见证时才有
	if pc.CurrentStudent != nil {
		var g domain.StudentAchievement
		err := s.db.WithContext(ctx).
			Where("student_id = ? AND log_id = ?", pc.CurrentStudent.ID, logID).
			First(&g).Error
		if err == nil {
			var a domain.Achievement
			if err := s.db.WithContext(ctx).First(&a, g.AchievementID).Error; err == nil {
				m["badge"] = toBadge(a)
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	var creator domain.User
	if err := s.db.WithContext(ctx).First(&creator, log.CreatorID).Error; err == nil {
		m["creator"] = map[string]any{"id": creator.ID, "nickname": creator.Nickname, "avatar": creator.Avatar}
	}

	var media []domain.GrowthMedia
	s.db.WithContext(ctx).Where("log_id = ?", logID).Order("sort_num ASC, id ASC").Find(&media)
	mediaIDs := make([]uint64, 0, len(media))
	for _, md := range media {
		mediaIDs = append(mediaIDs, md.ID)
	}
	confirmedByMedia, err := s.confirmedMediaStudents(ctx, mediaIDs)
	if err != nil {
		return nil, err
	}
	m["media"] = toFeedMedia(media, confirmedByMedia)

	var likesCount int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", logID).Count(&likesCount)
	var myLike int64
	s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ? AND user_id = ?", logID, pc.UserID).Count(&myLike)
	m["likes_count"] = likesCount
	m["is_liked"] = myLike > 0

	var commentsCount int64
	s.db.WithContext(ctx).Model(&domain.GrowthComment{}).Where("log_id = ?", logID).Count(&commentsCount)
	m["comments_count"] = commentsCount

	return m, nil
}

func (s *ParentPortalService) LikeGrowth(ctx context.Context, pc *ParentPortalContext, logID uint64) (map[string]any, error) {
	if _, err := s.verifyGrowthAccess(ctx, pc, logID); err != nil {
		return nil, err
	}
	// growth_likes 是软删除模型，而唯一索引 idx_log_user(log_id, user_id) 不含
	// deleted_at —— 取消点赞留下的软删行会占住唯一键，直接 Create 会撞键失败。
	// 所以先 Unscoped 查（含软删行）：软删的就地复活，查不到才插入。
	// 写入错误必须向上抛：此前忽略 Create 错误，接口会返回 is_liked:true 而库里没有行。
	var existing domain.GrowthLike
	err := s.db.WithContext(ctx).Unscoped().
		Where("log_id = ? AND user_id = ?", logID, pc.UserID).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		// 并发双击时两个请求都会查不到行，唯一索引 idx_log_user 只允许一个 Create 成功；
		// 点赞本身幂等，撞键说明对方已写入，不能把「点赞失败」抛给客户端。
		if e := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
			Create(&domain.GrowthLike{LogID: logID, UserID: pc.UserID}).Error; e != nil {
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
	var count int64
	if e := s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", logID).Count(&count).Error; e != nil {
		return nil, e
	}
	return map[string]any{"is_liked": true, "likes_count": count}, nil
}

func (s *ParentPortalService) UnlikeGrowth(ctx context.Context, pc *ParentPortalContext, logID uint64) (map[string]any, error) {
	if _, err := s.verifyGrowthAccess(ctx, pc, logID); err != nil {
		return nil, err
	}
	// 硬删：点赞没有审计价值；留软删行只会占住 idx_log_user 唯一键（见 LikeGrowth）。
	if e := s.db.WithContext(ctx).Unscoped().
		Where("log_id = ? AND user_id = ?", logID, pc.UserID).
		Delete(&domain.GrowthLike{}).Error; e != nil {
		return nil, e
	}
	var count int64
	if e := s.db.WithContext(ctx).Model(&domain.GrowthLike{}).Where("log_id = ?", logID).Count(&count).Error; e != nil {
		return nil, e
	}
	return map[string]any{"is_liked": false, "likes_count": count}, nil
}

func (s *ParentPortalService) ListGrowthComments(ctx context.Context, pc *ParentPortalContext, logID uint64) (map[string]any, error) {
	if _, err := s.verifyGrowthAccess(ctx, pc, logID); err != nil {
		return nil, err
	}
	var rows []domain.GrowthComment
	if err := s.db.WithContext(ctx).Preload("User").Where("log_id = ?", logID).Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		u := map[string]any{}
		if r.User != nil {
			u = map[string]any{"id": r.User.ID, "nickname": r.User.Nickname, "avatar": r.User.Avatar}
		}
		list[i] = map[string]any{"id": r.ID, "content": r.Content, "parent_id": r.ParentID, "user": u, "created_at": r.CreatedAt.UTC().Format(time.RFC3339)}
	}
	return map[string]any{"list": list, "total": len(rows)}, nil
}

func (s *ParentPortalService) CreateGrowthComment(ctx context.Context, pc *ParentPortalContext, logID uint64, content string, parentID *uint64) (map[string]any, error) {
	if strings.TrimSpace(content) == "" {
		return nil, ErrParentInvalidParams
	}
	if _, err := s.verifyGrowthAccess(ctx, pc, logID); err != nil {
		return nil, err
	}
	row := &domain.GrowthComment{LogID: logID, UserID: pc.UserID, Content: strings.TrimSpace(content)}
	if parentID != nil {
		row.ParentID = parentID
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": row.ID, "created_at": row.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *ParentPortalService) ListGrowthMedia(ctx context.Context, pc *ParentPortalContext, year, month int) (map[string]any, error) {
	db := s.db.WithContext(ctx).Model(&domain.GrowthMedia{}).
		Where("kindergarten_id = ?", pc.KindergartenID)
	if year > 0 && month > 0 {
		start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 1, 0)
		db = db.Where("created_at >= ? AND created_at < ?", start, end)
	}
	// 可见性与成长时光一致：只含孩子所在班级的全班记录媒体，或发给孩子的个体记录媒体。
	visWhere, visArgs := growthLogVisibilityWhere(pc.KindergartenID, pc.Students, nil)
	db = db.Where("log_id IN (SELECT id FROM growth_daily_logs WHERE "+visWhere+")", visArgs...)
	var rows []domain.GrowthMedia
	if err := db.Order("created_at DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return map[string]any{"list": rows, "total": len(rows), "year": year, "month": month}, nil
}

func (s *ParentPortalService) GetGrowthReport(ctx context.Context, pc *ParentPortalContext, studentID uint64, reportType string) (map[string]any, error) {
	// 归属校验：请求的孩子必须在家长的孩子名单内（白名单前置，越权请求不触碰数据）
	var target *domain.Student
	if studentID == 0 {
		if pc.CurrentStudent == nil {
			return nil, ErrParentInvalidParams
		}
		target = pc.CurrentStudent
	} else {
		for i := range pc.Students {
			if pc.Students[i].ID == studentID {
				target = &pc.Students[i]
				break
			}
		}
		if target == nil {
			return nil, ErrParentForbidden
		}
	}
	db := s.db.WithContext(ctx).Model(&domain.GrowthReport{}).
		Preload("Student").
		Where("student_id = ? AND kindergarten_id = ? AND status = ?", target.ID, pc.KindergartenID, 1)
	if reportType != "" {
		db = db.Where("report_type = ?", reportType)
	}
	var report domain.GrowthReport
	if err := db.Order("created_at DESC").First(&report).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[string]any{}, nil
		}
		return nil, err
	}
	var dims []domain.GrowthReportDimension
	s.db.WithContext(ctx).Where("report_id = ?", report.ID).Find(&dims)
	return map[string]any{"report": report, "dimensions": dims}, nil
}

// ==================== Tasks ====================

// taskScopeWhere 是家长端任务可见性的唯一实现点：班级任务按当前孩子所在班级，
// 个体任务只按目标孩子。拆成纯函数便于钉住条件形状（执行结果由集成验证承担）。
func taskScopeWhere(pc *ParentPortalContext) (string, []any) {
	return "kindergarten_id = ? AND ((target_student_id = 0 AND class_id = ?) OR target_student_id = ?)",
		[]any{pc.KindergartenID, pc.CurrentStudent.ClassID, pc.CurrentStudent.ID}
}

func taskScopeQuery(db *gorm.DB, pc *ParentPortalContext) *gorm.DB {
	where, args := taskScopeWhere(pc)
	return db.Where(where, args...)
}

func taskIsActive(t *domain.ParentTask, now time.Time) bool {
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, shanghaiLoc())
	start := time.Date(t.StartDate.Year(), t.StartDate.Month(), t.StartDate.Day(), 0, 0, 0, 0, shanghaiLoc())
	end := time.Date(t.EndDate.Year(), t.EndDate.Month(), t.EndDate.Day(), 0, 0, 0, 0, shanghaiLoc())
	return t.Status == 1 && !day.Before(start) && !day.After(end)
}

// taskPeriodCheckin 返回该孩子在当前周期内的提交（不存在时 found=false）。
//
// 周期口径与提交校验一致：daily 当天、weekly 滚动 7 天、once 整条任务。
// 列表用同一次查询同时得出「已提交」与「已回应」，避免两次读之间的状态漂移。
func taskPeriodCheckin(
	ctx context.Context, db *gorm.DB, t *domain.ParentTask, studentID uint64, now time.Time,
) (domain.TaskCheckIn, bool, error) {
	q := db.WithContext(ctx).Model(&domain.TaskCheckIn{}).Where("task_id = ? AND student_id = ?", t.ID, studentID)
	today := now.Format("2006-01-02")
	switch t.Frequency {
	case domain.TaskFrequencyDaily:
		q = q.Where("date = ?", today)
	case domain.TaskFrequencyWeekly:
		q = q.Where("date >= ?", now.AddDate(0, 0, -6).Format("2006-01-02"))
	}
	var c domain.TaskCheckIn
	// 查询失败必须上报：静默当成「本周期未提交」会让家长看到可点的提交入口，
	// 点下去拿到 422，而实际只是查询失败——把故障伪装成了业务规则。
	err := q.Order("date DESC, id DESC").First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return c, false, nil
	}
	if err != nil {
		return c, false, err
	}
	return c, true, nil
}

func taskSubmittedInPeriod(ctx context.Context, db *gorm.DB, t *domain.ParentTask, studentID uint64, now time.Time) (bool, error) {
	_, found, err := taskPeriodCheckin(ctx, db, t, studentID, now)
	return found, err
}

func (s *ParentPortalService) ListTasks(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	now := time.Now().In(shanghaiLoc())
	today := now.Format("2006-01-02")
	q := taskScopeQuery(s.db.WithContext(ctx).Model(&domain.ParentTask{}), pc).
		Where("status = 1 AND start_date <= ? AND end_date >= ?", today, today)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	offset := (page - 1) * pageSize
	var rows []domain.ParentTask
	if err := q.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	activeRows := make([]domain.ParentTask, 0, len(rows))
	taskIDs := make([]uint64, 0, len(rows))
	for _, r := range rows {
		if !taskIsActive(&r, now) {
			continue
		}
		activeRows = append(activeRows, r)
		taskIDs = append(taskIDs, r.ID)
	}
	// 一次查完本页的未读状态，避免每个任务各查一次。
	unread, err := s.unreadSupplementTaskIDs(ctx, pc, taskIDs)
	if err != nil {
		log.L(ctx).Error("parent list tasks count failed", "scope", "unread", "err", err)
		return nil, err
	}
	for _, r := range activeRows {
		periodCheckin, submitted, err := taskPeriodCheckin(ctx, s.db, &r, pc.CurrentStudent.ID, now)
		if err != nil {
			log.L(ctx).Error("parent list tasks count failed", "scope", "submitted", "task", r.ID, "err", err)
			return nil, err
		}
		// 「已回应」= 本周期提交已被主回应关闭；补充回应不改变 response_id，不影响这一态。
		responded := submitted && periodCheckin.ResponseID > 0
		var childCount int64
		if err := s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
			Where("task_id = ? AND student_id = ?", r.ID, pc.CurrentStudent.ID).Count(&childCount).Error; err != nil {
			log.L(ctx).Error("parent list tasks count failed", "scope", "checkin_count", "task", r.ID, "err", err)
			return nil, err
		}
		list = append(list, map[string]any{
			"id": r.ID, "title": r.Title, "description": r.Description, "intro": r.Intro,
			"learning_link": r.LearningLink, "frequency": r.Frequency, "target_student_id": r.TargetStudentID,
			"allow_public": r.AllowPublic, "recognition_ritual": r.RecognitionRitual,
			"star_scope": r.StarScope, "star_goal": r.StarGoal,
			"start_date": r.StartDate.Format("2006-01-02"), "end_date": r.EndDate.Format("2006-01-02"),
			"status": r.Status, "checked_in_today": submitted, "submitted_current_period": submitted,
			"responded": responded,
			"checkin_count": childCount, "has_unread_supplement": unread[r.ID],
			"created_at": r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) GetTask(ctx context.Context, pc *ParentPortalContext, id uint64) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	var t domain.ParentTask
	if err := taskScopeQuery(s.db.WithContext(ctx), pc).Where("id = ?", id).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	now := time.Now().In(shanghaiLoc())
	if !taskIsActive(&t, now) {
		return nil, ErrParentNotFound
	}
	submitted, err := taskSubmittedInPeriod(ctx, s.db, &t, pc.CurrentStudent.ID, now)
	if err != nil {
		log.L(ctx).Error("parent get task count failed", "scope", "submitted", "task", id, "err", err)
		return nil, err
	}

	var responses []domain.TaskResponse
	// 一条 period 回应会覆盖同一孩子的多条提交，裸 join 会让同一条主回应重复出现 N 次
	// （N = 该孩子落在周期内的提交数）。按 r.id 去重，responses[] 才是「主回应」列表。
	if err := s.db.WithContext(ctx).Table("task_responses AS r").
		Joins("INNER JOIN task_check_ins AS c ON c.response_id = r.id AND c.deleted_at IS NULL").
		Where("r.deleted_at IS NULL AND c.task_id = ? AND c.student_id = ?", id, pc.CurrentStudent.ID).
		Group("r.id").
		Order("r.created_at ASC").Find(&responses).Error; err != nil {
		log.L(ctx).Error("parent get task responses failed", "task", id, "err", err)
		return nil, err
	}
	mainIDs := make([]uint64, 0, len(responses))
	for _, r := range responses {
		mainIDs = append(mainIDs, r.ID)
	}
	supplements, err := s.taskSupplementsForStudent(ctx, id, pc.CurrentStudent.ID)
	if err != nil {
		log.L(ctx).Error("parent get task supplements failed", "task", id, "err", err)
		return nil, err
	}
	// 反馈按家长账号一条一条存：先把主回应与补充回应用到的 id 合并，一次查出 my_reaction，
	// 避免每条回应单独查一次。
	allIDs := append([]uint64{}, mainIDs...)
	for _, items := range supplements {
		for _, sp := range items {
			allIDs = append(allIDs, sp.ResponseID)
		}
	}
	reactions := map[uint64]string{}
	if len(allIDs) > 0 {
		var rows []domain.TaskResponseReaction
		if err := s.db.WithContext(ctx).
			Where("response_id IN ? AND user_id = ?", allIDs, pc.UserID).
			Find(&rows).Error; err != nil {
			log.L(ctx).Error("parent get task reactions failed", "task", id, "err", err)
			return nil, err
		}
		for _, r := range rows {
			reactions[r.ResponseID] = r.Reaction
		}
	}
	respOut := make([]map[string]any, 0, len(responses))
	for _, r := range responses {
		spOut := make([]map[string]any, 0, len(supplements[r.ID]))
		for _, sp := range supplements[r.ID] {
			spOut = append(spOut, map[string]any{
				"id": sp.ResponseID, "checkin_id": sp.CheckInID, "ritual": sp.Ritual,
				"sticker_key": sp.StickerKey, "content": sp.Content, "media_url": sp.MediaURL,
				"created_at":  sp.CreatedAt.UTC().Format(time.RFC3339),
				"my_reaction": reactions[sp.ResponseID],
			})
		}
		respOut = append(respOut, map[string]any{
			"id": r.ID, "ritual": r.Ritual, "sticker_key": r.StickerKey, "content": r.Content,
			"media_url": r.MediaURL, "response_type": r.ResponseType,
			"created_at":  r.CreatedAt.UTC().Format(time.RFC3339),
			"my_reaction": reactions[r.ID],
			"supplements": spOut,
		})
	}
	var reqs []string
	_ = json.Unmarshal([]byte(t.Requirements), &reqs)
	if reqs == nil {
		reqs = []string{}
	}
	unread, err := s.unreadSupplementTaskIDs(ctx, pc, []uint64{t.ID})
	if err != nil {
		log.L(ctx).Error("parent get task unread failed", "task", id, "err", err)
		return nil, err
	}
	return map[string]any{
		"id": t.ID, "title": t.Title, "description": t.Description, "intro": t.Intro,
		"requirements": reqs, "learning_link": t.LearningLink, "frequency": t.Frequency,
		"target_student_id": t.TargetStudentID, "allow_public": t.AllowPublic,
		"recognition_ritual": t.RecognitionRitual, "star_scope": t.StarScope, "star_goal": t.StarGoal,
		"start_date": t.StartDate.Format("2006-01-02"), "end_date": t.EndDate.Format("2006-01-02"),
		"status": t.Status, "checked_in_today": submitted, "submitted_current_period": submitted,
		"has_unread_supplement": unread[t.ID], "responses": respOut, "created_at": t.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

// MarkTaskRead 清除当前孩子在该任务下的补充回应未读标记（upsert）。
//
// 用 ON DUPLICATE 而不是「先查再写」：并发打开详情（连点、多端）不应 500，
// 查询与写入之间的窗口也不能产生第二行（唯一索引会直接拒绝）。
func (s *ParentPortalService) MarkTaskRead(ctx context.Context, pc *ParentPortalContext, taskID uint64) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	var t domain.ParentTask
	if err := taskScopeQuery(s.db.WithContext(ctx), pc).Where("id = ?", taskID).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	// 截断到毫秒：列是 datetime(3)，MySQL 会把不足毫秒的部分四舍五入。
	// 不截断时，读发生在 .0007s 会存成下一秒的 .001，把同毫秒的补充回应误判成已读。
	now := time.Now().In(shanghaiLoc()).Truncate(time.Millisecond)
	// 与任务详情同口径：未开始/已结束的任务对家长不可见，也不该写读标记。
	if !taskIsActive(&t, now) {
		return nil, ErrParentNotFound
	}
	mark := domain.TaskReadMark{
		KindergartenID: pc.KindergartenID,
		TaskID:         taskID,
		UserID:         pc.UserID,
		StudentID:      pc.CurrentStudent.ID,
		LastReadAt:     &now,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "task_id"}, {Name: "user_id"}, {Name: "student_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"last_read_at": now,
			// 唯一索引把软删行也算在内：命中旧行时必须复活，否则读标记永远写不进去。
			"deleted_at": nil,
		}),
	}).Create(&mark).Error; err != nil {
		return nil, err
	}
	return map[string]any{"last_read_at": now.UTC().Format(time.RFC3339)}, nil
}

// unreadSupplementTaskIDs 返回当前孩子有未读补充回应的任务 id 集合。
//
// 未读 = 该任务下存在补充回应，且该家长在该孩子上没有读标记，或补充回应
// 在 last_read_at 之后（含同一秒）。用 >= 是为了「补充与打开详情同秒」不漏判（D9）。
func (s *ParentPortalService) unreadSupplementTaskIDs(
	ctx context.Context,
	pc *ParentPortalContext,
	taskIDs []uint64,
) (map[uint64]bool, error) {
	out := make(map[uint64]bool, len(taskIDs))
	if len(taskIDs) == 0 || pc.CurrentStudent == nil {
		return out, nil
	}
	var ids []uint64
	// 手写 Table 不走软删：补充关系、提交、回应、读标记四处都要自己补 deleted_at。
	err := s.db.WithContext(ctx).
		Table("task_response_supplements AS s").
		Joins("INNER JOIN task_check_ins AS c ON c.id = s.check_in_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN task_responses AS r ON r.id = s.response_id AND r.deleted_at IS NULL").
		Joins("LEFT JOIN task_read_marks AS m ON m.task_id = c.task_id AND m.user_id = ? AND m.student_id = c.student_id AND m.deleted_at IS NULL", pc.UserID).
		Where("s.deleted_at IS NULL AND c.task_id IN ? AND c.student_id = ?", taskIDs, pc.CurrentStudent.ID).
		Where("(m.last_read_at IS NULL OR r.created_at >= m.last_read_at)").
		Group("c.task_id").
		Pluck("c.task_id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// taskSupplementRow 某条主回应下的补充回应（含补充回应自身的展示字段）。
// MainID 是这条补充所附着的**主**回应 id：补充关系里存的 response_id 是补充自己的，
// 主回应要经 checkin.response_id 反查。
type taskSupplementRow struct {
	ResponseID uint64    `gorm:"column:response_id"`
	CheckInID  uint64    `gorm:"column:check_in_id"`
	MainID     uint64    `gorm:"column:main_id"`
	Ritual     string    `gorm:"column:ritual"`
	StickerKey string    `gorm:"column:sticker_key"`
	Content    string    `gorm:"column:content"`
	MediaURL   string    `gorm:"column:media_url"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

// taskSupplementsForStudent 取当前孩子在该任务下收到的补充回应，按主回应 id 分组。
// 只按当前孩子的 checkin 反查：一条 period 回应可能给多个孩子各补一条，
// 家长不能看到别人孩子的那条。
func (s *ParentPortalService) taskSupplementsForStudent(ctx context.Context, taskID, studentID uint64) (map[uint64][]taskSupplementRow, error) {
	var rows []taskSupplementRow
	if err := s.db.WithContext(ctx).Table("task_response_supplements AS s").
		Select("s.response_id, s.check_in_id, c.response_id AS main_id, r.ritual, r.sticker_key, r.content, r.media_url, r.created_at").
		Joins("INNER JOIN task_check_ins AS c ON c.id = s.check_in_id AND c.deleted_at IS NULL").
		Joins("INNER JOIN task_responses AS r ON r.id = s.response_id AND r.deleted_at IS NULL").
		Where("c.task_id = ? AND c.student_id = ? AND c.response_id <> 0", taskID, studentID).
		Where("s.deleted_at IS NULL").
		Order("s.created_at ASC, s.id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint64][]taskSupplementRow, len(rows))
	for _, r := range rows {
		out[r.MainID] = append(out[r.MainID], r)
	}
	return out, nil
}

func (s *ParentPortalService) CheckinTask(ctx context.Context, pc *ParentPortalContext, taskID uint64, content string, mediaFileID uint64) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	now := time.Now().In(shanghaiLoc())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, shanghaiLoc())
	var out map[string]any
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var t domain.ParentTask
		// 先锁任务行：家长提交会写 task_check_in_states 与 task_check_ins，
		// 与 DeleteTask 的级联（先锁回应、再删 checkin/state）反序时会死锁。
		if err := taskScopeQuery(tx, pc).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = 1", taskID).First(&t).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrParentNotFound
			}
			return err
		}
		if !taskIsActive(&t, now) {
			return ErrParentInvalidParams
		}
		state := domain.TaskCheckInState{TaskID: taskID, StudentID: pc.CurrentStudent.ID}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&state).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("task_id = ? AND student_id = ?", taskID, pc.CurrentStudent.ID).First(&state).Error; err != nil {
			return err
		}
		if state.LastCheckInDate != nil {
			last := time.Date(state.LastCheckInDate.Year(), state.LastCheckInDate.Month(), state.LastCheckInDate.Day(), 0, 0, 0, 0, shanghaiLoc())
			switch t.Frequency {
			case domain.TaskFrequencyDaily:
				if !last.Before(today) {
					return ErrParentInvalidParams
				}
			case domain.TaskFrequencyWeekly:
				if !last.Before(today.AddDate(0, 0, -6)) {
					return ErrParentInvalidParams
				}
			case domain.TaskFrequencyOnce:
				return ErrParentInvalidParams
			}
		}
		var mediaURL string
		if mediaFileID > 0 {
			var f domain.File
			if err := tx.Where("id = ? AND user_id = ? AND biz_type = ? AND deleted_at IS NULL", mediaFileID, pc.UserID, "task").First(&f).Error; err != nil {
				return ErrParentInvalidParams
			}
			if f.FileType != "image" && f.FileType != "audio" {
				return ErrParentInvalidParams
			}
			mediaURL = f.FileURL
		}
		checkin := &domain.TaskCheckIn{
			TaskID: taskID, StudentID: pc.CurrentStudent.ID, UserID: pc.UserID,
			Date: today, Content: content, MediaFileID: mediaFileID, MediaURL: mediaURL,
		}
		if err := tx.Create(checkin).Error; err != nil {
			return ErrParentInvalidParams
		}
		if err := tx.Model(&state).Update("last_check_in_date", today).Error; err != nil {
			return err
		}

		out = map[string]any{
			"id": checkin.ID, "message": "已提交，等待老师回应",
			"created_at": checkin.CreatedAt.UTC().Format(time.RFC3339),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ParentPortalService) CheckinHistory(ctx context.Context, pc *ParentPortalContext, taskID uint64, page, pageSize int) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	// 与列表/详情同口径：个体任务只对目标孩子可见，跨家庭直接访问应被拒绝，
	// 不能返回空列表（否则「直接访问也被拒绝」的规格无法成立）。
	var t domain.ParentTask
	if err := taskScopeQuery(s.db.WithContext(ctx), pc).Where("id = ?", taskID).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	// 契约写了 page/page_size/total：先按总数分页，total 必须是全量条数而不是当前页长度，
	// 否则前端没法判断还有没有下一页。
	var total int64
	if err := s.db.WithContext(ctx).Model(&domain.TaskCheckIn{}).
		Where("task_id = ? AND student_id = ?", taskID, pc.CurrentStudent.ID).Count(&total).Error; err != nil {
		log.L(ctx).Error("parent checkin history count failed", "task", taskID, "err", err)
		return nil, err
	}
	var rows []domain.TaskCheckIn
	if err := s.db.WithContext(ctx).Preload("Student").
		Where("task_id = ? AND student_id = ?", taskID, pc.CurrentStudent.ID).
		Order("date DESC").Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&rows).Error; err != nil {
		log.L(ctx).Error("parent checkin history list failed", "task", taskID, "err", err)
		return nil, err
	}
	list := make([]map[string]any, len(rows))
	now := time.Now().In(shanghaiLoc())
	for i, r := range rows {
		studentName := ""
		if r.Student != nil {
			studentName = r.Student.Name
		}
		var approvedAt, revokedAt any
		if r.PublicApprovedAt != nil {
			approvedAt = r.PublicApprovedAt.UTC().Format(time.RFC3339)
		}
		if r.PublicRevokedAt != nil {
			revokedAt = r.PublicRevokedAt.UTC().Format(time.RFC3339)
		}
		list[i] = map[string]any{
			"id": r.ID, "date": r.Date.Format("2006-01-02"), "content": r.Content,
			"media_url": r.MediaURL, "student_name": studentName, "response_id": r.ResponseID,
			"is_public_requested":    r.IsPublicRequested,
			"public_scope_confirmed": r.PublicScopeConfirmed,
			"public_approved_at":     approvedAt,
			"public_revoked_at":      revokedAt,
			// 可见性由服务端按公开墙同一套条件算出，前端不要自己拼。
			"public_visible": publicCheckinVisible(&t, &rows[i], pc.CurrentStudent.ClassID, now),
			"created_at":     r.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	return map[string]any{"list": list, "total": total}, nil
}

// ==================== 公开申请 / 取消 / 班级公开墙 ====================

// lockCheckinForPublic 家长公开写路径的公共前置：定位 → 锁任务行 → 锁提交行 → 归属校验。
//
// 锁序固定为 parent_tasks → task_check_ins（handoff §12.2）：反过来会与
// deleteTaskCascade 的「锁回应 → 删 checkin」形成反序死锁。
//
// 这里刻意**不套** taskScopeQuery：
//   - 归属由「提交属于当前孩子」判定，已经足够（跨家庭 403）；
//   - 套上任务可见范围后，孩子转班会让原班级任务不再匹配，家长连自己的
//     内容都撤不下来，而内容仍留在原班级墙上——同意无法撤回。
//
// 「任务是否还能公开」只在申请时校验（见 RequestPublicCheckin）；取消/撤回永远可用。
func (s *ParentPortalService) lockCheckinForPublic(ctx context.Context, tx *gorm.DB, pc *ParentPortalContext, checkinID uint64) (*domain.ParentTask, *domain.TaskCheckIn, *checkinLocate, error) {
	loc, err := locateCheckin(ctx, tx, checkinID)
	if err != nil {
		if errors.Is(err, errTaskNotFound) {
			return nil, nil, nil, ErrParentNotFound
		}
		return nil, nil, nil, err
	}
	if loc.StudentID != pc.CurrentStudent.ID {
		return nil, nil, nil, ErrParentForbidden
	}
	var t domain.ParentTask
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND kindergarten_id = ?", loc.TaskID, pc.KindergartenID).First(&t).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrParentNotFound
		}
		return nil, nil, nil, err
	}
	var c domain.TaskCheckIn
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND task_id = ? AND student_id = ?", checkinID, t.ID, pc.CurrentStudent.ID).
		First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil, ErrParentNotFound
		}
		return nil, nil, nil, err
	}
	return &t, &c, loc, nil
}

// RequestPublicCheckin 家长申请把这条提交公开到班级墙。
// 申请不等于公开：清空旧批准与旧确认，等老师重新审核。
func (s *ParentPortalService) RequestPublicCheckin(ctx context.Context, pc *ParentPortalContext, checkinID uint64) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	now := time.Now().In(shanghaiLoc())
	var out map[string]any
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		t, c, loc, err := s.lockCheckinForPublic(ctx, tx, pc, checkinID)
		if err != nil {
			return err
		}
		// 申请必须落在「这个孩子当前可见的任务」上：转班后原班级任务不再属于他，
		// 提交回来只会让内容出现在一个他已经不属于的班级墙上。
		//
		// 契约把「任务仍属于孩子当前班级」列为允许条件（不满足 → 422）；
		// 任务指向别的孩子属于「不可见」，仍按 404，避免泄漏任务存在性。
		// 园所相等由 lockCheckinForPublic 的锁条件保证，这里只做防御。
		if t.KindergartenID != pc.KindergartenID ||
			(t.TargetStudentID > 0 && t.TargetStudentID != pc.CurrentStudent.ID) {
			return ErrParentNotFound
		}
		if t.ClassID != pc.CurrentStudent.ClassID {
			return ErrTaskPublicNotAllowed
		}
		// 个体任务与未开启公开的任务根本不该有公开入口；已结束/未开始的任务
		// 放上去也永远不会出现在墙上——直接拒绝比静默排进审核队列更诚实。
		if !t.AllowPublic || t.TargetStudentID > 0 {
			return ErrTaskPublicNotAllowed
		}
		if !taskIsActive(t, now) {
			return ErrTaskPublicNotAllowed
		}
		// 一键完成（无文字、无媒体）没有可公开的内容：直接拒绝，不要让它进审核队列。
		// 已批准的重复申请是幂等空操作，维持原行为（历史数据也能照常撤回）。
		if publicCheckinStatus(c) != TaskPublicApproved && !checkinHasPublicContent(c) {
			return ErrTaskPublicEmptyContent
		}
		// 已批准且未撤回时重复点击是幂等空操作：不能把已经公开的内容打回待审核，
		// 否则家长手滑点两下，墙上的记录就消失了。
		if publicCheckinStatus(c) != TaskPublicApproved {
			if err := tx.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
				Updates(publicRequestUpdates()).Error; err != nil {
				return err
			}
			var fresh domain.TaskCheckIn
			if err := tx.Where("id = ?", c.ID).First(&fresh).Error; err != nil {
				return err
			}
			c = &fresh
		}
		out = publicCheckinView(t, c, loc.StudentClassID, now)
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

// CancelPublicCheckin 家长取消公开申请（也用于撤下已公开的内容）。
func (s *ParentPortalService) CancelPublicCheckin(ctx context.Context, pc *ParentPortalContext, checkinID uint64) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	now := time.Now().In(shanghaiLoc())
	var out map[string]any
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		t, c, loc, err := s.lockCheckinForPublic(ctx, tx, pc, checkinID)
		if err != nil {
			return err
		}
		// 从未申请 / 已取消：幂等空操作，不刷新 public_revoked_at
		// （否则每点一次取消都会产生一条「新撤回」记录）。
		switch publicCheckinStatus(c) {
		case TaskPublicPending, TaskPublicApproved:
			// 审计留痕：公开状态会把审核人清空，日志保留「谁批准过、谁撤下」。
			priorStatus := publicCheckinStatus(c)
			priorApprover := c.PublicApprovedBy
			if err := tx.Model(&domain.TaskCheckIn{}).Where("id = ?", c.ID).
				Updates(publicRevokeUpdates(now)).Error; err != nil {
				return err
			}
			var fresh domain.TaskCheckIn
			if err := tx.Where("id = ?", c.ID).First(&fresh).Error; err != nil {
				return err
			}
			c = &fresh
			detail := fmt.Sprintf("task_id=%d prior_status=%s", t.ID, priorStatus)
			if priorApprover > 0 {
				detail += fmt.Sprintf(" approved_by=%d", priorApprover)
			}
			if err := tx.Create(publicAuditEntry(
				pc.KindergartenID, pc.UserID, PublicAuditCancel, "task_check_in", c.ID, detail,
			)).Error; err != nil {
				return err
			}
		}
		out = publicCheckinView(t, c, loc.StudentClassID, now)
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

// publicWallRow 公开墙一行。
type publicWallRow struct {
	CheckinID     uint64    `gorm:"column:checkin_id"`
	TaskID        uint64    `gorm:"column:task_id"`
	TaskTitle     string    `gorm:"column:task_title"`
	StudentID     uint64    `gorm:"column:student_id"`
	StudentName   string    `gorm:"column:student_name"`
	StudentAvatar string    `gorm:"column:student_avatar"`
	Date          time.Time `gorm:"column:date"`
	Content       string    `gorm:"column:content"`
	MediaURL      string    `gorm:"column:media_url"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

// publicWallQuery 构建公开墙查询。每次调用都新建一个 *gorm.DB：
// 同一个对象先 Count 再 Select 会被 GORM 的 Statement 复用污染（SELECT 子句残留），
// 所以这里做成工厂而不是在调用方复用它。
//
// `st.class_id = t.class_id` 是第九条之外的**安全补强**（D7「公开范围：本班家长可见」）：
//   - 孩子转班后，原班级墙不得继续展示他的内容；
//   - 管理员把任务改派到别的班（7.1 的结构字段锁定落地前可达）时，
//     原班孩子的照片/文字不得出现在新班家长面前。
// 左连接下学生缺失或已软删会得到 NULL，条件自然不成立。
func (s *ParentPortalService) publicWallQuery(ctx context.Context, pc *ParentPortalContext, today string) *gorm.DB {
	return s.db.WithContext(ctx).Table("task_check_ins AS c").
		Joins("INNER JOIN parent_tasks AS t ON t.id = c.task_id AND t.deleted_at IS NULL").
		Joins("LEFT JOIN students AS st ON st.id = c.student_id AND st.deleted_at IS NULL").
		Where("c.deleted_at IS NULL").
		Where("st.class_id = t.class_id").
		Where("t.kindergarten_id = ? AND t.class_id = ?", pc.KindergartenID, pc.CurrentStudent.ClassID).
		Where("t.allow_public = ? AND t.target_student_id = 0 AND t.status = 1", true).
		Where("t.end_date >= ?", today).
		Where("c.is_public_requested = ? AND c.public_scope_confirmed = ?", true, true).
		Where("c.public_approved_at IS NOT NULL AND c.public_revoked_at IS NULL")
}

// PublicWall 本班公开墙。可见条件逐字对应 design D7 / tasks 4.3。
//
// 用 Table(...) 手写查询时 GORM **不会**自动补软删除条件，c/t/st 三张表的
// deleted_at IS NULL 都必须自己写（handoff §12.2 的同类坑）。
func (s *ParentPortalService) PublicWall(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	now := time.Now().In(shanghaiLoc())
	today := now.Format("2006-01-02")
	var total int64
	if err := s.publicWallQuery(ctx, pc, today).Count(&total).Error; err != nil {
		log.L(ctx).Error("parent public wall count failed", "err", err)
		return nil, err
	}
	var rows []publicWallRow
	if err := s.publicWallQuery(ctx, pc, today).
		Select("c.id AS checkin_id, c.task_id, t.title AS task_title, c.student_id, " +
			"st.name AS student_name, st.avatar AS student_avatar, c.date, c.content, c.media_url, c.created_at").
		Order("c.created_at DESC, c.id DESC").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&rows).Error; err != nil {
		log.L(ctx).Error("parent public wall list failed", "err", err)
		return nil, err
	}
	list := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		list = append(list, map[string]any{
			"checkin_id": r.CheckinID, "task_id": r.TaskID, "task_title": r.TaskTitle,
			"student": map[string]any{"id": r.StudentID, "name": r.StudentName, "avatar": r.StudentAvatar},
			"date":    r.Date.Format("2006-01-02"), "content": r.Content, "media_url": r.MediaURL,
			"created_at": r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

// ReactToResponse 家长对老师回应做一键反馈（child_happy / received / thanks）。
//
// 反馈按 (response_id, user_id) 唯一，所以同一条汇总回应可以被多个家庭分别反馈、
// 同一家长可以改自己的那一行，互不覆盖。
//
// 权限：该回应下必须挂着**这个家长自己孩子**的提交——提交人本人算，
// 与该孩子有绑定关系的另一位家长也算（规格：家长只能对「与自己孩子提交相关」的回应反馈）。
// 不按 CurrentStudent 收窄，是因为一条 period 回应可能同时覆盖同一家长两个孩子的提交，
// 限定到某个孩子会逼家长对同一条回应反馈两次。
func (s *ParentPortalService) ReactToResponse(ctx context.Context, pc *ParentPortalContext, responseID uint64, reaction string) (map[string]any, error) {
	if !validTaskReaction(reaction) {
		return nil, ErrParentInvalidParams
	}
	// 权限校验与写入必须在同一个事务里、并锁住任务行：否则并发的 DeleteTask 可以在
	// 校验通过之后把回应软删、再让本请求写下一行指向已删回应的反馈。
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var resp domain.TaskResponse
		if err := tx.Where("id = ? AND kindergarten_id = ?", responseID, pc.KindergartenID).First(&resp).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrParentNotFound
			}
			return err
		}
		// 锁任务行：与 DeleteTask 的级联删除串行。
		var task domain.ParentTask
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND deleted_at IS NULL", resp.TaskID).First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrParentNotFound
			}
			return err
		}
		// 拿到任务锁后**当前读**确认回应仍然活着（上面那次 First 可能是快照读）。
		var live domain.TaskResponse
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).
			Where("id = ? AND deleted_at IS NULL", responseID).First(&live).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrParentNotFound
			}
			return err
		}
		mine, err := s.reactionAllowed(ctx, tx, responseID, pc.UserID)
		if err != nil {
			return err
		}
		if !mine {
			return ErrParentForbidden
		}
		// upsert：并发/重复点击不应 500，也不该产生第二行。
		//
		// 唯一键是 (response_id, user_id)，**不含 deleted_at**：一旦将来加了「取消反馈」把行软删除，
		// 这条 upsert 会命中那行已软删的记录、只改 reaction 而 deleted_at 仍非空，家长端就再也
		// 看不到自己的反馈（与 growth_likes 的历史回归同型）。所以这里顺手把 deleted_at 清空。
		row := domain.TaskResponseReaction{
			KindergartenID: pc.KindergartenID, ResponseID: responseID, UserID: pc.UserID, Reaction: reaction,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "response_id"}, {Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"reaction": reaction, "deleted_at": nil, "updated_at": time.Now(),
			}),
		}).Create(&row).Error; err != nil {
			log.L(ctx).Error("parent reaction upsert failed", "response", responseID, "err", err)
			return err
		}
		return nil
	})
	if isTaskLockConflict(err) {
		return nil, ErrTaskBusy
	}
	if err != nil {
		return nil, err
	}
	return map[string]any{"response_id": responseID, "reaction": reaction}, nil
}

// reactionAllowed 判断该家长能否对这条回应反馈：
// 主回应看提交的 response_id，补充回应看 task_response_supplements 关系；
// 两条路径都接受「提交人本人」或「与该孩子有绑定关系的家长」。
func (s *ParentPortalService) reactionAllowed(ctx context.Context, tx *gorm.DB, responseID, userID uint64) (bool, error) {
	var mine int64
	if err := tx.Model(&domain.TaskCheckIn{}).
		Where("response_id = ? AND user_id = ?", responseID, userID).Count(&mine).Error; err != nil {
		log.L(ctx).Error("parent reaction permission check failed", "scope", "main", "response", responseID, "err", err)
		return false, err
	}
	binding := "EXISTS (SELECT 1 FROM student_parents sp WHERE sp.student_id = c.student_id AND sp.user_id = ? AND sp.deleted_at IS NULL)"
	if mine == 0 {
		if err := tx.Table("task_check_ins AS c").
			Where("c.response_id = ? AND c.deleted_at IS NULL", responseID).
			Where("c.user_id = ? OR "+binding, userID, userID).
			Count(&mine).Error; err != nil {
			log.L(ctx).Error("parent reaction permission check failed", "scope", "main_co_parent", "response", responseID, "err", err)
			return false, err
		}
	}
	if mine == 0 {
		if err := tx.Table("task_response_supplements AS s").
			Joins("INNER JOIN task_check_ins AS c ON c.id = s.check_in_id AND c.deleted_at IS NULL").
			Where("s.response_id = ? AND s.deleted_at IS NULL", responseID).
			Where("c.user_id = ? OR "+binding, userID, userID).
			Count(&mine).Error; err != nil {
			log.L(ctx).Error("parent reaction permission check failed", "scope", "supplement", "response", responseID, "err", err)
			return false, err
		}
	}
	return mine > 0, nil
}

// ==================== Points & Achievements ====================

func (s *ParentPortalService) PointsOverview(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	var account domain.PointAccount
	s.db.WithContext(ctx).Where("student_id = ?", pc.CurrentStudent.ID).First(&account)
	totalPoints := account.TotalPoints - account.UsedPoints
	level := 1
	levelName := "新手上路"
	if totalPoints >= 500 {
		level = 3
		levelName = "积分达人"
	} else if totalPoints >= 200 {
		level = 2
		levelName = "活跃家长"
	}
	thisWeek := time.Now().AddDate(0, 0, -int(time.Now().Weekday())+1)
	var weekPoints int64
	s.db.WithContext(ctx).Model(&domain.PointTransaction{}).
		Where("student_id = ? AND created_at >= ? AND points > 0", pc.CurrentStudent.ID, thisWeek).Select("COALESCE(SUM(points), 0)").Scan(&weekPoints)
	return map[string]any{
		"total_points": totalPoints, "level": level, "level_name": levelName,
		"this_week": weekPoints, "this_month": int64(0),
	}, nil
}

func (s *ParentPortalService) PointsHistory(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	offset := (page - 1) * pageSize
	var total int64
	s.db.WithContext(ctx).Model(&domain.PointTransaction{}).Where("student_id = ?", pc.CurrentStudent.ID).Count(&total)
	var rows []domain.PointTransaction
	s.db.WithContext(ctx).Where("student_id = ?", pc.CurrentStudent.ID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows)
	return map[string]any{"list": rows, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) Leaderboard(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.Class == nil {
		return map[string]any{"list": []any{}, "my_rank": 0, "my_points": 0}, nil
	}
	var rows []struct {
		StudentID   uint64 `gorm:"column:student_id"`
		TotalPoints int64  `gorm:"column:total_points"`
		StudentName string `gorm:"column:student_name"`
	}
	s.db.WithContext(ctx).Raw(`
		SELECT pa.student_id, COALESCE(pa.total_points - pa.used_points, 0) AS total_points, s.name AS student_name
		FROM point_accounts pa INNER JOIN students s ON s.id = pa.student_id
		WHERE s.class_id = ? AND s.status = 1
		ORDER BY total_points DESC LIMIT 20`, pc.Class.ID).Scan(&rows)
	list := make([]map[string]any, len(rows))
	myRank, myPoints := 0, int64(0)
	for i, r := range rows {
		list[i] = map[string]any{"rank": i + 1, "student_id": r.StudentID, "student_name": r.StudentName, "total_points": r.TotalPoints}
		if r.StudentID == pc.CurrentStudent.ID {
			myRank = i + 1
			myPoints = r.TotalPoints
		}
	}
	return map[string]any{"period": "weekly", "list": list, "my_rank": myRank, "my_points": myPoints}, nil
}

func (s *ParentPortalService) ListAchievements(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	// 取得授予记录（含见证 log_id），而不是只取 achievement_id 集合
	var grants []domain.StudentAchievement
	s.db.WithContext(ctx).Where("student_id = ?", pc.CurrentStudent.ID).Find(&grants)
	earnedByAch := make(map[uint64]*domain.StudentAchievement, len(grants))
	for i := range grants {
		earnedByAch[grants[i].AchievementID] = &grants[i]
	}

	var all []domain.Achievement
	// 必须按园所过滤：不加的话会把其他园所配置的成就一并返回，
	// total_count / not_earned 都会串租户（CLAUDE.md：业务表全带 kindergarten_id）。
	s.db.WithContext(ctx).Where("kindergarten_id = ?", pc.KindergartenID).Find(&all)
	earned := make([]map[string]any, 0)
	notEarned := make([]map[string]any, 0)
	for _, a := range all {
		// 字段名与契约 / admin 侧对齐：icon_url（不是 icon）、description
		// （家长端 points/index.vue 渲染 a.description，之前实现没返回 → 永远空白）
		m := map[string]any{
			"id":          a.ID,
			"name":        a.Name,
			"description": a.Description,
			"icon_url":    a.Icon,
			"category":    a.Category,
			"points":      a.Points,
		}
		if g, ok := earnedByAch[a.ID]; ok {
			// log_id 可空：未绑定见证的授予，徽章不出现在任何成长记录卡片上
			if g.LogID != nil && *g.LogID != 0 {
				m["log_id"] = *g.LogID
			}
			m["earned_at"] = g.AchievedAt.UTC().Format(time.RFC3339)
			earned = append(earned, m)
		} else {
			notEarned = append(notEarned, m)
		}
	}
	return map[string]any{"earned": earned, "not_earned": notEarned, "earned_count": len(earned), "total_count": len(all)}, nil
}

// ==================== Feedback ====================

type ParentFeedbackInput struct {
	Category string `json:"category"`
	Content  string `json:"content"`
	Contact  string `json:"contact"`
	Images   string `json:"images"`
}

func (s *ParentPortalService) ListFeedback(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	offset := (page - 1) * pageSize
	var total int64
	s.db.WithContext(ctx).Model(&domain.Feedback{}).Where("kindergarten_id = ? AND user_id = ?", pc.KindergartenID, pc.UserID).Count(&total)
	var rows []domain.Feedback
	if err := s.db.WithContext(ctx).Where("kindergarten_id = ? AND user_id = ?", pc.KindergartenID, pc.UserID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, len(rows))
	for i, r := range rows {
		statusLabel := "已提交"
		switch r.Status {
		case 1:
			statusLabel = "已回复"
		case 2:
			statusLabel = "已关闭"
		}
		list[i] = map[string]any{
			"id": r.ID, "category": r.Category, "content": r.Content,
			"status": r.Status, "status_label": statusLabel,
			"reply": r.Reply, "created_at": r.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

func (s *ParentPortalService) CreateFeedback(ctx context.Context, pc *ParentPortalContext, req ParentFeedbackInput) (map[string]any, error) {
	if strings.TrimSpace(req.Content) == "" {
		return nil, ErrParentInvalidParams
	}
	fb := &domain.Feedback{
		KindergartenID: pc.KindergartenID, UserID: pc.UserID,
		Category: req.Category, Content: strings.TrimSpace(req.Content),
		Contact: req.Contact, Images: req.Images, Status: 0,
	}
	if err := s.db.WithContext(ctx).Create(fb).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": fb.ID, "status": 0, "status_label": "已提交", "created_at": fb.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

// ==================== Student Bind/Unbind ====================

func (s *ParentPortalService) BindStudent(ctx context.Context, pc *ParentPortalContext, inviteCode string) (map[string]any, error) {
	if strings.TrimSpace(inviteCode) == "" {
		return nil, ErrParentInvalidParams
	}
	// Find student by invite code (stored as student_no or dedicated field)
	var student domain.Student
	if err := s.db.WithContext(ctx).Where("student_no = ? AND kindergarten_id = ?", inviteCode, pc.KindergartenID).First(&student).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParentNotFound
		}
		return nil, err
	}
	// Check not already bound
	var existing int64
	s.db.WithContext(ctx).Model(&domain.StudentParent{}).Where("student_id = ? AND user_id = ?", student.ID, pc.UserID).Count(&existing)
	if existing > 0 {
		return nil, ErrParentInvalidParams
	}
	sp := &domain.StudentParent{StudentID: student.ID, UserID: pc.UserID, Relation: "other", IsPrimary: false}
	if err := s.db.WithContext(ctx).Create(sp).Error; err != nil {
		return nil, err
	}
	return map[string]any{"id": sp.ID, "student_id": student.ID, "student_name": student.Name, "created_at": sp.CreatedAt.UTC().Format(time.RFC3339)}, nil
}

func (s *ParentPortalService) UnbindStudent(ctx context.Context, pc *ParentPortalContext, id uint64) error {
	res := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, pc.UserID).Delete(&domain.StudentParent{})
	if res.RowsAffected == 0 {
		return ErrParentNotFound
	}
	return nil
}

// ==================== Settings ====================

func (s *ParentPortalService) GetSettings(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	return map[string]any{
		"notification": map[string]any{"notice_push": true, "leave_push": true, "pickup_push": true, "growth_push": true, "task_push": true, "announcement_push": true},
		"privacy":      map[string]any{"show_phone_to_teacher": true, "allow_growth_share": true},
	}, nil
}

func (s *ParentPortalService) UpdateSettings(ctx context.Context, pc *ParentPortalContext, settings map[string]any) error {
	// Settings are stored in-memory/session for now; persistent storage can be added later
	return nil
}

// ==================== Home Overview ====================

// pendingTaskCount 只统计当前孩子当前周期内尚未提交的有效任务（tasks 2.7）。
// 已结束/未开始的任务由 taskIsActive 排除；周期口径复用 taskSubmittedInPeriod，
// 不在这里再写一份 daily/weekly/once 判断。
func pendingTaskCount(ctx context.Context, db *gorm.DB, pc *ParentPortalContext, now time.Time) (int64, error) {
	if pc.CurrentStudent == nil {
		return 0, nil
	}
	var tasks []domain.ParentTask
	if err := taskScopeQuery(db.WithContext(ctx).Model(&domain.ParentTask{}), pc).
		Where("status = 1").Find(&tasks).Error; err != nil {
		return 0, err
	}
	var pending int64
	for _, t := range tasks {
		if !taskIsActive(&t, now) {
			continue
		}
		submitted, err := taskSubmittedInPeriod(ctx, db, &t, pc.CurrentStudent.ID, now)
		if err != nil {
			return 0, err
		}
		if !submitted {
			pending++
		}
	}
	return pending, nil
}

func (s *ParentPortalService) HomeOverview(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	now := time.Now()
	today := now.Format("2006-01-02")
	cn := ""
	if pc.Class != nil {
		cn = pc.Class.Name
	}

	studentsList := make([]map[string]any, len(pc.Students))
	for i, st := range pc.Students {
		cname := ""
		if st.Class != nil {
			cname = st.Class.Name
		}
		studentsList[i] = map[string]any{"id": st.ID, "name": st.Name, "avatar": st.Avatar, "class_name": cname, "class_id": st.ClassID}
	}

	// Today summary
	var attendance domain.Attendance
	err := s.db.WithContext(ctx).Where("student_id = ? AND date = ?", pc.CurrentStudent.ID, today).First(&attendance).Error
	attendanceStatus := "not_arrived"
	attendanceLabel := "未入园"
	arriveTime := ""
	if err == nil {
		if attendance.Status == domain.AttendancePresent {
			attendanceStatus = "present"
			attendanceLabel = "已入园"
		}
		if !attendance.CheckInTime.IsZero() {
			arriveTime = attendance.CheckInTime.Format("15:04")
		}
	}

	var summary domain.StudentDailySummary
	s.db.WithContext(ctx).Where("student_id = ? AND date = ?", pc.CurrentStudent.ID, today).First(&summary)

	var newGrowthCount int64
	{
		visWhere, visArgs := growthLogVisibilityWhere(pc.KindergartenID, pc.Students, pc.CurrentStudent)
		s.db.WithContext(ctx).Model(&domain.GrowthDailyLog{}).
			Where(visWhere, visArgs...).
			Where("growth_daily_logs.date = ?", today).Count(&newGrowthCount)
	}
	var newNoticeCount int64
	if pc.Class != nil {
		homeClassIDs := make([]uint64, 0, len(pc.Students))
		for _, st := range pc.Students {
			homeClassIDs = append(homeClassIDs, st.ClassID)
		}
		s.notice.FilterByTargetHit(s.db.WithContext(ctx).Model(&domain.Notice{}).
			Where("kindergarten_id = ? AND status = ? AND published_at >= ?", pc.KindergartenID, domain.NoticeStatusPublished, today),
			homeClassIDs).Count(&newNoticeCount)
	}
	pendingTasks, err := pendingTaskCount(ctx, s.db, pc, now)
	if err != nil {
		// 不把整个首页打成 500：待办角标降级为 0，但必须留下日志，
		// 否则「今天没有待办」和「查询失败」在页面上长得一模一样。
		log.L(ctx).Error("home pending tasks failed", "student", pc.CurrentStudent.ID, "err", err)
		pendingTasks = 0
	}

	var leave domain.LeaveRequest
	leaveStatus := map[string]any(nil)
	if err := s.db.WithContext(ctx).Where("student_id = ? AND start_date <= ? AND end_date >= ? AND status = ?", pc.CurrentStudent.ID, now, now, domain.LeaveStatusApproved).First(&leave).Error; err == nil {
		leaveStatus = map[string]any{"id": leave.ID, "reason": leave.Reason, "status": leave.Status, "status_label": "已同意"}
	}

	return map[string]any{
		"date":            now.Format("2006-01-02"),
		"weekday":         []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}[now.Weekday()],
		"current_student": map[string]any{"id": pc.CurrentStudent.ID, "name": pc.CurrentStudent.Name, "avatar": pc.CurrentStudent.Avatar, "class_name": cn, "class_id": pc.CurrentStudent.ClassID},
		"students":        studentsList,
		"today_summary": map[string]any{
			"attendance_status": attendanceStatus, "attendance_label": attendanceLabel, "arrive_time": arriveTime,
			"meal_rating": summary.MealRating, "meal_label": mealRatingLabel(int(summary.MealRating)),
			"nap_duration": summary.NapDuration, "sleep_quality": summary.SleepQuality,
			"sleep_label": sleepQualityLabel(int(summary.SleepQuality)),
			"mood_rating": summary.MoodRating, "mood_label": moodRatingLabel(int(summary.MoodRating)),
			"performance_tag":  summary.PerformanceTag,
			"new_growth_count": newGrowthCount, "new_notice_count": newNoticeCount,
			"pending_tasks": pendingTasks, "leave_status": leaveStatus,
		},
	}, nil
}

func mealRatingLabel(r int) string {
	switch r {
	case 3:
		return "很好"
	case 2:
		return "一般"
	case 1:
		return "较差"
	default:
		return ""
	}
}
func sleepQualityLabel(q int) string {
	switch q {
	case 3:
		return "很好"
	case 2:
		return "一般"
	case 1:
		return "较差"
	default:
		return ""
	}
}
func moodRatingLabel(m int) string {
	switch m {
	case 3:
		return "愉快"
	case 2:
		return "平稳"
	case 1:
		return "低落"
	default:
		return ""
	}
}

// ==================== Morning Health ====================

type ParentHealthRecord struct {
	CheckDate          string         `json:"check_date"`
	OverallStatus      int8           `json:"overall_status"`
	OverallStatusLabel string         `json:"overall_status_label"`
	Temperature        *float64       `json:"temperature"`
	AbnormalItems      []AbnormalItem `json:"abnormal_items"`
	Remark             string         `json:"remark"`
	IsIsolated         bool           `json:"is_isolated"`
}

// ListHealthRecords 返回某孩子最近 7 天已完成的晨检记录。
// 只看本人孩子；未绑定直接拒绝，不回退到第一个孩子。
func (s *ParentPortalService) ListHealthRecords(ctx context.Context, pc *ParentPortalContext, studentID uint64) ([]ParentHealthRecord, error) {
	if studentID == 0 {
		return nil, ErrParentInvalidParams
	}
	bound := false
	for _, st := range pc.Students {
		if st.ID == studentID {
			bound = true
			break
		}
	}
	if !bound {
		return nil, ErrParentForbidden
	}

	from := time.Now().AddDate(0, 0, -6).Format("2006-01-02")
	var rows []domain.MorningHealthCheck
	if err := s.db.WithContext(ctx).
		Where("kindergarten_id = ? AND student_id = ? AND check_status = ? AND date >= ? AND deleted_at IS NULL",
			pc.KindergartenID, studentID, domain.HealthCheckCompleted, from).
		Order("date DESC, id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	records := make([]ParentHealthRecord, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		records = append(records, ParentHealthRecord{
			CheckDate:          normalizeDate(row.Date),
			OverallStatus:      int8(row.OverallStatus),
			OverallStatusLabel: overallStatusLabel(row.OverallStatus),
			Temperature:        row.Temperature,
			AbnormalItems:      readableAbnormalItems(row),
			Remark:             row.Remark,
			IsIsolated:         row.IsIsolated,
		})
	}
	return records, nil
}

// ==================== Student Profile ====================

func (s *ParentPortalService) GetStudentProfile(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	cn := ""
	if pc.Class != nil {
		cn = pc.Class.Name
	}

	// Calculate age
	age := ""
	if !pc.CurrentStudent.Birthday.IsZero() {
		now := time.Now()
		years := now.Year() - pc.CurrentStudent.Birthday.Year()
		if now.Month() < pc.CurrentStudent.Birthday.Month() || (now.Month() == pc.CurrentStudent.Birthday.Month() && now.Day() < pc.CurrentStudent.Birthday.Day()) {
			years--
		}
		if years > 0 {
			age = fmt.Sprintf("%d岁", years)
		}
	}

	// Get parents list
	var parents []map[string]any
	var sps []domain.StudentParent
	s.db.WithContext(ctx).Preload("User").Where("student_id = ?", pc.CurrentStudent.ID).Find(&sps)
	for _, sp := range sps {
		p := map[string]any{"id": sp.ID, "relation": string(sp.Relation), "relation_label": RelationLabel(sp.Relation), "is_primary": sp.IsPrimary}
		if sp.User != nil {
			p["nickname"] = sp.User.Nickname
			p["avatar"] = sp.User.Avatar
			p["phone"] = maskPhone(sp.User.Phone)
		}
		parents = append(parents, p)
	}

	return map[string]any{
		"id": pc.CurrentStudent.ID, "name": pc.CurrentStudent.Name, "avatar": pc.CurrentStudent.Avatar,
		"gender": pc.CurrentStudent.Gender, "birthday": pc.CurrentStudent.Birthday, "age": age,
		"student_no": pc.CurrentStudent.StudentNo, "health_note": pc.CurrentStudent.HealthNote,
		"class_name": cn, "class_id": pc.CurrentStudent.ClassID,
		"address": pc.CurrentStudent.Address, "enroll_date": pc.CurrentStudent.EnrollDate,
		"parents": parents,
	}, nil
}

// ==================== Home / Announcements / Feeds ====================

func (s *ParentPortalService) ListAnnouncements(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	classIDs := make([]uint64, len(pc.Students))
	for i, st := range pc.Students {
		classIDs[i] = st.ClassID
	}
	if len(classIDs) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0)}, nil
	}
	offset := (page - 1) * pageSize
	db := s.db.WithContext(ctx).Model(&domain.Announcement{}).
		Where("kindergarten_id = ? AND (class_id IN ? OR class_id = 0) AND status = ?", pc.KindergartenID, classIDs, 1)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []domain.Announcement
	if err := db.Order("is_top DESC, published_at DESC").Offset(offset).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	list := make([]map[string]any, len(rows))
	for i, a := range rows {
		list[i] = map[string]any{"id": a.ID, "title": a.Title, "content": a.Content, "is_top": a.IsTop, "published_at": a.PublishedAt.Format("2006-01-02"), "created_at": a.CreatedAt.UTC().Format(time.RFC3339)}
	}
	return map[string]any{"list": list, "total": total}, nil
}

func (s *ParentPortalService) ListHomeFeeds(ctx context.Context, pc *ParentPortalContext, page, pageSize int) (map[string]any, error) {
	if len(pc.Students) == 0 {
		return map[string]any{"list": []any{}, "total": int64(0), "page": page, "page_size": pageSize}, nil
	}

	// 收集孩子所在的所有班级
	classIDs := make([]uint64, 0, len(pc.Students))
	seen := map[uint64]bool{}
	for _, st := range pc.Students {
		if !seen[st.ClassID] {
			seen[st.ClassID] = true
			classIDs = append(classIDs, st.ClassID)
		}
	}

	type feedItem struct {
		Type      string    `json:"type"`
		CreatedAt time.Time `json:"created_at"`
		Data      map[string]any
	}
	var items []feedItem

	// 1) 通知（最近 3 条）——EXISTS 命中判定，一条通知只出一行（多孩合并）
	var notices []domain.Notice
	s.notice.FilterByTargetHit(s.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ? AND status = ?", pc.KindergartenID, domain.NoticeStatusPublished), classIDs).
		Order("published_at DESC").Limit(3).Find(&notices)
	noticeIDs := make([]uint64, len(notices))
	for i, n := range notices {
		noticeIDs[i] = n.ID
	}
	readStatus, _ := s.notice.CheckReadStatus(ctx, pc.UserID, noticeIDs)

	for _, n := range notices {
		ts := n.CreatedAt
		if n.PublishedAt != nil {
			ts = *n.PublishedAt
		}
		items = append(items, feedItem{
			Type: "notice", CreatedAt: ts,
			Data: map[string]any{
				"id":    n.ID,
				"title": n.Title,
				"notice": map[string]any{
					"id": n.ID, "title": n.Title,
					"content_preview": growthContentPreview(n.Content, 80),
					"creator":         map[string]any{"nickname": ""},
					"is_read":         readStatus[n.ID],
				},
			},
		})
	}

	// 2) 成长记录（最近 3 条）。与成长时光同一可见性规则：不含发给其他孩子的个体记录。
	var logs []domain.GrowthDailyLog
	visWhere, visArgs := growthLogVisibilityWhere(pc.KindergartenID, pc.Students, nil)
	s.db.WithContext(ctx).Preload("Creator").Model(&domain.GrowthDailyLog{}).
		Where(visWhere, visArgs...).
		Order("date DESC, id DESC").Limit(3).Find(&logs)
	for _, l := range logs {
		creatorNick := ""
		if l.Creator != nil {
			creatorNick = l.Creator.Nickname
		}
		items = append(items, feedItem{
			Type: "growth", CreatedAt: l.CreatedAt,
			Data: map[string]any{
				"id": l.ID,
				"growth": map[string]any{
					"id":      l.ID,
					"content": l.Content,
					"creator": map[string]any{"nickname": creatorNick},
				},
			},
		})
	}

	// 3) 按时间倒序，取前 pageSize 条
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := int64(len(items))
	offset := (page - 1) * pageSize
	if offset >= len(items) {
		return map[string]any{"list": []any{}, "total": total, "page": page, "page_size": pageSize}, nil
	}
	end := offset + pageSize
	if end > len(items) {
		end = len(items)
	}
	pageItems := items[offset:end]

	list := make([]map[string]any, len(pageItems))
	for i, it := range pageItems {
		list[i] = map[string]any{
			"id":         it.Data["id"],
			"type":       it.Type,
			it.Type:      it.Data[it.Type],
			"created_at": it.CreatedAt.UTC().Format(time.RFC3339),
		}
	}
	return map[string]any{"list": list, "total": total, "page": page, "page_size": pageSize}, nil
}

// ==================== Class Info ====================

func (s *ParentPortalService) ClassInfo(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.Class == nil {
		return nil, ErrParentInvalidParams
	}
	var studentCount int64
	s.db.WithContext(ctx).Model(&domain.Student{}).Where("kindergarten_id = ? AND class_id = ? AND status = ?", pc.KindergartenID, pc.Class.ID, 1).Count(&studentCount)
	var teachers []map[string]any
	var cts []domain.ClassTeacher
	s.db.WithContext(ctx).Preload("User").Where("class_id = ? AND status = ?", pc.Class.ID, 1).Find(&cts)
	for _, ct := range cts {
		t := map[string]any{"id": ct.UserID, "title": string(ct.Title)}
		if ct.User != nil {
			t["nickname"] = ct.User.Nickname
			t["avatar"] = ct.User.Avatar
		}
		teachers = append(teachers, t)
	}
	return map[string]any{"id": pc.Class.ID, "name": pc.Class.Name, "grade": pc.Class.Grade, "academic_year": pc.Class.AcademicYear, "student_count": studentCount, "teachers": teachers}, nil
}

// ==================== Student Profile ====================

func (s *ParentPortalService) UpdateStudentProfile(ctx context.Context, pc *ParentPortalContext, req map[string]any) error {
	if pc.CurrentStudent == nil {
		return ErrParentInvalidParams
	}
	updates := map[string]any{}
	if v, ok := req["avatar"]; ok {
		updates["avatar"] = v
	}
	if v, ok := req["health_note"]; ok {
		updates["health_note"] = v
	}
	if v, ok := req["address"]; ok {
		updates["address"] = v
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Model(pc.CurrentStudent).Updates(updates).Error
}

// ListClassCalendar 查询当前学生所在班级的日历事件（家长可见视图）：
// 仅已发布、受众命中（本班 或 园级）；内部字段不外泄；变更要旨仅在提示窗口内下发。
func (s *ParentPortalService) ListClassCalendar(ctx context.Context, pc *ParentPortalContext, start, end string) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return map[string]any{"list": []any{}, "total": 0}, nil
	}
	enabled, err := loadTimetableEnabled(ctx, s.db, pc.KindergartenID)
	if err != nil {
		return nil, err
	}
	rows, err := s.calendar.FindParentEvents(ctx, pc.KindergartenID, pc.CurrentStudent.ClassID, start, end, enabled)
	if err != nil {
		return nil, err
	}
	meta, err := loadCalendarTypeMeta(ctx, s.db, pc.KindergartenID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	list := make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, parentCalendarEventMap(&rows[i], meta, now))
	}
	return map[string]any{"list": list, "total": len(list)}, nil
}

// GetDailyDesignation 只读查「今日接走人」指定，不产生任何副作用。
//
// 与 GetQRCode 的区别：GetQRCode 会按需建凭证（有副作用），
// 授权人列表页只需展示当前指定，不应因此生成凭证。
// 未指定时返回 designated=false（不是错误）。
func (s *ParentPortalService) GetDailyDesignation(ctx context.Context, pc *ParentPortalContext) (map[string]any, error) {
	if pc.CurrentStudent == nil {
		return nil, ErrParentInvalidParams
	}
	desig, err := s.pickup.FindDailyDesignation(ctx, pc.CurrentStudent.ID, time.Now())
	if err != nil {
		return nil, err
	}
	if desig == nil {
		return map[string]any{
			"student_id": pc.CurrentStudent.ID,
			"designated": false,
			"person":     nil,
		}, nil
	}
	person := map[string]any{
		"id":       desig.AuthorizationID,
		"name":     "",
		"relation": "",
		"avatar":   "",
	}
	if desig.Authorization != nil {
		person["name"] = desig.Authorization.Name
		person["relation"] = desig.Authorization.Relation
		person["avatar"] = desig.Authorization.Avatar
	}
	return map[string]any{
		"student_id":       pc.CurrentStudent.ID,
		"designated":       true,
		"authorization_id": desig.AuthorizationID,
		"person":           person,
	}, nil
}
