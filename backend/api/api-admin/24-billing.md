# 24. 我的套餐

> 对应前端页面：`admin/pages/billing/index.vue`
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 状态：🔲 未实现（变更 `add-saas-commercial-layer`）

园所端只读视图：展示本园当前套餐、已授权模块、到期日与降级状态。**不暴露订单金额、收款与分销信息**——那些属于平台方。

---

## GET `/admin/billing/subscription`

> 查询本园当前套餐与授权状态

```json
// Response
{
  "plan_name": "旗舰版",
  "modules": [
    {
      "code": "growth",
      "name": "成长记录与相册",
      "state": "active",
      "expires_at": "2027-05-05"
    },
    {
      "code": "diet",
      "name": "膳食营养",
      "state": "grace",
      "expires_at": "2027-04-20"
    }
  ],
  "expires_at": "2027-04-20",
  "grace_until": "2027-05-05",
  "state": "grace",
  "student_count": 180,
  "tier_max_students": 150,
  "over_limit": true
}
```

> `state` 取值：`active` 在期 / `grace` 宽限期 / `expired` 已降级。
> `student_count` 为当前在园幼儿数；`tier_max_students` 为最近一张生效订单所购档位的上限（0 表示无上限）。
> `over_limit` 为 `student_count > tier_max_students`，**只作提示，不阻断任何业务**。
> 园所级 `state` 取最紧急的模块状态（降级 > 宽限 > 在期），`expires_at` 取最早到期日。
> `grace_until` 仅在宽限期或降级状态下有值。
> 降级状态下 `modules[].state` 为 `expired`，页面需提示「可查看历史，暂不可新增」。
