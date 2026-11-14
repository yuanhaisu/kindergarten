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

  const applyFilter = (filter) => {
    noticeCards.forEach((card) => {
      const isUnread = card.dataset.unread === "true";
      const matchesCategory = card.dataset.category === filter;

      card.hidden = !(
        filter === "all" ||
        (filter === "unread" && isUnread) ||
        matchesCategory
      );
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
