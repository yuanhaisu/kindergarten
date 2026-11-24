// Package testdb 为需要真实 MySQL 的测试提供统一的测试库入口与安全护栏。
//
// 设计目标（#207 expand 阶段）：
//
//   - 显式测试 DSN：设置 KDG_TEST_MYSQL_DSN 后，测试只连由它派生的包级测试库，
//     不读取、不连接开发库的数据。
//   - 包级派生：派生库名带「分支 + 测试包」后缀，`go test ./...` 的并行包
//     各写各的库，避免整表清理类用例互相踩踏。
//   - 失败关闭：显式 DSN 无法确认是测试库、或与开发库同名时直接失败，
//     报错只打印脱敏后的 host:port/db，绝不回显账号口令。
//   - 门禁开关：设置 KDG_TEST_REQUIRE_MYSQL 后，「没有可用测试库」从跳过改为失败，
//     CI 不会因为漏配 DSN 而把整个包静默记为通过。
//   - 兼容现状：未设置 KDG_TEST_MYSQL_DSN 时保持现有本地行为（连 configs/app.ini
//     的开发库）；开发库回退由后续票（#208）移除。
//
// 本包不会 DROP 任何数据库；测试必须按唯一键在 Cleanup 里硬删自己写入的行。
package testdb

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	mysqldrv "github.com/go-sql-driver/mysql"
)

// TestDSNEnv 显式指定测试库的 DSN。设置后测试只连由它派生的包级测试库；
// 未设置时保持现有本地行为（连 configs/app.ini 的开发库）。
const TestDSNEnv = "KDG_TEST_MYSQL_DSN"

// RequireDBEnv 设为真值时，测试库不可用直接失败而不是跳过。CI 必须显式设置：
// 缺 MySQL 或漏配 DSN 时让门禁变红，而不是把「没跑」当成「通过」。
const RequireDBEnv = "KDG_TEST_REQUIRE_MYSQL"

// RequireMySQL 读取 RequireDBEnv。值写错时按「要求」处理：宁可失败关闭，
// 也不能因为拼错开关而静默跳过。authharness 复用同一判定，避免两套 helper
// 对同一个开关给出不同答案。
func RequireMySQL() bool {
	v := strings.TrimSpace(os.Getenv(RequireDBEnv))
	if v == "" {
		return false
	}
	required, err := strconv.ParseBool(v)
	if err != nil {
		return true
	}
	return required
}

// UnparseableTarget 是无法解析成 tcp host:port/db 时的固定兜底文案。
// 固定文案而不是回显原文，是为了杜绝对畸形 DSN 的凭据泄漏。
const UnparseableTarget = "无法解析 DSN（仅支持 tcp 形式）"

// Target 返回脱敏后的目标库 host:port/db，永不包含账号与口令。
// 只有 tcp 形式且带库名的 DSN 才可能返回具体值，其余一律返回 UnparseableTarget。
func Target(dsn string) string {
	cfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		return UnparseableTarget
	}
	if cfg.Net != "tcp" || cfg.Addr == "" || cfg.DBName == "" {
		return UnparseableTarget
	}
	return cfg.Addr + "/" + cfg.DBName
}

// Guard 校验显式测试 DSN 是否安全，失败时返回只含脱敏目标的错误。
//
// 规则（任一不满足即拒绝，fail closed）：
//   - 必须可解析为 tcp DSN，且带库名；
//   - 库名必须含 test（大小写不敏感）——无法确认是测试库就不跑；
//   - 提供了 devDSN 时，库名不得与开发库相同。
//
// 刻意不包装 ParseDSN 的原始错误：畸形 DSN 的分片可能落进解析错误的 %q 里，
// 从而把口令带进日志。错误正文只使用 Target 的脱敏结果。
func Guard(explicitDSN, devDSN string) error {
	if strings.TrimSpace(explicitDSN) == "" {
		return fmt.Errorf("%s 为空；请指向一个独立测试库", TestDSNEnv)
	}
	target := Target(explicitDSN)

	cfg, err := mysqldrv.ParseDSN(explicitDSN)
	if err != nil {
		return fmt.Errorf("%s 无法解析，拒绝运行（目标：%s）", TestDSNEnv, target)
	}
	if cfg.DBName == "" {
		return fmt.Errorf("%s 必须包含数据库名（目标：%s）", TestDSNEnv, target)
	}
	if !strings.Contains(strings.ToLower(cfg.DBName), "test") {
		return fmt.Errorf("%s 的库名 %q 不含 test，无法确认是测试库；为避免误写开发/生产数据，拒绝运行（目标：%s）",
			TestDSNEnv, cfg.DBName, target)
	}

	if devDSN != "" {
		devCfg, err := mysqldrv.ParseDSN(devDSN)
		if err != nil {
			return fmt.Errorf("无法解析开发库 DSN 用于护栏比对，拒绝运行（%s 目标：%s）", TestDSNEnv, target)
		}
		if devCfg.DBName != "" && devCfg.DBName == cfg.DBName {
			return fmt.Errorf("%s 指向开发库 %q；测试会写入并硬删数据，必须换独立测试库（目标：%s）",
				TestDSNEnv, cfg.DBName, target)
		}
	}
	return nil
}
