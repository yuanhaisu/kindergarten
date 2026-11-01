# OS-ALB-6b · 对抗性实现审查

- verdict: **FAIL**
- score: 68/100
- 源码覆盖: `teacher/src/pages/album/index.vue`、`teacher/src/api/teacher-albums.js`
- 审查范围: diff（686+/31−、1+/1−）+ untracked + build 配置 + 调用链（`/teacher/albums*`、`/teacher/class/students`、`/common/upload`）+ 选择器/编辑表单 + 契约 `backend/api/api-teacher/10-album.md` + 后端校验实现
- 基线: `19c2115` → `19c2115`（worker 未提交；HEAD 未变）

## DevTools 证据

- 构建: `cd teacher && npm run dev:mp-weixin` → `DONE Build complete`，`ready in 10095ms`；`dist/dev/mp-weixin/pages/album/index.{js,wxml,wxss}` mtime `2026-11-01 12:28:14`（源码 `12:23:51`）；watcher 清理后残留 0。`npm run build:mp-weixin` → exit 0。
- Token: `node scripts/check-tokens.mjs teacher` → 3 violations，全部为既有 `components/CalendarEventCard.vue:79/80`；album 文件新增 0。
- Secret scan: `node scripts/check-secrets.mjs --files ...` → 0 findings，exit 0。
- 目标端: teacher CDP 9222 / automator 9420；探针 `pages/more/index` 正常；IDE 项目「幼儿园老师端」，appid `wxbfb5f81a87f73a61`。
- Mock 声明: 8080 后端未启动（未重启）；在 AppService 内状态化 mock `/teacher/albums*`、`/teacher/class/students`、`/auth/login/wechat`，其余请求委托原生 `wx.request`；`wx.uploadFile` / `wx.chooseImage` 为上传与选图 seam（原生选图对话框不可 CDP 驱动）。真实上传/后端 E2E 为残余。
- 正常态: 列表/统计/月份卡片渲染；创建默认参与幼儿全选 4 人；3 张上传（file_id 7001–7003）；A/B 后照片 1 覆盖 A、照片 3 覆盖+恢复；POST 体 `participants:[1001,1002]`，仅照片 1 带 `student_ids:[1001]`。
- 编辑态: 打开详情→编辑，预填标题「秋日郊游」/分类「活动」/日期「10月24日」/参与者 A、B/照片摘要「1 人、默认 2、1 人」；未覆盖照片直接确定不产生覆盖；删 1 张 + 新增 1 张后 PUT 体保留 `id:201/203`、202 缺失、新照片无 `id` 且有 `file_id:7004`；覆盖照片保留 `student_ids`；保存后详情重拉「第 1 张 / 共 3 张」。
- 错误态: 列表 500 →「相册加载失败 / 网络异常，未获取到照片 / 重试」→ 恢复；PUT 500 → 表单保留、内联错误、按钮「重新保存」→ 重试成功；roster 500 → 提交禁用且不发 POST，picker 重试后恢复默认全选。
- 空态: 过滤空「这个状态下还没有相册 … 查看全部」→ 真空态「还没有照片 / 上传后可在这里按分类查看」。
- 交互: 全部用 `_verify/act.py` 真实 touch 事件；`wechat_navigate` 仅用于探针，不作为交互证据。
- Console: 页面重载后 `Runtime.discardConsoleEntries` + 3.5s 监听 → 0 console/exception/log 事件；早期 buffer 中的 `__v_skip` / `instanceof` 警告为 mock harness 自身缺陷（已修），mock 图片 404 与后端 down 的 index 页请求为环境噪声。
- 路由: 老师端无 tabBar，本 slice 只新增页内弹层，无路由改动。
- 截图: 未尝试（IDE 被终端覆盖，已知该形态截图会挂死）；改用 DOM/CSS 断言 + CDP 文本，建议真机复验。

## P1 · 阻塞问题 (1)

### 1. [logic] 相册含已离班/已软删参与幼儿时，编辑永远 422 且无法移除

- 证据:
  - 编辑预填全量写入 `album_students`：`teacher/src/pages/album/index.vue:627-631`；
  - 「取消全选」刻意保留不在名单里的历史 id：`index.vue:845`；
  - 选择器只渲染本班名单 `v-for="student in students"`：`index.vue:1366`；
  - 提交始终发送完整 `participants`：`index.vue:912`；
  - 后端只接受本园本班：`server/internal/service/teacher_album_service.go:685-691`，`FindStudentsInClass` 按 `class_id` 查（`server/internal/repository/album_repo.go:526`），转班/软删查不到 → 422（`teacher_portal.go:50`）；契约 `10-album.md:236,263`。
  - 详情契约会返回软删幼儿（name 空串，`10-album.md:188`；SQL LEFT JOIN `album_repo.go:178`）。
  - DevTools 注入 `album_students=[1001,1002,1005]`（1005 不在名单）：表单「已选 3 人：王小明、李小红」；picker 只有 4 行、确认「已选 3 人」；「全选」→5，「取消全选」→1，1005 无行可取消。
- 影响: 只要有一位参与幼儿转班/毕业/软删，相册编辑（哪怕只改标题）必然「请求参数无效」，无 UI 绕过。
- 建议: 在 picker 中渲染历史参与者行并标注「已离班/已删除」，允许显式取消；或提交前让用户确认移除，不要强制回传不可移除的 id。

## P2 · 建议 (4)

1. [logic] 编辑态未复用 roster 失败阻断（`index.vue:532-535` 仅 `!isEditing`）；`album_students` 为空时仍可提交 `[]`，与新建口径不一致。
2. [state] picker 勾选后又取消仍置 dirty，确定会写入与默认相同的显式覆盖（`index.vue:852-860`）；应与初始快照比较。
3. [contract] 详情 `photos[].students=[]` 无法区分「无关联/显式空覆盖」，编辑预填一律按继承处理（`index.vue:640-643`）；读取端两者都回退相册级，用户可见行为一致，记为契约限制。
4. [a11y] 照片 tile 打开逐张参与者无 aria-label（`index.vue:1299`），读屏仅得角标文本。

---

## 复审查（Round 2，2026-11-01）

- verdict: **PASS**（P1 已修复并运行时验证；0 P1 / 2 P2 建议）
- 改动: 仍仅 `teacher/src/pages/album/index.vue`、`teacher/src/api/teacher-albums.js`（839+/32−）；无 commit（HEAD 753be99）
- 构建: dev DONE（dist `05:32:31Z` > source `04:50:09Z`，watcher 0）；`build:mp-weixin` exit 0；token 3（既有 CalendarEventCard）；secret 0
- 目标端: 幼儿园老师端 / appid `wxbfb5f81a87f73a61`；探针 `pages/more/index` 通过；CDP 9222 + automator 9420
- Mock 声明: 8080 未启动且未重启；AppService 内状态化 mock albums/class-students/upload 等，交互全部 `_verify/act.py` 真实 touch

### P1 复验（原「已离班参与者不可移除 → 422」）

- 注入 `album_students=[1001,1002,1005]`（1005 不在 roster）：表单提示「已不在班级名单：陈小海」；picker 5 行，「陈小海 / 已离班/已删除」默认选中（is-on）。
- 全选→5 人；取消全选→1 人，陈小海行仍可见且选中（非沉默保留）。
- 保留陈小海时提交：按钮 is-disabled，提示「有 1 位幼儿已不在班级名单…」；点击 0 次 PUT。
- 取消陈小海并补选 1001/1002 后：PUT 体 `participants:[1002,1001]`，仅 roster id；照片 id 201/202/203 保留；未覆盖照片 202 无 `student_ids`。
- 逐张 dirty 修复：照片 2 picker 勾上 1003 再取消 → 仍「默认 2」；PUT 中 202 无 `student_ids`（继承保持）。
- roster 500 编辑阻断：hint「班级名单加载失败…」，提交禁用且无 PUT；恢复后 picker 自动重试，5 行含 1 条离班行。
- 空参与者 + roster 500：提交禁用并提示「…打开参与幼儿确认空名单」；显式确认后提交 `participants:[]`（无 photo `student_ids`）。
- aria-label：photo tile DOM 分别为「第 1 张，参与幼儿 1 人 / 第 2 张，参与幼儿默认 3 人 / 第 3 张，参与幼儿 1 人」。

### Console

- 无产品 JS 异常；2 条 error 均为环境/夹具（`mock.local` 图片 DNS、8080 down 的 home/overview 连接拒绝）；1 条 roster-500 用例的预期 `console.warn`；1 条 `[Vue warn]` 由审查探针枚举 `$vm` 触发。

### 残余

- 全部运行时证据为 mock 后端（按指令未重启 8080）；真实上传/后端 E2E 未复验。
- 照片级 stale id 阻断路径仅代码核验（本轮夹具的照片覆盖均为 roster id）。
- 列表空态本轮未重跑（不受 P1 修复影响，Round 1 已覆盖）。

### P2（建议）

1. `pickerStaleStudents` 以打开时 `pickerInitial` 计算离班行；不影响已验证流程，记录以备后续「恢复已移除的离班 id」需求。
2. 照片级 stale id 的提示文案仅代码核验，留待下轮夹具覆盖。
