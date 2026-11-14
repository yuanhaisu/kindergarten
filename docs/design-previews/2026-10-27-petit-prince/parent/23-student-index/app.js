/* 孩子信息 · 页面交互
 * 状态芯片由共享 shell.js 接管；这里只监听 mp:state，并同步无障碍状态。
 */
(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const main = app.querySelector('.mp-main');
  const chips = Array.from(app.querySelectorAll('.mp-demo-chip'));
  const backButton = app.querySelector('.mp-back');
  const toast = app.querySelector('[data-toast]');
  const healthCard = app.querySelector('[data-health-card]');
  const healthRetry = app.querySelector('[data-health-retry]');
  const refreshButton = app.querySelector('[data-refresh]');
  const errorRetry = app.querySelector('[data-error-retry]');
  const dialogLayer = app.querySelector('[data-dialog]');
  const dialog = dialogLayer ? dialogLayer.querySelector('.dialog') : null;
  const dialogDesc = app.querySelector('#dialog-desc');
  const dialogCancel = app.querySelector('[data-dialog-cancel]');
  const dialogConfirm = app.querySelector('[data-dialog-confirm]');

  let toastHideTimer = 0;
  let toastClearTimer = 0;
  let lastFocus = null;

  function syncState(state) {
    const current = state || app.dataset.state || 'normal';
    chips.forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === current));
    });
    if (main) main.setAttribute('aria-busy', String(current === 'loading'));
  }

  function setState(state) {
    app.dataset.state = state;
    syncState(state);
    document.dispatchEvent(new CustomEvent('mp:state', { detail: { state } }));
  }

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastHideTimer);
    window.clearTimeout(toastClearTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastHideTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastClearTimer = window.setTimeout(() => {
        toast.textContent = '';
      }, 200);
    }, 1800);
  }

  function runRetry(button, busyLabel, idleLabel, expectedState, done) {
    if (!button || button.getAttribute('aria-disabled') === 'true') return;
    button.setAttribute('aria-disabled', 'true');
    button.textContent = busyLabel;
    window.setTimeout(() => {
      button.removeAttribute('aria-disabled');
      button.textContent = idleLabel;
      if (app.dataset.state === expectedState) {
        done();
      }
    }, 1000);
  }

  // 共享外壳派发状态后，同步主区的忙碌标记与芯片按压态。
  document.addEventListener('mp:state', (event) => {
    const detail = event.detail || {};
    syncState(detail.state || app.dataset.state);
  });

  if (backButton) {
    backButton.addEventListener('click', () => showToast('返回上一页'));
  }

  if (errorRetry) {
    errorRetry.addEventListener('click', () => {
      runRetry(errorRetry, '重试中…', '重试', 'error', () => {
        setState('normal');
        showToast('已重新加载');
      });
    });
  }

  if (refreshButton) {
    refreshButton.addEventListener('click', () => {
      runRetry(refreshButton, '重试中…', '刷新档案', 'empty', () => {
        setState('normal');
        showToast('已重新加载');
      });
    });
  }

  if (healthRetry) {
    healthRetry.addEventListener('click', () => {
      if (healthRetry.getAttribute('aria-disabled') === 'true') return;
      healthRetry.setAttribute('aria-disabled', 'true');
      healthRetry.textContent = '重试中…';
      if (healthCard) {
        healthCard.setAttribute('data-retrying', 'true');
        healthCard.setAttribute('aria-busy', 'true');
      }
      window.setTimeout(() => {
        healthRetry.removeAttribute('aria-disabled');
        healthRetry.textContent = '点击重试';
        if (healthCard) {
          healthCard.removeAttribute('data-retrying');
          healthCard.setAttribute('aria-busy', 'false');
        }
        if (app.dataset.state === 'health-error') {
          setState('normal');
          showToast('晨检已更新');
        }
      }, 1000);
    });
  }

  function openDialog(name, className, trigger) {
    if (!dialogLayer || !dialog) return;
    if (dialogDesc) {
      dialogDesc.textContent = '确定解除与 ' + name + '（' + className + '）的绑定吗？';
    }
    lastFocus = trigger || null;
    dialogLayer.hidden = false;
    window.requestAnimationFrame(() => {
      if (dialogCancel) {
        dialogCancel.focus();
      } else {
        dialog.focus();
      }
    });
  }

  function closeDialog(restoreFocus) {
    if (!dialogLayer || dialogLayer.hidden) return;
    dialogLayer.hidden = true;
    if (restoreFocus !== false && lastFocus) {
      lastFocus.focus();
    }
    lastFocus = null;
  }

  Array.from(app.querySelectorAll('[data-unbind]')).forEach((button) => {
    button.addEventListener('click', () => {
      openDialog(
        button.getAttribute('data-child') || '',
        button.getAttribute('data-class') || '',
        button
      );
    });
  });

  if (dialogCancel) {
    dialogCancel.addEventListener('click', () => closeDialog(true));
  }

  if (dialogConfirm) {
    dialogConfirm.addEventListener('click', () => {
      closeDialog(true);
      showToast('已解绑');
    });
  }

  if (dialogLayer) {
    dialogLayer.addEventListener('click', (event) => {
      if (event.target === dialogLayer) closeDialog(true);
    });

    dialogLayer.addEventListener('keydown', (event) => {
      if (event.key !== 'Tab' || !dialog) return;
      const buttons = Array.from(dialog.querySelectorAll('button'));
      if (buttons.length === 0) return;
      const first = buttons[0];
      const last = buttons[buttons.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    });
  }

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && dialogLayer && !dialogLayer.hidden) {
      event.preventDefault();
      closeDialog(true);
    }
  });

  syncState(app.dataset.state);
})();
