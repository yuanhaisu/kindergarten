/* 积分与成就 · 任务积分提案
 * 预览态：?dataset=task-points  ·  ?sheet=rules|ledger|achievements
 * 只做页面内交互，不发网络请求，不引入框架。
 */
(function () {
  'use strict';

  var params = new URLSearchParams(window.location.search);
  var dataset = params.get('dataset') || 'task-points';
  document.documentElement.setAttribute('data-dataset', dataset);

  var app = document.querySelector('.mp-app');
  var main = document.querySelector('.mp-main');
  var layer = document.getElementById('sheet-layer');
  var toast = document.getElementById('toast');
  var panels = Array.prototype.slice.call(document.querySelectorAll('.sheet-panel'));
  var background = Array.prototype.slice.call(
    document.querySelectorAll('.mp-statusbar, .page-bar, .mp-main')
  );

  var activePanel = null;
  var lastTrigger = null;
  var closeTimer = 0;
  var toastTimer = 0;

  if (app) app.setAttribute('data-dataset', dataset);

  function panelFor(name) {
    return document.getElementById('sheet-' + name);
  }

  function setBackgroundInert(isInert) {
    background.forEach(function (el) {
      if (!el) return;
      if (isInert) {
        el.setAttribute('inert', '');
      } else {
        el.removeAttribute('inert');
      }
    });
  }

  function getFocusable(panel) {
    return Array.prototype.slice
      .call(
        panel.querySelectorAll(
          'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        )
      )
      .filter(function (el) {
        return !el.disabled && el.offsetParent !== null;
      });
  }

  function openSheet(name, trigger) {
    var panel = panelFor(name);
    if (!panel) return;

    panels.forEach(function (item) {
      item.classList.remove('is-active');
      if (item !== panel) item.hidden = true;
    });

    activePanel = panel;
    if (trigger) {
      lastTrigger = trigger;
    } else if (
      document.activeElement &&
      document.activeElement !== document.body &&
      document.activeElement !== document.documentElement
    ) {
      lastTrigger = document.activeElement;
    }

    panel.hidden = false;
    void panel.offsetHeight;
    panel.classList.add('is-active');

    layer.classList.add('is-open');
    layer.setAttribute('aria-hidden', 'false');
    main.classList.add('is-locked');
    setBackgroundInert(true);

    window.clearTimeout(closeTimer);

    var closeButton = panel.querySelector('.sheet-close');
    if (closeButton) {
      window.setTimeout(function () {
        try {
          closeButton.focus({ preventScroll: true });
        } catch (err) {
          closeButton.focus();
        }
      }, 60);
    }
  }

  function closeSheet() {
    var panel = activePanel;
    if (!panel) return;

    panel.classList.remove('is-active');
    layer.classList.remove('is-open');
    layer.setAttribute('aria-hidden', 'true');
    main.classList.remove('is-locked');
    setBackgroundInert(false);
    activePanel = null;

    window.clearTimeout(closeTimer);
    closeTimer = window.setTimeout(function () {
      panels.forEach(function (item) {
        if (!item.classList.contains('is-active')) item.hidden = true;
      });
    }, 320);

    var target =
      lastTrigger && lastTrigger !== document.body && lastTrigger !== document.documentElement
        ? lastTrigger
        : document.querySelector('[data-action="back"]');

    if (target && typeof target.focus === 'function') {
      window.setTimeout(function () {
        try {
          target.focus({ preventScroll: true });
        } catch (err) {
          target.focus();
        }
      }, 0);
    }

    lastTrigger = null;
  }

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-show');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-show');
    }, 2000);
  }

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!target || !target.closest) return;

    var opener = target.closest('[data-sheet]');
    if (opener) {
      event.preventDefault();
      openSheet(opener.getAttribute('data-sheet'), opener);
      return;
    }

    if (target.closest('[data-close]')) {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (target.closest('[data-action="back"]')) {
      showToast('返回「我的」');
    }
  });

  document.addEventListener('keydown', function (event) {
    if (!activePanel) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (event.key !== 'Tab') return;

    var focusable = getFocusable(activePanel);
    if (!focusable.length) {
      event.preventDefault();
      return;
    }

    var first = focusable[0];
    var last = focusable[focusable.length - 1];
    var inside = activePanel.contains(document.activeElement);

    if (event.shiftKey && (!inside || document.activeElement === first)) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && (!inside || document.activeElement === last)) {
      event.preventDefault();
      first.focus();
    }
  });

  var initialSheet = params.get('sheet');
  if (initialSheet && /^(rules|ledger|achievements)$/.test(initialSheet)) {
    window.setTimeout(function () {
      openSheet(initialSheet, null);
    }, 80);
  }
})();
