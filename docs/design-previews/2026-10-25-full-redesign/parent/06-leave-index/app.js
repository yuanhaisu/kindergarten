(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  var tabs = Array.prototype.slice.call(document.querySelectorAll('.status-tab'));
  var cards = Array.prototype.slice.call(document.querySelectorAll('[data-leave-card]'));
  var filterEmpty = document.querySelector('[data-filter-empty]');
  var listFooter = document.querySelector('[data-list-footer]');
  var footerCount = document.querySelector('[data-footer-count]');
  var retryButton = document.querySelector('[data-action="retry"]');
  var retryLabel = retryButton ? retryButton.querySelector('[data-retry-label]') : null;
  var toast = document.getElementById('toast');

  var currentFilter = 'all';
  var retryTimer = 0;
  var toastTimer = 0;

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function countByStatus(status) {
    return cards.filter(function (card) {
      return card.dataset.status === status;
    }).length;
  }

  function updateStats(state) {
    var values = { all: '—', pending: '—', approved: '—' };

    if (state === 'empty') {
      values = { all: '0', pending: '0', approved: '0' };
    } else if (state === 'normal') {
      values = {
        all: String(cards.length),
        pending: String(countByStatus('pending')),
        approved: String(countByStatus('approved'))
      };
    }

    Object.keys(values).forEach(function (key) {
      var node = document.querySelector('[data-stat="' + key + '"]');
      if (node) node.textContent = values[key];
    });

    var statsCard = document.querySelector('.stats-card');
    if (statsCard) statsCard.setAttribute('aria-busy', state === 'loading' ? 'true' : 'false');
  }

  function applyFilter(filter) {
    currentFilter = filter;
    var visibleCount = 0;

    tabs.forEach(function (tab) {
      var selected = tab.dataset.filter === filter;
      tab.classList.toggle('is-active', selected);
      tab.setAttribute('aria-pressed', selected ? 'true' : 'false');
    });

    cards.forEach(function (card) {
      var match = filter === 'all' || card.dataset.status === filter;
      var item = card.closest('.leave-item');
      if (item) item.hidden = !match;
      if (match) visibleCount += 1;
    });

    if (filterEmpty) filterEmpty.hidden = visibleCount !== 0;
    if (listFooter) listFooter.hidden = visibleCount === 0;
    if (footerCount) footerCount.textContent = String(visibleCount);
  }

  function resetRetry() {
    window.clearTimeout(retryTimer);
    retryTimer = 0;
    if (!retryButton) return;
    retryButton.disabled = false;
    retryButton.classList.remove('is-busy');
    retryButton.setAttribute('aria-busy', 'false');
    if (retryLabel) retryLabel.textContent = '重试';
  }

  function applyState(state) {
    resetRetry();
    updateStats(state);

    var isLoading = state === 'loading';
    tabs.forEach(function (tab) {
      tab.disabled = isLoading;
    });

    if (state === 'normal') {
      applyFilter('all');
    }
  }

  // 重试恢复时同步演示芯片选中态，避免 aria-pressed 与实际状态脱节。
  function setAppState(state) {
    app.dataset.state = state;
    document.querySelectorAll('.mp-demo-chip').forEach(function (chip) {
      chip.setAttribute('aria-pressed', chip.dataset.mpState === state ? 'true' : 'false');
    });
    applyState(state);
  }

  tabs.forEach(function (tab) {
    tab.addEventListener('click', function () {
      if (tab.disabled) return;
      applyFilter(tab.dataset.filter);
    });
  });

  cards.forEach(function (card) {
    card.addEventListener('click', function () {
      showToast('已打开请假详情');
    });
  });

  document.querySelectorAll('[data-action="create-leave"]').forEach(function (button) {
    button.addEventListener('click', function () {
      showToast('已打开发起请假');
    });
  });

  var viewAll = document.querySelector('[data-action="view-all"]');
  if (viewAll) {
    viewAll.addEventListener('click', function () {
      applyFilter('all');
      var allTab = tabs.filter(function (tab) {
        return tab.dataset.filter === 'all';
      })[0];
      if (allTab) allTab.focus();
    });
  }

  if (retryButton) {
    retryButton.addEventListener('click', function () {
      if (retryButton.disabled) return;

      retryButton.disabled = true;
      retryButton.classList.add('is-busy');
      retryButton.setAttribute('aria-busy', 'true');
      if (retryLabel) retryLabel.textContent = '重试中…';

      window.clearTimeout(retryTimer);
      retryTimer = window.setTimeout(function () {
        retryTimer = 0;
        // 模拟请求成功：回到正常态并恢复列表/统计，筛选沿用重试前的选择。
        var resumeFilter = currentFilter;
        setAppState('normal');
        if (resumeFilter !== 'all') applyFilter(resumeFilter);
        showToast('已重新加载');
      }, 1000);
    });
  }

  document.addEventListener('mp:state', function (event) {
    var nextState = event && event.detail && event.detail.state
      ? event.detail.state
      : app.dataset.state;

    applyState(nextState || 'normal');
  });

  // 首次渲染与外壳当前状态保持同步。
  applyState(app.dataset.state || 'normal');
  applyFilter(currentFilter);
})();
