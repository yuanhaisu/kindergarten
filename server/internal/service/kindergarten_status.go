package service

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
)

const kindergartenStatusCacheTTL = 60 * time.Second

// KindergartenStatusChecker 供鉴权与登录链路判断园所是否被停用。
// 声明在消费方（AuthService / 中间件），便于单测注入桩。
type KindergartenStatusChecker interface {
	Suspended(ctx context.Context, kindergartenID uint64) (bool, error)
}

// KindergartenStatusService 园所状态查询，Redis 缓存 60 秒；
// 平台方启停园所时主动失效，保证「停用立即生效」。
type KindergartenStatusService struct {
	kgs *repository.KindergartenRepo
	rdb *redis.Client
}

func NewKindergartenStatusService(kgs *repository.KindergartenRepo, rdb *redis.Client) *KindergartenStatusService {
	return &KindergartenStatusService{kgs: kgs, rdb: rdb}
}

func kindergartenStatusKey(id uint64) string {
	return "kg:suspended:" + strconv.FormatUint(id, 10)
}

func (s *KindergartenStatusService) Suspended(ctx context.Context, kindergartenID uint64) (bool, error) {
	if kindergartenID == 0 {
		return false, nil
	}
	key := kindergartenStatusKey(kindergartenID)
	if s.rdb != nil {
		if v, err := s.rdb.Get(ctx, key).Result(); err == nil {
			return v == "1", nil
		}
	}
	kg, err := s.kgs.FindByID(ctx, kindergartenID)
	if err != nil {
		return false, err
	}
	suspended := kg != nil && kg.Status == domain.KindergartenStatusSuspended
	if s.rdb != nil {
		value := "0"
		if suspended {
			value = "1"
		}
		_ = s.rdb.Set(ctx, key, value, kindergartenStatusCacheTTL).Err()
	}
	return suspended, nil
}

// Invalidate 平台方启停园所后调用，让下一次请求立刻读到新状态。
func (s *KindergartenStatusService) Invalidate(ctx context.Context, kindergartenID uint64) error {
	if s.rdb == nil || kindergartenID == 0 {
		return nil
	}
	return s.rdb.Del(ctx, kindergartenStatusKey(kindergartenID)).Err()
}
