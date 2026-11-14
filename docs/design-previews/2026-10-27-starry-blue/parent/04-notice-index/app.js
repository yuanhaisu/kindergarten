(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const filters = Array.from(document.querySelectorAll('.filter-pill'));
  const cards = Array.from(document.querySelectorAll('.notice-card'));
  const noticeList = document.getElementById('notice-list');
  const filterEmpty = document.getElementById('filter-empty');
  const viewAllBtn = document.getElementById('view-all-btn');
  const refreshBtn = document.getElementById('empty-refresh-btn');
  const retryBtn = document.getElementById('retry-btn');
  const toast = document.getElementById('toast');
  const statTotal = document.getElementById('stat-total');
  const statUnread = document.getElementById('stat-unread');
  const statToday = document.getElementById('stat-today');

  const NOTICE_CATEGORIES = ['homework', 'bring', 'health', 'other'];
  let selectedCategory = 'all';
  let toastTimer = 0;
  let retryTimer = 0;

  function currentState() {
    return app.getAttribute('data-state') || 'normal';
  }

  function matchesCategory(card) {
    const category = card.getAttribute('data-category');
    if (selectedCategory === 'all') return true;
    if (selectedCategory === 'notice') return NOTICE_CATEGORIES.includes(category);
    return category === selectedCategory;
  }

  function selectedCards() {
    return cards.filter(matchesCategory);
  }

  function setStats(total, unread, today) {
    if (statTotal) statTotal.textContent = String(total);
    if (statUnread) statUnread.textContent = String(unread);
    if (statToday) statToday.textContent = String(today);
  }

  function updateStats(list) {
    const unread = list.filter((card) => card.getAttribute('data-read') === 'false').length;
    const today = list.filter((card) => card.getAttribute('data-today') === 'true').length;
    setStats(list.length, unread, today);
  }

  function setFilterAvailability(available) {
    filters.forEach((button) => {
      button.disabled = !available;
    });
  }

  function resetRetryButton() {
    if (!retryBtn) return;
    window.clearTimeout(retryTimer);
    retryTimer = 0;
    retryBtn.disabled = false;
    retryBtn.textContent = '重试';
    retryBtn.classList.remove('is-busy');
  }

  function refreshNormalView() {
    const visible = selectedCards();
    const visibleSet = new Set(visible);

    cards.forEach((card) => {
      card.hidden = !visibleSet.has(card);
    });

    if (noticeList) noticeList.hidden = visible.length === 0;
    if (filterEmpty) filterEmpty.hidden = visible.length !== 0;
    updateStats(visible);
  }

  function selectCategory(category) {
    selectedCategory = category;

    filters.forEach((button) => {
      button.setAttribute('aria-pressed', String(button.getAttribute('data-category') === category));
    });

    if (currentState() === 'normal') {
      refreshNormalView();
    }
  }

  function updateState(state) {
    resetRetryButton();
    if (filterEmpty) filterEmpty.hidden = true;

    if (state === 'loading' || state === 'error') {
      setFilterAvailability(false);
      if (state === 'loading') return;
    } else {
      setFilterAvailability(true);
    }

    if (state === 'normal') {
      refreshNormalView();
      return;
    }

    if (state === 'empty') {
      setStats(0, 0, 0);
      return;
    }

    if (state === 'error') {
      setStats('—', '—', '—');
    }
  }

  function setAppState(state) {
    app.setAttribute('data-state', state);
    document.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === state));
    });
    updateState(state);
  }

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastTimer = 0;
    }, 2000);
  }

  filters.forEach((button) => {
    button.addEventListener('click', () => {
      if (button.disabled) return;
      selectCategory(button.getAttribute('data-category') || 'all');
    });
  });

  cards.forEach((card) => {
    card.addEventListener('click', () => {
      if (currentState() !== 'normal') return;

      if (card.getAttribute('data-read') === 'false') {
        card.setAttribute('data-read', 'true');
        refreshNormalView();
        showToast('已读 · 打开通知详情');
        return;
      }

      showToast('已打开通知详情');
    });
  });

  if (viewAllBtn) {
    viewAllBtn.addEventListener('click', () => {
      if (currentState() !== 'normal') return;
      selectCategory('all');
      const allButton = filters.find((button) => button.getAttribute('data-category') === 'all');
      if (allButton) allButton.focus({ preventScroll: true });
    });
  }

  if (refreshBtn) {
    refreshBtn.addEventListener('click', () => {
      if (currentState() !== 'empty') return;
      showToast('已重新加载');
    });
  }

  if (retryBtn) {
    retryBtn.addEventListener('click', () => {
      if (retryBtn.disabled || currentState() !== 'error') return;

      retryBtn.disabled = true;
      retryBtn.textContent = '重试中…';
      retryBtn.classList.add('is-busy');

      window.clearTimeout(retryTimer);
      retryTimer = window.setTimeout(() => {
        resetRetryButton();
        setAppState('normal');
        showToast('已重新加载');
      }, 1000);
    });
  }

  document.addEventListener('mp:state', (event) => {
    const state = event && event.detail && event.detail.state;
    updateState(state || currentState());
  });

  updateState(currentState());
})();
