(() => {
  const app = document.querySelector(".mp-app");
  if (!app) return;

  const main = app.querySelector(".mp-main");
  const tabbar = app.querySelector(".mp-nav");
  const layer = app.querySelector("#sheet-layer");
  const sheet = app.querySelector("#logout-sheet");
  const confirmLogout = app.querySelector("#logout-confirm");
  const toastEl = app.querySelector("#toast");

  let toastTimer = 0;
  let closeTimer = 0;
  let currentSheet = null;
  let lastFocus = null;

  const toast = (message) => {
    if (!toastEl) return;
    toastEl.textContent = message;
    toastEl.classList.add("is-visible");
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
      toastEl.classList.remove("is-visible");
    }, 2000);
  };

  const hideToast = () => {
    if (!toastEl) return;
    window.clearTimeout(toastTimer);
    toastEl.classList.remove("is-visible");
  };

  const setPageInert = (on) => {
    if (!main || !tabbar) return;
    if ("inert" in main) {
      main.inert = on;
      tabbar.inert = on;
    }
  };

  const openSheet = (trigger) => {
    if (!sheet || !layer || currentSheet) return;
    hideToast();
    window.clearTimeout(closeTimer);
    lastFocus = trigger || document.activeElement;
    sheet.hidden = false;
    layer.hidden = false;
    app.classList.add("is-sheet-open");
    setPageInert(true);
    void layer.offsetWidth;
    layer.classList.add("is-open");
    currentSheet = sheet;
    sheet.focus({ preventScroll: true });
  };

  const closeSheet = (restore = true, done = null) => {
    if (!currentSheet || !layer) return;
    const closing = currentSheet;
    currentSheet = null;
    layer.classList.remove("is-open");
    app.classList.remove("is-sheet-open");
    setPageInert(false);
    window.clearTimeout(closeTimer);
    closeTimer = window.setTimeout(() => {
      closing.hidden = true;
      layer.hidden = true;
      if (restore && lastFocus) lastFocus.focus({ preventScroll: true });
      if (done) done();
    }, 210);
  };

  app.addEventListener("click", (event) => {
    const target = event.target instanceof Element
      ? event.target.closest("[data-tab], [data-toast], [data-action]")
      : null;
    if (!target || !app.contains(target)) return;

    if (target.dataset.tab) {
      event.preventDefault();
      if (target.getAttribute("aria-current") === "page") {
        if (main && main.scrollTop > 0) {
          const reduce = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
          main.scrollTo({ top: 0, behavior: reduce ? "auto" : "smooth" });
        } else {
          toast("已在「我的」");
        }
      } else {
        toast(`前往${target.dataset.tab}`);
      }
      return;
    }

    if (target.dataset.toast) {
      toast(target.dataset.toast);
      return;
    }

    if (target.dataset.action === "logout") {
      openSheet(target);
    } else if (target.dataset.action === "close-sheet") {
      closeSheet();
    }
  });

  if (confirmLogout) {
    confirmLogout.addEventListener("click", () => {
      closeSheet(true, () => toast("已退出登录"));
    });
  }

  document.addEventListener("keydown", (event) => {
    if (!currentSheet) return;

    if (event.key === "Escape") {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (event.key !== "Tab") return;

    const focusables = Array.from(currentSheet.querySelectorAll(
      'button:not([disabled]), [href], input:not([disabled]), [tabindex]:not([tabindex="-1"])'
    )).filter((el) => el.offsetParent !== null);

    if (!focusables.length) {
      event.preventDefault();
      currentSheet.focus({ preventScroll: true });
      return;
    }

    const first = focusables[0];
    const last = focusables[focusables.length - 1];

    if (event.shiftKey && (document.activeElement === first || document.activeElement === currentSheet)) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });

  window.addEventListener("pagehide", () => {
    window.clearTimeout(toastTimer);
    window.clearTimeout(closeTimer);
  });
})();
