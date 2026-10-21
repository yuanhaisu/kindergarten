// facecheck — 腾讯云人脸识别接入验证与质量实测（一次性/运维辅助命令）。
//
// 阶段一（无参数）：验证凭据与 IAI 服务可用性，创建/确认测试人员库（幂等）。
//
//	go run ./cmd/facecheck
//
// 阶段二（带照片目录）：底库注册 + 1:N 搜索端到端实测（儿童脸质量评估）。
//
//	go run ./cmd/facecheck <照片目录> [--kg 999] [--group facecheck_test]
//
// 目录内文件命名约定：
//
//	<姓名>_<序号>.jpg        该姓名的底库照片（可多张，如 李明_1.jpg / 李明_2.jpg）
//	<任意>_probe.jpg         被测照片（如 合影_probe.jpg），对该园底库做 1:N 搜索
//
// 输出：每张 probe 的命中列表（学生名 + 分数），以及分数分布统计（供阈值校准）。
// 密钥来源：configs/app.ini 的 [face] 段（可用 FACE_SECRET_ID/FACE_SECRET_KEY 覆盖）。
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"kindergarten-server/internal/ai/face"
	"kindergarten-server/internal/config"
)

func main() {
	flagKg := flag.Int64("kg", 999, "测试用园所 ID（隔离在测试 Group 内）")
	flagGroup := flag.String("group", "facecheck_test", "覆盖园所分组名（直接指定 GroupId 时用）")
	flag.Parse()
	args := flag.Args()

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/app.ini"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal("load config: ", err)
	}
	if v := os.Getenv("FACE_SECRET_ID"); v != "" {
		cfg.Face.SecretID = v
	}
	if v := os.Getenv("FACE_SECRET_KEY"); v != "" {
		cfg.Face.SecretKey = v
	}
	if cfg.Face.SecretID == "" || cfg.Face.SecretKey == "" {
		log.Fatal("[face] secret_id/secret_key 未配置")
	}

	provider, err := face.New(cfg.Face)
	if err != nil {
		log.Fatal("new provider: ", err)
	}
	fmt.Printf("provider=%s region=%s\n", provider.Name(), cfg.Face.Region)

	if len(args) == 0 {
		smoke(provider, *flagKg)
		return
	}
	_ = flagGroup // 预留：直连指定 GroupId 的调试路径
	runDir(provider, *flagKg, args[0])
}

// smoke 阶段一：幂等建库确认凭据与服务可用。
func smoke(p face.Provider, kgID int64) {
	ctx := context.Background()
	// 以“测试学生 1 人”走一遍 EnsurePerson 的前置建库路径（不带图 → 用伪造 1x1 PNG 触发建库）
	// 简化：直接调 EnsurePerson 需要真实图；此处用 Search 触发服务调用（空图返回业务错误亦可证明连通）。
	_, err := p.Search(ctx, uint64(kgID), []byte("\x89PNG\r\n\x1a\nfacecheck-probe"))
	if err != nil && !strings.Contains(err.Error(), "NoFace") {
		// 无人脸是预期内业务结果；其它错误（鉴权/网络）才是失败
		log.Fatalf("✗ IAI 调用失败: %v", err)
	}
	fmt.Println("✓ 凭据有效：IAI 服务可用（Search 链路连通）")
	fmt.Println("  提示：完整验证请提供照片目录（阶段二）")
}

type person struct {
	name  string
	files []string
}

// runDir 阶段二：注册底库 → 搜索 probe → 打印命中与分数分布。
func runDir(p face.Provider, kgID int64, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal("read dir: ", err)
	}
	people := map[string]*person{}
	var probes []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue
		}
		base := strings.TrimSuffix(name, filepath.Ext(name))
		if strings.HasSuffix(base, "_probe") {
			probes = append(probes, filepath.Join(dir, name))
			continue
		}
		who := base
		if i := strings.Index(who, "_"); i > 0 {
			who = who[:i]
		}
		pp := people[who]
		if pp == nil {
			pp = &person{name: who}
			people[who] = pp
		}
		pp.files = append(pp.files, filepath.Join(dir, name))
	}
	if len(people) == 0 {
		log.Fatal("目录内未找到底库照片（命名：<姓名>_<序号>.jpg）")
	}

	ctx := context.Background()
	// 临时分配学生 ID（姓名按字典序稳定映射，便于报告复现）
	names := make([]string, 0, len(people))
	for n := range people {
		names = append(names, n)
	}
	sort.Strings(names)
	sidByName := map[string]uint64{}
	fmt.Println("── 底库注册 ──")
	for i, n := range names {
		sid := uint64(i + 1)
		sidByName[n] = sid
		var imgs [][]byte
		for _, f := range people[n].files {
			b, err := os.ReadFile(f)
			if err != nil {
				log.Fatalf("read %s: %v", f, err)
			}
			imgs = append(imgs, b)
		}
		ref, err := p.EnsurePerson(ctx, uint64(kgID), sid, imgs)
		if err != nil {
			fmt.Printf("  ✗ %s (stu=%d, %d 张): %v\n", n, sid, len(imgs), err)
			continue
		}
		fmt.Printf("  ✓ %s (stu=%d, %d 张) → %s\n", n, sid, len(imgs), ref)
	}

	fmt.Println("── 识别实测 ──")
	var allScores []float64
	for _, probe := range probes {
		b, err := os.ReadFile(probe)
		if err != nil {
			log.Fatalf("read %s: %v", probe, err)
		}
		res, err := p.Search(ctx, uint64(kgID), b)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", filepath.Base(probe), err)
			continue
		}
		matches := res.Matches
		fmt.Printf("\n▸ %s （%d 张人脸 / %d 条命中）\n", filepath.Base(probe), res.FaceNum, len(matches))
		for _, m := range matches {
			allScores = append(allScores, m.Score)
			name := "?"
			for n, sid := range sidByName {
				if sid == m.StudentID {
					name = n
					break
				}
			}
			fmt.Printf("    %-10s  stu=%d  score=%.2f\n", name, m.StudentID, m.Score)
		}
		if len(matches) == 0 {
			fmt.Println("    （无命中）")
		}
	}

	if len(allScores) > 0 {
		sort.Float64s(allScores)
		fmt.Printf("\n── 分数分布 ──\n  命中数=%d  min=%.2f  median=%.2f  max=%.2f\n",
			len(allScores), allScores[0], allScores[len(allScores)/2], allScores[len(allScores)-1])
		fmt.Println("  提示：结合人工核对结果校准 [face].suggest_high / suggest_low")
	}
}
