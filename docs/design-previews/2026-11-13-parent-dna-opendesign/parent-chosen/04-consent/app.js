(function () {
  "use strict";

  var rows = Array.prototype.slice.call(document.querySelectorAll(".consent-input"));
  var submit = document.getElementById("consent-submit");
  var decline = document.getElementById("consent-decline");
  var gate = document.getElementById("consent-gate");
  var dialogLayer = document.getElementById("decline-dialog");
  var dialog = document.getElementById("decline-sheet");
  var backButton = document.getElementById("decline-back");
  var exitButton = document.getElementById("decline-exit");

  if (!rows.length || !submit || !decline || !gate || !dialogLayer || !dialog || !backButton || !exitButton) {
    return;
  }

  var settled = false;
  var lastFocused = null;
  var focusables = [];
  var closeTimer = null;

  function syncGate() {
    if (settled) return;
    var missing = rows.filter(function (input) {
      return !input.checked;
    }).length;
    submit.disabled = missing > 0;
    gate.textContent = missing === 0 ? "三份约定均已勾选" : "还需勾选 " + missing + " 份约定";
  }

  function showSuccess() {
    settled = true;
    rows.forEach(function (input) {
      input.checked = true;
    });
    submit.disabled = true;
    submit.textContent = "已同意";
    gate.textContent = "已记录你的同意，正在进入绑定孩子…";
  }

  // ---- 暂不同的阻断确认区：底部抽屉 + 焦点陷阱 + ESC/关闭返回 ----

  function collectFocusables() {
    focusables = Array.prototype.slice
      .call(dialog.querySelectorAll("button:not([disabled]), [href], input:not([disabled])"))
      .filter(function (element) {
        return element.offsetParent !== null;
      });
  }

  function onDialogKeydown(event) {
    if (event.key === "Escape") {
      event.preventDefault();
      closeDialog();
      return;
    }
    if (event.key !== "Tab" || !focusables.length) return;
    var first = focusables[0];
    var last = focusables[focusables.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  function openDialog() {
    lastFocused = document.activeElement;
    // 程序化打开时焦点可能仍在 body，关闭后应回到触发的按钮。
    if (!lastFocused || lastFocused === document.body) lastFocused = decline;
    window.clearTimeout(closeTimer);
    dialogLayer.hidden = false;
    document.body.classList.add("dialog-open");
    // 先让 translateY(100%) 落到样式，再加 is-open 才能看到上滑。
    void dialogLayer.offsetHeight;
    dialogLayer.classList.add("is-open");
    collectFocusables();
    // 焦点落在最安全的「返回再想想」上。
    backButton.focus();
    document.addEventListener("keydown", onDialogKeydown, true);
  }

  function closeDialog() {
    if (dialogLayer.hidden) return;
    dialogLayer.classList.remove("is-open");
    document.body.classList.remove("dialog-open");
    document.removeEventListener("keydown", onDialogKeydown, true);
    if (lastFocused && typeof lastFocused.focus === "function") lastFocused.focus();
    window.clearTimeout(closeTimer);
    closeTimer = window.setTimeout(function () {
      dialogLayer.hidden = true;
    }, 200);
  }

  Array.prototype.slice.call(dialogLayer.querySelectorAll("[data-sheet-close]")).forEach(function (element) {
    element.addEventListener("click", closeDialog);
  });

  rows.forEach(function (input) {
    input.addEventListener("change", syncGate);
  });

  submit.addEventListener("click", function () {
    if (submit.disabled || settled) return;
    showSuccess();
    // 同意记录成功后固定进入绑定页；绑定逻辑是 #44。
    window.setTimeout(function () {
      window.location.href = "../03-bind/index.html";
    }, 900);
  });

  decline.addEventListener("click", openDialog);
  backButton.addEventListener("click", closeDialog);
  exitButton.addEventListener("click", function () {
    window.location.href = "../02-login/index.html";
  });

  // ---- 预览态 ----

  var state = new URLSearchParams(window.location.search).get("state");
  var byId = {};
  rows.forEach(function (input) {
    byId[input.id] = input;
  });

  if (state === "partial") {
    // 两行已勾、还差 1 份，用于复现 disabled 的提交按钮。
    if (byId["consent-privacy"]) byId["consent-privacy"].checked = true;
    if (byId["consent-terms"]) byId["consent-terms"].checked = true;
  }

  if (state === "all") {
    rows.forEach(function (input) {
      input.checked = true;
    });
  }

  if (state === "decline") {
    window.requestAnimationFrame(openDialog);
  }

  if (state === "success") {
    showSuccess();
  }

  syncGate();
})();
