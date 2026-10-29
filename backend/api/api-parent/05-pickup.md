# 家长端 API — 接送

> 基础路径：`/api/v1/parent` | 鉴权：`Authorization: Bearer <token>`（角色 `parent`）
>
> Response 均为 `data` 部分，省略外层 `{ "code": 0, "message": "success" }` 包装。

---

## 关系枚举（`relation` / `pickup_relation`）

接送关系统一为 **9 个稳定代码**，中文标签由前端映射；服务端不返回 `relation_label`。

| code | 中文标签 |
| --- | --- |
| `father` | 爸爸 |
| `mother` | 妈妈 |
| `grandfather` | 爷爷 |
| `grandmother` | 奶奶 |
| `maternal_grandfather` | 外公 |
| `maternal_grandmother` | 外婆 |
| `uncle` | 叔叔 |
| `aunt` | 阿姨 |
| `other` | 其他 |

- 旧 5 个 code（`father` / `mother` / `grandfather` / `grandmother` / `other`）继续有效，新增 4 个。
- 写接口（添加 / 编辑授权接送人）接受上表 9 个中文标签作为过渡兼容，并**归一化为代码落库**；
  存量历史数据若已是中文，读取时**原样返回**，前端映射需同时兼容 code 与中文标签。
- 不在上表内的值一律拒绝（校验错）。

---

### POST `/parent/pickup/designate`

> 指定今日接走人（本次契约变更新增）
>
> 指定结果覆盖该孩当日此前的指定，并使该孩当日已生成的接送凭证全部失效。
> 被指定人必须是该孩**已审核通过且有效**（未撤销、未过期）的授权接送人。

```json
// Query: ?student_id=1        // student_id 走 query，沿用本模块既有约定
// Body
{
  "authorization_id": 3
}

// Response
{
  "student_id": 1,
  "authorization_id": 3,
  "person": {
    "name": "奶奶（张三）",
    "relation": "grandmother",
    "avatar": "https://cdn.example.com/pickup/person/3.jpg"
  },
  "designated_at": "2026-03-19T08:00:00Z"
}
```

**错误**：被指定人未审核通过 / 已撤销 / 已过期 / 不属于该孩 → `40000x` 校验错

---

### GET `/parent/pickup/designation`

> 只读查「今日接走人」指定（本次新增，供授权人列表展示当前指定）
>
> 与 `GET /pickup/qrcode` 的区别：本端点**无副作用**，不会按需生成凭证。
> 未指定时 `designated` 为 `false`（不是错误）。

```json
// Query: ?student_id=1

// Response — 已指定
{
  "student_id": 1,
  "designated": true,
  "authorization_id": 3,
  "person": {
    "id": 3,
    "name": "奶奶（张三）",
    "relation": "grandmother",
    "avatar": "https://cdn.example.com/pickup/person/3.jpg"
  }
}

// Response — 未指定
{
  "student_id": 1,
  "designated": false,
  "person": null
}
```

---

### GET `/parent/pickup/qrcode`

> 获取接送二维码（绑「学生 + 今日接走人」）
>
> 前提：已通过 `POST /pickup/designate` 指定今日接走人。
> 凭证当日过期、一次有效；生成时会再次校验被指定人仍在有效授权集合内。

```json
// Query: ?student_id=1

// Response
{
  "qr_code": "a3f9c2e81b0d4f6a7c5e9d2b8a4f1c6e",
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
  "expire_at": "2026-03-19T23:59:59Z"
}
```

**字段说明**：

- `qr_code`：纯随机串，**不内嵌学生标识等可枚举数据**
- `student.photo` / `person.photo`：供老师侧人证对照（码上带名字方案）
- 响应**不含** `qr_image_url`（二维码图片由前端本地渲染）

---

### GET `/parent/pickup/records`

> 接送记录（分页）
>
> **按当前孩子隔离**：只返回该孩子的记录，不混入其他孩子。`student_id` 缺省时取当前孩子。

**Query 参数**：

- `student_id`（可选）：指定孩子，须为当前家长已绑定的孩子；缺省时使用当前孩子
- `page`（可选）：默认 1
- `page_size`（可选）：默认 20，最大 50

```json
// Request: ?student_id=1&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 301,
      "date": "2026-03-19",
      "type": 2,
      "type_label": "接走",
      "pickup_name": "爸爸（王明）",
      "pickup_relation": "father",
      "recorded_at": "2026-03-19T17:30:25Z",
      "is_abnormal": false
    },
    {
      "id": 300,
      "date": "2026-03-19",
      "type": 1,
      "type_label": "入园",
      "pickup_name": "妈妈（李丽）",
      "pickup_relation": "mother",
      "recorded_at": "2026-03-19T07:52:18Z",
      "is_abnormal": false
    }
  ],
  "total": 88,
  "page": 1,
  "page_size": 20
}
```

---

### GET `/parent/pickup/authorized-persons`

> 授权接送人列表（分页）
>
> **按当前孩子隔离**：授权记录挂在具体学生下（`student_id` 是授权记录自身的归属孩子），
> 列表只返回该孩子的授权人，不是家庭级名单；`student_id` 缺省时取当前孩子。
>
> **本次契约变更**：响应字段改为与实现一致 ——
> 移除 `id_card_tail`（不暴露证件号）、`status_label`（语义并入 `review_status_label`）；
> `phone` 为脱敏值；补 `review_status` / `review_status_label` 与分页字段。

**Query 参数**：

- `student_id`（可选）：指定孩子，须为当前家长已绑定的孩子；缺省时使用当前孩子
- `page`（可选）：默认 1
- `page_size`（可选）：默认 20，最大 50

```json
// Request: ?student_id=1&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 1,
      "student_id": 1,
      "student_name": "王小明",
      "class_name": "小一班",
      "name": "爸爸（王明）",
      "relation": "father",
      "phone": "138****8888",
      "avatar": "https://cdn.example.com/pickup/person/1.jpg",
      "auth_type": "permanent",
      "auth_type_label": "长期授权",
      "allow_pickup": true,
      "allow_view_record": true,
      "allow_view_status": false,
      "allow_growth_view": false,
      "notify_on_pickup": true,
      "status": 1,
      "review_status": 1,
      "review_status_label": "已通过"
    }
  ],
  "total": 3,
  "page": 1,
  "page_size": 20
}
```

**字段说明**：

- `phone`：脱敏后返回，不返回完整号码
- **不返回**证件号相关字段
- `relation`：关系代码，取值见「关系枚举」；前端按 code 映射中文标签
- `student_id` / `student_name`：该授权人所属的孩子（授权是孩子级，不是家庭级）
- `review_status`：`0` 待审核 / `1` 已通过 / `2` 已拒绝（来源 `domain.PickupAuthorization.ReviewStatus`）
- `status`：`1` 有效 / `0` 已撤销（与 `review_status` 是两套独立状态）
- 「有效接送人」判定口径 = `status = 1` **且** `review_status = 1`

---

### POST `/parent/pickup/authorized-persons`

> 添加授权接送人
>
> 授权人挂在该孩子下：`student_id` 走 **query**（与列表接口一致），body 中的同名字段被忽略。
> `relation` 必须是「关系枚举」中的 9 个 code（过渡期也接受 9 个中文标签，服务端归一化为 code）；
> 未知值返回校验错。

```json
// Request: ?student_id=1
{
  "name": "爷爷（王建国）",
  "relation": "grandfather",
  "phone": "13600005555",
  "sms_code": "123456",
  "id_card_number": "3101...",
  "avatar": "https://cdn.example.com/uploads/pickup/person_new.jpg",
  "auth_type": "permanent",
  "allow_pickup": true,
  "allow_view_record": true,
  "allow_view_status": false,
  "allow_growth_view": false,
  "notify_on_pickup": true
}

// Response
{
  "id": 4,
  "status": 0,
  "status_label": "待审核",
  "created_at": "2026-03-19T10:00:00Z"
}
```

---

### PUT `/parent/pickup/authorized-persons/:id`

> 编辑授权接送人
>
> `relation` 如提交，校验规则同添加接口（9 个 code，过渡期兼容 9 个中文标签并归一化）。

```json
// Request
{
  "name": "二姨（陈梅）",
  "relation": "aunt",
  "phone": "137****3333",
  "avatar": "https://cdn.example.com/uploads/pickup/person_new.jpg"
}

// Response
{
  "id": 4,
  "updated_at": "2026-03-19T10:30:00Z"
}
```

---

### DELETE `/parent/pickup/authorized-persons/:id`

> 删除授权接送人

```json
// Response
{}
```
