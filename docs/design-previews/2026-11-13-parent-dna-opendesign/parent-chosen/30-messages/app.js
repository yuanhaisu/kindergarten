(() => {
  const app = document.querySelector("#messages-app");
  if (!app) return;

  const params = new URLSearchParams(window.location.search);
  const state = params.get("state");
  const isListView = state !== "empty";

  app.classList.toggle("is-empty", !isListView);
  app.dataset.view = isListView ? "list" : "empty";

  const filters = Array.from(document.querySelectorAll(".seg__item"));
  const rows = Array.from(document.querySelectorAll(".msg-row"));
  const totalUnreadEl = document.querySelector("[data-unread-count]");
  const groupUnreadEl = document.querySelector("[data-group-unread]");
  const markAllButton = document.querySelector("[data-mark-all]");
  const main = document.querySelector(".mp-main");
  const backButton = document.querySelector(".page-bar__back");
  const homeButton = document.querySelector(".btn-primary");
  const sheetLayer = document.getElementById("message-sheet");
  const sheetClose = sheetLayer?.querySelector(".sheet-close");
  const sheetAvatar = sheetLayer?.querySelector("[data-sheet-avatar]");
  const sheetSender = sheetLayer?.querySelector("[data-sheet-sender]");
  const sheetTime = sheetLayer?.querySelector("[data-sheet-time]");
  const sheetBody = sheetLayer?.querySelector("[data-sheet-body]");
  const bodiesEl = document.getElementById("message-bodies");

  const bodies = (() => {
    if (!bodiesEl) return {};
    try {
      return JSON.parse(bodiesEl.textContent);
    } catch (error) {
      console.error("消息正文解析失败", error);
      return {};
    }
  })();

  const filterNames = new Set(filters.map((button) => button.dataset.filter));
  const isUnread = (row) => row.dataset.unread === "true";

  let sheetOpen = false;
  let lastFocused = null;

  const updateSeparators = () => {
    let hasVisibleRow = false;

    rows.forEach((row) => {
      const isVisible = !row.hidden;
      row.classList.toggle("has-separator", isVisible && hasVisibleRow);
      if (isVisible) hasVisibleRow = true;
    });
  };

  // 页头是全部未读，筛选区的“本组”只算当前可见行
  const updateCounts = () => {
    const total = rows.filter(isUnread).length;
    const group = rows.filter((row) => !row.hidden && isUnread(row)).length;

    if (totalUnreadEl) totalUnreadEl.textContent = String(total);
    if (groupUnreadEl) groupUnreadEl.textContent = String(group);
    if (markAllButton) markAllButton.disabled = total === 0;
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
    updateCounts();
  };

  const openSheet = (row) => {
    if (!sheetLayer) return;

    const paragraphs = bodies[row.dataset.messageId] || [];
    const sender = row.querySelector(".msg-sender");
    const avatar = row.querySelector(".avatar");

    if (sheetAvatar && avatar) {
      sheetAvatar.className = `sheet-avatar ${avatar.className}`;
      sheetAvatar.innerHTML = avatar.innerHTML;
    }
    if (sheetSender) sheetSender.textContent = sender ? sender.textContent.trim() : "";
    if (sheetTime) sheetTime.textContent = row.dataset.fullTime || "";
    if (sheetBody) {
      sheetBody.replaceChildren();
      paragraphs.forEach((text) => {
        const paragraph = document.createElement("p");
        paragraph.textContent = text;
        sheetBody.appendChild(paragraph);
      });
    }

    if (isUnread(row)) {
      row.dataset.unread = "false";
      updateCounts();
    }

    lastFocused = document.activeElement instanceof HTMLElement ? document.activeElement : row;
    sheetOpen = true;
    sheetLayer.removeAttribute("inert");
    sheetLayer.setAttribute("aria-hidden", "false");
    sheetLayer.classList.add("is-open");
    app.classList.add("is-sheet-open");
    main?.setAttribute("inert", "");
    window.requestAnimationFrame(() => {
      sheetClose?.focus({ preventScroll: true });
    });
  };

  const closeSheet = ({ restoreFocus = true } = {}) => {
    if (!sheetOpen || !sheetLayer) return;

    sheetOpen = false;
    sheetLayer.classList.remove("is-open");
    sheetLayer.setAttribute("aria-hidden", "true");
    sheetLayer.setAttribute("inert", "");
    app.classList.remove("is-sheet-open");
    main?.removeAttribute("inert");

    if (!restoreFocus) return;
    const target = lastFocused instanceof HTMLElement ? lastFocused : backButton;
    target?.focus({ preventScroll: true });
  };

  filters.forEach((button) => {
    button.addEventListener("click", () => setFilter(button.dataset.filter));
  });

  rows.forEach((row) => {
    row.addEventListener("click", () => openSheet(row));
  });

  markAllButton?.addEventListener("click", () => {
    let changed = false;

    rows.forEach((row) => {
      if (!isUnread(row)) return;
      row.dataset.unread = "false";
      changed = true;
    });

    if (changed) updateCounts();
  });

  sheetLayer?.addEventListener("click", (event) => {
    if (!(event.target instanceof Element)) return;
    if (event.target.closest("[data-close-sheet]")) closeSheet();
  });

  document.addEventListener("keydown", (event) => {
    if (!sheetOpen) return;

    if (event.key === "Escape") {
      event.preventDefault();
      closeSheet();
      return;
    }

    // 详情只有关闭一个可聚焦点，把 Tab 留在 sheet 内
    if (event.key === "Tab") {
      event.preventDefault();
      sheetClose?.focus({ preventScroll: true });
    }
  });

  const goBack = () => {
    if (window.history.length > 1) window.history.back();
  };

  backButton?.addEventListener("click", goBack);
  homeButton?.addEventListener("click", goBack);

  setFilter("all");

  if (state === "detail" && rows[0]) openSheet(rows[0]);
})();
