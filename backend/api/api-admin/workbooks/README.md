# Workbooks · 模块工作簿

> 每个模块 1 份工作簿，记录该模块实现过程中的 **上下文 + 决策 + 进度 + 问题**。
>
> 会话切换时，工作簿是该模块的"状态快照"，新会话读工作簿就能恢复到上次的位置。

---

## 文件命名

与接口规格一一对应：

- `_template.md` — 空白模板（只读，别改）
- `01-dashboard.md` — 对应 `01-dashboard.md`
- `02-classes.md` — 对应 `02-classes.md`
- `...`
- `20-export.md` — 对应 `20-export.md`

---

## 生命周期

```
[未开工] 不存在文件
   ↓ 第一次开工
[进行中] 复制 _template.md 创建，Status: In Progress
   ↓ 每个接口完成
[进行中] 更新"接口完成进度"表
   ↓ 会话暂停
[暂停中] 追加 Session Log 一条；Status: Paused
   ↓ 新会话继续
[进行中] 追加 Session Log 一条；Status: In Progress
   ↓ 所有接口完成 + 联调通过
[已完成] 填 DoD checklist；Status: Done
```

---

## 与其他文档的关系

| 文档 | 作用域 | 关系 |
|---|---|---|
| `EXECUTION_PLAN.md` | 全局方法论 | 工作簿引用它的 §5 Recipe / §6 DoD |
| `PROGRESS.md` | 全局进度表 | 工作簿是"详版"，PROGRESS 是"摘要打勾" |
| `SESSION_HANDOFF.md` | 全局接力卡 | 会话开始时指向"当前工作簿" |
| `workbooks/XX-*.md` | 模块工作簿 | 模块内的长期记忆 |

---

## 使用规则

1. **只有在开始做某模块时才创建该模块工作簿**，不要预先创建全部 20 个。
2. **模块工作簿创建后不要删除**，哪怕模块完成了——它是决策和踩坑的档案。
3. **工作簿的 `Interface Progress` 表**和 `PROGRESS.md` 对应模块的表要保持一致，以工作簿为准（更细）。
4. **每次会话结束**，在工作簿追加一条 `Session Log` 条目。
