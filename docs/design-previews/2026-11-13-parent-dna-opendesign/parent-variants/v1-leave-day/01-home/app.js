(() => {
  "use strict";

  const app = document.querySelector(".mp-app");
  const main = document.querySelector(".mp-main");
  const bottomNav = document.querySelector(".mp-nav");
  const toast = document.getElementById("toast");
  const sheetLayer = document.getElementById("child-sheet");
  const switchButton = document.getElementById("switch-child");

  if (!app || !main || !toast || !sheetLayer || !switchButton) {
    return;
  }

  const params = new URLSearchParams(window.location.search);
  if (params.get("state") === "leave") {
    app.dataset.state = "leave";
  }

  const closeButton = sheetLayer.querySelector(".sheet-close");
  const childOption = sheetLayer.querySelector(".child-option");
  const leaveCta = document.querySelector(".leave-cta");
  const taskCta = document.getElementById("task-cta");
  let lastFocused = null;
  let sheetOpen = false;
  let toastTimer = 0;

  function showToast(message) {
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add("is-visible");
    toastTimer = window.setTimeout(() => {
      toast.classList.remove("is-visible");
    }, 2000);
  }

  function openSheet() {
    lastFocused = document.activeElement;
    sheetOpen = true;
    sheetLayer.removeAttribute("inert");
    sheetLayer.setAttribute("aria-hidden", "false");
    sheetLayer.classList.add("is-open");
    app.classList.add("is-sheet-open");
    switchButton.setAttribute("aria-expanded", "true");
    main.setAttribute("inert", "");
    if (bottomNav) {
      bottomNav.setAttribute("inert", "");
    }
    window.requestAnimationFrame(() => {
      closeButton.focus({ preventScroll: true });
    });
  }

  function closeSheet(options = {}) {
    if (!sheetOpen) {
      return;
    }

    sheetOpen = false;
    sheetLayer.classList.remove("is-open");
    sheetLayer.setAttribute("aria-hidden", "true");
    sheetLayer.setAttribute("inert", "");
    app.classList.remove("is-sheet-open");
    switchButton.setAttribute("aria-expanded", "false");
    main.removeAttribute("inert");
    if (bottomNav) {
      bottomNav.removeAttribute("inert");
    }

    if (options.restoreFocus !== false) {
      const focusTarget =
        lastFocused instanceof HTMLElement && lastFocused !== document.body
          ? lastFocused
          : switchButton;
      focusTarget.focus({ preventScroll: true });
    }
  }

  switchButton.addEventListener("click", openSheet);

  sheetLayer.addEventListener("click", (event) => {
    if (!(event.target instanceof Element)) {
      return;
    }
    if (event.target.closest("[data-close-sheet]")) {
      closeSheet();
    }
  });

  if (childOption) {
    childOption.addEventListener("click", () => {
      const name = childOption.dataset.childName || "乐乐";
      closeSheet();
      showToast(`当前孩子：${name}`);
    });
  }

  if (leaveCta) {
    leaveCta.addEventListener("click", (event) => {
      event.preventDefault();
      showToast("请假详情 · 待审批");
    });
  }

  if (taskCta) {
    taskCta.addEventListener("click", (event) => {
      event.preventDefault();
      showToast("今晚一件事 · 今天还没提交");
    });
  }

  document.addEventListener("keydown", (event) => {
    if (!sheetOpen) {
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (event.key !== "Tab" || !closeButton || !childOption) {
      return;
    }

    const first = closeButton;
    const last = childOption;
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });

  document.addEventListener("click", (event) => {
    if (!(event.target instanceof Element)) {
      return;
    }

    const link = event.target.closest('a[href="#"]');
    if (!link) {
      return;
    }

    event.preventDefault();
    if (link.classList.contains("tab") && link.dataset.tab === "home") {
      main.scrollTop = 0;
    }
  });
})();
