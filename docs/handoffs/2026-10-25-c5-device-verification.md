# C-5 真机验证手册（2026-10-25）

> 用途：C-5 归档时无真机环境，5.4 / 9.3 / 9.5 / 9.7 / 12.6 与 12.3 的双端联跑按用户决定豁免。
> 设备可用后照本手册补验，通过后在归档 change 的 `tasks.md` 对应条目录入证据。
>
> 归档位置：`openspec/changes/archive/2026-10-20-redesign-parent-task-loop/tasks.md`
> 相关背景：`docs/handoffs/2026-10-21-parent-task-loop.md` §28.6

---

## 0. 一次性前置（最容易卡住的都在这里）

### 0.1 账号与版本

| 端 | AppID | 主体 | 真机要求 |
| --- | --- | --- | --- |
| 老师端 | `wxbfb5f81a87f73a61` | 个人号 | 老师微信号是该小程序**开发者/体验成员** |
| 家长端 | `wxdf989efff8796c53` | 非个人 | 家长微信号是**体验成员**（`trial`）或开发者（`develop`） |

- 分享码指向哪个版本由后端 `wechat.code_env_version` 决定（`release` / `trial` / `develop`，见 `server/internal/auth/wechat.go:78`）。
- **真机测试必须设 `trial` 或 `develop`**：设成 `release` 而正式版还没发布时，扫码会提示「小程序未发布」。
- 改完 `configs/app.ini` 的 `wechat.code_env_version` 后**重启后端**才生效。

### 0.2 手机可达的后端地址（默认构建连不上真机）

开发构建默认 `baseURL=http://127.0.0.1:8080/api/v1`（`teacher|parent/src/config/env.js`），手机访问不到。重新构建两端：

```bash
# 先查 PC 局域网 IP（PowerShell）
ipconfig | Select-String "IPv4"
```

```bash
# Windows PowerShell，分别进 teacher/ 与 parent/
$env:VITE_API_BASE_URL="http://<PC-LAN-IP>:8080/api/v1"
$env:VITE_UPLOAD_URL="http://<PC-LAN-IP>:8080/api/v1/common/upload"
npm run build:mp-weixin
```

前置检查（缺一不可）：

1. 手机与 PC 在同一局域网，手机浏览器打开 `http://<PC-LAN-IP>:8080/health` 返回 200；
2. Windows 防火墙放行 8080（或临时关闭专用网络防火墙）；
3. 小程序后台「开发管理 → 开发设置 → 服务器域名」对该域名放行，或在开发者工具/真机调试里开启「不校验合法域名」；
4. 上传**体验版**（老师端、家长端各一次），并把测试微信号加入体验成员；
5. 家长端账号已绑定孩子（`student_parents` 有绑定）。

### 0.3 数据准备

- 老师端建一个**班级任务**（`allow_public=1`、仪式贴纸、每日）用于 5.4/9.5/9.3。
- 老师端建一个**个体任务**（指定某个孩子）用于 12.3。
- 记录 `task_id`（列表点进详情后从地址或后端日志确认）。

### 0.4 基线记录（验证前先存一份）

```sql
-- 记下当前最大值，便于验证后只看新增行
SELECT MAX(id) AS max_checkin FROM task_check_ins;
SELECT MAX(id) AS max_response FROM task_responses;
SELECT MAX(id) AS max_message FROM messages;
```

---

## 1. 5.4 + 12.6 分享码生成与真实扫码

**目标**：老师生成的码指向**家长端**，扫码进入对应版本的家长端任务详情；`code_env_version` 非法时有明确错误。

### 步骤

1. 后端配置 `wechat.code_env_version = trial`，重启后端；确认日志无启动错误。
2. 老师端真机：工作台 → 亲子任务 → 点任务卡片进详情 → 点「生成小程序码」。
3. 首次会弹「保存到相册」系统授权 → 允许。
4. 打开微信「扫一扫」→ 从相册选刚保存的码。
5. 用**家长微信号**扫（不要用老师号）。

### 预期

| 判据 | 预期 |
| --- | --- |
| 生成成功 | toast「已保存，可转发班级群」，相册出现 `task-share-<id>.png` |
| 扫码进入 | 打开的是**家长端**小程序，落在 `pages/tasks/detail`，任务标题与老师端一致 |
| 版本 | `trial` 配置下进入体验版（右上角菜单可见「体验版」标记） |
| 非法版本 | 把 `code_env_version` 改成 `production` 重启，再点生成 → **422**，文案含「小程序码版本配置非法：wechat.code_env_version 只能是 release/trial/develop」 |

### 失败诊断

- 扫码提示「未发布」→ 家长端体验版没上传，或 `code_env_version=release`。
- 扫码进了老师端 → 后端用了老师端凭据（不应发生）；查 `GetParentWXACodeUnlimit` 的 role。
- 生成报「请检查家长端微信配置」→ `wechat.parent_app_id/parent_secret` 为空或错误。
- 码保存成功但扫描无反应 → 检查 `scene=t=<id>` 是否被截断（上限 32 字符）。

### 证据要记

- toast 截图 / 相册截图
- 扫码后页面截图（含任务标题）
- 后端响应里的 `env_version`（`GET /teacher/tasks/:id/share-card` 返回体）

---

## 2. 9.5 保存相册与转发

与第 1 节同一次操作即可覆盖，额外确认：

1. 点「生成小程序码」后**不要**立刻退出页面；保存系统弹窗期间按钮显示为忙碌态，超时（15s）后恢复可点。
2. 拒绝相册权限 → toast「请允许保存到相册后重试」（不是静默失败）。
3. 把码转发到文件传输助手/班级群，用家长号从聊天里打开 → 同样落到任务详情。

---

## 3. 9.3 语音回应（主回应 + 补充回应）

**目标**：真机录音 → 上传 → 回应落库；家长端可播放；补充语音不影响主回应。

### 3.1 主回应语音

1. 家长端先用该任务提交一条记录（文字或一键完成）。
2. 老师端真机进任务详情 → 「待回应」tab → 该行点「录音」。
3. 首次弹麦克风授权 → 允许；说 2–3 秒 → 点「停止」。
4. 等待「上传语音」结束。

**预期**：该行从待回应消失；toast「已回应」；切到「已回应」tab 能看到语音回应。

### 3.2 补充语音

1. 「已回应」tab → 点「补充回应（个别夸一句）」→「录语音补充」。
2. 录 2 秒 → 停止 → 等待上传。

**预期**：出现「已追加补充回应」；补充面板只有「贴纸/语音」，没有班级星星。

### 3.3 家长端播放

1. 家长端打开同一任务详情 → 「老师的回应」区点语音条。
2. 能听到声音；弱网下失败 toast「语音播放失败」，不静默。

### 库核对

```sql
-- 主回应：voice + single，media 已落
SELECT id, ritual, response_type, media_file_id, media_url, teacher_id
FROM task_responses WHERE task_id = <TASK_ID> ORDER BY id DESC LIMIT 3;

-- 补充：独立 TaskResponse + 关系行，主回应的 response_id 不变
SELECT r.id, r.ritual, r.response_type, s.check_in_id
FROM task_responses r JOIN task_response_supplements s ON s.response_id = r.id
WHERE r.task_id = <TASK_ID> ORDER BY r.id DESC LIMIT 3;

SELECT id, response_id FROM task_check_ins WHERE task_id = <TASK_ID> ORDER BY id;
```

### 失败诊断

| 现象 | 原因 |
| --- | --- |
| 「录音失败，请检查麦克风权限」 | 系统权限被拒，去设置里打开 |
| 上传后无回应 | `/common/upload` 失败或 `biz_type` 不是 `task`；看后端日志 |
| 409 / 「请刷新后重试」 | 同一行在录音期间被别的操作关闭（批量回应）；属预期防护 |
| 家长端播放失败 | `media_url` 为空或 OSS 地址手机不可达 |

---

## 4. 9.7 原生 picker 与全链路

**目标**：IDE 里无法真实点选的原生 picker，在真机上逐项点一遍。

### 老师端创建任务表单逐项

| 控件 | 操作 | 预期 |
| --- | --- | --- |
| 范围 | 选「个体」 | 出现学生选择；`allow_public` 自动关闭且不可开 |
| 学生 | 选目标孩子 | 保存后 `parent_tasks.target_student_id` = 该孩子 |
| 频率 | 每日/每周/一次性逐项切换 | 保存后 `frequency` 与所选一致 |
| 起止日期 | 选跨天区间 | `start_date/end_date` 与选择一致 |
| 回应仪式 | 贴纸/语音/班级星星 | 选班级星星时星星范围可选「按班级」 |
| 星星范围 | 按任务 → 目标 >0；按班级 → 隐藏任务目标 | 按班级时 `star_goal=0`，使用班级罐目标 |

### 预期失败路径

- 选「按班级」但该班本学期未配置罐 → 提交按钮禁用并提示「当前学期未配置」；强行提交 → 422「请先配置班级星星罐」。
- 标题/学习关联为空 → 前端拦截或 422。

### 库核对

```sql
SELECT id, target_student_id, frequency, start_date, end_date,
       recognition_ritual, star_scope, star_goal, allow_public
FROM parent_tasks WHERE id = <TASK_ID>;
```

---

## 5. 12.3 跨端单轮联跑

**目标**：同一轮里串起「老师建个体任务 → 目标家庭可见 → 家长提交 → 老师回应 → 家长反馈 → 老师补充 → 红点/给孩子看 → 读标记清除」。

### 推荐跑法

- **双设备**：老师手机（老师号）+ 家长手机（目标孩子家长号）同时在线；最接近真实。
- **单设备切换**：同一台手机先老师号跑完创建/回应，退出换家长号；数据连贯即可，注意切换后等首页刷新。
- **第二家长号**（验证「其他家庭不可见」）：用另一个家长号登录，确认列表里没有该个体任务。

### 步骤与预期

| # | 端 | 操作 | 预期 UI | 库核对 |
| --- | --- | --- | --- | --- |
| 1 | 老师 | 创建个体任务，目标选孩子 A | 列表出现该任务，标「个体任务」 | `parent_tasks.target_student_id = A`、`allow_public=0` |
| 2 | 家长 A | 任务列表 | 能看到该任务；状态「待完成」 | — |
| 3 | 家长 B | 任务列表 | **看不到**该任务 | — |
| 4 | 家长 A | 详情提交（一键完成或文字） | toast「已提交」；列表变「已提交」 | `task_check_ins` 新增 1 行，`response_id=0` |
| 5 | 老师 | 待回应 → 点贴纸/加星回应 | 行消失，toast「已回应/已加星」 | `task_check_ins.response_id` 指向新 `task_responses` |
| 6 | 家长 A | 详情看回应 → 点「孩子很开心」 | 按钮变 active；重进仍回显 | `task_response_reactions` 新增 `(response_id, user_id, child_happy)` |
| 7 | 老师 | 已回应 tab → 补充回应 → 贴纸 | 「已追加补充回应」 | 新 `task_responses`（single）+ `task_response_supplements`；主 `response_id` 不变 |
| 8 | 家长 A | 返回列表再进详情 | 列表有红点；进详情后红点消失 | `task_read_marks.last_read_at` 写入/更新 |
| 9 | 家长 A | 回应卡「给孩子看」 | 全屏放大贴纸/播放语音/老师的话 | — |
| 10 | 任意 | 查积分 | 任务流程不产生积分 | `point_transactions` 无 task 来源新增 |

### 「给孩子看」附加检查

- 全屏态不依赖孩子账号；返回后列表状态不变。
- 语音回应在全屏态可播放（与 9.3 一起验）。

### 失败诊断

- 家长 B 也能看到 → `target_student_id` 没落库或被置 0（查第 1 步）。
- 家长 A 看不到 → 检查孩子是否在任务班级、`student_id` 是否传对。
- 反馈后重进丢失 → 查 `task_response_reactions` 是否有行、`my_reaction` 是否返回。
- 红点不消失 → 进详情是否调用了 `POST /parent/tasks/:id/read`；查 `task_read_marks`。

---

## 6. 验证后收尾

1. 在归档 change 的 `tasks.md` 对应条目追加 `<br>` 证据（日期、设备、截图/库结果）。
2. 若 12.3 完成了单轮联跑，把它原来的「分段证据」说明改成联跑证据。
3. 恢复配置与构建：
   - `wechat.code_env_version` 改回 `release`（或团队约定的值），重启后端；
   - 两端构建恢复默认 `baseURL`（不要提交带 LAN IP 的 dist）。
4. 清理测试数据（**只对测试库执行**）：

```sql
-- 把 <TASK_ID> 换成测试任务 id；先确认没有真实数据挂在上面
DELETE s FROM task_response_supplements s
  JOIN task_responses r ON r.id = s.response_id WHERE r.task_id = <TASK_ID>;
DELETE FROM task_response_reactions WHERE response_id IN
  (SELECT id FROM task_responses WHERE task_id = <TASK_ID>);
DELETE FROM task_check_ins WHERE task_id = <TASK_ID>;
DELETE FROM task_responses WHERE task_id = <TASK_ID>;
DELETE FROM task_checkin_states WHERE task_id = <TASK_ID>;
DELETE FROM task_read_marks WHERE task_id = <TASK_ID>;
DELETE FROM messages
WHERE reference_id = <TASK_ID>
   OR action_path LIKE '%?id=<TASK_ID>&%'
   OR action_path LIKE '%?id=<TASK_ID>';
DELETE FROM parent_tasks WHERE id = <TASK_ID>;
```

---

## 附录：常见错误对照

| 现象 | 先查 |
| --- | --- |
| 真机所有接口「网络异常」 | 构建时 `VITE_API_BASE_URL` 是否指向 LAN IP；手机能否打开 `/health` |
| 登录失败 40002 | 后端日志里的 `wechat login failed`（C-5 已补日志）；确认 AppID/Secret 与端别 |
| 扫码进错端 | 分享码是否用了家长端凭据；`env_version` 是否与上传版本一致 |
| 录音无声音 | 麦克风权限、`recorder.start` 是否被上一段录音占用 |
| 相册保存超时 | 系统授权弹窗没点；15s 后按钮应恢复可点 |
