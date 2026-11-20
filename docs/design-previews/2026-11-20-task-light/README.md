# 亲子任务 · 轻量版（v1）

日期：2026-11-20

## 产品范围

v1 只验证一条主路径：

```
老师发布 → 分享卡片/海报 → 家长打开 → 绑定 → 一键完成 / 查看资源 → 老师统计
```

### 做

- 老师发布完成型 / 观看型任务
- 分享卡片、保存海报
- 家长一键完成，不上传任何媒体
- 观看型按资源进度记录
- 老师粘贴群聊文字，自动识别并标记
- 截止/手动结束

### 不做

- 家长上传照片、视频、音频
- 积分、连续打卡、星星罐
- 班级公开墙（23 暂不接）
- 老师回应仪式、补充回应、家长反馈
- 复杂任务状态、长期历史

## 页面

### 家长端

- 任务列表：`../2026-11-13-parent-dna-opendesign/parent-chosen/21-tasks/index.html`
- 任务详情（已记录）：`../2026-11-13-parent-dna-opendesign/parent-chosen/22-tasks-detail/index.html`
- 待完成：`.../22-tasks-detail/index.html?state=pending`
- 观看型：`.../22-tasks-detail/index.html?state=view`
- 资源查看页：`.../22-tasks-detail/index.html?state=viewer`
- 未绑定：`.../22-tasks-detail/index.html?state=unbound`
- 绑定后选孩子：`.../22-tasks-detail/index.html?state=child`
- 已结束：`.../22-tasks-detail/index.html?state=ended`

### 老师端

- 发布任务：`teacher/06-task-publish/index.html`
- 发布后分享：`teacher/06-task-publish/index.html?state=published`
- 含小程序码海报：`teacher/06-task-publish/index.html?state=poster`
- 任务统计：`teacher/07-task-stats/index.html`
- 粘贴聊天记录后：`teacher/07-task-stats/index.html?state=marked`

## 关键决策

- 家长点击「完成了」才算完成；打开资源页不直接完成，看到 90% 才记录。
- 观看型任务按资源进度自动记录；外部链接只能记录打开，不承诺自动完成。
- 群聊记录只解析文字；只有「完成 / 做了 / 打卡 / 看完」等明确语义才自动标记。
- 「收到 / 好的」进入「待确认」，不静默算完成；「请假 / 没做」不会被误标。
- 粘贴后自动识别，不需要逐个勾选孩子；未识别和未提及都会列出，可点名单手动修正。
- 未绑定先绑定；绑定后必须选择本次任务记录给哪个孩子。
- 任务必须有截止时间或手动结束，统计才有定稿时刻；到期后锁定，不再接受完成记录。
- 家长完成 5 分钟内可撤销；老师可点名单手动标记或取消。
- 海报必须带任务小程序码，否则静态海报无法回到具体任务。
