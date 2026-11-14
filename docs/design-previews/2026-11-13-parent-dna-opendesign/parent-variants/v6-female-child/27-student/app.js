/* 幼儿信息 · 女性 / 5 岁数据集 — 有限编辑与预览态 */
(() => {
  'use strict';

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];

  const app = $('#app');
  const view = $('#view-student');
  const layer = $('#sheet-layer');
  const sheet = $('#edit-sheet');
  const toastEl = $('#toast');
  const profileAvatar = $('#profile-avatar');
  const editAvatar = $('#edit-avatar');
  const careValue = $('#care-value');
  const careFact = $('#care-fact');
  const careNote = $('#care-note');
  const careCount = $('#care-count');
  const avatarInput = $('#avatar-input');
  const resetAvatarBtn = $('[data-action="reset-avatar"]');

  const LETTER = '乐';
  const MAX_NOTE = 60;
  const params = new URLSearchParams(location.search);
  if (params.get('dataset')) document.documentElement.dataset.dataset = params.get('dataset');

  let savedNote = '';
  let savedAvatar = '';
  let draftAvatar = null; // { url, isNew } | null
  let lastFocus = null;
  let closeTimer = 0;
  let toastDelay = 0;

  /* ---------- 轻提示 ---------- */
  let toastTimer = 0;
  const hideToast = () => {
    clearTimeout(toastTimer);
    toastEl.classList.remove('is-visible');
  };
  const toast = (message) => {
    hideToast();
    toastEl.textContent = message;
    void toastEl.offsetWidth;
    toastEl.classList.add('is-visible');
    toastTimer = setTimeout(hideToast, 2000);
  };

  /* ---------- 头像与健康备注 ---------- */
  const paintAvatar = (el, url) => {
    if (url) {
      el.style.backgroundImage = `url("${url}")`;
      el.textContent = '';
      el.classList.add('avatar--photo');
    } else {
      el.style.backgroundImage = '';
      el.textContent = LETTER;
      el.classList.remove('avatar--photo');
    }
  };

  const paintCare = (note) => {
    const text = note.trim();
    careValue.textContent = text || '无';
    careFact.classList.toggle('fact--wide', Boolean(text));
  };

  const paintDraftAvatar = () => {
    const url = draftAvatar ? draftAvatar.url : '';
    paintAvatar(editAvatar, url);
    resetAvatarBtn.hidden = !url;
  };

  const countNote = () => {
    careCount.textContent = `${careNote.value.length}/${MAX_NOTE}`;
  };

  /* ---------- 底部抽屉 ---------- */
  const openSheet = () => {
    clearTimeout(closeTimer);
    clearTimeout(toastDelay);
    hideToast();
    lastFocus = document.activeElement;
    careNote.value = savedNote;
    countNote();
    if (draftAvatar && draftAvatar.isNew) URL.revokeObjectURL(draftAvatar.url);
    draftAvatar = savedAvatar ? { url: savedAvatar, isNew: false } : null;
    paintDraftAvatar();

    layer.hidden = false;
    app.classList.add('is-sheet-open');
    if ('inert' in view) view.inert = true;
    void layer.offsetWidth;
    layer.classList.add('is-open');
    sheet.focus({ preventScroll: true });
  };

  const closeSheet = (restoreFocus = true) => {
    if (layer.hidden) return;
    layer.classList.remove('is-open');
    app.classList.remove('is-sheet-open');
    if ('inert' in view) view.inert = false;
    clearTimeout(closeTimer);
    closeTimer = setTimeout(() => {
      layer.hidden = true;
      if (draftAvatar && draftAvatar.isNew) URL.revokeObjectURL(draftAvatar.url);
      draftAvatar = null;
      if (restoreFocus && lastFocus && typeof lastFocus.focus === 'function') {
        lastFocus.focus({ preventScroll: true });
      }
    }, 210);
  };

  const save = () => {
    savedNote = careNote.value.trim();
    paintCare(savedNote);

    const nextUrl = draftAvatar ? draftAvatar.url : '';
    if (savedAvatar && savedAvatar !== nextUrl && savedAvatar.startsWith('blob:')) {
      URL.revokeObjectURL(savedAvatar);
    }
    savedAvatar = nextUrl;
    if (draftAvatar) draftAvatar.isNew = false;
    draftAvatar = null;

    paintAvatar(profileAvatar, savedAvatar);
    paintAvatar(editAvatar, savedAvatar);
    closeSheet();
    clearTimeout(toastDelay);
    toastDelay = setTimeout(() => toast('已保存'), 240);
  };

  const resetAvatar = () => {
    if (draftAvatar && draftAvatar.isNew) URL.revokeObjectURL(draftAvatar.url);
    draftAvatar = null;
    paintDraftAvatar();
  };

  /* ---------- 事件 ---------- */
  document.addEventListener('click', (e) => {
    const target = e.target.closest('[data-action]');
    if (!target) return;
    switch (target.dataset.action) {
      case 'back':
        toast('返回「我的」');
        break;
      case 'edit':
        openSheet();
        break;
      case 'close-sheet':
        closeSheet();
        break;
      case 'save':
        save();
        break;
      case 'pick-avatar':
        avatarInput.click();
        break;
      case 'reset-avatar':
        resetAvatar();
        break;
      default:
        break;
    }
  });

  avatarInput.addEventListener('change', () => {
    const file = avatarInput.files && avatarInput.files[0];
    avatarInput.value = '';
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      toast('请选择图片文件');
      return;
    }
    const url = URL.createObjectURL(file);
    if (draftAvatar && draftAvatar.isNew) URL.revokeObjectURL(draftAvatar.url);
    draftAvatar = { url, isNew: true };
    paintDraftAvatar();
  });

  careNote.addEventListener('input', countNote);

  document.addEventListener('keydown', (e) => {
    if (layer.hidden) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      closeSheet();
      return;
    }
    if (e.key !== 'Tab') return;
    const focusables = $$('button, textarea', sheet).filter((el) => !el.disabled && el.offsetParent !== null);
    if (!focusables.length) return;
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    if (e.shiftKey && (document.activeElement === first || document.activeElement === sheet)) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  });

  /* 预览态：?state=edit 直接打开抽屉；?dataset=female-5y 标记数据集 */
  if (params.get('state') === 'edit') {
    setTimeout(openSheet, 320);
  }
})();
