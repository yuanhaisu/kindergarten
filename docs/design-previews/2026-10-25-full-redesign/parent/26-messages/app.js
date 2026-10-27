/* 消息 · 门口的信箱
 * 只做四件事：点信置读、口径行同步、轻提示、失败态重试。
 * 状态切换由共享外壳写 data-state 并派发 mp:state，这里只负责接住。
 */
(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const unreadLine = document.getElementById('unreadLine');
  const unreadText = document.getElementById('unreadText');
  const totalText = document.getElementById('totalText');
  const toast = document.getElementById('pageToast');
  const retryBtn = document.getElementById('retryBtn');
  const refreshBtn = document.getElementById('refreshBtn');
  const cards = Array.from(document.querySelectorAll('.msg-card'));

  const TYPE_NAMES = {
    notice: '通知',
    leave: '请假',
    pickup: '接送',
    growth: '成长',
    task: '任务',
    system: '系统'
  };

  let unreadCount = cards.filter((card) => !card.classList.contains('is-read')).length;
  let toastTimer = 0;
  let retryTimer = 0;

  if (totalText) {
    totalText.textContent = String(cards.length);
  }

  function renderUnread() {
    if (!unreadLine || !unreadText) return;
    const isClear = unreadCount <= 0;
    unreadLine.classList.toggle('is-clear', isClear);
    unreadText.textContent = isClear ? '没有未读消息' : unreadCount + ' 条未读';
  }

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function hideToast() {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toastTimer = 0;
    toast.classList.remove('is-visible');
  }

  function markRead(card) {
    if (card.classList.contains('is-read')) return false;
    card.classList.add('is-read');
    card.dataset.read = 'true';
    const unreadLabel = card.querySelector('.msg-unread-label');
    if (unreadLabel) unreadLabel.remove();
    unreadCount = Math.max(0, unreadCount - 1);
    renderUnread();
    return true;
  }

  cards.forEach((card) => {
    card.addEventListener('click', () => {
      // 失败态不锁列表：源码刷新失败只是多一条 inline-error，卡片仍可点开
      const typeName = TYPE_NAMES[card.dataset.type] || '消息';
      const jumpable = String(card.dataset.action || '').startsWith('/pages/');
      const wasRead = card.classList.contains('is-read');

      if (!wasRead) {
        markRead(card);
        showToast(jumpable
          ? '已标记为已读 · 打开' + typeName + '详情'
          : '已标记为已读');
      } else {
        showToast(jumpable ? '打开' + typeName + '详情' : '消息已读');
      }
    });
  });

  function resetRetry() {
    if (!retryBtn) return;
    retryBtn.disabled = false;
    retryBtn.textContent = '重试';
  }

  function syncChips(state) {
    document.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
  }

  function setState(state) {
    app.dataset.state = state;
    syncChips(state);
  }

  function stopRetryTimer() {
    if (!retryTimer) return;
    window.clearTimeout(retryTimer);
    retryTimer = 0;
  }

  function applyState() {
    stopRetryTimer();
    hideToast();
    resetRetry();
  }

  document.addEventListener('mp:state', () => {
    applyState();
  });

  if (retryBtn) {
    retryBtn.addEventListener('click', () => {
      if (retryBtn.disabled) return;
      stopRetryTimer();
      retryBtn.disabled = true;
      retryBtn.textContent = '重试中…';
      retryTimer = window.setTimeout(() => {
        retryTimer = 0;
        setState('normal');
        applyState();
        showToast('已重新加载');
      }, 1000);
    });
  }

  if (refreshBtn) {
    refreshBtn.addEventListener('click', () => {
      showToast('已刷新');
    });
  }

  renderUnread();
})();
