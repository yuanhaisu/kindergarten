/**
 * 47 · 22 任务详情 V2 提案态。
 *
 * 22 要和 21 任务列表 / 24 积分页同一口径：保留原有的 +10，
 * 补「本周 3 / 3 天」连续打卡与「积分提案」标注，避免被当成现状。
 * 只断言静态 V2 文案；提交后的动态状态不在本票范围内，不设 `?state=done`。
 */
export default {
  checks: [
    {
      slug: "22-tasks-detail",
      must: ["+10 积分", "本周 3 / 3 天", "积分提案", "提案：需 06-task 恢复 points/streak"],
    },
    { slug: "22-tasks-detail", within: ".week-streak", must: ["本周 3 / 3 天"], mustNot: ["+10"] },
  ],
};
