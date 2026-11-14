package main

// seedbase 幂等初始化平台基础数据：字典、膳食分类/DRIs/食物/模板、园长角色。
//
// 这些种子原先挂在应用启动路径上，每次重启都会执行；现在改为显式命令：
//
//	CONFIG_PATH=configs/app.ini go run ./cmd/seedbase
//
// 重复执行不会产生重复数据。首次部署顺序：make migrate -> make seedbase。

import (
	"context"
	"log"

	"kindergarten-server/cmd/internal/seedutil"
	"kindergarten-server/internal/app"
	"kindergarten-server/internal/service"
)

func main() {
	db := seedutil.Open()

	app.SeedDictsAfterMigration(db)
	app.SeedDietAfterMigration(db)
	if err := service.SeedPrincipalRolesForAll(context.Background(), db); err != nil {
		log.Fatalf("seed principal roles: %v", err)
	}

	log.Println("seedbase completed")
}
