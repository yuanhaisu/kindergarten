/**
 * 47 · 22 任务详情 V2 口径。
 *
 * 22 要和 21 任务列表 / 24 积分页同一口径：保留 +10 与「本周 3 / 3 天」，
 * 但不再露出「积分提案」「提案：需 06-task 恢复 points/streak」这类评审标注。
 * 未打开状态与打卡交互由 55 的 ?state=unopened 用例覆盖。
 */
export default {
  checks: [
    {
      slug: "22-tasks-detail",
      must: ["+10 积分", "本周 3 / 3 天"],
      mustNot: ["积分提案", "提案：需 06-task 恢复 points/streak"],
    },
    { slug: "22-tasks-detail", within: ".week-streak", must: ["本周 3 / 3 天"], mustNot: ["+10"] },
  ],
};
