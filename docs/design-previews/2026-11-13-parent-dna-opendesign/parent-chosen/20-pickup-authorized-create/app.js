(() => {
  'use strict';

  const main = document.querySelector('.mp-main');
  const form = document.getElementById('auth-form');
  const nameInput = document.getElementById('name-input');
  const phoneInput = document.getElementById('phone-input');
  const nameError = document.getElementById('name-error');
  const phoneError = document.getElementById('phone-error');
  const photoThumb = document.getElementById('photo-thumb');
  const photoChange = document.getElementById('photo-change');
  const photoInput = document.getElementById('photo-input');
  const relationPicker = document.getElementById('relation-picker');
  const relationValue = document.getElementById('relation-value');
  const relationField = document.getElementById('relation-field');
  const validityPicker = document.getElementById('validity-picker');
  const validityValue = document.getElementById('validity-value');
  const validityField = document.getElementById('validity-field');
  const layer = document.getElementById('sheet-layer');
  const relationSheet = document.getElementById('relation-sheet');
  const validitySheet = document.getElementById('validity-sheet');
  const dateBlock = document.getElementById('date-block');
  const dateInput = document.getElementById('validity-date');
  const dateConfirm = document.getElementById('validity-date-confirm');
  const toast = document.getElementById('toast');
  const pageTitle = document.querySelector('.page-title');
  const pageSubtitle = document.querySelector('.page-subtitle');
  const reviewNote = document.querySelector('.review-note');
  const submitButton = document.querySelector('.btn-primary');
  const mode = new URLSearchParams(window.location.search).get('mode');

  let activeSheet = null;
  let lastTrigger = null;
  let closeTimer = 0;
  let focusTimer = 0;
  let toastTimer = 0;
  let photoUrl = null;
  let validityState = { mode: 'longterm', date: '' };

  const pad = (value) => String(value).padStart(2, '0');
  const toISODate = (date) => `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
  const formatDate = (iso) => {
    const parts = iso.split('-').map(Number);
    if (parts.length !== 3 || parts.some(Number.isNaN)) return '';
    return `${parts[0]}年${parts[1]}月${parts[2]}日`;
  };
  const defaultDate = () => {
    const date = new Date();
    date.setFullYear(date.getFullYear() + 1);
    return toISODate(date);
  };

  const phonePattern = /^1[3-9]\d{9}$/;
  const today = new Date();
  dateInput.min = toISODate(today);
  dateInput.value = defaultDate();

  function showToast(message) {
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function selectOption(sheet, value) {
    sheet.querySelectorAll('.sheet-option').forEach((option) => {
      option.setAttribute('aria-pressed', String(option.dataset.value === value));
    });
  }

  function syncRelationSheet() {
    selectOption(relationSheet, relationValue.textContent.trim());
  }

  function syncValiditySheet() {
    if (validityState.mode === 'date') {
      selectOption(validitySheet, '指定日期');
      dateBlock.hidden = false;
      dateInput.value = validityState.date;
    } else {
      selectOption(validitySheet, '长期授权');
      dateBlock.hidden = true;
    }
  }

  function openSheet(name, trigger) {
    if (activeSheet) return;

    window.clearTimeout(closeTimer);
    window.clearTimeout(focusTimer);

    lastTrigger = trigger;
    activeSheet = name === 'relation' ? relationSheet : validitySheet;
    relationSheet.hidden = name !== 'relation';
    validitySheet.hidden = name !== 'validity';

    if (name === 'relation') {
      syncRelationSheet();
    } else {
      syncValiditySheet();
    }

    layer.hidden = false;
    main.classList.add('is-locked');
    trigger.setAttribute('aria-expanded', 'true');
    void layer.offsetWidth;
    layer.classList.add('is-open');

    const selected = activeSheet.querySelector('.sheet-option[aria-pressed="true"]');
    const target = selected || activeSheet.querySelector('.sheet-close');
    focusTimer = window.setTimeout(() => {
      if (layer.classList.contains('is-open') && target) {
        target.focus({ preventScroll: true });
      }
    }, 60);

    document.addEventListener('keydown', onKeyDown);
  }

  function closeSheet() {
    if (!activeSheet) return;

    window.clearTimeout(focusTimer);
    layer.classList.remove('is-open');
    main.classList.remove('is-locked');
    document.removeEventListener('keydown', onKeyDown);

    if (lastTrigger) {
      lastTrigger.setAttribute('aria-expanded', 'false');
    }

    activeSheet = null;
    window.clearTimeout(closeTimer);
    closeTimer = window.setTimeout(() => {
      if (layer.classList.contains('is-open')) return;
      layer.hidden = true;
      relationSheet.hidden = true;
      validitySheet.hidden = true;
      if (lastTrigger) {
        lastTrigger.focus({ preventScroll: true });
      }
    }, 220);
  }

  function onKeyDown(event) {
    if (event.key === 'Escape') {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (event.key !== 'Tab' || !activeSheet) return;

    const focusables = Array.from(
      activeSheet.querySelectorAll('button:not([disabled]), input:not([disabled]), [href], [tabindex]:not([tabindex="-1"])')
    ).filter((element) => element.offsetParent !== null);

    if (!focusables.length) return;

    const first = focusables[0];
    const last = focusables[focusables.length - 1];

    if (!activeSheet.contains(document.activeElement)) {
      event.preventDefault();
      first.focus();
    } else if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  relationPicker.addEventListener('click', () => openSheet('relation', relationPicker));
  validityPicker.addEventListener('click', () => openSheet('validity', validityPicker));

  layer.addEventListener('click', (event) => {
    if (event.target.closest('[data-close-sheet]')) {
      closeSheet();
    }
  });

  relationSheet.addEventListener('click', (event) => {
    const option = event.target.closest('.sheet-option');
    if (!option) return;

    const value = option.dataset.value;
    relationValue.textContent = value;
    relationField.value = value;
    selectOption(relationSheet, value);
    closeSheet();
  });

  validitySheet.addEventListener('click', (event) => {
    const option = event.target.closest('.sheet-option');
    if (!option) return;

    if (option.dataset.value === '长期授权') {
      validityState = { mode: 'longterm', date: '' };
      validityValue.textContent = '长期授权';
      validityField.value = '长期授权';
      selectOption(validitySheet, '长期授权');
      closeSheet();
      return;
    }

    selectOption(validitySheet, '指定日期');
    dateBlock.hidden = false;
    if (!dateInput.value) {
      dateInput.value = validityState.date || defaultDate();
    }
    window.requestAnimationFrame(() => {
      if (!dateBlock.hidden) {
        dateInput.focus({ preventScroll: true });
      }
    });
  });

  dateConfirm.addEventListener('click', () => {
    if (!dateInput.value) {
      dateInput.focus();
      return;
    }

    const iso = dateInput.value;
    const label = formatDate(iso);
    if (!label) return;

    validityState = { mode: 'date', date: iso };
    validityValue.textContent = label;
    validityField.value = iso;
    selectOption(validitySheet, '指定日期');
    closeSheet();
  });

  photoChange.addEventListener('click', () => photoInput.click());

  photoInput.addEventListener('change', () => {
    const file = photoInput.files && photoInput.files[0];
    if (!file || !file.type.startsWith('image/')) return;

    if (photoUrl) URL.revokeObjectURL(photoUrl);
    photoUrl = URL.createObjectURL(file);

    const image = document.createElement('img');
    image.src = photoUrl;
    image.alt = '';
    photoThumb.replaceChildren(image);
    photoThumb.classList.add('has-photo');
    photoInput.value = '';
  });

  function updatePhotoLabel() {
    const name = nameInput.value.trim() || '授权人';
    photoThumb.setAttribute('aria-label', `${name}的人像照片`);
  }

  if (mode === 'edit') {
    document.title = '编辑授权人 · 接送';
    pageTitle.textContent = '编辑授权人';
    pageSubtitle.textContent = '修改陈秀兰的授权信息';
    submitButton.textContent = '保存修改';
    reviewNote.innerHTML = '修改后需园所重新审核，<span class="hl">审核通过</span>后生效。';
    nameInput.value = '陈秀兰';
    phoneInput.value = '13700004418';
    relationValue.textContent = '外婆';
    relationField.value = '外婆';
    validityValue.textContent = '长期授权';
    validityField.value = '长期授权';
    syncRelationSheet();
    syncValiditySheet();
    updatePhotoLabel();
  }

  document.querySelectorAll('.switch').forEach((toggle) => {
    toggle.addEventListener('click', () => {
      const checked = toggle.getAttribute('aria-checked') === 'true';
      toggle.setAttribute('aria-checked', String(!checked));
    });
  });

  function setFieldError(input, error, message) {
    error.textContent = message;
    error.hidden = false;
    input.setAttribute('aria-invalid', 'true');
  }

  function clearFieldError(input, error) {
    error.hidden = true;
    input.removeAttribute('aria-invalid');
  }

  nameInput.addEventListener('input', () => {
    clearFieldError(nameInput, nameError);
    updatePhotoLabel();
  });

  phoneInput.addEventListener('input', () => {
    clearFieldError(phoneInput, phoneError);
  });

  form.addEventListener('submit', (event) => {
    event.preventDefault();

    const name = nameInput.value.trim();
    const phone = phoneInput.value.trim();
    let firstInvalid = null;

    if (!name) {
      setFieldError(nameInput, nameError, '请填写姓名');
      firstInvalid = nameInput;
    } else {
      clearFieldError(nameInput, nameError);
    }

    if (!phonePattern.test(phone)) {
      setFieldError(phoneInput, phoneError, '请填写正确的手机号');
      if (!firstInvalid) firstInvalid = phoneInput;
    } else {
      clearFieldError(phoneInput, phoneError);
    }

    if (firstInvalid) {
      firstInvalid.focus();
      return;
    }

    showToast(mode === 'edit' ? '已保存修改' : '已提交，等待园所审核');
    window.setTimeout(() => {
      window.location.href = '../19-pickup-authorized/index.html';
    }, 900);
  });

  window.addEventListener('beforeunload', () => {
    if (photoUrl) URL.revokeObjectURL(photoUrl);
  });
})();
