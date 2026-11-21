package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

// 账号合并（#165）：把「登录时新建的待绑定账号」并入「园所预创建账号」。
//
// 方向固定：pending（只有客户端身份、无手机号） → target（有手机号、角色、幼儿）。
// 预创建账号的数据是园所人工录入的权威数据，因此只补空缺，不覆盖；pending 的
// 角色/幼儿/积分/教师档案/带班关系属于「尚未认领」的临时数据，直接丢弃。
//
// 整个合并是单事务：任何一步失败都回滚，不会留下半套关联。
var (
	// ErrIdentityMergeIdentityNotFound 待合并的客户端身份不存在。
	// 调用方应先完成登录并确定身份，再调用合并。
	ErrIdentityMergeIdentityNotFound = errors.New("IDENTITY_NOT_FOUND")
	// ErrIdentityMergeTargetNotFound 预创建账号不存在或已软删。
	ErrIdentityMergeTargetNotFound = errors.New("TARGET_USER_NOT_FOUND")
	// ErrIdentityMergePendingNotFound 身份指向的待绑定账号不存在或已软删。
	// 属于数据不一致，调用方应重新登录而不是继续合并。
	ErrIdentityMergePendingNotFound = errors.New("PENDING_USER_NOT_FOUND")
	// ErrIdentityMergePhoneAlreadyBound 预创建账号在同一客户端已绑定其他微信。
	// 这是业务冲突：不能用后到的 openid 覆盖先绑定的身份。
	ErrIdentityMergePhoneAlreadyBound = errors.New("PHONE_ALREADY_BOUND")
)

// IdentityMergeRequest 一次合并的定位信息。
//
// 身份由 (AppID, OpenID) 唯一定位，而不是由调用方传 pending user id：
// 重试时 pending 可能已被删除，只有身份仍能指回 target，从而幂等返回。
type IdentityMergeRequest struct {
	AppID        string
	OpenID       string
	TargetUserID uint64
	// ParentProfileKindergartenID 非 0 时，合并后确保 target 在该园有一条家长档案。
	ParentProfileKindergartenID uint64
}

// IdentityMergeResult 合并结果。AlreadyMerged 为 true 表示重复调用，
// 事务没有做新的写入。
type IdentityMergeResult struct {
	TargetUserID  uint64
	AlreadyMerged bool
}

type IdentityMergeService struct {
	db *gorm.DB
}

func NewIdentityMergeService(db *gorm.DB) *IdentityMergeService {
	return &IdentityMergeService{db: db}
}

func (s *IdentityMergeService) Merge(ctx context.Context, req IdentityMergeRequest) (*IdentityMergeResult, error) {
	if req.AppID == "" || req.OpenID == "" {
		return nil, fmt.Errorf("merge identity: app_id and open_id are required")
	}
	if req.TargetUserID == 0 {
		return nil, fmt.Errorf("merge identity: target user id is required")
	}

	result := &IdentityMergeResult{TargetUserID: req.TargetUserID}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		merged, err := mergeIdentityInTx(tx, req)
		if err != nil {
			return err
		}
		result.AlreadyMerged = merged
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// EnsureParentProfile 单独保证某账号在指定园所有一条家长档案（#167）。
//
// Merge 在「身份已属于 target」的幂等分支会提前返回，跳过档案创建；绑定流程
// 因此无条件再调用一次。已有活跃档案不覆盖，软删档案按 ensureParentProfile
// 的规则复活。
func (s *IdentityMergeService) EnsureParentProfile(ctx context.Context, userID, kindergartenID uint64) error {
	if userID == 0 || kindergartenID == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := lockUser(tx, userID, ErrIdentityMergeTargetNotFound)
		if err != nil {
			return err
		}
		return ensureParentProfile(tx, target, kindergartenID, "")
	})
}

// mergeIdentityInTx 返回 true 表示身份已属于 target（幂等重试）。
func mergeIdentityInTx(tx *gorm.DB, req IdentityMergeRequest) (alreadyMerged bool, err error) {
	// 锁住身份行，防止并发合并同一身份时两边都认为自己是第一个。
	var identity domain.UserIdentity
	err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("app_id = ? AND open_id = ?", req.AppID, req.OpenID).
		First(&identity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, ErrIdentityMergeIdentityNotFound
	}
	if err != nil {
		return false, fmt.Errorf("lock identity: %w", err)
	}

	// 先确认预创建账号仍然可用，再判断幂等：否则 target 被软删后
	// 重试会假报成功，调用方拿到的却是一个登录不进去的账号。
	target, err := lockUser(tx, req.TargetUserID, ErrIdentityMergeTargetNotFound)
	if err != nil {
		return false, err
	}
	if identity.UserID == target.ID {
		return true, nil
	}
	pending, err := lockUser(tx, identity.UserID, ErrIdentityMergePendingNotFound)
	if err != nil {
		return false, err
	}

	// 预创建账号在同一客户端已绑定其他微信时拒绝。先查一遍给出稳定的业务错误；
	// 并发下仍可能被唯一索引拦下（见下方 1062 映射）。
	var conflictCount int64
	if err := tx.Model(&domain.UserIdentity{}).
		Where("user_id = ? AND app_id = ?", target.ID, req.AppID).
		Count(&conflictCount).Error; err != nil {
		return false, fmt.Errorf("check target identity: %w", err)
	}
	if conflictCount > 0 {
		return false, ErrIdentityMergePhoneAlreadyBound
	}

	if err := movePendingIdentities(tx, pending.ID, target.ID); err != nil {
		return false, err
	}
	if err := mergeUniqueKeyRows(tx, pending.ID, target.ID); err != nil {
		return false, err
	}
	if err := moveAuthoredRows(tx, pending.ID, target.ID); err != nil {
		return false, err
	}

	pendingRealName, err := pendingParentRealName(tx, pending.ID, req.ParentProfileKindergartenID)
	if err != nil {
		return false, err
	}
	if err := dropPendingRelations(tx, pending.ID); err != nil {
		return false, err
	}
	if err := ensureParentProfile(tx, target, req.ParentProfileKindergartenID, pendingRealName); err != nil {
		return false, err
	}
	if err := fillEmptyProfile(tx, target, pending); err != nil {
		return false, err
	}
	if err := hardDeleteUser(tx, pending.ID); err != nil {
		return false, err
	}
	return false, nil
}

// uniqueKeyMerge 描述一张「同一用户 + 同一业务键只能一行」的关联表。
// 合并时先删掉待绑定账号与预创建账号键重复的行，再迁移其余行。
//
// 判定重复必须无视 deleted_at：这些表的唯一索引包含软删行，只按活跃行判断
// 会让 UPDATE 撞 1062。同理，删除用原始 SQL 硬删，避免软删行继续占用唯一键。
var uniqueKeyMergeTables = []struct {
	table   string
	bizKeys []string
}{
	{"user_consents", []string{"policy_version"}},
	{"notice_read_logs", []string{"notice_id"}},
	{"teacher_message_reads", []string{"message_type", "reference_id"}},
	{"parent_settings", []string{"kindergarten_id"}},
	{"growth_likes", []string{"log_id"}},
	{"growth_comment_likes", []string{"comment_id"}},
	{"task_read_marks", []string{"task_id", "student_id"}},
	{"task_response_reactions", []string{"response_id"}},
}

func mergeUniqueKeyRows(tx *gorm.DB, pendingID, targetID uint64) error {
	for _, t := range uniqueKeyMergeTables {
		join := ""
		for _, k := range t.bizKeys {
			join += fmt.Sprintf(" AND p.`%s` = t.`%s`", k, k)
		}
		dedupe := fmt.Sprintf(
			"DELETE p FROM `%s` p JOIN `%s` t ON t.user_id = ?%s WHERE p.user_id = ?",
			t.table, t.table, join)
		if err := tx.Exec(dedupe, targetID, pendingID).Error; err != nil {
			return fmt.Errorf("dedupe %s: %w", t.table, err)
		}
		if err := tx.Table(t.table).Where("user_id = ?", pendingID).
			Update("user_id", targetID).Error; err != nil {
			return fmt.Errorf("move %s: %w", t.table, err)
		}
	}
	return nil
}

// moveUserReferences 列出「作者/操作人」这类应随合并转移的引用。
// 预创建账号是这条自然人的正式账号，历史记录不应因为合并而丢失。
var moveUserReferences = []struct {
	table  string
	column string
}{
	{"messages", "user_id"},
	{"wechat_push_logs", "user_id"},
	{"files", "user_id"},
	{"file_records", "uploader_id"},
	{"feedbacks", "user_id"},
	{"growth_comments", "user_id"},
	{"growth_media", "uploader_id"},
	{"attendances", "recorder_id"},
	{"morning_health_checks", "operator_id"},
	{"pickup_records", "operator_id"},
	{"pickup_authorizations", "user_id"},
	{"pickup_authorizations", "creator_id"},
	{"leave_requests", "applicant_id"},
	{"leave_requests", "reviewer_id"},
	{"announcements", "creator_id"},
	{"calendar_events", "creator_id"},
	{"growth_daily_logs", "creator_id"},
	{"notices", "creator_id"},
	{"notice_templates", "creator_id"},
	{"parent_tasks", "creator_id"},
	{"task_check_ins", "user_id"},
	{"task_check_ins", "public_approved_by"},
	{"task_responses", "teacher_id"},
	{"albums", "creator_id"},
	{"albums", "published_by"},
	{"class_star_jars", "created_by"},
	{"class_star_jars", "updated_by"},
	{"content_audits", "submitter_id"},
	{"content_audits", "reviewer_id"},
	{"teacher_invitations", "user_id"},
}

func moveAuthoredRows(tx *gorm.DB, pendingID, targetID uint64) error {
	for _, ref := range moveUserReferences {
		if err := tx.Table(ref.table).Where(ref.column+" = ?", pendingID).
			Update(ref.column, targetID).Error; err != nil {
			return fmt.Errorf("move %s.%s: %w", ref.table, ref.column, err)
		}
	}
	return nil
}

// dropUserReferences 是「不搬、不覆盖」的关联：待绑定账号的角色、幼儿、
// 教师档案、带班关系都是未认领的临时数据，直接丢弃；预创建账号的行保持原样。
var dropUserReferences = []struct {
	table  string
	column string
}{
	{"user_roles", "user_id"},
	{"user_admin_roles", "user_id"},
	{"parent_profiles", "user_id"},
	{"teacher_profiles", "user_id"},
	{"student_parents", "user_id"},
	{"class_teachers", "user_id"},
}

func dropPendingRelations(tx *gorm.DB, pendingID uint64) error {
	for _, ref := range dropUserReferences {
		if err := tx.Exec(
			fmt.Sprintf("DELETE FROM `%s` WHERE `%s` = ?", ref.table, ref.column),
			pendingID).Error; err != nil {
			return fmt.Errorf("drop %s.%s: %w", ref.table, ref.column, err)
		}
	}
	return nil
}

// pendingParentRealName 在删除待绑定账号的档案前取出真实姓名，
// 供补齐预创建账号档案时使用。同一园所至多一条（唯一键），软删行也读取。
func pendingParentRealName(tx *gorm.DB, pendingID, kindergartenID uint64) (string, error) {
	if kindergartenID == 0 {
		return "", nil
	}
	var names []string
	if err := tx.Unscoped().Model(&domain.ParentProfile{}).
		Where("user_id = ? AND kindergarten_id = ?", pendingID, kindergartenID).
		Limit(1).Pluck("real_name", &names).Error; err != nil {
		return "", fmt.Errorf("load pending parent profile: %w", err)
	}
	if len(names) == 0 {
		return "", nil
	}
	return names[0], nil
}

// ensureParentProfile 保证预创建账号在指定园所有一条可用的家长档案。
//
// 已有活跃档案时不动（不覆盖）；只有软删档案占着 (user_id, kindergarten_id)
// 唯一键时复活它，而不是硬删重建——档案上的其他字段仍然有效。
func ensureParentProfile(tx *gorm.DB, target *domain.User, kindergartenID uint64, pendingRealName string) error {
	if kindergartenID == 0 {
		return nil
	}
	var profile domain.ParentProfile
	err := tx.Unscoped().
		Where("user_id = ? AND kindergarten_id = ?", target.ID, kindergartenID).
		First(&profile).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		realName := pendingRealName
		if realName == "" {
			realName = target.Nickname
		}
		created := &domain.ParentProfile{
			UserID:         target.ID,
			KindergartenID: kindergartenID,
			RealName:       realName,
			Status:         1,
		}
		if err := tx.Create(created).Error; err != nil {
			return fmt.Errorf("create parent profile: %w", err)
		}
		return nil
	case err != nil:
		return fmt.Errorf("load parent profile: %w", err)
	}

	if !profile.DeletedAt.Valid {
		return nil
	}
	updates := map[string]any{"deleted_at": nil}
	if profile.RealName == "" && pendingRealName != "" {
		updates["real_name"] = pendingRealName
	}
	if err := tx.Unscoped().Model(&domain.ParentProfile{}).
		Where("id = ?", profile.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("revive parent profile: %w", err)
	}
	return nil
}

func lockUser(tx *gorm.DB, id uint64, notFound error) (*domain.User, error) {
	var user domain.User
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock user %d: %w", id, err)
	}
	return &user, nil
}

// movePendingIdentities 把待绑定账号的全部客户端身份挂到预创建账号上。
//
// 不只移动当前登录的那一条：同一自然人的其他客户端身份也应一并归并。
// 若预创建账号在另一个 app 上已有身份，UPDATE 会撞 (user_id, app_id) 唯一键，
// 这同样属于「已绑定其他微信」，映射成业务冲突而不是 500。
func movePendingIdentities(tx *gorm.DB, pendingID, targetID uint64) error {
	err := tx.Model(&domain.UserIdentity{}).
		Where("user_id = ?", pendingID).
		Update("user_id", targetID).Error
	if err == nil {
		return nil
	}
	if repository.IsDuplicateKey(err) {
		return ErrIdentityMergePhoneAlreadyBound
	}
	return fmt.Errorf("move identities %d -> %d: %w", pendingID, targetID, err)
}

// fillEmptyProfile 只补预创建账号的空缺字段，不覆盖园所录入的数据。
func fillEmptyProfile(tx *gorm.DB, target, pending *domain.User) error {
	updates := map[string]any{}
	if target.Nickname == "" && pending.Nickname != "" {
		updates["nickname"] = pending.Nickname
	}
	if target.Avatar == "" && pending.Avatar != "" {
		updates["avatar"] = pending.Avatar
	}
	if len(updates) == 0 {
		return nil
	}
	if err := tx.Model(&domain.User{}).Where("id = ?", target.ID).Updates(updates).Error; err != nil {
		return fmt.Errorf("fill target profile: %w", err)
	}
	return nil
}

func hardDeleteUser(tx *gorm.DB, id uint64) error {
	if err := tx.Unscoped().Delete(&domain.User{}, id).Error; err != nil {
		return fmt.Errorf("hard delete pending user %d: %w", id, err)
	}
	return nil
}
