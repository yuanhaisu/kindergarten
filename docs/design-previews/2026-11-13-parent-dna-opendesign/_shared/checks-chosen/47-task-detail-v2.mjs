/**
 * 47 · 22 任务详情 V2 提案态。
 *
 * 22 要和 21 任务列表 / 24 积分页同一口径：保留原有的 +10，
 * 补「本周 2 / 3 天」连续打卡与「积分提案」标注，避免被当成现状。
 * 提交打卡后（?state=done）与 21 的打卡交互一致：第三格点亮，文案变 3 / 3。
 */
export default {
  checks: [
    {
      slug: "22-tasks-detail",
      must: ["+10 积分", "本周 2 / 3 天", "积分提案", "提案：需 06-task 恢复 points/streak"],
    },
    { slug: "22-tasks-detail", within: ".week-streak", must: ["本周 2 / 3 天"], mustNot: ["+10"] },
    { slug: "22-tasks-detail", state: "done", must: ["本周 3 / 3 天", "今天已打卡"] },
  ],
};
