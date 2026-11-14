/* 请假申请 · 家长端原型交互 */
(() => {
  'use strict';

  const TODAY = '2026-10-26';
  const MIN_YEAR = 2026;
  const MIN_MONTH = 9; // 0-based, 2026 年 10 月
  const WEEKDAY_NAMES = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
  const VALID_STATES = ['normal', 'empty', 'loading', 'submitting', 'failed'];
  const SAMPLE = {
    type: 'sick',
    start: '2026-10-28',
    end: '2026-10-29',
    reason: '乐乐昨晚开始发烧 38.5℃，今天已到儿科门诊就诊，医生建议在家休息两天观察，特此请假。',
    name: '王芳',
    phone: '13812345678'
  };

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const refs = {
    form: document.getElementById('leaveForm'),
    reasonInput: document.getElementById('reasonInput'),
    pickupName: document.getElementById('pickupName'),
    pickupPhone: document.getElementById('pickupPhone'),
    startDateRow: document.getElementById('startDateRow'),
    endDateRow: document.getElementById('endDateRow'),
    startDateValue: document.getElementById('startDateValue'),
    startDateWeekday: document.getElementById('startDateWeekday'),
    endDateValue: document.getElementById('endDateValue'),
    endDateWeekday: document.getElementById('endDateWeekday'),
    startDateError: document.getElementById('startDateError'),
    endDateError: document.getElementById('endDateError'),
    daysTotal: document.getElementById('daysTotal'),
    daysNote: document.getElementById('daysNote'),
    reasonError: document.getElementById('reasonError'),
    reasonFoot: document.getElementById('reasonCharCount'),
    charCount: document.getElementById('charCount'),
    uploadGrid: document.getElementById('uploadGrid'),
    addPhotoBtn: document.getElementById('addPhotoBtn'),
    submitButton: document.getElementById('submitButton'),
    submitLabel: document.getElementById('submitLabel'),
    submitHint: document.getElementById('submitHint'),
    submitFailure: document.getElementById('submitFailure'),
    retryBtn: document.getElementById('retryBtn'),
    toast: document.getElementById('toast'),
    toastText: document.getElementById('toastText'),
    sheetLayer: document.getElementById('dateSheet'),
    sheetPanel: document.getElementById('dateSheetPanel'),
    dateSheetTitle: document.getElementById('dateSheetTitle'),
    prevMonth: document.getElementById('prevMonth'),
    nextMonth: document.getElementById('nextMonth'),
    calendarMonth: document.getElementById('calendarMonth'),
    calendarGrid: document.getElementById('calendarGrid'),
    cancelDate: document.getElementById('cancelDate'),
    confirmDate: document.getElementById('confirmDate')
  };

  if (!refs.form || !refs.submitButton) return;

  const typeTiles = Array.from(document.querySelectorAll('[data-leave-type]'));
  const demoChips = Array.from(document.querySelectorAll('.mp-demo-chip'));

  let formState = {
    type: '',
    start: '',
    end: '',
    reason: '',
    name: '',
    phone: '',
    photos: []
  };
  let errorsVisible = false;
  let sheetTarget = null;
  let sheetPending = null;
  let sheetReturnFocus = null;
  let sheetView = { year: MIN_YEAR, month: MIN_MONTH };
  let submitTimer = null;
  let toastTimer = null;
  let photoSeq = 0;

  bindEvents();
  fillSampleData();
  syncChips('normal');
  render();

  function bindEvents() {
    typeTiles.forEach((tile) => {
      tile.addEventListener('click', () => {
        if (isLocked() || !tile.dataset.leaveType) return;
        formState.type = tile.dataset.leaveType;
        syncTypeTiles();
      });
    });

    refs.reasonInput.addEventListener('input', () => {
      formState.reason = refs.reasonInput.value;
      render();
    });

    refs.pickupName.addEventListener('input', () => {
      formState.name = refs.pickupName.value;
    });

    refs.pickupPhone.addEventListener('input', () => {
      const digits = refs.pickupPhone.value.replace(/\D/g, '').slice(0, 11);
      if (refs.pickupPhone.value !== digits) refs.pickupPhone.value = digits;
      formState.phone = digits;
    });

    refs.startDateRow.addEventListener('click', () => openDateSheet('start'));
    refs.endDateRow.addEventListener('click', () => openDateSheet('end'));

    refs.form.addEventListener('submit', handleSubmit);
    refs.retryBtn.addEventListener('click', retrySubmission);

    refs.addPhotoBtn.addEventListener('click', addPhoto);
    refs.uploadGrid.addEventListener('click', handleUploadClick);

    refs.prevMonth.addEventListener('click', () => moveMonth(-1));
    refs.nextMonth.addEventListener('click', () => moveMonth(1));
    refs.calendarGrid.addEventListener('click', handleCalendarClick);
    refs.cancelDate.addEventListener('click', () => closeDateSheet(true));
    refs.confirmDate.addEventListener('click', confirmDateSelection);
    refs.sheetLayer.querySelectorAll('[data-close-sheet]').forEach((button) => {
      button.addEventListener('click', () => closeDateSheet(true));
    });

    document.addEventListener('mp:state', handleExternalState);
    document.addEventListener('keydown', handleDocumentKeydown);
  }

  function fillSampleData() {
    errorsVisible = false;
    formState = {
      type: SAMPLE.type,
      start: SAMPLE.start,
      end: SAMPLE.end,
      reason: SAMPLE.reason,
      name: SAMPLE.name,
      phone: SAMPLE.phone,
      photos: [createPhoto('病假条')]
    };
    syncFields();
    renderPhotos();
  }

  function fillEmptyData() {
    errorsVisible = true;
    formState = {
      type: '',
      start: '',
      end: '',
      reason: '',
      name: '',
      phone: '',
      photos: []
    };
    syncFields();
    renderPhotos();
  }

  function clearFormAfterSuccess() {
    errorsVisible = false;
    formState = {
      type: '',
      start: '',
      end: '',
      reason: '',
      name: '',
      phone: '',
      photos: []
    };
    syncFields();
    renderPhotos();
  }

  function createPhoto(label) {
    photoSeq += 1;
    return { id: photoSeq, label };
  }

  function syncFields() {
    refs.reasonInput.value = formState.reason;
    refs.pickupName.value = formState.name;
    refs.pickupPhone.value = formState.phone;
    syncTypeTiles();
  }

  function syncTypeTiles() {
    typeTiles.forEach((tile) => {
      const selected = tile.dataset.leaveType === formState.type;
      tile.classList.toggle('is-selected', selected);
      tile.setAttribute('aria-pressed', String(selected));
    });
  }

  function render() {
    const mode = app.dataset.state || 'normal';
    const locked = isLocked();
    const missing = getMissingFields();
    const complete = missing.length === 0;
    const days = getInclusiveDays();

    updateDateReadout(refs.startDateValue, refs.startDateWeekday, formState.start);
    updateDateReadout(refs.endDateValue, refs.endDateWeekday, formState.end);

    refs.daysTotal.textContent = days === null ? '共 — 天' : `共 ${days} 天`;
    refs.daysNote.textContent = days === null ? '选择日期后自动计算' : '含首尾两天';

    refs.charCount.textContent = String(formState.reason.length);
    refs.reasonFoot.classList.toggle('is-near', formState.reason.length >= 450);

    const showFieldErrors = errorsVisible && !locked;
    toggleError(refs.startDateError, refs.startDateRow, showFieldErrors && !formState.start);
    toggleError(refs.endDateError, refs.endDateRow, showFieldErrors && !formState.end);
    toggleError(refs.reasonError, refs.reasonInput, showFieldErrors && !formState.reason.trim());

    const disableFields = locked;
    typeTiles.forEach((tile) => { tile.disabled = disableFields; });
    refs.startDateRow.disabled = disableFields;
    refs.endDateRow.disabled = disableFields;
    refs.reasonInput.disabled = disableFields;
    refs.pickupName.disabled = disableFields;
    refs.pickupPhone.disabled = disableFields;

    refs.uploadGrid.querySelectorAll('[data-remove-photo]').forEach((button) => {
      button.disabled = disableFields;
    });
    refs.addPhotoBtn.disabled = disableFields;
    refs.addPhotoBtn.hidden = formState.photos.length >= 3;

    refs.submitButton.disabled = locked;
    refs.submitButton.setAttribute('aria-disabled', String(locked || !complete));
    refs.submitButton.classList.toggle('is-loading', mode === 'submitting');
    refs.submitButton.setAttribute('aria-busy', String(mode === 'submitting'));
    refs.submitLabel.textContent = mode === 'submitting' ? '提交中…' : '提交申请';
    refs.form.setAttribute('aria-busy', String(mode === 'submitting'));

    refs.submitFailure.hidden = mode !== 'failed';

    let hint = '';
    let blocking = false;
    if (mode === 'loading') {
      hint = '正在加载…';
    } else if (mode === 'submitting') {
      hint = '正在提交，请稍候';
    } else if (!complete) {
      hint = `还需填写：${missing.join('、')}`;
      blocking = true;
    } else {
      hint = `共 ${days || 0} 天 · 提交后进入待审批`;
    }
    refs.submitHint.textContent = hint;
    refs.submitHint.classList.toggle('is-blocking', blocking);
  }

  function updateDateReadout(valueEl, weekdayEl, iso) {
    if (iso) {
      valueEl.textContent = iso;
      valueEl.classList.add('is-set');
      weekdayEl.textContent = weekdayLabel(iso);
    } else {
      valueEl.textContent = '请选择';
      valueEl.classList.remove('is-set');
      weekdayEl.textContent = '';
    }
  }

  function toggleError(errorEl, controlEl, shouldShow) {
    if (!errorEl) return;
    errorEl.hidden = !shouldShow;
    if (controlEl) controlEl.setAttribute('aria-invalid', String(shouldShow));
  }

  function renderPhotos() {
    refs.uploadGrid.querySelectorAll('.photo-thumb').forEach((thumb) => thumb.remove());

    formState.photos.forEach((photo, index) => {
      const thumb = document.createElement('div');
      thumb.className = 'photo-thumb';
      thumb.dataset.photoId = String(photo.id);
      thumb.dataset.odId = `attachment-photo-${photo.id}`;
      thumb.innerHTML = [
        '<button class="remove-photo" type="button">',
        '<svg width="10" height="10" viewBox="0 0 10 10" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" aria-hidden="true"><path d="m2 2 6 6M8 2 2 8"/></svg>',
        '</button>',
        '<svg class="photo-icon" width="26" height="26" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="2.5" width="18" height="19" rx="3"/><path d="m6.5 16 3.3-3.7 2.7 2.7 2.2-2.2 2.8 3.2"/><circle cx="8.5" cy="8" r="1.3"/></svg>',
        '<span class="photo-name"></span>'
      ].join('');

      const removeButton = thumb.querySelector('.remove-photo');
      const name = photo.label || `照片 ${index + 1}`;
      removeButton.dataset.removePhoto = String(photo.id);
      removeButton.setAttribute('aria-label', `删除附件 ${name}`);
      removeButton.disabled = isLocked();
      thumb.querySelector('.photo-name').textContent = name;

      refs.uploadGrid.insertBefore(thumb, refs.addPhotoBtn);
    });

    refs.addPhotoBtn.hidden = formState.photos.length >= 3;
    refs.addPhotoBtn.disabled = isLocked();
  }

  function addPhoto() {
    if (isLocked() || formState.photos.length >= 3) return;
    formState.photos.push(createPhoto(`照片 ${formState.photos.length + 1}`));
    renderPhotos();
    render();
  }

  function handleUploadClick(event) {
    const removeButton = event.target.closest('[data-remove-photo]');
    if (!removeButton || isLocked()) return;

    const photoId = Number(removeButton.dataset.removePhoto);
    if (!Number.isFinite(photoId)) return;
    const shouldRestoreFocus = document.activeElement === removeButton;

    formState.photos = formState.photos.filter((photo) => photo.id !== photoId);
    renderPhotos();
    render();

    if (shouldRestoreFocus) {
      const nextTarget = refs.uploadGrid.querySelector('[data-remove-photo]') || refs.addPhotoBtn;
      if (nextTarget && !nextTarget.hidden) nextTarget.focus({ preventScroll: true });
    }
  }

  function handleSubmit(event) {
    event.preventDefault();
    if (isLocked()) return;

    errorsVisible = true;
    if (getMissingFields().length > 0) {
      render();
      return;
    }

    startSubmission();
  }

  function startSubmission() {
    clearSubmitTimer();
    errorsVisible = false;
    app.dataset.state = 'submitting';
    syncChips('submitting');
    syncDemoOffset('submitting');
    render();

    submitTimer = window.setTimeout(() => {
      submitTimer = null;
      clearFormAfterSuccess();
      app.dataset.state = 'normal';
      syncChips('normal');
      syncDemoOffset('normal');
      render();
      showToast('提交成功');
    }, 1400);
  }

  function retrySubmission() {
    if (app.dataset.state !== 'failed' || isLocked()) return;
    clearSubmitTimer();
    errorsVisible = false;
    app.dataset.state = 'normal';
    syncChips('normal');
    syncDemoOffset('normal');
    render();
    showToast('已重新提交');
    if (!refs.submitButton.disabled) {
      refs.submitButton.focus({ preventScroll: true });
    }
  }

  function handleExternalState(event) {
    const requested = event && event.detail ? event.detail.state : '';
    if (!VALID_STATES.includes(requested)) return;

    clearSubmitTimer();
    hideToast();
    if (!refs.sheetLayer.hidden) closeDateSheet(false);

    errorsVisible = requested === 'empty';

    if (requested === 'empty') {
      fillEmptyData();
    } else if (requested === 'normal' || requested === 'submitting' || requested === 'failed') {
      fillSampleData();
    }

    app.dataset.state = requested;
    syncChips(requested);
    syncDemoOffset(requested);
    render();
  }

  // The failed state grows the submit dock by the failure bar, so the demo
  // bar has to move up with it; 120px clears the normal dock only.
  function syncDemoOffset(state) {
    const demo = document.querySelector('.mp-demo');
    if (!demo) return;
    demo.style.bottom = state === 'failed'
      ? 'calc(176px + env(safe-area-inset-bottom))'
      : 'calc(120px + env(safe-area-inset-bottom))';
  }

  function syncChips(state) {
    demoChips.forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
  }

  function isLocked() {
    const mode = app.dataset.state;
    return mode === 'submitting' || mode === 'loading';
  }

  function getMissingFields() {
    const missing = [];
    if (!formState.start) missing.push('开始日期');
    if (!formState.end) missing.push('结束日期');
    if (!formState.reason.trim()) missing.push('请假原因');
    return missing;
  }

  function openDateSheet(target) {
    if (isLocked() || sheetTarget) return;

    sheetTarget = target;
    sheetReturnFocus = target === 'start' ? refs.startDateRow : refs.endDateRow;

    let currentValue = target === 'start' ? formState.start : formState.end;
    const minDate = getSheetMinDate(target);
    if (currentValue && currentValue < minDate) currentValue = '';
    sheetPending = currentValue || null;

    const anchor = sheetPending
      || (target === 'end' && formState.start ? formState.start : TODAY);
    const anchorDate = parseISODate(anchor);
    sheetView = normalizeViewMonth(anchorDate.getFullYear(), anchorDate.getMonth());

    refs.dateSheetTitle.textContent = target === 'start' ? '开始日期' : '结束日期';
    refs.calendarGrid.setAttribute(
      'aria-label',
      target === 'start' ? '选择开始日期' : '选择结束日期'
    );
    refs.sheetLayer.hidden = false;
    document.body.classList.add('sheet-open');
    renderCalendar();
    syncConfirmState();

    window.requestAnimationFrame(() => {
      if (!refs.sheetLayer.hidden) refs.sheetPanel.focus({ preventScroll: true });
    });
  }

  function getSheetMinDate(target) {
    if (target === 'end' && formState.start) {
      return formState.start > TODAY ? formState.start : TODAY;
    }
    return TODAY;
  }

  function normalizeViewMonth(year, month) {
    if (year < MIN_YEAR || (year === MIN_YEAR && month < MIN_MONTH)) {
      return { year: MIN_YEAR, month: MIN_MONTH };
    }
    return { year, month };
  }

  function renderCalendar() {
    if (!sheetTarget) return;

    const year = sheetView.year;
    const month = sheetView.month;
    const daysInMonth = new Date(year, month + 1, 0).getDate();
    const firstWeekday = new Date(year, month, 1).getDay();
    const leadingBlanks = (firstWeekday + 6) % 7;
    const minDate = getSheetMinDate(sheetTarget);
    const fragment = document.createDocumentFragment();

    refs.calendarMonth.textContent = `${year}年${month + 1}月`;
    refs.prevMonth.disabled = year === MIN_YEAR && month === MIN_MONTH;

    for (let i = 0; i < leadingBlanks; i += 1) {
      const blank = document.createElement('span');
      blank.className = 'calendar-blank';
      blank.setAttribute('aria-hidden', 'true');
      fragment.appendChild(blank);
    }

    for (let day = 1; day <= daysInMonth; day += 1) {
      const iso = toISODate(year, month, day);
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'calendar-day';
      button.dataset.date = iso;
      button.textContent = String(day);
      button.setAttribute('aria-pressed', String(sheetPending === iso));
      button.setAttribute(
        'aria-label',
        `${year}年${month + 1}月${day}日 ${weekdayLabel(iso)}${iso === TODAY ? '，今天' : ''}`
      );

      if (iso === TODAY) {
        button.classList.add('is-today');
        button.setAttribute('aria-current', 'date');
      }
      if (sheetPending === iso) button.classList.add('is-selected');
      if (iso < minDate) button.disabled = true;

      fragment.appendChild(button);
    }

    refs.calendarGrid.replaceChildren(fragment);
    syncConfirmState();
  }

  function handleCalendarClick(event) {
    const target = event.target;
    if (!target || typeof target.closest !== 'function') return;
    const dayButton = target.closest('.calendar-day');
    if (!dayButton || dayButton.disabled || !dayButton.dataset.date) return;

    sheetPending = dayButton.dataset.date;
    refs.calendarGrid.querySelectorAll('.calendar-day').forEach((button) => {
      const isSelected = button.dataset.date === sheetPending;
      button.classList.toggle('is-selected', isSelected);
      button.setAttribute('aria-pressed', String(isSelected));
    });
    syncConfirmState();
  }

  function syncConfirmState() {
    const minDate = sheetTarget ? getSheetMinDate(sheetTarget) : TODAY;
    refs.confirmDate.disabled = !sheetPending || sheetPending < minDate;
  }

  function moveMonth(delta) {
    if (!sheetTarget) return;

    const next = new Date(sheetView.year, sheetView.month + delta, 1);
    const minimum = new Date(MIN_YEAR, MIN_MONTH, 1);
    if (next < minimum) return;

    sheetView = { year: next.getFullYear(), month: next.getMonth() };
    sheetPending = null;
    renderCalendar();
  }

  function confirmDateSelection() {
    if (!sheetTarget || !sheetPending || refs.confirmDate.disabled) return;

    const target = sheetTarget;
    const pickedDate = sheetPending;
    const returnTarget = sheetReturnFocus;

    if (target === 'start') {
      formState.start = pickedDate;
      if (formState.end && formState.end < formState.start) {
        formState.end = formState.start;
      }
    } else {
      formState.end = pickedDate;
    }

    closeDateSheet(false);
    render();

    if (returnTarget && !returnTarget.disabled) {
      returnTarget.focus({ preventScroll: true });
    }
  }

  function closeDateSheet(restoreFocus) {
    if (refs.sheetLayer.hidden) return;

    const returnTarget = sheetReturnFocus;
    refs.sheetLayer.hidden = true;
    document.body.classList.remove('sheet-open');
    sheetTarget = null;
    sheetPending = null;
    sheetReturnFocus = null;

    if (restoreFocus && returnTarget && !returnTarget.disabled) {
      returnTarget.focus({ preventScroll: true });
    }
  }

  function handleDocumentKeydown(event) {
    if (refs.sheetLayer.hidden) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      closeDateSheet(true);
      return;
    }

    if (event.key !== 'Tab') return;

    const focusable = Array.from(
      refs.sheetPanel.querySelectorAll('button:not([disabled])')
    ).filter((element) => element.offsetParent !== null);

    if (focusable.length === 0) return;

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    const active = document.activeElement;

    if (event.shiftKey && (active === first || active === refs.sheetPanel)) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && active === last) {
      event.preventDefault();
      first.focus();
    }
  }

  function showToast(message) {
    if (!refs.toast || !refs.toastText) return;
    window.clearTimeout(toastTimer);
    refs.toastText.textContent = message;
    refs.toast.hidden = false;
    toastTimer = window.setTimeout(() => {
      toastTimer = null;
      refs.toast.hidden = true;
    }, 2200);
  }

  function hideToast() {
    window.clearTimeout(toastTimer);
    toastTimer = null;
    if (refs.toast) refs.toast.hidden = true;
  }

  function clearSubmitTimer() {
    if (submitTimer !== null) {
      window.clearTimeout(submitTimer);
      submitTimer = null;
    }
  }

  function getInclusiveDays() {
    if (!formState.start || !formState.end) return null;
    if (formState.end < formState.start) return null;
    const diff = parseISODate(formState.end).getTime() - parseISODate(formState.start).getTime();
    return Math.round(diff / 86400000) + 1;
  }

  function parseISODate(iso) {
    const parts = iso.split('-').map(Number);
    return new Date(parts[0], parts[1] - 1, parts[2], 12, 0, 0, 0);
  }

  function toISODate(year, month, day) {
    return [
      String(year),
      String(month + 1).padStart(2, '0'),
      String(day).padStart(2, '0')
    ].join('-');
  }

  function weekdayLabel(iso) {
    return WEEKDAY_NAMES[parseISODate(iso).getDay()];
  }
})();
