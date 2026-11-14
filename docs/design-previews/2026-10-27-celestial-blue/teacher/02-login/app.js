/* 老师登录 · 养成星球
 * 页面交互只做三件事：切换 data-state、同步 aria、弹 toast。
 * 演示条的折叠与 chip 选中由 ../../_shared/shell.js 负责，这里不重复绑定。
 *
 * 状名：normal / empty / unagreed / loading / pending / error
 *   empty / unagreed  → 协议未勾选（empty 是首屏初值，对齐源码 agreed = ref(false)）
 *   loading / pending → 登录中（loading 来自「加载」芯片，pending 来自点登录）
 */
(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  var btn = document.getElementById('login-btn');
  var agreeRow = document.getElementById('agree-row');
  var toastEl = document.getElementById('toast');
  var chips = Array.prototype.slice.call(document.querySelectorAll('.mp-demo-chip'));

  var LOGIN_DELAY = 1400;
  var TOAST_DURATION = 2000;
  var CHECKED = ['normal', 'error', 'loading', 'pending'];
  var BUSY = ['loading', 'pending'];

  var loginTimer = null;
  var toastTimer = null;

  function current() {
    return app.dataset.state || 'normal';
  }

  function isBusy(state) {
    return BUSY.indexOf(state) >= 0;
  }

  function clearLoginTimer() {
    if (loginTimer) {
      clearTimeout(loginTimer);
      loginTimer = null;
    }
  }

  /* 视觉全部由 styles.css 的 [data-state] 规则接管，这里只补 a11y 与可点性 */
  function syncA11y() {
    var state = current();
    var busy = isBusy(state);
    btn.disabled = busy;
    agreeRow.setAttribute('aria-checked', String(CHECKED.indexOf(state) >= 0));
    agreeRow.setAttribute('aria-disabled', String(busy));
    agreeRow.setAttribute('tabindex', busy ? '-1' : '0');
  }

  function setState(next) {
    clearLoginTimer();
    app.dataset.state = next;
    // 非芯片状态（unagreed / pending）不对应任何 chip，四枚全部取消选中
    chips.forEach(function (chip) {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === next));
    });
    syncA11y();
  }

  function showToast(message) {
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(function () {
      toastEl.classList.remove('is-visible');
      toastTimer = null;
    }, TOAST_DURATION);
  }

  /* 演示条切状态时，别让上一轮的登录定时器把状态又改回去 */
  document.addEventListener('mp:state', function () {
    clearLoginTimer();
    syncA11y();
  });

  btn.addEventListener('click', function () {
    var state = current();
    if (isBusy(state)) return;

    if (CHECKED.indexOf(state) < 0) {
      setState('unagreed');
      showToast('请先阅读并同意用户协议和隐私政策');
      return;
    }

    setState('pending');
    loginTimer = setTimeout(function () {
      loginTimer = null;
      setState('normal');
      showToast('登录成功');
    }, LOGIN_DELAY);
  });

  agreeRow.addEventListener('click', function () {
    if (isBusy(current())) return;
    setState(CHECKED.indexOf(current()) >= 0 ? 'empty' : 'normal');
  });

  syncA11y();
})();
