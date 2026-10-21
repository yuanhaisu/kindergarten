package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
)

// 这些测试打真实 MySQL（本地 configs/app.ini），覆盖 notice-multitarget 的分母口径
// 与目标集命中判定：多孩家长分母只计 1、两班目标的命中/未命中、迁移等价（单目标=原单班）。
// 对应 spec: openspec/changes/notice-multitarget/specs/notice-targeting/spec.md
// 用独立测试园所/班级（高 ID），每例 t.Cleanup 硬删自建行，不污染真实数据。

const (
	testNoticeKG     uint64 = 990101
	testNoticeCreator uint64 = 990101
)

var noticeTestDB *gorm.DB

func openNoticeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	if noticeTestDB != nil {
		return noticeTestDB
	}
	cfgPath := filepath.Join("..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skipf("跳过：找不到 %s（本地无 DB 配置）", cfgPath)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.MySQL.DSN == "" {
		t.Skip("跳过：MySQL DSN 为空")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// 等价于部署步骤 1（AutoMigrate 建新表）；存量表已存在则跳过。
	if err := db.AutoMigrate(&domain.NoticeTarget{}); err != nil {
		t.Fatalf("auto migrate notice_targets: %v", err)
	}
	noticeTestDB = db
	return db
}

// noticeFixture 测试所需的班级/家长/学生骨架。ID 全部显式指定，便于 cleanup。
type noticeFixture struct {
	db  *gorm.DB
	repo *NoticeRepo
	// classIDs[0..1] 为两个目标班，classIDs[2] 为无关班
	classIDs [3]uint64
}

func newNoticeFixture(t *testing.T) *noticeFixture {
	t.Helper()
	db := openNoticeTestDB(t)
	ctx := context.Background()
	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)

	cleanupNoticeData(t, db)

	var kgCount int64
	db.WithContext(ctx).Model(&domain.Kindergarten{}).Where("id = ?", testNoticeKG).Count(&kgCount)
	if kgCount == 0 {
		require.NoError(t, db.WithContext(ctx).Create(&domain.Kindergarten{
			BaseModel: domain.BaseModel{ID: testNoticeKG},
			Name:      "测试园所-通知目标",
		}).Error)
	}
	var userCount int64
	db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", testNoticeCreator).Count(&userCount)
	if userCount == 0 {
		require.NoError(t, db.WithContext(ctx).Create(&domain.User{
			BaseModel: domain.BaseModel{ID: testNoticeCreator},
			Phone:     "13900000101",
			Nickname:  "测试通知老师",
		}).Error)
	}

	f := &noticeFixture{db: db, repo: NewNoticeRepo(db)}
	for i := range f.classIDs {
		classID := uint64(990102 + i) // 990102 / 990103 / 990104
		f.classIDs[i] = classID
		require.NoError(t, db.WithContext(ctx).Create(&domain.Class{
			BaseModel:      domain.BaseModel{ID: classID},
			KindergartenID: testNoticeKG,
			Name:           fmt.Sprintf("测试%d班", classID),
			Status:         1,
		}).Error)
	}

	t.Cleanup(func() { cleanupNoticeData(t, db) })
	_ = validDate
	return f
}

// cleanupNoticeData 硬删本测试的全部自建行（含软删行）。
// 子表先删：student_parents 引用 students。
func cleanupNoticeData(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	del := func(sql string, args ...any) {
		if err := db.WithContext(ctx).Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup: %v", err)
		}
	}
	del("DELETE FROM notice_read_logs WHERE user_id >= 990100 AND user_id < 990200")
	del("DELETE FROM notice_targets WHERE kindergarten_id = ?", testNoticeKG)
	del("DELETE FROM notices WHERE kindergarten_id = ?", testNoticeKG)
	del("DELETE FROM student_parents WHERE user_id >= 990100 AND user_id < 990200")
	del("DELETE FROM students WHERE kindergarten_id = ?", testNoticeKG)
	del("DELETE FROM classes WHERE kindergarten_id = ?", testNoticeKG)
	// 园所与创建者是跨 fixture 共享的固定测试 ID，不删。
}

// addParentWithKids 建一个家长 + N 个孩子（每个孩子一个班），返回家长 UserID。
// 家长 ID 从 parentSeq 递增，落在 990100+ 区间供 cleanup。
func (f *noticeFixture) addParentWithKids(t *testing.T, parentSeq uint64, classIDs ...uint64) uint64 {
	t.Helper()
	ctx := context.Background()
	validDate := time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC)
	userID := 990100 + parentSeq
	var userCount int64
	f.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Count(&userCount)
	if userCount == 0 {
		require.NoError(t, f.db.WithContext(ctx).Create(&domain.User{
			BaseModel: domain.BaseModel{ID: userID},
			Phone:     fmt.Sprintf("1390000%04d", parentSeq),
			Nickname:  fmt.Sprintf("测试家长%d", parentSeq),
		}).Error)
	}
	for i, classID := range classIDs {
		studentID := 990100 + parentSeq*10 + uint64(i)
		require.NoError(t, f.db.WithContext(ctx).Create(&domain.Student{
			SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: studentID}},
			KindergartenID:  testNoticeKG,
			ClassID:         classID,
			Name:            fmt.Sprintf("测试学生%d", studentID),
			Birthday:        validDate,
			EnrollDate:      validDate,
			Status:          1,
		}).Error)
		require.NoError(t, f.db.WithContext(ctx).Create(&domain.StudentParent{
			StudentID: studentID,
			UserID:    userID,
			Relation:  "mother",
		}).Error)
	}
	return userID
}

// makeNotice 建一条通知并落目标集。classID 仅为满足旧列 not null（迁移期写首个目标班）。
func (f *noticeFixture) makeNotice(t *testing.T, noticeID uint64, targetClassIDs ...uint64) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, f.db.WithContext(ctx).Create(&domain.Notice{
		SoftDeleteModel: domain.SoftDeleteModel{BaseModel: domain.BaseModel{ID: noticeID}},
		KindergartenID:  testNoticeKG,
		ClassID:         targetClassIDs[0],
		CreatorID:       testNoticeCreator,
		Category:        domain.NoticeCategoryOther,
		Title:           fmt.Sprintf("测试通知%d", noticeID),
		Content:         "正文",
		Status:          domain.NoticeStatusPublished,
	}).Error)
	require.NoError(t, f.repo.ReplaceNoticeTargets(ctx, testNoticeKG, noticeID, targetClassIDs))
}

func (f *noticeFixture) markRead(t *testing.T, noticeID, userID uint64) {
	t.Helper()
	require.NoError(t, f.repo.RecordReadLog(context.Background(), noticeID, userID))
}

// TC-NEW-NTC-0011：双孩家长在分母中只计一人（目标为两班，家长两孩分属两班）。
func TestNoticeReadStats_MultiChildParentCountedOnce(t *testing.T) {
	f := newNoticeFixture(t)
	ctx := context.Background()

	multiChild := f.addParentWithKids(t, 1, f.classIDs[0], f.classIDs[1])
	for seq := uint64(2); seq <= 9; seq++ {
		// 另外 8 位家长各一个孩子，交替落在两个目标班
		f.addParentWithKids(t, seq, f.classIDs[(seq-2)%2])
	}
	f.makeNotice(t, 990101, f.classIDs[0], f.classIDs[1])

	readUsers, _, parentTotal, err := f.repo.NoticeReadStats(ctx, testNoticeKG, 990101)
	require.NoError(t, err)
	assert.Equal(t, int64(9), parentTotal, "分母应为去重家长数 9（双孩家长只计 1）")
	assert.Equal(t, int64(0), readUsers)

	f.markRead(t, 990101, multiChild)
	readUsers, _, parentTotal, err = f.repo.NoticeReadStats(ctx, testNoticeKG, 990101)
	require.NoError(t, err)
	assert.Equal(t, int64(9), parentTotal)
	assert.Equal(t, int64(1), readUsers, "双孩家长读过之后分子计 1")
}

// TC-NEW-NTC-0006/0007/0008：两班目标的命中/未命中——命中任一目标班即可见。
func TestNoticeTargetHit_TwoClassHitAndMiss(t *testing.T) {
	f := newNoticeFixture(t)
	ctx := context.Background()

	hitBoth := f.addParentWithKids(t, 1, f.classIDs[0], f.classIDs[1]) // 两孩都在目标班
	hitOne := f.addParentWithKids(t, 2, f.classIDs[0])                 // 仅一孩命中
	miss := f.addParentWithKids(t, 3, f.classIDs[2])                   // 孩子在无关班
	f.makeNotice(t, 990101, f.classIDs[0], f.classIDs[1])

	// EXISTS 命中：家长侧按「孩子班级集」判定，一条通知只出一行（多孩合并）
	parentClasses := func(userID uint64) []uint64 {
		var ids []uint64
		require.NoError(t, f.db.WithContext(ctx).Table("student_parents AS sp").
			Joins("INNER JOIN students s ON s.id = sp.student_id").
			Where("sp.user_id = ?", userID).Pluck("s.class_id", &ids).Error)
		return ids
	}
	for name, userID := range map[string]uint64{"双孩家长": hitBoth, "单孩命中家长": hitOne} {
		hit, err := f.repo.ExistsNoticeTargetHit(ctx, 990101, parentClasses(userID))
		require.NoError(t, err)
		assert.True(t, hit, "%s应可见", name)
	}
	hit, err := f.repo.ExistsNoticeTargetHit(ctx, 990101, parentClasses(miss))
	require.NoError(t, err)
	assert.False(t, hit, "无关班家长不可见")

	// 空班级集永远不可见（防御分支）
	hit, err = f.repo.ExistsNoticeTargetHit(ctx, 990101, nil)
	require.NoError(t, err)
	assert.False(t, hit)

	// 列表 EXISTS：命中通知只出一行
	q := f.repo.FilterByTargetHit(f.db.WithContext(ctx).Table("notices AS notices").Where("id = ?", 990101), parentClasses(hitBoth))
	var n int64
	require.NoError(t, q.Count(&n).Error)
	assert.Equal(t, int64(1), n, "命中通知应只出一行（多孩合并）")
}

// TC-NEW-NTC-0013：迁移等价——单目标通知的分母与迁移前单班口径（COUNT(DISTINCT sp.user_id) WHERE class_id=X）一致。
func TestNoticeReadStats_MigrationEquivalenceSingleTarget(t *testing.T) {
	f := newNoticeFixture(t)
	ctx := context.Background()

	// 5 位家长各一个孩子，全在单目标班
	for seq := uint64(1); seq <= 5; seq++ {
		f.addParentWithKids(t, seq, f.classIDs[0])
	}
	// 无关班的家长不得进分母
	f.addParentWithKids(t, 6, f.classIDs[2])
	f.makeNotice(t, 990101, f.classIDs[0])

	for seq := uint64(1); seq <= 3; seq++ {
		f.markRead(t, 990101, 990100+seq)
	}

	readUsers, _, parentTotal, err := f.repo.NoticeReadStats(ctx, testNoticeKG, 990101)
	require.NoError(t, err)

	// 迁移前的单班口径（旧 SQL 原样对照）
	var legacyTotal int64
	require.NoError(t, f.db.WithContext(ctx).Raw(`
SELECT COALESCE(COUNT(DISTINCT sp.user_id), 0)
FROM student_parents sp
INNER JOIN students s ON s.id = sp.student_id AND s.deleted_at IS NULL
WHERE s.class_id = ? AND s.kindergarten_id = ?`,
		f.classIDs[0], testNoticeKG).Scan(&legacyTotal).Error)

	assert.Equal(t, legacyTotal, parentTotal, "单目标分母应与迁移前单班口径一致")
	assert.Equal(t, int64(5), parentTotal)
	assert.Equal(t, int64(3), readUsers, "迁移后已读数 3/5 不变")
}

// ReplaceNoticeTargets：整体替换 + 内部去重 + 空集拒绝。
func TestReplaceNoticeTargets_ReplaceDedupeRejectEmpty(t *testing.T) {
	f := newNoticeFixture(t)
	ctx := context.Background()
	f.makeNotice(t, 990101, f.classIDs[0])

	// 重选目标（含重复项）→ 并集去重
	require.NoError(t, f.repo.ReplaceNoticeTargets(ctx, testNoticeKG, 990101,
		[]uint64{f.classIDs[1], f.classIDs[2], f.classIDs[1]}))
	ids, err := f.repo.ListNoticeTargetClassIDs(ctx, 990101)
	require.NoError(t, err)
	assert.Equal(t, []uint64{f.classIDs[1], f.classIDs[2]}, ids, "重选为整体替换且去重")

	// 空集拒绝
	err = f.repo.ReplaceNoticeTargets(ctx, testNoticeKG, 990101, nil)
	assert.Error(t, err, "目标集合必须非空")
}
