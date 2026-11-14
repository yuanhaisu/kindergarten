# 相册详情（家长端）

> 生成方式：从代码现状补齐（2026-11-06）

- 源码：`parent/src/pages/album/detail.vue`
- 接口：`backend/api/api-parent/`（班级相册详情）
- 设计系统：`user:nurture-planet-local`，surface=parent

## 信息层级

1. 相册头：标题、日期、分类标签、可见张数。
2. 照片网格：3 列正方形，接口已按可见性过滤，空数组是正常结果。
3. 页脚说明：点开照片可保存原图。
4. 外壳：返回「班级相册」，无 tabBar。

## 覆盖状态

| 状态 | 表现 |
|---|---|
| 正常 | 9 张照片 + 页脚说明 |
| 空 | 「这本相册里暂时没有孩子的照片」+「看看其它相册」 |
| 加载 | 头部 + 3×3 网格骨架 |
| 失败 / 404 | 「相册无法查看」+ 重试 + 返回相册列表；不泄露相册是否存在 |

## Token 映射

- 背景 / 卡片：`--bg`、`--surface`、`--border`
- 文字：`--fg`、`--fg-2`、`--muted`
- 品牌橙：`--accent`、`--accent-deep`、`--accent-wash`、`--accent-gradient`、`--on-accent`
- 圆角：`--radius-card`、`--radius-md`、`--radius-pill`
- 间距 / 字号：`--space-xs…xl`、`--text-xs…2xl`

## 已知差异

- 照片占位用 CSS 渐变 + 线性图标，真实图片由接口下发；原型不引入外网资源。
- 代码的灯箱/保存原图交互不在原型内实现，只标注入口意图。
