# 23. 膳食管理（带量食谱）

> 对应前端页面：`admin/pages/diet/config.vue`（基本参数）、`admin/pages/diet/plans/index.vue`（膳食计划）、`admin/pages/diet/plans/[id].vue`（周食谱编辑器）、`admin/pages/diet/plans/[id]/analysis.vue`（营养分析）、`admin/pages/diet/recipes/index.vue`（菜谱库）、`admin/pages/diet/foods/index.vue`（资源库）
>
> 基础路径：`/api/v1/admin`，需 `Authorization: Bearer <token>`，角色 `admin`。
>
> 权限码：`diet:view`（查看配置/食物/菜谱/周食谱/模板/报表）、`diet:manage`（维护配置/食物/菜谱/周食谱）、`diet:export`（导出营养分析 Excel）。**全部接口须逐条登记进 `middleware/admin_perm.go`**（fail-closed，漏登记会使服务启动失败）。
>
> Casbin：`admin` 组为通配时 `rbac_policy.csv` 不改；实现前先核对，若策略不是通配则补 `/api/v1/admin/diet/*` 的 policy（design D7）。
>
> 状态：🔲 未实现（变更 `add-quantified-recipe`）。
>
> 契约变更：本次新增本文件与 `README.md` 索引；学年复用既有 `GET /admin/settings/academic-years`，本文件不重复定义。
>
> 字段基线：参考产品页面/接口（ADR-0004）与 `docs/research/2026-11-01-oyujian-diet/notes.md`、`docs/research/2026-11-01-nutrition-data/notes.md`。字段溯源见文末。

## 接口清单（27 个）

| 分组 | 方法 | 路径 | 权限 | 说明 |
|---|---|---|---|---|
| 配置 | GET | `/admin/diet/config` | `diet:view` | 供餐配置（单例，缺省返回默认值） |
| 配置 | PUT | `/admin/diet/config` | `diet:manage` | 保存供餐配置（整组替换餐次） |
| 分类 | GET | `/admin/diet/food-categories` | `diet:view` | 全局两级食物分类树 |
| 食物 | GET | `/admin/diet/foods` | `diet:view` | 食物列表（列表列，不含 27 项全量） |
| 食物 | POST | `/admin/diet/foods` | `diet:manage` | 新增园所食物 |
| 食物 | GET | `/admin/diet/foods/:id` | `diet:view` | 食物详情（27 项全量） |
| 食物 | PUT | `/admin/diet/foods/:id` | `diet:manage` | 修改食物 |
| 食物 | DELETE | `/admin/diet/foods/:id` | `diet:manage` | 删除食物（被菜谱引用时拒绝） |
| 食物 | POST | `/admin/diet/foods/batch-delete` | `diet:manage` | 批量删除（逐条判定引用） |
| 食物 | GET | `/admin/diet/foods/:id/replace-candidates` | `diet:view` | 同类替换候选 |
| 菜谱 | GET | `/admin/diet/recipes` | `diet:view` | 菜谱列表 |
| 菜谱 | POST | `/admin/diet/recipes` | `diet:manage` | 新增/替换菜谱 |
| 菜谱 | GET | `/admin/diet/recipes/:id` | `diet:view` | 菜谱详情（配料 + 营养） |
| 菜谱 | PUT | `/admin/diet/recipes/:id` | `diet:manage` | 改名 / 改配方 |
| 菜谱 | DELETE | `/admin/diet/recipes/:id` | `diet:manage` | 删除菜谱（软删除） |
| 菜谱 | POST | `/admin/diet/recipes/merge` | `diet:manage` | 合并重复菜谱 |
| 菜谱 | GET | `/admin/diet/recipes/suggest` | `diet:view` | 编辑器菜名联想 |
| 食谱 | GET | `/admin/diet/plans` | `diet:view` | 周食谱列表 |
| 食谱 | POST | `/admin/diet/plans` | `diet:manage` | 新建周食谱（自定义/模板/复制） |
| 食谱 | GET | `/admin/diet/plans/:id` | `diet:view` | 周食谱详情（含配置快照与明细） |
| 食谱 | PUT | `/admin/diet/plans/:id` | `diet:manage` | 整单保存明细 |
| 食谱 | DELETE | `/admin/diet/plans/:id` | `diet:manage` | 删除周食谱（软删除） |
| 食谱 | POST | `/admin/diet/plans/:id/clear` | `diet:manage` | 清空明细（保留食谱） |
| 模板 | GET | `/admin/diet/templates` | `diet:view` | 本园可用模板列表（只读） |
| 模板 | GET | `/admin/diet/templates/:id` | `diet:view` | 模板详情 |
| 报表 | GET | `/admin/diet/plans/:id/analysis` | `diet:view` | 营养分析（7 组表 + 结论） |
| 报表 | GET | `/admin/diet/plans/:id/analysis/export` | `diet:export` | 导出营养分析 Excel（xlsx 二进制） |

---

## 通用约定

### 响应包络

除导出接口外，响应均为：

```json
{ "code": 0, "message": "success", "data": { } }
```

下文只写 `data` 部分。错误统一走既有包络：

| HTTP | code | 场景 |
|---:|---:|---|
| 404 | 40400 | 食物 / 菜谱 / 食谱 / 模板不存在，或跨园所访问 |
| 409 | 40900 | 菜谱名称冲突（需前端弹「替换 / 仅本次」）、周食谱同一学年重名、食物仍被菜谱引用 |
| 422 | 42200 | 参数与业务校验失败（餐次时间/占比、重量口径、克数、菜名等） |
| 403 | 40300 | 缺少 `diet:*` 权限 |

### 分页

列表接口统一用 `?page=1&page_size=20`，响应：

```json
{ "list": [], "total": 0, "page": 1, "page_size": 20 }
```

`page_size` 上限 100。不分页的接口（分类、联想、候选、模板列表）只返回 `list`。

### 日期与时间

- 日期：`YYYY-MM-DD`（周食谱起止、明细日期）。
- 餐次时间：`HH:mm`（24 小时制，两位补零）。
- 时间戳：ISO 8601（`2026-11-01T09:00:00Z`），仅出现在 `created_at`/`updated_at`。

### 营养字段（27 项）

键名、中文名、单位与参考产品 `nutri*` 字段一一对应。**数值均为「每 100g 可食部」的浓度值**；缺测为 `null`，前端显示「—」，**任何合计都不得把 null 当 0**。

| key | 中文名 | 单位 | 参考字段 |
|---|---|---|---|
| `energy` | 能量（热量） | 千卡 | `nutriEnergy` |
| `heat` | 热量 | 千焦 | `nutriHeat`（= `energy × 4.186`） |
| `protein` | 蛋白质 | 克 | `nutriProtein` |
| `fats` | 脂肪 | 克 | `nutriFats` |
| `carb` | 碳水化合物 | 克 | `nutriCarbohydrate` |
| `dietary_fiber` | 不溶性纤维 | 克 | `nutriDietaryFiber` |
| `ash` | 灰分 | 克 | `nutriAsh` |
| `water` | 水分 | 克 | `nutriWater` |
| `cholesterol` | 胆固醇 | 毫克 | `nutriCholesterol` |
| `retinol` | 视黄醇 | 微克 | `nutriRetinol` |
| `carotin` | 胡萝卜素 | 微克 | `nutriCarotin` |
| `vitamin_a` | 维生素A | 微克 RE | `nutriVitaminA` |
| `vitamin_b1` | 硫胺素（维生素B1） | 毫克 | `nutriVitaminB1` |
| `vitamin_b2` | 核黄素（维生素B2） | 毫克 | `nutriVitaminB2` |
| `nicotinic_acid` | 尼克酸（烟酸） | 毫克 | `nutriNicotinicAcid` |
| `vitamin_c` | 维生素C | 毫克 | `nutriVitaminC` |
| `vitamin_e` | 维生素E | 毫克 α-TE | `nutriVitaminE` |
| `calcium` | 钙 | 毫克 | `nutriCa` |
| `phosphorus` | 磷 | 毫克 | `nutriP` |
| `potassium` | 钾 | 毫克 | `nutriK` |
| `sodium` | 钠 | 毫克 | `nutriNa` |
| `magnesium` | 镁 | 毫克 | `nutriMg` |
| `iron` | 铁 | 毫克 | `nutriFe` |
| `zinc` | 锌 | 毫克 | `nutriZn` |
| `selenium` | 硒 | 微克 | `nutriSe` |
| `copper` | 铜 | 毫克 | `nutriCu` |
| `manganese` | 锰 | 毫克 | `nutriMn` |

> **契约裁剪**：`retinol` / `carotin` 只出现在食物列表、食物维护（详情/新建/修改）与周食谱明细的 `nutrition_snapshot` 中，**不出现在任何报表响应**（营养分析 12 项数值、结论、导出都不含）。列表收录这两列是因为参考产品食物列表页确实展示它们（调研 notes.md §2.6），报表侧仍按「契约字段必须有页面出处」裁剪。这仍是契约裁剪，不是数据模型裁剪——`Food` 与营养快照保留 27 项。

### 枚举

**餐次 `meal_key`**（固定 6 个，顺序固定）：

| meal_key | 名称 | 默认时间 | 默认供能占比 |
|---|---|---|---:|
| `breakfast` | 早餐 | 07:00 | 25 |
| `morning_snack` | 早点 | 09:00 | 5 |
| `lunch` | 午餐 | 11:00 | 35 |
| `afternoon_snack` | 午点 | 14:30 | 5 |
| `dinner` | 晚餐 | 17:00 | 25 |
| `evening_snack` | 晚点 | 20:00 | 5（默认不启用） |

**托管形式 `care_type`**：`day_care` 日托 / `full_care` 全托。

**重量口径 `food_input_type`**：仅 `market_weight`（市品重量）。其他值一律 422。

**周食谱创建方式 `created_method`**：`custom` 自定义 / `template` 选用模板 / `copy` 复制历史。

**食物来源 `source`**：`system` 平台预置 / `custom` 园所自建。

**膳食构成 16 类 `group16`**（20 个一级类合并；报告页只渲染标 `*` 的 10 类）：

| key | 名称 | key | 名称 |
|---|---|---|---|
| `grain` * | 谷类 | `nut` | 坚果 |
| `tuber` * | 薯类 | `instant_food` | 速食小吃（13+14） |
| `vegetable` * | 蔬菜（3+4） | `beverage` | 饮料 |
| `fruit` * | 水果 | `confection` | 糖果蜜饯 |
| `meat` * | 畜禽肉类（6+7） | `oil` * | 油脂类（17+18） |
| `egg` * | 蛋类 | `condiment` | 调味品 |
| `aquatic` * | 水产类 | `other` | 其他 |
| `soybean` * | 大豆 | `dairy` * | 乳制品 |

**食物属性 `source_flags`**：`animal` 动物性 / `plant` 植物性 / `milk` 乳类 / `meat` 肉类。数组形式返回，可多选。

**采购分类 `group12`**：沿用参考产品 37–48（37 谷、38 豆、39 肉、40 蛋、41 奶、42 蔬菜、43 糕点、44 水果、45 调味品、46 海鲜、47 油、48 坚果）；`null` 表示无槽位（薯类/饮料/糖蜜饯）。仅食物维护使用，本变更报表不消费。

---

## 一、供餐配置

### GET `/admin/diet/config`

> 供餐配置是园所单例。首次读取返回默认值（日托、5 餐、市品重量、系数 0.95）。

```json
{
  "care_type": "day_care",
  "care_type_label": "日托",
  "food_input_type": "market_weight",
  "food_input_type_label": "市品重量",
  "coefficient": 0.95,
  "meals": [
    { "meal_key": "breakfast",       "name": "早餐", "time": "07:00", "energy_percent": 25, "sort": 1, "enabled": true },
    { "meal_key": "morning_snack",   "name": "早点", "time": "09:00", "energy_percent": 5,  "sort": 2, "enabled": true },
    { "meal_key": "lunch",           "name": "午餐", "time": "11:00", "energy_percent": 35, "sort": 3, "enabled": true },
    { "meal_key": "afternoon_snack", "name": "午点", "time": "14:30", "energy_percent": 5,  "sort": 4, "enabled": true },
    { "meal_key": "dinner",          "name": "晚餐", "time": "17:00", "energy_percent": 25, "sort": 5, "enabled": true },
    { "meal_key": "evening_snack",   "name": "晚点", "time": "20:00", "energy_percent": 5,  "sort": 6, "enabled": false }
  ]
}
```

- `coefficient`（折算系数）**只读**：= 已启用餐次 `energy_percent` 之和 ÷ 100，默认 0.95。
- `name` / `sort` **只读**：按 `meal_key` 固定顺序由服务端写入；参考产品也不允许改名或重排餐次。

### PUT `/admin/diet/config`

> 整组保存餐次（6 个 `meal_key` 必须各出现一次）。保存后只影响此后新建的周食谱。

```json
// Request
{
  "care_type": "day_care",
  "food_input_type": "market_weight",
  "meals": [
    { "meal_key": "breakfast",       "time": "07:00", "energy_percent": 25, "enabled": true },
    { "meal_key": "morning_snack",   "time": "09:00", "energy_percent": 5,  "enabled": true },
    { "meal_key": "lunch",           "time": "11:00", "energy_percent": 35, "enabled": true },
    { "meal_key": "afternoon_snack", "time": "14:30", "energy_percent": 5,  "enabled": true },
    { "meal_key": "dinner",          "time": "17:00", "energy_percent": 25, "enabled": true },
    { "meal_key": "evening_snack",   "time": "20:00", "energy_percent": 5,  "enabled": false }
  ]
}

// Response（与 GET 同构）
{ "care_type": "day_care", "coefficient": 0.95, "meals": [ "..." ] }
```

校验（失败一律 422，配置保持原值）：

1. `meals` 恰好包含 6 个已知 `meal_key`，各一次。
2. 已启用餐次的时间严格递增（早餐 < 早点 < 午餐 < 午点 < 晚餐 < 晚点）。
3. 至少一餐启用；已启用餐次的 `energy_percent` 必须 > 0。
4. 相邻餐次共享上限（`care_type=day_care` 时早餐+早点 ≤ 29，否则 ≤ 30；午餐+午点 ≤ 39；晚餐+晚点 ≤ 30）。
5. `food_input_type` 只接受 `market_weight`，传 `edible_weight` 等值返回 422（TC-NEW-DIE-0007）。
6. `care_type` 只接受 `day_care` / `full_care`。

---

## 二、食物分类

### GET `/admin/diet/food-categories`

> 全局只读两级树（无租户字段）。一级 20 类，二级按 `parent_id` 挂载。

```json
{
  "list": [
    {
      "id": 1, "parent_id": 0, "level": 1, "name": "谷类及制品",
      "children": [
        { "id": 21, "parent_id": 1, "level": 2, "name": "小麦" },
        { "id": 22, "parent_id": 1, "level": 2, "name": "稻米" }
      ]
    },
    { "id": 2, "parent_id": 0, "level": 1, "name": "薯类、淀粉及制品", "children": [ "..." ] }
  ]
}
```

---

## 三、食物库

> 多租户：每个园所拥有自己的食物副本；`kg=0` 是平台预置哨兵，**任何响应都不得返回 `kg=0` 行**。
> 删除规则：被菜谱（`RecipeItem`）引用 → 拒绝；只被历史周食谱引用 → 允许（历史行有名称与营养快照）。

### GET `/admin/diet/foods`

> 列表只返回参考产品列表页展示的列（名称/来源/食部/核心营养），不含 27 项全量。

```json
// Query: ?keyword=面&category_id=1&page=1&page_size=20
// category_id 可传一级（含其全部二级）或二级分类；keyword 按名称模糊匹配

// Response
{
  "list": [
    {
      "id": 10001,
      "name": "面条",
      "spell_code": "MT",
      "category_id": 21,
      "category_name": "小麦",
      "source": "system",
      "edible_percent": 100,
      "energy": 286,
      "protein": 8.3,
      "fats": 0.7,
      "carb": 61.9,
      "retinol": null,
      "carotin": null,
      "calcium": 12,
      "iron": 2.5,
      "zinc": 0.9,
      "phosphorus": 112,
      "nicotinic_acid": 2.5,
      "vitamin_a": 0,
      "vitamin_b1": 0.35,
      "vitamin_b2": 0.1,
      "vitamin_c": 0
    }
  ],
  "total": 1744,
  "page": 1,
  "page_size": 20
}
```

> 列表 15 项营养取自参考列表页的核心列（能量/蛋白质/脂肪/碳水/视黄醇/胡萝卜素/钙/铁/锌/磷/尼克酸/维生素A/B1/B2/C）。
> `retinol` / `carotin` 可空，缺失时同样按「—」渲染；它们只出现在食物列表、详情与新建/修改中，**营养分析、结论与导出不得返回**。
> 任一值为 `null` 时按「—」渲染。

### POST `/admin/diet/foods`

> 新增园所自建食物（`source = custom`）。HTTP **201**。

```json
// Request
{
  "name": "自制南瓜饼",
  "category_id": 62,
  "edible_percent": 100,
  "group12": 43,
  "group16": "instant_food",
  "source_flags": ["plant"],
  "nutrients": {
    "energy": 210, "heat": 879.06, "protein": 4.2, "fats": 6.1, "carb": 33.5,
    "dietary_fiber": null, "ash": null, "water": null, "cholesterol": null,
    "retinol": null, "carotin": null, "vitamin_a": null, "vitamin_b1": null,
    "vitamin_b2": null, "nicotinic_acid": null, "vitamin_c": null, "vitamin_e": null,
    "calcium": 28, "phosphorus": 60, "potassium": 120, "sodium": 15,
    "magnesium": 12, "iron": 0.8, "zinc": 0.4, "selenium": null, "copper": null, "manganese": null
  }
}

// Response
{ "id": 12001, "name": "自制南瓜饼", "spell_code": "ZZNGB" }
```

- `name`、`category_id`（叶子分类）、`edible_percent` 必填。
- `spell_code` 不接受客户端输入，由服务端按名称生成拼音码。
- `group12` / `group16` / `source_flags` 可选；缺省时按所选一级分类推导（`group12` 无槽位的三类保持 `null`）。
- `nutrients` 可只传有值的键；**未传的键写 NULL，不写 0**。`heat` 缺省且 `energy` 有值时 = `energy × 4.186`。
- `nutrients` 中不允许出现 `edible_percent`（食部在顶层）。

### GET `/admin/diet/foods/:id`

> 食物详情，27 项全量。

```json
{
  "id": 10001,
  "name": "面条",
  "spell_code": "MT",
  "category_id": 21,
  "category_name": "小麦",
  "category_parent_id": 1,
  "source": "system",
  "edible_percent": 100,
  "group12": 37,
  "group16": "grain",
  "source_flags": ["plant"],
  "nutrients": {
    "energy": 286, "heat": 1197.2, "protein": 8.3, "fats": 0.7, "carb": 61.9,
    "dietary_fiber": null, "ash": null, "water": null, "cholesterol": null,
    "retinol": null, "carotin": null, "vitamin_a": null, "vitamin_b1": 0.35,
    "vitamin_b2": 0.1, "nicotinic_acid": 2.5, "vitamin_c": 0, "vitamin_e": null,
    "calcium": 12, "phosphorus": 112, "potassium": null, "sodium": null,
    "magnesium": null, "iron": 2.5, "zinc": 0.9, "selenium": null, "copper": null,
    "manganese": null
  }
}
```

### PUT `/admin/diet/foods/:id`

> 修改食物。字段均可选；`nutrients` 一旦提交即**整体替换**（未提交的键写 NULL）。
> 修改平台预置副本会把该行标记为「园所已修改」，平台升级不会再覆盖它（`is_customized` 内部标记，不对外返回）。

```json
// Request（与 POST 同构，字段均可选）
{ "name": "面条（标准粉）", "edible_percent": 100, "nutrients": { "energy": 286 } }

// Response
{ "id": 10001 }
```

### DELETE `/admin/diet/foods/:id`

> 软删除。被本园任一菜谱引用时返回 409 `食物仍被菜谱引用，无法删除`；只被历史周食谱引用时允许删除。

```json
// Response
{}
```

### POST `/admin/diet/foods/batch-delete`

> 批量删除。逐条判定：被菜谱引用或不存在/跨园的跳过，不中断整批。

```json
// Request
{ "ids": [10001, 10002, 10003] }

// Response
{
  "deleted": 2,
  "skipped": [
    { "id": 10003, "name": "鸡蛋", "reason": "referenced_by_recipe" }
  ]
}
```

`reason` 取值：`referenced_by_recipe` / `not_found`。

### GET `/admin/diet/foods/:id/replace-candidates`

> 同类替换候选：与当前食物**同一个一级分类**下的其他食物（不限于叶子分类，这样「猪瘦肉 → 牛腩」可命中）。按名称排序。

```json
// Query: ?keyword=&limit=20   limit 默认 20，上限 100

// Response
{
  "list": [
    { "id": 10021, "name": "牛腩", "spell_code": "NN", "category_id": 45, "category_name": "牛" }
  ]
}
```

---

## 四、菜谱库

> 菜谱 = 一组「食物 + 一人均量克数」的命名快照。园内名称唯一由服务层保证；重名时走「替换 / 仅本次」分支。
> 菜谱变更不影响历史周食谱（历史行存菜名与营养快照）。

### GET `/admin/diet/recipes`

> 列表只返回名称、拼音码、配料数。

```json
// Query: ?keyword=云吞&page=1&page_size=20

// Response
{
  "list": [
    { "id": 1, "name": "鲜菇鲜肉云吞", "spell_code": "XGXRYT", "item_count": 3 }
  ],
  "total": 2360,
  "page": 1,
  "page_size": 20
}
```

### POST `/admin/diet/recipes`

> 保存菜谱。名称不存在 → 新增；名称已存在且 `replace=false` → 409（前端弹「替换原菜谱 / 否，仅限本次使用」）；
> 选「仅本次」= **不调用本接口**，把配方留在当前周食谱里；选「替换」→ 以 `replace=true` 重试，覆盖原配方。

```json
// Request
{
  "name": "鲜菇鲜肉云吞",
  "items": [
    { "food_id": 10001, "grams": 10 },
    { "food_id": 10005, "grams": 5 },
    { "food_id": 10011, "grams": 5 }
  ],
  "replace": false
}

// Response 201（新增）或 200（替换）
{ "id": 1, "name": "鲜菇鲜肉云吞", "spell_code": "XGXRYT", "item_count": 3 }

// 同名冲突（replace=false）
// HTTP 409, code 40900
{ "code": 40900, "message": "菜谱名称已存在" }
```

- `name` 必填非空；`items` 非空；每个 `grams > 0`。
- 同一菜谱内同一种食物的多行在保存时**合并为一行，克数相加**（TC-NEW-DIE-0021）。
- `replace=true` 且无同名菜谱时等同新增。

### GET `/admin/diet/recipes/:id`

> 菜谱详情：配料明细与各项食物当前的 27 项营养（编辑器载入配方时一次取全）。

```json
{
  "id": 1,
  "name": "鲜菇鲜肉云吞",
  "spell_code": "XGXRYT",
  "items": [
    {
      "food_id": 10001,
      "food_name": "小麦粉",
      "grams": 10,
      "sort": 1,
      "nutrients": { "energy": 344, "protein": 11.2, "...": "27 项，同食物详情" }
    }
  ]
}
```

### PUT `/admin/diet/recipes/:id`

> 改名与/或改配方（菜谱库管理页）。`items` 提交即整体替换；改名撞上其他菜谱名称 → 409。

```json
// Request
{ "name": "鲜菇鲜肉云吞", "items": [ { "food_id": 10001, "grams": 12 } ] }

// Response
{ "id": 1 }
```

### DELETE `/admin/diet/recipes/:id`

> 软删除。历史周食谱保持原菜名、克数与营养快照。

```json
// Response
{}
```

### POST `/admin/diet/recipes/merge`

> 合并重复菜谱：把源菜谱的同种食物克数累加到目标菜谱，然后删除源菜谱。

```json
// Request
{ "source_id": 2, "target_id": 1 }

// Response
{ "target_id": 1, "merged_item_count": 4 }
```

`source_id == target_id` → 422。

### GET `/admin/diet/recipes/suggest`

> 编辑器菜名联想：按名称包含或拼音码前缀匹配，返回候选（不含配料）。

```json
// Query: ?keyword=云吞&limit=18   limit 默认 18，上限 100

// Response
{
  "list": [
    { "id": 1, "name": "鲜菇鲜肉云吞", "spell_code": "XGXRYT", "item_count": 3 }
  ]
}
```

---

## 五、周食谱

> 周期固定为周一至周日 7 天；名称在同一学年内唯一；无明细的日期允许留空。
> 保存即生效，无草稿/发布状态。

### GET `/admin/diet/plans`

> 列表只返回名称、周期、天数、创建方式。

```json
// Query: ?academic_year_id=31846&keyword=第九&page=1&page_size=20
// academic_year_id 缺省时取当前学年（academic_years.is_current）

// Response
{
  "list": [
    {
      "id": 1017248,
      "name": "第九周",
      "academic_year_id": 31846,
      "start_date": "2026-10-26",
      "end_date": "2026-11-01",
      "day_num": 5,
      "week_of_year": 9,
      "created_method": "custom",
      "created_method_label": "自定义"
    }
  ],
  "total": 151,
  "page": 1,
  "page_size": 20
}
```

### POST `/admin/diet/plans`

> 新建周食谱，三种方式三选一。

```json
// 自定义
{ "name": "第九周", "academic_year_id": 31846, "start_date": "2026-10-26", "end_date": "2026-11-01", "week_of_year": 9, "created_method": "custom" }

// 选用模板
{ "name": "第十周", "academic_year_id": 31846, "start_date": "2026-11-02", "end_date": "2026-11-08", "week_of_year": 10, "created_method": "template", "template_id": 3 }

// 复制历史
{ "name": "第十周", "academic_year_id": 31846, "start_date": "2026-11-02", "end_date": "2026-11-08", "week_of_year": 10, "created_method": "copy", "source_plan_id": 1017248 }

// Response 201
{
  "id": 1017249,
  "name": "第十周",
  "start_date": "2026-11-02",
  "end_date": "2026-11-08",
  "day_num": 5,
  "week_of_year": 10,
  "warnings": []
}
```

行为与校验：

1. `start_date` 必须是周一，`end_date` 必须等于 `start_date + 6 天`。
2. `academic_year_id` 必须属于本园；名称在本学年内唯一，重名 409。
3. `created_method` 决定附加字段：`template` 必须带 `template_id`；`copy` 必须带 `source_plan_id`；自定义不接受这两个字段。
4. 创建时把当前供餐配置、折算系数、按班级人数加权的 11 项 DRIs 目标值与权重固化进 `config_snapshot`；此后修改配置不回溯。
5. 模板：复制模板条目到新食谱，`day_num` = 模板有效天数；模板缺失的食物跳过，写入 `warnings`。
6. 复制历史：按**星期几**映射到新周期（周一→新周一），用当前食物库重新固化营养快照；源食谱里已删除的食物跳过，写入 `warnings`。
7. `week_of_year` 可选，由前端新建向导填写（如「第九周」对应 9）；缺省写 0，只作展示，不参与任何计算。
8. 自定义创建的空白食谱 `day_num = 0`。

### GET `/admin/diet/plans/:id`

> 详情：配置快照 + 全部明细。快照与 `nutrition_snapshot` 都是**只读**返回，不接受写入。

```json
{
  "id": 1017248,
  "name": "第九周",
  "academic_year_id": 31846,
  "start_date": "2026-10-26",
  "end_date": "2026-11-01",
  "day_num": 5,
  "week_of_year": 9,
  "created_method": "custom",
  "created_method_label": "自定义",
  "source_template_id": null,
  "source_plan_id": null,
  "config_snapshot": {
    "care_type": "day_care",
    "food_input_type": "market_weight",
    "coefficient": 0.95,
    "meals": [
      { "meal_key": "breakfast", "name": "早餐", "time": "07:00", "energy_percent": 25, "sort": 1, "enabled": true }
      // 其余 5 个餐次同 GET /admin/diet/config
    ],
    "dri_targets": [
      { "nutrient_key": "energy", "label": "能量", "unit": "千卡", "value": 1484.26 },
      { "nutrient_key": "protein", "label": "蛋白质", "unit": "克", "value": 50.7 }
      // 共 11 项：能量/蛋白质/脂肪/碳水/维生素A/B1/B2/C/钙/锌/铁
    ],
    "dri_weights": [
      { "age_band": 3, "weight": 0.267 },
      { "age_band": 4, "weight": 0.326 },
      { "age_band": 5, "weight": 0.408 },
      { "age_band": 6, "weight": 0 }
    ]
  },
  "items": [
    {
      "id": 1,
      "date": "2026-10-26",
      "weekday": 1,
      "meal_key": "lunch",
      "recipe_name": "肉酱意大利面",
      "recipe_id": 12,
      "food_id": 10001,
      "food_name": "面条",
      "grams": 30,
      "sort": 1,
      "nutrition_snapshot": {
        "edible_percent": 100,
        "energy": 286,
        "heat": 1197.2,
        "...": "27 项，同食物详情；缺测为 null"
      }
    }
  ]
}
```

> `nutrition_snapshot` 供编辑器实时小计与报表计算，返回的是保存当时的浓度值；食物库后续修改不影响它。
> `weekday` 由 `date` 推导（1=周一 … 7=周日）。

### PUT `/admin/diet/plans/:id`

> **整单替换**：提交完整 `items` 数组。服务端在一个事务内校验 → 同菜同食物合并 → 用当前食物库重新生成营养快照 → 替换全部明细。
> 保存即生效；清空与删除需前端二次确认。

```json
// Request
{
  "name": "第九周（修订）",
  "items": [
    {
      "date": "2026-10-26",
      "meal_key": "lunch",
      "recipe_name": "肉酱意大利面",
      "recipe_id": 12,
      "food_id": 10001,
      "food_name": "面条",
      "grams": 30
    }
  ]
}

// Response
{ "id": 1017248, "day_num": 5, "item_count": 187 }
```

规则：

1. `items` 必填；空数组等同清空明细（前端仍需二次确认）。
2. `date` 必须在 `[start_date, end_date]` 内；`meal_key` 必须是快照里已启用的餐次。
3. `recipe_name` 非空；`grams > 0`；`food_id` 必填且必须是本园在用的食物。
4. 同一道菜（同 `date` + `meal_key` + `recipe_name`）内的同一种食物**合并为一行，克数相加**。
5. `food_name` 以服务端读取的食物名称为准（客户端传值仅作旧数据兜底）。
6. `sort` 不接受客户端输入：服务端按 `items` 数组顺序在 `(date, meal_key)` 分组内重写；位置调整 = 调整数组顺序。
7. `day_num` 重算为有明细的不同日期数。
8. 校验失败返回 422，`message` 必须包含具体菜名与食物名（如「肉酱意大利面」的「猪瘦肉」一人均量必须大于 0）。
9. `name` 可选；改名撞同学年其他食谱 → 409。

### DELETE `/admin/diet/plans/:id`

> 软删除。删除后列表与营养分析都不再返回该食谱。

```json
// Response
{}
```

### POST `/admin/diet/plans/:id/clear`

> 清空全部明细，保留食谱本身；`day_num` 归零。前端负责二次确认。

```json
// Response
{ "id": 1017248, "day_num": 0 }
```

---

## 六、模板食谱

> 模板按园种子化、只读；条目引用本园 `food_id`，不跨租户引用。
> 选用模板后复制成独立的周食谱，后续修改不影响模板。

### GET `/admin/diet/templates`

```json
{
  "list": [
    {
      "id": 3,
      "name": "湖南省食谱模板01",
      "day_num": 5,
      "meal_keys": ["breakfast", "morning_snack", "lunch", "afternoon_snack", "dinner"],
      "item_count": 46
    }
  ]
}
```

### GET `/admin/diet/templates/:id`

```json
{
  "id": 3,
  "name": "湖南省食谱模板01",
  "day_num": 5,
  "meal_keys": ["breakfast", "morning_snack", "lunch", "afternoon_snack", "dinner"],
  "items": [
    { "weekday": 1, "meal_key": "breakfast", "recipe_name": "肉酱意大利面", "food_id": 10001, "food_name": "面条", "grams": 30, "sort": 1 }
  ]
}
```

---

## 七、营养分析

### GET `/admin/diet/plans/:id/analysis`

> 只读该食谱的 `config_snapshot` 与明细的 `nutrition_snapshot`，返回 7 组表与结论。
> 无明细时返回 422 `该食谱暂无明细，无法生成营养分析`。

7 组表与响应键的对应（参考产品内部字段名 → 本契约键名）：

| # | 参考产品字段 | 契约键名 | 名称 |
|---|---|---|---|
| 1 | `nutrition` | `nutrition` | 营养分析表（12 项数值 × 摄入/DRIs/比较%） |
| 2 | `energy` | `macro_energy` | 三大营养素供能 |
| 3 | `scgnb` | `meal_energy` | 三餐供能比 |
| 4 | `ssgcb` | `composition` | 膳食构成（16 类，页面渲染 10 类） |
| 5 | `tgly` | `calcium_iron` | 钙铁来源 |
| 6 | `dbz` | `protein_source` | 蛋白质来源 |
| 7 | `fats` | `fat_source` | 脂肪来源 |
| — | `analysis` | `conclusion` | 自动结论（结构化） |

通用口径（必须按此实现，不得「修正」）：

- 可食部克数 = `grams × edible_percent / 100`。
- 平均每人每日（`intake`）= `Σ(可食部克数 × 每100g营养素 / 100) ÷ day_num ÷ 系数`；NULL 跳过。
- 比较% = `intake ÷ dri × 100`（等价于 `原始人均 ÷ (dri × 系数)`）；`dri` 为全日原值，不缩放。
- `heat` 的 `dri` = `energy` 的 `dri × 4.186`（快照只存 11 项，千焦不在其中）；脂肪/碳水目标值已在快照里派生。
- 评价：`< 80%` 低、`80%–120%`（含端点）合格、`> 120%` 高。

```json
{
  "plan": {
    "id": 1017248,
    "name": "第九周",
    "start_date": "2026-10-26",
    "end_date": "2026-11-01",
    "day_num": 5,
    "coefficient": 0.95
  },

  "nutrition": {
    "items": [
      { "key": "energy",     "label": "热量",     "unit": "千卡", "intake": 1341.75, "dri": 1484.26, "compare_percent": 90.4,   "comment": "合格" },
      { "key": "heat",       "label": "热量",     "unit": "千焦", "intake": 5616.57, "dri": 6213.11, "compare_percent": 90.4,   "comment": "合格" },
      { "key": "protein",    "label": "蛋白质",   "unit": "克",   "intake": 52.46,   "dri": 50.7,    "compare_percent": 103.48, "comment": "合格" },
      { "key": "fats",       "label": "脂肪",     "unit": "克",   "intake": 42.74,   "dri": 49.47,   "compare_percent": 86.39,  "comment": "合格" },
      { "key": "carb",       "label": "碳水化合物","unit": "克",  "intake": 196.55,  "dri": 209.05,  "compare_percent": 94.02,  "comment": "合格" },
      { "key": "vitamin_a",  "label": "维生素A",  "unit": "微克", "intake": 580.74,  "dri": 573.31,  "compare_percent": 101.3,  "comment": "合格" },
      { "key": "vitamin_b1", "label": "维生素B1", "unit": "毫克", "intake": 0.68,    "dri": 0.67,    "compare_percent": 102.12, "comment": "合格" },
      { "key": "vitamin_b2", "label": "维生素B2", "unit": "毫克", "intake": 0.56,    "dri": 0.67,    "compare_percent": 83.27,  "comment": "合格" },
      { "key": "vitamin_c",  "label": "维生素C",  "unit": "毫克", "intake": 64.23,   "dri": 67.33,   "compare_percent": 95.4,   "comment": "合格" },
      { "key": "calcium",    "label": "钙",       "unit": "毫克", "intake": 245.66,  "dri": 746.61,  "compare_percent": 32.9,   "comment": "低" },
      { "key": "zinc",       "label": "锌",       "unit": "毫克", "intake": 8.73,    "dri": 11.2,    "compare_percent": 77.91,  "comment": "低" },
      { "key": "iron",       "label": "铁",       "unit": "毫克", "intake": 14.98,   "dri": 12,      "compare_percent": 124.82, "comment": "高" }
    ]
  },

  "macro_energy": {
    "denominator": 1380.7,
    "rows": [
      { "key": "kcal", "label": "千卡", "protein": 202.8, "fats": 445.23, "carb": 836.2 },
      { "key": "kj",   "label": "千焦", "protein": 848.92, "fats": 1863.73, "carb": 3500.33 },
      {
        "key": "percent", "label": "占总热量%",
        "protein": 15.2, "fats": 27.86, "carb": 56.94,
        "recommended": { "protein": "12-15%", "fats": "30-35%", "carb": "50-60%" }
      },
      {
        "key": "grams_ratio", "label": "营养素比例",
        "protein": 1, "fats": 0.81, "carb": 3.75,
        "recommended": { "protein": "1", "fats": "1", "carb": "4-5" }
      }
    ]
  },

  "meal_energy": {
    "meals": [
      { "meal_key": "breakfast",       "label": "早餐", "energy": 174.88, "percent": 11.78 },
      { "meal_key": "morning_snack",   "label": "早点", "energy": 195.89, "percent": 13.2 },
      { "meal_key": "lunch",           "label": "午餐", "energy": 482.46, "percent": 32.51 },
      { "meal_key": "afternoon_snack", "label": "午点", "energy": 73.15,  "percent": 4.93 },
      { "meal_key": "dinner",          "label": "晚餐", "energy": 348.27, "percent": 23.46 },
      { "meal_key": "evening_snack",   "label": "晚点", "energy": 0,      "percent": 0 }
    ],
    "total_percent": 85.88
  },

  "composition": {
    "groups": [
      { "key": "grain",        "label": "谷类",     "grams": 163.62, "displayed": true },
      { "key": "tuber",        "label": "薯类",     "grams": 9.16,   "displayed": true },
      { "key": "vegetable",    "label": "蔬菜",     "grams": 194.2,  "displayed": true },
      { "key": "fruit",        "label": "水果",     "grams": 130.4,  "displayed": true },
      { "key": "meat",         "label": "畜禽肉类", "grams": 100.33, "displayed": true },
      { "key": "egg",          "label": "蛋类",     "grams": 16.53,  "displayed": true },
      { "key": "aquatic",      "label": "水产类",   "grams": 3.79,   "displayed": true },
      { "key": "soybean",      "label": "大豆",     "grams": 18.95,  "displayed": true },
      { "key": "dairy",        "label": "乳制品",   "grams": 40,     "displayed": true },
      { "key": "oil",          "label": "油脂类",   "grams": 13.26,  "displayed": true },
      { "key": "nut",          "label": "坚果",     "grams": 12.21,  "displayed": false },
      { "key": "instant_food", "label": "速食小吃", "grams": 9.47,   "displayed": false },
      { "key": "beverage",     "label": "饮料",     "grams": 1.05,   "displayed": false },
      { "key": "confection",   "label": "糖果蜜饯", "grams": 4.21,   "displayed": false },
      { "key": "condiment",    "label": "调味品",   "grams": 14.44,  "displayed": false },
      { "key": "other",        "label": "其他",     "grams": 0,      "displayed": false }
    ]
  },

  "calcium_iron": {
    "calcium": {
      "total": 245.66,
      "sources": [
        { "key": "animal", "label": "动物性", "value": 105.91, "percent": 43.11 },
        { "key": "plant",  "label": "植物性", "value": 134.12, "percent": 54.6 },
        { "key": "milk",   "label": "乳类",   "value": 69.47,  "percent": 28.28 }
      ]
    },
    "iron": {
      "total": 14.98,
      "sources": [
        { "key": "meat",     "label": "肉类",   "value": 2.35,  "percent": 15.69 },
        { "key": "non_meat", "label": "非肉类", "value": 12.63, "percent": 84.31 }
      ]
    }
  },

  "protein_source": {
    "total": 52.46,
    "items": [
      { "key": "animal",   "label": "动物蛋白", "value": 28.36, "percent": 54.05 },
      { "key": "bean",     "label": "豆类蛋白", "value": 4.43,  "percent": 8.44 },
      { "key": "required", "label": "要求",     "value": 26.23, "percent": 50 }
    ],
    "animal_bean_percent": 62.49
  },

  "fat_source": {
    "total": 42.74,
    "items": [
      { "key": "animal", "label": "动物性", "value": 22.38, "percent": 52.36, "displayed": true },
      { "key": "plant",  "label": "植物性", "value": 8.48,  "percent": 19.84, "displayed": true },
      { "key": "other",  "label": "其他",   "value": 11.88, "percent": 27.8,  "displayed": false }
    ]
  },

  "conclusion": {
    "food_kind_count": 97,
    "condiment_kind_count": 7,
    "non_condiment_kind_count": 90,
    "dri_percent": {
      "energy": 90.4, "protein": 103.48, "fats": 86.39, "carb": 94.02,
      "vitamin_a": 101.3, "vitamin_b1": 102.12, "vitamin_b2": 83.27,
      "vitamin_c": 95.4, "calcium": 32.9, "iron": 124.82, "zinc": 77.91
    },
    "macro_percent": { "protein": 15.2, "fats": 27.86, "carb": 56.94 },
    "macro_ratio": "1 : 0.81 : 3.75",
    "animal_bean_protein_percent": 62.49,
    "suggestions": [
      { "nutrient_key": "calcium", "level": "low", "text": "钙摄入量偏低。建议补充一些牛奶与奶制品鱼类、大豆与豆制品、杏仁、酵母。" },
      { "nutrient_key": "zinc",    "level": "low", "text": "锌摄入量偏低。建议补充一些牛肉、乳酪、蛋鱼、豆类、果仁、麦牙、粗粮。" }
    ]
  }
}
```

> 示例数值取自调研样本，仅示结构。**必须保留的非标准口径**（ADR-0004，不在本变更修 bug）：
>
> - 同一响应存在三个能量总量：`nutrition.energy.intake = 1341.75`、`meal_energy.meals` 合计 = 1274.65（未除 0.95）、`macro_energy.denominator = 1380.7`（蛋白×4+脂肪×9+碳水×4）。三者不做统一。
> - `meal_energy.total_percent = 85.88`，不归一化为 100%。
> - `composition` 返回 16 类、页面只渲染 `displayed=true` 的 10 类。
> - `calcium_iron.calcium` 三来源不互斥（乳类同时计入动物性），合计 126%；铁两来源互斥且穷尽。
> - `protein_source` 只统计动物 + 豆类，三项合计不等于总蛋白。
> - `fat_source` 返回 3 项、页面只渲染 `displayed=true` 的动物性/植物性两列；其他脂 = 总脂肪 − 动物脂 − 植物脂（残差）。
> - 营养分析表页面渲染 **14 列**：在 12 项数值之外，前端插入「视黄醇当量」列（复用 `vitamin_a` 的值）与「胡萝卜素」列（固定显示「-」）。`retinol` / `carotin` 不出现在营养分析响应里。
> - `conclusion.suggestions` 只包含评价为「低」或「高」且有固定话术的营养素；「合格」不触发建议。

### GET `/admin/diet/plans/:id/analysis/export`

> 导出营养分析 Excel。同步生成，直接返回二进制；**不是 JSON 包络**。

```
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename*=UTF-8''%E8%90%A5%E5%85%BB%E5%88%86%E6%9E%90.xlsx
```

- 5 个 sheet：`带量食谱营养分析表`、`带量食谱营养分析图`、`带量食谱三餐供能比`、`带量食谱膳食构成表`、`带量食谱钙铁来源`。
- 内容与页面一致：营养表 14 列（视黄醇当量列复用维生素A，胡萝卜素列「-」）、膳食构成 **10 列**、脂肪来源 **2 列**。
- 不含 `retinol` / `carotin`；`otherfats` 只出现在「营养分析图」数据 sheet。
- sheet 名按页面用词用「三餐供能比」（参考导出误写「三餐功能比」，不复刻错字）。
- 导出失败（如食谱无明细）仍返回 JSON 错误包络。

---

## 附一、字段溯源

| 分组 | 字段来源 |
|---|---|
| 配置 | `notes.md` §2.3（groupObj 表单与接口：托管形式、餐次、时间、占比、重量口径、相邻上限） |
| 食物分类 | `notes.md` §2.10（`getFoodTypeList`：20 个一级类 + 二级） |
| 食物 | `notes.md` §2.6（dietInfo / orgFood、新增食物弹窗 27 项与单位）、§2.9–2.10（group12/group16 映射）、`notes.md`（营养数据）§1.1/§3 |
| 菜谱 | `notes.md` §3.6（getOrgRecipeInfo：食物 + 一人均量克数、拼音码、替换/仅本次） |
| 周食谱 | `notes.md` §2.4 / §2.5（diet、dietInfo 187 行样本）、§3.7（createMethod）、§3.8（三种创建方式） |
| 模板 | `notes.md` §3.8（getAvilableDietary / queryDietary：可用模板、dayNum、餐次集合） |
| 营养分析 | `notes.md` §3.5 / §3.9 / §3.11 / §3.12（7 组表、阈值、DRIs、0.95）、`report-page.jsp` / `print-report.jsp` |
| 导出 | `notes.md` §3.13 与 `api-samples/nutrition-analysis-export.xls` |

**不进入对外响应的字段**：`kindergarten_id`、`created_by`、`updated_by`、`is_customized`、`preset_key`（架构/运维需要）。
**只读返回**：`config_snapshot`、`nutrition_snapshot`、`coefficient`、`sort`（详情中）、`day_num`、`weekday`。
