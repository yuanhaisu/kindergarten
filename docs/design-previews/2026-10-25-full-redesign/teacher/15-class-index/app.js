/* 我的班级 · 原型交互
 * 依赖 ../../_shared/shell.js 提供的四态切换（mp:state）。
 */
(() => {
  const app = document.querySelector('.mp-app');
  if (!app) return;

  const grid = document.getElementById('studentGrid');
  const kids = grid ? Array.from(grid.querySelectorAll('.kid')) : [];
  const filters = Array.from(document.querySelectorAll('.filter'));
  const search = document.getElementById('studentSearch');
  const noResults = document.getElementById('noResults');
  const clearFilters = document.getElementById('clearFilters');
  const roster = document.getElementById('roster');
  const toolbar = document.getElementById('toolbar');

  const sheet = document.getElementById('sheetLayer');
  const sheetClose = document.getElementById('sheetClose');
  const sheetAvatar = document.getElementById('sheetAvatar');
  const sheetName = document.getElementById('sheetName');
  const sheetPill = document.getElementById('sheetPill');
  const sheetPillText = document.getElementById('sheetPillText');
  const sheetTime = document.getElementById('sheetTime');
  const sheetStatus = document.getElementById('sheetStatus');
  const sheetPersons = document.getElementById('sheetPersons');
  const sheetDeadline = document.getElementById('sheetDeadline');

  const toastEl = document.getElementById('toast');
  const bellBtn = document.getElementById('bellBtn');
  const backBtn = document.querySelector('.mp-back');
  // 接送截止是班级级单一值（后端每个学生返回同一 deadline），不按人取值
  const CLASS_DEADLINE = app.getAttribute('data-pickup-deadline') || '—';
  const emptyRefresh = document.getElementById('emptyRefresh');
  const retryBtn = document.getElementById('retryBtn');

  const STATUS_TEXT = {
    picked_up: '已接走',
    not_picked_up: '未接走',
    not_arrived: '未入园',
  };

  let activeFilter = 'all';
  let lastTrigger = null;
  let toastTimer = null;
  let stateTimer = null;

  /* 筛选 + 搜索 ------------------------------------------------------- */
  function applyFilters() {
    if (!kids.length) return;
    const keyword = (search ? search.value : '').trim();

    let visible = 0;
    kids.forEach((kid) => {
      const statusOk = activeFilter === 'all' || kid.getAttribute('data-status') === activeFilter;
      const name = kid.getAttribute('data-name') || '';
      const nameOk = !keyword || name.indexOf(keyword) !== -1;
      const show = statusOk && nameOk;
      kid.hidden = !show;
      if (show) visible += 1;
    });

    if (noResults) {
      noResults.setAttribute('data-visible', visible === 0 ? 'true' : 'false');
    }
    if (roster) roster.setAttribute('data-filtered', activeFilter);
  }

  filters.forEach((button) => {
    button.addEventListener('click', () => {
      activeFilter = button.getAttribute('data-filter') || 'all';
      filters.forEach((item) => {
        item.setAttribute('aria-pressed', String(item === button));
      });
      applyFilters();
    });
  });

  if (search) {
    search.addEventListener('input', applyFilters);
  }

  if (clearFilters) {
    clearFilters.addEventListener('click', () => {
      activeFilter = 'all';
      filters.forEach((item) => {
        item.setAttribute('aria-pressed', String(item.getAttribute('data-filter') === 'all'));
      });
      if (search) {
        search.value = '';
        search.focus();
      }
      applyFilters();
    });
  }

  /* 学生简要信息卡 ----------------------------------------------------- */
  function openSheet(kid) {
    if (!sheet) return;
    const name = kid.getAttribute('data-name') || '';
    const status = kid.getAttribute('data-status') || 'not_picked_up';
    const statusText = STATUS_TEXT[status] || '未接走';

    if (sheetAvatar) sheetAvatar.textContent = name.slice(0, 1) || '?';
    if (sheetName) sheetName.textContent = name;
    if (sheetPill) sheetPill.setAttribute('data-status', status);
    if (sheetPillText) sheetPillText.textContent = statusText;
    if (sheetTime) sheetTime.textContent = kid.getAttribute('data-time') || '—';
    if (sheetStatus) sheetStatus.textContent = statusText;
    if (sheetPersons) sheetPersons.textContent = kid.getAttribute('data-persons') || '—';
    if (sheetDeadline) sheetDeadline.textContent = CLASS_DEADLINE;

    lastTrigger = kid;
    sheet.setAttribute('data-open', 'true');
    sheet.setAttribute('aria-hidden', 'false');
    document.body.style.overflow = 'hidden';
    if (sheetClose) sheetClose.focus();
  }

  function closeSheet() {
    if (!sheet || sheet.getAttribute('data-open') !== 'true') return;
    sheet.setAttribute('data-open', 'false');
    sheet.setAttribute('aria-hidden', 'true');
    document.body.style.overflow = '';
    if (lastTrigger) {
      lastTrigger.focus();
      lastTrigger = null;
    }
  }

  kids.forEach((kid) => {
    kid.addEventListener('click', () => openSheet(kid));
  });

  if (sheet) {
    sheet.querySelectorAll('[data-close-sheet]').forEach((el) => {
      el.addEventListener('click', closeSheet);
    });
  }
  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') closeSheet();
  });

  /* Toast ------------------------------------------------------------- */
  function showToast(message) {
    if (!toastEl) return;
    toastEl.textContent = message;
    toastEl.setAttribute('data-show', 'true');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
      toastEl.setAttribute('data-show', 'false');
    }, 1600);
  }

  if (bellBtn) {
    bellBtn.addEventListener('click', () => showToast('打开消息'));
  }

  // 本页既可从工作台进入，也可从「我的」进入，返回不写死目的地
  if (backBtn) {
    backBtn.addEventListener('click', () => showToast('返回上一页'));
  }

  document.querySelectorAll('.notice-list .notice').forEach((notice) => {
    notice.addEventListener('click', () => {
      const title = notice.getAttribute('data-title') || '';
      showToast(title ? `打开通知「${title}」` : '打开通知');
    });
  });

  /* 四态模拟 ----------------------------------------------------------- */
  function syncStateChrome(state) {
    if (toolbar) toolbar.setAttribute('aria-disabled', String(state === 'loading'));
    document.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === state));
    });
  }

  function setState(state) {
    app.setAttribute('data-state', state);
    syncStateChrome(state);
  }

  function simulateLoading() {
    closeSheet();
    window.clearTimeout(stateTimer);
    setState('loading');
    stateTimer = window.setTimeout(() => {
      setState('normal');
      applyFilters();
    }, 800);
  }

  if (emptyRefresh) emptyRefresh.addEventListener('click', simulateLoading);
  if (retryBtn) retryBtn.addEventListener('click', simulateLoading);

  document.addEventListener('mp:state', (event) => {
    const state = event.detail && event.detail.state
      ? event.detail.state
      : app.getAttribute('data-state');
    syncStateChrome(state);
    if (state !== 'normal') closeSheet();
    if (state === 'normal') applyFilters();
  });

  syncStateChrome(app.getAttribute('data-state') || 'normal');
  applyFilters();
})();
