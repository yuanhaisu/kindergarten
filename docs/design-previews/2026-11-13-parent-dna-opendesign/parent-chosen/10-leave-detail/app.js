(() => {
  "use strict";

  const main = document.querySelector(".mp-main");
  const trigger = document.getElementById("withdraw-trigger");
  const layer = document.getElementById("withdraw-sheet");
  const sheet = layer ? layer.querySelector(".sheet") : null;
  const safeButton = layer ? layer.querySelector(".sheet-safe") : null;
  const confirmButton = document.getElementById("withdraw-confirm");
  const resubmitSection = document.getElementById("resubmit-section");
  const resubmitButton = document.getElementById("resubmit-leave");
  const toast = document.getElementById("toast");

  if (!main || !trigger || !layer || !sheet || !safeButton || !confirmButton || !toast) {
    return;
  }

  let lastFocused = null;
  let focusTimer = 0;
  let toastTimer = 0;

  const isOpen = () => layer.classList.contains("is-open");

  function focusWithoutScroll(element) {
    if (element instanceof HTMLElement) {
      element.focus({ preventScroll: true });
    }
  }

  function getFocusableElements() {
    return Array.from(
      sheet.querySelectorAll(
        "button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex='-1'])"
      )
    ).filter((element) => element.getClientRects().length > 0);
  }

  function openSheet() {
    if (isOpen()) {
      return;
    }

    lastFocused = document.activeElement instanceof HTMLElement ? document.activeElement : trigger;
    window.clearTimeout(focusTimer);

    layer.classList.add("is-open");
    layer.setAttribute("aria-hidden", "false");
    trigger.setAttribute("aria-expanded", "true");
    main.classList.add("is-sheet-open");

    focusTimer = window.setTimeout(() => {
      if (isOpen()) {
        focusWithoutScroll(safeButton);
      }
    }, 30);
  }

  function closeSheet({ restoreFocus = true } = {}) {
    if (!isOpen()) {
      return;
    }

    window.clearTimeout(focusTimer);
    layer.classList.remove("is-open");
    layer.setAttribute("aria-hidden", "true");
    trigger.setAttribute("aria-expanded", "false");
    main.classList.remove("is-sheet-open");

    if (restoreFocus) {
      focusWithoutScroll(lastFocused || trigger);
    }
  }

  function showToast(message) {
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add("is-visible");

    toastTimer = window.setTimeout(() => {
      toast.classList.remove("is-visible");
    }, 2000);
  }

  trigger.addEventListener("click", openSheet);

  layer.querySelectorAll("[data-sheet-close]").forEach((element) => {
    element.addEventListener("click", () => closeSheet());
  });

  confirmButton.addEventListener("click", () => {
    closeSheet();
    showToast("撤回申请已提交");
  });

  document.addEventListener("keydown", (event) => {
    if (!isOpen()) {
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (event.key !== "Tab") {
      return;
    }

    const focusable = getFocusableElements();
    if (!focusable.length) {
      event.preventDefault();
      return;
    }

    const first = focusable[0];
    const last = focusable[focusable.length - 1];

    if (!sheet.contains(document.activeElement)) {
      event.preventDefault();
      focusWithoutScroll(first);
      return;
    }

    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      focusWithoutScroll(last);
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      focusWithoutScroll(first);
    }
  });

  const requestedState = new URLSearchParams(window.location.search).get("state");
  if (requestedState === "cancelled") {
    const title = document.querySelector(".status-card__title");
    const desc = document.querySelector(".status-card__desc");
    if (title) title.textContent = "已取消";
    if (desc) desc.textContent = "你已撤回这次请假，如需请假请重新提交。";
    const approval = document.querySelector(".approval-card");
    if (approval) approval.style.display = "none";
    const withdraw = document.querySelector(".withdraw-section");
    if (withdraw) withdraw.style.display = "none";
    if (resubmitSection) resubmitSection.hidden = false;
  }

  if (resubmitButton) {
    resubmitButton.addEventListener("click", () => {
      const reasonCopy = document.querySelector(".reason-copy");
      const reasonText = reasonCopy ? reasonCopy.innerText.replace(/\s+/g, " ").trim() : "";
      const query = new URLSearchParams({
        from: "10",
        type: "sick",
        start: "2026-11-13",
        end: "2026-11-13",
        reason: reasonText,
      });
      window.location.href = "../09-leave-create/index.html?" + query.toString();
    });
  }

  window.addEventListener(
    "pagehide",
    () => {
      window.clearTimeout(focusTimer);
      window.clearTimeout(toastTimer);
    },
    { once: true }
  );
})();
