# 08. 接送管理

> 对应前端页面：`admin/pages/pickup/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 已实现。

---

## GET `/admin/pickup/overview`

> 全园接送概览（对应前端 `getPickupOverview`）

```json
// Query: ?date=2026-03-19

// Response
{
  "date": "2026-03-19",
  "total_students": 280,
  "arrived": 280,
  "picked_up": 120,
  "not_picked_up": 160,
  "abnormal_count": 2,
  "by_class": [
    { "class_id": 1, "class_name": "小一班", "arrived": 28, "picked_up": 12, "not_picked_up": 16 },
    { "class_id": 2, "class_name": "小二班", "arrived": 29, "picked_up": 15, "not_picked_up": 14 }
  ]
}
```

---

## GET `/admin/pickup/records`

> 全园接送记录（对应前端 `getPickupRecords`）
>
> 明细列对齐 `design-data/admin-05-接送记录页.json`：
> 日期 / 学生姓名 / 班级 / 接送类型 / 接送人 / 接送时间 / **核验人** / 状态 / 操作。
> 其中 `operator` 即「核验人」（园所方登记核验的教职工）。

```json
// Query: ?class_id=&date=2026-03-19&is_abnormal=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 301,
      "date": "2026-03-19",
      "student": { "id": 1, "name": "王小明", "class_name": "小一班" },
      "type": 2,
      "type_label": "接走",
      "method": 1,
      "method_label": "扫码",
      "pickup_name": "奶奶（张三）",
      "pickup_relation": "grandmother",
      "operator": { "id": 5, "nickname": "李师傅" },
      "is_abnormal": false,
      "is_authorized_override": false,
      "recorded_at": "2026-03-19T17:30:25Z"
    }
  ],
  "total": 560,
  "page": 1,
  "page_size": 20
}
```

**字段说明**：

- `operator`：契约语义为「核验人」（对齐设计稿列名），不建双人流程，故无独立「接送人」账号列
- `is_authorized_override`：本次接走是否走了授权例外通道

---

## GET `/admin/pickup/abnormal`

> 异常接送记录（对应前端 `getPickupAbnormal`）

```json
// Query: ?date=&page=1&page_size=20

// Response
{
  "list": [
    {
      "id": 305,
      "student": { "id": 5, "name": "张小强", "class_name": "中一班" },
      "type": 2,
      "type_label": "接走",
      "pickup_name": "陌生人",
      "is_authorized": false,
      "abnormal_reason": "非授权人员接送，已拒绝并通知家长",
      "operator": { "nickname": "李师傅" },
      "recorded_at": "2026-03-19T17:45:00Z"
    }
  ],
  "total": 2
}
```

---

## GET `/admin/pickup/authorized-persons`

> 授权接送人列表（对应前端 `getAuthorizedPersons`）
>
> **本次契约变更：补文档** —— 该端点此前已在 `router.go` 与前端 `admin/api/pickup.ts` 存在，但漏写进本文档。

```json
// Query: ?status=pending&page=1&page_size=20
// status: pending=待审核 / approved=已通过 / rejected=已拒绝，不传查全部

// Response
{
  "list": [
    {
      "id": 3,
      "student": { "id": 1, "name": "王小明", "class_name": "小一班" },
      "name": "奶奶（张三）",
      "relation": "grandmother",
      "phone": "138****8888",
      "auth_type": "permanent",
      "auth_type_label": "长期授权",
      "allow_pickup": true,
      "notify_on_pickup": true,
      "status": 1,
      "review_status": 0,
      "review_status_label": "待审核",
      "audited_at": null,
      "audit_remark": "",
      "created_at": "2026-03-19T09:00:00Z"
    }
  ],
  "total": 2,
  "page": 1,
  "page_size": 20
}
```

**字段说明**：

- `phone` 脱敏返回；**不返回**证件号相关字段
- 排序：待审核优先（`review_status ASC, created_at DESC`）
- `review_status`：`0` 待审核 / `1` 已通过 / `2` 已拒绝

---

## PUT `/admin/pickup/authorized-persons/:id/audit`

> 审核授权接送人（对应前端 `auditPickupAuthorizedPerson`）

```json
// Request
{
  "action": "approve",           // approve / reject
  "remark": ""
}

// Response
{
  "id": 4,
  "status": 1,
  "status_label": "已审核",
  "audited_at": "2026-03-19T10:00:00Z"
}
```

---

## GET `/admin/pickup/overdue`

> 滞留名单 + 各班超时未接计数
>
> 对应 `design-data/admin-05` 的「本月超时未接」统计卡。
> **实时口径**：此刻已入园且仍未接走（有接走记录即从名单移出）。**按学生计**，非事件数。
> 累计口径见下方 `GET /admin/pickup/overdue/count`——两者语义不同，不可混用。

```json
// Query: ?date=2026-03-19&class_id=

// Response
{
  "date": "2026-03-19",
  "overdue_total": 3,
  "by_class": [
    {
      "class_id": 1,
      "class_name": "小一班",
      "pickup_deadline": "16:00",
      "overdue_count": 2,
      "students": [
        {
          "student_id": 1,
          "name": "王小明",
          "avatar": "https://cdn.example.com/avatar/s1.jpg",
          "arrive_time": "07:52",
          "pickup_deadline": "16:00"
        }
      ]
    }
  ]
}
```

**字段说明**：

- `pickup_deadline`：班级放学时刻，未配置为缺省 `16:00`
- 学生挂在 `by_class[].students` 下，**无扁平 `list` 字段**；需要扁平行时由前端摊平
- 超时时长（分钟）后端暂不返回，前端按 `pickup_deadline`（HH:MM）与当前时间推算
- 计数以**学生**为单位；实时口径，被接走即从 `overdue_total` 中消失

---

## GET `/admin/pickup/overdue/count`

> 超时未接**累计**计数（按学生计）。口径见 spec TC-NEW-PKU-111：
> 当日越过班级放学时刻即计一次，**随后被接走也不清零**。
> 与上面列表的实时口径不同——做「立即处理」的告警用列表，做统计卡片用本接口。

```json
// Query: ?date=2026-03-19&class_id=

// Response
{
  "date": "2026-03-19",
  "overdue_not_picked": 5
}
```

**字段说明**：

- `date`：回显的查询日期（未传参时为空串，由后端按当日解析）
- `overdue_not_picked`：当日累计超时未接学生数
