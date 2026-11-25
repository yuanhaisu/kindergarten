package testdb

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

const (
	// testMarker 是派生库名里必须保留的标记：护栏据此确认目标是测试库。
	testMarker = "_test_"
	// maxBaseLen 给哈希后缀与标记留出空间，确保派生名不超过 MySQL 的 64 字符上限。
	maxBaseLen = 40
	// unknownBranch 在 HEAD 读不到时作为分支占位，仍然产出合法的派生名。
	unknownBranch = "unknown"
)

// deriveDatabaseName 从基础测试库名派生「分支 + 测试包」隔离的库名。
//
// 基础库名保留原样（仅做字符净化与长度截断），去掉已有的 _test 后缀避免
// 出现 kindergarten_test_test_xxx；后缀是 branch||package 的 8 位哈希，
// 分支名本身不进入库名，避免斜杠等非法字符。
func deriveDatabaseName(baseDB, branch, pkg string) string {
	base := sanitizeDBName(baseDB)
	if base == "" {
		base = "kindergarten"
	}
	if strings.HasSuffix(strings.ToLower(base), "_test") && len(base) >= len("_test") {
		base = base[:len(base)-len("_test")]
	}
	if base == "" {
		base = "kindergarten"
	}
	if len(base) > maxBaseLen {
		base = base[:maxBaseLen]
	}
	return base + testMarker + shortHash(branch+"|"+pkg)
}

// packageScope 返回当前测试二进制对应的包名。go test 每个包编译出独立的
// <pkg>.test[.exe]，因此同一进程内恒定、不同测试包必然不同。
func packageScope() string {
	name := filepath.Base(os.Args[0])
	name = strings.TrimSuffix(name, ".exe")
	name = strings.TrimSuffix(name, ".test")
	return sanitizeDBName(name)
}

// currentBranch 读取仓库 HEAD 的分支名；worktree（.git 文件指向 gitdir）同样适用。
// 读取失败时返回 unknown：派生名仍然稳定，只是无法按分支隔离。
func currentBranch(repoRoot string) string {
	headPath, err := findGitHEAD(repoRoot)
	if err != nil {
		return unknownBranch
	}
	b, err := os.ReadFile(headPath)
	if err != nil {
		return unknownBranch
	}
	v := strings.TrimSpace(string(b))
	v = strings.TrimPrefix(v, "ref: refs/heads/")
	if v == "" {
		return unknownBranch
	}
	return v
}

// findGitHEAD 兼容普通仓库（.git 目录）与 worktree（.git 文件指向 gitdir）。
func findGitHEAD(start string) (string, error) {
	dir := start
	for {
		gitPath := filepath.Join(dir, ".git")
		info, err := os.Stat(gitPath)
		if err == nil {
			if info.IsDir() {
				return filepath.Join(gitPath, "HEAD"), nil
			}
			b, err := os.ReadFile(gitPath)
			if err != nil {
				return "", err
			}
			line := strings.TrimSpace(string(b))
			if rest, ok := strings.CutPrefix(line, "gitdir:"); ok {
				gitdir := strings.TrimSpace(rest)
				if !filepath.IsAbs(gitdir) {
					gitdir = filepath.Join(dir, gitdir)
				}
				return filepath.Join(gitdir, "HEAD"), nil
			}
			return "", &headParseError{path: gitPath}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", &headParseError{path: gitPath}
		}
		dir = parent
	}
}

type headParseError struct{ path string }

func (e *headParseError) Error() string { return "无法解析 " + e.path }

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])[:8]
}

// sanitizeDBName 只保留 MySQL 标识符可用字符，其余替换为下划线。
func sanitizeDBName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
