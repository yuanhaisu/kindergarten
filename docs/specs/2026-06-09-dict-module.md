# 通用字典模块

> **状态**: ✅已完成
> **创建**: 2026-06-09
> **关联 Skill**: system-enums

## 背景与动机

项目中所有下拉选项、标签、筛选分类、评分等级等 22+ 类枚举值全部硬编码在前端 `.vue` 文件中。产品每改一个标签就要改代码发版。需要建立统一的字典管理系统，管理员在后台配置，前端实时获取。

## 数据模型

两张表，per-kindergarten 多租户隔离：

```
sys_dict_types                    sys_dict_items
├── kindergarten_id               ├── type_id (FK)
├── type_key (同园所唯一)           ├── item_key
├── type_name                     ├── item_label
├── description                   ├── extra (JSON: 颜色/图标)
├── is_system (系统内置不可删)      ├── sort_order
├── sort_order                    ├── is_enabled
└── SoftDeleteModel               └── SoftDeleteModel
```

种子数据：`performance_tag` 类型 + 9 个标签项（乐于助人、积极发言…），幂等写入。

## API 设计

**管理后台** (`/api/v1/admin/dicts`): 9 个端点，覆盖类型和项的完整 CRUD + 批量排序。

**教师端** (`/api/v1/teacher/dicts`): 2 个只读端点 — 按 key 获取 / 批量获取。仅返回 is_enabled=true 的项，按 sort_order 排序。

## 前端交互

- **管理后台**: 新增 `admin/pages/settings/dict.vue` — 左侧类型列表 + 右侧字典项表格，支持行内编辑、启用/禁用开关、弹窗 CRUD
- **教师端**: `teacher/src/pages/growth/create.vue` 的表现标签从硬编码数组改为 `fetchDict("performance_tag")`，失败时 fallback 原值

## 边界与异常

- 系统内置类型（is_system=true）不可删除
- 有子项的类型拒绝删除（需先清空子项）
- 字典 API 失败时前端使用硬编码默认值，不阻塞页面
- 种子数据幂等：已存在的 kindergarten 不会重复创建

## 验收标准

- [x] 三端编译通过
- [x] 启动后端 → AutoMigrate 创建表 → seeder 写入 performance_tag
- [x] `GET /admin/dicts/types` 返回类型列表含 item_count
- [x] Admin CRUD 完整可用
- [x] `GET /teacher/dicts/performance_tag` 返回 9 个标签
- [x] 教师端成长记录创建页从 API 加载标签
