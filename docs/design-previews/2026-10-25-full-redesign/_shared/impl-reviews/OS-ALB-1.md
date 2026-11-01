# OS-ALB-1 · 对抗性实现审查

- verdict: PASS
- score: 90/100
- 源码覆盖: `server/internal/domain/{album,parent_setting,student}.go`、`docs/migrations/class-album-parent-sharing.sql`
- 审查范围: diff + untracked + GORM tag/索引命名推导 + 迁移/回滚 SQL + dev DB 独立只读核对 + AutoMigrate 影响面
- 基线: 703ca04 → 703ca04（无 commit）

## 验证证据

- 构建: `cd server && go build ./...` → exit 0
- Vet: `go vet ./internal/domain/` → exit 0；`go test ./internal/domain/` → exit 0（no test files）
- Token: 不适用（无前端改动）
- Secret scan: `node scripts/check-secrets.mjs --files <4 files>` → exit 0，0 findings
- 产物密钥扫描: `.claude/runs/OS-ALB-1/**` 13 个文件按 app.ini 的 DSN/账号/口令/host 字面量与通用 token 正则扫描 → 0 命中；该目录已 gitignore
- 正常态: MySQL 8.4.0；`dbcheck` + `information_schema` 只读查询确认 5 张表列/索引/唯一键与 design D2 一致
- 空态/错误态: 不适用（后端 schema/迁移切片，无页面/路由/接口改动）
- 交互: 不适用；迁移幂等复跑 2 次（审查者独立执行）→ 42 statements, 0 failures, 0 行变更
- Console/截图/路由: 不适用；未申请 DevTools 锁，8080 未监听、未启动

## DB 事实

| 表 | 行数 | 结构 |
| --- | --- | --- |
| albums | 0 | 复合索引 4 列列序正确；status/default draft、published_at、published_by 类型与默认值正确 |
| album_photos | 0 | file_id 可空 + 索引 |
| album_students | 0 | uk_album_student(album_id, student_id) + student_id 索引 |
| album_photo_students | 0 | uk_album_photo_student(photo_id, student_id) + student_id 索引 |
| parent_settings | 1 | 旧值 0 保留为 allow_class_share=0；allow_external_share=0；DEFAULT 1/0 |
| students | 28 | status_changed_at datetime(3) NULL |

- albums published=0；album_photos 回填 0+0=0；files 重复 URL=0
- 其它表行数迁移前后一致：students 28 / files 24 / growth_media 21 / growth_log_students 9
- before 快照无新列/新表（1054/1146），after 与实时查询一致

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (4)

### 1. [rollback/AutoMigrate] 保留新列的回滚路径不安全
- 证据: `server/internal/app/provider.go:87` 注册 `&domain.ParentSetting{}`；`parent_setting.go:28-29` 无 `default:` tag；GORM v1.31.1 `migrator.go:545-550` 会在下次启动 DROP DEFAULT；旧二进制 INSERT 缺这两列时在 STRICT_TRANS_TABLES 下报 1364。`class-album-parent-sharing.sql:209` / `rollback.sql:4` 的「旧代码不认识新列也能正常运行」对插入不成立。
- 建议: 修订回滚说明；回滚前执行删列或补回默认值，并在首次新代码启动后复核 information_schema。

### 2. [transitional data] 4.1 之前新建的设置行会把班内共享写成 false
- 证据: `parent_portal_service.go:3822-3829` 建行只写旧字段/旧列，新列零值被 GORM 显式写入，列默认值不生效；D6 要求新家长默认开启班内共享。
- 建议: 4.1 显式写默认值，并为该窗口产生的行提供识别/回填策略或调整上线顺序。

### 3. [evidence gap] 历史数据路径未在真实行上跑过
- 证据: dev albums/album_photos 均 0 行；转 draft 与 file_id 回填影响 0 行；无重复 file_url。worker 已披露。
- 建议: 在数据副本上造含重复 URL / 软删 file 的样本跑一遍再上生产。

### 4. [process/secrets] worker 曾把 dev DSN 打进会话输出，产物未泄漏
- 证据: 产物扫描 0 命中、check-secrets 0 findings；worker 已主动承认 grep 命中 app.ini dsn 行。
- 建议: 若转录会分享或该口令跨环境复用则轮换；保持 host/port/db-only 输出。

## 残留风险

- 新表未注册进 `provider.go` AutoMigrate（清单外）；后续切片应登记，否则新库/单测缺表。
- `parent_setting.go:25-27` 注释「API 不再返回这个键」在 HEAD 不成立（`parent_portal_service.go:3725` 仍返回）。
- 生产 ADD INDEX 未显式 `ALGORITHM=INPLACE, LOCK=NONE`，大表需按 DBA 口径评估。

## 结论

PASS。4 个清单文件与 design D2/迁移计划一致；build/vet/secrets 通过；DB 证据与快照交叉一致；幂等复跑 0 失败 0 行变更。P2 第 1、2 条须在 4.1 与回滚预案中闭环。
