# add-diet-group-objects 进度板

> 工作流：`.claude/skills/slice-delivery/SKILL.md`（适配 admin + Go，无微信小程序）
> base_sha：`076d3a4`
> 提交策略：每个 slice PASS 后 commit + push；分支 `feat/add-quantified-recipe`
> 推进策略：PASS + commit 后自动进入下一 slice；仅在 FAIL/BLOCKED、需产品决策时停下

| Slice | Tasks | 目标 | 状态 | base | head | verdict | commit |
|---|---|---|---|---|---|---|---|
| DGO-S1 | 1.1–1.3 | 模型 + AutoMigrate + 默认对象种子/回填 | passed | 076d3a4 | e87ad00 | .claude/runs/dgo-s1/verdict.json | e87ad00 |
| DGO-S2 | 2.1–2.2 | 仓储 + 集体对象服务 | passed | e87ad00 | 4dcff90 | .claude/runs/dgo-s2/verdict.json | 4dcff90 |
| DGO-S3a | 2.3 | 周食谱绑定对象 + DRIs 加权改造 | passed | 4dcff90 | aac8739 | .claude/runs/dgo-s3a/verdict.json | aac8739 |
| DGO-S3b | 2.4 | handler/router/wire/权限 + 移除旧配置服务 | passed | aac8739 | a197ef4 | .claude/runs/dgo-s3b/verdict.json | a197ef4 |
| DGO-S4 | 3.1–3.2 | 契约：集体对象 + 周食谱对象字段 | passed | a197ef4 | 7dd004a | .claude/runs/dgo-s4/verdict.json | 7dd004a |
| DGO-S5 | 4.1–4.4 | admin API/types + 基本参数页 + 计划页 | passed | 7dd004a | c99eafa | .claude/runs/dgo-s5/verdict.json | c99eafa |
| DGO-S6 | 5.1–5.4 | TEST_CATALOG + 后端重启 + Playwright E2E + 最终验收 | in_progress | c99eafa | | | |

## 关键决策

- 就餐班级选项从 `GET /admin/classes?page_size=200` 实时拉取，按学年 `academic_year` 过滤、按 `grade` 分组全选；不新增班级接口。
- 旧 `diet_meal_configs` 表保留不读写；新表三张，避免 AutoMigrate 无法删除旧唯一索引。
- 不提供集体对象删除；名称唯一由服务层校验。
- 周食谱 `group_object_id` 必填；旧行迁移回填同学年默认对象。

## Deferred P2

- S3a：delta spec 的回退算例写 1484.26，实现为 1485.95（AQR-S6 既有漂移），S6 统一校准并在 TEST_CATALOG 注明。
- S2：seed 测试园所 ID 990994 与 `diet_recipe_test.go` 重复（同包串行暂不冲突）；`ClassRepo` 无 `ListByIDs`，列表按 class_id 逐个查询（有请求内缓存）。
- S3b：重名返回 422（已按实际写入契约）；删除 `diet_meal_config_test.go` 时丢了共享校验的边界用例，S6 补回；周食谱列表暂无可选 `group_object_id` 筛选（design 标注可选，未阻塞验收）。
