# 模块 7：接送管理 API

> 对应开发计划：模块 7：接送管理
>
> 基础路径：`/api/v1/teacher`
>
> 认证：`Authorization: Bearer <token>`，用户角色为 `teacher`
>
> 以下 Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

## 接口清单

- `GET /teacher/pickup/overview`
- `GET /teacher/pickup/students`
- `POST /teacher/pickup/scan`
- `POST /teacher/pickup/manual`
- `GET /teacher/pickup/records`
- `GET /teacher/pickup/records/:id`
- `PUT /teacher/pickup/records/:id/abnormal`
- `POST /teacher/pickup/notify-overdue` —（本次契约变更新增）
- `GET /teacher/pickup/students/:id/authorized-persons` —（本次契约变更新增，teacher-pickup-auth-picker）
- `PUT /teacher/pickup/authorized-persons/:id/approve` —（补文档，路由已注册）
- `PUT /teacher/pickup/authorized-persons/:id/reject` —（补文档，路由已注册）

> **本次为契约变更**（pickup-custody-evidence + teacher-pickup-auth-picker）：`scan`/`manual`/`students`/`overview`/`records/:id` 均有改动；`students/:id/authorized-persons` 为新增。

---

## 6. 接送管理

### GET `/teacher/pickup/overview`

> 今日接送统计
>
> **本次契约变更**：新增 `overdue_not_picked`（超时未接计数，**按学生计**，非事件数）。

```json
// Response
{
  "date": "2026-03-19",
  "total": 30,
  "arrived": 28,
  "picked_up": 12,
  "not_picked_up": 16,
  "not_arrived": 2,
  "overdue_not_picked": 3,
  "abnormal": 0
}
```

**字段说明**：

- `overdue_not_picked`：超过所在班级放学时刻仍未接走的学生数。口径以**学生**为计数单位，同一学生当日只计一次，随后被接走也不清零
- `total` / `not_arrived`：实现已返回，本文档补齐

---

### GET `/teacher/pickup/students`

> 未接走 / 全部学生接送状态列表
>
> **本次契约变更**：每人新增 `is_overdue` / `pickup_deadline` / `last_remark` / `is_authorized_override`；
> 新增 `authorized_person_list`（结构化授权接送人，`authorized_persons` 转为兼容字段）。

```json
// Query: ?status=not_picked_up
// status: arrived=已入园 / picked_up=已接走 / not_picked_up=未接走，不传查全部

// Response
{
  "list": [
    {
      "student_id": 1,
      "name": "王小明",
      "avatar": "https://cdn.example.com/avatar/s1.jpg",
      "authorized_persons": "爸爸、妈妈、爷爷",
      "authorized_person_list": [
        { "name": "爸爸", "relation": "father" },
        { "name": "妈妈", "relation": "mother" },
        { "name": "爷爷", "relation": "grandfather" }
      ],
      "arrive_time": "07:52",
      "pickup_status": "not_picked_up",
      "is_overdue": true,
      "pickup_deadline": "16:00",
      "last_remark": "",
      "is_authorized_override": false
    },
    {
      "student_id": 2,
      "name": "李小花",
      "avatar": "https://cdn.example.com/avatar/s2.jpg",
      "authorized_persons": "妈妈、奶奶",
      "authorized_person_list": [
        { "name": "妈妈", "relation": "mother" },
        { "name": "奶奶", "relation": "grandmother" }
      ],
      "arrive_time": "08:05",
      "pickup_status": "not_picked_up",
      "is_overdue": false,
      "pickup_deadline": "16:00",
      "last_remark": "",
      "is_authorized_override": false
    }
  ],
  "total": 16
}
```

**字段说明**（接口粒度：仅列表渲染所需，详情字段不进本响应）：

- `is_overdue`：当前时间已过 `pickup_deadline` 且尚未接走 → 用于滞留名单高亮
- `pickup_deadline`：该生所在班级的放学时刻，格式 `HH:mm`；未配置时为缺省 `16:00`
- `last_remark`：当日最后一次登记的理由（重复接走 / 提前接走时必填）
- `is_authorized_override`：当日最后一次接走是否走了授权例外通道
- `authorized_person_list`：有效授权接送人的结构化列表，元素 `{name: string, relation: string}`。
  过滤口径与 `authorized_persons` 完全一致（`status=1` 且 `review_status=1` 且未软删），
  顺序也与之一致（同一 `ORDER BY id`）；无有效授权人时为 `[]`
- `relation`：关系 9 码闭集 —— `father` / `mother` / `grandfather` / `grandmother` /
  `maternal_grandfather` / `maternal_grandmother` / `uncle` / `aunt` / `other`。
  **接口只返回码值，中文名由端上映射**；库值为空时原样返回空字符串，接口不编造
- `authorized_persons`：**兼容字段**（姓名顿号串，无有效授权人时为「暂无」），
  供旧版页面渲染；新页面请改用 `authorized_person_list`

---

### POST `/teacher/pickup/scan`

> 扫码接走登记
>
> **本次契约变更**：
> - 请求**只带** `qr_code`，**移除** `student_id` —— 学生与接走人由后端从凭证自解释
> - 凭证绑「学生 + 今日接走人」，登记时**再次校验**接走人仍在有效授权集合内
> - 凭证用后即失效（一次有效）
> - 响应新增 `person`（供老师侧人证对照），`student` 补 `photo`

```json
// Request
{
  "qr_code": "a3f9c2e81b0d4f6a7c5e9d2b8a4f1c6e",
  "remark": ""
}

// Response
{
  "record_id": 301,
  "student": {
    "id": 1,
    "name": "王小明",
    "photo": "https://cdn.example.com/student/1.jpg"
  },
  "person": {
    "name": "奶奶（张三）",
    "relation": "grandmother",
    "photo": "https://cdn.example.com/pickup/person/3.jpg"
  },
  "type": 2,
  "type_label": "接走",
  "is_authorized": true,
  "is_authorized_override": false,
  "recorded_at": "2026-03-19T17:30:25Z"
}
```

**错误**：

| 情形 | 返回 |
|---|---|
| 凭证无效 / 已过期 / 已使用 | `INVALID_QR_CODE` |
| 接走人授权已被撤销或过期 | 明确错误，不产生记录 |
| 连续同向（如已接走再接走） | 明确错误 |
| 重复接走（当日第 2+ 次）未填 `remark` | 校验错，提示需填理由 |
| 提前接走（早于班级放学时刻）未填 `remark` | 校验错，提示需填理由 |

**理由必填规则**：重复接走与提前接走时 `remark` 必填，其余情形可选。

---

### POST `/teacher/pickup/manual`

> 手动登记
>
> **本次契约变更**：
> - 新增 `authorization_id`（可空）/ `is_authorized_override` / `override_name`，区分「从授权人列表选人」与「标例外」
> - `remark` 约束收紧：重复接走、提前接走时必填
> - **送入（type=1）不记交接人**、不要理由（有意偏离 `design-data/admin-05`，见 proposal 分歧清单）

```json
// Request — 从授权人列表选人
{
  "student_id": 1,
  "type": 2,
  "authorization_id": 3,
  "is_authorized_override": false,
  "remark": ""
}

// Request — 标记为非授权 / 临时接送（例外通道）
{
  "student_id": 1,
  "type": 2,
  "authorization_id": null,
  "is_authorized_override": true,
  "override_name": "邻居王五",
  "remark": "家长临时委托，已电话确认"
}

// Response
{
  "record_id": 302,
  "student": {
    "id": 1,
    "name": "王小明",
    "photo": "https://cdn.example.com/student/1.jpg"
  },
  "person": {
    "name": "奶奶（张三）",
    "relation": "grandmother",
    "photo": "https://cdn.example.com/pickup/person/3.jpg"
  },
  "type": 2,
  "type_label": "接走",
  "is_authorized": true,
  "is_authorized_override": false,
  "recorded_at": "2026-03-19T17:35:00Z"
}
```

**字段说明**：

- `authorization_id`：从授权人列表选人时必填；标例外时为 `null`
- `is_authorized_override`：`true` 表示本次为授权例外，此时 `override_name` 必填
- `override_name`：例外情形下的来接者姓名，用于事后复核
- 例外**不阻断**登记，仅留痕可统计
- **例外同时置异常**：`is_authorized_override=true` 的接走记录落库时同时写
  `is_abnormal=true`，`abnormal_reason` 取 `remark`（trim 后为空则用固定文案
  「非授权接送人例外放行」）。非例外的手动接走、入园、扫码路径一律不自动置异常
- `is_authorized`：本次接走人是否在有效授权集合内（例外通道时为 `false`）

---

### GET `/teacher/pickup/records`

> 接送记录列表

```json
// Query: ?date=2026-03-19&student_id=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 301,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "type": 2,
      "type_label": "接走",
      "method": 1,
      "method_label": "扫码",
      "pickup_name": "爸爸（王明）",
      "pickup_relation": "father",
      "operator": { "id": 5, "nickname": "李师傅" },
      "is_abnormal": false,
      "recorded_at": "2026-03-19T17:30:25Z"
    },
    {
      "id": 300,
      "student": { "id": 1, "name": "王小明", "avatar": "..." },
      "type": 1,
      "type_label": "入园",
      "method": 1,
      "method_label": "扫码",
      "pickup_name": "妈妈（李丽）",
      "pickup_relation": "mother",
      "operator": { "id": 5, "nickname": "李师傅" },
      "is_abnormal": false,
      "recorded_at": "2026-03-19T07:52:18Z"
    }
  ],
  "total": 56
}
```

---

### GET `/teacher/pickup/records/:id`

> 接送记录详情
>
> **本次契约变更**：响应改为 `{"record": {...}}` 包装（以实现为准，此前文档为扁平对象）。

```json
// Response
{
  "record": {
    "id": 301,
    "student": {
      "id": 1, "name": "王小明", "avatar": "...",
      "class_name": "小一班"
    },
    "date": "2026-03-19",
    "type": 2,
    "type_label": "接走",
    "method": 1,
    "method_label": "扫码",
    "pickup_name": "奶奶（张三）",
    "pickup_relation": "grandmother",
    "pickup_phone": "138****8888",
    "operator": {
      "id": 5,
      "nickname": "李师傅",
      "title": "门卫"
    },
    "is_abnormal": false,
    "abnormal_reason": "",
    "is_authorized_override": false,
    "remark": "",
    "recorded_at": "2026-03-19T17:30:25Z"
  }
}
```

**字段说明**：

- `operator`：登记/核验人（园所方）。契约列名对齐 `design-data/admin-05` 的「核验人」语义
- `is_authorized_override`：本次接走是否走了授权例外通道

---

### PUT `/teacher/pickup/records/:id/abnormal`

> 标记接送异常（老师手动开启/取消；系统仅在授权例外登记时自动置异常）

```json
// Request
{
  "is_abnormal": true,
  "abnormal_reason": "超时未接走，已联系家长"
}

// Response
{
  "id": 301,
  "is_abnormal": true,
  "abnormal_reason": "超时未接走，已联系家长",
  "updated_at": "2026-03-19T18:30:00Z"
}
```

**说明**：重复接走、提前接走**不**自动置异常，只保留相应理由；授权例外不在此列 ——
例外接走落库时**自动置 `is_abnormal=true`** 并写入异常原因（见 `POST /teacher/pickup/manual`），
其余路径仍只能由老师通过本接口手动标记或取消。

---

### POST `/teacher/pickup/notify-overdue`

> 一键通知滞留儿童家长（本次契约变更新增）
>
> 对「已到园且超过班级放学时刻仍未接走」的儿童，向其家长投递接送类站内消息。
> 重复触发留痕可查，不覆盖历史。

```json
// Request
{
  "student_id": 1
}

// Response
{
  "student_id": 1,
  "notified": true,
  "message_id": 8801,
  "notified_at": "2026-03-19T18:05:12Z"
}
```

**错误**：学生不属于本班 → 拒绝；学生非滞留状态 → 校验错。

---

### GET `/teacher/pickup/students/:id/authorized-persons`

> 按学生查有效授权接送人（本次契约变更新增，teacher-pickup-auth-picker）
>
> 供手动登记弹窗「选授权人」模式使用，懒加载。
> 仅返回**有效**授权人（status=1 且 review_status=1 且未过期），确保列表与后端校验口径一致。

```json
// Response
{
  "list": [
    {
      "id": 3,
      "name": "奶奶（张三）",
      "relation": "grandmother",
      "phone": "138****8888",
      "avatar": "https://cdn.example.com/pickup/person/3.jpg",
      "auth_type": "permanent"
    },
    {
      "id": 5,
      "name": "爸爸（王明）",
      "relation": "father",
      "phone": "139****0001",
      "avatar": "",
      "auth_type": "temporary"
    }
  ],
  "total": 2
}
```

**字段说明**：

- `id`：授权记录 ID，提交 `POST /pickup/manual` 时作为 `authorization_id`
- `phone`：脱敏值（同家长端 `GET /parent/pickup/authorized-persons`）
- `relation`：关系标识（`father`/`mother`/`grandfather`/`grandmother`/`other`）
- `auth_type`：`permanent`（长期）/ `temporary`（临时）
- **不返回**证件号相关字段

**错误**：学生不属于本班 → 拒绝。

---

### PUT `/teacher/pickup/authorized-persons/:id/approve`

> 审核授权接送人 — 通过（补文档，路由已注册）

```json
// Body
{
  "remark": ""
}

// Response
{
  "id": 3,
  "review_status": 1,
  "review_status_label": "已通过",
  "audited_at": "2026-03-19T10:00:00Z"
}
```

### PUT `/teacher/pickup/authorized-persons/:id/reject`

> 审核授权接送人 — 拒绝（补文档，路由已注册）

```json
// Body
{
  "remark": "证件不符"
}

// Response
{
  "id": 3,
  "review_status": 2,
  "review_status_label": "已拒绝",
  "audited_at": "2026-03-19T10:00:00Z"
}
```

**限制**：仅限**本班**学生的授权人；仅 `review_status = 0`（待审核）可审，否则返回 `PICKUP_AUTH_NOT_PENDING`。
