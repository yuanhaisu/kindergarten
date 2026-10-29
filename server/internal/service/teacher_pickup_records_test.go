package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
)

// 老师端接送记录按日期过滤必须与写入侧同一墙钟（Asia/Shanghai）。
//
// 回归点：PickupRecords 曾用裸 time.Parse 解析 date（UTC 零点）。DSN 的 loc=Local
// 会让驱动在绑定参数前执行 t.In(Local)，UTC 零点因此变成 +08 08:00；DATE 列只有
// 零点，比较恒不等 → 带 date 的查询恒 0 条。这里以本地零点写 2 条当日 + 1 条昨日
// 记录，锁住「今天 / 昨天 / 不带 date / 非法 date」四种行为。
//
// 真 DB 跑，复用 teacher_pickup_create_test.go 的 fixture 与 cleanupPickupData。
func TestPickupRecords_FiltersByLocalDate(t *testing.T) {
	f := newPickupFixture(t, "00:00", testPickupClass)

	// 一次取样贯穿建数据与查询，避免恰好跨零点时前后「今天」不一致。
	today := teacherPortalDay(time.Now())
	yesterday := today.AddDate(0, 0, -1)

	todayIDs := []uint64{
		createPickupRecordOn(t, f, today, today.Add(8*time.Hour), domain.PickupTypeDropOff),
		createPickupRecordOn(t, f, today, today.Add(16*time.Hour), domain.PickupTypePickUp),
	}
	yesterdayID := createPickupRecordOn(t, f, yesterday, yesterday.Add(9*time.Hour), domain.PickupTypeDropOff)

	// 边界诊断：DATE 列必须落在与查询参数同一个本地日期上。
	var rawDate string
	require.NoError(t, f.svc.db.Raw(
		"SELECT DATE_FORMAT(date, '%Y-%m-%d') FROM pickup_records WHERE id = ?", todayIDs[0],
	).Scan(&rawDate).Error)
	assert.Equal(t, today.Format("2006-01-02"), rawDate, "DATE 列应为本地日期")
	t.Logf("边界: DB DATE=%s local-today=%s yesterday=%s", rawDate,
		today.Format("2006-01-02"), yesterday.Format("2006-01-02"))

	idsOf := func(resp map[string]any) []uint64 {
		t.Helper()
		rows, ok := resp["list"].([]domain.PickupRecord)
		require.True(t, ok, "list 应为 []domain.PickupRecord")
		ids := make([]uint64, 0, len(rows))
		for _, r := range rows {
			ids = append(ids, r.ID)
		}
		return ids
	}
	// brief 要求：每个用例都断言 total 与 list 条数一致。
	assertTotalMatchesList := func(resp map[string]any) {
		t.Helper()
		assert.Equal(t, int64(len(idsOf(resp))), asI64(resp["total"]),
			"total 必须与 list 条数一致")
	}

	// 今天：只命中当日 2 条（修复前为 0）。
	resp, err := f.svc.PickupRecords(ctx(), f.tc, 1, 20, today.Format("2006-01-02"), nil)
	require.NoError(t, err)
	assertTotalMatchesList(resp)
	assert.Equal(t, int64(2), asI64(resp["total"]), "date=今天 必须返回当日 2 条")
	assert.ElementsMatch(t, todayIDs, idsOf(resp), "date=今天 不得混入昨日记录")
	assert.Equal(t, 1, resp["page"], "分页语义不变：page 回显")
	assert.Equal(t, 20, resp["page_size"], "分页语义不变：page_size 回显")

	// 昨天：只命中昨日 1 条。
	resp, err = f.svc.PickupRecords(ctx(), f.tc, 1, 20, yesterday.Format("2006-01-02"), nil)
	require.NoError(t, err)
	assertTotalMatchesList(resp)
	assert.Equal(t, int64(1), asI64(resp["total"]), "date=昨天 必须返回 1 条")
	assert.Equal(t, []uint64{yesterdayID}, idsOf(resp), "date=昨天 只能是昨日那条")

	// 不带 date：全部 3 条，顺序按 recorded_at DESC（今天的 16:00 在最前）。
	resp, err = f.svc.PickupRecords(ctx(), f.tc, 1, 20, "", nil)
	require.NoError(t, err)
	assertTotalMatchesList(resp)
	assert.Equal(t, int64(3), asI64(resp["total"]), "不带 date 必须返回全部 3 条")
	assert.ElementsMatch(t, append(append([]uint64{}, todayIDs...), yesterdayID), idsOf(resp),
		"不带 date 不得过滤任一记录")

	// 非法 date：沿用现状——解析失败即忽略 date，不报错、不过滤（本切片不改语义）。
	resp, err = f.svc.PickupRecords(ctx(), f.tc, 1, 20, "2026-13-40", nil)
	require.NoError(t, err, "非法 date 现状不返回错误")
	assertTotalMatchesList(resp)
	assert.Equal(t, int64(3), asI64(resp["total"]), "非法 date 现状为不过滤")

	// 清理证据：与 fixture 同一个 cleanupPickupData，硬删后本班接送记录 0 残留
	//（用 Unscoped 统计，软删行也不会漏网）。
	cleanupPickupData(t, f.svc.db, f.classID)
	var residue int64
	require.NoError(t, f.svc.db.Unscoped().Model(&domain.PickupRecord{}).
		Where("student_id = ?", f.studentID).Count(&residue).Error)
	t.Logf("cleanup residue pickup_records=%d", residue)
	assert.Zero(t, residue, "cleanupPickupData 后 pickup_records 不应残留")
}

// createPickupRecordOn 直接落一条接送记录：Date 用本地零点，RecordedAt 用本地时间。
// 不经 CreatePickupRecord，避免交替序列/理由必填等业务规则干扰日期过滤本身。
func createPickupRecordOn(t *testing.T, f *pickupFixture, date, recordedAt time.Time, typ domain.PickupType) uint64 {
	t.Helper()
	row := &domain.PickupRecord{
		KindergartenID: testPickupKG,
		ClassID:        f.classID,
		StudentID:      f.studentID,
		Date:           date,
		Type:           typ,
		Method:         domain.PickupMethodManual,
		OperatorID:     testPickupOperatorID,
		RecordedAt:     recordedAt,
	}
	require.NoError(t, f.svc.db.WithContext(ctx()).Create(row).Error)
	return row.ID
}
