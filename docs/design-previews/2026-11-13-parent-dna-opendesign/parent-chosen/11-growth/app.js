(function () {
  'use strict';

  var root = document.documentElement;
  var params = null;

  try {
    params = new URLSearchParams(window.location.search);
  } catch (e) {
    /* no query support — keep the default feed */
  }

  var initialState = params ? params.get('state') : null;
  if (initialState === 'empty') {
    root.setAttribute('data-view', 'empty');
  }

  function ready(fn) {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', fn);
    } else {
      fn();
    }
  }

  ready(function () {
    var app = document.querySelector('.mp-app');
    var main = document.querySelector('.mp-main');
    var tabbar = document.querySelector('.tabbar');

    /* ----- scope switch: 只看孩子 / 看全部 ----- */

    var modes = Array.prototype.slice.call(document.querySelectorAll('[data-mode]'));
    var records = Array.prototype.slice.call(document.querySelectorAll('[data-record]'));
    var hint = document.getElementById('scope-hint');
    var hints = {
      individual: '只显示与乐乐相关的内容；切换「看全部」可浏览班级动态',
      class: '包含班级其他孩子的动态；切换「只看孩子」可聚焦当前孩子'
    };

    function setMode(mode) {
      modes.forEach(function (button) {
        button.setAttribute('aria-pressed', button.getAttribute('data-mode') === mode ? 'true' : 'false');
      });
      records.forEach(function (card) {
        var visible = mode !== 'individual' || card.getAttribute('data-scope') === 'individual';
        card.hidden = !visible;
      });
      if (hint && hints[mode]) {
        hint.textContent = hints[mode];
      }
    }

    modes.forEach(function (button) {
      button.addEventListener('click', function () {
        setMode(button.getAttribute('data-mode'));
      });
    });

    /* ----- photo viewer sheet ----- */

    var layer = document.getElementById('photo-layer');
    var viewer = document.getElementById('photo-viewer');
    var frame = document.getElementById('photo-frame');
    var countNode = document.getElementById('photo-count');
    var captionNode = document.getElementById('photo-caption');
    var closeButton = layer ? layer.querySelector('.sheet-close') : null;
    var tileClasses = ['tile--sage', 'tile--gold', 'tile--pine', 'tile--blue'];
    var sheetOpen = false;
    var lastFocused = null;

    function openViewer(tile) {
      if (!layer || !viewer || !frame || !countNode || !captionNode || !closeButton) {
        return;
      }

      var grid = tile.closest('.photo-grid');
      var record = tile.closest('.record');
      var titleNode = record ? record.querySelector('.record-title') : null;
      var total = grid ? parseInt(grid.getAttribute('data-photos'), 10) || 1 : 1;
      var index = parseInt(tile.getAttribute('data-photo-index'), 10) || 1;
      var tileClass = 'tile--sage';

      tileClasses.forEach(function (name) {
        if (tile.classList.contains(name)) {
          tileClass = name;
        }
      });

      frame.className = 'viewer-frame ' + tileClass;
      countNode.textContent = '照片 ' + index + ' / ' + total;
      captionNode.textContent = titleNode ? titleNode.textContent.trim() : '成长记录';

      lastFocused = document.activeElement;
      sheetOpen = true;
      layer.removeAttribute('inert');
      layer.setAttribute('aria-hidden', 'false');
      layer.classList.add('is-open');
      if (app) {
        app.classList.add('is-sheet-open');
      }
      if (main) {
        main.setAttribute('inert', '');
      }
      if (tabbar) {
        tabbar.setAttribute('inert', '');
      }
      window.requestAnimationFrame(function () {
        closeButton.focus({ preventScroll: true });
      });
    }

    function closeViewer(settings) {
      if (!sheetOpen || !layer) {
        return;
      }

      sheetOpen = false;
      layer.classList.remove('is-open');
      layer.setAttribute('aria-hidden', 'true');
      layer.setAttribute('inert', '');
      if (app) {
        app.classList.remove('is-sheet-open');
      }
      if (main) {
        main.removeAttribute('inert');
      }
      if (tabbar) {
        tabbar.removeAttribute('inert');
      }

      if (!settings || settings.restoreFocus !== false) {
        var focusTarget = lastFocused instanceof HTMLElement ? lastFocused : null;
        if (focusTarget) {
          focusTarget.focus({ preventScroll: true });
        }
      }
    }

    if (app) {
      app.addEventListener('click', function (event) {
        if (!(event.target instanceof Element)) {
          return;
        }

        var tile = event.target.closest('.photo-tile');
        if (tile) {
          event.preventDefault();
          openViewer(tile);
          return;
        }

        if (event.target.closest('[data-close-sheet]')) {
          closeViewer();
        }
      });
    }

    document.addEventListener('keydown', function (event) {
      if (!sheetOpen) {
        return;
      }

      if (event.key === 'Escape') {
        event.preventDefault();
        closeViewer();
        return;
      }

      if (event.key === 'Tab' && closeButton) {
        event.preventDefault();
        closeButton.focus();
      }
    });

    if (initialState === 'photo' && main) {
      var firstTile = main.querySelector('.photo-tile');
      if (firstTile) {
        openViewer(firstTile);
      }
    }

    /* ----- like toggle ----- */

    Array.prototype.forEach.call(document.querySelectorAll('.like-btn'), function (button) {
      button.addEventListener('click', function () {
        var liked = button.getAttribute('aria-pressed') === 'true';
        var next = !liked;
        var countEl = button.querySelector('.like-count');
        var count = parseInt(countEl.textContent, 10) || 0;

        count = Math.max(0, count + (next ? 1 : -1));
        button.setAttribute('aria-pressed', next ? 'true' : 'false');
        countEl.textContent = String(count);
        button.setAttribute(
          'aria-label',
          (next ? '取消点赞' : '点赞') + '，当前 ' + count + ' 人喜欢'
        );
      });
    });
  });
})();
