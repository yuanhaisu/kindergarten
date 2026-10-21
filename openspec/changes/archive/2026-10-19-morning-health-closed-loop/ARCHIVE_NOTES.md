# Archive Notes — morning-health-closed-loop

归档日期：2026-10-19

## 实际触点与设计阶段的差异

- **契约包络修正（验证中发现）**：家长晨检列表最初返回裸数组，契约与页面按 `data.list` 读取。UI 实测暴露后，`ParentHealthRecords` 改为 `gin.H{"list": data}`，与 `api-parent/12-health.md` 和页面一致。设计阶段未预见此问题。
- **照片引用表迁移**：按 design D3 执行 `docs/migrations/2026-10-18-health-check-media-file-ref.sql`；回填时发现 `files.file_url` 与 `health_check_media.file_url` 排序规则不一致，SQL 增加 `COLLATE utf8mb4_unicode_ci`。已用 `go run ./cmd/dbcheck health_check_media` 核实。
- **原生相机选择器**：CDP `Page.setInterceptFileChooserDialog` 无法拦截（弹出的是 Windows `#32770` 对话框）。新增 `_verify/select-file-dialog.ps1`（Win32 枚举 + SendKeys）完成自动化，并实测上传→关联→预览→提交全链路。
- **race detector 工具链**：本机 `C:\MinGW` 是 32 位，`-race` 报 `64-bit mode not compiled in`。安装 WinLibs 64 位 GCC 并设置用户级 `CC` 后，`make -C server check` 全量 `-race` 通过。
- 其余触点与 `design.md` 一致：domain/repository/service/handler/router/contract/teacher/parent/admin 均按设计落地；Casbin 通配覆盖新端点，无需改策略。

## 验证摘要

- 37/37 tasks 完成；`openspec validate --strict` 通过。
- `go vet` / `golangci-lint` 0 issues / `make -C server check`（含 `-race`）通过。
- 真实接口与 UI 验证：家长查看（含未绑定 403）、医生提交（需观察/劝返+隔离）、通知落库、照片上传/关联/回读、跨租户 404、家长端幼儿档案渲染与切孩。
- 测试数据：本地库保留 `morning_health_checks` 995002/995004/995006/995008/995010/995012 等验证记录。

## 后续

- 体温策略作为独立变更 `health-temperature-policy` 另行推进。
- 一键健康 / 单卡步进属于交互重构，需现场计时与安全评估后再立项。
