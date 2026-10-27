(() => {
  const app = document.querySelector('.mp-app');
  if (!app) return;

  const tabs = Array.from(document.querySelectorAll('.seg-btn'));
  const reports = Array.from(document.querySelectorAll('.report'));
  const toast = document.getElementById('toast');
  const back = document.querySelector('.mp-back');
  const emptyCopy = document.querySelector('.state-empty .state-copy');
  const emptyAction = document.querySelector('.state-empty .state-btn');
  const retryAction = document.querySelector('.state-error .state-btn');
  let toastTimer = 0;
  // 空态只属于当前周期：出口指向下一个周期，避免在月报空态里再让用户「查看月报」
  const emptyExits = {
    weekly: { copy: '老师通常周五整理周报，先看看月报或学期报告', label: '查看月报', period: 'monthly', toast: '已切换到月报' },
    monthly: { copy: '这个月的报告还没整理好，先看看学期报告吧', label: '查看学期报告', period: 'term', toast: '已切换到学期报告' },
    term: { copy: '学期报告还在整理中，先回成长时光看看其他记录吧', label: '返回成长时光', period: null, toast: '返回成长时光' },
  };
  let currentPeriod = 'weekly';

  function setPeriod(period) {
    currentPeriod = period;
    reports.forEach((report) => {
      report.hidden = report.dataset.period !== period;
    });
    tabs.forEach((tab) => {
      tab.setAttribute('aria-pressed', String(tab.dataset.period === period));
    });
  }

  function setTabsDisabled(disabled) {
    tabs.forEach((tab) => {
      tab.disabled = disabled;
      tab.setAttribute('aria-disabled', String(disabled));
    });
  }

  function syncDemoChip(state) {
    document.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
  }

  function setAppState(state) {
    app.dataset.state = state;
    syncDemoChip(state);
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

  function returnToNormal(period, message) {
    setAppState('normal');
    setTabsDisabled(false);
    setPeriod(period);
    if (message) showToast(message);
  }

  tabs.forEach((tab) => {
    tab.addEventListener('click', () => {
      if (app.dataset.state === 'loading' || app.dataset.state === 'error') return;
      if (tab.getAttribute('aria-pressed') === 'true') return;
      setPeriod(tab.dataset.period);
      // 空态是「本周期没有报告」，切到另一周期就等于换了数据源，应回到正常态
      if (app.dataset.state === 'empty') setAppState('normal');
    });
  });

  if (back) {
    back.addEventListener('click', () => showToast('返回成长时光'));
  }

  function syncEmptyExit() {
    const exit = emptyExits[currentPeriod] || emptyExits.weekly;
    if (emptyCopy) emptyCopy.textContent = exit.copy;
    if (emptyAction) emptyAction.textContent = exit.label;
  }

  if (emptyAction) {
    emptyAction.addEventListener('click', () => {
      const exit = emptyExits[currentPeriod] || emptyExits.weekly;
      // 学期没有更长的周期可去，出口与返回按钮同口径，只提示不模拟路由
      if (exit.period) returnToNormal(exit.period, exit.toast);
      else showToast(exit.toast);
    });
  }

  if (retryAction) {
    retryAction.addEventListener('click', () => {
      // 失败期间分段禁用，currentPeriod 仍停在失败前选中的周期
      returnToNormal(currentPeriod, '已重新加载');
    });
  }

  document.addEventListener('mp:state', (event) => {
    const state = (event.detail && event.detail.state) || 'normal';
    const blocked = state === 'loading' || state === 'error';
    setTabsDisabled(blocked);
    if (state === 'normal') setPeriod('weekly');
    if (state === 'empty') syncEmptyExit();
  });

  setPeriod('weekly');
  setTabsDisabled(false);
})();
