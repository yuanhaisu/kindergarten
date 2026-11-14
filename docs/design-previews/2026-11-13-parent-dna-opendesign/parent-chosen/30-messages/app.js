(() => {
  const app = document.querySelector("#messages-app");
  if (!app) return;

  const isListView = new URLSearchParams(window.location.search).get("state") !== "empty";
  app.classList.toggle("is-empty", !isListView);
  app.dataset.view = isListView ? "list" : "empty";

  const filters = Array.from(document.querySelectorAll(".seg__item"));
  const rows = Array.from(document.querySelectorAll(".msg-row"));
  const unreadCount = document.querySelector("[data-unread-count]");
  const backButton = document.querySelector(".page-bar__back");
  const homeButton = document.querySelector(".btn-primary");

  const filterNames = new Set(filters.map((button) => button.dataset.filter));

  const updateSeparators = () => {
    let hasVisibleRow = false;

    rows.forEach((row) => {
      const isVisible = !row.hidden;
      row.classList.toggle("has-separator", isVisible && hasVisibleRow);
      if (isVisible) hasVisibleRow = true;
    });
  };

  const updateUnreadCount = () => {
    const count = rows.filter(
      (row) => !row.hidden && row.dataset.unread === "true",
    ).length;

    if (unreadCount) unreadCount.textContent = String(count);
  };

  const setFilter = (filter) => {
    if (!filterNames.has(filter)) return;

    filters.forEach((button) => {
      const isActive = button.dataset.filter === filter;
      button.classList.toggle("is-active", isActive);
      button.setAttribute("aria-pressed", String(isActive));
    });

    rows.forEach((row) => {
      row.hidden = filter !== "all" && row.dataset.category !== filter;
    });

    updateSeparators();
    updateUnreadCount();
  };

  filters.forEach((button) => {
    button.addEventListener("click", () => setFilter(button.dataset.filter));
  });

  rows.forEach((row) => {
    row.addEventListener("click", () => {
      if (row.dataset.unread !== "true") return;
      row.dataset.unread = "false";
      updateUnreadCount();
    });
  });

  const goBack = () => {
    if (window.history.length > 1) window.history.back();
  };

  backButton?.addEventListener("click", goBack);
  homeButton?.addEventListener("click", goBack);

  setFilter("all");
})();
