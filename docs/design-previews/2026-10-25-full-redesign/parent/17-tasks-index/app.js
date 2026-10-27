/* 亲子任务 · 页面交互
 * 筛选与计数、卡片反馈、空/加载/内外错误态、1 秒重试恢复。
 */
(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const taskList = document.getElementById('taskList');
  const cards = Array.from(document.querySelectorAll('.task-card'));
  const segments = Array.from(document.querySelectorAll('.segment'));
  const pendingBadge = document.getElementById('pendingBadge');
  const pendingNote = document.getElementById('pendingNote');
  const pendingNoteCount = document.getElementById('pendingNoteCount');
  const inlineError = document.getElementById('inlineError');
  const skeleton = document.getElementById('taskSkeleton');
  const emptyState = document.getElementById('emptyState');
  const emptyTitle = document.getElementById('emptyTitle');
  const emptyDesc = document.getElementById('emptyDesc');
  const emptyAction = document.getElementById('emptyAction');
  const emptyActionLabel = document.getElementById('emptyActionLabel');
  const errorState = document.getElementById('errorState');
  const errorTitle = document.getElementById('errorTitle');
  const errorDesc = document.getElementById('errorDesc');
  const fullRetry = document.getElementById('fullRetry');
  const wallEntry = document.getElementById('wallEntry');
  const backButton = document.getElementById('backButton');
  const toast = document.getElementById('toast');
  const screenReaderStatus = document.getElementById('screenReaderStatus');
  const demoChips = Array.from(document.querySelectorAll('.mp-demo-chip'));
  const retryButtons = Array.from(document.querySelectorAll('[data-retry]'));

  if (!taskList || !cards.length) return;

  const STATES = ['normal', 'loading', 'empty', 'error-inline', 'error'];
  const FILTER_LABELS = { active: '进行中', checked: '已打卡', all: '全部' };
  const FILTERS = {
    active: (task) => task.status === '1' && !task.submitted,
    checked: (task) => task.submitted,
    all: () => true
  };
  const EMPTY_COPY = {
    active: {
      title: '当前没有待完成的任务',
      desc: '老师发布新任务后会出现在这里',
      action: '查看全部'
    },
    checked: {
      title: '暂无任务',
      desc: '本周期还没有提交记录',
      action: '查看待办'
    },
    all: {
      title: '暂无任务',
      desc: '老师发布任务后会出现在这里',
      action: '查看待办'
    }
  };

  const tasks = cards.map((card) => ({
    el: card,
    id: card.dataset.taskId,
    title: card.dataset.title || '任务',
    status: card.dataset.taskStatus || '1',
    submitted: card.dataset.submitted === 'true'
  }));

  // 原型默认「全部」：一屏看全待完成/已提交/已回应/已结束四种卡片状态；
  // 真机默认是「进行中」，切换逻辑与源码一致。
  let currentFilter = 'all';
  let retryTimer = 0;
  let toastTimer = 0;
  let retrying = false;

  function currentState() {
    const value = app.dataset.state || 'normal';
    return STATES.indexOf(value) >= 0 ? value : 'normal';
  }

  function pendingCount() {
    return tasks.filter(FILTERS.active).length;
  }

  function visibleTasks() {
    return currentState() === 'empty' ? [] : tasks;
  }

  function filteredTasks() {
    return visibleTasks().filter(FILTERS[currentFilter]);
  }

  function announce(message) {
    if (!screenReaderStatus) return;
    screenReaderStatus.textContent = '';
    window.setTimeout(() => {
      screenReaderStatus.textContent = message;
    }, 20);
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

  function setEmptyCopy() {
    const copy = EMPTY_COPY[currentFilter] || EMPTY_COPY.all;
    if (emptyTitle) emptyTitle.textContent = copy.title;
    if (emptyDesc) emptyDesc.textContent = copy.desc;
    if (emptyActionLabel) emptyActionLabel.textContent = copy.action;
  }

  function setRetryCopy(busy) {
    retryButtons.forEach((button) => {
      const label = button.querySelector('[data-retry-label]');
      if (!label) return;
      if (!button.dataset.idleLabel) {
        button.dataset.idleLabel = label.textContent.trim();
      }
      label.textContent = busy ? '正在重试…' : button.dataset.idleLabel;
      button.disabled = busy;
      if (busy) {
        button.setAttribute('aria-busy', 'true');
      } else {
        button.removeAttribute('aria-busy');
      }
    });
  }

  function syncDemoChips(state) {
    demoChips.forEach((chip) => {
      chip.setAttribute(
        'aria-pressed',
        String(chip.getAttribute('data-mp-state') === state)
      );
    });
  }

  function updateView() {
    const state = currentState();
    const list = filteredTasks();
    const hasRows = (state === 'normal' || state === 'error-inline') && list.length > 0;
    const showEmpty = state === 'empty' || ((state === 'normal' || state === 'error-inline') && list.length === 0);
    const showError = state === 'error';
    const showInlineError = state === 'error-inline';
    const showSkeleton = state === 'loading';
    const listIds = new Set(list.map((task) => task.id));

    cards.forEach((card) => {
      card.hidden = !listIds.has(card.dataset.taskId);
    });

    taskList.hidden = !hasRows;
    if (skeleton) skeleton.hidden = !showSkeleton;
    if (emptyState) emptyState.hidden = !showEmpty;
    if (errorState) errorState.hidden = !showError;
    if (inlineError) inlineError.hidden = !showInlineError;

    if (showEmpty) setEmptyCopy();

    const pending = pendingCount();
    const showCounts = state === 'normal' || state === 'error-inline';
    const activeCount = showCounts ? pending : 0;

    if (pendingBadge) {
      pendingBadge.textContent = String(activeCount);
      pendingBadge.hidden = !(showCounts && activeCount > 0);
    }

    const activeFilter = segments.find((segment) => segment.dataset.filter === 'active');
    if (activeFilter) {
      activeFilter.setAttribute(
        'aria-label',
        activeCount > 0 ? '进行中，' + activeCount + ' 个任务待提交' : '进行中'
      );
    }

    if (pendingNote) {
      // 待完成提醒是页面级信息，不随分类隐藏（源码只在「进行中」显示；
      // 原型默认「全部」仍需保留这条提醒）。
      pendingNote.hidden = !(
        (state === 'normal' || state === 'error-inline') &&
        pending > 0
      );
    }
    if (pendingNoteCount) pendingNoteCount.textContent = String(pending);

    if (showError) {
      const bound = app.getAttribute('data-student-bound') !== 'false';
      if (errorTitle) errorTitle.textContent = bound ? '任务加载失败' : '未绑定孩子信息';
      if (errorDesc) {
        errorDesc.textContent = bound
          ? '网络开了小差，请稍后再试'
          : '请先在首页绑定孩子，再查看亲子任务';
      }
      if (fullRetry) fullRetry.hidden = !bound;
    }
  }

  function setFilter(name, shouldAnnounce) {
    if (!FILTERS[name]) return;
    currentFilter = name;
    segments.forEach((segment) => {
      const selected = segment.dataset.filter === name;
      segment.classList.toggle('is-selected', selected);
      segment.setAttribute('aria-pressed', String(selected));
    });
    updateView();

    if (shouldAnnounce) {
      const state = currentState();
      const count = filteredTasks().length;
      const message = state === 'empty' || count === 0
        ? '已切换到' + FILTER_LABELS[name] + '，暂无任务'
        : '已切换到' + FILTER_LABELS[name] + '，' + count + ' 个任务';
      announce(message);
    }
  }

  function cancelRetry() {
    window.clearTimeout(retryTimer);
    retryTimer = 0;
    retrying = false;
    setRetryCopy(false);
  }

  function finishRetry() {
    retryTimer = 0;
    retrying = false;
    setRetryCopy(false);
    app.dataset.state = 'normal';
    syncDemoChips('normal');
    updateView();
    announce('任务已重新加载');
    showToast('已重新加载');
  }

  function startRetry() {
    if (retrying) return;
    const state = currentState();
    if (state !== 'error' && state !== 'error-inline') return;
    retrying = true;
    setRetryCopy(true);
    announce('正在重新加载任务');
    retryTimer = window.setTimeout(finishRetry, 1000);
  }

  segments.forEach((segment) => {
    segment.addEventListener('click', () => {
      setFilter(segment.dataset.filter, true);
    });
  });

  if (emptyAction) {
    emptyAction.addEventListener('click', () => {
      // 空态 CTA 要真的离开空态：先回到正常数据态，再切到另一个分类，
      // 否则在「原型演示-空态」下点按钮只会换一句空态文案。
      app.dataset.state = 'normal';
      syncDemoChips('normal');
      setFilter(currentFilter === 'active' ? 'all' : 'active', true);
    });
  }

  retryButtons.forEach((button) => {
    button.addEventListener('click', startRetry);
  });

  if (wallEntry) {
    wallEntry.addEventListener('click', () => {
      showToast('正在打开班级墙');
    });
  }

  if (backButton) {
    backButton.addEventListener('click', () => {
      showToast('正在返回首页');
    });
  }

  taskList.addEventListener('click', (event) => {
    const target = event.target;
    if (!(target instanceof Element)) return;
    const button = target.closest('button[data-action]');
    if (!button || button.disabled) return;
    if (button.dataset.action === 'detail') {
      showToast('正在打开任务详情');
      return;
    }
    if (button.dataset.intent === 'history') {
      showToast('正在打开打卡记录');
      return;
    }
    showToast('正在打开提交流程');
  });

  document.addEventListener('mp:state', (event) => {
    const next = event.detail && event.detail.state;
    if (STATES.indexOf(next) < 0) return;
    cancelRetry();
    updateView();
    if (next === 'loading') announce('正在加载任务');
    if (next === 'empty') announce('任务列表为空');
    if (next === 'error-inline') announce('加载失败，已保留上次的任务列表');
    if (next === 'error') announce('任务加载失败');
    if (next === 'normal') announce('已恢复为正常任务列表');
  });

  setRetryCopy(false);
  updateView();
})();
