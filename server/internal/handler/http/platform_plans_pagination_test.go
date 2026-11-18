package handler

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/pkg/pagination"
)

// 分页夹具用专属 status 值圈定数据范围，避免与开发库里的真实套餐互相干扰。
// 长度必须落在 plans.status 的 varchar(16) 内。
const planPaginationStatus = "t1pag_scope"

type planListEnvelope struct {
	Code int `json:"code"`
	Data struct {
		List     []service.PlanRow `json:"list"`
		Total    int64             `json:"total"`
		Page     int               `json:"page"`
		PageSize int               `json:"page_size"`
	} `json:"data"`
}

func requestPlanList(t *testing.T, h *Handler, query string) planListEnvelope {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/platform/plans?"+query, nil)
	h.PlatformListPlans(c)
	require.Equal(t, 200, w.Code)
	var body planListEnvelope
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 0, body.Code)
	return body
}

// fakePlanCatalog 按收到的分页参数切片，模拟真实 service 的行为；
// 同时记录入参，用于断言 handler 的归一化与状态筛选透传。
type fakePlanCatalog struct {
	rows       []service.PlanRow
	lastQuery  pagination.Query
	lastStatus string
}

func (f *fakePlanCatalog) ListModules() []service.ModuleDef { return nil }

func (f *fakePlanCatalog) ListPlans(_ context.Context, status string, pq pagination.Query) ([]service.PlanRow, int64, int, int, error) {
	f.lastQuery = pq
	f.lastStatus = status
	start := pq.Offset()
	if start > len(f.rows) {
		start = len(f.rows)
	}
	end := start + pq.PageSize
	if end > len(f.rows) {
		end = len(f.rows)
	}
	return f.rows[start:end], int64(len(f.rows)), pq.Page, pq.PageSize, nil
}

func (f *fakePlanCatalog) Create(context.Context, service.PlanRequest, string, string) (*domain.Plan, error) {
	return nil, nil
}

func (f *fakePlanCatalog) Update(context.Context, uint64, service.PlanRequest, string, string) error {
	return nil
}

func (f *fakePlanCatalog) ListTiers(context.Context, uint64) ([]domain.PlanTier, error) {
	return nil, nil
}

func (f *fakePlanCatalog) ReplaceTiers(context.Context, uint64, []domain.PlanTier, string, string) error {
	return nil
}

// GET /platform/plans 是真实分页：page/page_size 生效，第 2 页与第 1 页数据不同，
// total 始终是筛选后的真实总数。
func TestPlatformListPlansPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &fakePlanCatalog{rows: []service.PlanRow{
		{ID: 1, Code: "t1pag-a"},
		{ID: 2, Code: "t1pag-b"},
		{ID: 3, Code: "t1pag-c"},
		{ID: 4, Code: "t1pag-d"},
	}}
	h := &Handler{planSvc: fake}

	first := requestPlanList(t, h, "page=1&page_size=2")
	require.Len(t, first.Data.List, 2)
	assert.EqualValues(t, 4, first.Data.Total)
	assert.Equal(t, 1, first.Data.Page)
	assert.Equal(t, 2, first.Data.PageSize)
	assert.Equal(t, "t1pag-a", first.Data.List[0].Code)
	assert.Equal(t, "t1pag-b", first.Data.List[1].Code)

	second := requestPlanList(t, h, "page=2&page_size=2")
	require.Len(t, second.Data.List, 2)
	assert.EqualValues(t, 4, second.Data.Total, "第 2 页的 total 仍是真实总数")
	assert.Equal(t, 2, second.Data.Page)
	assert.Equal(t, 2, second.Data.PageSize)
	assert.Equal(t, "t1pag-c", second.Data.List[0].Code)
	assert.Equal(t, "t1pag-d", second.Data.List[1].Code)

	// 缺省与越界参数按契约归一化。
	def := requestPlanList(t, h, "")
	assert.Len(t, def.Data.List, 4)
	assert.Equal(t, 1, def.Data.Page)
	assert.Equal(t, 20, def.Data.PageSize)

	clamped := requestPlanList(t, h, "page=0&page_size=500")
	assert.Equal(t, 1, clamped.Data.Page)
	assert.Equal(t, 100, clamped.Data.PageSize)

	// status 筛选透传给 service。
	_ = requestPlanList(t, h, "status=archived&page=1&page_size=2")
	assert.Equal(t, "archived", fake.lastStatus)
	assert.Equal(t, 1, fake.lastQuery.Page)
	assert.Equal(t, 2, fake.lastQuery.PageSize)
}

func openPlanHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	cfgPath := filepath.Join("..", "..", "..", "configs", "app.ini")
	if _, err := os.Stat(cfgPath); err != nil {
		t.Skip("跳过：本地无 DB 配置")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil || cfg.MySQL.DSN == "" {
		t.Skip("跳过：本地无可用 DB 配置")
	}
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		t.Skipf("跳过：无法连接开发库: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.Plan{}, &domain.PlanModule{}, &domain.PlanTier{}, &domain.PlatformOperationLog{},
	); err != nil {
		t.Fatalf("自动迁移套餐测试表失败: %v", err)
	}
	return db
}

func cleanupPlanPaginationFixtures(t *testing.T, db *gorm.DB) {
	t.Helper()
	del := func(sql string, args ...any) {
		if err := db.Exec(sql, args...).Error; err != nil {
			t.Logf("cleanup %q: %v", sql, err)
		}
	}
	const plansIn = "SELECT id FROM plans WHERE code LIKE 't1pag-%'"
	del("DELETE FROM platform_operation_logs WHERE target_type = 'plan' AND target_id IN (" + plansIn + ")")
	del("DELETE FROM plan_tiers WHERE plan_id IN (" + plansIn + ")")
	del("DELETE FROM plan_modules WHERE plan_id IN (" + plansIn + ")")
	del("DELETE FROM plans WHERE code LIKE 't1pag-%'")
}

// 集成路径打通 handler → service → repository：验证 offset/limit 与真实 total。
// 本地开发库不可用时跳过（CI 不跑 DB 测试）。
func TestPlatformListPlansPaginationIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openPlanHandlerTestDB(t)
	cleanupPlanPaginationFixtures(t, db)
	t.Cleanup(func() { cleanupPlanPaginationFixtures(t, db) })

	planSvc := service.NewPlanService(repository.NewPlanRepo(db), repository.NewPlatformLogRepo(db))
	h := &Handler{planSvc: planSvc}
	ctx := context.Background()
	for i, code := range []string{"t1pag-alpha", "t1pag-beta", "t1pag-gamma"} {
		_, err := planSvc.Create(ctx, service.PlanRequest{
			Code: code, Name: "T1分页" + code, Tier: i + 1, PriceCents: 100000, TermMonths: 6,
			ModuleCodes: []string{"base"}, Status: planPaginationStatus,
		}, "t1pag_ops", "127.0.0.1")
		require.NoError(t, err)
	}

	first := requestPlanList(t, h, "status="+planPaginationStatus+"&page=1&page_size=2")
	require.Len(t, first.Data.List, 2)
	assert.EqualValues(t, 3, first.Data.Total)
	assert.Equal(t, 1, first.Data.Page)
	assert.Equal(t, 2, first.Data.PageSize)
	assert.Equal(t, "t1pag-alpha", first.Data.List[0].Code)
	assert.Equal(t, "t1pag-beta", first.Data.List[1].Code)

	second := requestPlanList(t, h, "status="+planPaginationStatus+"&page=2&page_size=2")
	require.Len(t, second.Data.List, 1)
	assert.EqualValues(t, 3, second.Data.Total)
	assert.Equal(t, 2, second.Data.Page)
	assert.Equal(t, "t1pag-gamma", second.Data.List[0].Code)

	empty := requestPlanList(t, h, "status="+planPaginationStatus+"&page=9&page_size=2")
	assert.Empty(t, empty.Data.List)
	assert.EqualValues(t, 3, empty.Data.Total)
}
