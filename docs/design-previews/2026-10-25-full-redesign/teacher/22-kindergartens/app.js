(function () {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const backBtn = document.getElementById('backBtn');
  const toast = document.getElementById('toast');
  const retryBtn = document.getElementById('retryBtn');
  const dialog = document.getElementById('switchDialog');
  const dialogScrim = dialog ? dialog.querySelector('[data-dialog-close]') : null;
  const cancelBtn = document.getElementById('switchCancel');
  const confirmBtn = document.getElementById('switchConfirm');
  const currentName = document.getElementById('currentName');
  const listFoot = document.getElementById('listFoot');

  const cards = Array.from(app.querySelectorAll('.kg-card'));
  const names = { sun: '阳光幼儿园', forest: '森林幼儿园' };

  let toastTimer = 0;
  let pendingKindergarten = '';

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 1800);
  }

  function openDialog(id) {
    if (!dialog) return;
    pendingKindergarten = id;
    const desc = document.getElementById('switchDialogDesc');
    if (desc) {
      desc.textContent = '切换后将只显示「' + names[id] + '」的数据，未保存的操作会失效。';
    }
    dialog.hidden = false;
    if (confirmBtn) confirmBtn.focus();
  }

  function closeDialog() {
    if (!dialog) return;
    dialog.hidden = true;
    pendingKindergarten = '';
  }

  function currentMark() {
    const mark = document.createElement('span');
    mark.className = 'current-mark';
    mark.setAttribute('aria-hidden', 'true');
    mark.innerHTML = '<svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m2.5 7.2 3 3 6-6.4"/></svg>';
    return mark;
  }

  function switchTail() {
    const tail = document.createElement('span');
    tail.className = 'switch-tail';
    tail.innerHTML = '<span class="switch-text">切换</span><svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m5 2.5 4.5 4.5L5 11.5"/></svg>';
    return tail;
  }

  function currentHint() {
    const hint = document.createElement('span');
    hint.className = 'current-hint';
    hint.textContent = '当前使用中';
    return hint;
  }

  function renderCurrent(id) {
    cards.forEach(function (card) {
      const isCurrent = card.dataset.kindergarten === id;
      card.classList.toggle('is-current', isCurrent);
      if (isCurrent) {
        card.setAttribute('aria-current', 'true');
      } else {
        card.removeAttribute('aria-current');
      }

      const meta = card.querySelector('.kg-meta');
      const hint = meta ? meta.querySelector('.current-hint') : null;
      if (isCurrent && meta && !hint) meta.appendChild(currentHint());
      if (!isCurrent && hint) hint.remove();

      const mark = card.querySelector('.current-mark');
      const tail = card.querySelector('.switch-tail');
      if (isCurrent && !mark) {
        if (tail) tail.remove();
        card.appendChild(currentMark());
      }
      if (!isCurrent && !tail) {
        if (mark) mark.remove();
        card.appendChild(switchTail());
      }
    });
    if (currentName) currentName.textContent = names[id] || '';
    if (listFoot) listFoot.hidden = cards.length > 1;
  }

  cards.forEach(function (card) {
    card.addEventListener('click', function () {
      if (app.dataset.state !== 'normal') return;
      if (card.classList.contains('is-current')) return;
      openDialog(card.dataset.kindergarten);
    });
  });

  if (cancelBtn) cancelBtn.addEventListener('click', closeDialog);
  if (dialogScrim) dialogScrim.addEventListener('click', closeDialog);
  if (confirmBtn) {
    confirmBtn.addEventListener('click', function () {
      const id = pendingKindergarten;
      closeDialog();
      if (!id) return;
      renderCurrent(id);
      showToast('已切换到' + names[id]);
    });
  }

  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape' && dialog && !dialog.hidden) closeDialog();
  });

  document.addEventListener('mp:state', function (event) {
    closeDialog();
    if (event.detail && event.detail.state !== 'normal') return;
  });

  if (retryBtn) {
    retryBtn.addEventListener('click', function () {
      app.dataset.state = 'loading';
      window.setTimeout(function () {
        app.dataset.state = 'normal';
        showToast('已重新加载');
      }, 600);
    });
  }

  if (backBtn) {
    backBtn.addEventListener('click', function () {
      if (window.history.length > 1) window.history.back();
      else showToast('返回「更多功能」');
    });
  }

  if (listFoot) listFoot.hidden = cards.length > 1;
})();
