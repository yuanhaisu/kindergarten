(() => {
  'use strict';

  /* ---------- 常量与取样 ---------- */

  const SAMPLE_CONTENT = '亲子任务的截止时间能改到晚上九点吗？下班路上经常来不及拍照提交。';
  const NEW_DATE = '10月26日 周一';
  const MAX_LENGTH = 500;
  const SUBMIT_DELAY = 1300;
  const HIGHLIGHT_DURATION = 2400;
  const TOAST_DURATION = 2000;
  const STATES = ['normal', 'required', 'empty', 'loading', 'submitting', 'failed'];
  const LIST_STATES = ['ok', 'loading', 'empty', 'failed'];
  const SUBMIT_STATES = ['idle', 'required', 'submitting', 'failed'];
  /* 演示芯片是「场景」，一个场景同时铺开列表与提交两个维度；
     真实交互只动其中一个维度，互不覆盖 */
  const SCENARIO_LIST = {
    normal: 'ok',
    required: 'ok',
    empty: 'empty',
    loading: 'loading',
    submitting: 'ok',
    failed: 'failed'
  };
  const SCENARIO_SUBMIT = {
    normal: 'idle',
    required: 'required',
    empty: 'idle',
    loading: 'idle',
    submitting: 'submitting',
    failed: 'failed'
  };
  const CATEGORIES = {
    suggestion: { label: '建议', tone: 'info' },
    complaint: { label: '投诉', tone: 'warning' },
    bug: { label: '问题', tone: 'danger' }
  };

  const app = document.querySelector('.mp-app');
  const page = document.querySelector('.page');
  const main = document.querySelector('.mp-main');
  const noteCard = document.querySelector('.note-card');
  const textarea = document.getElementById('feedback-content');
  const recordsSection = document.getElementById('my-feedback');
  const recordsList = document.getElementById('records-list');

  if (!app || !textarea || !recordsList) return;

  /* 成功态始终作为整页，与表单页面互为兄弟节点 */
  const successView = document.querySelector('.success-view');
  if (page && main && successView && page.contains(successView)) {
    main.insertBefore(successView, page.nextElementSibling);
  }

  const pills = Array.from(document.querySelectorAll('.type-pill'));
  const counter = document.querySelector('.counter');
  const numberEl = document.getElementById('char-count');
  const toast = document.getElementById('toast');
  const submitBtn = document.getElementById('submit-btn');
  const submitRetry = document.getElementById('submit-retry');
  const listRetry = document.getElementById('list-retry');
  const emptyWrite = document.getElementById('empty-write');
  const countLabel = document.getElementById('records-count');
  const successTitle = document.getElementById('success-title');
  const successType = document.getElementById('success-type');
  const successDate = document.getElementById('success-date');
  const successContent = document.getElementById('success-content');
  const successRecordsBtn = document.getElementById('success-view-records');
  const successAgainBtn = document.getElementById('success-again');

  const initialRecords = Array.from(recordsList.querySelectorAll('.record'));
  const motionQuery = window.matchMedia ? window.matchMedia('(prefers-reduced-motion: reduce)') : null;

  let currentCategory = 'suggestion';
  let formLocked = false;
  let submitTimer = null;
  let toastTimer = null;
  let highlightTimer = null;
  let pendingDraft = null;
  let lastInserted = null;
  let localSeq = 0;

  const prefersReducedMotion = () => Boolean(motionQuery && motionQuery.matches);

  /* ---------- 基础工具 ---------- */

  function setCounter(value) {
    const target = numberEl || counter;
    if (!counter) { /* 没有计数器时静默忽略 */ }
    else {
      counter.removeAttribute('aria-hidden');
    }
    if (numberEl) numberEl.removeAttribute('aria-hidden');
    if (!target) return;
    if (target === counter) target.textContent = `${value} / ${MAX_LENGTH}`;
    else target.textContent = String(value);
  }

  function setContent(value) {
    textarea.value = value;
    setCounter(value.length);
  }

  function selectType(type) {
    if (!CATEGORIES[type]) return;
    currentCategory = type;
    pills.forEach((pill) => {
      pill.setAttribute('aria-pressed', String(pill.dataset.category === type));
    });
  }

  function setFormLocked(locked) {
    formLocked = Boolean(locked);
    pills.forEach((pill) => { pill.disabled = formLocked; });
    textarea.readOnly = formLocked;
    if (!noteCard) return;
    if (formLocked) noteCard.setAttribute('inert', '');
    else noteCard.removeAttribute('inert');
  }

  function syncCount() {
    if (!countLabel) return;
    const total = recordsList.querySelectorAll('.record').length;
    const listState = app.dataset.listState;
    let text = `共 ${total} 条`;
    if (listState === 'empty') text = '共 0 条';
    else if (listState === 'loading') text = '加载中';
    else if (listState === 'failed') text = '加载失败';
    countLabel.textContent = text;
  }

  /* 芯片只标记「当前演示场景」；真实交互后场景不成立时置为 custom，避免芯片谎报状态 */
  function setScenario(name) {
    const known = STATES.indexOf(name) >= 0 ? name : null;
    app.dataset.state = known || 'custom';
    document.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(known !== null && chip.dataset.mpState === known));
    });
  }

  /* 列表与提交是两条独立通道：任一通道重试都不得改写另一条的失败/成功，也不得把没加载成功的记录显示出来 */
  function setListState(next) {
    app.dataset.listState = LIST_STATES.indexOf(next) >= 0 ? next : 'ok';
    syncCount();
  }

  function setSubmitState(next) {
    const value = SUBMIT_STATES.indexOf(next) >= 0 ? next : 'idle';
    app.dataset.submitState = value;
    const isRequired = value === 'required';
    const isBusy = value === 'submitting';
    if (submitBtn) {
      submitBtn.setAttribute('aria-disabled', String(isRequired || isBusy));
      submitBtn.setAttribute('aria-busy', String(isBusy));
    }
    textarea.setAttribute('aria-invalid', String(isRequired));
  }

  /* 演示芯片：场景 → 两个维度一起铺开 */
  function setState(state) {
    const scenario = STATES.indexOf(state) >= 0 ? state : 'normal';
    setScenario(scenario);
    setListState(SCENARIO_LIST[scenario]);
    setSubmitState(SCENARIO_SUBMIT[scenario]);
  }

  function scrollToY(top) {
    window.scrollTo({
      top: Math.max(0, Math.round(top)),
      behavior: prefersReducedMotion() ? 'auto' : 'smooth'
    });
  }

  function scrollToElement(element) {
    if (!element) return;
    const rect = element.getBoundingClientRect();
    scrollToY(rect.top + window.scrollY - 72);
  }

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-visible');
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastTimer = null;
    }, TOAST_DURATION);
  }

  function hideToast() {
    if (toastTimer) {
      clearTimeout(toastTimer);
      toastTimer = null;
    }
    if (toast) toast.classList.remove('is-visible');
  }

  /* ---------- 记录时间轴 ---------- */

  function collapseRecord(record) {
    if (!record) return;
    record.classList.remove('record--new');
    const card = record.querySelector('.record-card');
    if (card) card.classList.remove('is-open');
    const toggle = record.querySelector('.record-toggle');
    if (toggle) {
      toggle.setAttribute('aria-expanded', 'false');
      toggle.setAttribute('aria-label', '展开反馈内容');
    }
  }

  function restoreRecords() {
    recordsList.replaceChildren(...initialRecords);
    initialRecords.forEach(collapseRecord);
    syncExpanders();
  }

  /* 只有真正被三行截断的记录才显示展开箭头；否则箭头点了没有变化 */
  function syncExpanders() {
    recordsList.querySelectorAll('.record-card').forEach((card) => {
      if (card.classList.contains('is-open')) return;
      const text = card.querySelector('.record-text');
      const toggle = card.querySelector('.record-toggle');
      if (!text || !toggle) return;
      const expandable = text.scrollHeight > text.clientHeight + 1;
      toggle.hidden = !expandable;
      card.classList.toggle('is-static', !expandable);
    });
  }

  function insertRecord(category, content) {
    const meta = CATEGORIES[category] || CATEGORIES.suggestion;
    const seq = ++localSeq;
    const textId = `record-text-local-${seq}`;
    const record = document.createElement('li');
    record.className = 'record';
    record.dataset.local = 'true';
    record.dataset.recordId = `local-${seq}`;
    record.innerHTML = [
      '<span class="record-dot record-dot--submitted" aria-hidden="true"></span>',
      '<article class="record-card">',
      '<div class="record-meta">',
      '<span class="badge"></span>',
      '<span class="record-status">',
      '<span class="record-status-dot record-status-dot--submitted" aria-hidden="true"></span>',
      '已提交',
      '</span>',
      '<span class="record-date"></span>',
      '<button class="record-toggle" type="button" aria-expanded="false" aria-label="展开反馈内容">',
      '<svg class="record-chevron" width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 6.5 8 10.5l4-4"/></svg>',
      '</button>',
      '</div>',
      '<p class="record-text"></p>',
      '</article>'
    ].join('');
    const card = record.querySelector('.record-card');
    if (card) card.dataset.odId = `feedback-card-local-${seq}`;
    const badge = record.querySelector('.badge');
    if (badge) {
      badge.classList.add(`badge--${meta.tone}`);
      badge.textContent = meta.label;
    }
    const date = record.querySelector('.record-date');
    if (date) date.textContent = NEW_DATE;
    const text = record.querySelector('.record-text');
    if (text) {
      text.id = textId;
      text.textContent = content;
    }
    const toggle = record.querySelector('.record-toggle');
    if (toggle) toggle.setAttribute('aria-controls', textId);
    recordsList.prepend(record);
    syncExpanders();
    syncCount();
    return record;
  }

  function toggleRecord(card) {
    if (!card) return;
    const open = card.classList.toggle('is-open');
    const toggle = card.querySelector('.record-toggle');
    if (!toggle) return;
    toggle.setAttribute('aria-expanded', String(open));
    toggle.setAttribute('aria-label', open ? '收起反馈内容' : '展开反馈内容');
  }

  /* ---------- 提交 / 成功 / 返回 ---------- */

  function focusComposer() {
    scrollToElement(noteCard);
    window.requestAnimationFrame(() => {
      if (formLocked) return;
      textarea.focus({ preventScroll: true });
    });
  }

  function startSubmit(content) {
    if (formLocked) return;
    hideToast();
    pendingDraft = {
      category: currentCategory,
      content
    };
    /* 只动提交维度：列表仍是失败态时继续保持失败，不因重试提交而假装加载成功 */
    setScenario('submitting');
    setSubmitState('submitting');
    setFormLocked(true);
    if (submitTimer) clearTimeout(submitTimer);
    submitTimer = window.setTimeout(() => {
      submitTimer = null;
      showSuccess();
    }, SUBMIT_DELAY);
  }

  function handleSubmit() {
    if (formLocked) return;
    if (app.dataset.submitState === 'required') {
      focusComposer();
      return;
    }
    const content = textarea.value.trim();
    if (!content) {
      setScenario('required');
      setSubmitState('required');
      focusComposer();
      return;
    }
    startSubmit(content);
  }

  function showSuccess() {
    if (!pendingDraft) return;
    const draft = pendingDraft;
    pendingDraft = null;
    const meta = CATEGORIES[draft.category] || CATEGORIES.suggestion;

    if (successType) {
      successType.className = `badge badge--${meta.tone}`;
      successType.textContent = meta.label;
    }
    if (successDate) successDate.textContent = NEW_DATE;
    if (successContent) successContent.textContent = draft.content;

    lastInserted = insertRecord(draft.category, draft.content);
    setContent('');
    /* 成功后表单已清空，回来时必须就是阻断态，而不是让空内容顶着品牌渐变可点 */
    setScenario(null);
    setSubmitState('required');
    document.body.dataset.view = 'success';
    setFormLocked(true);
    window.scrollTo({ top: 0, behavior: 'auto' });
    window.requestAnimationFrame(() => {
      if (successTitle) successTitle.focus({ preventScroll: true });
    });
  }

  function viewRecords() {
    const target = lastInserted;
    document.body.dataset.view = 'form';
    setFormLocked(false);
    setScenario(null);
    setSubmitState('required');
    setContent('');
    window.requestAnimationFrame(() => {
      scrollToElement(recordsSection);
      if (!target) return;
      target.classList.add('record--new');
      if (highlightTimer) clearTimeout(highlightTimer);
      highlightTimer = window.setTimeout(() => {
        target.classList.remove('record--new');
        highlightTimer = null;
      }, HIGHLIGHT_DURATION);
      const toggle = target.querySelector('.record-toggle');
      if (toggle && !toggle.hidden) toggle.focus({ preventScroll: true });
    });
  }

  function writeAnother() {
    document.body.dataset.view = 'form';
    setFormLocked(false);
    setScenario(null);
    setSubmitState('required');
    setContent('');
    window.scrollTo({ top: 0, behavior: 'auto' });
    window.requestAnimationFrame(() => {
      textarea.focus({ preventScroll: true });
    });
  }

  /* ---------- 演示状态 ---------- */

  function applyDemoState(input) {
    const state = STATES.indexOf(input) >= 0 ? input : 'normal';
    if (submitTimer) {
      clearTimeout(submitTimer);
      submitTimer = null;
    }
    if (highlightTimer) {
      clearTimeout(highlightTimer);
      highlightTimer = null;
    }
    hideToast();
    pendingDraft = null;
    lastInserted = null;
    document.body.dataset.view = 'form';
    setFormLocked(false);
    restoreRecords();
    if (state === 'empty') recordsList.replaceChildren();
    selectType('suggestion');
    setContent(state === 'required' ? '' : SAMPLE_CONTENT);
    if (state === 'submitting') {
      /* 「提交中」必须自己走出转圈：复用真实提交的 1.3s 计时进入成功态 */
      setListState(SCENARIO_LIST[state]);
      startSubmit(SAMPLE_CONTENT);
      return;
    }
    setState(state);
  }

  function recoverRecords() {
    if (app.dataset.listState !== 'failed') return;
    /* 只恢复列表：提交失败条保持原样，等它自己的重试来清 */
    setListState('ok');
    setScenario(null);
    showToast('已刷新');
    window.requestAnimationFrame(() => {
      const firstToggle = recordsList.querySelector('.record-toggle:not([hidden])');
      if (firstToggle) firstToggle.focus({ preventScroll: true });
    });
  }

  /* ---------- 事件绑定 ---------- */

  pills.forEach((pill) => {
    pill.addEventListener('click', () => {
      if (formLocked) return;
      selectType(pill.dataset.category);
    });
  });

  textarea.addEventListener('input', () => {
    if (textarea.value.length > MAX_LENGTH) {
      textarea.value = textarea.value.slice(0, MAX_LENGTH);
    }
    setCounter(textarea.value.length);
    const empty = !textarea.value.trim();
    const submitState = app.dataset.submitState;
    if (submitState === 'required' && !empty) setSubmitState('idle');
    else if (empty && submitState === 'idle') setSubmitState('required');
  });

  if (submitBtn) submitBtn.addEventListener('click', handleSubmit);

  if (submitRetry) {
    submitRetry.addEventListener('click', () => {
      const content = textarea.value.trim();
      if (!content) {
        /* 只把提交维度打回阻断态：列表若是失败态，仍由列表自己的重试来恢复 */
        setScenario('required');
        setSubmitState('required');
        focusComposer();
        return;
      }
      startSubmit(content);
    });
  }

  if (listRetry) listRetry.addEventListener('click', recoverRecords);
  if (emptyWrite) emptyWrite.addEventListener('click', focusComposer);

  recordsList.addEventListener('click', (event) => {
    const card = event.target.closest('.record-card');
    if (!card || !recordsList.contains(card)) return;
    const toggle = card.querySelector('.record-toggle');
    if (!toggle || toggle.hidden) return;
    toggleRecord(card);
  });

  if (successRecordsBtn) successRecordsBtn.addEventListener('click', viewRecords);
  if (successAgainBtn) successAgainBtn.addEventListener('click', writeAnother);

  document.addEventListener('mp:state', (event) => {
    const next = event.detail && event.detail.state;
    applyDemoState(next);
  });

  /* ---------- 初始化 ---------- */

  function initA11y() {
    textarea.setAttribute('aria-label', '反馈内容');
    const described = (textarea.getAttribute('aria-describedby') || '')
      .split(/\s+/)
      .filter((id) => id && document.getElementById(id));
    if (counter) {
      if (!counter.id) counter.id = 'char-counter';
      if (described.indexOf(counter.id) < 0) described.push(counter.id);
      counter.removeAttribute('aria-hidden');
    }
    if (numberEl) numberEl.removeAttribute('aria-hidden');
    textarea.setAttribute('aria-describedby', described.join(' '));
    recordsList.setAttribute('aria-label', '我的反馈记录');
    if (successTitle) {
      if (!successTitle.id) successTitle.id = 'success-title';
      successTitle.setAttribute('tabindex', '-1');
      if (successView) successView.setAttribute('aria-labelledby', successTitle.id);
    }
  }

  initA11y();

  const initiallyPressed = pills.find((pill) => pill.getAttribute('aria-pressed') === 'true');
  if (initiallyPressed && CATEGORIES[initiallyPressed.dataset.category]) {
    selectType(initiallyPressed.dataset.category);
  } else {
    selectType('suggestion');
  }
  setCounter(textarea.value.length);
  setState('normal');
  setFormLocked(false);
  syncExpanders();
  window.requestAnimationFrame(syncExpanders);
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(syncExpanders).catch(() => {});
  }
})();
