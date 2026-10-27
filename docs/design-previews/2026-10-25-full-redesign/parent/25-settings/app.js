(function () {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const rows = Array.from(app.querySelectorAll('.switch-row'));
  const demoPickupRow = app.querySelector('.switch-row[data-key="pickup_push"]');
  const toast = document.getElementById('toast');
  const backBtn = document.getElementById('backBtn');
  const logoutBtn = document.getElementById('logoutBtn');
  const dialog = document.getElementById('logoutDialog');
  const dialogScrim = dialog ? dialog.querySelector('[data-dialog-close]') : null;
  const dialogCancel = document.getElementById('logoutCancel');
  const dialogConfirm = document.getElementById('logoutConfirm');
  const reloadBtn = document.getElementById('reloadBtn');
  const retryBtn = document.getElementById('retryBtn');

  const syncTimers = new Map();
  let toastTimer = 0;
  let retryTimer = 0;
  let lastFocused = null;

  function currentState() {
    return app.dataset.state || 'normal';
  }

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 1800);
  }

  function hideToast() {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.classList.remove('is-visible');
  }

  function clearRowSync() {
    syncTimers.forEach(function (timer) {
      window.clearTimeout(timer);
    });
    syncTimers.clear();
    rows.forEach(function (row) {
      row.classList.remove('is-syncing');
      row.removeAttribute('aria-disabled');
      row.removeAttribute('aria-busy');
    });
  }

  function resetRetry() {
    if (retryTimer) {
      window.clearTimeout(retryTimer);
      retryTimer = 0;
    }
    if (reloadBtn) {
      reloadBtn.disabled = false;
      reloadBtn.textContent = '重新加载';
    }
    if (retryBtn) {
      retryBtn.disabled = false;
      retryBtn.textContent = '重试';
    }
  }

  function closeDialog(restoreFocus) {
    if (!dialog || dialog.hidden) return;
    dialog.hidden = true;
    document.documentElement.classList.remove('has-dialog');
    document.body.classList.remove('has-dialog');
    if (restoreFocus && lastFocused && document.contains(lastFocused) && typeof lastFocused.focus === 'function') {
      lastFocused.focus({ preventScroll: true });
    }
    lastFocused = null;
  }

  function applyDemoState(state) {
    hideToast();
    clearRowSync();
    resetRetry();
    closeDialog(false);

    if (state === 'disabled' && demoPickupRow) {
      demoPickupRow.setAttribute('aria-disabled', 'true');
      demoPickupRow.setAttribute('aria-busy', 'true');
    }
  }

  function settleSwitch(row) {
    syncTimers.delete(row);
    row.classList.remove('is-syncing');
    row.removeAttribute('aria-disabled');
    row.removeAttribute('aria-busy');
    showToast('已更新');
  }

  function toggleRow(row) {
    if (!row) return;
    if (row.classList.contains('is-syncing')) return;
    if (row.getAttribute('aria-disabled') === 'true') return;

    const state = currentState();
    if (state === 'empty' || state === 'loading' || state === 'error') return;
    if (state === 'disabled' && row === demoPickupRow) return;

    const nextChecked = row.getAttribute('aria-checked') !== 'true';
    row.setAttribute('aria-checked', String(nextChecked));
    row.classList.add('is-syncing');
    row.setAttribute('aria-disabled', 'true');
    row.setAttribute('aria-busy', 'true');

    const timer = window.setTimeout(function () {
      settleSwitch(row);
    }, 450);
    syncTimers.set(row, timer);
  }

  function goToNormal() {
    const chips = Array.from(document.querySelectorAll('.mp-demo-chip'));
    const normalChip = document.querySelector('.mp-demo-chip[data-mp-state="normal"]');

    if (normalChip) normalChip.click();

    if (currentState() !== 'normal') {
      app.dataset.state = 'normal';
      chips.forEach(function (chip) {
        chip.setAttribute('aria-pressed', String(chip === normalChip));
      });
    }
  }

  function startRetry(button) {
    if (!button || button.disabled || retryTimer) return;
    const state = currentState();
    if (state !== 'empty' && state !== 'error') return;

    button.disabled = true;
    button.textContent = '重试中…';

    retryTimer = window.setTimeout(function () {
      retryTimer = 0;
      goToNormal();
      button.disabled = false;
      button.textContent = button === reloadBtn ? '重新加载' : '重试';
      showToast('设置已刷新');
      if (rows[0]) rows[0].focus({ preventScroll: true });
    }, 1000);
  }

  function openDialog() {
    if (!dialog || !dialog.hidden) return;
    hideToast();
    lastFocused = document.activeElement;
    dialog.hidden = false;
    document.documentElement.classList.add('has-dialog');
    document.body.classList.add('has-dialog');
    // Focus the dialog itself, not 取消：焦点环若落在「取消」上，会被读成
    // 「推荐这个选项」，与右侧的危险操作语义相反。Tab 仍由下面的键盘处理接管。
    const dialogEl = dialog.querySelector('.dialog');
    if (dialogEl) dialogEl.focus({ preventScroll: true });
    else if (dialogCancel) dialogCancel.focus({ preventScroll: true });
  }

  rows.forEach(function (row) {
    row.addEventListener('click', function () {
      toggleRow(row);
    });
  });

  if (backBtn) {
    backBtn.addEventListener('click', function () {
      showToast('返回上一页');
    });
  }

  if (reloadBtn) {
    reloadBtn.addEventListener('click', function () {
      startRetry(reloadBtn);
    });
  }

  if (retryBtn) {
    retryBtn.addEventListener('click', function () {
      startRetry(retryBtn);
    });
  }

  if (logoutBtn) {
    logoutBtn.addEventListener('click', openDialog);
  }

  if (dialogScrim) {
    dialogScrim.addEventListener('click', function () {
      closeDialog(true);
    });
  }

  if (dialogCancel) {
    dialogCancel.addEventListener('click', function () {
      closeDialog(true);
    });
  }

  if (dialogConfirm) {
    dialogConfirm.addEventListener('click', function () {
      closeDialog(true);
      showToast('已退出登录');
    });
  }

  document.addEventListener('keydown', function (event) {
    if (!dialog || dialog.hidden) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      closeDialog(true);
      return;
    }

    if (event.key !== 'Tab') return;

    const focusable = Array.from(dialog.querySelectorAll('button:not([disabled])'));
    if (!focusable.length) return;

    const first = focusable[0];
    const last = focusable[focusable.length - 1];

    if (!dialog.contains(document.activeElement)) {
      event.preventDefault();
      first.focus();
      return;
    }

    // 初始焦点在对话框容器上：Tab / Shift+Tab 直接跳到首/末按钮，
    // 否则 Shift+Tab 会漏回遮罩背后的「退出登录」按钮。
    const dialogEl = dialog.querySelector('.dialog');
    if (document.activeElement === dialogEl) {
      event.preventDefault();
      (event.shiftKey ? last : first).focus();
      return;
    }

    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });

  document.addEventListener('mp:state', function (event) {
    const state = event.detail && event.detail.state;
    if (!state) return;
    applyDemoState(state);
  });
})();
