(() => {
  'use strict';

  const ATTACHMENT_BASE64 = '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';

  const app = document.querySelector('#app');
  const pageBar = document.querySelector('.page-bar');
  const main = document.querySelector('#notice-content');
  const actionBar = document.querySelector('.notice-action-bar');
  const backButton = document.querySelector('[data-od-id="notice-back"]');
  const downloadButton = document.querySelector('[data-od-id="attachment-download"]');
  const signupCta = document.querySelector('#signupCta');
  const signupSheet = document.querySelector('#signupSheet');
  const signupPanel = signupSheet?.querySelector('.sheet');
  const cancelButton = signupSheet?.querySelector('[data-sheet-close]');
  const backdrop = signupSheet?.querySelector('[data-sheet-dismiss]');
  const confirmButton = document.querySelector('#confirmSignup');
  const toast = document.querySelector('#toast');

  const backgroundRegions = [pageBar, main, actionBar].filter(Boolean);
  let lastFocusedElement = null;
  let toastTimer = null;

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function setBackgroundInert(isInert) {
    backgroundRegions.forEach((region) => {
      if (isInert) {
        region.setAttribute('inert', '');
      } else {
        region.removeAttribute('inert');
      }
    });
  }

  function openSheet() {
    if (!signupSheet || signupSheet.classList.contains('is-open')) return;

    lastFocusedElement = document.activeElement;
    signupSheet.setAttribute('aria-hidden', 'false');
    setBackgroundInert(true);
    app.classList.add('is-sheet-open');
    signupSheet.classList.add('is-open');

    window.setTimeout(() => {
      cancelButton?.focus({ preventScroll: true });
    }, 40);
  }

  function closeSheet() {
    if (!signupSheet || !signupSheet.classList.contains('is-open')) return;

    signupSheet.classList.remove('is-open');
    app.classList.remove('is-sheet-open');
    setBackgroundInert(false);

    if (lastFocusedElement instanceof HTMLElement) {
      lastFocusedElement.focus({ preventScroll: true });
    }

    signupSheet.setAttribute('aria-hidden', 'true');
    lastFocusedElement = null;
  }

  function handleSheetKeydown(event) {
    if (event.key === 'Escape') {
      event.preventDefault();
      closeSheet();
      return;
    }

    if (event.key !== 'Tab' || !signupPanel) return;

    const focusableElements = Array.from(
      signupPanel.querySelectorAll('button:not([disabled]), [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')
    ).filter((element) => element.getClientRects().length > 0);

    if (focusableElements.length === 0) return;

    const first = focusableElements[0];
    const last = focusableElements[focusableElements.length - 1];

    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  function base64ToBytes(base64) {
    const binary = window.atob(base64);
    const bytes = new Uint8Array(binary.length);

    for (let index = 0; index < binary.length; index += 1) {
      bytes[index] = binary.charCodeAt(index);
    }

    return bytes;
  }

  function downloadAttachment() {
    try {
      const bytes = base64ToBytes(ATTACHMENT_BASE64);
      const blob = new Blob([bytes], {
        type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
      });
      const objectUrl = URL.createObjectURL(blob);
      const downloadLink = document.createElement('a');

      downloadLink.href = objectUrl;
      downloadLink.download = '报名表.xlsx';
      downloadLink.rel = 'noopener';
      document.body.appendChild(downloadLink);
      downloadLink.click();
      downloadLink.remove();

      window.setTimeout(() => URL.revokeObjectURL(objectUrl), 2000);
      showToast('报名表已开始下载');
    } catch (error) {
      showToast('下载没有开始，请稍后再试');
    }
  }

  backButton?.addEventListener('click', () => {
    if (window.history.length > 1) {
      window.history.back();
    }
  });

  downloadButton?.addEventListener('click', downloadAttachment);
  signupCta?.addEventListener('click', openSheet);
  cancelButton?.addEventListener('click', closeSheet);
  backdrop?.addEventListener('click', closeSheet);
  signupSheet?.addEventListener('keydown', handleSheetKeydown);

  confirmButton?.addEventListener('click', () => {
    closeSheet();
    window.setTimeout(() => showToast('报名成功'), 220);
  });
})();
