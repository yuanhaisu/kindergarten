(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  var loginBtn = document.getElementById('loginBtn');
  var agreeInput = document.getElementById('agreeInput');
  var agreeRow = document.getElementById('agreeRow');
  var toast = document.getElementById('loginToast');
  var toastText = document.getElementById('toastText');
  var errorTitle = document.getElementById('errorTitle');
  var errorDesc = document.getElementById('errorDesc');
  var demoToggle = document.querySelector('.mp-demo-toggle');
  var chips = Array.prototype.slice.call(document.querySelectorAll('.mp-demo-chip'));

  if (!app || !loginBtn || !agreeInput || !toast || !toastText) return;

  var LOGIN_DELAY = 1400;
  var TOAST_HOLD = 2600;
  var state = 'normal';
  var loginTimer = 0;
  var toastHoldTimer = 0;
  var toastHideTimer = 0;
  var nudgeTimer = 0;

  function setState(next) {
    var busy = next === 'loading';
    state = next;
    app.setAttribute('data-state', next);
    app.setAttribute('aria-busy', busy ? 'true' : 'false');
    loginBtn.disabled = busy;
    loginBtn.setAttribute('aria-busy', busy ? 'true' : 'false');
    agreeInput.disabled = busy;

    chips.forEach(function (chip) {
      var active = chip.getAttribute('data-mp-state') === next;
      chip.setAttribute('aria-pressed', String(active));
    });
  }

  function setErrorCopy(title, desc) {
    if (errorTitle) errorTitle.textContent = title;
    if (errorDesc) errorDesc.textContent = desc;
  }

  function hideToast(immediate) {
    window.clearTimeout(toastHoldTimer);
    window.clearTimeout(toastHideTimer);
    toast.classList.remove('is-on');

    if (immediate) {
      toast.hidden = true;
      return;
    }

    toastHideTimer = window.setTimeout(function () {
      toast.hidden = true;
    }, 200);
  }

  function showToast(message, tone) {
    window.clearTimeout(toastHoldTimer);
    window.clearTimeout(toastHideTimer);

    toastText.textContent = message;
    toast.setAttribute('data-tone', tone || 'warning');
    toast.hidden = false;
    void toast.offsetWidth;
    toast.classList.add('is-on');

    toastHoldTimer = window.setTimeout(function () {
      hideToast(false);
    }, TOAST_HOLD);
  }

  function nudgeAgree() {
    if (!agreeRow) return;

    window.clearTimeout(nudgeTimer);
    agreeRow.classList.remove('is-nudged');
    void agreeRow.offsetWidth;
    agreeRow.classList.add('is-nudged');

    nudgeTimer = window.setTimeout(function () {
      agreeRow.classList.remove('is-nudged');
    }, 500);
  }

  function cancelLogin() {
    window.clearTimeout(loginTimer);
    loginTimer = 0;
  }

  function startLogin() {
    if (state === 'loading') return;

    if (!agreeInput.checked) {
      setState('empty');
      nudgeAgree();
      showToast('请先阅读并同意用户协议和隐私政策', 'warning');
      return;
    }

    hideToast(true);
    setState('loading');

    loginTimer = window.setTimeout(function () {
      loginTimer = 0;

      if (state !== 'loading') return;

      if (window.navigator.onLine === false) {
        // 失败原因只留在卡片内常驻错误条，不再叠加同文案 toast（避免同屏重复与双重播报）
        setState('error');
        setErrorCopy('网络异常', '请检查网络后重试');
        return;
      }

      setState('normal');
      setErrorCopy('微信登录失败', '请稍后重试');
      showToast('登录成功', 'success');
    }, LOGIN_DELAY);
  }

  function closeDemo() {
    if (window.OdShell && typeof window.OdShell.closeDemo === 'function') {
      window.OdShell.closeDemo();
    }

    if (demoToggle && typeof demoToggle.focus === 'function') {
      try {
        demoToggle.focus({ preventScroll: true });
      } catch (error) {
        demoToggle.focus();
      }
    }
  }

  function applyDemo(next) {
    cancelLogin();
    hideToast(true);

    if (next === 'normal') {
      agreeInput.checked = true;
      setState('normal');
      setErrorCopy('微信登录失败', '请稍后重试');
    } else if (next === 'empty') {
      agreeInput.checked = false;
      setState('empty');
      nudgeAgree();
      showToast('请先阅读并同意用户协议和隐私政策', 'warning');
    } else if (next === 'loading') {
      agreeInput.checked = true;
      setState('loading');
    } else if (next === 'error') {
      agreeInput.checked = true;
      setState('error');
      setErrorCopy('微信登录失败', '请稍后重试');
    } else {
      return;
    }

    closeDemo();
  }

  loginBtn.addEventListener('click', startLogin);

  agreeInput.addEventListener('change', function () {
    if (state === 'loading') return;

    // 勾选框是唯一真相源：勾上回正常态，取消则进入「协议未勾选」
    setState(agreeInput.checked ? 'normal' : 'empty');

    if (agreeInput.checked && !toast.hidden && toast.getAttribute('data-tone') === 'warning') {
      hideToast(false);
    }
  });

  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      applyDemo(chip.getAttribute('data-mp-state'));
    });
  });

  setState('normal');
})();
