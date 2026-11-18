(() => {
  const toast = document.querySelector("[data-toast]");
  const sheet = document.getElementById("health-note-sheet");
  const scrim = document.getElementById("health-note-scrim");
  const editButton = document.getElementById("edit-health-note");
  const cancelButton = document.getElementById("health-note-cancel");
  const saveButton = document.getElementById("health-note-save");
  const input = document.getElementById("health-note-input");
  const valueEl = document.getElementById("health-note-value");
  let hideTimer = 0;
  let clearTimer = 0;

  const showToast = (message) => {
    if (!toast) return;
    window.clearTimeout(hideTimer);
    window.clearTimeout(clearTimer);
    toast.textContent = message;
    toast.classList.add("is-visible");
    hideTimer = window.setTimeout(() => {
      toast.classList.remove("is-visible");
      clearTimer = window.setTimeout(() => {
        toast.textContent = "";
      }, 200);
    }, 2000);
  };

  const closeSheet = () => {
    if (sheet) sheet.hidden = true;
    if (editButton) editButton.focus({ preventScroll: true });
  };

  document.querySelectorAll("[data-action]").forEach((button) => {
    button.addEventListener("click", () => {
      if (button.getAttribute("data-action") === "back") {
        showToast("返回「我的」");
      }
    });
  });

  if (sheet && editButton && input) {
    editButton.addEventListener("click", () => {
      if (valueEl) input.value = valueEl.textContent.trim();
      sheet.hidden = false;
      input.focus({ preventScroll: true });
    });
  }

  if (scrim) scrim.addEventListener("click", closeSheet);
  if (cancelButton) cancelButton.addEventListener("click", closeSheet);

  if (saveButton && input && sheet) {
    saveButton.addEventListener("click", () => {
      if (valueEl) valueEl.textContent = input.value.trim() || "无";
      closeSheet();
      showToast("健康备注已保存");
    });
  }

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape" && sheet && !sheet.hidden) {
      event.preventDefault();
      closeSheet();
    }
  });
})();
