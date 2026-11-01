# OS-ALB-3 · 对抗性实现审查

- verdict: **PASS**
- score: 90/100
- 源码覆盖: `server/internal/service/parent_portal_service.go`、`teacher_portal_service.go`、`parent_settings_test.go`、`teacher_class_student_contact_test.go`（新增）
- 审查范围: diff + untracked + 契约文档 + 测试目录 + 受影响调用链（handler 未改，透传）
- 基线: `0a6e9bf`（HEAD 未变，无 commit）

## 验证证据（backend-only，无 DevTools）

- 构建: `cd server && go build ./...` → exit 0
- Vet: `cd server && go vet ./internal/...` → exit 0
- 测试: `cd server && go test ./internal/service/ -run 'TestParentSettings|TestTeacherClassStudentParents' -count=1 -timeout 600s` → exit 0，`ok ... 20.270s`；9 个 `TestParentSettings_*` 顶层（含 8 子用例）+ 3 个 `TestTeacherClassStudentParents*` 全 PASS，无 Skip（真实 MySQL）
- Secret scan: `node scripts/check-secrets.mjs --files <7 files>` → `Secret scan: 0 findings`，exit 0
- 范围: 内容 diff 恰为 manifest 7 文件；无 album / parent-album / admin / 前端 / domain / router 改动；HEAD == base
- 空态: 无绑定幼儿返回 `parents: []`（非 null）
- 错误态: 非法分组/键/类型 → `ErrParentInvalidParams` → handler 422；跨班/跨园 → 404
- DevTools / 截图: 不适用（brief 明确禁止前端与 DevTools；本切片零 UI 改动）

## P1 · 阻塞问题 (0)

无。

## P2 · 建议 (5)

1. **[test-gap] upsert 新列缺真正的 conflict 断言** — `UpdatePersists` 是 INSERT；两次 conflict 写入的新列前后值相同，即使 `DoUpdates` 漏列也会通过。源码确认 `allow_growth_share`/`allow_class_share`/`allow_external_share` 均在列表，建议补「第二次写入翻转新列并重读」用例。
2. **[docs] 过时注释** — `parent_portal_service.go:3662` 仍写「8 个开关」，现为 9 个。
3. **[evidence] 计数偏差** — worker 称 10 个 `TestParentSettings`，实际 9 个顶层 + 8 子用例；测试结果不受影响。
4. **[residual-risk] 家长设置页过渡态** — `parent/src/pages/settings/index.vue` 仍读旧键 `allow_growth_share`；新后端 GET 不再返回，页面回落 false 显示，切换以旧键写入映射到班内共享。属 8.4 收口范围，后端行为为 brief 明令，但上线窗口内 UI 有误导。
5. **[pre-existing/security] 学生详情 PII** — handler 原样序列化 `domain.Student`，新契约示例明确含 `id_card_number`/`address`；基线已如此，建议后续投影收敛。

## 残余风险

- 迁移 SQL（`docs/migrations/class-album-parent-sharing.sql`）必须先于新后端执行；测试构造迁移后形态，不执行脚本本身。
- 设备端/UI 展示属 7.3/8.4，未在本切片验证范围。
