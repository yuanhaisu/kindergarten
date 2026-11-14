/* 班级信息 · 页面交互
 * 四态芯片由共享 shell.js 统一绑定；这里只监听 mp:state 并同步界面。
 */
(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const toast = document.querySelector('.toast');
  const backButton = document.querySelector('button[aria-label="返回"]');
  const entryCells = Array.from(document.querySelectorAll('.entry-cell'));
  const stateChips = Array.from(document.querySelectorAll('[data-mp-state]'));
  const stateActions = Array.from(document.querySelectorAll('.state-action'));

  let toastTimer = 0;
  let reloadTimer = 0;

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
    }, 1800);
  }

  function hideToast() {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.classList.remove('is-visible');
  }

  function resetActions() {
    stateActions.forEach((button) => {
      button.setAttribute('aria-disabled', 'false');
      if (button.dataset.idleLabel) {
        button.textContent = button.dataset.idleLabel;
      }
    });
  }

  function syncState(state) {
    app.dataset.state = state;
    app.setAttribute('aria-busy', state === 'loading' ? 'true' : 'false');
    stateChips.forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
  }

  function startReload(button) {
    if (!button || button.getAttribute('aria-disabled') === 'true') return;

    window.clearTimeout(reloadTimer);
    resetActions();

    button.setAttribute('aria-disabled', 'true');
    button.textContent = '重试中…';
    app.setAttribute('aria-busy', 'true');

    reloadTimer = window.setTimeout(() => {
      reloadTimer = 0;
      resetActions();
      syncState('normal');
      showToast('已重新加载');
    }, 1000);
  }

  stateActions.forEach((button) => {
    button.dataset.idleLabel = button.textContent.trim();
    button.addEventListener('click', () => startReload(button));
  });

  entryCells.forEach((button) => {
    button.addEventListener('click', () => {
      showToast(button.dataset.toast || '已打开');
    });
  });

  if (backButton) {
    backButton.addEventListener('click', () => showToast('返回上一页'));
  }

  document.addEventListener('mp:state', (event) => {
    window.clearTimeout(reloadTimer);
    reloadTimer = 0;
    resetActions();
    hideToast();

    const nextState = event.detail && event.detail.state;
    if (nextState) {
      app.setAttribute('aria-busy', nextState === 'loading' ? 'true' : 'false');
      stateChips.forEach((chip) => {
        chip.setAttribute('aria-pressed', String(chip.dataset.mpState === nextState));
      });
    }
  });
})();
