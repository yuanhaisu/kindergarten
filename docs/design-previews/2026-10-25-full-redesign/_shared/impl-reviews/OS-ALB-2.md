# OS-ALB-2 · 对抗性实现审查（老师相册生命周期 + 照片-幼儿关联）

- verdict: PASS
- score: 88/100
- 基线: 2759bc8 → 2759bc8（无 commit）
- 范围: OpenSpec `class-album-parent-sharing` tasks 2.1–2.6 + 3.1–3.5；纯后端切片，brief 禁止前端与 DevTools
- 源码覆盖: `album_repo.go`、`teacher_album_service.go`、`teacher_album_test.go`、`teacher_portal.go`、`router.go`、`provider.go`、`10-album.md`、`TEST_CATALOG.md`

## 证据

- 变更集: 8 文件与 worker manifest 完全一致；按 mtime 扫描 2759bc8 以来所有变更路径，窗口内仅这 8 个；窗口内新增未跟踪文件 0；HEAD 未移动。
- `cd server && go build ./...` → exit 0
- `cd server && go vet ./internal/...` → exit 0
- `gofmt -l <6 Go 文件>` → 无输出
- `go test ./internal/service/ -run 'TestTeacherAlbum' -count=1 -timeout 600s` → exit 0，ok 96.755s，22 Test 全 PASS（含 8 个越权子用例）
- `go test ./internal/handler/http/ -count=1` → exit 0，0.185s
- `go test ./internal/app/ -count=1` → exit 0，0.207s（全量路由注册无冲突）
- `node scripts/check-secrets.mjs --files <8 文件>` → 0 findings
- Token/DevTools/截图/console：后端切片不适用

## 需求核验

- 状态机：创建 draft；publish 允许 draft/unpublished，unpublish 仅 published；原子 `UPDATE ... WHERE status IN`；非法转换 422；编辑不改状态；软删任意状态且留行。
- 统计：仅 published 且未软删；草稿/下架/软删测试均断言排除。
- 编辑：提交 ID 保留原行，未提交硬删照片+关联，photo_count/cover_url 同事务重算，重复提交幂等；越权照片 ID 整单拒绝。
- 关联：album_students + album_photo_students 事务写入；file_id 校验文件为图片且上传者在本园有 teacher 角色；source_media_id 同园同班+未软删+image，按 growth_log_students 自动带出，显式 student_ids（含 []）覆盖。
- 未授权计数：只认显式 `allow_class_share=0`（缺行允许），SQL 任一监护人关闭即命中；照片级优先、相册级兜底，三类场景均有单测。
- 审计：operation_logs 的 module/action/target/operator/detail 均由测试断言，含 4 种动作。
- 契约与 TC：新增端点/字段/错误码与 handler 一致；TC-0008..0017 与测试函数一一对应且无重复；provider AutoMigrate 含两张新表。
- 清理：cleanup 全部硬删，残留测试逐表断言 0；本次复跑全绿。

## P1 · 阻塞问题（0）

无越界改动、无假声明、无失败校验、无验收级缺陷。

## P2 · 建议（7）

1. [test] 多监护人「任一关闭」无对应用例（fixture 每幼儿仅一个监护人），回归成「全部关闭」仍会全绿。
2. [contract] 上传返回字段是 `id`，契约写成「拿到 file_id」；PUT 未写明省略 participants 会清空相册级参与者。
3. [performance] file_id 归属校验逐张查询（最多 50 条 SQL），其余关联查询均已批量化。
4. [test] 四个新 handler 无直接用例；acceptance 要求的 handler 命令虽通过但未覆盖新端点。
5. [audit] 审计写在事务外、失败仅记日志，与规格「MUST 记录操作人与时间」在极端失败下有缝；契约已声明取舍。
6. [design drift] D3 状态图未画 unpublished→published，实现与契约已允许并测试。
7. [concurrency] 编辑不检查 album UPDATE 的 RowsAffected；并发软删窗口内可继续写照片/关联并写审计（单用户场景概率极低）。

## 残余风险 / 范围外

- 家长端可见性（5.x）、管理端（6.x）、老师端页面（7.x）、albumretention（10.1）未实现；规格中的「家长可见」断言需待家长服务落地后端到端验证。
- 省略 photos[].id 会重建照片行（ID 漂移），契约要求保留照片回传 ID；当前无其它表引用 album_photos。
- 旧 URL 照片 file_id 为 null（已测）；source_media 自动带出的幼儿不做本班校验（刻意取舍，转班/毕业历史仍参与隐私过滤）。

## 增量复审（round 2 · pre-commit fixes）

- 范围仍为同 8 文件；本轮仅 `album_repo.go`/`teacher_album_service.go`/`teacher_album_test.go`/`10-album.md` 变更，HEAD 未动。
- 复跑：`go build ./...`=0；`go vet ./internal/...`=0；service 组=0（ok 104.506s，24 Test 全 PASS）；handler=0；app 路由=0；check-secrets=0；gofmt 无输出。
- RowsAffected 守卫：0 行更新返回 `ErrAlbumNotFound`，service 映射 404。`TestTeacherAlbumRepoUpdateRejectsDeletedAlbum` 先软删再调 repo.Update，断言哨兵错误 + 照片数不变 + 关联 0；去掉守卫会多写 1 张照片与 1 条参与者，测试必失败（反事实成立）。不存在相册分支亦断言 0 写入。
- 多监护人：新增第二位显式关闭的监护人、第一位无设置行，断言 any-off 计数 1；ALL-guardians-off 回归会得 0 而失败。
- 隔离：别班 publish / 他园 unpublish / 别班 delete 均断言 404。
- 契约：上传响应 `id` == `photos[].file_id`；participants/photo student_ids 全量替换语义已写入。
- round-1 的 P2-1/2/7 关闭；剩余 5 项 P2（file_id 逐张校验、新 handler 无用例、审计 best-effort、D3 图漂移、UpdateStatus 并发软删 422）。P1=0，verdict PASS，score 91。
