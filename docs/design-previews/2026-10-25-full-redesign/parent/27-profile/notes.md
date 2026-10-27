# 我的（家长端）

- 源码：`parent/src/pages/profile/index.vue`（tabBar 第 4 项；`onShow` 只读 `auth.getUser()` 本地缓存，**无本页网络请求**）
- 接口：登录态来自 `POST /api/v1/auth/login/wechat` 与 `GET /common/user/profile`（`backend/api/api-common.md`）；`RoleInfo.kindergarten_name` 见 `server/internal/service/auth_service.go`
- 设计系统：`user:nurture-planet-local`，surface=parent
- 生成方式：OpenDesign（run 约 11 分钟，agent 在 20KB 上限下自行裁掉了 `<use>` 上多余的 `viewBox`；交付前按项目终态重新覆盖过 harness 的较早拷贝，并修掉装饰弧溢出视口与硬边两处问题）
- 方向：**紧凑身份卡 + 四段式设置清单**。头部不占大 hero，把「我是谁、在哪个园」压进一张 96px 身份卡；下方 12 个入口按「成长与任务 / 在园日常 / 孩子与班级 / 消息与设置」四段排布，**严格保持源码顺序**；把源码里定义了却从未渲染的 `icon` 键补成线性图标。整页冷雾白底，暖色只出现在头像、角色胶囊与右上角的同心轨道，面积控制在 5% 以内。

## 信息层级

1. 身份卡：56px 渐变头像（昵称首字「乐」）、乐乐妈妈、「家长」暖色胶囊、`138****6688 · 阳光幼儿园`；右上角同心轨道是纯装饰（`aria-hidden`）
2. 成长与任务：亲子任务、积分成就、班级日历、成长报告
3. 在园日常：请假记录、接送记录、授权人管理
4. 孩子与班级：我的孩子、班级信息
5. 消息与设置：通知公告、意见反馈、设置
6. 退出登录：`--danger-wash` 胶囊按钮，点击弹微信风格居中确认框（取消 / 退出登录）
7. 页脚：养成星球 · 家长端
8. 外壳：无返回箭头；tabBar 4 项，**我的**选中

## 覆盖状态

| 状态 | 表现 |
| --- | --- |
| 正常 | 身份卡完整 + 12 个菜单项；退出登录可见 |
| 空 | 身份卡换成「还没有读取到家长信息」+ 主按钮「重新登录」；菜单保留；退出登录隐藏 |
| 加载 | 身份卡换骨架（圆头像 + 两条线）；菜单整体换 3 行骨架卡；退出登录隐藏 |
| 失败 | 身份卡换成「账号信息加载失败 / 网络开小差了，请稍后重试」+「重试」；菜单保留；点重试立即禁用并显示「重试中…」与旋转指示，约 1s 后回正常态并 toast「已重新加载」 |

> 源码本页没有网络请求，空 / 加载 / 失败三态是对「后续接 `GET /common/user/profile`」的预演，不是现状复刻。演示条四个 chip 与 `_shared/shell.js` 的 `mp:state` 同步。

## Token 映射

| 元素 | Token |
| --- | --- |
| 页面底 / 卡片底 | `--bg` / `--surface` |
| 身份卡 | `--surface`、`--radius-2xl`、`--shadow-sm` |
| 头像 | `--accent-gradient`、`--on-brand`、`--radius-pill` |
| 轨道装饰 | `--accent-soft`（径向柔光 + 1px 同心弧，`transparent` 收边） |
| 角色胶囊 | `color-mix(--accent 11%)`、`--accent-deep`、`--radius-pill` |
| 次级文字 | `--muted`（脚注不用 `--hint`，见 `_shared/findings.md` token #3） |
| 菜单卡 | `--surface`、`--radius-xl`、`--shadow-sm`、`--border` 分割线 |
| 图标砖 | `--accent-wash`、`--accent-deep`、`--radius-lg` |
| 退出登录 | `--danger`、`--danger-wash`、`--radius-pill` |
| 弹层 | `--surface`、`--radius-xl`、`--shadow-lg`、`--fg`（遮罩 `.38`） |
| 间距 / 字号 / 动效 | `--space-*`、`--text-*`、`--page-pad-x`、`--motion-fast/base` |

## 与现状的差异

- **退出登录**：源码的 profile 页**没有**退出入口，登出在 `pages/settings/index.vue`（`uni.showModal` → `clearAuth()` → `reLaunch` 到登录页）。原型按批次要求在本页页尾加了主按钮 + 确认框，交互语义与设置页一致。若产品决定保留在设置页，删掉按钮与 dialog 即可，不影响其余结构。
- **幼儿园名**：`user.roles[0].kindergarten_name` 契约里有、源码没用；原型把它放在手机号后，让身份卡一眼能确认「哪个园」。
- **图标**：源码 `menuItems[].icon` 定义了 12 个 key，但模板从未渲染；原型按 key 画成 18px 线性图标。
- **分组**：源码是一张卡直列 12 行；原型分四段并加段标题，条目顺序不变、未增删、未合并。
- **点击行为**：源码按 `switchTab` / `navigateTo` 真跳转；单页原型统一用 toast 占位（当前 tab 提示「已在…」，其余 tab 提示「为原型占位」）。
- **空 / 加载 / 失败**：源码无（只读缓存），属设计补充。

## 遗留 / 需产品确认

- **退出登录放哪**：本页 vs 设置页，需产品定；两处并存会出现双入口。
- 12 项是否要按使用频率重排或收纳（当前严格按源码顺序，没有「常用」置顶，也没有折叠分组）。
- 「我的孩子」在源码里是单孩假设；多孩家庭如何在本页切换，源码与契约都没有表达。
- 身份卡右侧的轨道装饰纯视觉，不承载信息；如嫌装饰感强可直接去掉 `.identity-orbit`，卡片仍是完整的紧凑卡。
- `index.html` 19.6KB，压在批次 20KB 提示线内。
