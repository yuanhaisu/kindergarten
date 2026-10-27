(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  var toast = document.getElementById('toast');
  var toastTimer = 0;
  var validStates = {
    normal: true,
    empty: true,
    loading: true,
    error: true
  };

  function hideToast() {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toastTimer = 0;
    toast.classList.remove('is-on');
  }

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.remove('is-on');
    void toast.offsetWidth;
    toast.classList.add('is-on');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-on');
      toastTimer = 0;
    }, 1900);
  }

  function syncChips(state) {
    var chips = app.querySelectorAll('[data-mp-state]');
    var index;
    for (index = 0; index < chips.length; index += 1) {
      chips[index].setAttribute(
        'aria-pressed',
        chips[index].getAttribute('data-mp-state') === state ? 'true' : 'false'
      );
    }
  }

  function setState(state) {
    if (validStates[state] !== true) return;
    app.setAttribute('data-state', state);
    app.setAttribute('aria-busy', state === 'loading' ? 'true' : 'false');
    syncChips(state);
    hideToast();
  }

  document.addEventListener('click', function (event) {
    var target = event.target;
    var row;
    var chip;
    var entry;
    var firstRow;

    if (!target || typeof target.closest !== 'function') return;

    row = target.closest('.menu-row');
    if (row && app.contains(row)) {
      entry = row.getAttribute('data-entry');
      if (entry) showToast('已打开『' + entry + '』');
      return;
    }

    chip = target.closest('[data-mp-state]');
    if (chip && app.contains(chip)) {
      setState(chip.getAttribute('data-mp-state'));
      return;
    }

    if (target.closest('#empty-back')) {
      showToast('已返回工作台');
      return;
    }

    if (target.closest('#retry-button')) {
      setState('normal');
      firstRow = app.querySelector('.menu-row');
      if (firstRow) firstRow.focus({ preventScroll: true });
      return;
    }

    if (target.closest('.mp-back')) {
      showToast('已返回老师工作台');
    }
  });
})();
