package app

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// readinessTimeout 限制一次就绪检查的总时长，避免探针被卡住的依赖拖死。
const readinessTimeout = 2 * time.Second

type readinessCheck struct {
	name  string
	check func(context.Context) error
}

type readiness struct {
	checks []readinessCheck
}

// WithInfra 把基础设施依赖挂到 AppHandler 上，供 /ready 检查。
// 未装配依赖时（单元测试）保持 ready=nil，/ready 直接视为就绪。
func (h *AppHandler) WithInfra(db *gorm.DB, rdb *redis.Client) *AppHandler {
	checks := make([]readinessCheck, 0, 2)
	if db != nil {
		checks = append(checks, readinessCheck{
			name: "mysql",
			check: func(ctx context.Context) error {
				sqlDB, err := db.DB()
				if err != nil {
					return err
				}
				return sqlDB.PingContext(ctx)
			},
		})
	}
	if rdb != nil {
		checks = append(checks, readinessCheck{
			name: "redis",
			check: func(ctx context.Context) error {
				return rdb.Ping(ctx).Err()
			},
		})
	}
	return h.WithReadyChecks(checks...)
}

// WithReadyChecks 允许测试注入就绪检查，避免依赖真实 MySQL/Redis。
func (h *AppHandler) WithReadyChecks(checks ...readinessCheck) *AppHandler {
	h.ready = &readiness{checks: checks}
	return h
}

// Ready 是给负载均衡/发布流水线使用的就绪探针；/health 只表示进程存活。
// 只返回依赖名与 ok/down，不暴露连接串或错误详情。
func (h *AppHandler) Ready(c *gin.Context) {
	if h.ready == nil {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
	defer cancel()

	status := http.StatusOK
	result := gin.H{"status": "ready"}
	checks := gin.H{}
	for _, check := range h.ready.checks {
		if err := check.check(ctx); err != nil {
			status = http.StatusServiceUnavailable
			result["status"] = "not_ready"
			checks[check.name] = "down"
			continue
		}
		checks[check.name] = "ok"
	}
	result["checks"] = checks
	c.JSON(status, result)
}
