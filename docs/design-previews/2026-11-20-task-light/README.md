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
- 未绑定：`.../22-tasks-detail/index.html?state=unbound`

### 老师端

- 发布任务：`teacher/06-task-publish/index.html`
- 发布后分享：`teacher/06-task-publish/index.html?state=published`
- 任务统计：`teacher/07-task-stats/index.html`
- 粘贴聊天记录后：`teacher/07-task-stats/index.html?state=marked`

## 关键决策

- 家长点击「完成了」才算完成；打开资源只算「已查看」。
- 观看型任务按资源进度自动记录；外部链接只能记录打开，不承诺自动完成。
- 群聊记录只解析文字；图片/语音/视频不自动识别。
- 粘贴后自动识别，不需要逐个勾选孩子；「撤销本次」保留纠错能力。
- 未绑定先绑定；手机号不匹配时给「联系老师 / 了解入园」两个选项，不自动跳招生页。
- 任务必须有截止时间或手动结束，统计才有定稿时刻。
