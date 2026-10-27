/* 我的（老师端）· 交互
 * 只做 toast / 退出确认对话框 / 失败重试 / 空态刷新。
 * 四态切换归 _shared/shell.js 管，这里不重复实现。 */

(function () {
  "use strict";

  var app = document.querySelector(".mp-app");
  var toastEl = document.getElementById("toast");
  var overlay = document.getElementById("logout-overlay");
  var logoutBtn = document.getElementById("logout");
  var confirmBtn = document.getElementById("logout-confirm");
  var retryBtn = document.getElementById("retry");
  var refreshBtn = document.getElementById("refresh-profile");
  var backBtn = document.querySelector(".mp-back");
  var lastFocus = null;
  var toastTimer = null;
  var refreshTimer = null;

  function toast(message) {
    if (!toastEl) return;
    toastEl.textContent = message;
    toastEl.classList.add("is-on");
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () {
      toastEl.classList.remove("is-on");
    }, 2000);
  }

  /* ---------- 退出登录对话框 ---------- */

  function openDialog() {
    if (!overlay) return;
    lastFocus = document.activeElement;
    overlay.classList.add("is-open");
    overlay.removeAttribute("aria-hidden");
    // 遮罩的 visibility 有过渡，同步 focus 会打在仍不可见的元素上
    requestAnimationFrame(function () {
      if (confirmBtn) confirmBtn.focus();
    });
  }

  function closeDialog(restoreFocus) {
    if (!overlay || !overlay.classList.contains("is-open")) return;
    overlay.classList.remove("is-open");
    overlay.setAttribute("aria-hidden", "true");
    if (restoreFocus && lastFocus && typeof lastFocus.focus === "function") {
      lastFocus.focus();
    }
    lastFocus = null;
  }

  if (overlay) overlay.setAttribute("aria-hidden", "true");

  if (logoutBtn) {
    logoutBtn.addEventListener("click", openDialog);
  }

  if (overlay) {
    overlay.addEventListener("click", function (event) {
      if (event.target === overlay) closeDialog(true);
    });

    overlay.querySelectorAll("[data-close]").forEach(function (btn) {
      btn.addEventListener("click", function () {
        closeDialog(true);
      });
    });
  }

  if (confirmBtn) {
    confirmBtn.addEventListener("click", function () {
      closeDialog(false);
      toast("已退出登录");
    });
  }

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape") closeDialog(true);
  });

  /* ---------- 设置入口（源码三个都没有落地页） ---------- */

  document.querySelectorAll("[data-feature]").forEach(function (cell) {
    cell.addEventListener("click", function () {
      toast("功能开发中");
    });
  });

  /* ---------- 失败态重试：切回正常态 ---------- */

  /* 四态由外壳属性驱动；代码切态时同步演示 chip 的选中态，保持二者一致 */
  function setState(state) {
    if (app) app.setAttribute("data-state", state);
    document.querySelectorAll(".mp-demo-chip").forEach(function (chip) {
      var active = chip.getAttribute("data-mp-state") === state;
      chip.setAttribute("aria-pressed", active ? "true" : "false");
    });
  }

  if (retryBtn) {
    retryBtn.addEventListener("click", function () {
      setState("normal");
      toast("今日数据已更新");
    });
  }

  /* ---------- 空态刷新资料 ---------- */

  /* 源码 loadProfile() 每次进页都重跑 getUser() + overview，「刷新」就是再取一次数。
   * 旧实现只 toast「资料已刷新」而页面仍停在 empty、园所仍是「未绑定园所」，
   * 是虚假成功反馈；现在走 加载 → 正常，让按钮真的有结果。 */
  function resetRefreshBtn() {
    if (!refreshBtn) return;
    refreshBtn.disabled = false;
    refreshBtn.removeAttribute("aria-busy");
    refreshBtn.textContent = "刷新资料";
  }

  /* 演示条手动切态时作废未到点的刷新，避免定时器回头覆盖用户刚选的状态 */
  document.addEventListener("mp:state", function () {
    if (refreshTimer === null) return;
    clearTimeout(refreshTimer);
    refreshTimer = null;
    resetRefreshBtn();
  });

  if (refreshBtn) {
    refreshBtn.addEventListener("click", function () {
      if (refreshTimer !== null) return;
      refreshBtn.disabled = true;
      refreshBtn.setAttribute("aria-busy", "true");
      refreshBtn.textContent = "刷新中…";
      setState("loading");
      refreshTimer = setTimeout(function () {
        refreshTimer = null;
        resetRefreshBtn();
        setState("normal");
        toast("资料已更新");
      }, 900);
    });
  }

  /* ---------- 返回 ---------- */

  if (backBtn) {
    backBtn.addEventListener("click", function () {
      toast("已返回 老师工作台");
    });
  }
})();
