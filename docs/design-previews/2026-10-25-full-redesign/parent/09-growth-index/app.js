/* 养成星球 · 家长端成长时光（原型交互）
 * 页面由 OpenDesign 产出结构，app.js 为手写补齐：
 * 视角切换（只看孩子 / 看全部）、点赞、轻提示、重试与四态联动。
 * 外壳（状态栏/导航栏/tabBar/演示条）由 ../../_shared/shell.js 负责。
 */
(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  var page = document.querySelector('.page');
  var toastEl = document.querySelector('.toast');
  var toastText = toastEl ? toastEl.querySelector('.toast__text') : null;
  var toastTimer = null;

  function showToast(message) {
    if (!toastEl || !toastText || !message) return;
    toastText.textContent = message;
    toastEl.classList.add('is-shown');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toastEl.classList.remove('is-shown');
    }, 2000);
  }

  /* 记录与照片的可见性规则 ------------------------------------------------ */
  var records = Array.prototype.slice.call(document.querySelectorAll('.rec'));

  function photosOf(record) {
    return Array.prototype.slice.call(record.querySelectorAll('.photo'));
  }

  function recordVisible(record, view) {
    if (view === 'all') return true;
    if (record.getAttribute('data-scope') === 'individual') return true;
    var photos = photosOf(record);
    if (photos.length === 0) return true; // 纯文字班级记录默认保留
    return photos.some(function (photo) {
      return photo.getAttribute('data-tagged') === 'true';
    });
  }

  function photoVisible(record, photo, view) {
    if (view === 'all') return true;
    if (record.getAttribute('data-scope') === 'individual') return true;
    return photo.getAttribute('data-tagged') === 'true';
  }

  function countVisible(view) {
    var recCount = 0;
    var photoCount = 0;
    records.forEach(function (record) {
      if (!recordVisible(record, view)) return;
      recCount += 1;
      photosOf(record).forEach(function (photo) {
        if (photoVisible(record, photo, view)) photoCount += 1;
      });
    });
    return { records: recCount, photos: photoCount };
  }

  /* 分段控件 -------------------------------------------------------------- */
  var segButtons = Array.prototype.slice.call(document.querySelectorAll('.seg-btn'));
  var segHint = document.querySelector('.seg-hint');
  var timelineCount = document.querySelector('[data-timeline-count]');
  var monthsEl = document.querySelector('.months');
  var modeEmptyEl = document.querySelector('.mode-empty');
  var modeEmptyTitleEl = document.getElementById('modeEmptyTitle');
  var modeEmptyTextEl = document.getElementById('modeEmptyText');
  var childNameEl = document.getElementById('childName');
  var childName = childNameEl ? childNameEl.textContent.trim() || '孩子' : '孩子';

  var segMeta = {
    child: document.querySelector('.seg-btn[data-view="child"] .seg-btn__meta'),
    all: document.querySelector('.seg-btn[data-view="all"] .seg-btn__meta')
  };

  var childTotals = countVisible('child');
  var allTotals = countVisible('all');
  if (segMeta.child) segMeta.child.textContent = childTotals.records + ' 条 · ' + childTotals.photos + ' 张';
  if (segMeta.all) segMeta.all.textContent = allTotals.records + ' 条 · ' + allTotals.photos + ' 张';

  var hints = {
    child: '只显示与' + childName + '确认相关的内容；切换「看全部」可浏览班级动态',
    all: '包含班级其他孩子的动态；切换「只看孩子」可聚焦当前孩子'
  };

  /* 视角筛选后为空（feeds 非空但当前视角 0 条）：标题与说明随视角变化，
   * 与源码 mode-empty 一致；「看全部」入口仍由顶部常驻切换承担，不重复。 */
  var modeEmptyCopy = {
    child: {
      title: '还没有与' + childName + '相关的成长记录',
      text: '可以先看看班级动态，或等老师发布新的记录。'
    },
    all: {
      title: '班级还没有发布成长记录',
      text: '老师发布后会出现在这里。'
    }
  };

  function applyView(view) {
    var recCount = 0;
    var photoCount = 0;

    records.forEach(function (record) {
      var isVisible = recordVisible(record, view);
      record.hidden = !isVisible;
      if (isVisible) recCount += 1;

      var photos = photosOf(record);
      var shown = 0;

      if (!isVisible) {
        photos.forEach(function (photo) { photo.hidden = true; });
      } else {
        photos.forEach(function (photo) {
          var visible = photoVisible(record, photo, view);
          photo.hidden = !visible;
          if (visible) shown += 1;
        });
        photoCount += shown;
      }

      var grid = record.querySelector('.photo-grid');
      if (grid && photos.length) grid.setAttribute('data-count', String(shown));

      var more = record.querySelector('.rec-more');
      if (more) {
        var hiddenCount = photos.length - shown;
        var showMore = isVisible && view === 'child'
          && record.getAttribute('data-scope') !== 'individual'
          && hiddenCount > 0;
        more.hidden = !showMore;
        if (showMore) {
          var moreText = more.querySelector('.rec-more__text');
          if (moreText) {
            moreText.textContent = '还有 ' + hiddenCount + ' 张班级照片，切到「看全部」查看';
          }
        }
      }
    });

    document.querySelectorAll('.month-group').forEach(function (group) {
      group.hidden = !group.querySelector('.rec:not([hidden])');
    });

    // 视角筛选后为空：保留时光轴标题与「0 条」，用标题 + 说明代替空白列表。
    // 全局空态（一条记录都没有）仍由 data-state="empty" 承担，两者互斥。
    var viewEmpty = records.length > 0 && recCount === 0;
    if (monthsEl) monthsEl.hidden = viewEmpty;
    if (modeEmptyEl) modeEmptyEl.hidden = !viewEmpty;
    if (viewEmpty) {
      var copy = modeEmptyCopy[view] || modeEmptyCopy.child;
      if (modeEmptyTitleEl) modeEmptyTitleEl.textContent = copy.title;
      if (modeEmptyTextEl) modeEmptyTextEl.textContent = copy.text;
    }

    segButtons.forEach(function (button) {
      var active = button.getAttribute('data-view') === view;
      button.classList.toggle('is-active', active);
      button.setAttribute('aria-pressed', String(active));
    });

    if (segHint) segHint.textContent = hints[view];
    if (timelineCount) timelineCount.textContent = recCount + ' 条';

    app.setAttribute('data-view', view);
  }

  segButtons.forEach(function (button) {
    button.addEventListener('click', function () {
      if (button.getAttribute('aria-disabled') === 'true') return;
      if (button.classList.contains('is-active')) return;
      applyView(button.getAttribute('data-view') === 'all' ? 'all' : 'child');
    });
  });

  /* 点赞 ------------------------------------------------------------------ */
  Array.prototype.forEach.call(document.querySelectorAll('.js-like'), function (button) {
    button.addEventListener('click', function (event) {
      event.stopPropagation();
      if (button.getAttribute('aria-disabled') === 'true') return;
      var liked = button.getAttribute('aria-pressed') === 'true';
      var next = !liked;
      button.setAttribute('aria-pressed', String(next));
      button.classList.toggle('is-liked', next);
      var label = button.querySelector('.like-label');
      if (label) label.textContent = next ? '已点赞' : '点赞';
      showToast(next ? '已点赞' : '已取消点赞');
    });
  });

  /* 评论 / 分享 / 导出 ---------------------------------------------------- */
  function bindToastAction(selector, message) {
    Array.prototype.forEach.call(document.querySelectorAll(selector), function (button) {
      button.addEventListener('click', function (event) {
        event.stopPropagation();
        if (button.getAttribute('aria-disabled') === 'true') return;
        showToast(message);
      });
    });
  }

  bindToastAction('.js-comment', '已打开成长详情');
  bindToastAction('.js-share', '已唤起微信分享');
  bindToastAction('.js-export', '已打开成长相册');

  /* 卡片点击 -------------------------------------------------------------- */
  records.forEach(function (record) {
    record.addEventListener('click', function (event) {
      if (event.target.closest('button')) return;
      showToast('已打开成长详情');
    });
    record.addEventListener('keydown', function (event) {
      if (event.key !== 'Enter' && event.key !== ' ') return;
      event.preventDefault();
      showToast('已打开成长详情');
    });
  });

  /* 演示状态联动 ---------------------------------------------------------- */
  var retryButton = document.querySelector('.js-retry');
  var refreshButton = document.querySelector('.js-refresh');
  var retryTimer = null;

  function interactiveControls() {
    return document.querySelectorAll('.seg-btn, .js-export, .js-like, .js-comment, .js-share');
  }

  function resetRetry() {
    window.clearTimeout(retryTimer);
    if (!retryButton) return;
    retryButton.disabled = false;
    retryButton.textContent = '重试';
  }

  function syncState(state) {
    var locked = state === 'loading' || state === 'error';
    Array.prototype.forEach.call(interactiveControls(), function (control) {
      control.setAttribute('aria-disabled', String(locked));
    });
    if (page) page.setAttribute('aria-busy', String(state === 'loading'));
    if (state !== 'error') resetRetry();
  }

  function setDemoState(state) {
    app.setAttribute('data-state', state);
    Array.prototype.forEach.call(document.querySelectorAll('.mp-demo-chip'), function (chip) {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === state));
    });
    syncState(state);
  }

  document.addEventListener('mp:state', function (event) {
    syncState(event && event.detail ? event.detail.state : 'normal');
  });

  if (retryButton) {
    retryButton.addEventListener('click', function () {
      if (retryButton.disabled) return;
      retryButton.disabled = true;
      retryButton.textContent = '重试中…';
      window.clearTimeout(retryTimer);
      retryTimer = window.setTimeout(function () {
        resetRetry();
        setDemoState('normal');
        showToast('已重新加载');
      }, 900);
    });
  }

  if (refreshButton) {
    refreshButton.addEventListener('click', function () {
      setDemoState('normal');
      showToast('已刷新');
    });
  }

  /* 初始化 ---------------------------------------------------------------- */
  applyView('child');
  syncState(app.getAttribute('data-state') || 'normal');
})();
