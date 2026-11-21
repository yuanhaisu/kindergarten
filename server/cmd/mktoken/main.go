// mktoken — 验证辅助工具：为指定手机号+角色签发一枚本地 JWT。
//
// 用途：在无法完成微信登录的自动化环境里，驱动家长端/老师端 API 做
// 本变更的集成验收（TC-NEW-VIS-*、TC-NEW-TGR-*）。签名密钥取自
// configs/app.ini（可用 CONFIG_PATH 覆盖），与本地后端同源。
//
// 家长/老师角色必须绑定小程序客户端（#163）：省略 -client 时按角色从
// clients 注册表取唯一启用中的客户端；也可以用 -client 指定 AppID。
// admin 使用内建 admin_web 作用域（#166），必须能确定唯一园所；其余角色
// （如 doctor）签发不带客户端作用域的旧式凭证。
//
// 仅限本机验证使用；不进入 HTTP 服务，且产物是短期 token。
// 用法：go run ./cmd/mktoken [-client <app_id>] <phone> <role>
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kindergarten-server/internal/auth"
	"kindergarten-server/internal/config"
	"kindergarten-server/internal/domain"
	"kindergarten-server/internal/repository"
	"kindergarten-server/internal/service"
)

func main() {
	clientAppID := flag.String("client", "", "小程序 AppID；省略时按角色查唯一启用中的客户端")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: go run ./cmd/mktoken [-client <app_id>] <phone> <role>")
	}
	flag.Parse()
	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}
	phone, role := flag.Arg(0), flag.Arg(1)

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if cfg.JWT.Secret == "" {
		fmt.Fprintln(os.Stderr, "jwt.secret 为空，无法签发")
		os.Exit(1)
	}

	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db: %v\n", err)
		os.Exit(1)
	}

	// 与密码登录同一口径：按活跃手机号（phone_active）找账号，软删账号不命中。
	user, err := repository.NewUserRepo(db).FindByPhone(context.Background(), phone)
	if err != nil {
		fmt.Fprintf(os.Stderr, "find user %s: %v\n", phone, err)
		os.Exit(1)
	}
	if user == nil {
		fmt.Fprintf(os.Stderr, "user %s not found\n", phone)
		os.Exit(1)
	}

	var roles []domain.UserRole
	if err := db.Where("user_id = ?", user.ID).Find(&roles).Error; err != nil {
		fmt.Fprintf(os.Stderr, "load roles: %v\n", err)
		os.Exit(1)
	}

	roleNames := make([]string, 0, len(roles))
	var kgID uint64
	for _, r := range roles {
		// 优先取本次请求角色绑定的园所；否则回退到任意一个园所（parent/teacher
		// 可能只挂了角色而没有园所，token 允许 kg=0）。
		if r.Role == role && r.KindergartenID != nil {
			kgID = *r.KindergartenID
		}
		roleNames = append(roleNames, r.Role)
	}
	if kgID == 0 {
		for _, r := range roles {
			if r.KindergartenID != nil {
				kgID = *r.KindergartenID
				break
			}
		}
	}

	target := role
	found := false
	for _, r := range roleNames {
		if r == role {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "user %s has roles %v, not %s\n", phone, roleNames, role)
		os.Exit(1)
	}

	scope, err := resolveClientScope(db, target, *clientAppID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve client: %v\n", err)
		os.Exit(1)
	}

	ttl := time.Duration(cfg.JWT.AccessTTLMin) * time.Minute
	if ttl <= 0 {
		ttl = time.Hour
	}
	var token string
	switch {
	case scope.AppID != "":
		fmt.Fprintf(os.Stderr, "client: %s (scope=%s/%d)\n", scope.AppID, scope.Kind, scope.ID)
		token, err = auth.SignClientAccess([]byte(cfg.JWT.Secret), fmt.Sprint(user.ID), []string{target}, kgID, scope, ttl)
	case target == "admin":
		// 管理端凭证必须带内建 admin_web 作用域且恰好绑定一个园所（#166）。
		token, err = auth.SignAdminAccess([]byte(cfg.JWT.Secret), fmt.Sprint(user.ID), []string{target}, kgID, ttl)
	default:
		token, err = auth.SignAccess([]byte(cfg.JWT.Secret), fmt.Sprint(user.ID), []string{target}, kgID, ttl)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "sign: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(token)
}

// resolveClientScope 决定这枚 token 绑定哪个小程序客户端。
// 非小程序角色返回零值，走旧式签发路径。
func resolveClientScope(db *gorm.DB, role, requestedAppID string) (auth.ClientScope, error) {
	if role != domain.ClientRoleParent && role != domain.ClientRoleTeacher {
		if requestedAppID != "" {
			return auth.ClientScope{}, fmt.Errorf("-client 只适用于 %s / %s 角色", domain.ClientRoleParent, domain.ClientRoleTeacher)
		}
		return auth.ClientScope{}, nil
	}

	clientSvc := service.NewClientService(repository.NewClientRepo(db))
	ctx := context.Background()

	var client *domain.Client
	var err error
	if requestedAppID != "" {
		client, err = clientSvc.ResolveEnabledByAppID(ctx, requestedAppID)
	} else {
		client, err = clientSvc.FindEnabledForRole(ctx, role)
	}
	if err != nil {
		return auth.ClientScope{}, err
	}
	if client.Role != role {
		return auth.ClientScope{}, fmt.Errorf("客户端 %s 的角色是 %s，与请求角色 %s 不一致", client.AppID, client.Role, role)
	}
	return auth.ClientScope{AppID: client.AppID, Kind: client.ScopeKind, ID: client.ScopeID}, nil
}
