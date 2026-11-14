(function () {
  'use strict';

  var params = new URLSearchParams(window.location.search);
  var app = document.querySelector('.mp-app');
  var defaultView = document.getElementById('default-view');
  var emptyView = document.getElementById('empty-view');

  if (params.get('state') === 'empty') {
    app.classList.add('is-empty');
    defaultView.hidden = true;
    emptyView.hidden = false;
  }

  var chips = Array.prototype.slice.call(document.querySelectorAll('.filter-chip'));
  var cards = Array.prototype.slice.call(document.querySelectorAll('.record-card'));
  var list = document.querySelector('.record-list');
  var filterEmpty = document.getElementById('filter-empty');
  var filterEmptyText = document.getElementById('filter-empty-text');
  var filterReset = document.getElementById('filter-reset');
  var footer = document.getElementById('list-footer');

  function applyFilter(value) {
    var visibleCount = 0;
    var activeLabel = '全部';

    cards.forEach(function (card) {
      var matches = value === 'all' || card.dataset.status === value;
      card.hidden = !matches;
      if (matches) {
        visibleCount += 1;
      }
    });

    chips.forEach(function (chip) {
      var isActive = chip.dataset.filter === value;
      chip.setAttribute('aria-pressed', isActive ? 'true' : 'false');
      if (isActive) {
        activeLabel = chip.textContent.trim();
      }
    });

    var hasNoResults = visibleCount === 0;
    list.hidden = hasNoResults;
    filterEmpty.hidden = !hasNoResults;
    footer.textContent = '共 ' + visibleCount + ' 条请假记录';

    if (hasNoResults) {
      filterEmptyText.textContent =
        value === 'all' ? '还没有请假记录' : '没有' + activeLabel + '的请假记录';
    }
  }

  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      applyFilter(chip.dataset.filter);
    });
  });

  filterReset.addEventListener('click', function () {
    applyFilter('all');
    var allChip = chips.filter(function (chip) {
      return chip.dataset.filter === 'all';
    })[0];
    if (allChip) {
      allChip.focus();
    }
  });

  var layer = document.getElementById('detail-layer');
  var scrim = document.getElementById('detail-scrim');
  var closeButton = document.getElementById('detail-close');
  var detailType = document.getElementById('detail-type');
  var detailTitle = document.getElementById('detail-title');
  var detailDate = document.getElementById('detail-date');
  var detailReason = document.getElementById('detail-reason');
  var detailSubmitted = document.getElementById('detail-submitted');
  var detailStatus = document.getElementById('detail-status');
  var lastFocused = null;
  var isSheetOpen = false;
  var closeTimer = null;

  function statusClass(status) {
    if (status === 'pending') {
      return 'tag tag--pending';
    }
    if (status === 'approved') {
      return 'tag tag--approved';
    }
    return 'tag';
  }

  function openDetail(card) {
    if (closeTimer) {
      window.clearTimeout(closeTimer);
      closeTimer = null;
    }

    lastFocused = card;
    detailType.textContent = card.dataset.type || '';
    detailTitle.textContent = card.dataset.title || '';
    detailDate.textContent = card.dataset.date || '';
    detailReason.textContent = card.dataset.reason || '';
    detailSubmitted.textContent = card.dataset.submitted || '';
    detailStatus.textContent = card.dataset.statusLabel || '';
    detailStatus.className = statusClass(card.dataset.status || '');

    layer.classList.remove('is-closing');
    layer.hidden = false;
    isSheetOpen = true;

    window.requestAnimationFrame(function () {
      layer.classList.add('is-open');
      closeButton.focus({ preventScroll: true });
    });
  }

  function closeDetail() {
    if (!isSheetOpen) {
      return;
    }

    isSheetOpen = false;
    layer.classList.add('is-closing');
    layer.classList.remove('is-open');

    closeTimer = window.setTimeout(function () {
      layer.hidden = true;
      layer.classList.remove('is-closing');
      closeTimer = null;
      if (lastFocused && document.contains(lastFocused)) {
        lastFocused.focus({ preventScroll: true });
      }
    }, 200);
  }

  cards.forEach(function (card) {
    card.addEventListener('click', function () {
      openDetail(card);
    });

    card.addEventListener('keydown', function (event) {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        openDetail(card);
      }
    });
  });

  scrim.addEventListener('click', closeDetail);
  closeButton.addEventListener('click', closeDetail);

  document.addEventListener('keydown', function (event) {
    if (!isSheetOpen) {
      return;
    }

    if (event.key === 'Escape') {
      event.preventDefault();
      closeDetail();
    }

    if (event.key === 'Tab') {
      event.preventDefault();
      closeButton.focus({ preventScroll: true });
    }
  });
})();
