(function () {
  'use strict';

  var app = document.getElementById('app');
  var webSurface = document.getElementById('webSurface');
  var retryBtn = document.getElementById('retryBtn');
  var backBtn = document.getElementById('backBtn');
  var errorBackBtn = document.getElementById('errorBackBtn');
  var emptyBackBtn = document.getElementById('emptyBackBtn');
  var toastEl = document.getElementById('toast');
  var chips = Array.prototype.slice.call(document.querySelectorAll('.mp-demo-chip'));
  var currentState = 'normal';
  var toastTimer = 0;
  var reloadTimer = 0;

  if (!app) return;

  function hideToast() {
    window.clearTimeout(toastTimer);
    toastTimer = 0;
    if (toastEl) toastEl.classList.remove('is-visible');
  }

  function showToast(message) {
    if (!toastEl) return;
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toastTimer = 0;
      toastEl.classList.remove('is-visible');
    }, 1800);
  }

  function syncChips(state) {
    chips.forEach(function (chip) {
      var active = chip.getAttribute('data-mp-state') === state;
      chip.setAttribute('aria-pressed', String(active));
    });
  }

  function cancelReload() {
    window.clearTimeout(reloadTimer);
    reloadTimer = 0;
  }

  function applyState(state) {
    if (!state) return;
    currentState = state;
    app.setAttribute('data-state', state);
    syncChips(state);
    if (webSurface) webSurface.setAttribute('aria-busy', String(state === 'loading'));
  }

  function setState(state) {
    cancelReload();
    applyState(state);
  }

  function startReload() {
    if (currentState === 'loading') return;
    showToast('正在重新加载网页');
    setState('loading');
    reloadTimer = window.setTimeout(function () {
      reloadTimer = 0;
      applyState('normal');
      showToast('网页已重新加载');
    }, 1200);
  }

  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      var nextState = chip.getAttribute('data-mp-state');
      if (!nextState) return;
      hideToast();
      setState(nextState);
    });
  });

  if (retryBtn) retryBtn.addEventListener('click', startReload);

  if (backBtn) {
    backBtn.addEventListener('click', function () {
      showToast('返回上一页');
    });
  }
  if (errorBackBtn) {
    errorBackBtn.addEventListener('click', function () {
      showToast('返回上一页');
    });
  }
  if (emptyBackBtn) {
    emptyBackBtn.addEventListener('click', function () {
      showToast('返回上一页');
    });
  }

  applyState('normal');
})();
