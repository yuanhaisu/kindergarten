/* 我的 · 家长端交互
 * 状态演示由 ../../_shared/shell.js 驱动；此文件负责页面状态、toast、确认框。
 */
(function () {
  'use strict';

  var app = document.getElementById('app');
  var page = document.getElementById('profilePage');
  var announcer = document.getElementById('stateAnnouncer');
  var toast = document.getElementById('toast');
  var logoutButton = document.getElementById('logoutButton');
  var dialogLayer = document.getElementById('logoutDialog');
  var cancelButton = document.getElementById('logoutCancel');
  var confirmButton = document.getElementById('logoutConfirm');
  var reloginButton = document.getElementById('reloginButton');
  var retryButton = document.getElementById('retryButton');
  var retryLabel = document.getElementById('retryLabel');
  var demoPanel = document.getElementById('demoPanel');
  var normalCard = page ? page.querySelector('.identity-pane--user') : null;

  var STATES = ['normal', 'empty', 'loading', 'error'];
  var ANNOUNCEMENTS = {
    normal: '已显示家长账号信息',
    empty: '还没有读取到家长信息',
    loading: '正在加载家长账号信息',
    error: '账号信息加载失败'
  };

  var toastTimer = 0;
  var toastCleanupTimer = 0;
  var retryTimer = 0;
  var lastFocus = null;

  if (!app || !page) {
    return;
  }

  function syncChips(state) {
    if (!demoPanel) {
      return;
    }
    var chips = demoPanel.querySelectorAll('.mp-demo-chip');
    Array.prototype.forEach.call(chips, function (chip) {
      chip.setAttribute('aria-pressed', chip.getAttribute('data-mp-state') === state ? 'true' : 'false');
    });
  }

  function resetRetry() {
    if (retryTimer) {
      window.clearTimeout(retryTimer);
      retryTimer = 0;
    }
    if (!retryButton) {
      return;
    }
    retryButton.disabled = false;
    retryButton.classList.remove('is-retrying');
    retryButton.setAttribute('aria-busy', 'false');
    if (retryLabel) {
      retryLabel.textContent = '重试';
    }
  }

  function announce(state) {
    if (!announcer) {
      return;
    }
    announcer.textContent = ANNOUNCEMENTS[state] || '';
  }

  function applyState(state, options) {
    var next = STATES.indexOf(state) === -1 ? 'normal' : state;
    var settings = options || {};

    if (next !== 'error') {
      resetRetry();
    }

    page.setAttribute('data-state', next);
    app.setAttribute('data-state', next);
    page.setAttribute('aria-busy', next === 'loading' ? 'true' : 'false');
    syncChips(next);

    if (settings.announce) {
      announce(next);
    }
  }

  function showToast(message) {
    if (!toast) {
      return;
    }
    if (toastTimer) {
      window.clearTimeout(toastTimer);
    }
    if (toastCleanupTimer) {
      window.clearTimeout(toastCleanupTimer);
    }
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
      toastTimer = 0;
      toastCleanupTimer = window.setTimeout(function () {
        if (!toast.classList.contains('is-visible')) {
          toast.textContent = '';
        }
        toastCleanupTimer = 0;
      }, 260);
    }, 1800);
  }

  function focusIdentity() {
    if (!normalCard) {
      return;
    }
    window.requestAnimationFrame(function () {
      try {
        normalCard.focus({ preventScroll: true });
      } catch (error) {
        normalCard.focus();
      }
    });
  }

  function openDialog() {
    if (!dialogLayer || !dialogLayer.hidden) {
      return;
    }
    lastFocus = document.activeElement;
    dialogLayer.hidden = false;
    document.body.classList.add('dialog-open');
    window.requestAnimationFrame(function () {
      if (cancelButton) {
        try {
          cancelButton.focus({ preventScroll: true });
        } catch (error) {
          cancelButton.focus();
        }
      }
    });
  }

  function closeDialog(restoreFocus) {
    if (!dialogLayer || dialogLayer.hidden) {
      return;
    }
    dialogLayer.hidden = true;
    document.body.classList.remove('dialog-open');
    var target = lastFocus;
    lastFocus = null;

    if (restoreFocus !== false && target && typeof target.focus === 'function' && document.documentElement.contains(target)) {
      try {
        target.focus({ preventScroll: true });
      } catch (error) {
        target.focus();
      }
    }
  }

  /* 菜单跳转在本原型中以 toast 代替 */
  var menuRows = document.querySelectorAll('.menu-row[data-menu-title]');
  Array.prototype.forEach.call(menuRows, function (row) {
    row.addEventListener('click', function () {
      showToast('已打开「' + row.getAttribute('data-menu-title') + '」');
    });
  });

  /* tabBar 只做单页反馈，不离开当前原型 */
  var tabButtons = document.querySelectorAll('.profile-tabbar [data-tab-title]');
  Array.prototype.forEach.call(tabButtons, function (button) {
    button.addEventListener('click', function () {
      var title = button.getAttribute('data-tab-title');
      if (button.getAttribute('aria-current') === 'page') {
        showToast('已在「' + title + '」');
      } else {
        showToast('「' + title + '」页为原型占位');
      }
    });
  });

  if (reloginButton) {
    reloginButton.addEventListener('click', function () {
      applyState('normal', { announce: false });
      showToast('已重新登录');
      focusIdentity();
    });
  }

  if (retryButton) {
    retryButton.addEventListener('click', function () {
      if (retryButton.disabled) {
        return;
      }
      retryButton.disabled = true;
      retryButton.classList.add('is-retrying');
      retryButton.setAttribute('aria-busy', 'true');
      if (retryLabel) {
        retryLabel.textContent = '重试中…';
      }
      retryTimer = window.setTimeout(function () {
        retryTimer = 0;
        applyState('normal', { announce: false });
        showToast('已重新加载');
        focusIdentity();
      }, 1000);
    });
  }

  if (logoutButton) {
    logoutButton.addEventListener('click', openDialog);
  }

  if (cancelButton) {
    cancelButton.addEventListener('click', function () {
      closeDialog(true);
    });
  }

  if (confirmButton) {
    confirmButton.addEventListener('click', function () {
      closeDialog(true);
      showToast('已退出登录（原型）');
    });
  }

  if (dialogLayer) {
    dialogLayer.addEventListener('click', function (event) {
      if (event.target === dialogLayer) {
        closeDialog(true);
        return;
      }
      if (event.target && event.target.classList && event.target.classList.contains('dialog-backdrop')) {
        closeDialog(true);
      }
    });

    dialogLayer.addEventListener('keydown', function (event) {
      if (event.key !== 'Tab') {
        return;
      }
      var focusable = dialogLayer.querySelectorAll('button:not(:disabled)');
      if (!focusable.length) {
        return;
      }
      var first = focusable[0];
      var last = focusable[focusable.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    });
  }

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && dialogLayer && !dialogLayer.hidden) {
      event.preventDefault();
      closeDialog(true);
    }
  });

  document.addEventListener('mp:state', function (event) {
    var next = event && event.detail ? event.detail.state : '';
    applyState(next, { announce: true });
  });

  applyState(app.getAttribute('data-state') || 'normal', { announce: false });
})();
