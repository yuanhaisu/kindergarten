(() => {
  "use strict";

  const app = document.querySelector(".mp-app");
  const main = document.querySelector(".mp-main");
  const bottomNav = document.querySelector(".mp-nav");
  const sheetLayer = document.getElementById("child-sheet-layer");
  const sheet = document.getElementById("child-sheet");
  const switchButton = document.getElementById("child-switch");
  const childAvatar = document.getElementById("child-avatar");
  const childName = document.getElementById("child-name");
  const childMeta = document.getElementById("child-meta");
  const statPickup = document.getElementById("stat-pickup");
  const statGrowth = document.getElementById("stat-growth");
  const statPoints = document.getElementById("stat-points");

  /* Placeholder links stay put; without this they would jump the scroll
   * container or change the URL hash. */
  if (app) {
    app.addEventListener("click", (event) => {
      const link = event.target.closest('a[href="#"]');
      if (!link) return;
      event.preventDefault();
    });
  }

  if (!app || !main || !sheetLayer || !sheet || !switchButton || !childAvatar || !childName || !childMeta || !statPickup || !statGrowth || !statPoints) {
    return;
  }

  const closeButton = sheet.querySelector(".sheet-close");
  const options = Array.from(sheet.querySelectorAll(".child-option"));
  const lastOption = options[options.length - 1];
  let lastFocused = null;
  let sheetOpen = false;

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

  function closeSheet(settings = {}) {
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

    if (settings.restoreFocus !== false) {
      const focusTarget = lastFocused instanceof HTMLElement ? lastFocused : switchButton;
      focusTarget.focus({ preventScroll: true });
    }
  }

  function selectChild(option) {
    options.forEach((item) => {
      item.setAttribute("aria-pressed", String(item === option));
    });
    childAvatar.textContent = option.dataset.childInitial || "";
    childName.textContent = option.dataset.childName || "";
    childMeta.textContent = option.dataset.childRowMeta || "";
    const [pickup, growth, points] = (option.dataset.childStats || "").split(",");
    if (pickup && growth && points) {
      statPickup.textContent = pickup;
      statGrowth.textContent = growth;
      statPoints.textContent = points;
    }
    closeSheet();
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

  options.forEach((option) => {
    option.addEventListener("click", () => selectChild(option));
  });

  document.addEventListener("keydown", (event) => {
    if (!sheetOpen) {
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (event.key !== "Tab" || !closeButton || !lastOption) {
      return;
    }

    if (event.shiftKey && document.activeElement === closeButton) {
      event.preventDefault();
      lastOption.focus();
    } else if (!event.shiftKey && document.activeElement === lastOption) {
      event.preventDefault();
      closeButton.focus();
    }
  });

  if (new URLSearchParams(window.location.search).get("state") === "switch") {
    openSheet();
  }
})();
