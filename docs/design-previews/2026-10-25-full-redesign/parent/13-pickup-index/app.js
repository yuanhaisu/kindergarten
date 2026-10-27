/* 养成星球 · 家长端 · 接送记录
 * 页面交互：四态联动、历史记录轻提示、接送二维码底部弹层。
 * 四态由共享外壳的演示条写入 data-state，并派发 mp:state。
 */
(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  /* 轻提示 -------------------------------------------------------------- */
  var toastEl = document.getElementById('toast');
  var toastTimer = null;

  function showToast(message) {
    if (!toastEl || !message) return;
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toastEl.classList.remove('is-visible');
    }, 2000);
  }

  Array.prototype.forEach.call(document.querySelectorAll('[data-toast]'), function (el) {
    el.addEventListener('click', function () {
      showToast(el.getAttribute('data-toast'));
    });
  });

  /* 四态 ---------------------------------------------------------------- */
  var main = document.querySelector('.mp-main');
  var live = document.getElementById('state-live');
  var qrButtons = [document.getElementById('qr-open'), document.getElementById('empty-qr-open')];
  var retryBtn = document.getElementById('retry');
  var retryTimer = null;
  /* 今日卡三态由当天记录派生：无记录 → 未入园；末条为入园 → 在园中；末条为接走 → 已接走。
   * 原型没有真实数据源，用演示芯片切换，data-day 写到 .mp-app 上供 styles.css 分支。 */
  var DAY_BY_STATE = { normal: 'in', picked: 'out', pre: 'pre' };
  var LIVE_TEXT = {
    normal: '',
    picked: '今日接送已完成',
    pre: '今天还没有入园记录',
    empty: '今天还没有接送记录',
    loading: '正在加载接送记录',
    error: '接送记录加载失败'
  };

  function syncState(state) {
    app.setAttribute('data-day', DAY_BY_STATE[state] || 'in');
    if (main) main.setAttribute('aria-busy', state === 'loading' ? 'true' : 'false');
    for (var i = 0; i < qrButtons.length; i++) {
      if (qrButtons[i]) qrButtons[i].disabled = state === 'loading' || state === 'error';
    }
    if (live) live.textContent = LIVE_TEXT[state] || '';
  }

  function setChips(state) {
    var chips = document.querySelectorAll('.mp-demo-chip');
    for (var i = 0; i < chips.length; i++) {
      chips[i].setAttribute('aria-pressed', String(chips[i].getAttribute('data-mp-state') === state));
    }
  }

  function applyState(state) {
    app.setAttribute('data-state', state);
    setChips(state);
    syncState(state);
  }

  function resetRetry() {
    window.clearTimeout(retryTimer);
    retryTimer = null;
    if (!retryBtn) return;
    retryBtn.disabled = false;
    retryBtn.textContent = '重试';
  }

  document.addEventListener('mp:state', function (event) {
    var next = event && event.detail ? event.detail.state : 'normal';
    syncState(next);
    resetRetry();
  });

  if (retryBtn) {
    retryBtn.addEventListener('click', function () {
      if (retryBtn.disabled) return;
      retryBtn.disabled = true;
      retryBtn.textContent = '重试中…';
      window.clearTimeout(retryTimer);
      retryTimer = window.setTimeout(function () {
        retryTimer = null;
        applyState('normal');
        resetRetry();
        showToast('已重新加载');
      }, 600);
    });
  }

  syncState(app.getAttribute('data-state') || 'normal');

  /* 历史记录 ------------------------------------------------------------ */
  Array.prototype.forEach.call(document.querySelectorAll('[data-record]'), function (row) {
    row.addEventListener('click', function () {
      var date = row.getAttribute('data-date') || '';
      var type = row.getAttribute('data-type') || '';
      showToast('已打开 ' + date + ' ' + type + '记录');
    });
  });

  Array.prototype.forEach.call(document.querySelectorAll('[data-refresh]'), function (btn) {
    var spinTimer = null;
    btn.addEventListener('click', function () {
      btn.classList.remove('is-spinning');
      void btn.offsetWidth;
      btn.classList.add('is-spinning');
      window.clearTimeout(spinTimer);
      spinTimer = window.setTimeout(function () {
        btn.classList.remove('is-spinning');
      }, 650);
      showToast('已刷新接送记录');
    });
  });

  /* 底部导航 ------------------------------------------------------------ */
  Array.prototype.forEach.call(document.querySelectorAll('.mp-tab'), function (tab) {
    tab.addEventListener('click', function () {
      var label = tab.getAttribute('data-tab') || '';
      if (tab.getAttribute('aria-current') === 'page') {
        showToast('当前已在' + label);
      } else {
        showToast('已切换到' + label);
      }
    });
  });

  /* 二维码弹层 ---------------------------------------------------------- */
  var layer = document.getElementById('qr-layer');
  var sheet = document.getElementById('qr-sheet');
  var closeBtn = document.getElementById('sheet-close');
  var triggers = [document.getElementById('qr-open'), document.getElementById('empty-qr-open')];
  var isOpen = false;
  var lastTrigger = null;

  function getFocusable() {
    if (!sheet) return [];
    var nodes = sheet.querySelectorAll('button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])');
    var list = [];
    for (var i = 0; i < nodes.length; i++) {
      if (nodes[i].offsetParent !== null) list.push(nodes[i]);
    }
    return list;
  }

  function openSheet(trigger) {
    if (!layer || isOpen) return;
    isOpen = true;
    lastTrigger = trigger || null;
    layer.removeAttribute('inert');
    layer.setAttribute('aria-hidden', 'false');
    layer.classList.add('is-open');
    document.body.style.overflow = 'hidden';
    if (toastEl) toastEl.classList.add('is-sheet');
    window.setTimeout(function () {
      if (isOpen && closeBtn) closeBtn.focus({ preventScroll: true });
    }, 40);
  }

  function closeSheet() {
    if (!layer || !isOpen) return;
    isOpen = false;
    layer.classList.remove('is-open');
    document.body.style.overflow = '';
    if (toastEl) toastEl.classList.remove('is-sheet');
    if (lastTrigger && typeof lastTrigger.focus === 'function') {
      lastTrigger.focus({ preventScroll: true });
    }
    layer.setAttribute('aria-hidden', 'true');
    layer.setAttribute('inert', '');
  }

  Array.prototype.forEach.call(triggers, function (btn) {
    if (!btn) return;
    btn.addEventListener('click', function () { openSheet(btn); });
  });

  if (closeBtn) closeBtn.addEventListener('click', closeSheet);

  var scrim = layer ? layer.querySelector('.sheet-scrim') : null;
  if (scrim) scrim.addEventListener('click', closeSheet);

  document.addEventListener('keydown', function (event) {
    if (!isOpen) return;
    if (event.key === 'Escape' || event.key === 'Esc') {
      event.preventDefault();
      closeSheet();
      return;
    }
    if (event.key !== 'Tab') return;
    var items = getFocusable();
    if (items.length < 1) return;
    var first = items[0];
    var last = items[items.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });

  /* 二维码占位：25×25 模块，只用 currentColor 绘制 ---------------------- */
  var qrPath = document.getElementById('qr-path');
  var qrFrame = document.getElementById('qr-frame');
  var refreshBtn = document.getElementById('qr-refresh');
  var qrSeed = 20261026;

  function mulberry32(a) {
    return function () {
      a |= 0;
      a = (a + 0x6D2B79F5) | 0;
      var t = Math.imul(a ^ (a >>> 15), 1 | a);
      t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
      return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
  }

  function buildQr(seed) {
    if (!qrPath) return;
    var n = 25;
    var grid = [];
    var reserved = [];
    var x;
    var y;

    for (y = 0; y < n; y++) {
      grid[y] = [];
      reserved[y] = [];
      for (x = 0; x < n; x++) {
        grid[y][x] = false;
        reserved[y][x] = false;
      }
    }

    function reserve(x0, y0, w, h) {
      for (var ry = y0; ry < y0 + h; ry++) {
        if (ry < 0 || ry >= n) continue;
        for (var rx = x0; rx < x0 + w; rx++) {
          if (rx < 0 || rx >= n) continue;
          reserved[ry][rx] = true;
        }
      }
    }

    function set(x0, y0, on) {
      if (x0 >= 0 && x0 < n && y0 >= 0 && y0 < n) grid[y0][x0] = on;
    }

    function finder(x0, y0) {
      for (var fy = 0; fy < 7; fy++) {
        for (var fx = 0; fx < 7; fx++) {
          var edge = fx === 0 || fx === 6 || fy === 0 || fy === 6;
          var core = fx >= 2 && fx <= 4 && fy >= 2 && fy <= 4;
          set(x0 + fx, y0 + fy, edge || core);
        }
      }
    }

    reserve(0, 0, 8, 8);
    reserve(n - 8, 0, 8, 8);
    reserve(0, n - 8, 8, 8);
    finder(0, 0);
    finder(n - 7, 0);
    finder(0, n - 7);

    for (var i = 8; i < n - 8; i++) {
      reserve(i, 6, 1, 1);
      reserve(6, i, 1, 1);
      set(i, 6, i % 2 === 0);
      set(6, i, i % 2 === 0);
    }

    for (y = 16; y < 21; y++) {
      for (x = 16; x < 21; x++) {
        var ring = x === 16 || x === 20 || y === 16 || y === 20;
        set(x, y, ring || (x === 18 && y === 18));
      }
    }
    reserve(16, 16, 5, 5);

    var rnd = mulberry32(seed);
    for (y = 0; y < n; y++) {
      for (x = 0; x < n; x++) {
        if (!reserved[y][x] && rnd() < 0.47) grid[y][x] = true;
      }
    }

    var d = '';
    for (y = 0; y < n; y++) {
      for (x = 0; x < n; x++) {
        if (grid[y][x]) d += 'M' + x + ' ' + y + 'h1v1h-1Z';
      }
    }
    qrPath.setAttribute('d', d);
  }

  if (refreshBtn) {
    refreshBtn.addEventListener('click', function () {
      if (refreshBtn.disabled) return;
      refreshBtn.disabled = true;
      if (qrFrame) qrFrame.classList.add('is-refreshing');
      window.setTimeout(function () {
        qrSeed = (Date.now() % 2147483647) >>> 0;
        buildQr(qrSeed);
        if (qrFrame) qrFrame.classList.remove('is-refreshing');
        refreshBtn.disabled = false;
        if (isOpen) showToast('二维码已刷新');
      }, 420);
    });
  }

  buildQr(qrSeed);
})();
