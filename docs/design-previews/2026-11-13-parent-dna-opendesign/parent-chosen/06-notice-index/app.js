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

  // 「通知」是作业/带物/健康/其他的前端聚合视图：契约里没有 category=notice，
  // 这个筛选只在本地合并子分类，不会被当成请求参数。
  const NOTICE_AGGREGATE = ["homework", "bring", "health", "other"];

  const matchesFilter = (card, filter) => {
    const isUnread = card.dataset.unread === "true";
    const category = card.dataset.category;

    if (filter === "all") return true;
    if (filter === "unread") return isUnread;
    if (filter === "notice") return NOTICE_AGGREGATE.includes(category);
    return category === filter;
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
