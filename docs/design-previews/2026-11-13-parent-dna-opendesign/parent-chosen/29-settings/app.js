/* 设置 · 简化提案 — 交互与文档正文 */
(() => {
  'use strict';

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];
  const KEY = 'nurture-parent-settings-v4';

  /* ---------- 设置状态 ---------- */
  const state = { school: true, teacher: true, quiet: true, phone: true, quietStart: '20:00', quietEnd: '07:30' };
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) || 'null');
    if (saved && typeof saved === 'object') {
      ['school', 'teacher', 'quiet', 'phone'].forEach((k) => {
        if (typeof saved[k] === 'boolean') state[k] = saved[k];
      });
      ['quietStart', 'quietEnd'].forEach((k) => {
        if (/^\d{2}:\d{2}$/.test(saved[k])) state[k] = saved[k];
      });
    }
  } catch (_) { /* 隐私模式下不持久化 */ }
  const persist = () => {
    try { localStorage.setItem(KEY, JSON.stringify(state)); } catch (_) { /* 忽略 */ }
  };

  const app = $('#app');
  const settingsView = $('#view-settings');
  const layer = $('#sheet-layer');
  const toastEl = $('#toast');
  const schoolTag = $('#row-school-tag');
  const teacherTag = $('#row-teacher-tag');
  const quietValue = $('#row-quiet-value');
  const phoneSwitch = $('#phone-visibility');
  const updateStatus = $('#update-status');

  const paintSettings = () => {
    [[schoolTag, state.school], [teacherTag, state.teacher]].forEach(([el, on]) => {
      el.textContent = on ? '已开启' : '已关闭';
      el.classList.toggle('state-tag--off', !on);
    });
    quietValue.textContent = state.quiet ? `${state.quietStart}–${state.quietEnd}` : '未开启';
    phoneSwitch.checked = state.phone;
  };
  paintSettings();

  phoneSwitch.addEventListener('change', () => {
    state.phone = phoneSwitch.checked;
    persist();
    toast(state.phone ? '老师可以看到你的手机号' : '已对老师隐藏手机号');
  });

  /* ---------- 轻提示 ---------- */
  let toastTimer = 0;
  const toast = (message) => {
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toastEl.classList.remove('is-visible'), 2000);
  };
  const hideToast = () => {
    clearTimeout(toastTimer);
    toastEl.classList.remove('is-visible');
  };

  /* ---------- 底部面板 ---------- */
  const sheets = $$('.sheet');
  let currentSheet = null;
  let lastFocus = null;
  let closeTimer = 0;

  const setInert = (on) => {
    if ('inert' in settingsView) {
      settingsView.inert = on;
    }
  };

  const openSheet = (sheet, trigger) => {
    if (!sheet) return;
    hideToast();
    clearTimeout(closeTimer);
    lastFocus = trigger || document.activeElement;
    sheets.forEach((s) => { if (s !== sheet) s.hidden = true; });
    sheet.hidden = false;
    layer.hidden = false;
    app.classList.add('is-sheet-open');
    setInert(true);
    void layer.offsetWidth;
    layer.classList.add('is-open');
    currentSheet = sheet;
    sheet.focus({ preventScroll: true });
  };

  const closeSheet = (restore = true, done = null) => {
    if (!currentSheet) return;
    const sheet = currentSheet;
    currentSheet = null;
    layer.classList.remove('is-open');
    app.classList.remove('is-sheet-open');
    setInert(false);
    clearTimeout(closeTimer);
    closeTimer = setTimeout(() => {
      sheet.hidden = true;
      layer.hidden = true;
      if (restore && lastFocus) lastFocus.focus({ preventScroll: true });
      if (done) done();
    }, 210);
  };

  document.addEventListener('keydown', (e) => {
    if (!currentSheet) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      closeSheet();
      return;
    }
    if (e.key !== 'Tab') return;
    const focusables = $$('button:not([disabled]),input:not([disabled]),[href],[tabindex]:not([tabindex="-1"])', currentSheet)
      .filter((el) => el.offsetParent !== null);
    if (!focusables.length) {
      e.preventDefault();
      currentSheet.focus();
      return;
    }
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    if (e.shiftKey && (document.activeElement === first || document.activeElement === currentSheet)) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  });

  /* ---------- 消息通知 ---------- */
  const notifySheet = $('#sheet-notify');
  const notifyTitle = $('#notify-sheet-title');
  const notifyDesc = $('#notify-sheet-desc');
  const notifyLabel = $('#notify-switch-label');
  const notifySwitch = $('#notify-switch');
  const NOTIFY_COPY = {
    school: {
      title: '园所通知',
      desc: '园所公告、活动与审批结果会通过微信服务通知送达。关闭后仍可在消息页查看。',
      label: '接收园所通知'
    },
    teacher: {
      title: '老师消息',
      desc: '老师的私信与回复会通过微信服务通知提醒。关闭后仍可在消息页查看。',
      label: '接收老师消息'
    }
  };
  let notifyKind = 'school';
  let notifyDraft = true;

  const openNotify = (kind, trigger) => {
    notifyKind = kind;
    notifyDraft = state[kind];
    const copy = NOTIFY_COPY[kind];
    notifyTitle.textContent = copy.title;
    notifyDesc.textContent = copy.desc;
    notifyLabel.textContent = copy.label;
    notifySwitch.setAttribute('aria-checked', String(notifyDraft));
    openSheet(notifySheet, trigger);
  };
  notifySwitch.addEventListener('click', () => {
    notifyDraft = !notifyDraft;
    notifySwitch.setAttribute('aria-checked', String(notifyDraft));
  });
  $('#notify-save').addEventListener('click', () => {
    state[notifyKind] = notifyDraft;
    persist();
    paintSettings();
    closeSheet(true, () => toast(notifyDraft ? '通知已开启' : '通知已关闭'));
  });

  /* ---------- 免打扰时段 ---------- */
  const quietSheet = $('#sheet-quiet');
  const quietSwitch = $('#quiet-switch');
  const startInput = $('#quiet-start');
  const endInput = $('#quiet-end');
  const quietError = $('#quiet-error');
  const presets = $$('.preset', $('#quiet-presets'));
  let quietDraft = true;

  const paintQuiet = () => {
    quietSwitch.setAttribute('aria-checked', String(quietDraft));
    startInput.disabled = !quietDraft;
    endInput.disabled = !quietDraft;
    presets.forEach((p) => {
      p.disabled = !quietDraft;
      p.setAttribute('aria-pressed', String(
        p.dataset.start === startInput.value && p.dataset.end === endInput.value
      ));
    });
  };
  const openQuiet = (trigger) => {
    quietDraft = state.quiet;
    startInput.value = state.quietStart;
    endInput.value = state.quietEnd;
    quietError.hidden = true;
    startInput.removeAttribute('aria-invalid');
    endInput.removeAttribute('aria-invalid');
    paintQuiet();
    openSheet(quietSheet, trigger);
  };
  const quietErrorShow = (message, input) => {
    quietError.textContent = message;
    quietError.hidden = false;
    input.setAttribute('aria-invalid', 'true');
    input.focus({ preventScroll: true });
  };
  quietSwitch.addEventListener('click', () => {
    quietDraft = !quietDraft;
    quietError.hidden = true;
    paintQuiet();
  });
  [startInput, endInput].forEach((input) => {
    input.addEventListener('input', () => {
      quietError.hidden = true;
      input.removeAttribute('aria-invalid');
      paintQuiet();
    });
  });
  presets.forEach((p) => {
    p.addEventListener('click', () => {
      if (p.disabled) return;
      startInput.value = p.dataset.start;
      endInput.value = p.dataset.end;
      quietError.hidden = true;
      paintQuiet();
    });
  });
  $('#quiet-save').addEventListener('click', () => {
    const start = startInput.value;
    const end = endInput.value;
    if (quietDraft && (!start || !end)) {
      quietErrorShow('请选择开始和结束时间', !start ? startInput : endInput);
      return;
    }
    if (quietDraft && start === end) {
      quietErrorShow('开始和结束时间不能相同', startInput);
      return;
    }
    state.quiet = quietDraft;
    if (start && end) {
      state.quietStart = start;
      state.quietEnd = end;
    }
    persist();
    paintSettings();
    closeSheet(true, () => toast(quietDraft ? `免打扰已设为 ${state.quietStart}–${state.quietEnd}` : '免打扰已关闭'));
  });

  /* ---------- 注销账号 ---------- */
  const ack = $('#delete-ack');
  const deleteSubmit = $('#delete-submit');
  const deleteStatus = $('#delete-status');
  const resetDelete = () => {
    ack.checked = false;
    deleteSubmit.disabled = true;
    deleteStatus.hidden = true;
  };
  ack.addEventListener('change', () => {
    deleteSubmit.disabled = !ack.checked;
  });
  deleteSubmit.addEventListener('click', () => {
    if (!ack.checked) return;
    deleteStatus.textContent = 'Phase 2 接口未接入 · 本次未提交';
    deleteStatus.hidden = false;
    deleteSubmit.disabled = true;
  });

  /* ---------- 检查更新 ---------- */
  let updateTimer = 0;
  const checkUpdate = (row) => {
    if (row.disabled) return;
    row.disabled = true;
    updateStatus.textContent = '正在检查';
    clearTimeout(updateTimer);
    updateTimer = setTimeout(() => {
      updateStatus.textContent = '已是最新';
      row.disabled = false;
      if (!currentSheet) toast('已是最新版本');
    }, 1400);
  };
  updateStatus.setAttribute('aria-live', 'polite');

  /* ---------- 动作分发 ---------- */
  document.addEventListener('click', (e) => {
    const target = e.target.closest('[data-action]');
    if (!target) return;
    switch (target.dataset.action) {
      case 'back-settings':
        toast('已在设置首页');
        break;
      case 'close-sheet':
        closeSheet();
        break;
      case 'notify-school':
        openNotify('school', target);
        break;
      case 'notify-teacher':
        openNotify('teacher', target);
        break;
      case 'quiet':
        openQuiet(target);
        break;
      case 'child':
        openSheet($('#sheet-child'), target);
        break;
      case 'delete':
        resetDelete();
        openSheet($('#sheet-delete'), target);
        break;
      case 'logout':
        openSheet($('#sheet-logout'), target);
        break;
      case 'check-update':
        checkUpdate(target);
        break;
      default:
        break;
    }
  });

  $('#logout-confirm').addEventListener('click', () => {
    closeSheet(true, () => toast('已退出登录'));
  });
})();
