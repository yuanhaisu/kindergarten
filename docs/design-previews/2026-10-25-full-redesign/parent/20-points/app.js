/* 积分与成就 · 页面交互
 * tab 切换、明细加载更多、排行榜/明细反馈、徽章详情底部面板、四态切换与重试。
 * 外壳状态由 ../../_shared/shell.js 写入 .mp-app[data-state]，本文件只跟随后更新文案。
 */
(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const tabs = Array.from(document.querySelectorAll('.tab'));
  const panels = Array.from(document.querySelectorAll('.panel'));
  const ledger = document.getElementById('ledger');
  const loadMore = document.getElementById('loadMore');
  const toast = document.getElementById('toast');
  const backButton = document.getElementById('backButton');
  const emptyTitle = document.getElementById('emptyTitle');
  const emptyDesc = document.getElementById('emptyDesc');
  const emptyAction = document.getElementById('emptyAction');
  const retryButton = document.getElementById('retryButton');
  const retryLabel = document.getElementById('retryLabel');
  const sheet = document.getElementById('badgeSheet');
  const sheetBackdrop = document.getElementById('sheetBackdrop');
  const sheetIcon = document.getElementById('sheetIcon');
  const sheetName = document.getElementById('sheetName');
  const sheetDesc = document.getElementById('sheetDesc');
  const sheetMeta = document.getElementById('sheetMeta');
  const sheetClose = document.getElementById('sheetClose');
  const sheetConfirm = document.getElementById('sheetConfirm');
  const badges = Array.from(document.querySelectorAll('.badge'));

  if (!tabs.length || !panels.length) return;

  const EMPTY_COPY = {
    points: {
      title: '还没有积分记录',
      desc: '完成任务、坚持出勤都会获得积分',
      action: '查看亲子任务'
    },
    rank: {
      title: '班级还没有排名',
      desc: '老师开启积分活动后会显示本班排行',
      action: '回到积分'
    },
    achievements: {
      title: '还没有获得徽章',
      desc: '老师授予徽章后会出现在这里',
      action: '查看成长记录'
    }
  };

  let currentTab = 'points';
  let toastTimer = 0;
  let retryTimer = 0;
  let lastFocused = null;

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastTimer = 0;
    }, 1800);
  }

  function applyEmptyCopy() {
    const copy = EMPTY_COPY[currentTab] || EMPTY_COPY.points;
    if (emptyTitle) emptyTitle.textContent = copy.title;
    if (emptyDesc) emptyDesc.textContent = copy.desc;
    if (emptyAction) emptyAction.textContent = copy.action;
  }

  function activateTab(name) {
    currentTab = name;
    tabs.forEach((tab) => {
      const on = tab.dataset.tab === name;
      tab.classList.toggle('is-active', on);
      tab.setAttribute('aria-selected', String(on));
    });
    panels.forEach((panel) => {
      panel.hidden = panel.dataset.panel !== name;
    });
    applyEmptyCopy();
  }

  tabs.forEach((tab) => {
    tab.addEventListener('click', () => {
      if (tab.dataset.tab === currentTab) return;
      activateTab(tab.dataset.tab || 'points');
    });
  });

  function setDemoState(state) {
    document.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
  }

  function resetRetry() {
    window.clearTimeout(retryTimer);
    retryTimer = 0;
    if (retryButton) retryButton.disabled = false;
    if (retryLabel) retryLabel.textContent = '重试';
  }

  if (retryButton) {
    retryButton.addEventListener('click', () => {
      if (retryButton.disabled) return;
      retryButton.disabled = true;
      if (retryLabel) retryLabel.textContent = '正在重试…';
      retryTimer = window.setTimeout(() => {
        app.dataset.state = 'normal';
        setDemoState('normal');
        resetRetry();
        showToast('已重新加载');
      }, 1000);
    });
  }

  document.addEventListener('mp:state', (event) => {
    const state = event.detail && event.detail.state ? event.detail.state : 'normal';
    if (state !== 'error') resetRetry();
    applyEmptyCopy();
  });

  if (loadMore) {
    loadMore.addEventListener('click', () => {
      if (loadMore.disabled) return;
      appendHistoryRows();
      loadMore.disabled = true;
      loadMore.textContent = '已经到底了';
      showToast('已加载更早记录');
    });
  }

  function appendHistoryRows() {
    if (!ledger) return;
    const rows = [
      { source: '任务打卡', amount: '+10', desc: '完成任务【连续阅读第 12 天】', time: '10月12日 09:10', balance: '1190' },
      { source: '成长日志被推荐', amount: '+10', desc: '成长记录《秋天的第一片叶子》被老师推荐', time: '10月8日 17:45', balance: '1180' }
    ];
    rows.forEach((row) => {
      const li = document.createElement('li');
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'ledger-row';
      button.dataset.source = row.source;
      button.dataset.amount = row.amount;
      button.innerHTML = [
        '<span class="ledger-icon is-earn" aria-hidden="true">',
        '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 6v12M6 12h12"/></svg>',
        '</span>',
        '<span class="ledger-main">',
        `<span class="ledger-title">${row.source}</span>`,
        `<span class="ledger-desc">${row.desc}</span>`,
        `<span class="ledger-time">${row.time}</span>`,
        '</span>',
        '<span class="ledger-side">',
        `<span class="ledger-amount is-plus">${row.amount}</span>`,
        `<span class="ledger-balance">余额 ${row.balance}</span>`,
        '</span>'
      ].join('');
      li.appendChild(button);
      ledger.appendChild(li);
    });
    bindRowFeedback();
  }

  function bindRowFeedback() {
    Array.from(document.querySelectorAll('.ledger-row')).forEach((row) => {
      if (row.dataset.bound === 'true') return;
      row.dataset.bound = 'true';
      row.addEventListener('click', () => {
        showToast(`${row.dataset.source || '积分记录'} · ${row.dataset.amount || ''} 积分`);
      });
    });
    Array.from(document.querySelectorAll('.rank-row')).forEach((row) => {
      if (row.dataset.bound === 'true') return;
      row.dataset.bound = 'true';
      row.addEventListener('click', () => {
        showToast(`${row.dataset.name || '同学'} · 本周 ${row.dataset.points || '0'} 积分`);
      });
    });
  }

  function openSheet(badge) {
    if (!sheet || !sheetBackdrop) return;
    const earned = badge.dataset.earned === 'true';
    const icon = badge.querySelector('svg');
    const name = badge.dataset.name || '成就徽章';
    const desc = badge.dataset.desc || '';

    if (sheetIcon) {
      sheetIcon.className = `sheet-icon ${earned ? 'is-earned' : 'is-locked'}`;
      sheetIcon.innerHTML = icon ? icon.outerHTML : '';
    }
    if (sheetName) sheetName.textContent = name;
    if (sheetDesc) sheetDesc.textContent = desc;
    if (sheetMeta) {
      sheetMeta.textContent = earned
        ? `${badge.dataset.date || ''}获得 · +${badge.dataset.points || '0'} 积分`
        : '尚未获得';
      sheetMeta.classList.toggle('is-locked', !earned);
    }

    lastFocused = badge;
    sheetBackdrop.hidden = false;
    sheet.hidden = false;
    if (sheetClose) sheetClose.focus();
  }

  function closeSheet() {
    if (!sheet || !sheetBackdrop) return;
    sheet.hidden = true;
    sheetBackdrop.hidden = true;
    if (lastFocused && typeof lastFocused.focus === 'function') {
      lastFocused.focus();
    }
    lastFocused = null;
  }

  badges.forEach((badge) => {
    badge.addEventListener('click', () => openSheet(badge));
  });

  if (sheetClose) sheetClose.addEventListener('click', closeSheet);
  if (sheetConfirm) sheetConfirm.addEventListener('click', closeSheet);
  if (sheetBackdrop) sheetBackdrop.addEventListener('click', closeSheet);
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && sheet && !sheet.hidden) closeSheet();
  });

  if (emptyAction) {
    emptyAction.addEventListener('click', () => {
      if (currentTab === 'rank') {
        activateTab('points');
        showToast('已回到积分');
        return;
      }
      if (currentTab === 'achievements') {
        showToast('打开成长记录');
        return;
      }
      showToast('打开亲子任务');
    });
  }

  if (backButton) {
    backButton.addEventListener('click', () => showToast('返回我的'));
  }

  bindRowFeedback();
  applyEmptyCopy();
})();
