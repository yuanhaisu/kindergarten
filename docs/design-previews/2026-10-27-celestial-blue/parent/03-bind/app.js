(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  var form = document.getElementById('bindForm');
  if (!app || !form) return;

  var SAMPLE_CODE = 'KP20241015';
  var REJECTED_CODE = '000000';
  var DEMO_ERROR_CODE = 'KP20241016';
  var MIN_LENGTH = 6;
  var MAX_LENGTH = 20;
  var VALID_STATES = ['normal', 'empty', 'loading', 'error'];
  var COPY = {
    helper: '邀请码由班级老师提供，每位孩子一个',
    missing: '还差邀请码',
    short: '邀请码至少 6 位',
    notFound: '未找到该学生，请检查邀请码',
    emptySubmit: '请输入邀请码',
    returning: '返回上一页'
  };

  var input = document.getElementById('inviteCode');
  var codeCount = document.getElementById('codeCount');
  var fieldReport = document.getElementById('fieldReport');
  var fieldReportText = document.getElementById('fieldReportText');
  var infoIcon = document.querySelector('.report-icon-info');
  var errorIcon = document.querySelector('.report-icon-error');
  var submitBtn = document.getElementById('submitBtn');
  var scanBtn = document.getElementById('scanBtn');
  var refillBtn = document.getElementById('refillBtn');
  var ctaHint = document.getElementById('ctaHint');
  var ctaHintText = document.getElementById('ctaHintText');
  var backBtn = document.getElementById('backBtn');
  var scanScreen = document.getElementById('scanScreen');
  var scanClose = document.getElementById('scanClose');
  var successToast = document.getElementById('successToast');
  var simpleToast = document.getElementById('simpleToast');
  var stateChips = Array.prototype.slice.call(document.querySelectorAll('.mp-demo-chip'));

  var state = 'normal';
  var errorMessage = '';
  // Which kind of error is on screen: 'client' = too short (user is still
  // typing), 'server' = the backend rejected the code. Only the server kind
  // warrants "重新填写"; clearing the field never helps a half-typed code.
  var errorKind = '';
  var hintMessage = '';
  var lastFocused = null;
  var timers = {
    submit: 0,
    scan: 0,
    success: 0,
    successHide: 0,
    toast: 0,
    toastHide: 0,
    back: 0
  };

  function pad2(number) {
    return (number < 10 ? '0' : '') + number;
  }

  function normalizeCode(value) {
    return String(value || '').toUpperCase().replace(/^\s+|\s+$/g, '').slice(0, MAX_LENGTH);
  }

  function setReport(message, isError) {
    fieldReportText.textContent = message;
    fieldReport.classList.toggle('is-error', isError);
    infoIcon.hidden = isError;
    errorIcon.hidden = !isError;
    input.setAttribute('aria-invalid', isError ? 'true' : 'false');
  }

  function render() {
    var value = input.value;
    var length = value.length;
    var busy = state === 'loading';

    app.setAttribute('data-state', state);
    app.setAttribute('aria-busy', busy ? 'true' : 'false');
    input.disabled = busy;
    submitBtn.disabled = busy || length < MIN_LENGTH;
    scanBtn.disabled = busy;
    submitBtn.setAttribute('aria-busy', busy ? 'true' : 'false');
    codeCount.textContent = pad2(length);

    if (state === 'error' && errorMessage) {
      setReport(errorMessage, true);
    } else if (hintMessage) {
      setReport(hintMessage, false);
    } else {
      setReport(COPY.helper, false);
    }

    if (!busy && state === 'empty' && length === 0) {
      ctaHintText.textContent = COPY.missing;
      ctaHint.hidden = false;
    } else {
      ctaHint.hidden = true;
    }

    refillBtn.hidden = !(state === 'error' && errorKind === 'server');

    stateChips.forEach(function (chip) {
      var active = chip.getAttribute('data-mp-state') === state;
      chip.setAttribute('aria-pressed', String(active));
    });
  }

  function showSimpleToast(message) {
    window.clearTimeout(timers.toast);
    window.clearTimeout(timers.toastHide);
    simpleToast.textContent = message;
    simpleToast.hidden = false;
    void simpleToast.offsetWidth;
    simpleToast.classList.add('is-on');
    timers.toast = window.setTimeout(function () {
      simpleToast.classList.remove('is-on');
      timers.toastHide = window.setTimeout(function () {
        simpleToast.hidden = true;
        simpleToast.textContent = '';
      }, 180);
    }, 1600);
  }

  function hideSimpleToast(immediate) {
    window.clearTimeout(timers.toast);
    window.clearTimeout(timers.toastHide);
    simpleToast.classList.remove('is-on');
    if (immediate) {
      simpleToast.hidden = true;
      simpleToast.textContent = '';
    } else {
      timers.toastHide = window.setTimeout(function () {
        simpleToast.hidden = true;
        simpleToast.textContent = '';
      }, 180);
    }
  }

  function flashBack() {
    window.clearTimeout(timers.back);
    backBtn.classList.add('back-flash');
    timers.back = window.setTimeout(function () {
      backBtn.classList.remove('back-flash');
    }, 1100);
  }

  function showSuccess() {
    hideSuccess(true);
    successToast.hidden = false;
    void successToast.offsetWidth;
    successToast.classList.add('is-on');
    timers.success = window.setTimeout(function () {
      successToast.classList.remove('is-on');
      flashBack();
      timers.successHide = window.setTimeout(function () {
        successToast.hidden = true;
      }, 200);
    }, 1000);
  }

  function hideSuccess(immediate) {
    window.clearTimeout(timers.success);
    window.clearTimeout(timers.successHide);
    successToast.classList.remove('is-on');
    if (immediate) {
      successToast.hidden = true;
    } else {
      timers.successHide = window.setTimeout(function () {
        successToast.hidden = true;
      }, 200);
    }
  }

  function closeScan(restoreFocus) {
    window.clearTimeout(timers.scan);
    timers.scan = 0;
    if (scanScreen.hidden) return;
    scanScreen.hidden = true;
    document.body.classList.remove('is-locked');
    if (restoreFocus && lastFocused && typeof lastFocused.focus === 'function') {
      try { lastFocused.focus({ preventScroll: true }); } catch (error) {}
    }
  }

  function cancelPending() {
    window.clearTimeout(timers.submit);
    timers.submit = 0;
    closeScan(false);
    hideSuccess(true);
    hideSimpleToast(true);
    window.clearTimeout(timers.back);
    backBtn.classList.remove('back-flash');
  }

  function applyDemoState(next) {
    if (VALID_STATES.indexOf(next) === -1) return;
    cancelPending();
    errorMessage = '';
    errorKind = '';
    hintMessage = '';

    if (next === 'normal') {
      input.value = SAMPLE_CODE;
    } else if (next === 'empty') {
      input.value = '';
    } else if (next === 'loading') {
      input.value = normalizeCode(input.value) || SAMPLE_CODE;
    } else if (next === 'error') {
      input.value = DEMO_ERROR_CODE;
      errorMessage = COPY.notFound;
      errorKind = 'server';
    }

    state = next;
    render();
  }

  function startSubmit(code) {
    if (state === 'loading') return;
    hideSimpleToast(true);
    hideSuccess(true);
    errorMessage = '';
    errorKind = '';
    hintMessage = '';
    state = 'loading';
    render();

    timers.submit = window.setTimeout(function () {
      timers.submit = 0;
      if (state !== 'loading') return;

      if (code === REJECTED_CODE || code === DEMO_ERROR_CODE) {
        errorMessage = COPY.notFound;
        errorKind = 'server';
        state = 'error';
        render();
        return;
      }

      errorMessage = '';
      state = 'normal';
      render();
      showSuccess();
    }, 1200);
  }

  function submitForm(event) {
    if (event) event.preventDefault();
    if (state === 'loading') return;

    var code = normalizeCode(input.value);
    input.value = code;

    if (!code) {
      errorMessage = '';
      errorKind = '';
      hintMessage = '';
      state = 'empty';
      render();
      showSimpleToast(COPY.emptySubmit);
      try { input.focus({ preventScroll: true }); } catch (error) {}
      return;
    }

    if (code.length < MIN_LENGTH) {
      errorMessage = COPY.short;
      errorKind = 'client';
      hintMessage = '';
      state = 'error';
      render();
      try { input.focus({ preventScroll: true }); } catch (error) {}
      return;
    }

    startSubmit(code);
  }

  function handleInput() {
    if (state === 'loading') return;
    var normalized = normalizeCode(input.value);
    if (input.value !== normalized) input.value = normalized;

    errorMessage = '';
    errorKind = '';
    hintMessage = '';
    if (input.value.length === 0) {
      state = 'empty';
    } else {
      // Typing never turns the field red — a half-typed code is not a failure.
      // The short-code error waits for blur or submit; while typing we only
      // report how many characters are still missing.
      state = 'normal';
      if (input.value.length < MIN_LENGTH) {
        hintMessage = '还差 ' + (MIN_LENGTH - input.value.length) + ' 位';
      }
    }
    render();
  }

  function openScan() {
    if (state === 'loading') return;
    hideSimpleToast(true);
    hideSuccess(true);
    lastFocused = document.activeElement;
    scanScreen.hidden = false;
    document.body.classList.add('is-locked');
    try { scanClose.focus({ preventScroll: true }); } catch (error) {}

    timers.scan = window.setTimeout(function () {
      timers.scan = 0;
      closeScan(false);
      input.value = SAMPLE_CODE;
      startSubmit(SAMPLE_CODE);
    }, 1400);
  }

  input.addEventListener('input', handleInput);
  input.addEventListener('keydown', function (event) {
    if (event.key !== 'Enter' || event.isComposing) return;
    event.preventDefault();
    submitForm(event);
  });

  // Leaving the field is when a too-short code becomes actionable: before this
  // point the user may simply still be typing.
  input.addEventListener('blur', function () {
    if (state === 'loading') return;
    var value = normalizeCode(input.value);
    if (!value || value.length >= MIN_LENGTH) return;
    errorMessage = COPY.short;
    errorKind = 'client';
    hintMessage = '';
    state = 'error';
    render();
  });

  form.addEventListener('submit', submitForm);

  submitBtn.addEventListener('click', function (event) {
    if (submitBtn.disabled) {
      event.preventDefault();
      return;
    }
  });

  refillBtn.addEventListener('click', function () {
    input.value = '';
    errorMessage = '';
    errorKind = '';
    hintMessage = '';
    state = 'empty';
    render();
    try {
      input.focus({ preventScroll: true });
      input.setSelectionRange(0, 0);
    } catch (error) {}
  });

  scanBtn.addEventListener('click', openScan);
  scanClose.addEventListener('click', function () {
    closeScan(true);
  });

  backBtn.addEventListener('click', function () {
    flashBack();
    showSimpleToast(COPY.returning);
  });

  document.addEventListener('keydown', function (event) {
    if (scanScreen.hidden) return;
    if (event.key === 'Escape') {
      event.preventDefault();
      closeScan(true);
      return;
    }
    if (event.key === 'Tab') {
      event.preventDefault();
      try { scanClose.focus({ preventScroll: true }); } catch (error) {}
    }
  });

  stateChips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      var next = chip.getAttribute('data-mp-state');
      document.dispatchEvent(new CustomEvent('mp:state', {
        detail: { state: next, source: 'bind-page' }
      }));
      if (window.OdShell && window.OdShell.closeDemo) window.OdShell.closeDemo();
    });
  });

  document.addEventListener('mp:state', function (event) {
    var detail = event && event.detail;
    if (!detail || typeof detail.state !== 'string') return;
    applyDemoState(detail.state);
  });

  input.value = normalizeCode(input.value);
  render();
})();
