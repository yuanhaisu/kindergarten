/* pages/auth/bind · 绑定孩子 — 预览交互
 * 无框架、无网络请求。预览态：?state=empty | error | sheet | done
 */
(() => {
  'use strict';

  const form = document.getElementById('bind-form');
  const nameInput = document.getElementById('child-name');
  const codeInput = document.getElementById('bind-code');
  const nameError = document.getElementById('child-name-error');
  const codeError = document.getElementById('bind-code-error');
  const codeHint = document.getElementById('bind-code-hint');
  const trigger = document.getElementById('relation-trigger');
  const relationValue = document.getElementById('relation-value');
  const sheetLayer = document.getElementById('relation-sheet');
  const sheetOptions = Array.from(sheetLayer.querySelectorAll('.sheet__option'));
  const radios = Array.from(sheetLayer.querySelectorAll('input[name="relation"]'));
  const toast = document.getElementById('toast');
  const contactButton = document.querySelector('[data-od-id="contact-teacher"]');
  const backgroundRegions = [
    document.querySelector('.page-bar'),
    document.querySelector('.mp-main'),
  ];

  const CLOSE_MS = 200;
  let sheetOpen = false;
  let closeTimer = null;
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

  /* ---------- 关系抽屉 ---------- */

  function syncSelection() {
    sheetOptions.forEach((option) => {
      const radio = option.querySelector('input[type="radio"]');
      option.classList.toggle('is-selected', radio.checked);
    });
    const selected = radios.find((radio) => radio.checked);
    if (selected) relationValue.textContent = selected.value;
  }

  function setBackgroundInert(inert) {
    backgroundRegions.forEach((region) => {
      if (!region) return;
      if (inert) region.setAttribute('inert', '');
      else region.removeAttribute('inert');
    });
  }

  function openSheet() {
    if (sheetOpen) return;
    sheetOpen = true;
    window.clearTimeout(closeTimer);
    sheetLayer.hidden = false;
    trigger.setAttribute('aria-expanded', 'true');
    setBackgroundInert(true);
    syncSelection();
    window.requestAnimationFrame(() => {
      sheetLayer.classList.add('is-open');
      const checked = radios.find((radio) => radio.checked) || radios[0];
      if (checked) checked.focus({ preventScroll: true });
    });
  }

  function closeSheet() {
    if (!sheetOpen) return;
    sheetOpen = false;
    window.clearTimeout(closeTimer);
    sheetLayer.classList.remove('is-open');
    trigger.setAttribute('aria-expanded', 'false');
    setBackgroundInert(false);
    closeTimer = window.setTimeout(() => {
      sheetLayer.hidden = true;
      trigger.focus({ preventScroll: true });
    }, CLOSE_MS);
  }

  trigger.addEventListener('click', openSheet);

  sheetLayer.querySelector('[data-sheet-close]').addEventListener('click', closeSheet);

  sheetLayer.addEventListener('keydown', (event) => {
    if (event.key === 'Escape') {
      event.preventDefault();
      closeSheet();
      return;
    }
    if (event.key === 'Tab') {
      event.preventDefault();
      const checked = radios.find((radio) => radio.checked) || radios[0];
      if (checked) checked.focus({ preventScroll: true });
    }
  });

  radios.forEach((radio) => {
    radio.addEventListener('change', () => {
      if (radio.checked) syncSelection();
    });
  });

  sheetOptions.forEach((option) => {
    option.addEventListener('click', () => {
      const radio = option.querySelector('input[type="radio"]');
      if (!radio.checked) {
        radio.checked = true;
        radio.dispatchEvent(new Event('change', { bubbles: true }));
      }
      syncSelection();
      closeSheet();
    });
  });

  /* ---------- 表单 ---------- */

  nameInput.addEventListener('input', () => {
    clearInvalid(nameInput, nameError, null);
  });

  codeInput.addEventListener('input', () => {
    const digits = codeInput.value.replace(/\D/g, '').slice(0, 6);
    if (codeInput.value !== digits) codeInput.value = digits;
    clearInvalid(codeInput, codeError, codeHint);
  });

  form.addEventListener('submit', (event) => {
    event.preventDefault();
    const name = nameInput.value.trim();
    const code = codeInput.value.replace(/\D/g, '').slice(0, 6);
    if (codeInput.value !== code) codeInput.value = code;
    let firstInvalid = null;

    if (!name) {
      setInvalid(nameInput, nameError, null);
      firstInvalid = nameInput;
    } else {
      clearInvalid(nameInput, nameError, null);
    }

    if (code.length !== 6) {
      setInvalid(codeInput, codeError, codeHint);
      if (!firstInvalid) firstInvalid = codeInput;
    } else {
      clearInvalid(codeInput, codeError, codeHint);
    }

    if (firstInvalid) {
      firstInvalid.focus({ preventScroll: true });
      return;
    }

    showToast(`${name}已绑定成功`);
  });

  contactButton.addEventListener('click', () => {
    showToast('请向班级老师索取绑定码');
  });

  /* ---------- 预览态 ---------- */

  const state = new URLSearchParams(window.location.search).get('state');

  if (state === 'empty') {
    nameInput.value = '';
    codeInput.value = '';
  }

  if (state === 'error') {
    codeInput.value = '38421';
    setInvalid(codeInput, codeError, codeHint);
  }

  if (state === 'sheet') {
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(openSheet);
    });
  }

  if (state === 'done') {
    window.setTimeout(() => {
      showToast('乐乐已绑定成功');
    }, 300);
  }
})();
