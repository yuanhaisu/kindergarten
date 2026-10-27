/* 添加授权接送人 · 家长端（子页）
 * 交互：关系底部选择面板、权限开关、就地校验、提交中 / 失败 / 重试、状态芯片联动。
 * 芯片切换由 ../../_shared/shell.js 负责，本文件只监听 mp:state，不重复绑定点击。
 */
(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  var formArea = document.getElementById('formArea');
  var nameInput = document.getElementById('inputName');
  var phoneInput = document.getElementById('inputPhone');
  var idInput = document.getElementById('inputId');
  var nameField = document.getElementById('fieldName');
  var phoneField = document.getElementById('fieldPhone');
  var nameError = document.getElementById('nameError');
  var phoneError = document.getElementById('phoneError');
  var relationBtn = document.getElementById('relationBtn');
  var relationValue = document.getElementById('relationValue');
  var permRows = Array.prototype.slice.call(document.querySelectorAll('.perm-row'));
  var noteEl = document.getElementById('submitNote');
  var submitBtn = document.getElementById('submitBtn');
  var retryBtn = document.getElementById('retryBtn');
  var sheetLayer = document.getElementById('sheetLayer');
  var sheetPanel = document.getElementById('sheetPanel');
  var sheetOptions = document.getElementById('sheetOptions');
  var sheetCancel = document.getElementById('sheetCancel');
  var sheetScrim = document.getElementById('sheetScrim');
  var toastEl = document.getElementById('toast');
  var demoEl = document.querySelector('.mp-demo');
  var backBtn = document.querySelector('.mp-back');

  var SAMPLE = {
    name: '王建国',
    phone: '13812345678',
    id: '310101195503124012',
    relation: '爷爷'
  };

  var DEFAULT_PERMS = {
    allow_pickup: true,
    allow_view_record: false,
    allow_view_status: false,
    allow_growth_view: false,
    notify_on_pickup: true
  };

  var APPROVAL_NOTE = '提交后需园所审核，通过后才可接送';
  var reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

  var relation = SAMPLE.relation;
  var isBusy = false;
  var busyTimer = null;
  var toastTimer = null;
  var closeTimer = null;

  /* ---- 校验与常驻说明 -------------------------------------------------- */

  function setNote(text, blocked) {
    if (noteEl.textContent !== text) noteEl.textContent = text;
    noteEl.classList.toggle('is-blocked', Boolean(blocked));
  }

  function setFieldError(key, on) {
    var field = key === 'name' ? nameField : phoneField;
    var input = key === 'name' ? nameInput : phoneInput;
    var error = key === 'name' ? nameError : phoneError;
    field.classList.toggle('is-error', on);
    input.setAttribute('aria-invalid', String(on));
    error.hidden = !on;
  }

  function clearErrors() {
    setFieldError('name', false);
    setFieldError('phone', false);
  }

  function isNameValid() {
    return nameInput.value.trim().length > 0;
  }

  function isPhoneValid() {
    return /^\d{11}$/.test(phoneInput.value.trim());
  }

  function getMissing() {
    var missing = [];
    if (!isNameValid()) missing.push('姓名');
    if (!isPhoneValid()) missing.push('手机号');
    if (!relation) missing.push('关系');
    return missing;
  }

  function updateBlocking() {
    var state = app.dataset.state;
    if (state === 'loading' || state === 'submitting') return;

    var missing = getMissing();
    var complete = missing.length === 0;

    setNote(complete ? APPROVAL_NOTE : '还需填写：' + missing.join('、'), !complete);
    submitBtn.classList.toggle('is-disabled', !complete);
    submitBtn.setAttribute('aria-disabled', String(!complete));
  }

  /* ---- 表单数据 -------------------------------------------------------- */

  function setRelationUI(value) {
    relation = value;
    relationValue.textContent = value;
    relationBtn.dataset.empty = value ? 'false' : 'true';
    Array.prototype.forEach.call(sheetOptions.children, function (option) {
      var on = option.dataset.value === value;
      option.classList.toggle('is-selected', on);
      option.setAttribute('aria-selected', String(on));
    });
  }

  function restoreSample() {
    nameInput.value = SAMPLE.name;
    phoneInput.value = SAMPLE.phone;
    idInput.value = SAMPLE.id;
    setRelationUI(SAMPLE.relation);
    permRows.forEach(function (row) {
      row.setAttribute('aria-checked', String(DEFAULT_PERMS[row.dataset.perm] === true));
    });
  }

  function applyEmpty() {
    nameInput.value = '';
    phoneInput.value = '';
    setRelationUI(SAMPLE.relation);
    setFieldError('name', true);
    setFieldError('phone', true);
  }

  function setControlsDisabled(disabled) {
    nameInput.disabled = disabled;
    phoneInput.disabled = disabled;
    idInput.disabled = disabled;
    relationBtn.disabled = disabled;
    permRows.forEach(function (row) {
      row.disabled = disabled;
    });
  }

  /* ---- 演示状态 -------------------------------------------------------- */

  function syncDemo(state) {
    if (!demoEl) return;
    demoEl.style.bottom = state === 'failed'
      ? 'calc(164px + env(safe-area-inset-bottom))'
      : 'calc(112px + env(safe-area-inset-bottom))';
  }

  function syncChips(state) {
    Array.prototype.forEach.call(document.querySelectorAll('.mp-demo-chip'), function (chip) {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
  }

  function applyState(state) {
    window.clearTimeout(busyTimer);
    isBusy = false;
    formArea.classList.remove('is-busy');
    submitBtn.classList.remove('is-busy');

    if (state !== 'normal') closeSheet(false);

    if (state === 'loading') {
      setControlsDisabled(true);
      submitBtn.classList.add('is-disabled');
      submitBtn.setAttribute('aria-disabled', 'true');
      setNote('正在加载…', false);
      syncDemo(state);
      return;
    }

    if (state === 'submitting') {
      setControlsDisabled(true);
      submitBtn.classList.remove('is-disabled');
      submitBtn.setAttribute('aria-disabled', 'true');
      setNote('正在提交，请稍候', false);
      syncDemo(state);
      return;
    }

    setControlsDisabled(false);

    if (state === 'normal') {
      restoreSample();
      clearErrors();
    } else if (state === 'empty') {
      applyEmpty();
    } else if (state === 'failed') {
      if (getMissing().length) restoreSample();
      clearErrors();
    }

    updateBlocking();
    syncDemo(state);
  }

  /* ---- 关系选择面板 ---------------------------------------------------- */

  function openSheet() {
    if (relationBtn.disabled) return;

    window.clearTimeout(closeTimer);
    sheetLayer.hidden = false;
    relationBtn.setAttribute('aria-expanded', 'true');

    var selected = sheetOptions.querySelector('.sheet-option.is-selected') ||
      sheetOptions.querySelector('.sheet-option');
    if (selected) selected.focus({ preventScroll: true });

    window.requestAnimationFrame(function () {
      window.requestAnimationFrame(function () {
        if (!sheetLayer.hidden) sheetLayer.dataset.open = 'true';
      });
    });
  }

  function closeSheet(restoreFocus) {
    if (sheetLayer.hidden) return;

    sheetLayer.dataset.open = 'false';
    relationBtn.setAttribute('aria-expanded', 'false');

    window.clearTimeout(closeTimer);
    closeTimer = window.setTimeout(function () {
      if (sheetLayer.dataset.open !== 'true') sheetLayer.hidden = true;
    }, reducedMotion.matches ? 0 : 240);

    if (restoreFocus !== false) relationBtn.focus({ preventScroll: true });
  }

  function selectRelation(value) {
    setRelationUI(value);
    closeSheet();
    updateBlocking();
  }

  relationBtn.addEventListener('click', openSheet);
  sheetScrim.addEventListener('click', function () {
    closeSheet();
  });
  sheetCancel.addEventListener('click', function () {
    closeSheet();
  });

  sheetOptions.addEventListener('click', function (event) {
    var option = event.target.closest('.sheet-option');
    if (!option) return;
    selectRelation(option.dataset.value);
  });

  sheetOptions.addEventListener('keydown', function (event) {
    if (event.key !== 'ArrowDown' && event.key !== 'ArrowUp') return;
    var options = Array.prototype.slice.call(sheetOptions.querySelectorAll('.sheet-option'));
    var index = options.indexOf(document.activeElement);
    if (index === -1) return;
    event.preventDefault();
    var next = event.key === 'ArrowDown'
      ? (index + 1) % options.length
      : (index - 1 + options.length) % options.length;
    options[next].focus({ preventScroll: true });
  });

  document.addEventListener('keydown', function (event) {
    if (sheetLayer.hidden) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (event.key !== 'Tab') return;
    var focusables = Array.prototype.slice.call(
      sheetPanel.querySelectorAll('button:not([tabindex="-1"])')
    );
    if (!focusables.length) return;
    var first = focusables[0];
    var last = focusables[focusables.length - 1];

    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });

  /* ---- 权限开关 -------------------------------------------------------- */

  permRows.forEach(function (row) {
    row.addEventListener('click', function () {
      if (row.disabled) return;
      row.setAttribute('aria-checked', row.getAttribute('aria-checked') === 'true' ? 'false' : 'true');
    });
  });

  /* ---- 输入 ------------------------------------------------------------ */

  nameInput.addEventListener('input', function () {
    if (nameField.classList.contains('is-error')) setFieldError('name', false);
    updateBlocking();
  });

  phoneInput.addEventListener('input', function () {
    var digits = phoneInput.value.replace(/\D/g, '').slice(0, 11);
    if (digits !== phoneInput.value) phoneInput.value = digits;
    if (phoneField.classList.contains('is-error')) setFieldError('phone', false);
    updateBlocking();
  });

  idInput.addEventListener('input', function () {
    var cleaned = idInput.value.replace(/[^0-9xX]/g, '').slice(0, 18).toUpperCase();
    if (cleaned !== idInput.value) idInput.value = cleaned;
  });

  /* ---- Toast ----------------------------------------------------------- */

  function toast(message) {
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toastEl.classList.remove('is-visible');
    }, 2000);
  }

  /* ---- 提交 ------------------------------------------------------------ */

  function setBusy(duration, done) {
    if (isBusy) return;
    isBusy = true;
    window.clearTimeout(busyTimer);

    formArea.classList.add('is-busy');
    submitBtn.classList.add('is-busy');
    submitBtn.classList.remove('is-disabled');
    submitBtn.setAttribute('aria-disabled', 'true');
    setNote('正在提交，请稍候', false);
    setControlsDisabled(true);
    closeSheet(false);

    busyTimer = window.setTimeout(function () {
      isBusy = false;
      formArea.classList.remove('is-busy');
      submitBtn.classList.remove('is-busy');
      setControlsDisabled(false);
      updateBlocking();
      if (typeof done === 'function') done();
    }, duration);
  }

  submitBtn.addEventListener('click', function () {
    var state = app.dataset.state;
    if (isBusy || state === 'submitting' || state === 'loading') return;

    // 失败态下重新提交：先退出 failed（与 retryBtn 一致），
    // 否则失败条会与「已提交」toast 同屏，状态自相矛盾。
    if (state === 'failed') {
      app.dataset.state = 'normal';
      syncChips('normal');
      syncDemo('normal');
    }

    var missing = getMissing();
    if (missing.length) {
      if (missing.indexOf('姓名') !== -1) setFieldError('name', true);
      if (missing.indexOf('手机号') !== -1) setFieldError('phone', true);
      updateBlocking();

      if (missing.indexOf('姓名') !== -1) nameInput.focus();
      else if (missing.indexOf('手机号') !== -1) phoneInput.focus();
      else relationBtn.focus();
      return;
    }

    setBusy(1400, function () {
      toast('已提交，待园所审核');
    });
  });

  retryBtn.addEventListener('click', function () {
    if (isBusy) return;
    app.dataset.state = 'normal';
    syncChips('normal');
    syncDemo('normal');
    setBusy(1200, function () {
      toast('已提交，待园所审核');
    });
  });

  if (backBtn) {
    backBtn.addEventListener('click', function () {
      toast('已返回授权接送人');
    });
  }

  /* ---- 演示状态联动 ---------------------------------------------------- */

  document.addEventListener('mp:state', function (event) {
    var state = event.detail && event.detail.state ? event.detail.state : 'normal';
    applyState(state);
  });

  /* ---- 初始化 ---------------------------------------------------------- */

  restoreSample();
  clearErrors();
  updateBlocking();
})();
