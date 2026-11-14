/* 请假详情 · 家长端页面交互
 * 状态 chip 的选中与 data-state 写入由 shell.js 负责；本文件监听 mp:state，
 * 并处理撤销弹窗、toast、失败重试、返回提示与撤销后的降级态。
 */
(() => {
  const app = document.querySelector('.mp-app');
  if (!app) return;

  const main = document.querySelector('.mp-main');
  const navbar = document.querySelector('.mp-navbar');
  const demo = document.querySelector('.mp-demo');
  const backButton = document.querySelector('.mp-back');
  const chips = Array.from(document.querySelectorAll('.mp-demo-chip'));
  const dialog = document.getElementById('revokeDialog');
  const revokeButton = document.getElementById('revokeBtn');
  const cancelButton = document.getElementById('revokeCancel');
  const confirmButton = document.getElementById('revokeConfirm');
  const retryButton = document.getElementById('retryBtn');
  const toast = document.getElementById('toast');

  let lastFocused = null;
  let toastTimer = 0;
  let retryTimer = 0;

  const statusHeadingIds = {
    approved: 'statusApprovedTitle',
    pending: 'statusPendingTitle',
    rejected: 'statusRejectedTitle',
    revoked: 'statusRevokedTitle'
  };

  function setChipState(state) {
    chips.forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
  }

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toast.textContent = '';
    }, 2200);
  }

  function setBackgroundInert(isInert) {
    [main, navbar, demo].forEach((element) => {
      if (!element) return;
      if (isInert) element.setAttribute('inert', '');
      else element.removeAttribute('inert');
    });
  }

  function focusStatusHeading(state) {
    const id = statusHeadingIds[state];
    if (!id) return;
    const heading = document.getElementById(id);
    if (!heading) return;
    heading.setAttribute('tabindex', '-1');
    heading.focus({ preventScroll: true });
  }

  function openDialog() {
    if (!dialog || app.dataset.state !== 'pending') return;
    lastFocused = document.activeElement;
    dialog.hidden = false;
    setBackgroundInert(true);
    document.body.style.overflow = 'hidden';
    window.requestAnimationFrame(() => {
      if (cancelButton) cancelButton.focus();
    });
  }

  function closeDialog(restoreFocus = true) {
    if (!dialog || dialog.hidden) return;
    dialog.hidden = true;
    setBackgroundInert(false);
    document.body.style.overflow = '';

    if (
      restoreFocus &&
      lastFocused &&
      document.contains(lastFocused) &&
      typeof lastFocused.focus === 'function' &&
      lastFocused.offsetParent !== null
    ) {
      lastFocused.focus({ preventScroll: true });
    }
    lastFocused = null;
  }

  function resetRetryButton() {
    if (!retryButton) return;
    retryButton.disabled = false;
    retryButton.removeAttribute('aria-busy');
  }

  function cancelPendingRetry() {
    if (!retryTimer) return;
    window.clearTimeout(retryTimer);
    retryTimer = 0;
    resetRetryButton();
  }

  if (backButton) {
    backButton.addEventListener('click', () => {
      showToast('已返回请假记录');
    });
  }

  if (revokeButton) {
    revokeButton.addEventListener('click', openDialog);
  }

  if (cancelButton) {
    cancelButton.addEventListener('click', () => closeDialog());
  }

  if (dialog) {
    dialog.addEventListener('click', (event) => {
      if (event.target === dialog || event.target.hasAttribute('data-dialog-close')) {
        closeDialog();
      }
    });

    dialog.addEventListener('keydown', (event) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeDialog();
        return;
      }
      if (event.key !== 'Tab' || !cancelButton || !confirmButton) return;

      const first = cancelButton;
      const last = confirmButton;
      const active = document.activeElement;

      if (event.shiftKey) {
        if (active === first || !dialog.contains(active)) {
          event.preventDefault();
          last.focus();
        }
      } else if (active === last || !dialog.contains(active)) {
        event.preventDefault();
        first.focus();
      }
    });
  }

  if (confirmButton) {
    confirmButton.addEventListener('click', () => {
      closeDialog(false);
      app.dataset.state = 'revoked';
      setChipState('revoked');
      showToast('已撤销');
      focusStatusHeading('revoked');
    });
  }

  if (retryButton) {
    retryButton.addEventListener('click', () => {
      if (app.dataset.state !== 'error') return;

      cancelPendingRetry();
      retryButton.disabled = true;
      retryButton.setAttribute('aria-busy', 'true');
      app.dataset.state = 'loading';
      setChipState('loading');

      retryTimer = window.setTimeout(() => {
        retryTimer = 0;
        app.dataset.state = 'approved';
        setChipState('approved');
        resetRetryButton();
        focusStatusHeading('approved');
        showToast('请假详情已重新加载');
      }, 760);
    });
  }

  document.addEventListener('mp:state', () => {
    cancelPendingRetry();
    closeDialog(false);
  });
})();
