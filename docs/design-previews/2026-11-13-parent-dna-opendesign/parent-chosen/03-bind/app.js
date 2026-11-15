/* pages/auth/bind · 绑定孩子 — 预览交互
 * 无框架、无网络请求。预览态：?state=empty | invalid | error | success
 */
(() => {
  'use strict';

  const form = document.getElementById('bind-form');
  const intro = document.getElementById('bind-intro');
  const codeInput = document.getElementById('bind-code');
  const codeError = document.getElementById('bind-code-error');
  const codeHint = document.getElementById('bind-code-hint');
  const alertBox = document.getElementById('bind-alert');
  const retryButton = document.getElementById('bind-retry');
  const successPanel = document.getElementById('bind-success');
  const enterButton = document.getElementById('success-enter');
  const toast = document.getElementById('toast');
  const contactButton = document.querySelector('[data-od-id="contact-teacher"]');
  const main = document.querySelector('.mp-main');

  // 契约：invite_code 为 6–20 位字母数字；提交前 trim 首尾空格。
  const CODE_PATTERN = /^[A-Za-z0-9]{6,20}$/;

  let toastTimer = null;
  let toastHideTimer = null;

  /* ---------- 错误态 ---------- */

  function setInvalid(input, errorEl, hintEl) {
    input.classList.add('is-invalid');
    input.setAttribute('aria-invalid', 'true');
    errorEl.hidden = false;
    if (hintEl) hintEl.hidden = true;
  }

  function clearInvalid(input, errorEl, hintEl) {
    input.classList.remove('is-invalid');
    input.removeAttribute('aria-invalid');
    errorEl.hidden = true;
    if (hintEl) hintEl.hidden = false;
  }

  function showAlert() {
    alertBox.hidden = false;
  }

  function hideAlert() {
    alertBox.hidden = true;
  }

  function showSuccess() {
    form.hidden = true;
    intro.hidden = true;
    successPanel.hidden = false;
    if (main) main.scrollTop = 0;
  }

  /* ---------- Toast ---------- */

  function showToast(message) {
    window.clearTimeout(toastTimer);
    window.clearTimeout(toastHideTimer);
    toast.textContent = message;
    toast.hidden = false;
    window.requestAnimationFrame(() => {
      toast.classList.add('is-visible');
    });
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastHideTimer = window.setTimeout(() => {
        toast.hidden = true;
      }, 150);
    }, 2000);
  }

  /* ---------- 表单 ---------- */

  codeInput.addEventListener('input', () => {
    clearInvalid(codeInput, codeError, codeHint);
  });

  form.addEventListener('submit', (event) => {
    event.preventDefault();
    const code = codeInput.value.trim();
    // 回写 trim 后的值：粘贴带空格时用户能看到实际提交的内容。
    if (codeInput.value !== code) codeInput.value = code;

    if (!CODE_PATTERN.test(code)) {
      setInvalid(codeInput, codeError, codeHint);
      codeInput.focus({ preventScroll: true });
      return;
    }

    clearInvalid(codeInput, codeError, codeHint);
    hideAlert();
    showSuccess();
  });

  retryButton.addEventListener('click', () => {
    hideAlert();
    clearInvalid(codeInput, codeError, codeHint);
    codeInput.value = '';
    codeInput.focus({ preventScroll: true });
  });

  contactButton.addEventListener('click', () => {
    showToast('请向班级老师索取绑定码');
  });

  enterButton.addEventListener('click', () => {
    showToast('已进入首页（预览）');
  });

  /* ---------- 预览态 ---------- */

  const state = new URLSearchParams(window.location.search).get('state');

  if (state === 'empty') {
    codeInput.value = '';
  }

  if (state === 'invalid') {
    codeInput.value = 'AB12';
    setInvalid(codeInput, codeError, codeHint);
  }

  if (state === 'error') {
    showAlert();
  }

  if (state === 'success') {
    showSuccess();
  }
})();
