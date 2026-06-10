# 老师端小程序 Calicat 设计问题处理清单

来源：Calicat 设计稿 `2034425111476019200`，画布节点 `4ebc2015-baed-4a57-9733-9310e05a608a`。

用途：实现老师端小程序页面时引用，优先处理影响适配、交互、可点击性和开发交付稳定性的问题。

## 1. 页面响应式适配不足

### 问题

设计稿声明支持 320px-430px 宽度，但首页中大量使用固定宽度，例如 `342px` 卡片、`310px` 内容区、`78px` 功能入口。若在 320px 宽设备上按固定宽度实现，卡片和网格可能溢出。

### 影响

- 小屏设备出现横向溢出或内容被裁切。
- 功能入口间距不稳定，文字可能换行或挤压。
- 后续维护多个页面时容易出现重复适配问题。

### 处理方法

- 页面外层使用 `w-full min-w-[320px] max-w-[430px] mx-auto` 或小程序等价容器。
- 内容区统一使用左右安全边距，例如 `px-6`，卡片宽度使用 `w-full`。
- 功能入口使用 4 等分网格，而不是固定 x 坐标或固定总宽。
- 文本较长的标题和描述使用单行省略或最多两行省略。

### 实现要点

```vue
<template>
  <view class="min-h-screen bg-slate-50">
    <view class="mx-auto w-full min-w-[320px] max-w-[430px] px-6">
      <view class="w-full rounded-2xl bg-white p-4 shadow-sm">
        <!-- card content -->
      </view>
    </view>
  </view>
</template>
```

功能入口建议：

```vue
<view class="grid grid-cols-4 gap-y-6">
  <view v-for="item in shortcuts" :key="item.key" class="flex flex-col items-center">
    <view class="flex h-14 w-14 items-center justify-center rounded-2xl">
      <!-- icon -->
    </view>
    <text class="mt-2 text-[13px] text-slate-700">{{ item.label }}</text>
  </view>
</view>
```

### 验收标准

- 320px、375px、390px、430px 宽度下均无横向滚动。
- 首页卡片、功能入口、待办列表均保持左右边距一致。
- 长文本不会撑破卡片。

## 2. 底部导航未明确固定

### 问题

首页设计中底部导航位于纵向内容流最后，页面高度超过常见首屏高度。如果实现时照此布局，底部导航会跟随页面滚动，而不是固定在屏幕底部。

### 影响

- 用户滚动待办事项时底部导航不可见。
- 首页、班级、消息、我的之间切换效率降低。
- 与小程序常见 tabbar 行为不一致。

### 处理方法

- 使用小程序原生 tabbar 时，底部导航交给框架配置，不在页面内重复实现。
- 若需要自定义底部导航，使用固定定位并处理安全区。
- 页面主体增加底部 padding，避免内容被导航遮挡。

### 实现要点

```vue
<template>
  <view class="min-h-screen bg-slate-50 pb-[calc(74px+env(safe-area-inset-bottom))]">
    <slot />

    <view class="fixed inset-x-0 bottom-0 z-50 bg-white px-6 pt-3 shadow-[0_-1px_8px_rgba(15,23,42,0.06)]">
      <view class="mx-auto flex h-[50px] max-w-[430px] items-center justify-between">
        <!-- tab items -->
      </view>
      <view class="h-[env(safe-area-inset-bottom)]" />
    </view>
  </view>
</template>
```

### 验收标准

- 页面滚动时底部导航始终可见。
- 最后一条待办或表单内容不会被底部导航遮挡。
- iPhone 安全区设备底部留白正常。

## 3. 原型交互数据缺失

### 问题

Calicat MCP 返回当前画布“不存在图层交互数据”。PRD 中描述了点击跳转、统计项跳转、按钮反馈、弹窗关闭等交互，但原型没有可追踪连线。

### 影响

- 开发时无法直接从原型确认页面跳转关系。
- 测试用例容易遗漏关键路径。
- 评审时难以验证流程闭环。

### 处理方法

- 实现前维护一份路由映射表，明确每个入口跳转目标。
- 对所有可点击区域统一封装点击反馈和跳转逻辑。
- 对弹窗类操作明确打开、关闭、确认、校验失败等状态。

### 实现要点

建议维护入口配置：

```ts
const shortcuts = [
  { key: 'notice', label: '发布通知', path: '/pages/notice/create' },
  { key: 'growth', label: '成长记录', path: '/pages/growth/index' },
  { key: 'pickup', label: '接送管理', path: '/pages/pickup/index' },
  { key: 'task', label: '亲子任务', path: '/pages/task/index' },
  { key: 'calendar', label: '班级日历', path: '/pages/calendar/index' },
  { key: 'leave', label: '请假审批', path: '/pages/leave/index' },
  { key: 'album', label: '班级相册', path: '/pages/album/index' },
  { key: 'more', label: '更多', path: '/pages/more/index' },
]
```

首页需要补齐的交互：

- 通知铃铛：进入通知中心。
- 今日统计项：进入对应详情页。
- 功能入口：进入对应业务页面。
- 待办“同意”：二次确认或直接提交审批。
- 待办“拒绝”：打开拒绝请假弹窗。
- 待办“提醒”：触发未读家长提醒。
- “查看全部”：进入待办列表。
- 底部导航：首页、班级、消息、我的切换。

### 验收标准

- 每个可点击入口都有明确跳转或状态变化。
- 主要业务路径可被 E2E 用例覆盖。
- 弹窗关闭、取消、确认、校验失败均有可见反馈。

## 4. 图标交付方式不稳定

### 问题

设计中大量图标来自 `remixicon` 字体字符。若小程序端没有正确引入字体或映射，可能出现乱码。

### 影响

- 图标在开发环境、真机或打包后显示不一致。
- 后续替换图标成本较高。
- 字体文件加载失败会影响核心入口识别。

### 处理方法

- 优先使用 SVG 图标组件或已接入的小程序图标组件库。
- 若继续使用 iconfont，需要在项目中统一注册并维护图标名称映射。
- 不在业务组件里直接写字体私有区字符。

### 实现要点

建议用语义化 icon key：

```ts
const shortcuts = [
  { key: 'notice', label: '发布通知', icon: 'notification', colorClass: 'text-blue-500 bg-blue-100' },
  { key: 'growth', label: '成长记录', icon: 'record', colorClass: 'text-green-500 bg-green-100' },
]
```

组件中只接收 `icon` 名称：

```vue
<AppIcon :name="item.icon" class="h-6 w-6" />
```

### 验收标准

- 真机、开发者工具、构建产物中图标显示一致。
- 业务代码中不出现不可读的字体私有区字符。
- 新增或替换图标只需要修改映射或资源文件。

## 5. 操作按钮点击区域偏小

### 问题

待办项中的“同意”“拒绝”“提醒”“去记录”等按钮视觉高度约 30px，低于移动端更舒适的 40px-44px 点击区域。

### 影响

- 老师在移动场景中容易误触或点不中。
- 连续审批任务时操作效率下降。
- 可访问性体验不足。

### 处理方法

- 主要操作按钮最小高度调整为 40px。
- 次要按钮可以保持轻量视觉，但点击热区不能小于视觉热区。
- 待办项中高频操作可允许整块操作区域响应点击。

### 实现要点

```vue
<button class="min-h-10 rounded-lg px-4 text-sm font-medium active:scale-[0.98]">
  同意
</button>
```

拒绝、提醒等按钮保持同样最小高度：

```vue
<button class="min-h-10 rounded-lg bg-red-50 px-4 text-sm font-medium text-red-500 active:scale-[0.98]">
  拒绝
</button>
```

### 验收标准

- 所有按钮最小点击高度不低于 40px。
- 快速连续点击时不会误触相邻按钮。
- 点击反馈明显但不过度干扰操作。

## 6. 首页信息密度偏高

### 问题

首页同时展示顶部身份信息、今日统计、8 个功能入口、3 条待办和底部导航。业务完整，但首屏信息密度偏高。

### 影响

- 新用户首次进入时需要较多阅读成本。
- 高优先级待办容易被普通入口分散注意。
- 后续新增模块时首页会继续膨胀。

### 处理方法

- 保留 8 个功能入口，但按高频程度排序。
- 待办事项只突出最高优先级 2-3 条，更多内容进入“查看全部”。
- 对不同类型待办使用清晰优先级，例如审批类优先于提醒类。
- 如后续功能增加，将低频入口收纳到“更多”。

### 实现要点

首页待办建议按优先级排序：

```ts
const priorityMap = {
  leaveApproval: 100,
  unreadNotice: 80,
  growthRecord: 60,
}
```

展示逻辑：

```ts
const visibleTodos = computed(() =>
  todos.value
    .slice()
    .sort((a, b) => priorityMap[b.type] - priorityMap[a.type])
    .slice(0, 3),
)
```

### 验收标准

- 首页默认只展示最多 3 条待办。
- 高优先级审批类待办排在前面。
- “查看全部”能进入完整待办列表。

## 7. 表单页发布前校验需要补充

### 问题

发布通知页视觉结构完整，但设计中没有明确标题、内容、附件大小、接收范围、定时发送时间等校验和错误态。

### 影响

- 开发时容易只实现静态表单，缺少异常提示。
- 老师发布失败时不知道如何修正。
- 后端接口错误无法清晰反馈给用户。

### 处理方法

- 标题、内容、接收范围设置必填校验。
- 图片/视频限制数量、大小和格式。
- 定时发送开启后必须选择未来时间。
- 发布按钮需要 loading、disabled 和错误反馈状态。

### 实现要点

```ts
function validateNoticeForm(form: NoticeForm) {
  if (!form.title.trim()) return '请输入通知标题'
  if (!form.content.trim()) return '请输入通知内容'
  if (!form.receiverIds.length) return '请选择接收范围'
  if (form.scheduleEnabled && !isFutureTime(form.scheduleTime)) {
    return '请选择未来的发送时间'
  }
  return ''
}
```

### 验收标准

- 必填项为空时不能发布，并展示明确错误提示。
- 发布中按钮不可重复点击。
- 发布成功后有成功反馈并返回列表或详情页。
- 发布失败时保留表单内容。

## 8. 拒绝请假弹窗需要防误操作

### 问题

拒绝请假弹窗已经包含必填原因和快捷原因，但“确认拒绝”是高风险操作，需要更明确的校验和提交状态。

### 影响

- 老师可能未填写原因直接提交。
- 网络延迟时可能重复提交。
- 家长收到的拒绝原因可能不完整或不礼貌。

### 处理方法

- 拒绝原因必填，未填写时禁用确认按钮或展示错误。
- 点击快捷原因后自动填入或追加到输入框。
- 提交时按钮进入 loading 状态。
- 提交成功后关闭弹窗并刷新请假状态。

### 实现要点

```ts
const canReject = computed(() => rejectReason.value.trim().length > 0 && !submitting.value)

async function submitReject() {
  if (!canReject.value) return
  submitting.value = true
  try {
    await rejectLeaveApplication(applicationId.value, rejectReason.value.trim())
    closeRejectDialog()
    refreshLeaveList()
  } finally {
    submitting.value = false
  }
}
```

### 验收标准

- 未填写拒绝原因时不能提交。
- 快捷原因选择后输入框内容同步变化。
- 提交期间不能重复点击。
- 拒绝成功后列表状态更新。

## 实现优先级

1. 高优先级：响应式适配、底部导航固定、交互路由映射、按钮点击区域。
2. 中优先级：发布通知表单校验、拒绝请假防误操作、图标组件化。
3. 低优先级：首页信息密度优化、功能入口排序策略。

## 推荐实现检查清单

- [ ] 页面容器适配 320px-430px，无横向滚动。
- [ ] 底部导航固定，并处理安全区。
- [ ] 首页所有入口都有明确跳转或状态变化。
- [ ] 所有按钮点击区域不低于 40px。
- [ ] 图标使用组件或统一映射，不直接写私有区字符。
- [ ] 发布通知页有完整表单校验、loading、错误态。
- [ ] 拒绝请假弹窗有必填校验、快捷原因、提交防重复。
- [ ] 首页待办按优先级排序，默认最多展示 3 条。
