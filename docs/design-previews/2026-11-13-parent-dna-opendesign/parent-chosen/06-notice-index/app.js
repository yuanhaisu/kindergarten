(() => {
  "use strict";

  const app = document.querySelector(".mp-app");
  const listView = document.querySelector("[data-list-view]");
  const emptyView = document.querySelector("[data-empty-view]");
  const filterButtons = Array.from(document.querySelectorAll(".filter-chip[data-filter]"));
  const noticeCards = Array.from(document.querySelectorAll(".notice-card[data-notice]"));

  if (!app || !listView || !emptyView) return;

  const requestedState = new URLSearchParams(window.location.search).get("state");
  const isEmpty = requestedState === "empty";

  app.dataset.state = isEmpty ? "empty" : "list";
  listView.hidden = isEmpty;
  emptyView.hidden = !isEmpty;

  // 票 #46 / #41：筛选口径必须覆盖 API 通知分类的全部六类。
  // task / activity 各自成一个 tab；homework / bring / health / other 四类合并进「通知」。
  // 契约里没有 category=notice，这个筛选只在本地合并子分类，不会被当成请求参数。
  // 新增分类若忘记登记，落到「通知」并告警，而不是从所有筛选里静默消失。
  const FILTER_BY_CATEGORY = {
    task: "task",
    activity: "activity",
    homework: "notice",
    bring: "notice",
    health: "notice",
    other: "notice",
  };

  const filterFor = (category) => {
    const filter = FILTER_BY_CATEGORY[category];
    if (filter) return filter;
    console.warn(`[06] 未知通知分类「${category}」，已归入「通知」筛选`);
    return "notice";
  };

  const matchesFilter = (card, filter) => {
    const isUnread = card.dataset.unread === "true";

    if (filter === "all") return true;
    if (filter === "unread") return isUnread;
    return filterFor(card.dataset.category) === filter;
  };

  const applyFilter = (filter) => {
    noticeCards.forEach((card) => {
      card.hidden = !matchesFilter(card, filter);
    });
  };

  filterButtons.forEach((button) => {
    button.addEventListener("click", () => {
      if (button.getAttribute("aria-pressed") === "true") return;

      filterButtons.forEach((candidate) => {
        const isActive = candidate === button;
        candidate.classList.toggle("is-active", isActive);
        candidate.setAttribute("aria-pressed", String(isActive));
      });

      applyFilter(button.dataset.filter);
    });
  });
})();
