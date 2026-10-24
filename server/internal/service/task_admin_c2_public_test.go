package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/domain"
	"kindergarten-server/pkg/pagination"
)

// 7.3 管理端查看公开内容：只读、复用公开墙同一套判定。

func (f *c2Fixture) createTaskWith(t *testing.T, classID uint64, title string, mut func(*CreateTaskReq)) uint64 {
	t.Helper()
	req := CreateTaskReq{
		Title:        title,
		LearningLink: "配合本周主题",
		ClassID:      classID,
		Frequency:    string(domain.TaskFrequencyDaily),
		StartDate:    f.date(0),
		EndDate:      f.date(6),
	}
	if mut != nil {
		mut(&req)
	}
	out, err := f.svc.CreateTask(f.ctx, testC2KG, testC2Admin, req, "", "")
	require.NoError(t, err)
	return out["id"].(uint64)
}

func (f *c2Fixture) addPublicCheckin(t *testing.T, taskID uint64, approved bool) domain.TaskCheckIn {
	t.Helper()
	now := time.Now()
	c := domain.TaskCheckIn{
		TaskID: taskID, StudentID: testC2Student, UserID: testC2Parent, Date: f.today,
		Content: "今天读了绘本", IsPublicRequested: true, PublicScopeConfirmed: approved,
	}
	if approved {
		c.PublicApprovedAt = &now
		c.PublicApprovedBy = testC2Admin
	}
	require.NoError(t, f.db.Create(&c).Error)
	return c
}

func (f *c2Fixture) publicCheckins(t *testing.T, taskID uint64, status string) map[string]any {
	t.Helper()
	out, err := f.svc.ListPublicCheckins(f.ctx, testC2KG, taskID, status,
		pagination.Query{Page: 1, PageSize: 20})
	require.NoError(t, err)
	return out
}

func TestAdminListPublicCheckinsShowsApproved(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskWith(t, testC2Class, "C2公开任务", func(r *CreateTaskReq) { r.AllowPublic = true })
	c := f.addPublicCheckin(t, taskID, true)

	out := f.publicCheckins(t, taskID, "")
	assert.Equal(t, int64(1), out["total"])
	list := out["list"].([]map[string]any)
	require.Len(t, list, 1)
	assert.Equal(t, c.ID, list[0]["checkin_id"])
	assert.Equal(t, "approved", list[0]["public_status"])
	assert.True(t, list[0]["public_visible"].(bool), "任务允许公开、孩子仍在任务班级 → 墙上可见")
	assert.Equal(t, testC2Student, list[0]["student_id"])
	assert.Equal(t, "C2孩子A", list[0]["student_name"])
	assert.Equal(t, "今天读了绘本", list[0]["content"])
}

func TestAdminListPublicCheckinsFiltersByStatus(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskWith(t, testC2Class, "C2公开筛选任务", func(r *CreateTaskReq) { r.AllowPublic = true })
	approved := f.addPublicCheckin(t, taskID, true)
	// 从未申请的提交：不属于公开列表。
	private := domain.TaskCheckIn{
		TaskID: taskID, StudentID: testC2StudentC, UserID: testC2Parent,
		Date: f.today.AddDate(0, 0, -1), Content: "私密内容",
	}
	require.NoError(t, f.db.Create(&private).Error)
	// 已撤回的提交。
	now := time.Now()
	revoked := domain.TaskCheckIn{
		TaskID: taskID, StudentID: testC2Student, UserID: testC2Parent, Date: f.today.AddDate(0, 0, -2),
		IsPublicRequested: false, PublicRevokedAt: &now,
	}
	require.NoError(t, f.db.Create(&revoked).Error)

	pub := f.publicCheckins(t, taskID, "public")
	assert.Equal(t, int64(1), pub["total"])
	assert.Equal(t, approved.ID, pub["list"].([]map[string]any)[0]["checkin_id"])

	rev := f.publicCheckins(t, taskID, "revoked")
	assert.Equal(t, int64(1), rev["total"])
	assert.Equal(t, revoked.ID, rev["list"].([]map[string]any)[0]["checkin_id"])

	all := f.publicCheckins(t, taskID, "all")
	assert.Equal(t, int64(3), all["total"], "status=all 不按公开状态过滤")
}

func TestAdminListPublicCheckinsMarksInvisibleWhenTaskClosed(t *testing.T) {
	f := setupC2(t)
	// 任务创建时不允许公开，但库里留有一条已批准的记录（例如公开被关闭后）。
	// public_status 仍是 approved（状态真实），public_visible 必须是 false（墙上没有）。
	taskID := f.createTaskWith(t, testC2Class, "C2关闭公开任务", nil)
	f.addPublicCheckin(t, taskID, true)

	out := f.publicCheckins(t, taskID, "public")
	require.Equal(t, int64(1), out["total"])
	item := out["list"].([]map[string]any)[0]
	assert.Equal(t, "approved", item["public_status"])
	assert.False(t, item["public_visible"].(bool), "任务 allow_public=false 时墙上不可见")
}

func TestAdminListPublicCheckinsRejectsBadStatus(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskWith(t, testC2Class, "C2状态校验任务", nil)
	_, err := f.svc.ListPublicCheckins(f.ctx, testC2KG, taskID, "pending", pagination.Query{})
	require.EqualError(t, err, "BAD_PUBLIC_STATUS")
}

func TestAdminListPublicCheckinsForeignTaskNotFound(t *testing.T) {
	f := setupC2(t)
	_, err := f.svc.ListPublicCheckins(f.ctx, testC2KG, 999999, "public", pagination.Query{})
	require.ErrorIs(t, err, errTaskNotFound)
}

func TestAdminListPublicCheckinsPaginates(t *testing.T) {
	f := setupC2(t)
	taskID := f.createTaskWith(t, testC2Class, "C2分页任务", func(r *CreateTaskReq) { r.AllowPublic = true })
	now := time.Now()
	students := []uint64{testC2Student, testC2StudentC, testC2StudentB}
	for i := 0; i < 3; i++ {
		require.NoError(t, f.db.Create(&domain.TaskCheckIn{
			TaskID: taskID, StudentID: students[i], UserID: testC2Parent,
			Date: f.today.AddDate(0, 0, -i), Content: "公开内容",
			IsPublicRequested: true, PublicScopeConfirmed: true,
			PublicApprovedAt: &now, PublicApprovedBy: testC2Admin,
		}).Error)
	}

	first, err := f.svc.ListPublicCheckins(f.ctx, testC2KG, taskID, "public",
		pagination.Query{Page: 1, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(3), first["total"], "total 是过滤后的总数，不是本页条数")
	assert.Len(t, first["list"].([]map[string]any), 2)
	assert.Equal(t, 1, first["page"])
	assert.Equal(t, 2, first["page_size"])

	second, err := f.svc.ListPublicCheckins(f.ctx, testC2KG, taskID, "public",
		pagination.Query{Page: 2, PageSize: 2})
	require.NoError(t, err)
	assert.Len(t, second["list"].([]map[string]any), 1)

	// 缺省分页参数必须被归一化：不能让一次请求把全表拉走。
	def, err := f.svc.ListPublicCheckins(f.ctx, testC2KG, taskID, "public", pagination.Query{})
	require.NoError(t, err)
	assert.Equal(t, 1, def["page"])
	assert.Equal(t, 20, def["page_size"])
}
