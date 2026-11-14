(function () {
  'use strict';

  /* ?variant=signup（默认）| signup-closed（可选截止后态） */
  var variant = 'signup';
  try {
    var search = new URLSearchParams(window.location.search);
    if (search.get('variant') === 'signup-closed') variant = 'signup-closed';
  } catch (err) {
    variant = 'signup';
  }
  document.documentElement.setAttribute('data-variant', variant);

  var app = document.getElementById('app');
  var pageBar = document.querySelector('.page-bar');
  var main = document.getElementById('main');
  var actionBar = document.getElementById('action-bar');

  var cta = document.getElementById('signup-cta');
  var ctaLabel = document.getElementById('signup-cta-label');
  var cancelBtn = document.getElementById('cancel-signup');
  var signedNote = document.getElementById('signed-note');
  var deadlineText = document.getElementById('deadline-text');

  var layer = document.getElementById('signup-sheet-layer');
  var scrim = document.getElementById('sheet-scrim');
  var panel = document.getElementById('signup-sheet');
  var confirmBtn = document.getElementById('confirm-signup');
  var dismissBtn = document.getElementById('dismiss-sheet');
  var saveBtn = document.querySelector('[data-od-id="save-attachment"]');

  var toastEl = document.getElementById('toast');

  var isClosed = variant === 'signup-closed';
  var signedUp = false;
  var isOpen = false;
  var isClosing = false;
  var toastTimer = null;

  var prefersReduced = !!(window.matchMedia &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches);
  var closeDelay = prefersReduced ? 0 : 200;

  function showToast(message) {
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toastEl.classList.remove('is-visible');
    }, 2000);
  }

  function setBackgroundInert(on) {
    [pageBar, main, actionBar].forEach(function (el) {
      if (!el) return;
      if (on) el.setAttribute('inert', '');
      else el.removeAttribute('inert');
    });
  }

  function focusEl(el) {
    if (!el) return;
    try {
      el.focus({ preventScroll: true });
    } catch (err) {
      el.focus();
    }
  }

  function setSignedUp(next) {
    signedUp = next;
    signedNote.hidden = !next;
    cancelBtn.hidden = !next;
    cta.disabled = next || isClosed;
    cta.classList.toggle('is-signed', next);

    if (next) {
      ctaLabel.textContent = '已报名';
      cta.removeAttribute('aria-haspopup');
    } else {
      ctaLabel.textContent = isClosed ? '报名已截止' : '我要报名';
      if (!isClosed) cta.setAttribute('aria-haspopup', 'dialog');
    }
  }

  function openSheet() {
    if (signedUp || isClosed || isOpen) return;

    window.clearTimeout(toastTimer);
    toastEl.classList.remove('is-visible');

    isOpen = true;
    isClosing = false;
    setBackgroundInert(true);
    app.classList.add('is-sheet-open');

    layer.hidden = false;
    void layer.offsetHeight;
    layer.classList.add('is-open');
    focusEl(panel);
  }

  function closeSheet(afterClose) {
    if (!isOpen || isClosing) return;

    isClosing = true;
    layer.classList.remove('is-open');
    app.classList.remove('is-sheet-open');

    window.setTimeout(function () {
      layer.hidden = true;
      isOpen = false;
      isClosing = false;
      setBackgroundInert(false);
      if (typeof afterClose === 'function') afterClose();
    }, closeDelay);
  }

  function dismissSheet() {
    closeSheet(function () {
      focusEl(cta);
    });
  }

  function applyClosedState() {
    isClosed = true;
    deadlineText.innerHTML = '报名已于<strong>本周四 18:00</strong>截止，不再补报。';
    cta.disabled = true;
    cta.classList.add('is-closed');
    cta.removeAttribute('aria-haspopup');
    ctaLabel.textContent = '报名已截止';
  }

  cta.addEventListener('click', function () {
    if (signedUp || isClosed) return;
    openSheet();
  });

  confirmBtn.addEventListener('click', function () {
    if (isClosing) return;

    closeSheet(function () {
      setSignedUp(true);
      showToast('报名成功');
      focusEl(cancelBtn);
    });
  });

  dismissBtn.addEventListener('click', dismissSheet);
  scrim.addEventListener('click', dismissSheet);

  cancelBtn.addEventListener('click', function () {
    if (!signedUp) return;
    setSignedUp(false);
    showToast('已取消报名');
    focusEl(cta);
  });

  if (saveBtn) {
    saveBtn.addEventListener('click', function () {
      showToast('活动海报已保存');
    });
  }

  layer.addEventListener('keydown', function (e) {
    if (!isOpen) return;

    if (e.key === 'Escape') {
      if (isClosing) return;
      e.preventDefault();
      dismissSheet();
      return;
    }

    if (e.key !== 'Tab' || isClosing) return;

    var focusables = Array.prototype.slice.call(
      panel.querySelectorAll('button:not([disabled])')
    );
    if (!focusables.length) return;

    var first = focusables[0];
    var last = focusables[focusables.length - 1];
    var active = document.activeElement;

    if (active === panel) {
      e.preventDefault();
      focusEl(e.shiftKey ? last : first);
      return;
    }

    if (e.shiftKey && active === first) {
      e.preventDefault();
      focusEl(last);
    } else if (!e.shiftKey && active === last) {
      e.preventDefault();
      focusEl(first);
    }
  });

  setSignedUp(false);
  if (isClosed) applyClosedState();
})();
