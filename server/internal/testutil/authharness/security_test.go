package authharness_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"golang.org/x/crypto/bcrypt"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/middleware"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
	"kindergarten-server/internal/testutil/authharness"
)

var phoneSeq atomic.Uint64

// uniquePhone 生成 11 位手机号。测试库跨用例共享，历史残留不能与新用例撞号。
func uniquePhone() string {
	n := (time.Now().UnixNano() + int64(phoneSeq.Add(1))*7919) % 1_000_000_000
	return fmt.Sprintf("19%09d", n)
}

// createUser 写入一个带密码的账号；role 非空时同时写入角色。Cleanup 先删角色再删账号。
func createUser(t *testing.T, h *authharness.Harness, phone, password, role string) *domain.User {
	t.Helper()
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("哈希密码失败：%v", err)
	}
	u := &domain.User{Phone: phone, Nickname: "u-" + phone, PasswordHash: hash}
	if err := h.DB.Create(u).Error; err != nil {
		t.Fatalf("创建用户失败：%v", err)
	}
	t.Cleanup(func() {
		// FK 是 RESTRICT：先清所有引用该账号的子行，再删账号。
		for _, model := range []any{
			&domain.TeacherInvitation{},
			&domain.TeacherProfile{},
			&domain.ParentProfile{},
			&domain.ClassTeacher{},
			&domain.StudentParent{},
			&domain.UserRole{},
		} {
			h.HardDelete(t, model, "user_id = ?", u.ID)
		}
		h.HardDelete(t, &domain.User{}, "id = ?", u.ID)
	})
	if role != "" {
		if err := h.DB.Create(&domain.UserRole{UserID: u.ID, Role: role}).Error; err != nil {
			t.Fatalf("创建角色失败：%v", err)
		}
	}
	return u
}

func loginWithPassword(t *testing.T, h *authharness.Harness, phone, password string) *authharness.Response {
	t.Helper()
	return h.PostJSON(t, "/api/v1/auth/login/password", map[string]string{
		"phone":    phone,
		"password": password,
	})
}

// 密码登录在 v1 只服务管理后台；家长/老师账号即使密码正确也必须被拒，
// 否则预建账号的弱口令就是一条提权入口（#174）。
func TestPasswordLoginRejectsNonAdminAccounts(t *testing.T) {
	h := authharness.New(t)

	for _, role := range []string{"parent", "teacher"} {
		t.Run(role, func(t *testing.T) {
			phone := uniquePhone()
			createUser(t, h, phone, "Password@123", role)

			res := loginWithPassword(t, h, phone, "Password@123")
			if res.Status != http.StatusForbidden {
				t.Fatalf("期望 403，实际 %d，body=%s", res.Status, res.Body)
			}
			if code := res.EnvelopeCode(t); code != 40300 {
				t.Fatalf("期望业务码 40300，实际 %d", code)
			}
		})
	}
}

// 管理员密码登录必须保持可用，防止“关掉越权”顺手把后台登录也关掉。
func TestPasswordLoginAllowsAdminAccounts(t *testing.T) {
	h := authharness.New(t)

	phone := uniquePhone()
	createUser(t, h, phone, "Password@123", "admin")

	res := loginWithPassword(t, h, phone, "Password@123")
	if res.Status != http.StatusOK {
		t.Fatalf("期望 200，实际 %d，body=%s", res.Status, res.Body)
	}
	var data struct {
		Token string `json:"token"`
	}
	res.DecodeData(t, &data)
	if data.Token == "" {
		t.Fatalf("管理员登录未返回 token：%s", res.Body)
	}
}

// fixture 提供一个幼儿园 + 一个班级，并负责按依赖顺序硬删本用例写入的行。
// 测试库跨用例共享，残留会占唯一索引（users.phone_active、profile 唯一键）。
type fixture struct {
	kg     *domain.Kindergarten
	class  *domain.Class
	phones []string
}

func newFixture(t *testing.T, h *authharness.Harness) *fixture {
	t.Helper()
	f := &fixture{}
	kg := &domain.Kindergarten{Name: "测试园-" + uniquePhone(), Status: "active"}
	if err := h.DB.Create(kg).Error; err != nil {
		t.Fatalf("创建园所失败：%v", err)
	}
	f.kg = kg
	cl := &domain.Class{KindergartenID: kg.ID, Name: "小一班-" + uniquePhone(), Grade: "小班", AcademicYear: "2026"}
	if err := h.DB.Create(cl).Error; err != nil {
		t.Fatalf("创建班级失败：%v", err)
	}
	f.class = cl
	t.Cleanup(func() { f.cleanup(t, h) })
	return f
}

func (f *fixture) track(phone string) string {
	f.phones = append(f.phones, phone)
	return phone
}

func (f *fixture) cleanup(t *testing.T, h *authharness.Harness) {
	t.Helper()
	db := h.DB.Unscoped()

	var userIDs []uint64
	if len(f.phones) > 0 {
		if err := db.Model(&domain.User{}).Where("phone IN ?", f.phones).Pluck("id", &userIDs).Error; err != nil {
			t.Errorf("cleanup: 查询用户失败：%v", err)
		}
	}
	var studentIDs []uint64
	db.Model(&domain.Student{}).Where("kindergarten_id = ?", f.kg.ID).Pluck("id", &studentIDs)
	var classIDs []uint64
	db.Model(&domain.Class{}).Where("kindergarten_id = ?", f.kg.ID).Pluck("id", &classIDs)

	if len(userIDs) > 0 {
		db.Where("user_id IN ?", userIDs).Delete(&domain.TeacherInvitation{})
		db.Where("user_id IN ?", userIDs).Delete(&domain.TeacherProfile{})
		db.Where("user_id IN ?", userIDs).Delete(&domain.ParentProfile{})
		db.Where("user_id IN ?", userIDs).Delete(&domain.ClassTeacher{})
		db.Where("user_id IN ?", userIDs).Delete(&domain.StudentParent{})
	}
	if len(studentIDs) > 0 {
		db.Where("student_id IN ?", studentIDs).Delete(&domain.StudentParent{})
	}
	db.Where("kindergarten_id = ?", f.kg.ID).Delete(&domain.OperationLog{})
	db.Where("kindergarten_id = ?", f.kg.ID).Delete(&domain.Student{})
	db.Where("kindergarten_id = ?", f.kg.ID).Delete(&domain.TeacherProfile{})
	db.Where("kindergarten_id = ?", f.kg.ID).Delete(&domain.ParentProfile{})
	db.Where("kindergarten_id = ?", f.kg.ID).Delete(&domain.TeacherInvitation{})
	db.Where("class_id IN ?", classIDs).Delete(&domain.ClassTeacher{})
	db.Where("user_id IN ?", userIDs).Delete(&domain.UserRole{})
	db.Where("kindergarten_id = ?", f.kg.ID).Delete(&domain.UserRole{})
	if len(userIDs) > 0 {
		db.Where("id IN ?", userIDs).Delete(&domain.User{})
	}
	db.Where("kindergarten_id = ?", f.kg.ID).Delete(&domain.Class{})
	db.Where("id = ?", f.kg.ID).Delete(&domain.Kindergarten{})
}

// decodeReq 用 JSON 解码构造 service 请求：部分请求 DTO 的内层类型未导出，
// 外部测试包无法直接写复合字面量。JSON 字段名就是 handler 的契约，不会漂移。
func decodeReq[T any](t *testing.T, raw string) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("构造请求失败：%v；json=%s", err, raw)
	}
	return v
}

// 密码登录按 IP + 手机号限流：同一出口 IP 下不同账号互不挤占配额，
// 同一账号 10 次/分钟后第 11 次直接 429（#174）。
func TestPasswordLoginRateLimitIsScopedByPhone(t *testing.T) {
	h := authharness.New(t)

	phoneA := uniquePhone()
	createUser(t, h, phoneA, "Right@123", "admin")
	phoneB := uniquePhone()
	createUser(t, h, phoneB, "Right@123", "admin")

	for i := 0; i < 10; i++ {
		res := loginWithPassword(t, h, phoneA, "Wrong@123")
		if res.Status != http.StatusBadRequest {
			t.Fatalf("第 %d 次失败登录期望 400，实际 %d，body=%s", i+1, res.Status, res.Body)
		}
	}

	res := loginWithPassword(t, h, phoneA, "Wrong@123")
	if res.Status != http.StatusTooManyRequests {
		t.Fatalf("超过配额期望 429，实际 %d，body=%s", res.Status, res.Body)
	}
	if code := res.EnvelopeCode(t); code != 42900 {
		t.Fatalf("期望业务码 42900，实际 %d", code)
	}

	// 正确密码也不能突破本账号窗口。
	if res := loginWithPassword(t, h, phoneA, "Right@123"); res.Status != http.StatusTooManyRequests {
		t.Fatalf("超限后正确密码仍应 429，实际 %d，body=%s", res.Status, res.Body)
	}

	// 同一出口 IP 的另一个园长不受影响——幼儿园共用出口 IP 不能被误伤。
	if res := loginWithPassword(t, h, phoneB, "Right@123"); res.Status != http.StatusOK {
		t.Fatalf("同 IP 另一账号应可登录，实际 %d，body=%s", res.Status, res.Body)
	}
}

// 密码登录限流是失败关闭：Redis 写失败时不能放行，否则故障期间等于没有防爆破。
func TestPasswordLoginRateLimitFailsClosedWhenStoreUnavailable(t *testing.T) {
	h := authharness.New(t)

	phone := uniquePhone()
	createUser(t, h, phone, "Right@123", "admin")

	h.Redis.SetFailWrites(true)
	res := loginWithPassword(t, h, phone, "Right@123")
	if res.Status != http.StatusServiceUnavailable {
		t.Fatalf("存储故障期望 503，实际 %d，body=%s", res.Status, res.Body)
	}
	if code := res.EnvelopeCode(t); code != 50300 {
		t.Fatalf("期望业务码 50300，实际 %d", code)
	}

	h.Redis.SetFailWrites(false)
	if res := loginWithPassword(t, h, phone, "Right@123"); res.Status != http.StatusOK {
		t.Fatalf("存储恢复后应可登录，实际 %d，body=%s", res.Status, res.Body)
	}
}

// logCapture 捕获 slog 记录，用于断言失败开放时确实留下了告警。
type logCapture struct {
	mu      sync.Mutex
	records []slog.Record
}

func (l *logCapture) Enabled(context.Context, slog.Level) bool { return true }

func (l *logCapture) Handle(_ context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, r.Clone())
	return nil
}

func (l *logCapture) WithAttrs([]slog.Attr) slog.Handler { return l }
func (l *logCapture) WithGroup(string) slog.Handler      { return l }

func (l *logCapture) hasWarning(prefix string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.records {
		if r.Level != slog.LevelWarn || !strings.Contains(r.Message, "allowing request") {
			continue
		}
		found := false
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == "prefix" && a.Value.String() == prefix {
				found = true
				return false
			}
			return true
		})
		if found {
			return true
		}
	}
	return false
}

func captureLogs(t *testing.T) *logCapture {
	t.Helper()
	prev := slog.Default()
	lc := &logCapture{}
	slog.SetDefault(slog.New(lc))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return lc
}

// 微信登录限流按 app_id + IP、阈值取数百/分钟：整所幼儿园共用出口 IP
// 的正常登录不能被误伤；只有同一 app_id 的洪峰才 429（#174）。
//
// 这里用未注册的 app_id：请求最终会被 #175 的新登录 handler 以 403 拒绝
// （客户端未注册），但限流配额仍按 app_id + IP 累计，断言的是限流层行为。
func TestWeChatLoginRateLimitIsHighAndScopedByApp(t *testing.T) {
	h := authharness.New(t)

	payload := map[string]string{"app_id": "wx-teacher", "code": "one-time-code"}
	for i := 0; i < middleware.WeChatLoginMaxPerMinute; i++ {
		res := h.PostJSON(t, "/api/v1/auth/login/wechat", payload)
		if res.Status != http.StatusForbidden {
			t.Fatalf("第 %d 次请求期望 403（客户端未注册），实际 %d，body=%s", i+1, res.Status, res.Body)
		}
	}

	res := h.PostJSON(t, "/api/v1/auth/login/wechat", payload)
	if res.Status != http.StatusTooManyRequests {
		t.Fatalf("超过高阈值期望 429，实际 %d，body=%s", res.Status, res.Body)
	}

	// 同一 IP、不同 app_id 仍有独立配额（放行到 handler 后仍是未注册的 403）。
	res = h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{"app_id": "wx-parent", "code": "one-time-code"})
	if res.Status != http.StatusForbidden {
		t.Fatalf("不同 app_id 不应被连带限流（仍应是客户端未注册的 403），实际 %d，body=%s", res.Status, res.Body)
	}
}

// 微信登录限流失败开放：存储故障时必须放行到 handler（未注册客户端 → 403）
// 并告警，不能把登录打成 503。
func TestWeChatLoginRateLimitFailsOpenWithWarning(t *testing.T) {
	h := authharness.New(t)
	logs := captureLogs(t)

	h.Redis.SetFailWrites(true)
	res := h.PostJSON(t, "/api/v1/auth/login/wechat", map[string]string{"app_id": "wx-teacher", "code": "c"})
	if res.Status != http.StatusForbidden {
		t.Fatalf("存储故障时应放行到 handler（403），实际 %d，body=%s", res.Status, res.Body)
	}
	if !logs.hasWarning("wechat_login") {
		t.Fatalf("失败开放必须留下告警日志")
	}
}

// 绑定限流按 app_id + IP + 手机号，20 次/分钟；失败开放但必须告警。
// 绑定端点由 #167/#168 落地，这里用同一中间件挂测试路由验证契约。
func TestBindPhoneRateLimitIsScopedAndFailsOpen(t *testing.T) {
	h := authharness.New(t)
	logs := captureLogs(t)

	const limit = 3
	h.Engine.POST("/test/bind", middleware.BindPhoneRateLimit(h.RedisClient, limit, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	body := map[string]string{"app_id": "wx-parent", "phone": "13800000001"}

	for i := 0; i < limit; i++ {
		res := h.PostJSON(t, "/test/bind", body)
		if res.Status != http.StatusNoContent {
			t.Fatalf("第 %d 次绑定期望 204，实际 %d，body=%s", i+1, res.Status, res.Body)
		}
	}
	if res := h.PostJSON(t, "/test/bind", body); res.Status != http.StatusTooManyRequests {
		t.Fatalf("超过绑定配额期望 429，实际 %d，body=%s", res.Status, res.Body)
	}

	// 同 app、同 IP、不同手机号不受影响（一所幼儿园多个家长共用出口 IP）。
	otherPhone := map[string]string{"app_id": "wx-parent", "phone": "13800000002"}
	if res := h.PostJSON(t, "/test/bind", otherPhone); res.Status != http.StatusNoContent {
		t.Fatalf("不同手机号不应被连带限流，实际 %d，body=%s", res.Status, res.Body)
	}
	// 同手机号、不同 app 也不受影响。
	otherApp := map[string]string{"app_id": "wx-other", "phone": "13800000001"}
	if res := h.PostJSON(t, "/test/bind", otherApp); res.Status != http.StatusNoContent {
		t.Fatalf("不同 app_id 不应被连带限流，实际 %d，body=%s", res.Status, res.Body)
	}

	h.Redis.SetFailWrites(true)
	if res := h.PostJSON(t, "/test/bind", body); res.Status != http.StatusNoContent {
		t.Fatalf("存储故障时应放行绑定，实际 %d，body=%s", res.Status, res.Body)
	}
	if !logs.hasWarning("bind_phone") {
		t.Fatalf("绑定限流失败开放必须留下告警日志")
	}
}

func passwordHashOf(t *testing.T, h *authharness.Harness, phone string) string {
	t.Helper()
	var u domain.User
	if err := h.DB.Where("phone = ?", phone).First(&u).Error; err != nil {
		t.Fatalf("读取用户失败（phone=%s）：%v", phone, err)
	}
	return u.PasswordHash
}

// 预建的家长/老师账号不得持有任何可用口令：管理员新建/导入学生、老师、家长
// 都只解析账号，不写 password_hash（#174）。园长初始密码由开通流程负责，不在此列。
func TestPreCreatedAccountsHaveNoUsablePassword(t *testing.T) {
	h := authharness.New(t)
	f := newFixture(t, h)
	ctx := context.Background()

	users := repository.NewUserRepo(h.DB)
	roles := repository.NewUserRoleRepo(h.DB)
	students := repository.NewStudentRepo(h.DB)
	classes := repository.NewClassRepo(h.DB)
	kgRepo := repository.NewKindergartenRepo(h.DB)
	oplog := repository.NewOperationLogRepo(h.DB)

	studentSvc := service.NewStudentService(h.DB, students, classes, users, roles, kgRepo, oplog)
	parentSvc := service.NewParentService(h.DB, repository.NewParentRepo(h.DB), students, users, roles, kgRepo, oplog)
	teacherSvc := service.NewTeacherService(h.DB, repository.NewTeacherRepo(h.DB), classes, users, roles, kgRepo, oplog,
		repository.NewTeacherInvitationRepo(h.DB), nil)

	// 1) 新建学生：家长账号由 FindOrCreateByPhone 创建。
	studentPhone := f.track(uniquePhone())
	req := decodeReq[service.CreateStudentReq](t, fmt.Sprintf(`{
		"name":"小明","class_id":%d,"birthday":"2020-09-01","enroll_date":"2026-09-01",
		"parents":[{"name":"小明妈妈","phone":%q,"relation":"mother","is_primary":true}]
	}`, f.class.ID, studentPhone))
	student, err := studentSvc.CreateStudent(ctx, f.kg.ID, 1, req, "", "")
	if err != nil {
		t.Fatalf("CreateStudent 失败：%v", err)
	}

	// 2) 新建老师。
	teacherPhone := f.track(uniquePhone())
	treq := decodeReq[service.CreateTeacherReq](t, fmt.Sprintf(`{
		"real_name":"张老师","nickname":"张老师","phone":%q,"join_date":"2026-09-01",
		"class_assignments":[{"class_id":%d,"title":"head"}]
	}`, teacherPhone, f.class.ID))
	if _, err := teacherSvc.CreateTeacher(ctx, f.kg.ID, 1, treq, "", ""); err != nil {
		t.Fatalf("CreateTeacher 失败：%v", err)
	}

	// 3) 新建家长（绑定到上面创建的学生）。
	parentPhone := f.track(uniquePhone())
	preq := decodeReq[service.CreateParentReq](t, fmt.Sprintf(`{
		"real_name":"李家长","nickname":"李家长","phone":%q,
		"student_bindings":[{"student_id":%d,"relation":"father","is_primary":true}]
	}`, parentPhone, student.ID))
	if _, err := parentSvc.CreateParent(ctx, f.kg.ID, 1, preq, "", ""); err != nil {
		t.Fatalf("CreateParent 失败：%v", err)
	}

	// 4) Excel 导入学生。
	importPhone := f.track(uniquePhone())
	book := excelize.NewFile()
	sheet := book.GetSheetName(0)
	_ = book.SetCellValue(sheet, "A1", "姓名")
	_ = book.SetCellValue(sheet, "B1", "班级")
	_ = book.SetCellValue(sheet, "C1", "家长手机")
	_ = book.SetCellValue(sheet, "A2", "小红")
	_ = book.SetCellValue(sheet, "B2", f.class.Name)
	_ = book.SetCellValue(sheet, "C2", importPhone)
	buf, err := book.WriteToBuffer()
	if err != nil {
		t.Fatalf("生成 xlsx 失败：%v", err)
	}
	result, err := studentSvc.ImportStudentsExcel(ctx, f.kg.ID, 1, buf.Bytes(), "", "")
	if err != nil {
		t.Fatalf("ImportStudentsExcel 失败：%v", err)
	}
	if result.SuccessCount != 1 {
		t.Fatalf("导入应成功 1 行，实际 %d，明细=%+v", result.SuccessCount, result.FailDetails)
	}

	cases := []struct {
		name            string
		phone           string
		defaultPassword string
	}{
		{"create-student", studentPhone, "KindyParent#1"},
		{"create-teacher", teacherPhone, "KindyTeacher#1"},
		{"create-parent", parentPhone, "KindyParent#1"},
		{"import-students", importPhone, "KindyParent#1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if hash := passwordHashOf(t, h, tc.phone); hash != "" {
				t.Fatalf("预建账号不应写入密码哈希，实际非空（len=%d）", len(hash))
			}
			if _, err := h.Auth.PasswordLogin(ctx, tc.phone, tc.defaultPassword); err == nil {
				t.Fatalf("默认口令 %q 不应能登录", tc.defaultPassword)
			}
		})
	}

	// 已存在的账号密码不得被“新建家长/老师”覆盖。
	existingPhone := f.track(uniquePhone())
	createUser(t, h, existingPhone, "KeepMe@123", "")
	existingReq := decodeReq[service.CreateParentReq](t, fmt.Sprintf(`{
		"real_name":"王家长","nickname":"王家长","phone":%q,
		"student_bindings":[{"student_id":%d,"relation":"mother","is_primary":false}]
	}`, existingPhone, student.ID))
	if _, err := parentSvc.CreateParent(ctx, f.kg.ID, 1, existingReq, "", ""); err != nil {
		t.Fatalf("对既有账号 CreateParent 失败：%v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHashOf(t, h, existingPhone)), []byte("KeepMe@123")); err != nil {
		t.Fatalf("既有账号密码被覆盖：%v", err)
	}
}
