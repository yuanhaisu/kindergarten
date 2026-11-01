package app

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/domain"
	handler "kindergarten-server/internal/handler/http"
	"kindergarten-server/internal/service"
)

const testJWTSecret = "b09-test-secret"

func newRouterForTest(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine, err := NewRouter(NewAppHandler(&handler.Handler{}), nil, JWTSecret(testJWTSecret), nil)
	require.NoError(t, err)
	return engine
}

// /auth/logout 的注册语句曾被同一行的「退出登录」注释吞掉：语句属于注释，
// 路由从未出现在路由表里，客户端退出登录永远 404，黑名单写入也永远不会发生。
func TestAuthLogoutRouteRegistered(t *testing.T) {
	engine := newRouterForTest(t)

	var paths []string
	for _, r := range engine.Routes() {
		if r.Method == http.MethodPost && r.Path == "/api/v1/auth/logout" {
			return
		}
		paths = append(paths, r.Method+" "+r.Path)
	}
	t.Fatalf("POST /api/v1/auth/logout 未注册；已注册路由：%s", strings.Join(paths, ", "))
}

// 无 token 时必须在 AuthJWT 就被拦下，不能落到 handler。
func TestAuthLogoutWithoutTokenIsUnauthorized(t *testing.T) {
	engine := newRouterForTest(t)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------- 假 Redis：只实现认证路径用到的命令 ----------

type fakeRedisValue struct {
	value     string
	expiresAt time.Time // 零值表示永不过期
}

// fakeRedisServer 是进程内的最小 Redis（SET/GET/DEL/EXISTS/TTL/PING）。
// 用它而不是真实 Redis：测试不需要外部依赖，也不会污染开发库里真实的会话与黑名单。
type fakeRedisServer struct {
	ln         net.Listener
	mu         sync.Mutex
	data       map[string]fakeRedisValue
	failReads  bool
	failWrites bool
}

func newFakeRedisServer(t *testing.T) *fakeRedisServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	f := &fakeRedisServer{ln: ln, data: map[string]fakeRedisValue{}}
	go f.serve()
	t.Cleanup(func() { _ = ln.Close() })
	return f
}

func (f *fakeRedisServer) addr() string { return f.ln.Addr().String() }

func (f *fakeRedisServer) setFailReads(v bool)  { f.mu.Lock(); f.failReads = v; f.mu.Unlock() }
func (f *fakeRedisServer) setFailWrites(v bool) { f.mu.Lock(); f.failWrites = v; f.mu.Unlock() }

// exists 同时返回剩余 TTL，用于断言黑名单确实带过期时间。
func (f *fakeRedisServer) exists(key string) (bool, time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.lookupLocked(key)
	if !ok {
		return false, 0
	}
	if v.expiresAt.IsZero() {
		return true, -1
	}
	return true, time.Until(v.expiresAt)
}

func (f *fakeRedisServer) lookupLocked(key string) (fakeRedisValue, bool) {
	v, ok := f.data[key]
	if !ok {
		return fakeRedisValue{}, false
	}
	if !v.expiresAt.IsZero() && time.Now().After(v.expiresAt) {
		delete(f.data, key)
		return fakeRedisValue{}, false
	}
	return v, true
}

func (f *fakeRedisServer) serve() {
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.handle(conn)
	}
}

func (f *fakeRedisServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	r := bufio.NewReader(conn)
	for {
		cmd, err := readRESPArray(r)
		if err != nil {
			return
		}
		f.dispatch(conn, cmd)
	}
}

func (f *fakeRedisServer) dispatch(conn net.Conn, cmd []string) {
	if len(cmd) == 0 {
		_, _ = conn.Write([]byte("+OK\r\n"))
		return
	}
	switch strings.ToUpper(cmd[0]) {
	case "HELLO":
		// go-redis 只在非 RedisError 时判定连接失败；回 ERR 即退回 RESP2。
		_, _ = conn.Write([]byte("-ERR unknown command 'hello'\r\n"))
	case "PING":
		_, _ = conn.Write([]byte("+PONG\r\n"))
	case "SET":
		f.cmdSet(conn, cmd)
	case "GET":
		f.cmdGet(conn, cmd)
	case "DEL":
		f.cmdDel(conn, cmd)
	case "EXISTS":
		f.cmdExists(conn, cmd)
	case "TTL":
		f.cmdTTL(conn, cmd)
	default:
		_, _ = conn.Write([]byte("+OK\r\n"))
	}
}

func (f *fakeRedisServer) cmdSet(conn net.Conn, cmd []string) {
	if len(cmd) < 3 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'set'\r\n"))
		return
	}
	f.mu.Lock()
	fail := f.failWrites
	f.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated write failure\r\n"))
		return
	}

	v := fakeRedisValue{value: cmd[2]}
	now := time.Now()
	for i := 3; i+1 < len(cmd); i++ {
		n, err := strconv.Atoi(cmd[i+1])
		if err != nil {
			continue
		}
		switch strings.ToUpper(cmd[i]) {
		case "EX":
			v.expiresAt = now.Add(time.Duration(n) * time.Second)
		case "PX":
			v.expiresAt = now.Add(time.Duration(n) * time.Millisecond)
		}
		i++
	}

	f.mu.Lock()
	f.data[cmd[1]] = v
	f.mu.Unlock()
	_, _ = conn.Write([]byte("+OK\r\n"))
}

func (f *fakeRedisServer) cmdGet(conn net.Conn, cmd []string) {
	if len(cmd) < 2 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'get'\r\n"))
		return
	}
	f.mu.Lock()
	fail := f.failReads
	v, ok := f.lookupLocked(cmd[1])
	f.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated read failure\r\n"))
		return
	}
	if !ok {
		_, _ = conn.Write([]byte("$-1\r\n"))
		return
	}
	_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(v.value), v.value)
}

func (f *fakeRedisServer) cmdDel(conn net.Conn, cmd []string) {
	f.mu.Lock()
	fail := f.failWrites
	n := 0
	if !fail {
		for _, key := range cmd[1:] {
			if _, ok := f.lookupLocked(key); ok {
				delete(f.data, key)
				n++
			}
		}
	}
	f.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated write failure\r\n"))
		return
	}
	_, _ = fmt.Fprintf(conn, ":%d\r\n", n)
}

func (f *fakeRedisServer) cmdExists(conn net.Conn, cmd []string) {
	f.mu.Lock()
	fail := f.failReads
	n := 0
	if !fail {
		for _, key := range cmd[1:] {
			if _, ok := f.lookupLocked(key); ok {
				n++
			}
		}
	}
	f.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated read failure\r\n"))
		return
	}
	_, _ = fmt.Fprintf(conn, ":%d\r\n", n)
}

func (f *fakeRedisServer) cmdTTL(conn net.Conn, cmd []string) {
	if len(cmd) < 2 {
		_, _ = conn.Write([]byte("-ERR wrong number of arguments for 'ttl'\r\n"))
		return
	}
	f.mu.Lock()
	fail := f.failReads
	v, ok := f.lookupLocked(cmd[1])
	f.mu.Unlock()
	if fail {
		_, _ = conn.Write([]byte("-ERR simulated read failure\r\n"))
		return
	}
	switch {
	case !ok:
		_, _ = conn.Write([]byte(":-2\r\n"))
	case v.expiresAt.IsZero():
		_, _ = conn.Write([]byte(":-1\r\n"))
	default:
		_, _ = fmt.Fprintf(conn, ":%d\r\n", int64(time.Until(v.expiresAt)/time.Second))
	}
}

func readRESPArray(r *bufio.Reader) ([]string, error) {
	head, err := readCRLFLine(r)
	if err != nil {
		return nil, err
	}
	if len(head) == 0 || head[0] != '*' {
		return nil, fmt.Errorf("expected RESP array, got %q", head)
	}
	n, err := strconv.Atoi(head[1:])
	if err != nil {
		return nil, err
	}
	cmd := make([]string, 0, n)
	for i := 0; i < n; i++ {
		line, err := readCRLFLine(r)
		if err != nil {
			return nil, err
		}
		if len(line) == 0 || line[0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got %q", line)
		}
		size, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		cmd = append(cmd, string(buf[:size]))
	}
	return cmd, nil
}

func readCRLFLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), nil
}

func newFakeRedisClient(t *testing.T) (*redis.Client, *fakeRedisServer) {
	t.Helper()
	f := newFakeRedisServer(t)
	rdb := redis.NewClient(&redis.Options{
		Addr:            f.addr(),
		Protocol:        2,
		DisableIdentity: true,
		PoolSize:        1,
		MaxRetries:      -1,
		DialTimeout:     time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
	})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb, f
}

// ---------- 假仓库：登录之外的最小用户图 ----------

type fakeUserRepo struct {
	users map[uint64]*domain.User
}

func (f *fakeUserRepo) FindByID(_ context.Context, id uint64) (*domain.User, error) {
	return f.users[id], nil
}
func (f *fakeUserRepo) FindByPhone(context.Context, string) (*domain.User, error) { return nil, nil }
func (f *fakeUserRepo) FindByUnionID(context.Context, string) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) FindOtherByUnionID(context.Context, string, uint64) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) FindOtherByPhone(context.Context, string, uint64) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) FindByOpenID(context.Context, string) (*domain.User, error) {
	return nil, nil
}
func (f *fakeUserRepo) Create(context.Context, *domain.User) error { return nil }
func (f *fakeUserRepo) Update(context.Context, *domain.User) error { return nil }
func (f *fakeUserRepo) UpdateFields(context.Context, uint64, map[string]any) error {
	return nil
}

type fakeRoleRepo struct{}

func (fakeRoleRepo) FindByUserID(_ context.Context, userID uint64) ([]domain.UserRole, error) {
	return []domain.UserRole{{UserID: userID, Role: "parent"}}, nil
}

func newTestEngine(t *testing.T, rdb *redis.Client) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	users := &fakeUserRepo{users: map[uint64]*domain.User{
		1: {BaseModel: domain.BaseModel{ID: 1}, Nickname: "家长A", Phone: "13800000001"},
		2: {BaseModel: domain.BaseModel{ID: 2}, Nickname: "家长B", Phone: "13800000002"},
	}}
	svc := service.NewAuthService(nil, users, fakeRoleRepo{}, nil, nil, rdb, nil,
		[]byte(testJWTSecret), 15*time.Minute, 7*24*time.Hour)

	// NewHandler 只接位置参数；本测试只需要 authSvc，其余服务传 nil。
	h := handler.NewHandler(svc,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	engine, err := NewRouter(NewAppHandler(h), nil, JWTSecret(testJWTSecret), nil)
	require.NoError(t, err)
	return engine
}

func doRequest(t *testing.T, engine *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func signAccess(t *testing.T, uid string, ttl time.Duration) string {
	t.Helper()
	token, err := auth.SignAccess([]byte(testJWTSecret), uid, []string{"parent"}, 4, ttl)
	require.NoError(t, err)
	return token
}

// signLegacyAccess 模拟加 jti 之前签发的 token（claims 里没有 jti）。
func signLegacyAccess(t *testing.T, uid string, ttl time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := auth.AccessClaims{
		UserID: uid,
		Roles:  []string{"parent"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "kindergarten-api",
			Subject:   uid,
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	require.NoError(t, err)
	return token
}

func jtiOf(t *testing.T, token string) string {
	t.Helper()
	claims, err := auth.ParseAccess([]byte(testJWTSecret), token)
	require.NoError(t, err)
	require.NotEmpty(t, claims.ID)
	return claims.ID
}

// ---------- 用例 ----------

// 登出 A 后：A 的 access 401、refresh 401；B 的 access/refresh 都不受影响。
func TestLogoutRevokesOnlyItsOwnTokens(t *testing.T) {
	rdb, f := newFakeRedisClient(t)
	engine := newTestEngine(t, rdb)
	ctx := context.Background()

	accessA := signAccess(t, "1", 15*time.Minute)
	accessB := signAccess(t, "2", 15*time.Minute)
	refreshA, refreshB := "refresh-token-a", "refresh-token-b"
	require.NoError(t, auth.SaveRefresh(ctx, rdb, refreshA, "1", 7*24*time.Hour))
	require.NoError(t, auth.SaveRefresh(ctx, rdb, refreshB, "2", 7*24*time.Hour))

	const profilePath = "/api/v1/common/user/profile"
	require.Equal(t, http.StatusOK, doRequest(t, engine, http.MethodGet, profilePath, accessB, "").Code,
		"对照组 B 应能正常访问")

	w := doRequest(t, engine, http.MethodPost, "/api/v1/auth/logout", accessA,
		fmt.Sprintf(`{"refresh_token":%q}`, refreshA))
	require.Equal(t, http.StatusOK, w.Code)

	assert.Equal(t, http.StatusUnauthorized,
		doRequest(t, engine, http.MethodGet, profilePath, accessA, "").Code, "A 的 access 应已失效")
	assert.Equal(t, http.StatusOK,
		doRequest(t, engine, http.MethodGet, profilePath, accessB, "").Code, "B 的 access 不应受影响")

	assert.Equal(t, http.StatusUnauthorized,
		doRequest(t, engine, http.MethodPost, "/api/v1/auth/refresh", "", fmt.Sprintf(`{"refresh_token":%q}`, refreshA)).Code,
		"A 的 refresh 应已失效")

	w = doRequest(t, engine, http.MethodPost, "/api/v1/auth/refresh", "", fmt.Sprintf(`{"refresh_token":%q}`, refreshB))
	require.Equal(t, http.StatusOK, w.Code, "B 的 refresh 不应受影响：%s", w.Body.String())
	var envelope struct {
		Code int `json:"code"`
		Data struct {
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	require.Equal(t, 0, envelope.Code)
	assert.NotEmpty(t, jtiOf(t, envelope.Data.Token), "refresh 轮换出的新 access 必须带 jti")
	assert.NotEqual(t, refreshB, envelope.Data.RefreshToken, "refresh 轮换必须换新 token")

	// 黑名单按 jti 写、TTL 覆盖 token 剩余有效期，且不再使用 token 前缀。
	for _, key := range []string{"token_blacklist:" + jtiOf(t, accessA), "token_blacklist:" + refreshA} {
		ok, ttl := f.exists(key)
		assert.Truef(t, ok, "缺少黑名单键 %s", key)
		assert.Greater(t, ttl, time.Duration(0), "黑名单键 %s 必须带过期时间", key)
	}
	_, accessTTL := f.exists("token_blacklist:" + jtiOf(t, accessA))
	assert.Greater(t, accessTTL, 14*time.Minute, "access 黑名单 TTL 必须覆盖 15 分钟的剩余有效期")
	_, refreshTTL := f.exists("token_blacklist:" + refreshA)
	assert.Greater(t, refreshTTL, 6*24*time.Hour, "refresh 黑名单 TTL 必须覆盖 refresh 的剩余有效期")
	ok, _ := f.exists("token_blacklist:" + jtiOf(t, accessB))
	assert.False(t, ok, "B 的 token 不应被写入黑名单")
}

// 旧 token（无 jti）仍能解析并通过校验；登出对它只是空操作。
func TestLegacyTokenWithoutJTIIsStillAccepted(t *testing.T) {
	rdb, _ := newFakeRedisClient(t)
	engine := newTestEngine(t, rdb)

	legacy := signLegacyAccess(t, "1", 15*time.Minute)
	claims, err := auth.ParseAccess([]byte(testJWTSecret), legacy)
	require.NoError(t, err)
	require.Empty(t, claims.ID)

	assert.Equal(t, http.StatusOK,
		doRequest(t, engine, http.MethodGet, "/api/v1/common/user/profile", legacy, "").Code)
	assert.Equal(t, http.StatusOK,
		doRequest(t, engine, http.MethodPost, "/api/v1/auth/logout", legacy, "{}").Code)
}

// Redis 写失败不能静默成功。
func TestLogoutFailsWhenBlacklistWriteFails(t *testing.T) {
	rdb, f := newFakeRedisClient(t)
	engine := newTestEngine(t, rdb)
	access := signAccess(t, "1", 15*time.Minute)

	f.setFailWrites(true)
	w := doRequest(t, engine, http.MethodPost, "/api/v1/auth/logout", access, "{}")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// Redis 读失败按失败关闭处理：不能放行一枚可能已被登出的 token。
func TestAuthJWTFailsClosedWhenBlacklistReadFails(t *testing.T) {
	rdb, f := newFakeRedisClient(t)
	engine := newTestEngine(t, rdb)
	access := signAccess(t, "1", 15*time.Minute)

	f.setFailReads(true)
	assert.Equal(t, http.StatusServiceUnavailable,
		doRequest(t, engine, http.MethodGet, "/api/v1/common/user/profile", access, "").Code)
	assert.Equal(t, http.StatusServiceUnavailable,
		doRequest(t, engine, http.MethodPost, "/api/v1/auth/refresh", "", `{"refresh_token":"refresh-token-a"}`).Code)
}
