# 实施差异记录（归档时补）

记录设计阶段与实际落地之间的差异，供未来回溯。

## 与 design.md 的差异

1. **佣金结算时机**：design 假定状态按到账日推导即可见；实现发现状态只在台账创建时判定一次，满 30 天不会自动转为可结算。补 `CommissionRepo.PromoteEligible`，在台账读取前惰性推进（无定时任务，幂等）。
2. **模块-路由归属**：design 倾向逐条枚举路由；最终采用命名空间匹配（`/growth`、`/album`、`/diet`）。2026-11-07 用户裁定接受该取舍：新增接口不会漏网，代价是删除单条路由不触发启动失败。
3. **试用授权粒度**：design 写「一条试用授权」；因一条授权只承载一个模块，回填迁移按模块展开（6 园 × 5 模块 = 30 行）。
4. **归因冻结的验证方式**：系统没有修改归因的写入路径，DST-0001 以结构性测试覆盖（确认并收款后详情中的分销商与比例不变），而不是构造一个「改归因」请求。

## 实际触碰的层

- Go：domain / repository / service（plan、order、entitlement、distributor、platform、kindergarten_status、auth）
- 中间件：`PlatformAuth`、`RequireActiveKindergarten`、`ModuleGate`
- 路由与 Casbin：`/api/v1/platform/*`、`/common/entitlements`、`/admin/billing/subscription`
- 契约：`backend/api/api-platform/*`、`api-admin/24-billing.md`、`api-common.md`
- Admin：`/platform/*` 控制台（登录、园所、套餐、订单、分销商、佣金）、`/billing` 我的套餐、平台守卫与菜单
- 小程序：两端 entitlements API / utils，老师端工作台与更多页、家长端成长/报告/相册/我的/班级共 6 处入口裁剪
- 迁移：`00002_saas_commercial_layer`、`00003_backfill_commercial_entitlements`、`00004_platform_ops_and_password_flag`
- 测试：service 单测、仓库租户隔离、中间件表驱动、HTTP E2E、无头浏览器 UI E2E、微信 IDE CDP 真实渲染验证
