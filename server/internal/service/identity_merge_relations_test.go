package service_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/service"
)

func execSQL(t *testing.T, db *gorm.DB, query string, args ...any) {
	t.Helper()
	require.NoError(t, db.Exec(query, args...).Error, "执行测试 SQL：%s", query)
}

func countSQL(t *testing.T, db *gorm.DB, query string, args ...any) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(query, args...).Scan(&n).Error, "统计测试 SQL：%s", query)
	return n
}

// 隐私同意按「用户 × 版本」唯一。预创建账号已有的版本保留原记录（来源/时间都不变），
// 只有待绑定账号独有的版本迁移过去，重复版本直接丢弃。
func TestIdentityMerge_ConsentKeepsTargetDuplicateAndMovesPendingOnly(t *testing.T) {
	f := newMergeFixture(t)
	targetAt := time.Now().Add(-2 * time.Hour).Truncate(time.Millisecond)
	pendingAt := time.Now().Add(-time.Hour).Truncate(time.Millisecond)

	execSQL(t, f.DB, "INSERT INTO user_consents (user_id, policy_version, source, ip, user_agent, consented_at) VALUES (?,?,?,?,?,?)",
		f.Target.ID, "2026-11-v1", "admin", "10.0.0.1", "target-agent", targetAt)
	execSQL(t, f.DB, "INSERT INTO user_consents (user_id, policy_version, source, ip, user_agent, consented_at) VALUES (?,?,?,?,?,?)",
		f.Pending.ID, "2026-11-v1", "parent", "10.0.0.2", "pending-agent", pendingAt)
	execSQL(t, f.DB, "INSERT INTO user_consents (user_id, policy_version, source, ip, user_agent, consented_at) VALUES (?,?,?,?,?,?)",
		f.Pending.ID, "2026-12-v2", "parent", "10.0.0.3", "pending-agent", pendingAt)

	_, err := f.merge(t)
	require.NoError(t, err)

	type consentRow struct {
		PolicyVersion string
		Source        string
		ConsentedAt   time.Time
	}
	var rows []consentRow
	require.NoError(t, f.DB.Raw(
		"SELECT policy_version, source, consented_at FROM user_consents WHERE user_id = ? ORDER BY policy_version",
		f.Target.ID).Scan(&rows).Error)
	require.Len(t, rows, 2)
	assert.Equal(t, "2026-11-v1", rows[0].PolicyVersion)
	assert.Equal(t, "admin", rows[0].Source, "同版本必须保留预创建账号的同意记录")
	assert.True(t, targetAt.Equal(rows[0].ConsentedAt), "不得用待绑定账号的时间覆盖")
	assert.Equal(t, "2026-12-v2", rows[1].PolicyVersion)
	assert.Equal(t, "parent", rows[1].Source)
	assert.True(t, pendingAt.Equal(rows[1].ConsentedAt))
	assert.Zero(t, countSQL(t, f.DB, "SELECT COUNT(*) FROM user_consents WHERE user_id = ?", f.Pending.ID))
}

// 带唯一键的关联表：目标已有同键行时删掉待绑定账号的重复行；仅待绑定有的行迁移过去。
// 注意唯一索引包含软删行，所以重复判定必须无视 deleted_at。
func TestIdentityMerge_DedupesPerTableUniqueKeys(t *testing.T) {
	t.Run("notice_read_logs 含软删重复行", func(t *testing.T) {
		f := newMergeFixture(t)
		execSQL(t, f.DB, "INSERT INTO notice_read_logs (notice_id, user_id, read_at, deleted_at) VALUES (?,?,NOW(),NOW())",
			9001, f.Target.ID)
		execSQL(t, f.DB, "INSERT INTO notice_read_logs (notice_id, user_id, read_at) VALUES (?,?,NOW())",
			9001, f.Pending.ID)
		execSQL(t, f.DB, "INSERT INTO notice_read_logs (notice_id, user_id, read_at) VALUES (?,?,NOW())",
			9002, f.Pending.ID)

		_, err := f.merge(t)
		require.NoError(t, err)

		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM notice_read_logs WHERE notice_id = 9001"), "同一通知只保留一行")
		assert.Equal(t, int64(0), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM notice_read_logs WHERE notice_id = 9001 AND user_id = ?", f.Pending.ID))
		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM notice_read_logs WHERE notice_id = 9002 AND user_id = ?", f.Target.ID),
			"待绑定账号独有的已读记录应迁移")
	})

	t.Run("teacher_message_reads", func(t *testing.T) {
		f := newMergeFixture(t)
		execSQL(t, f.DB, "INSERT INTO teacher_message_reads (user_id, message_type, reference_id) VALUES (?,?,?)",
			f.Target.ID, "leave", 9201)
		execSQL(t, f.DB, "INSERT INTO teacher_message_reads (user_id, message_type, reference_id) VALUES (?,?,?)",
			f.Pending.ID, "leave", 9201)
		execSQL(t, f.DB, "INSERT INTO teacher_message_reads (user_id, message_type, reference_id) VALUES (?,?,?)",
			f.Pending.ID, "leave", 9202)

		_, err := f.merge(t)
		require.NoError(t, err)

		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM teacher_message_reads WHERE message_type = 'leave' AND reference_id = 9201"))
		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM teacher_message_reads WHERE reference_id = 9202 AND user_id = ?", f.Target.ID))
	})

	t.Run("parent_settings", func(t *testing.T) {
		f := newMergeFixture(t)
		insertSettings := func(userID, kgID uint64) {
			execSQL(t, f.DB, `INSERT INTO parent_settings
				(user_id, kindergarten_id, notice_push, leave_push, pickup_push, growth_push, task_push,
				 announcement_push, show_phone_to_teacher, allow_growth_share, allow_class_share, allow_external_share)
				VALUES (?,?,1,1,1,1,1,1,1,1,1,1)`, userID, kgID)
		}
		insertSettings(f.Target.ID, f.KG.ID)
		insertSettings(f.Pending.ID, f.KG.ID)
		insertSettings(f.Pending.ID, f.KG.ID+1000000)

		_, err := f.merge(t)
		require.NoError(t, err)

		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM parent_settings WHERE kindergarten_id = ?", f.KG.ID))
		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM parent_settings WHERE kindergarten_id = ? AND user_id = ?",
			f.KG.ID+1000000, f.Target.ID))
	})

	t.Run("growth_likes", func(t *testing.T) {
		f := newMergeFixture(t)
		execSQL(t, f.DB, "INSERT INTO growth_likes (log_id, user_id) VALUES (?,?)", 9101, f.Target.ID)
		execSQL(t, f.DB, "INSERT INTO growth_likes (log_id, user_id) VALUES (?,?)", 9101, f.Pending.ID)
		execSQL(t, f.DB, "INSERT INTO growth_likes (log_id, user_id) VALUES (?,?)", 9102, f.Pending.ID)

		_, err := f.merge(t)
		require.NoError(t, err)

		assert.Equal(t, int64(1), countSQL(t, f.DB, "SELECT COUNT(*) FROM growth_likes WHERE log_id = 9101"))
		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM growth_likes WHERE log_id = 9102 AND user_id = ?", f.Target.ID))
	})

	t.Run("growth_comment_likes", func(t *testing.T) {
		f := newMergeFixture(t)
		execSQL(t, f.DB, "INSERT INTO growth_comment_likes (comment_id, user_id) VALUES (?,?)", 9301, f.Target.ID)
		execSQL(t, f.DB, "INSERT INTO growth_comment_likes (comment_id, user_id) VALUES (?,?)", 9301, f.Pending.ID)
		execSQL(t, f.DB, "INSERT INTO growth_comment_likes (comment_id, user_id) VALUES (?,?)", 9302, f.Pending.ID)

		_, err := f.merge(t)
		require.NoError(t, err)

		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM growth_comment_likes WHERE comment_id = 9301"))
		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM growth_comment_likes WHERE comment_id = 9302 AND user_id = ?", f.Target.ID))
	})

	t.Run("task_read_marks", func(t *testing.T) {
		f := newMergeFixture(t)
		execSQL(t, f.DB, "INSERT INTO task_read_marks (kindergarten_id, task_id, user_id, student_id) VALUES (?,?,?,?)",
			f.KG.ID, 9401, f.Target.ID, f.Student.ID)
		execSQL(t, f.DB, "INSERT INTO task_read_marks (kindergarten_id, task_id, user_id, student_id) VALUES (?,?,?,?)",
			f.KG.ID, 9401, f.Pending.ID, f.Student.ID)
		execSQL(t, f.DB, "INSERT INTO task_read_marks (kindergarten_id, task_id, user_id, student_id) VALUES (?,?,?,?)",
			f.KG.ID, 9402, f.Pending.ID, f.Student.ID)

		_, err := f.merge(t)
		require.NoError(t, err)

		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM task_read_marks WHERE task_id = 9401"))
		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM task_read_marks WHERE task_id = 9402 AND user_id = ?", f.Target.ID))
	})

	t.Run("task_response_reactions", func(t *testing.T) {
		f := newMergeFixture(t)
		execSQL(t, f.DB, "INSERT INTO task_response_reactions (kindergarten_id, response_id, user_id, reaction) VALUES (?,?,?,?)",
			f.KG.ID, 9501, f.Target.ID, "like")
		execSQL(t, f.DB, "INSERT INTO task_response_reactions (kindergarten_id, response_id, user_id, reaction) VALUES (?,?,?,?)",
			f.KG.ID, 9501, f.Pending.ID, "like")
		execSQL(t, f.DB, "INSERT INTO task_response_reactions (kindergarten_id, response_id, user_id, reaction) VALUES (?,?,?,?)",
			f.KG.ID, 9502, f.Pending.ID, "heart")

		_, err := f.merge(t)
		require.NoError(t, err)

		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM task_response_reactions WHERE response_id = 9501"))
		assert.Equal(t, int64(1), countSQL(t, f.DB,
			"SELECT COUNT(*) FROM task_response_reactions WHERE response_id = 9502 AND user_id = ?", f.Target.ID))
	})
}

// 角色、幼儿、教师档案、带班关系属于「不搬、不覆盖」：待绑定账号的行删除，
// 预创建账号原有的行保持原样。
func TestIdentityMerge_DropsPendingRelationsAndKeepsTargetOnes(t *testing.T) {
	f := newMergeFixture(t)

	execSQL(t, f.DB, "INSERT INTO user_roles (user_id, role, kindergarten_id) VALUES (?,?,?)",
		f.Target.ID, "parent", f.KG.ID)
	execSQL(t, f.DB, "INSERT INTO user_roles (user_id, role, kindergarten_id) VALUES (?,?,?)",
		f.Pending.ID, "teacher", f.KG.ID)

	execSQL(t, f.DB, "INSERT INTO student_parents (student_id, user_id, relation) VALUES (?,?,?)",
		f.Student.ID, f.Target.ID, "妈妈")
	execSQL(t, f.DB, "INSERT INTO student_parents (student_id, user_id, relation) VALUES (?,?,?)",
		f.Student.ID, f.Pending.ID, "爸爸")

	execSQL(t, f.DB, "INSERT INTO class_teachers (class_id, user_id, title) VALUES (?,?,?)",
		f.Class.ID, f.Target.ID, "主班")
	execSQL(t, f.DB, "INSERT INTO class_teachers (class_id, user_id, title) VALUES (?,?,?)",
		f.Class.ID, f.Pending.ID, "配班")

	execSQL(t, f.DB, "INSERT INTO parent_profiles (user_id, kindergarten_id, real_name, status) VALUES (?,?,?,1)",
		f.Target.ID, f.KG.ID, "预创建家长")
	execSQL(t, f.DB, "INSERT INTO parent_profiles (user_id, kindergarten_id, real_name, status) VALUES (?,?,?,1)",
		f.Pending.ID, f.KG.ID, "待绑定家长")

	execSQL(t, f.DB, "INSERT INTO teacher_profiles (user_id, kindergarten_id, real_name) VALUES (?,?,?)",
		f.Pending.ID, f.KG.ID, "待绑定老师")

	execSQL(t, f.DB, "INSERT INTO user_admin_roles (user_id, admin_role_id, kindergarten_id) VALUES (?,?,?)",
		f.Pending.ID, 1, f.KG.ID)

	_, err := f.merge(t)
	require.NoError(t, err)

	for _, tc := range []struct {
		table string
	}{
		{"user_roles"}, {"student_parents"}, {"class_teachers"},
		{"parent_profiles"}, {"teacher_profiles"}, {"user_admin_roles"},
	} {
		assert.Zero(t, countSQL(t, f.DB,
			fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE user_id = ?", tc.table), f.Pending.ID),
			"%s 不得保留待绑定账号的行", tc.table)
	}

	assert.Equal(t, int64(1), countSQL(t, f.DB,
		"SELECT COUNT(*) FROM user_roles WHERE user_id = ? AND role = 'parent'", f.Target.ID))
	assert.Equal(t, int64(1), countSQL(t, f.DB,
		"SELECT COUNT(*) FROM student_parents WHERE user_id = ? AND relation = '妈妈'", f.Target.ID))
	assert.Equal(t, int64(1), countSQL(t, f.DB,
		"SELECT COUNT(*) FROM class_teachers WHERE user_id = ? AND title = '主班'", f.Target.ID))
	assert.Equal(t, int64(1), countSQL(t, f.DB,
		"SELECT COUNT(*) FROM parent_profiles WHERE user_id = ? AND real_name = '预创建家长'", f.Target.ID),
		"预创建账号的家长档案不得被覆盖")
}

// 家长档案缺失时补齐：待绑定账号在该园有档案（含真实姓名）时，姓名一并带过来。
func TestIdentityMerge_EnsuresTargetParentProfile(t *testing.T) {
	f := newMergeFixture(t)
	execSQL(t, f.DB, "INSERT INTO parent_profiles (user_id, kindergarten_id, real_name, status) VALUES (?,?,?,1)",
		f.Pending.ID, f.KG.ID, "王妈妈")

	_, err := f.mergeWith(t, service.IdentityMergeRequest{ParentProfileKindergartenID: f.KG.ID})
	require.NoError(t, err)

	var profile domain.ParentProfile
	require.NoError(t, f.DB.Where("user_id = ? AND kindergarten_id = ?", f.Target.ID, f.KG.ID).
		First(&profile).Error, "预创建账号必须获得家长档案")
	assert.Equal(t, "王妈妈", profile.RealName)
	assert.Equal(t, int8(1), profile.Status)
}

// 预创建账号已有活跃家长档案时不覆盖。
func TestIdentityMerge_KeepsExistingTargetParentProfile(t *testing.T) {
	f := newMergeFixture(t)
	execSQL(t, f.DB, "INSERT INTO parent_profiles (user_id, kindergarten_id, real_name, status) VALUES (?,?,?,1)",
		f.Target.ID, f.KG.ID, "李妈妈")
	execSQL(t, f.DB, "INSERT INTO parent_profiles (user_id, kindergarten_id, real_name, status) VALUES (?,?,?,1)",
		f.Pending.ID, f.KG.ID, "王妈妈")

	_, err := f.mergeWith(t, service.IdentityMergeRequest{ParentProfileKindergartenID: f.KG.ID})
	require.NoError(t, err)

	var profile domain.ParentProfile
	require.NoError(t, f.DB.Where("user_id = ? AND kindergarten_id = ?", f.Target.ID, f.KG.ID).
		First(&profile).Error)
	assert.Equal(t, "李妈妈", profile.RealName, "已有家长档案不得被覆盖")
}

// 软删的家长档案占着 (user_id, kindergarten_id) 唯一键，合并时复活而不是重建。
func TestIdentityMerge_RevivesSoftDeletedTargetParentProfile(t *testing.T) {
	f := newMergeFixture(t)
	execSQL(t, f.DB, `INSERT INTO parent_profiles (user_id, kindergarten_id, real_name, status, deleted_at)
		VALUES (?,?,?,1,NOW())`, f.Target.ID, f.KG.ID, "旧档案")

	_, err := f.mergeWith(t, service.IdentityMergeRequest{ParentProfileKindergartenID: f.KG.ID})
	require.NoError(t, err)

	var profile domain.ParentProfile
	require.NoError(t, f.DB.Where("user_id = ? AND kindergarten_id = ?", f.Target.ID, f.KG.ID).
		First(&profile).Error, "软删档案必须被复活")
	assert.Equal(t, "旧档案", profile.RealName)
}

// 全表扫描：合并后不得有任何列继续引用待绑定账号。
// 列名集合与 schema 对齐（operation_logs 是审计流水，刻意保留原 user_id）。
func TestIdentityMerge_LeavesNoOrphanReferences(t *testing.T) {
	f := newMergeFixture(t)
	seedAllReferenceTypes(t, f)

	_, err := f.merge(t)
	require.NoError(t, err)

	type refColumn struct {
		TableName  string `gorm:"column:TABLE_NAME"`
		ColumnName string `gorm:"column:COLUMN_NAME"`
	}
	var cols []refColumn
	require.NoError(t, f.DB.Raw(`
		SELECT TABLE_NAME, COLUMN_NAME
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		  AND DATA_TYPE IN ('bigint','int','mediumint','smallint','tinyint')
		  AND COLUMN_NAME IN ('user_id','creator_id','applicant_id','reviewer_id','operator_id',
			'uploader_id','recorder_id','teacher_id','published_by','created_by','updated_by',
			'public_approved_by','submitter_id')
		  AND TABLE_NAME <> 'operation_logs'
		ORDER BY TABLE_NAME, COLUMN_NAME`).Scan(&cols).Error)
	require.NotEmpty(t, cols)
	for _, c := range cols {
		q := fmt.Sprintf("SELECT COUNT(*) FROM `%s` WHERE `%s` = ?", c.TableName, c.ColumnName)
		assert.Zero(t, countSQL(t, f.DB, q, f.Pending.ID),
			"%s.%s 仍引用已合并账号", c.TableName, c.ColumnName)
	}

	// 迁移类数据必须仍然存在，只是换了所有者。
	var msgCount int64
	require.NoError(t, f.DB.Model(&domain.Message{}).Where("user_id = ?", f.Target.ID).Count(&msgCount).Error)
	assert.EqualValues(t, 1, msgCount, "消息应迁移到预创建账号，而不是被清理")
	var noticeCount int64
	require.NoError(t, f.DB.Model(&domain.Notice{}).Where("creator_id = ?", f.Target.ID).Count(&noticeCount).Error)
	assert.EqualValues(t, 1, noticeCount, "通知应迁移到预创建账号")
}

// seedAllReferenceTypes 在每一类引用列上各造一行待绑定账号的数据：
// 有外键的、无外键的、迁移的、去重的、丢弃的。任何一类漏处理都会在合并时
// 报外键错误或在全表扫描里暴露。
func seedAllReferenceTypes(t *testing.T, f *mergeFixture) {
	t.Helper()
	kg, class, student := f.KG.ID, f.Class.ID, f.Student.ID
	uid := f.Pending.ID

	execSQL(t, f.DB, "INSERT INTO user_consents (user_id, policy_version, source, ip, user_agent, consented_at) VALUES (?,?,?,?,?,NOW())",
		uid, "orphan-v1", "parent", "", "")
	execSQL(t, f.DB, "INSERT INTO messages (user_id, type, title) VALUES (?,?,?)", uid, "system", "合并测试消息")
	execSQL(t, f.DB, "INSERT INTO wechat_push_logs (user_id, template_id, content, status) VALUES (?,?,?,0)",
		uid, "tpl", "{}")
	execSQL(t, f.DB, "INSERT INTO files (user_id, biz_type, file_url) VALUES (?,?,?)", uid, "growth", "oss://f")
	execSQL(t, f.DB, "INSERT INTO file_records (kindergarten_id, uploader_id, file_name, file_url) VALUES (?,?,?,?)",
		kg, uid, "a.png", "oss://a")
	execSQL(t, f.DB, "INSERT INTO feedbacks (kindergarten_id, user_id, category, content) VALUES (?,?,?,?)",
		kg, uid, "suggestion", "测试反馈")
	execSQL(t, f.DB, "INSERT INTO growth_comments (log_id, user_id, content) VALUES (?,?,?)",
		9701, uid, "测试评论")
	execSQL(t, f.DB, "INSERT INTO growth_media (kindergarten_id, class_id, uploader_id, file_url, file_type) VALUES (?,?,?,?,?)",
		kg, class, uid, "oss://m", "image")
	execSQL(t, f.DB, "INSERT INTO growth_likes (log_id, user_id) VALUES (?,?)", 9702, uid)
	execSQL(t, f.DB, "INSERT INTO growth_comment_likes (comment_id, user_id) VALUES (?,?)", 9703, uid)
	execSQL(t, f.DB, "INSERT INTO attendances (kindergarten_id, class_id, student_id, date, status, recorder_id) VALUES (?,?,?,?,?,?)",
		kg, class, student, "2026-01-01", 1, uid)
	execSQL(t, f.DB, "INSERT INTO pickup_records (kindergarten_id, class_id, student_id, date, type, method, operator_id) VALUES (?,?,?,?,1,1,?)",
		kg, class, student, "2026-01-01", uid)
	var pickupRecordID uint64
	require.NoError(t, f.DB.Raw("SELECT id FROM pickup_records WHERE operator_id = ? ORDER BY id DESC LIMIT 1", uid).
		Scan(&pickupRecordID).Error)
	require.NotZero(t, pickupRecordID)
	execSQL(t, f.DB, "INSERT INTO morning_health_checks (kindergarten_id, class_id, student_id, pickup_record_id, date, operator_id) VALUES (?,?,?,?,?,?)",
		kg, class, student, pickupRecordID, "2026-01-01", uid)
	execSQL(t, f.DB, "INSERT INTO pickup_authorizations (student_id, name, phone, relation, user_id, creator_id) VALUES (?,?,?,?,?,?)",
		student, "王阿姨", "13800000000", "其他", uid, uid)
	execSQL(t, f.DB, "INSERT INTO leave_requests (kindergarten_id, class_id, student_id, applicant_id, reason, start_date, end_date, reviewer_id) VALUES (?,?,?,?,?,?,?,?)",
		kg, class, student, uid, "病假", "2026-01-02", "2026-01-03", uid)
	execSQL(t, f.DB, "INSERT INTO announcements (kindergarten_id, creator_id, title) VALUES (?,?,?)",
		kg, uid, "测试公告")
	execSQL(t, f.DB, "INSERT INTO calendar_events (kindergarten_id, class_id, creator_id, title, event_type, event_date) VALUES (?,?,?,?,?,?)",
		kg, class, uid, "测试日程", "activity", "2026-01-04")
	execSQL(t, f.DB, "INSERT INTO growth_daily_logs (kindergarten_id, class_id, creator_id, date, content) VALUES (?,?,?,?,?)",
		kg, class, uid, "2026-01-05", "测试成长日志")
	execSQL(t, f.DB, "INSERT INTO notices (kindergarten_id, class_id, creator_id, category, title) VALUES (?,?,?,?,?)",
		kg, class, uid, "daily", "测试通知")
	execSQL(t, f.DB, "INSERT INTO notice_templates (kindergarten_id, creator_id, title) VALUES (?,?,?)",
		kg, uid, "测试模板")
	execSQL(t, f.DB, "INSERT INTO user_consents (user_id, policy_version, source, ip, user_agent, consented_at) VALUES (?,?,?,?,?,NOW())",
		uid, "orphan-v2", "parent", "", "")
	execSQL(t, f.DB, "INSERT INTO parent_settings (user_id, kindergarten_id, notice_push, leave_push, pickup_push, growth_push, task_push, announcement_push, show_phone_to_teacher, allow_growth_share, allow_class_share, allow_external_share) VALUES (?,?,1,1,1,1,1,1,1,1,1,1)",
		uid, kg)
	execSQL(t, f.DB, "INSERT INTO notice_read_logs (notice_id, user_id, read_at) VALUES (?,?,NOW())", 9801, uid)
	execSQL(t, f.DB, "INSERT INTO teacher_message_reads (user_id, message_type, reference_id) VALUES (?,?,?)",
		uid, "notice", 9802)
	execSQL(t, f.DB, "INSERT INTO parent_tasks (kindergarten_id, class_id, creator_id, title, frequency, start_date, end_date) VALUES (?,?,?,?,?,?,?)",
		kg, class, uid, "测试任务", "daily", "2026-01-01", "2026-01-31")
	var taskID uint64
	require.NoError(t, f.DB.Raw("SELECT id FROM parent_tasks WHERE creator_id = ? ORDER BY id DESC LIMIT 1", uid).
		Scan(&taskID).Error)
	require.NotZero(t, taskID)
	execSQL(t, f.DB, "INSERT INTO task_check_ins (task_id, student_id, user_id, date, public_approved_by) VALUES (?,?,?,?,?)",
		taskID, student, uid, "2026-01-06", uid)
	execSQL(t, f.DB, "INSERT INTO task_read_marks (kindergarten_id, task_id, user_id, student_id) VALUES (?,?,?,?)",
		kg, taskID, uid, student)
	execSQL(t, f.DB, "INSERT INTO task_responses (kindergarten_id, task_id, teacher_id, ritual, response_type) VALUES (?,?,?,?,?)",
		kg, taskID, uid, "star", "text")
	execSQL(t, f.DB, "INSERT INTO task_response_reactions (kindergarten_id, response_id, user_id, reaction) VALUES (?,?,?,?)",
		kg, 9901, uid, "like")
	execSQL(t, f.DB, "INSERT INTO albums (kindergarten_id, class_id, creator_id, title, category, album_date, published_by) VALUES (?,?,?,?,?,?,?)",
		kg, class, uid, "测试相册", "daily", "2026-01-07", uid)
	execSQL(t, f.DB, "INSERT INTO class_star_jars (kindergarten_id, class_id, term, created_by, updated_by) VALUES (?,?,?,?,?)",
		kg, class, "2026-orphan", uid, uid)
	execSQL(t, f.DB, "INSERT INTO content_audits (kindergarten_id, target_type, target_id, submitter_id, reviewer_id) VALUES (?,?,?,?,?)",
		kg, "notice", 9803, uid, uid)
	execSQL(t, f.DB, "INSERT INTO teacher_invitations (token, user_id, kindergarten_id) VALUES (?,?,?)",
		"invite-"+f.Suffix, uid, kg)
	execSQL(t, f.DB, "INSERT INTO user_roles (user_id, role, kindergarten_id) VALUES (?,?,?)", uid, "parent", kg)
	execSQL(t, f.DB, "INSERT INTO user_admin_roles (user_id, admin_role_id, kindergarten_id) VALUES (?,?,?)", uid, 1, kg)
	execSQL(t, f.DB, "INSERT INTO parent_profiles (user_id, kindergarten_id, real_name, status) VALUES (?,?,?,1)",
		uid, kg, "待绑定家长")
	execSQL(t, f.DB, "INSERT INTO teacher_profiles (user_id, kindergarten_id, real_name) VALUES (?,?,?)",
		uid, kg, "待绑定老师")
	execSQL(t, f.DB, "INSERT INTO student_parents (student_id, user_id, relation) VALUES (?,?,?)",
		student, uid, "爸爸")
	execSQL(t, f.DB, "INSERT INTO class_teachers (class_id, user_id, title) VALUES (?,?,?)",
		class, uid, "配班")
}

// 合并是单事务：中途失败必须整体回滚，身份/同意/消息都留在待绑定账号上。
func TestIdentityMerge_RollsBackWhenLaterStepFails(t *testing.T) {
	f := newMergeFixture(t)
	execSQL(t, f.DB, "INSERT INTO user_consents (user_id, policy_version, source, ip, user_agent, consented_at) VALUES (?,?,?,?,?,NOW())",
		f.Pending.ID, "rollback-v1", "parent", "", "")
	execSQL(t, f.DB, "INSERT INTO messages (user_id, type, title) VALUES (?,?,?)",
		f.Pending.ID, "system", "回滚测试消息")

	const cbName = "test:fail_teacher_invitation_move"
	require.NoError(t, f.DB.Callback().Update().Before("gorm:update").Register(cbName, func(tx *gorm.DB) {
		// teacher_invitations 位于迁移列表末尾，注入失败可以证明此前的写入也回滚了。
		if tx.Statement.Table == "teacher_invitations" {
			_ = tx.AddError(errors.New("injected merge failure"))
		}
	}))
	t.Cleanup(func() { _ = f.DB.Callback().Update().Remove(cbName) })

	_, err := f.merge(t)
	require.Error(t, err)

	identity := f.identity(t)
	assert.Equal(t, f.Pending.ID, identity.UserID, "失败后身份必须仍在待绑定账号上")
	assert.True(t, f.userExists(t, f.Pending.ID), "失败后待绑定账号不得被删除")
	assert.Equal(t, int64(1), countSQL(t, f.DB,
		"SELECT COUNT(*) FROM user_consents WHERE user_id = ?", f.Pending.ID), "同意记录必须回滚")
	assert.Equal(t, int64(1), countSQL(t, f.DB,
		"SELECT COUNT(*) FROM messages WHERE user_id = ?", f.Pending.ID), "消息必须回滚")
}
