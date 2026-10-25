# 19. 操作日志

> 对应前端页面：`admin/pages/logs/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：✅ 1 个接口已实现。

---

## GET `/admin/logs`

> 操作日志列表（对应前端 `getOperationLogs`）

```json
// Query: ?module=student&operator_id=&start_date=2026-03-19&end_date=2026-03-19&page=1&page_size=20
// module: student / teacher / parent / notice / growth / attendance / pickup / task / points / achievement / announcement / admin / system
// task 模块的公开审核 action：approve_public（老师批准）/ revoke_public（老师撤回或拒绝）/
// cancel_public（家长取消）/ revoke_public_task（关闭任务公开，批量撤销）。

// Response
{
  "list": [
    {
      "id": 1,
      "module": "student",
      "module_label": "学生管理",
      "action": "create",
      "action_label": "创建",
      "target": "学生 赵小刚（2026301）",
      "operator": {
        "id": 100,
        "nickname": "张园长",
        "ip": "192.168.1.100"
      },
      "detail": {
        "student_name": "赵小刚",
        "class_name": "小一班"
      },
      "created_at": "2026-03-19T10:00:00Z"
    },
    {
      "id": 2,
      "module": "teacher",
      "module_label": "老师管理",
      "action": "update",
      "action_label": "编辑",
      "target": "老师 赵老师",
      "operator": {
        "id": 100,
        "nickname": "张园长",
        "ip": "192.168.1.100"
      },
      "detail": {
        "changed_fields": ["education", "class_assignments"]
      },
      "created_at": "2026-03-19T10:30:00Z"
    }
  ],
  "total": 1200,
  "page": 1,
  "page_size": 20
}
```
