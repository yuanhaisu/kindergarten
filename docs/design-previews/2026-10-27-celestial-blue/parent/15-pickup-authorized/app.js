(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  const list = document.getElementById('rosterList');
  if (!app || !list) return;

  const assignedBanner = document.getElementById('assignedBanner');
  const unassignedBanner = document.getElementById('unassignedBanner');
  const assignedName = document.getElementById('assignedPersonName');
  const rosterCount = document.getElementById('rosterCount');
  const addButton = document.getElementById('addPersonButton');
  const emptyAddButton = document.getElementById('addPersonEmptyButton');
  const backButton = document.querySelector('.mp-back');
  const designateDialog = document.getElementById('designateDialog');
  const designateBody = document.getElementById('designateDialogBody');
  const confirmDesignate = document.getElementById('confirmDesignate');
  const deleteDialog = document.getElementById('deleteDialog');
  const deleteBody = document.getElementById('deleteDialogBody');
  const confirmDelete = document.getElementById('confirmDelete');
  const retryButton = document.getElementById('retryButton');
  const toast = document.getElementById('toast');
  const chips = Array.prototype.slice.call(document.querySelectorAll('.mp-demo-chip'));
  const dialogs = [designateDialog, deleteDialog].filter(Boolean);
  const initialRosterMarkup = list.innerHTML;
  const initialDesignatedCard = list.querySelector('.person-card.is-designated');
  const initialDesignatedId = initialDesignatedCard
    ? initialDesignatedCard.getAttribute('data-person-id')
    : null;

  let designatedId = initialDesignatedId;
  let pendingDesignateId = null;
  let pendingDeleteId = null;
  let currentState = app.getAttribute('data-state') || 'normal';
  let lastFocused = null;
  let toastTimer = null;
  let retryTimer = null;

  function getCards() {
    return Array.prototype.slice.call(list.querySelectorAll('.person-card'));
  }

  function findCard(id) {
    if (!id) return null;
    const all = getCards();
    for (let i = 0; i < all.length; i += 1) {
      if (all[i].getAttribute('data-person-id') === id) return all[i];
    }
    return null;
  }

  function personName(card) {
    return card ? card.getAttribute('data-name') || '' : '';
  }

  function isEligible(card) {
    return Boolean(
      card &&
        card.getAttribute('data-review-status') === '1' &&
        card.getAttribute('data-status') === '1'
    );
  }

  function unavailableMessage(card) {
    if (!card) return '该接送人当前不可指定';
    if (card.getAttribute('data-review-status') !== '1') {
      return '该接送人尚未通过园所审核';
    }
    return '该接送人授权已撤销';
  }

  function isVisible(element) {
    return Boolean(element && element.isConnected && element.getClientRects().length);
  }

  function focusElement(element) {
    if (!element || typeof element.focus !== 'function') return;
    try {
      element.focus({ preventScroll: true });
    } catch (error) {
      element.focus();
    }
  }

  function showToast(message) {
    if (!toast) return;
    if (toastTimer !== null) {
      window.clearTimeout(toastTimer);
      toastTimer = null;
    }
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastTimer = null;
    }, 2000);
  }

  function refreshCount() {
    if (!rosterCount) return;
    rosterCount.textContent = getCards().length + ' 位授权接送人';
  }

  function applyDesignation(nextId) {
    const all = getCards();
    let activeCard = null;

    if (nextId) {
      for (let i = 0; i < all.length; i += 1) {
        if (all[i].getAttribute('data-person-id') === nextId) {
          activeCard = all[i];
          break;
        }
      }
    }

    designatedId = activeCard ? nextId : null;

    all.forEach((card) => {
      const isActive = card === activeCard;
      card.classList.toggle('is-designated', isActive);

      const todayBadge = card.querySelector('[data-today-badge]');
      if (todayBadge) todayBadge.hidden = !isActive;

      const designateButton = card.querySelector('[data-action="designate"]');
      if (designateButton) {
        designateButton.hidden = isActive;
        designateButton.setAttribute('aria-disabled', isEligible(card) ? 'false' : 'true');
      }
    });

    if (assignedBanner && unassignedBanner && assignedName) {
      if (activeCard) {
        assignedName.textContent = personName(activeCard);
        assignedBanner.hidden = false;
        unassignedBanner.hidden = true;
      } else {
        assignedBanner.hidden = true;
        unassignedBanner.hidden = false;
      }
    }
  }

  function restoreInitialRoster() {
    list.innerHTML = initialRosterMarkup;
    applyDesignation(initialDesignatedId);
    refreshCount();
  }

  function syncChips(state) {
    chips.forEach((chip) => {
      const isActive = chip.getAttribute('data-mp-state') === state;
      chip.setAttribute('aria-pressed', isActive ? 'true' : 'false');
    });
  }

  function closeDialogs(restoreFocus) {
    const returnTarget = lastFocused;
    dialogs.forEach((layer) => {
      layer.setAttribute('aria-hidden', 'true');
      layer.setAttribute('inert', '');
    });
    lastFocused = null;
    pendingDesignateId = null;
    pendingDeleteId = null;
    if (restoreFocus && isVisible(returnTarget)) {
      focusElement(returnTarget);
    }
  }

  function openDialog(layer, initialFocus, trigger) {
    if (!layer) return;
    lastFocused = isVisible(trigger)
      ? trigger
      : document.activeElement && typeof document.activeElement.focus === 'function'
        ? document.activeElement
        : null;

    dialogs.forEach((otherLayer) => {
      if (otherLayer !== layer) {
        otherLayer.setAttribute('aria-hidden', 'true');
        otherLayer.setAttribute('inert', '');
      }
    });

    layer.removeAttribute('inert');
    layer.setAttribute('aria-hidden', 'false');

    const focusTarget = initialFocus || layer.querySelector('.modal-button');
    if (focusTarget) {
      window.requestAnimationFrame(() => {
        if (layer.getAttribute('aria-hidden') === 'false') {
          focusElement(focusTarget);
        }
      });
    }
  }

  function openDesignateDialog(card, trigger) {
    if (!card) return;
    pendingDesignateId = card.getAttribute('data-person-id');
    const name = personName(card);

    if (designateBody) {
      designateBody.textContent = designatedId
        ? '换为 ' + name + ' 后，今日已生成的接送二维码将全部失效，需要重新获取。确定继续？'
        : '确定指定 ' + name + ' 为今日接走人吗？';
    }

    openDialog(
      designateDialog,
      designateDialog ? designateDialog.querySelector('.modal-button--cancel') : null,
      trigger
    );
  }

  function openDeleteDialog(card, trigger) {
    if (!card) return;
    pendingDeleteId = card.getAttribute('data-person-id');

    if (deleteBody) {
      deleteBody.textContent = '确定删除 ' + personName(card) + ' 吗？';
    }

    openDialog(
      deleteDialog,
      deleteDialog ? deleteDialog.querySelector('.modal-button--cancel') : null,
      trigger
    );
  }

  function setPageState(nextState) {
    app.setAttribute('data-state', nextState);
    currentState = nextState;
    syncChips(nextState);
    if (nextState !== 'normal') closeDialogs(false);
  }

  function cancelRetry() {
    if (retryTimer !== null) {
      window.clearTimeout(retryTimer);
      retryTimer = null;
    }
    if (retryButton) {
      retryButton.disabled = false;
      retryButton.removeAttribute('aria-busy');
      retryButton.textContent = '重试';
    }
  }

  list.addEventListener('click', (event) => {
    const origin = event.target;
    if (!(origin instanceof Element)) return;

    const actionButton = origin.closest('[data-action]');
    if (!actionButton || !list.contains(actionButton)) return;

    const card = actionButton.closest('.person-card');
    if (!card) return;

    const action = actionButton.getAttribute('data-action');
    if (action === 'designate') {
      if (card.classList.contains('is-designated')) return;
      if (!isEligible(card)) {
        showToast(unavailableMessage(card));
        return;
      }
      openDesignateDialog(card, actionButton);
      return;
    }

    if (action === 'delete') {
      openDeleteDialog(card, actionButton);
    }
  });

  if (confirmDesignate) {
    confirmDesignate.addEventListener('click', () => {
      const card = findCard(pendingDesignateId);
      if (!card) {
        closeDialogs(true);
        return;
      }

      if (!isEligible(card)) {
        const message = unavailableMessage(card);
        closeDialogs(false);
        showToast(message);
        return;
      }

      const id = card.getAttribute('data-person-id');
      const name = personName(card);
      applyDesignation(id);
      closeDialogs(false);
      showToast('今日接走人：' + name);

      const updatedCard = findCard(id);
      const nextAction = updatedCard ? updatedCard.querySelector('.delete-button') : null;
      if (nextAction) {
        window.requestAnimationFrame(() => focusElement(nextAction));
      }
    });
  }

  if (confirmDelete) {
    confirmDelete.addEventListener('click', () => {
      const card = findCard(pendingDeleteId);
      if (!card) {
        closeDialogs(true);
        return;
      }

      const removedId = card.getAttribute('data-person-id');
      const removedWasDesignated = removedId === designatedId;
      card.remove();

      applyDesignation(removedWasDesignated ? null : designatedId);
      refreshCount();
      closeDialogs(false);
      showToast('已删除');

      if (getCards().length === 0) {
        setPageState('empty');
        window.requestAnimationFrame(() => focusElement(emptyAddButton));
      } else {
        window.requestAnimationFrame(() => focusElement(addButton));
      }
    });
  }

  document.querySelectorAll('[data-close-dialog]').forEach((button) => {
    button.addEventListener('click', () => closeDialogs(true));
  });

  dialogs.forEach((layer) => {
    layer.addEventListener('click', (event) => {
      if (event.target === layer) closeDialogs(true);
    });

    layer.addEventListener('keydown', (event) => {
      if (event.key !== 'Tab') return;

      const focusable = Array.prototype.slice
        .call(layer.querySelectorAll('button:not([disabled])'))
        .filter(isVisible);

      if (focusable.length === 0) {
        event.preventDefault();
        return;
      }

      const first = focusable[0];
      const last = focusable[focusable.length - 1];

      if (!layer.contains(document.activeElement)) {
        event.preventDefault();
        focusElement(first);
        return;
      }

      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        focusElement(last);
      } else if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        focusElement(first);
      }
    });
  });

  document.addEventListener('keydown', (event) => {
    if (event.key !== 'Escape') return;
    const openLayer = dialogs.find(
      (layer) => layer.getAttribute('aria-hidden') === 'false'
    );
    if (!openLayer) return;
    event.preventDefault();
    closeDialogs(true);
  });

  if (addButton) {
    addButton.addEventListener('click', () => showToast('已打开添加授权接送人'));
  }

  if (emptyAddButton) {
    emptyAddButton.addEventListener('click', () => showToast('已打开添加授权接送人'));
  }

  if (backButton) {
    backButton.addEventListener('click', () => showToast('已返回接送'));
  }

  if (retryButton) {
    retryButton.addEventListener('click', () => {
      if (retryButton.disabled) return;

      retryButton.disabled = true;
      retryButton.setAttribute('aria-busy', 'true');
      retryButton.textContent = '重试中…';

      retryTimer = window.setTimeout(() => {
        retryTimer = null;
        retryButton.disabled = false;
        retryButton.removeAttribute('aria-busy');
        retryButton.textContent = '重试';
        setPageState('normal');
        showToast('已重新加载');
        window.requestAnimationFrame(() => focusElement(addButton));
      }, 1000);
    });
  }

  document.addEventListener('mp:state', (event) => {
    const detail = event.detail || {};
    const nextState =
      detail.state || app.getAttribute('data-state') || 'normal';
    const previousState = currentState;

    cancelRetry();

    if (nextState !== 'normal') {
      closeDialogs(false);
    }

    if (nextState === 'normal' && previousState !== 'normal') {
      restoreInitialRoster();
    }

    currentState = nextState;
  });

  applyDesignation(designatedId);
  refreshCount();
  syncChips(currentState);
})();
