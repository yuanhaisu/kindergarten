// Package authharness 提供认证/身份测试的**单一 HTTP 缝**：
// 真实 Gin 引擎 + 真实测试库 + 假微信 + 假 Redis。
//
// 为什么只有这一条缝：后续身份票要验证「登录 → token → 中间件 → 业务接口」的
// 完整链路。如果分别对 service 和中间件做替身测试，两边都通过但拼起来仍可能
// 不通（角色裁剪、园所门禁、Casbin 都是装配层行为）。这里只拼真实组件，
// 替身仅限外部依赖（微信、Redis、数据库实例）。
package authharness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"kindergarten-server/internal/config"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/rbac"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/fakeredis"
)

// Harness 是一次测试的完整运行环境。
type Harness struct {
	Config *config.Config
	DB     *gorm.DB

	Redis       *fakeredis.Server
	RedisClient *redis.Client

	WeChat *FakeWeChat
	Auth   *service.AuthService
	// Handler 暴露给后续票在 Engine 上追加路由（如 teacher/parent 业务组）。
	Handler *handler.Handler
	Engine  *gin.Engine
	// Enforcer 供需要验证角色裁剪的测试直接检查策略。
	Enforcer *casbin.Enforcer

	Secret []byte

	DevDatabase  string
	TestDatabase string
	TestDSN      string
	// TestDSNFromEnv 为 true 表示测试库由 KDG_TEST_MYSQL_DSN 显式指定。
	TestDSNFromEnv bool
}

// New 装配测试环境。测试库不可用时 t.Skip，配置/迁移错误时 t.Fatal：
// 「跳过」只用于环境缺失，不允许把真实故障伪装成跳过。
func New(t testing.TB) *Harness {
	t.Helper()

	setup := sharedTestDB()
	if setup.skipReason != "" {
		t.Skipf("authharness: %s", setup.skipReason)
	}
	if setup.err != nil {
		t.Fatalf("authharness: %v", setup.err)
	}

	redisServer := fakeredis.New(t)
	rdb := redisServer.Client(t)
	fakeWx := NewFakeWeChat()

	userRepo := repository.NewUserRepo(setup.db)
	userRoleRepo := repository.NewUserRoleRepo(setup.db)
	kgStatus := service.NewKindergartenStatusService(repository.NewKindergartenRepo(setup.db), rdb)
	adminAuthz := service.NewAdminAuthzService(repository.NewAdminRepo(setup.db), userRoleRepo)

	secret, atTTL, rtTTL := jwtConfig(setup.cfg)
	authSvc := service.NewAuthService(
		setup.db,
		userRepo,
		userRoleRepo,
		repository.NewTeacherInvitationRepo(setup.db),
		adminAuthz,
		kgStatus,
		rdb,
		fakeWx,
		secret,
		atTTL,
		rtTTL,
		repository.NewConsentRepo(setup.db),
		false,
	)

	h := &Harness{
		Config:         setup.cfg,
		DB:             setup.db,
		Redis:          redisServer,
		RedisClient:    rdb,
		WeChat:         fakeWx,
		Auth:           authSvc,
		Secret:         secret,
		DevDatabase:    setup.devDatabase,
		TestDatabase:   setup.testDatabase,
		TestDSN:        setup.testDSN,
		TestDSNFromEnv: setup.dsnFromEnv,
	}

	engine, hnd, enforcer, err := buildEngine(h, setup.repoRoot, kgStatus)
	if err != nil {
		t.Fatalf("authharness: 装配 HTTP 引擎失败：%v", err)
	}
	h.Engine = engine
	h.Handler = hnd
	h.Enforcer = enforcer
	return h
}

func jwtConfig(cfg *config.Config) (secret []byte, atTTL, rtTTL time.Duration) {
	secret = []byte("authharness-test-jwt-secret")
	atTTL = 15 * time.Minute
	rtTTL = 7 * 24 * time.Hour
	if cfg == nil {
		return secret, atTTL, rtTTL
	}
	if cfg.JWT.Secret != "" {
		secret = []byte(cfg.JWT.Secret)
	}
	if cfg.JWT.AccessTTLMin > 0 {
		atTTL = time.Duration(cfg.JWT.AccessTTLMin) * time.Minute
	}
	if cfg.JWT.RefreshTTLDay > 0 {
		rtTTL = time.Duration(cfg.JWT.RefreshTTLDay) * 24 * time.Hour
	}
	return secret, atTTL, rtTTL
}

// buildEngine 复刻 router.go 中认证与 common 分组的装配：
// AuthJWT → RequireActiveKindergarten → CasbinRBAC → ModuleGate。
//
// 只注册这些分组，不拉整棵业务依赖树；后续票需要业务组时在 h.Handler 上追加，
// 继续复用同一套中间件与同一个测试库。
func buildEngine(h *Harness, repoRoot string, kgStatus service.KindergartenStatusChecker) (*gin.Engine, *handler.Handler, *casbin.Enforcer, error) {
	enforcer, err := rbac.NewEnforcer(&config.CasbinConfig{
		Enable:     true,
		ModelPath:  filepath.Join(repoRoot, "configs", "rbac_model.conf"),
		PolicyPath: filepath.Join(repoRoot, "configs", "rbac_policy.csv"),
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("初始化 Casbin：%w", err)
	}

	hnd := authOnlyHandler(h.Auth)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())

	v1 := r.Group("/api/v1")
	// v1 短信通道已关闭（#162），路由保留但统一走拒绝处理，与 router.go 保持一致。
	v1.POST("/auth/sms/send", hnd.SMSDisabled)
	v1.POST("/auth/login/sms", hnd.SMSDisabled)
	v1.POST("/auth/login/wechat", hnd.WeChatLogin)
	v1.POST("/auth/login/password", hnd.PasswordLogin)
	v1.POST("/auth/refresh", hnd.Refresh)

	authed := v1.Group("")
	authed.Use(middleware.AuthJWT(h.Secret, h.Auth.TokenRevoker()))
	authed.Use(middleware.RequireActiveKindergarten(kgStatus))
	authed.Use(middleware.CasbinRBAC(enforcer))
	authed.Use(middleware.ModuleGate(nil))
	{
		authed.POST("/auth/logout", hnd.Logout)
		authed.GET("/common/user/profile", hnd.GetProfile)
		authed.PUT("/common/user/profile", hnd.UpdateProfile)
		authed.POST("/common/user/bind-invite", hnd.BindInvite)
		authed.POST("/common/user/bind-parent-phone", hnd.BindParentPhone)
		authed.GET("/common/consent", hnd.GetConsent)
		authed.POST("/common/consent", hnd.PostConsent)
	}

	return r, hnd, enforcer, nil
}

// authOnlyHandler 只装配 AuthService，其余服务保持 nil。
//
// 认证/common 路由只依赖 authSvc，无需为了一条测试缝拉起整棵依赖树。
// 这里显式传全参而不是反射/零值构造：handler.NewHandler 增删参数时编译失败，
// 强制同步，避免测试缝悄悄错位。
func authOnlyHandler(authSvc *service.AuthService) *handler.Handler {
	return handler.NewHandler(
		authSvc,
		nil, // fileSvc
		nil, // msgSvc
		nil, // systemSvc
		nil, // dashSvc
		nil, // classSvc
		nil, // studentSvc
		nil, // teacherSvc
		nil, // parentSvc
		nil, // settingsSvc
		nil, // adminSvc
		nil, // roleSvc
		nil, // attendanceSvc
		nil, // leaveSvc
		nil, // pickupSvc
		nil, // growthSvc
		nil, // taskSvc
		nil, // pointSvc
		nil, // achievementSvc
		nil, // noticeSvc
		nil, // announcementSvc
		nil, // auditSvc
		nil, // oplogSvc
		nil, // exportSvc
		nil, // teacherPortalSvc
		nil, // parentPortalSvc
		nil, // dictSvc
		nil, // healthCheckSvc
		nil, // faceSvc
		nil, // gardenCalendarSvc
		nil, // timetableSvc
		nil, // dietGroupObjectSvc
		nil, // dietFoodSvc
		nil, // dietRecipeSvc
		nil, // dietPlanSvc
		nil, // dietAnalysisSvc
		nil, // platformSvc
		nil, // planSvc
		nil, // orderSvc
		nil, // distSvc
		nil, // entSvc
	)
}

// Response 是 HTTP 单缝的响应快照。
type Response struct {
	Status int
	Body   []byte
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// EnvelopeCode 返回统一响应包络里的业务码（成功为 0）。
func (r *Response) EnvelopeCode(t testing.TB) int {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("authharness: 解析响应包络失败：%v；body=%s", err, r.Body)
	}
	return env.Code
}

// DecodeData 把包络 data 解到 dst。
func (r *Response) DecodeData(t testing.TB, dst any) {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("authharness: 解析响应包络失败：%v；body=%s", err, r.Body)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		t.Fatalf("authharness: 响应没有 data；code=%d message=%q body=%s", env.Code, env.Message, r.Body)
	}
	if err := json.Unmarshal(env.Data, dst); err != nil {
		t.Fatalf("authharness: 解析 data 失败：%v；data=%s", err, env.Data)
	}
}

// Do 直接通过进程内 Gin 引擎发请求，不经过网络端口。
func (h *Harness) Do(t testing.TB, method, path, token string, body []byte) *Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	h.Engine.ServeHTTP(w, req)
	return &Response{Status: w.Code, Body: w.Body.Bytes()}
}

func (h *Harness) Get(t testing.TB, path, token string) *Response {
	t.Helper()
	return h.Do(t, http.MethodGet, path, token, nil)
}

func (h *Harness) PostJSON(t testing.TB, path string, payload any) *Response {
	t.Helper()
	return h.PostJSONWithToken(t, path, "", payload)
}

func (h *Harness) PostJSONWithToken(t testing.TB, path, token string, payload any) *Response {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("authharness: 序列化请求体失败：%v", err)
	}
	return h.Do(t, http.MethodPost, path, token, body)
}

// HardDelete 绕过软删真正删除行。
//
// 必须硬删：软删行仍占用唯一索引（后续 users.phone_active 是 STORED 生成列 +
// 唯一索引），残留会让下一个用例撞键。空 WHERE 直接拒绝，防止误清全表。
func (h *Harness) HardDelete(t testing.TB, model any, where string, args ...any) {
	t.Helper()
	if strings.TrimSpace(where) == "" {
		t.Fatalf("authharness: HardDelete 必须带 WHERE 条件，禁止清空整表")
	}
	if err := h.DB.Unscoped().Where(where, args...).Delete(model).Error; err != nil {
		t.Fatalf("authharness: 硬删失败（where=%s）：%v", where, err)
	}
}
