(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  var toastEl = document.querySelector('.toast');
  var viewer = document.getElementById('album-viewer');
  var viewerSheet = viewer ? viewer.querySelector('.viewer-sheet') : null;
  var viewerPhoto = document.getElementById('viewer-photo');
  var viewerTitle = document.getElementById('viewer-title');
  var viewerDate = document.getElementById('viewer-date');
  var viewerIndex = document.getElementById('viewer-index');
  var viewerTotal = document.getElementById('viewer-total');
  var prevBtn = document.getElementById('viewer-prev');
  var nextBtn = document.getElementById('viewer-next');
  var chips = Array.prototype.slice.call(document.querySelectorAll('.mp-demo-chip'));
  var filters = Array.prototype.slice.call(document.querySelectorAll('.filter-chip'));
  var moreGroups = Array.prototype.slice.call(document.querySelectorAll('.month-group[data-more]'));
  var moreBar = document.querySelector('.album-more');
  var moreBtn = document.querySelector('[data-action="show-all"]');
  var expanded = false;
  var activeCategory = '全部';
  var toastTimer = 0;
  var closeTimer = 0;
  var viewerState = null;
  var returnFocus = null;
  var depth = [
    { size: '112% 112%', pos: '46% 32%' },
    { size: '168% 168%', pos: '78% 18%' },
    { size: '142% 142%', pos: '22% 82%' }
  ];

  function showToast(message) {
    if (!toastEl) return;
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toastEl.classList.remove('is-visible');
    }, 1800);
  }

  function syncChips(state) {
    chips.forEach(function (chip) {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === state));
    });
  }

  function applyState(state) {
    if (!app) return;
    app.setAttribute('data-state', state);
    syncChips(state);
    closeViewer(true);
    window.scrollTo(0, 0);
    window.clearTimeout(toastTimer);
    if (toastEl) toastEl.classList.remove('is-visible');
  }

  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      applyState(chip.getAttribute('data-mp-state'));
    });
  });

  function applyFilter(category) {
    activeCategory = category;
    Array.prototype.forEach.call(document.querySelectorAll('.album-card'), function (card) {
      var show = category === '全部' || card.getAttribute('data-category') === category;
      card.hidden = !show;
    });
    Array.prototype.forEach.call(document.querySelectorAll('.month-group'), function (group) {
      var count = group.querySelectorAll('.album-card:not([hidden])').length;
      // 更早月份默认折叠：只有展开后（或按分类筛选跨全部相册时）才可见。
      var collapsed = !expanded && group.hasAttribute('data-more');
      group.hidden = count === 0 || collapsed;
      var counter = group.querySelector('[data-month-count]');
      if (counter) counter.textContent = count + ' 册';
    });
    if (moreBar) moreBar.hidden = expanded || !moreGroups.length;
  }

  /* 展开更早月份，让「相册数 12」里的另外 6 册真正可达。 */
  function expandEarlier(withFocus) {
    if (expanded) return;
    expanded = true;
    applyFilter(activeCategory);
    if (!withFocus) return;
    var first = moreGroups[0];
    var title = first ? first.querySelector('.month-title') : null;
    if (title) {
      title.setAttribute('tabindex', '-1');
      try { title.focus({ preventScroll: true }); } catch (error) {}
    }
  }

  if (moreBtn) {
    moreBtn.addEventListener('click', function () { expandEarlier(true); });
  }

  filters.forEach(function (button) {
    button.addEventListener('click', function () {
      var category = button.getAttribute('data-category');
      filters.forEach(function (item) {
        var active = item === button;
        item.classList.toggle('is-active', active);
        item.setAttribute('aria-pressed', String(active));
      });
      // 分类筛选覆盖全部相册（含更早月份），先展开再算册数，避免筛选结果被折叠吞掉。
      if (category !== '全部') expandEarlier(false);
      applyFilter(category);
    });
  });

  Array.prototype.forEach.call(document.querySelectorAll('[data-action="upload"]'), function (button) {
    button.addEventListener('click', function () {
      showToast('上传功能开发中');
    });
  });

  function bookClass(card) {
    var tile = card.querySelector('.album-photo');
    if (!tile) return '';
    for (var i = 0; i < tile.classList.length; i += 1) {
      if (tile.classList[i].indexOf('ph-') === 0) return tile.classList[i];
    }
    return '';
  }

  function cardTitle(card) {
    var el = card.querySelector('.album-name');
    return el ? el.textContent.trim() : '';
  }

  function cardDate(card) {
    var el = card.querySelector('.album-date');
    return el ? el.textContent.trim() : '';
  }

  function photoTotal(card) {
    var el = card.querySelector('.album-count');
    var total = el ? parseInt(el.textContent, 10) : 1;
    return total > 0 ? total : 1;
  }

  function paintViewer() {
    if (!viewerState || !viewerPhoto) return;
    var card = viewerState.card;
    var total = photoTotal(card);
    var index = viewerState.index;
    var book = bookClass(card);
    ['ph-1', 'ph-2', 'ph-3', 'ph-4', 'ph-5', 'ph-6'].forEach(function (name) {
      viewerPhoto.classList.remove(name);
    });
    if (book) viewerPhoto.classList.add(book);
    var tone = depth[index % depth.length];
    viewerPhoto.style.backgroundSize = tone.size;
    viewerPhoto.style.backgroundPosition = tone.pos;
    if (viewerIndex) viewerIndex.textContent = String(index + 1);
    if (viewerTotal) viewerTotal.textContent = String(total);
    if (prevBtn) prevBtn.disabled = index <= 0;
    if (nextBtn) nextBtn.disabled = index >= total - 1;
  }

  function openViewer(card, index, trigger) {
    if (!viewer || !viewerTitle || !viewerDate) return;
    window.clearTimeout(closeTimer);
    closeTimer = 0;
    returnFocus = trigger || null;
    viewerState = { card: card, index: index };
    viewerTitle.textContent = cardTitle(card);
    viewerDate.textContent = cardDate(card);
    paintViewer();
    viewer.hidden = false;
    document.body.classList.add('is-locked');
    void viewer.offsetWidth;
    viewer.classList.add('is-open');
    var closeButton = viewer.querySelector('.viewer-close');
    window.setTimeout(function () {
      if (!viewer.hidden && closeButton) {
        try { closeButton.focus({ preventScroll: true }); } catch (error) {}
      }
    }, 30);
  }

  function closeViewer(immediate) {
    if (!viewer || viewer.hidden) return;
    if (closeTimer) return;
    viewer.classList.remove('is-open');
    document.body.classList.remove('is-locked');
    function finish() {
      closeTimer = 0;
      viewer.hidden = true;
      viewerState = null;
      if (returnFocus && document.contains(returnFocus)) {
        try { returnFocus.focus({ preventScroll: true }); } catch (error) {}
      }
      returnFocus = null;
    }
    if (immediate) {
      finish();
      return;
    }
    closeTimer = window.setTimeout(finish, 300);
  }

  function stepViewer(delta) {
    if (!viewerState || !viewer || viewer.hidden) return;
    var total = photoTotal(viewerState.card);
    var next = Math.max(0, Math.min(total - 1, viewerState.index + delta));
    if (next === viewerState.index) return;
    viewerState.index = next;
    paintViewer();
  }

  Array.prototype.forEach.call(document.querySelectorAll('.album-card'), function (card) {
    var title = cardTitle(card);
    var info = card.querySelector('.album-info');
    if (info) {
      info.addEventListener('click', function () {
        showToast('查看相册「' + title + '」');
      });
    }
    Array.prototype.forEach.call(card.querySelectorAll('.album-photo'), function (tile, index) {
      tile.addEventListener('click', function () {
        openViewer(card, index, tile);
      });
    });
  });

  if (viewer) {
    Array.prototype.forEach.call(viewer.querySelectorAll('[data-viewer-close]'), function (button) {
      button.addEventListener('click', function () { closeViewer(false); });
    });
    if (prevBtn) prevBtn.addEventListener('click', function () { stepViewer(-1); });
    if (nextBtn) nextBtn.addEventListener('click', function () { stepViewer(1); });
  }

  document.addEventListener('keydown', function (event) {
    if (!viewer || viewer.hidden) return;
    if (event.key === 'Escape') {
      event.preventDefault();
      closeViewer(false);
      return;
    }
    if (event.key === 'ArrowLeft') {
      event.preventDefault();
      stepViewer(-1);
      return;
    }
    if (event.key === 'ArrowRight') {
      event.preventDefault();
      stepViewer(1);
      return;
    }
    if (event.key === 'Tab' && viewerSheet) {
      var focusables = viewerSheet.querySelectorAll('button:not([disabled])');
      if (!focusables.length) return;
      var first = focusables[0];
      var last = focusables[focusables.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    }
  });

  var retry = document.querySelector('[data-action="retry"]');
  if (retry) {
    retry.addEventListener('click', function () {
      if (retry.disabled) return;
      retry.disabled = true;
      applyState('loading');
      window.setTimeout(function () {
        retry.disabled = false;
        if (app && app.getAttribute('data-state') === 'loading') applyState('normal');
      }, 600);
    });
  }

  applyFilter('全部');
  syncChips(app ? app.getAttribute('data-state') || 'normal' : 'normal');
})();
