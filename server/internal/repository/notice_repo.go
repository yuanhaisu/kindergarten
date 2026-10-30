package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
)

type NoticeRepo struct {
	db *gorm.DB
}

func NewNoticeRepo(db *gorm.DB) *NoticeRepo {
	return &NoticeRepo{db: db}
}

// NoticeListRow 管理端通知列表行。
// ClassNames 为目标班级名列表（多目标按目标表聚合），不再是单值 class_id/class_name。
type NoticeListRow struct {
	ID          uint64     `gorm:"column:id"`
	Category    string     `gorm:"column:category"`
	Title       string     `gorm:"column:title"`
	ClassNames  []string   `gorm:"-"`
	CreatorID   uint64     `gorm:"column:creator_id"`
	CreatorNick string     `gorm:"column:creator_nick"`
	Status      int8       `gorm:"column:status"`
	PublishedAt *time.Time `gorm:"column:published_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	ReadUsers   int64      `gorm:"column:read_users"`
	ParentTotal int64      `gorm:"column:parent_total"`
}

// noticeTargetParentTotalSQL 目标范围去重家长数——分母的唯一实现，两端统计共用。
// 同一家长有多个孩子在目标范围内只计一人（COUNT(DISTINCT sp.user_id)）。
const noticeTargetParentTotalSQL = `(SELECT COUNT(DISTINCT sp.user_id) FROM notice_targets t
	 INNER JOIN students s ON s.class_id = t.class_id AND s.kindergarten_id = t.kindergarten_id AND s.deleted_at IS NULL
	 INNER JOIN student_parents sp ON sp.student_id = s.id AND sp.deleted_at IS NULL
	 WHERE t.notice_id = n.id AND t.deleted_at IS NULL)`

// FilterByTargetHit 把「目标集命中任一给定班级」的 EXISTS 条件挂到 notices 查询上。
// 家长端传孩子班级集、老师端传任教班级集：命中即可见，一条通知只出一行（多孩合并免费获得）。
func (r *NoticeRepo) FilterByTargetHit(q *gorm.DB, classIDs []uint64) *gorm.DB {
	if len(classIDs) == 0 {
		return q.Where("1 = 0")
	}
	return q.Where(`EXISTS (SELECT 1 FROM notice_targets t
		WHERE t.notice_id = notices.id AND t.deleted_at IS NULL AND t.class_id IN ?)`, classIDs)
}

// FilterByTargetHitOrOwnDraft 「目标集命中且非他人草稿」或「本人草稿」。
// 草稿允许没有目标班（D2），只按 EXISTS 过滤会把作者自己的空目标草稿藏掉；
// 有目标班的草稿也不能因此对同班其他老师可见。
func (r *NoticeRepo) FilterByTargetHitOrOwnDraft(q *gorm.DB, classIDs []uint64, userID uint64) *gorm.DB {
	if len(classIDs) == 0 {
		return q.Where("status = ? AND creator_id = ?", domain.NoticeStatusDraft, userID)
	}
	return q.Where(`((EXISTS (SELECT 1 FROM notice_targets t
		WHERE t.notice_id = notices.id AND t.deleted_at IS NULL AND t.class_id IN ?)
		AND (status <> ? OR creator_id = ?))
		OR (status = ? AND creator_id = ?))`,
		classIDs, domain.NoticeStatusDraft, userID, domain.NoticeStatusDraft, userID)
}

// ExistsNoticeTargetHit 单条通知的目标集是否命中任一给定班级。
func (r *NoticeRepo) ExistsNoticeTargetHit(ctx context.Context, noticeID uint64, classIDs []uint64) (bool, error) {
	if len(classIDs) == 0 {
		return false, nil
	}
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.NoticeTarget{}).
		Where("notice_id = ? AND deleted_at IS NULL AND class_id IN ?", noticeID, classIDs).
		Count(&n).Error
	return n > 0, err
}

// ListNoticeTargetClassIDs 通知的目标班级 ID 集。
func (r *NoticeRepo) ListNoticeTargetClassIDs(ctx context.Context, noticeID uint64) ([]uint64, error) {
	var ids []uint64
	err := r.db.WithContext(ctx).Model(&domain.NoticeTarget{}).
		Where("notice_id = ? AND deleted_at IS NULL", noticeID).
		Order("class_id ASC").
		Pluck("class_id", &ids).Error
	return ids, err
}

// ClassNamesByNoticeIDs 批量取通知的目标班级名（按 class_id 升序），供列表聚合。
func (r *NoticeRepo) ClassNamesByNoticeIDs(ctx context.Context, kindergartenID uint64, noticeIDs []uint64) (map[uint64][]string, error) {
	out := make(map[uint64][]string, len(noticeIDs))
	if len(noticeIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		NoticeID uint64 `gorm:"column:notice_id"`
		Name     string `gorm:"column:name"`
	}
	err := r.db.WithContext(ctx).Table("notice_targets AS t").
		Select("t.notice_id, c.name").
		Joins("INNER JOIN classes AS c ON c.id = t.class_id AND c.deleted_at IS NULL").
		Where("t.kindergarten_id = ? AND t.deleted_at IS NULL AND t.notice_id IN ?", kindergartenID, noticeIDs).
		Order("t.class_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.NoticeID] = append(out[row.NoticeID], row.Name)
	}
	return out, nil
}

// ReplaceNoticeTargets 整体替换通知目标集（编辑时显式重选，不做增量）。
// 目标行硬删重建：联合唯一键 (notice_id, class_id) 与软删共存会挡住重建。
func (r *NoticeRepo) ReplaceNoticeTargets(ctx context.Context, kindergartenID, noticeID uint64, classIDs []uint64) error {
	uniq := make([]uint64, 0, len(classIDs))
	seen := make(map[uint64]struct{}, len(classIDs))
	for _, id := range classIDs {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniq = append(uniq, id)
	}
	if len(uniq) == 0 {
		return errors.New("EMPTY_NOTICE_TARGETS")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("notice_id = ?", noticeID).Delete(&domain.NoticeTarget{}).Error; err != nil {
			return err
		}
		rows := make([]domain.NoticeTarget, 0, len(uniq))
		for _, classID := range uniq {
			rows = append(rows, domain.NoticeTarget{
				NoticeID: noticeID, ClassID: classID, KindergartenID: kindergartenID,
			})
		}
		return tx.Create(&rows).Error
	})
}

// ClearNoticeTargets 清空通知目标集：草稿允许没有接收范围（D2），发布前必须重新选择。
func (r *NoticeRepo) ClearNoticeTargets(ctx context.Context, noticeID uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("notice_id = ?", noticeID).
		Delete(&domain.NoticeTarget{}).Error
}

func (r *NoticeRepo) baseNoticeListQuery(
	kindergartenID uint64,
	classID *uint64,
	category string,
	status *int8,
	keyword string,
) *gorm.DB {
	q := r.db.Table("notices AS n").
		Joins("INNER JOIN users AS u ON u.id = n.creator_id AND u.deleted_at IS NULL").
		Where("n.kindergarten_id = ? AND n.deleted_at IS NULL", kindergartenID)
	if classID != nil && *classID > 0 {
		// 按目标班级过滤：命中目标集即返回
		q = q.Where(`EXISTS (SELECT 1 FROM notice_targets t
			WHERE t.notice_id = n.id AND t.deleted_at IS NULL AND t.class_id = ?)`, *classID)
	}
	if category != "" {
		q = q.Where("n.category = ?", category)
	}
	if status != nil {
		q = q.Where("n.status = ?", *status)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		q = q.Where("n.title LIKE ?", "%"+kw+"%")
	}
	return q
}

func (r *NoticeRepo) ListNotices(
	ctx context.Context,
	kindergartenID uint64,
	classID *uint64,
	category string,
	status *int8,
	keyword string,
	offset, limit int,
) ([]NoticeListRow, int64, error) {
	base := r.baseNoticeListQuery(kindergartenID, classID, category, status, keyword)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	sel := base.Session(&gorm.Session{}).
		Select(`n.id,
			n.category,
			n.title,
			n.creator_id,
			u.nickname AS creator_nick,
			n.status,
			n.published_at,
			n.created_at,
			(SELECT COUNT(DISTINCT rl.user_id) FROM notice_read_logs rl WHERE rl.notice_id = n.id) AS read_users,
			`+noticeTargetParentTotalSQL+` AS parent_total`).
		Order("n.created_at DESC, n.id DESC").
		Offset(offset).Limit(limit)
	var rows []NoticeListRow
	if err := sel.Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	names, err := r.ClassNamesByNoticeIDs(ctx, kindergartenID, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].ClassNames = names[rows[i].ID]
	}
	return rows, total, nil
}

// FindNoticeByID 园所内通知（未删），预加载班级与创建者。
func (r *NoticeRepo) FindNoticeByID(ctx context.Context, kindergartenID, noticeID uint64) (*domain.Notice, error) {
	var n domain.Notice
	err := r.db.WithContext(ctx).
		Preload("Class").
		Preload("Creator").
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, noticeID).
		First(&n).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *NoticeRepo) ListAttachmentsByNoticeID(ctx context.Context, noticeID uint64) ([]domain.NoticeAttachment, error) {
	var list []domain.NoticeAttachment
	err := r.db.WithContext(ctx).
		Where("notice_id = ?", noticeID).
		Order("sort_num ASC, id ASC").
		Find(&list).Error
	return list, err
}

// AttachmentCountsByNoticeIDs 批量统计附件数，老师端列表 attachments_count 的唯一来源。
// 一次聚合而不是逐条 N+1；没有附件的通知不出现在 map 中，调用方按 0 处理。
func (r *NoticeRepo) AttachmentCountsByNoticeIDs(ctx context.Context, noticeIDs []uint64) (map[uint64]int, error) {
	out := make(map[uint64]int, len(noticeIDs))
	if len(noticeIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		NoticeID uint64 `gorm:"column:notice_id"`
		Cnt      int    `gorm:"column:cnt"`
	}
	err := r.db.WithContext(ctx).Model(&domain.NoticeAttachment{}).
		Select("notice_id, COUNT(*) AS cnt").
		Where("notice_id IN ?", noticeIDs).
		Group("notice_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.NoticeID] = row.Cnt
	}
	return out, nil
}

func (r *NoticeRepo) FindLatestAudit(ctx context.Context, kindergartenID, noticeID uint64) (*domain.ContentAudit, error) {
	var ca domain.ContentAudit
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND target_type = ? AND target_id = ? AND deleted_at IS NULL",
			kindergartenID, domain.AuditTargetNotice, noticeID).
		Order("id DESC").
		First(&ca).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ca, nil
}

// SaveContentAudit 新建或更新同目标审核记录（单条覆盖）。
func (r *NoticeRepo) SaveContentAudit(ctx context.Context, ca *domain.ContentAudit) error {
	var existing domain.ContentAudit
	err := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND target_type = ? AND target_id = ? AND deleted_at IS NULL",
			ca.KindergartenID, ca.TargetType, ca.TargetID).
		First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(ca).Error
		}
		return err
	}
	existing.ReviewerID = ca.ReviewerID
	existing.Status = ca.Status
	existing.ReviewedAt = ca.ReviewedAt
	existing.RejectReason = ca.RejectReason
	return r.db.WithContext(ctx).Save(&existing).Error
}

// FindUserDisplay 取用户昵称与头像（未删用户）。
func (r *NoticeRepo) FindUserDisplay(ctx context.Context, userID uint64) (nickname, avatar string, ok bool) {
	var u domain.User
	err := r.db.WithContext(ctx).Select("id", "nickname", "avatar").
		Where("id = ? AND deleted_at IS NULL", userID).
		First(&u).Error
	if err != nil {
		return "", "", false
	}
	return u.Nickname, u.Avatar, true
}

// NoticeReadStats 某通知的已读人数、确认人数、目标范围去重家长数（分母，多孩只计一人）。
// 老师端/管理端共用这一处，保证两端 read_rate 一致。已读、已确认同为按 user_id 去重。
func (r *NoticeRepo) NoticeReadStats(ctx context.Context, kindergartenID, noticeID uint64) (readUsers, confirmed, parentTotal int64, err error) {
	err = r.db.WithContext(ctx).Raw(`
SELECT COALESCE(COUNT(DISTINCT sp.user_id), 0) AS parent_total
FROM notice_targets t
INNER JOIN students s ON s.class_id = t.class_id AND s.kindergarten_id = t.kindergarten_id AND s.deleted_at IS NULL
INNER JOIN student_parents sp ON sp.student_id = s.id AND sp.deleted_at IS NULL
WHERE t.notice_id = ? AND t.kindergarten_id = ? AND t.deleted_at IS NULL
`, noticeID, kindergartenID).Scan(&parentTotal).Error
	if err != nil {
		return 0, 0, 0, err
	}
	err = r.db.WithContext(ctx).Raw(
		`SELECT COUNT(DISTINCT user_id) FROM notice_read_logs WHERE notice_id = ?`,
		noticeID,
	).Scan(&readUsers).Error
	if err != nil {
		return 0, 0, 0, err
	}
	err = r.db.WithContext(ctx).Model(&domain.NoticeReadLog{}).
		Where("notice_id = ? AND confirmed_at IS NOT NULL", noticeID).
		Count(&confirmed).Error
	if err != nil {
		return 0, 0, 0, err
	}
	return readUsers, confirmed, parentTotal, nil
}

func (r *NoticeRepo) IsClassInKindergarten(ctx context.Context, kindergartenID, classID uint64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&domain.Class{}).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, classID).
		Count(&n).Error
	return n > 0, err
}

// ListClassIDsByGrades 年级圈选的班级快照展开：发布时刻该年级下的班级集合。
// 调用方在发布时取一次并落目标表，之后班级增减不影响已发布通知（D2 快照语义）。
func (r *NoticeRepo) ListClassIDsByGrades(ctx context.Context, kindergartenID uint64, grades []string) ([]uint64, error) {
	if len(grades) == 0 {
		return nil, nil
	}
	var ids []uint64
	err := r.db.WithContext(ctx).Model(&domain.Class{}).
		Where("kindergarten_id = ? AND grade IN ? AND deleted_at IS NULL", kindergartenID, grades).
		Order("id ASC").
		Pluck("id", &ids).Error
	return ids, err
}

func (r *NoticeRepo) CreateNotice(ctx context.Context, n *domain.Notice) error {
	return r.db.WithContext(ctx).Create(n).Error
}

func (r *NoticeRepo) CreateAttachments(ctx context.Context, list []domain.NoticeAttachment) error {
	if len(list) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&list).Error
}

// ListDueScheduledNotices 到期待发布的定时通知（status=1 且 scheduled_at<=now），
// 按预定时间升序供后台 worker 逐条发布；软删通知不参与。
func (r *NoticeRepo) ListDueScheduledNotices(ctx context.Context, now time.Time, limit int) ([]domain.Notice, error) {
	if limit <= 0 {
		limit = 100
	}
	var list []domain.Notice
	err := r.db.WithContext(ctx).
		Where("status = ? AND scheduled_at <= ? AND deleted_at IS NULL", domain.NoticeStatusScheduled, now).
		Order("scheduled_at ASC, id ASC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

// PublishScheduledNotice 条件发布：只有仍处于定时状态且已到点的通知才会被置为已发布。
// 返回 true 表示本次调用真正发布了该通知；false 表示 0 行受影响——已被其它实例/轮次
// 发布，或状态在扫描后变化，调用方直接跳过即可（幂等与多实例安全靠这个条件更新）。
func (r *NoticeRepo) PublishScheduledNotice(ctx context.Context, noticeID uint64, now time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("id = ? AND status = ? AND scheduled_at <= ? AND deleted_at IS NULL",
			noticeID, domain.NoticeStatusScheduled, now).
		Updates(map[string]interface{}{
			"status":       domain.NoticeStatusPublished,
			"published_at": now,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

func (r *NoticeRepo) UpdateNoticeFields(ctx context.Context, kindergartenID, noticeID uint64, patch map[string]interface{}) (int64, error) {
	res := r.db.WithContext(ctx).Model(&domain.Notice{}).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, noticeID).
		Updates(patch)
	return res.RowsAffected, res.Error
}

func (r *NoticeRepo) SoftDeleteNotice(ctx context.Context, kindergartenID, noticeID uint64) (int64, error) {
	res := r.db.WithContext(ctx).
		Where("kindergarten_id = ? AND id = ? AND deleted_at IS NULL", kindergartenID, noticeID).
		Delete(&domain.Notice{})
	return res.RowsAffected, res.Error
}

func (r *NoticeRepo) DeleteAttachmentsByNoticeID(ctx context.Context, noticeID uint64) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("notice_id = ?", noticeID).
		Delete(&domain.NoticeAttachment{}).Error
}

// RecordReadLog 写入通知已读记录（幂等：同一用户+通知仅一条记录，重复调用更新 read_at）。
func (r *NoticeRepo) RecordReadLog(ctx context.Context, noticeID, userID uint64) error {
	now := time.Now()
	log := domain.NoticeReadLog{NoticeID: noticeID, UserID: userID, ReadAt: now}
	return r.db.WithContext(ctx).
		Where("notice_id = ? AND user_id = ?", noticeID, userID).
		Assign(map[string]any{"read_at": now}).
		FirstOrCreate(&log).Error
}

// RecordConfirmLog 写入通知确认记录。
func (r *NoticeRepo) RecordConfirmLog(ctx context.Context, noticeID, userID uint64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&domain.NoticeReadLog{}).
		Where("notice_id = ? AND user_id = ?", noticeID, userID).
		Updates(map[string]any{"confirmed_at": now}).Error
}

// CheckReadStatus 批量查询某用户对一批通知的已读状态。
func (r *NoticeRepo) CheckReadStatus(ctx context.Context, userID uint64, noticeIDs []uint64) (map[uint64]bool, error) {
	if len(noticeIDs) == 0 {
		return map[uint64]bool{}, nil
	}
	var rows []domain.NoticeReadLog
	if err := r.db.WithContext(ctx).Where("user_id = ? AND notice_id IN ?", userID, noticeIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[uint64]bool, len(rows))
	for _, row := range rows {
		result[row.NoticeID] = !row.ReadAt.IsZero()
	}
	return result, nil
}

// FindReadLogsByUserAndNoticeIDs 批量取某用户对一批通知的已读/确认记录（按 notice_id 索引）。
// 家长端列表的 is_read/is_confirmed 与详情的 read_at/confirmed_at 共用这一处：
// 逐条查会把一页 20 条退化成 20 次查询。没有记录的通知不出现在 map 中。
func (r *NoticeRepo) FindReadLogsByUserAndNoticeIDs(ctx context.Context, userID uint64, noticeIDs []uint64) (map[uint64]domain.NoticeReadLog, error) {
	out := make(map[uint64]domain.NoticeReadLog, len(noticeIDs))
	if len(noticeIDs) == 0 {
		return out, nil
	}
	var rows []domain.NoticeReadLog
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND notice_id IN ?", userID, noticeIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.NoticeID] = row
	}
	return out, nil
}
