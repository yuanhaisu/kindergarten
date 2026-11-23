package testdb_test

import (
	"os"
	"strings"
	"testing"

	"kindergarten-server/internal/testutil/testdb"
)

// Open 是唯一会真正连库的入口：设置显式测试 DSN 时必须连派生库并跑完迁移，
// 且派生库不能是开发库；未设置时保留现有本地行为（连配置里的开发库）。
func TestOpenConnectsToDerivedDatabaseWhenTestDSNSet(t *testing.T) {
	if os.Getenv(testdb.TestDSNEnv) == "" {
		t.Skipf("未设置 %s：派生库路径由 CI 或显式本地运行覆盖", testdb.TestDSNEnv)
	}
	info := testdb.OpenInfo(t)

	if !info.FromEnv {
		t.Fatalf("%s 已设置时必须走派生库路径", testdb.TestDSNEnv)
	}
	if !strings.Contains(info.Database, "_test_") {
		t.Fatalf("派生库名必须带 _test_ 标记：%q", info.Database)
	}
	if info.DevDatabase != "" && info.Database == info.DevDatabase {
		t.Fatalf("派生库不得等于开发库：%q", info.Database)
	}
	if info.Target == testdb.UnparseableTarget || !strings.HasSuffix(info.Target, "/"+info.Database) {
		t.Fatalf("Target 应打印派生库的脱敏目标：%q", info.Target)
	}
	if !info.DB.Migrator().HasTable("users") {
		t.Fatal("派生库必须执行 goose 迁移（缺少 users 表）")
	}
	var current string
	if err := info.DB.Raw("SELECT DATABASE()").Scan(&current).Error; err != nil {
		t.Fatalf("读取当前库失败：%v", err)
	}
	if current != info.Database {
		t.Fatalf("实际连接的库 %q != 声明的派生库 %q", current, info.Database)
	}
}

func TestOpenFallsBackToConfigWhenTestDSNAbsent(t *testing.T) {
	if os.Getenv(testdb.TestDSNEnv) != "" {
		t.Skipf("已设置 %s，回退路径不适用", testdb.TestDSNEnv)
	}
	info := testdb.OpenInfo(t) // 无配置时内部会 skip，表示现有本地行为（无 DB 不跑）
	if info.FromEnv {
		t.Fatal("未设置显式 DSN 时不应走派生路径")
	}
	if info.Database == "" {
		t.Fatal("回退路径必须解析出目标库名")
	}
	var current string
	if err := info.DB.Raw("SELECT DATABASE()").Scan(&current).Error; err != nil {
		t.Fatalf("读取当前库失败：%v", err)
	}
	if current != info.Database {
		t.Fatalf("回退路径实际连接 %q != 声明 %q", current, info.Database)
	}
}
