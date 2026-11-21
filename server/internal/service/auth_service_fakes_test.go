package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"kindergarten-server/internal/domain"
)

// --- mocks ---
//
// AuthService 的仓储依赖在单测里用内存替身：这些用例只验纯逻辑
// （跨端关联判定 / 账号归属），不需要真实数据库。
//
// #164 关闭微信登录旧路径后，替身只服务 cross_end_identity 用例；
// #173 清理旧身份模型（users.open_id/union_id）时一并删除。

type fakeUserRepo struct {
	byOpenID map[string]*domain.User
	byUnion  map[string]*domain.User
	created  []*domain.User
	updated  []*domain.User
	nextID   uint64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byOpenID: map[string]*domain.User{},
		byUnion:  map[string]*domain.User{},
		nextID:   1,
	}
}

func (f *fakeUserRepo) FindByUnionID(ctx context.Context, unionID string) (*domain.User, error) {
	if unionID == "" {
		return nil, nil
	}
	return f.byUnion[unionID], nil
}

// FindOtherByUnionID / FindOtherByPhone 必须排除自己——同一键下有两条 User，
// 不排除就会命中自己那条、查不到另一端。
func (f *fakeUserRepo) FindOtherByUnionID(ctx context.Context, unionID string, excludeID uint64) (*domain.User, error) {
	if unionID == "" {
		return nil, nil
	}
	for _, u := range f.allByID() {
		if u.UnionID == unionID && u.ID != excludeID {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeUserRepo) FindOtherByPhone(ctx context.Context, phone string, excludeID uint64) (*domain.User, error) {
	if phone == "" {
		return nil, nil
	}
	for _, u := range f.allByID() {
		if u.Phone == phone && u.ID != excludeID {
			return u, nil
		}
	}
	return nil, nil
}

// allByID 按 id 升序返回，避免 map 迭代顺序导致用例不稳定。
func (f *fakeUserRepo) allByID() []*domain.User {
	seen := map[uint64]bool{}
	var out []*domain.User
	for _, u := range f.byOpenID {
		if !seen[u.ID] {
			seen[u.ID] = true
			out = append(out, u)
		}
	}
	for _, u := range f.byUnion {
		if !seen[u.ID] {
			seen[u.ID] = true
			out = append(out, u)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].ID < out[i].ID {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (f *fakeUserRepo) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	for _, u := range f.byOpenID {
		if u.Phone == phone {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeUserRepo) FindByOpenID(ctx context.Context, openID string) (*domain.User, error) {
	return f.byOpenID[openID], nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id uint64) (*domain.User, error) {
	for _, u := range f.byOpenID {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (f *fakeUserRepo) index(u *domain.User) {
	if u.OpenID != "" {
		f.byOpenID[u.OpenID] = u
	}
	if u.UnionID != "" {
		f.byUnion[u.UnionID] = u
	}
}

func (f *fakeUserRepo) Create(ctx context.Context, user *domain.User) error {
	user.ID = f.nextID
	f.nextID++
	f.index(user)
	f.created = append(f.created, user)
	return nil
}

func (f *fakeUserRepo) Update(ctx context.Context, user *domain.User) error {
	f.index(user)
	f.updated = append(f.updated, user)
	return nil
}

func (f *fakeUserRepo) UpdateFields(ctx context.Context, id uint64, fields map[string]any) error {
	return nil
}

type fakeRoleRepo struct{}

func (f *fakeRoleRepo) FindByUserID(ctx context.Context, userID uint64) ([]domain.UserRole, error) {
	return nil, nil
}

// deadRedis 指向不存在的地址：链路若意外需要 Redis 会返回连接错误而不是 panic。
// 本组用例只验关联查询，不签发令牌。
func deadRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: 0, DialTimeout: 50 * time.Millisecond})
}

func newAuthServiceWithRepo(users *fakeUserRepo) *AuthService {
	return &AuthService{
		userRepo:     users,
		userRoleRepo: &fakeRoleRepo{},
		rdb:          deadRedis(),
		secret:       []byte("test-secret"),
		atTTL:        time.Minute,
		rtTTL:        time.Hour,
	}
}
