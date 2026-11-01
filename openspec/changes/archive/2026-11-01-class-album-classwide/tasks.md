# Tasks

## 1. 数据与后端

- [x] 1.1 `KindergartenAppSetting` 增加 `ClassAlbumClasswideEnabled`（默认 false）；迁移 SQL + `dbcheck kindergarten_app_settings` 核对；`go build ./...` 通过
- [x] 1.2 `SettingsService` 读写 `album.classwide_enabled`；`GET/PUT /admin/settings` 契约同步；单测覆盖默认 false/开启/关闭/审计
- [x] 1.3 `ParentPortalService` 相册列表/详情支持 `scope=mine|all`：默认 mine；all 仅在开关开启时；关闭 403、非法 422、跨班 404；单测覆盖两种 scope、opt-out、自己的孩子、多孩
- [x] 1.4 列表响应增加 `classwide_enabled`；详情/契约同步；确认无 N+1
- [x] 1.5 更新 `backend/api/api-parent/13-class-albums.md`、`api-admin/18-settings.md`；登记 `TC-NEW-CAW-0001..0011`
- [x] 1.6 定向测试：`go test ./internal/service/ -run 'TestParentClassAlbum|TestSettings.*Classwide|TestClasswide' -count=1 -timeout 600s`；`go vet ./internal/...`

## 2. 管理端 [P]

- [x] 2.1 `admin/api/settings.ts` 类型增加 `album.classwide_enabled`；设置页增加开关与说明，保存调用现有 PUT；`npm run build` exit 0
- [x] 2.2 无 settings 权限时开关不可见/不可保存；复用现有权限码；无头 Chrome 实测读写与刷新
- [x] 2.3 不新增 token/硬编码；`check-tokens admin` 基线不增

## 3. 家长端 [P]

- [x] 3.1 `parent/src/api/parent-class-album.js` 支持 `scope`；列表页读取 `classwide_enabled`，开启时渲染「我的孩子 / 全班」分段
- [x] 3.2 切换 scope 重置分页并重新请求；详情链接携带 scope；开关关闭不显示全班入口
- [x] 3.3 空态/错误态沿用 Phase 1；全班视图空态文案；token 基线 3 不增
- [x] 3.4 `build:mp-weixin` exit 0；DevTools 真实点击切换 + 真实后端列表/详情

## 4. 集成验证

- [x] 4.1 真实后端 E2E：园所开关关闭 → 家长 only mine；管理员开启 → 家长切全班 → opt-out 合影对其他家长隐藏、自己孩子可见 → 管理员关闭 → 全班入口消失/403
- [x] 4.2 数据清理：测试设置恢复原值、测试相册/照片/关联硬删，残留 0；审计按测试 ID 清理
- [x] 4.3 三端构建、token 基线、secret 0；对照 specs 逐条勾选，更新 PROGRESS；archive `class-album-classwide`
