(function () {
  'use strict';

  var app = document.getElementById('app');
  if (!app) return;

  var webCanvas = document.getElementById('webCanvas');
  var openBtn = document.getElementById('openBtn');
  var retryBtn = document.getElementById('retryBtn');
  var backBtn = document.getElementById('backBtn');
  var confirmBackBtn = document.getElementById('confirmBackBtn');
  var errorBackBtn = document.getElementById('errorBackBtn');
  var emptyBackBtn = document.getElementById('emptyBackBtn');
  var toastEl = document.getElementById('toast');
  var chips = Array.prototype.slice.call(document.querySelectorAll('[data-mp-state]'));
  var allowedStates = ['confirm', 'normal', 'empty', 'loading', 'error'];
  var currentState = 'normal';
  var openTimer = null;
  var toastTimer = null;

  function isAllowedState(state) {
    return allowedStates.indexOf(state) !== -1;
  }

  function cancelOpen() {
    if (openTimer !== null) {
      window.clearTimeout(openTimer);
      openTimer = null;
    }
  }

  function hideToast() {
    if (toastTimer !== null) {
      window.clearTimeout(toastTimer);
      toastTimer = null;
    }
    if (toastEl) {
      toastEl.classList.remove('is-visible');
    }
  }

  function showToast(message) {
    if (!toastEl) return;
    if (toastTimer !== null) {
      window.clearTimeout(toastTimer);
    }
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toastTimer = null;
      toastEl.classList.remove('is-visible');
    }, 1800);
  }

  function syncChips(state) {
    chips.forEach(function (chip) {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === state));
    });
  }

  function applyState(state) {
    var nextState = isAllowedState(state) ? state : 'normal';
    var isLoading = nextState === 'loading';

    currentState = nextState;
    app.setAttribute('data-state', nextState);
    syncChips(nextState);

    if (webCanvas) {
      webCanvas.setAttribute('aria-busy', String(isLoading));
    }
  }

  // 确认后才挂载 web-view：挂载即铺满整页，此前的确认与来源信息必须在它之前。
  function openExternal() {
    if (currentState === 'loading') return;

    cancelOpen();
    applyState('loading');

    openTimer = window.setTimeout(function () {
      openTimer = null;
      applyState('normal');
    }, 1200);
  }

  function onBack() {
    showToast('返回上一页');
  }

  document.addEventListener('mp:state', function (event) {
    var state = event && event.detail ? event.detail.state : '';
    if (!isAllowedState(state)) return;

    cancelOpen();
    hideToast();
    applyState(state);
  });

  if (openBtn) openBtn.addEventListener('click', openExternal);
  if (retryBtn) retryBtn.addEventListener('click', openExternal);
  if (backBtn) backBtn.addEventListener('click', onBack);
  if (confirmBackBtn) confirmBackBtn.addEventListener('click', onBack);
  if (errorBackBtn) errorBackBtn.addEventListener('click', onBack);
  if (emptyBackBtn) emptyBackBtn.addEventListener('click', onBack);

  window.addEventListener('beforeunload', cancelOpen);

  applyState(app.getAttribute('data-state') || 'normal');
})();
