(function () {
  'use strict';

  const $ = (sel, root) => (root || document).querySelector(sel);
  const $$ = (sel, root) => Array.from((root || document).querySelectorAll(sel));

  const POLICY = { temperature_required: false, fever_threshold: 37.3 };
  const CHECKS = [
    { key: 'throat_status', label: '喉咙', category: 'throat', options: ['正常', '发红', '红肿'] },
    { key: 'hand_status', label: '手部', category: 'hand', options: ['正常', '红疹', '疱疹'] },
    { key: 'mouth_status', label: '口腔', category: 'mouth', options: ['正常', '溃疡', '疱疹'] },
    { key: 'eye_status', label: '眼睛', category: 'eye', options: ['正常', '发红', '分泌物'] },
    { key: 'skin_status', label: '皮肤', category: 'skin', options: ['正常', '红疹', '水疱'] }
  ];
  const CHECK_BY_KEY = {};
  const CHECK_BY_CATEGORY = {};
  CHECKS.forEach(function (item) {
    CHECK_BY_KEY[item.key] = item;
    CHECK_BY_CATEGORY[item.category] = item;
  });
  const STATUS = [
    { label: '健康', cls: 'success' },
    { label: '需观察', cls: 'warning' },
    { label: '劝返', cls: 'danger' }
  ];
  const TOASTS = ['健康入园 · 已通知家长', '需观察 · 已通知家长', '劝返就医 · 已通知家长'];

  function mk(id, name, className, classId, check, overall, time) {
    return {
      id: id,
      pickup_record_id: 900 + id,
      student_id: 300 + id,
      student_name: name,
      class_name: className,
      class_id: classId,
      check_status: check,
      overall_status: overall,
      pickup_time: time,
      avatar: ''
    };
  }

  const SEED = [
    mk(1, '乐乐', '小一班', 11, 0, 0, '08:12'),
    mk(2, '小雨', '小一班', 11, 0, 0, '08:20'),
    mk(3, '朵朵', '中二班', 12, 0, 0, '08:26'),
    mk(4, '安安', '中二班', 12, 0, 0, '08:31'),
    mk(5, '果果', '大一班', 13, 0, 0, '08:38'),
    mk(6, '沐沐', '小一班', 11, 1, 0, '08:02'),
    mk(7, '一一', '小一班', 11, 1, 0, '08:06'),
    mk(8, '糖糖', '小一班', 11, 1, 1, '08:09'),
    mk(9, '小宝', '中二班', 12, 1, 0, '08:04'),
    mk(10, '米粒', '中二班', 12, 1, 0, '08:11'),
    mk(11, '天天', '中二班', 12, 1, 0, '08:15'),
    mk(12, '豆豆', '中二班', 12, 1, 2, '08:18'),
    mk(13, '辰辰', '大一班', 13, 1, 0, '07:58'),
    mk(14, '悦悦', '大一班', 13, 1, 0, '08:01'),
    mk(15, '壮壮', '大一班', 13, 1, 0, '08:05'),
    mk(16, '苗苗', '大一班', 13, 1, 0, '08:08'),
    mk(17, '圆圆', '大一班', 13, 1, 1, '08:14'),
    mk(18, '阿布', '大一班', 13, 1, 0, '08:22')
  ];

  let records = SEED.map(function (r) { return Object.assign({}, r); });
  const byId = new Map(records.map(function (r) { return [r.id, r]; }));
  let completedOrder = records
    .filter(function (r) { return r.check_status === 1; })
    .sort(function (a, b) { return a.pickup_time.localeCompare(b.pickup_time); })
    .map(function (r) { return r.id; });
  const classes = Array.from(new Set(records.map(function (r) { return r.class_id; })))
    .sort(function (a, b) { return a - b; })
    .map(function (id) {
      const row = records.find(function (r) { return r.class_id === id; });
      return { id: id, name: row ? row.class_name : String(id) };
    });

  let state = 'normal';
  let activeClass = 'all';
  let openId = null;
  let completedOpen = true;
  let drafts = new Map();
  let firstSubmitFailed = false;
  let submitTimer = 0;
  let stateTimer = 0;
  let toastTimer = 0;
  let activeSubmit = null;
  let retryBusy = false;

  const ESC = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
  function esc(value) {
    return String(value).replace(/[&<>"']/g, function (ch) { return ESC[ch]; });
  }
  function initial(name) {
    return String(name).slice(0, 1) || '?';
  }
  function draftFor(id) {
    if (!drafts.has(id)) {
      drafts.set(id, {
        temperature: '',
        throat_status: 0,
        hand_status: 0,
        mouth_status: 0,
        eye_status: 0,
        skin_status: 0,
        remark: '',
        is_isolated: false,
        photos: [],
        error: false,
        failedStatus: 0,
        submitting: false
      });
    }
    return drafts.get(id);
  }
  function scoped() {
    if (activeClass === 'all') return records;
    return records.filter(function (r) { return String(r.class_id) === activeClass; });
  }
  function scopedCounts() {
    const list = scoped();
    return {
      total: list.length,
      pending: list.filter(function (r) { return r.check_status === 0; }).length,
      done: list.filter(function (r) { return r.check_status === 1; }).length
    };
  }
  function pendingRows() {
    return scoped()
      .filter(function (r) { return r.check_status === 0; })
      .sort(function (a, b) { return a.pickup_time.localeCompare(b.pickup_time); });
  }
  function completedRows() {
    return completedOrder
      .map(function (id) { return byId.get(id); })
      .filter(function (r) {
        return r && (activeClass === 'all' || String(r.class_id) === activeClass);
      });
  }
  // 空态 = 今天的晨检已全部完成：直接由 records 派生，提交过的结论不会被重置，
  // 也不再有「点击后才补齐」的假动作。
  function emptyRows() {
    return records.slice().sort(function (a, b) { return a.pickup_time.localeCompare(b.pickup_time); });
  }

  function buildFilter() {
    const items = [{ id: 'all', name: '全部' }].concat(classes);
    $('#filterWrap').innerHTML = items.map(function (item) {
      return '<button type="button" class="class-chip" data-class="' + item.id + '" aria-pressed="' +
        (String(item.id) === activeClass) + '"><span class="class-chip-face">' + esc(item.name) + '</span></button>';
    }).join('');
  }
  function updateFilterPressed() {
    $$('#filterWrap .class-chip').forEach(function (btn) {
      btn.setAttribute('aria-pressed', String(btn.dataset.class === activeClass));
    });
  }
  function renderPolicy() {
    const required = !!POLICY.temperature_required;
    $('#policyPill').classList.toggle('is-required', required);
    $('#policyText').textContent = required
      ? '本园要求测温 · ' + POLICY.fever_threshold + '°C 预警'
      : '按需测温 · 体温选填';
  }
  function renderStats() {
    if (state === 'error') {
      $('#statTotal').textContent = '—';
      $('#statPending').textContent = '—';
      $('#statDone').textContent = '—';
      $('#pendingStat').classList.remove('stat--hot');
      return;
    }
    const counts = state === 'empty'
      ? { total: records.length, pending: 0, done: records.length }
      : scopedCounts();
    $('#statTotal').textContent = String(counts.total);
    $('#statPending').textContent = String(counts.pending);
    $('#statDone').textContent = String(counts.done);
    $('#pendingStat').classList.toggle('stat--hot', state === 'normal' && counts.pending > 0);
  }
  // 空态文案里的数字也由 records 派生，避免和列表/统计三处各说各话。
  function renderEmptyCopy() {
    const attention = records.filter(function (r) { return r.overall_status !== 0; }).length;
    $('#emptyTotal').textContent = String(records.length);
    $('#emptyAttention').textContent = String(attention);
  }
  function renderPending() {
    const section = $('#pendingSection');
    if (state !== 'normal') {
      section.hidden = true;
      return;
    }
    section.hidden = false;
    const list = pendingRows();
    $('#pendingCount').textContent = list.length + ' 人';
    $('#pendingHint').textContent = list.length ? '点开逐项检查' : '本班已全部完成';
    if (!list.length) {
      $('#pendingList').innerHTML = '<p class="quiet-empty"><svg class="ico" width="18" height="18" aria-hidden="true"><use href="#i-check"></use></svg><span>本班的晨检都完成了</span></p>';
      return;
    }
    $('#pendingList').innerHTML = list.map(childHTML).join('');
  }

  function childHTML(record) {
    const open = openId === record.id;
    return '<article class="child-card' + (open ? ' is-open' : '') + '" data-id="' + record.id + '">' +
      '<button type="button" class="child-toggle" data-action="toggle" aria-expanded="' + open + '"' +
      (open ? ' aria-controls="panel-' + record.id + '"' : '') + '>' +
        '<span class="avatar avatar--md" aria-hidden="true">' + esc(initial(record.student_name)) + '</span>' +
        '<span class="child-main">' +
          '<span class="child-name">' + esc(record.student_name) + '</span>' +
          '<span class="child-meta">' + esc(record.class_name) + ' · ' + esc(record.pickup_time) + ' 入园</span>' +
        '</span>' +
        '<span class="child-trail">' +
          '<span class="child-check">' + (open ? '检查中' : '待晨检') + '</span>' +
          '<span class="child-collapse" aria-hidden="true">' + (open ? '收起' : '展开') +
            '<svg class="ico" width="14" height="14"><use href="#i-chevron-down"></use></svg>' +
          '</span>' +
        '</span>' +
      '</button>' +
      (open ? panelHTML(record, draftFor(record.id)) : '') +
      '</article>';
  }

  function panelHTML(record, draft) {
    const required = !!POLICY.temperature_required;
    const hint = tempWarning(draft);
    const rows = CHECKS.map(function (item) {
      const active = draft[item.key];
      const chips = item.options.map(function (label, val) {
        return '<button type="button" class="check-chip" data-field="' + item.key + '" data-val="' + val +
          '" aria-pressed="' + (active === val) + '"><span class="check-chip-face">' + label + '</span></button>';
      }).join('');
      return '<div class="check-row">' +
        '<span class="check-label" id="ck-' + item.category + '-' + record.id + '">' + item.label + '</span>' +
        '<div class="check-chips" role="group" aria-labelledby="ck-' + item.category + '-' + record.id + '">' + chips + '</div>' +
        '<button type="button" class="photo-mini" data-category="' + item.category + '" aria-label="为' + item.label +
          '异常拍照"' + (active === 0 ? ' hidden' : '') + '>' +
          '<span class="photo-mini-face"><svg class="ico" width="15" height="15" aria-hidden="true"><use href="#i-camera"></use></svg></span>' +
        '</button>' +
      '</div>';
    }).join('');
    const submitDisabled = draft.submitting ? ' disabled' : '';
    return '<div class="check-panel" id="panel-' + record.id + '" role="group" aria-label="' +
      esc(record.student_name) + '的晨检项目">' +
      '<div class="field-row">' +
        '<label class="field-label" for="temp-' + record.id + '">体温' +
          '<span class="field-note' + (required ? ' is-required' : '') + '">' + (required ? '必填' : '选填') + '</span>' +
        '</label>' +
        '<div class="temp-field">' +
          '<input class="temp-input" id="temp-' + record.id + '" data-field="temperature" type="text" inputmode="decimal" autocomplete="off" placeholder="36.5" value="' +
            esc(draft.temperature) + '" aria-describedby="temphint-' + record.id + '">' +
          '<span class="temp-unit" aria-hidden="true">°C</span>' +
        '</div>' +
      '</div>' +
      '<p class="temp-hint" id="temphint-' + record.id + '"' + (hint ? '' : ' hidden') + '>' +
        '<span class="temp-hint-dot" aria-hidden="true"></span><span class="temp-hint-text">' + esc(hint) + '</span>' +
      '</p>' +
      '<div class="check-list">' + rows + '</div>' +
      '<div class="remark-block">' +
        '<label class="field-label" for="remark-' + record.id + '">备注</label>' +
        '<textarea class="remark-input" id="remark-' + record.id + '" data-field="remark" maxlength="200" placeholder="异常情况说明（选填）">' +
          esc(draft.remark) + '</textarea>' +
        '<div class="remark-foot"><span class="remark-count">' + draft.remark.length + '/200</span></div>' +
      '</div>' +
      '<div class="photos-block"' + (draft.photos.length ? '' : ' hidden') + '>' +
        '<p class="photos-title">异常照片</p>' +
        '<ul class="photo-list">' + photoListHTML(draft) + '</ul>' +
      '</div>' +
      '<button type="button" class="isolation-row" data-action="isolation" role="switch" aria-checked="' +
        draft.is_isolated + '">' +
        '<span class="isolation-copy">已带到隔离室等待家长</span>' +
        '<span class="switch-track" aria-hidden="true"><span class="switch-knob"></span></span>' +
      '</button>' +
      '<div class="submit-area">' +
        '<div class="submit-error" role="alert"' + (draft.error ? '' : ' hidden') + '>' +
          '<div class="submit-error-top">' +
            '<p class="submit-error-text">提交失败，请重试</p>' +
            '<button type="button" class="retry-link" data-action="retry">重试</button>' +
          '</div>' +
          '<p class="submit-error-demo">演示：本次提交被模拟为网络失败</p>' +
        '</div>' +
        '<div class="submit-actions' + (draft.submitting ? ' is-submitting' : '') + '">' +
          '<button type="button" class="submit-primary" data-action="submit" data-status="0"' + submitDisabled + '>健康入园</button>' +
          '<div class="submit-secondary">' +
            '<button type="button" class="submit-warn" data-action="submit" data-status="1"' + submitDisabled + '>需观察</button>' +
            '<button type="button" class="submit-danger" data-action="submit" data-status="2"' + submitDisabled + '>劝返就医</button>' +
          '</div>' +
        '</div>' +
      '</div>' +
    '</div>';
  }

  function photoListHTML(draft) {
    return draft.photos.map(function (photo) {
      return '<li class="photo-thumb"><svg class="ico" width="16" height="16" aria-hidden="true"><use href="#i-camera"></use></svg><span>' +
        esc(photo.label) + '</span></li>';
    }).join('');
  }
  function tempWarning(draft) {
    if (!draft.temperature.trim()) return '';
    const value = Number(draft.temperature);
    if (!Number.isFinite(value) || value < POLICY.fever_threshold) return '';
    return value + '°C 高于阈值 ' + POLICY.fever_threshold + '°C';
  }
  function completedRowHTML(record) {
    const status = STATUS[record.overall_status] || STATUS[0];
    return '<li class="completed-row">' +
      '<span class="avatar avatar--sm" aria-hidden="true">' + esc(initial(record.student_name)) + '</span>' +
      '<span class="completed-main">' +
        '<span class="completed-name">' + esc(record.student_name) + '</span>' +
        '<span class="completed-class">' + esc(record.class_name) + '</span>' +
      '</span>' +
      '<span class="status-tag status-tag--' + status.cls + '"><span class="status-dot" aria-hidden="true"></span>' + status.label + '</span>' +
    '</li>';
  }
  function renderCompleted() {
    const section = $('#completedSection');
    if (state === 'loading' || state === 'error') {
      section.hidden = true;
      return;
    }
    const rows = state === 'empty' ? emptyRows() : completedRows();
    $('#completedCount').textContent = String(rows.length);
    $('#completedList').innerHTML = rows.map(completedRowHTML).join('');
    section.hidden = state === 'normal' && !rows.length;
    $('#completedToggle').setAttribute('aria-expanded', String(completedOpen));
    $('#completedToggleText').textContent = completedOpen ? '收起' : '展开';
    $('#completedList').hidden = !completedOpen;
  }
  function render() {
    const normal = state === 'normal';
    $('#filterWrap').hidden = !normal;
    $('#pendingSection').hidden = !normal;
    $('#loadingView').hidden = state !== 'loading';
    $('#emptyView').hidden = state !== 'empty';
    $('#errorView').hidden = state !== 'error';
    renderPolicy();
    renderStats();
    renderEmptyCopy();
    if (normal) {
      updateFilterPressed();
      renderPending();
    }
    renderCompleted();
  }

  function showToast(message) {
    const toast = $('.toast');
    toast.textContent = message;
    toast.classList.add('is-visible');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () {
      toast.classList.remove('is-visible');
      toastTimer = 0;
    }, 2000);
  }
  function scrollToEl(target) {
    if (!target) return;
    // The document scrolls, not .mp-main (the shell only pins the chrome).
    const top = target.getBoundingClientRect().top + window.scrollY - 72;
    const reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    try {
      window.scrollTo({ top: Math.max(0, top), behavior: reduce ? 'auto' : 'smooth' });
    } catch (err) {
      window.scrollTo(0, Math.max(0, top));
    }
  }
  function focusTemp(id) {
    const input = $('.child-card[data-id="' + id + '"] .temp-input');
    if (input) input.focus({ preventScroll: true });
  }
  function focusAfterSubmit() {
    const next = $('#pendingList .child-toggle');
    if (next) {
      next.focus({ preventScroll: true });
      return;
    }
    const completed = $('#completedToggle');
    if (completed && !$('#completedSection').hidden) {
      completed.focus({ preventScroll: true });
      scrollToEl($('#completedSection'));
    }
  }

  function setState(next) {
    clearTimeout(stateTimer);
    stateTimer = 0;
    if (activeSubmit) {
      clearTimeout(submitTimer);
      submitTimer = 0;
      const busyDraft = drafts.get(activeSubmit.id);
      if (busyDraft) busyDraft.submitting = false;
      activeSubmit = null;
    }
    state = next;
    activeClass = 'all';
    openId = null;
    retryBusy = false;
    if (next === 'empty') completedOpen = true;
    $('.mp-app').dataset.state = next;
    const retry = $('#retryLoad');
    if (retry) {
      retry.disabled = false;
      retry.textContent = '重试';
    }
    render();
    $$('.mp-demo-chip').forEach(function (chip) {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === next));
    });
    if (next !== 'normal') {
      window.scrollTo(0, 0);
    }
    // 演示用：加载骨架保持到用户切回其它状态，便于审阅时观察（真实请求会在成功后自行结束）。
  }

  function submitCheck(id, status) {
    if (state !== 'normal' || activeSubmit) return;
    if (![0, 1, 2].includes(status)) return;
    const record = byId.get(id);
    if (!record || record.check_status !== 0) return;
    const draft = draftFor(id);
    if (draft.submitting) return;
    const raw = draft.temperature.trim();
    if (!raw && POLICY.temperature_required) {
      showToast('本园要求录入体温');
      focusTemp(id);
      return;
    }
    if (raw && !Number.isFinite(Number(raw))) {
      showToast('请输入正确的体温');
      focusTemp(id);
      return;
    }
    const payload = {
      temperature: raw ? Number(raw) : null,
      throat_status: draft.throat_status,
      hand_status: draft.hand_status,
      mouth_status: draft.mouth_status,
      eye_status: draft.eye_status,
      skin_status: draft.skin_status,
      overall_status: status,
      is_isolated: draft.is_isolated,
      remark: draft.remark.trim()
    };
    draft.submitting = true;
    draft.error = false;
    draft.failedStatus = status;
    openId = id;
    const card = $('.child-card[data-id="' + id + '"]');
    if (card) {
      const error = $('.submit-error', card);
      if (error) error.hidden = true;
      $$('input, textarea, button', card).forEach(function (el) { el.disabled = true; });
      const actions = $('.submit-actions', card);
      if (actions) actions.classList.add('is-submitting');
      const clicked = $('[data-action="submit"][data-status="' + status + '"]', card);
      if (clicked) clicked.textContent = '提交中…';
    }
    activeSubmit = { id: id, status: status, payload: payload };
    clearTimeout(submitTimer);
    submitTimer = setTimeout(function () {
      submitTimer = 0;
      finishSubmit();
    }, 500);
  }

  function finishSubmit() {
    const job = activeSubmit;
    if (!job) return;
    activeSubmit = null;
    const id = job.id;
    const status = job.status;
    const payload = job.payload;
    const record = byId.get(id);
    const draft = drafts.get(id);
    if (!record || !draft) {
      render();
      return;
    }
    draft.submitting = false;
    if (!firstSubmitFailed) {
      firstSubmitFailed = true;
      draft.error = true;
      render();
      const retry = $('.child-card[data-id="' + id + '"] .retry-link');
      if (retry) retry.focus({ preventScroll: true });
      return;
    }
    record.check_status = 1;
    record.overall_status = payload.overall_status;
    completedOrder = [id].concat(completedOrder.filter(function (item) { return item !== id; }));
    drafts.delete(id);
    openId = null;
    render();
    const notify = { notified_count: 1, notify_failed_count: 0 };
    const isolated = payload.is_isolated ? ' · 已隔离' : '';
    showToast(notify.notify_failed_count > 0
      ? '已记录 · ' + notify.notify_failed_count + ' 位家长通知失败，请线下告知'
      : TOASTS[status] + isolated);
    focusAfterSubmit();
  }

  function bindEvents() {
    $('#backBtn').addEventListener('click', function () { showToast('返回工作台'); });
    $('#eventEntry').addEventListener('click', function () { showToast('已打开园级事件'); });
    $('.mp-demo').addEventListener('click', function (event) {
      const chip = event.target.closest('[data-mp-state]');
      if (chip) setState(chip.dataset.mpState);
    });
    $('#filterWrap').addEventListener('click', function (event) {
      const chip = event.target.closest('.class-chip');
      if (!chip || activeSubmit) return;
      activeClass = chip.dataset.class;
      openId = null;
      render();
    });
    $('#completedToggle').addEventListener('click', function () {
      completedOpen = !completedOpen;
      renderCompleted();
    });
    $('#emptyReview').addEventListener('click', function () {
      if (state !== 'empty') return;
      // 只做定位，不改任何数据：列表本来就是完整的 18 条。
      completedOpen = true;
      renderCompleted();
      scrollToEl($('#completedSection'));
    });
    $('#retryLoad').addEventListener('click', function () {
      if (state !== 'error' || retryBusy) return;
      retryBusy = true;
      const button = $('#retryLoad');
      button.disabled = true;
      button.textContent = '重试中…';
      stateTimer = setTimeout(function () {
        stateTimer = 0;
        setState('normal');
      }, 700);
    });

    const list = $('#pendingList');
    list.addEventListener('click', function (event) {
      const card = event.target.closest('.child-card');
      if (!card) return;
      const id = Number(card.dataset.id);
      const draft = draftFor(id);
      const toggle = event.target.closest('.child-toggle');
      if (toggle) {
        if (activeSubmit) return;
        openId = openId === id ? null : id;
        renderPending();
        const next = $('.child-card[data-id="' + id + '"] .child-toggle');
        if (next) next.focus({ preventScroll: true });
        return;
      }
      if (draft.submitting) return;
      const chip = event.target.closest('.check-chip');
      if (chip) {
        const field = chip.dataset.field;
        const val = Number(chip.dataset.val);
        draft[field] = val === 0 ? 0 : (draft[field] === val ? 0 : val);
        $$('.check-chip[data-field="' + field + '"]', card).forEach(function (btn) {
          btn.setAttribute('aria-pressed', String(Number(btn.dataset.val) === draft[field]));
        });
        const meta = CHECK_BY_KEY[field];
        if (meta) {
          const photo = $('.photo-mini[data-category="' + meta.category + '"]', card);
          if (photo) photo.hidden = draft[field] === 0;
        }
        return;
      }
      const photoBtn = event.target.closest('.photo-mini');
      if (photoBtn) {
        const meta = CHECK_BY_CATEGORY[photoBtn.dataset.category];
        if (!meta) return;
        draft.photos.push({
          file_id: 'demo-photo-' + Date.now() + '-' + draft.photos.length,
          category: meta.category,
          ref_desc: meta.label + '异常',
          label: meta.label
        });
        const block = $('.photos-block', card);
        const photoList = $('.photo-list', card);
        if (photoList) photoList.innerHTML = photoListHTML(draft);
        if (block) block.hidden = !draft.photos.length;
        showToast('照片已保存');
        return;
      }
      const isolation = event.target.closest('.isolation-row');
      if (isolation) {
        draft.is_isolated = !draft.is_isolated;
        isolation.setAttribute('aria-checked', String(draft.is_isolated));
        return;
      }
      const retry = event.target.closest('.retry-link');
      if (retry) {
        submitCheck(id, draft.failedStatus);
        return;
      }
      const submit = event.target.closest('[data-action="submit"]');
      if (submit) submitCheck(id, Number(submit.dataset.status));
    });

    list.addEventListener('input', function (event) {
      const card = event.target.closest('.child-card');
      if (!card) return;
      const draft = draftFor(Number(card.dataset.id));
      if (draft.submitting) return;
      if (event.target.classList.contains('temp-input')) {
        draft.temperature = event.target.value;
        const hint = $('.temp-hint', card);
        const text = $('.temp-hint-text', card);
        const message = tempWarning(draft);
        if (text) text.textContent = message;
        if (hint) hint.hidden = !message;
      } else if (event.target.classList.contains('remark-input')) {
        draft.remark = event.target.value;
        const count = $('.remark-count', card);
        if (count) count.textContent = draft.remark.length + '/200';
      }
    });
  }

  buildFilter();
  bindEvents();
  setState('normal');
})();
