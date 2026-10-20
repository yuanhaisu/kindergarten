(function () {
  'use strict';

  var body = document.body;
  var records = Array.prototype.slice.call(document.querySelectorAll('.record'));
  var months = Array.prototype.slice.call(document.querySelectorAll('.month'));
  var viewButtons = Array.prototype.slice.call(document.querySelectorAll('[data-view]'));
  var chips = Array.prototype.slice.call(document.querySelectorAll('.chip[data-kind]'));
  var timeline = document.getElementById('timeline');
  var emptyState = document.getElementById('emptyState');
  var emptyTitle = document.getElementById('emptyTitle');
  var emptyText = document.getElementById('emptyText');
  var emptyCta = document.getElementById('emptyCta');
  var errorState = document.getElementById('errorState');
  var retryBtn = document.getElementById('retryBtn');
  var viewHint = document.getElementById('viewHint');
  var feedSummary = document.getElementById('feedSummary');
  var toast = document.getElementById('toast');
  var moreBtn = document.getElementById('moreBtn');
  var moreMenu = document.getElementById('moreMenu');
  var exportBtn = document.getElementById('exportBtn');
  var engageTemplate = document.getElementById('engageTemplate');
  var toastTimer;
  var bootTimer;

  var state = { view: 'child', kind: 'all' };
  var copy = {
    child: {
      hint: '只看孩子：只显示与小明确认相关的内容',
      emptyTitle: '还没有小明的专属记录',
      emptyText: '换个分类看看，或切到「看全部」浏览班级动态。'
    },
    all: {
      hint: '看全部：包含班级其他孩子的动态',
      emptyTitle: '班级还没有发布成长记录',
      emptyText: '老师发布后，会第一时间出现在这里。'
    }
  };

  function mountEngagement() {
    if (!engageTemplate) return;
    records.forEach(function (record) {
      var card = record.querySelector('.record__card');
      if (!card) return;
      var fragment = engageTemplate.content.cloneNode(true);
      var liked = record.dataset.liked === 'true';
      var comments = Number(record.dataset.comments || 0);
      var likeButton = fragment.querySelector('[data-like]');
      var likeLabel = fragment.querySelector('[data-like-label]');
      var commentsText = fragment.querySelector('[data-comments-text]');
      if (liked) {
        likeButton.classList.add('is-liked');
        likeButton.setAttribute('aria-pressed', 'true');
        likeLabel.textContent = '已赞';
      }
      commentsText.textContent = comments > 0 ? comments + ' 条评论' : '评论';
      card.appendChild(fragment);
    });
  }

  mountEngagement();

  function photosOf(record) {
    return Array.prototype.slice.call(record.querySelectorAll('.photo'));
  }

  function photosForView(record, view) {
    var photos = photosOf(record);
    if (view === 'all' || record.dataset.scope === 'individual') return photos;
    return photos.filter(function (photo) {
      return photo.dataset.child === 'true';
    });
  }

  function recordIsInView(record, view) {
    if (view === 'all') return true;
    if (record.dataset.owner === 'other') return false;
    if (record.dataset.scope === 'individual') return true;
    return photosForView(record, view).length > 0;
  }

  function countForView(view) {
    return records.reduce(function (total, record) {
      if (!recordIsInView(record, view)) return total;
      total.records += 1;
      total.photos += photosForView(record, view).length;
      return total;
    }, { records: 0, photos: 0 });
  }

  function refreshViewCounts() {
    ['child', 'all'].forEach(function (view) {
      var output = document.querySelector('[data-count-view="' + view + '"]');
      var counts = countForView(view);
      if (output) output.textContent = counts.records + ' 条 · ' + counts.photos + ' 张';
    });
  }

  function updateOwnerBadge(record) {
    var badge = record.querySelector('[data-owner-badge]');
    if (!badge) return;
    var isPersonal = record.dataset.scope === 'individual' || record.dataset.owner === 'child';
    var fallback;
    if (isPersonal) {
      fallback = '专属记录';
    } else if (record.dataset.owner === 'other') {
      fallback = '其他孩子';
    } else {
      fallback = state.view === 'child' ? '集体 · 含小明' : '集体';
    }
    var preferred = state.view === 'child' ? badge.dataset.badgeChild : badge.dataset.badgeAll;
    badge.textContent = preferred || fallback;
    badge.dataset.tone = isPersonal ? 'personal' : (record.dataset.owner === 'other' ? 'other' : 'collective');
  }

  function updateControls() {
    viewButtons.forEach(function (button) {
      var active = button.dataset.view === state.view;
      button.classList.toggle('is-active', active);
      button.setAttribute('aria-pressed', String(active));
    });
    chips.forEach(function (chip) {
      var active = chip.dataset.kind === state.kind;
      chip.classList.toggle('is-active', active);
      chip.setAttribute('aria-pressed', String(active));
    });
    viewHint.textContent = copy[state.view].hint;
  }

  function applyFilters() {
    var visibleRecords = 0;
    var visiblePhotos = 0;

    records.forEach(function (record) {
      var inView = recordIsInView(record, state.view);
      var matchesKind = state.kind === 'all' || record.dataset.kind === state.kind;
      var show = inView && matchesKind;
      var allPhotos = photosOf(record);
      var shownPhotos = photosForView(record, state.view);
      var grid = record.querySelector('[data-photo-grid]');
      var note = record.querySelector('.record__note');
      var hiddenPhotoCount = allPhotos.length - shownPhotos.length;

      allPhotos.forEach(function (photo) {
        photo.hidden = shownPhotos.indexOf(photo) === -1;
      });
      if (grid) grid.dataset.count = String(Math.max(shownPhotos.length, 1));

      if (note) {
        note.hidden = !(show && hiddenPhotoCount > 0);
        note.textContent = hiddenPhotoCount > 0
          ? '还有 ' + hiddenPhotoCount + ' 张班级照片，切到「看全部」查看'
          : '';
      }

      record.hidden = !show;
      updateOwnerBadge(record);

      if (show) {
        visibleRecords += 1;
        visiblePhotos += shownPhotos.length;
      }
    });

    months.forEach(function (month) {
      var count = month.querySelectorAll('.record:not([hidden])').length;
      var label = month.querySelector('[data-month-count]');
      month.hidden = count === 0;
      if (label) label.textContent = count + ' 条';
    });

    updateControls();
    feedSummary.textContent = '当前显示 ' + visibleRecords + ' 条记录，' + visiblePhotos + ' 张照片。';

    if (visibleRecords === 0) {
      timeline.hidden = true;
      emptyState.hidden = false;
      emptyTitle.textContent = copy[state.view].emptyTitle;
      emptyText.textContent = copy[state.view].emptyText;
      emptyCta.hidden = state.view !== 'child';
    } else {
      timeline.hidden = false;
      emptyState.hidden = true;
    }
  }

  function showToast(message) {
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2400);
  }

  function showError() {
    window.clearTimeout(bootTimer);
    body.classList.remove('is-loading');
    timeline.hidden = true;
    emptyState.hidden = true;
    errorState.hidden = false;
  }

  function finishLoading() {
    try {
      body.classList.remove('is-loading');
      errorState.hidden = true;
      applyFilters();
    } catch (error) {
      showError();
    }
  }

  function retry() {
    errorState.hidden = true;
    body.classList.add('is-loading');
    window.clearTimeout(bootTimer);
    var reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    bootTimer = window.setTimeout(finishLoading, reduced ? 100 : 650);
  }

  viewButtons.forEach(function (button) {
    button.addEventListener('click', function () {
      if (state.view === button.dataset.view) return;
      state.view = button.dataset.view;
      applyFilters(true);
    });
  });

  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      if (state.kind === chip.dataset.kind) return;
      state.kind = chip.dataset.kind;
      applyFilters(true);
    });
  });

  emptyCta.addEventListener('click', function () {
    state.view = 'all';
    state.kind = 'all';
    applyFilters(true);
    var allButton = document.querySelector('[data-view="all"]');
    if (allButton) allButton.focus();
  });

  retryBtn.addEventListener('click', retry);

  document.querySelectorAll('[data-like]').forEach(function (button) {
    button.addEventListener('click', function () {
      var liked = button.getAttribute('aria-pressed') !== 'true';
      var label = button.querySelector('[data-like-label]');
      button.setAttribute('aria-pressed', String(liked));
      button.classList.toggle('is-liked', liked);
      if (label) label.textContent = liked ? '已赞' : '赞';
      showToast(liked ? '已点赞' : '已取消点赞');
    });
  });

  document.querySelectorAll('[data-share]').forEach(function (button) {
    button.addEventListener('click', async function () {
      var record = button.closest('.record');
      var teacher = record && record.querySelector('.record__teacher');
      var meta = record && record.querySelector('.record__meta');
      var text = '小明的成长时光 · ' + (teacher ? teacher.textContent : '老师记录');
      if (meta) text += ' · ' + meta.textContent.replace(/\s+/g, ' ').trim();
      var shareData = { title: '小明的成长时光', text: text, url: window.location.href };

      try {
        if (navigator.share) {
          await navigator.share(shareData);
        } else if (navigator.clipboard && navigator.clipboard.writeText) {
          await navigator.clipboard.writeText(text + ' ' + window.location.href);
          showToast('记录链接已复制');
        } else {
          showToast('分享功能暂不可用');
        }
      } catch (error) {
        if (!error || error.name !== 'AbortError') showToast('分享功能暂不可用');
      }
    });
  });

  exportBtn.addEventListener('click', function () {
    showToast('已开始整理小明的成长相册');
  });

  function closeMenu(returnFocus) {
    if (moreMenu.hidden) return;
    moreMenu.hidden = true;
    moreBtn.setAttribute('aria-expanded', 'false');
    if (returnFocus) moreBtn.focus();
  }

  moreBtn.addEventListener('click', function (event) {
    event.stopPropagation();
    if (moreMenu.hidden) {
      moreMenu.hidden = false;
      moreBtn.setAttribute('aria-expanded', 'true');
      var firstItem = moreMenu.querySelector('button');
      if (firstItem) firstItem.focus();
    } else {
      closeMenu(false);
    }
  });

  moreMenu.querySelectorAll('button[data-toast]').forEach(function (item) {
    item.addEventListener('click', function () {
      showToast(item.dataset.toast);
      closeMenu(true);
    });
  });

  document.addEventListener('click', function (event) {
    if (!moreMenu.hidden && !moreMenu.contains(event.target) && event.target !== moreBtn) {
      closeMenu(false);
    }
  });

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && !moreMenu.hidden) closeMenu(true);
  });

  document.querySelectorAll('[data-toast]').forEach(function (button) {
    if (button.closest('.more-menu')) return;
    button.addEventListener('click', function () {
      showToast(button.dataset.toast);
    });
  });

  refreshViewCounts();
  updateControls();

  var reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  bootTimer = window.setTimeout(finishLoading, reduceMotion ? 100 : 520);
})();
