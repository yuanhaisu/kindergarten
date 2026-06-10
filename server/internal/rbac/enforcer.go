package rbac

import (
	"kindergarten-server/internal/config"

	"github.com/casbin/casbin/v2"
)

func NewEnforcer(cfg *config.CasbinConfig) (*casbin.Enforcer, error) {
	if !cfg.Enable {
		return nil, nil
	}
	return casbin.NewEnforcer(cfg.ModelPath, cfg.PolicyPath)
}
