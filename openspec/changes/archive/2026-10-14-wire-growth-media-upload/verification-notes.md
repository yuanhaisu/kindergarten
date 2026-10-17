# 验证说明（实现期发现、环境限制与覆盖边界，供归档回溯）

> 对应 config.yaml 的 archive 要求。记录实现/验证过程中超出规划范围的发现与环境边界。

## 实现期发现并修复的问题

| # | 问题 | 根因 | 修复 |
|---|---|---|---|
| 1 | 创建记录时媒体**入库顺序与提交顺序不一致** | 素材按 `map[uint64]File` 遍历建行，Go map 迭代序随机 | `createGrowthMediaForFiles` 改为按 `items`（提交顺序）遍历取 `filesByID[it.FileID]`；更新路径同样按提交序建行。实测 `a→b` 提交后库内顺序一致 |
| 2 | 首轮验证误报「编辑后 media=3」 | 验证脚本用位置 `ms[0]` 取保留项，而顺序 bug 使位置不可靠 | 脚本改为按 `ref_desc` 定位保留项；顺序 bug 修复后重测全绿 |

## 环境限制（影响验证覆盖，需在真机/配好 OSS 后补测）

| 限制 | 影响 | 缓解 |
|---|---|---|
| ~~本机 `[oss]` 段为空~~ **已于验证期间配置**（endpoint/access_key/bucket 填入，app.ini 已 gitignore） | ~~真实上传跑不通~~ **已解除** | 见下方「真实 OSS 上传验证」 |
| **老师端 UI 交互（6.3）**：已补测，结果与边界见下节 | 启动 IDE 后完成了入口点击、状态机、新构建渲染的验证；「真实选图 → 上传成功」的完整 UI 回路仍受限 | 见下方「UI 级验证（DevTools）」节 |

## UI 级验证（DevTools，实机点击）

**已验（CDP 真实鼠标/键盘事件）**：

```
新构建确实加载：新提示「每张不超过20MB」渲染；旧「自动识别孩子」承诺消失
点击「添加」按钮 → 真实触发 wx.chooseImage（逻辑层探针确认）
上传状态机：上传中（遮罩）→ 上传失败·重试（红色遮罩）→ 点击重试 → 回「上传中」 ✓
失败项可移除（✕）→ 列表恢复
```

**验证期间发现并修复的真实体验缺口**：请求挂起时页面会**永远停在「上传中」**（无前端超时、发布被无限阻断）。已补 30s 超时护栏（超时判失败、可重试），修复后实测「上传中 → 30s → 上传失败·重试」流转正确。

**完整 UI 回路（修正后补验，全部 PASS）**：

```
（此前曾误判「模拟器对 127.0.0.1 的 upload 不可达」——实为验证环境中后端进程在反复崩溃，
 请求打到死进程上表现为永远 pending。后端恢复后复测：localhost 与 127.0.0.1 均 2 秒内 SUCCESS 200。）

真实点击「添加」→ mock chooseImage 返回沙盒真实文件 → wx.uploadFile 200（落 OSS）
→ 条目渲染完成态、描述输入行出现（描述可录入）
→ 填入正文、真实点击「发布」→ 记录 48 创建
→ 详情媒体 = 1（ref_desc="UI-real-upload"、真实 OSS URL）
→ 家长端 feed 含该记录、卡片照片 thumb_url 指向 OSS ✓
```

**唯一的真实限制**：原生文件选择器无法被自动化合成（DevTools 限制，非代码问题）——自动化验证中 `chooseImage` 由 mock 返回沙盒文件；真实选图动作建议在人工走查中补一次（其余全链路已自动验通）。

## 真实 OSS 上传验证（配置凭据后补测，全部 PASS）

```
真实 multipart 上传 2 张 PNG → 200，files 行落库（file_id 4/5）
file_url  = https://kindergarten-dev.oss-cn-beijing.aliyuncs.com/uploads/growth/2026/10/<uuid>.png
thumb_url = 同 URL + ?x-oss-process=image/resize,w_200
OSS 对象 HEAD 检查：两个 file_url 均 200（对象真实存在、可访问）
带真实 file_id 发布记录 → 详情 media=2（含 ref_desc）
家长端 feed 含该记录、卡片媒体=2（照片经管道完整到达家长端）
验证数据已清理：files/growth_media 相关行 0 残留
```

## 验证覆盖摘要（API 级，真实 JWT + 真实数据）

```
上传拒绝分支（真实 multipart）：21MB → 42201；.txt → 42202                    ✅
创建带 2 个 file_id：详情 media=2，且入库顺序与提交一致（a→b）                 ✅
编辑保留 a + 新增 c（移除 b）：media=2，保留项在位、被移除项 log_id=NULL       ✅
保留项 ref_desc 更新生效、新增项描述正确                                       ✅
跨记录引用 file_id → 422/42200（拒绝且无写入）                                 ✅
编辑态越界 media.id → 422；记录未被破坏                                        ✅
不存在的 file_id → 422                                                         ✅
空数组清空附件 → 媒体行解除绑定（log_id=NULL，文件行保留）而非删除             ✅
```

后端：`go test -count=1 ./...` 全绿；`golangci-lint` 本次改动文件无新增；6 个媒体判定单测 PASS。

## 数据与工具残留

- 验证产生的模拟 `files`（3 行）与 `growth_media`（7 行）已全部删除（核对计数为 0）
- 临时工具 `cmd/devsql` 已删除
- **未决**：`cmd/mktoken` 的存留（自 `fix-growth-visibility-teacher-preview` 起挂账）
