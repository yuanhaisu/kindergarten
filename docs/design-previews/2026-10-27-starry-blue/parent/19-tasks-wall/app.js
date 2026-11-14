(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const preview = document.getElementById('photo-preview');
  const previewArt = document.getElementById('preview-art');
  const previewCaption = document.getElementById('preview-caption');
  const previewClose = document.getElementById('preview-close');
  const toast = document.getElementById('proto-toast');
  const staleBanner = document.getElementById('stale-banner');
  const retryButton = document.getElementById('retry-load');
  const goTasksButton = document.getElementById('go-tasks');
  const backButton = document.querySelector('.mp-back');
  const normalChip = document.querySelector('.mp-demo-chip[data-mp-state="normal"]');

  let toastTimer = 0;
  let retryTimer = 0;
  let lastPhotoTrigger = null;

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastTimer = 0;
    }, 1800);
  }

  function resetRetryButton() {
    if (!retryButton) return;
    retryButton.disabled = false;
    retryButton.textContent = '重试';
  }

  function restoreNormal() {
    if (normalChip) normalChip.click();
  }

  function openPreview(trigger) {
    if (!preview || !previewArt || !previewCaption || !previewClose) return;
    const photo = trigger.dataset.photo || 'reading';
    const child = trigger.dataset.child || '同班家长';
    const task = trigger.dataset.task || '';

    lastPhotoTrigger = trigger;
    previewArt.dataset.photo = photo;
    previewCaption.textContent = task ? `${child} · ${task}` : child;
    preview.hidden = false;
    previewClose.focus({ preventScroll: true });
  }

  function closePreview(restoreFocus) {
    if (!preview || preview.hidden) return;
    preview.hidden = true;
    if (restoreFocus && lastPhotoTrigger && document.contains(lastPhotoTrigger)) {
      lastPhotoTrigger.focus({ preventScroll: true });
    }
    lastPhotoTrigger = null;
  }

  document.querySelectorAll('.photo-hit').forEach((button) => {
    button.addEventListener('click', () => openPreview(button));
  });

  if (previewClose) {
    previewClose.addEventListener('click', () => closePreview(true));
  }

  if (preview) {
    preview.querySelectorAll('[data-preview-close]').forEach((el) => {
      el.addEventListener('click', () => closePreview(true));
    });
  }

  document.addEventListener('keydown', (event) => {
    if (!preview || preview.hidden) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      closePreview(true);
      return;
    }

    if (event.key === 'Tab') {
      event.preventDefault();
      if (previewClose) previewClose.focus({ preventScroll: true });
    }
  });

  if (retryButton) {
    retryButton.addEventListener('click', () => {
      if (retryButton.disabled) return;

      retryButton.disabled = true;
      retryButton.textContent = '正在重试…';
      window.clearTimeout(retryTimer);

      retryTimer = window.setTimeout(() => {
        retryTimer = 0;
        resetRetryButton();
        restoreNormal();
        showToast('已重新加载');
      }, 900);
    });
  }

  if (staleBanner) {
    staleBanner.addEventListener('click', () => {
      restoreNormal();
      showToast('已重新加载');
    });
  }

  if (goTasksButton) {
    goTasksButton.addEventListener('click', () => {
      showToast('已打开任务列表');
    });
  }

  if (backButton) {
    backButton.addEventListener('click', () => {
      showToast('已返回任务列表');
    });
  }

  document.addEventListener('mp:state', () => {
    if (retryTimer) {
      window.clearTimeout(retryTimer);
      retryTimer = 0;
    }
    resetRetryButton();
    closePreview(false);
  });
})();
