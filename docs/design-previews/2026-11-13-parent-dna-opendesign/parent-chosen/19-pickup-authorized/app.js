(function () {
  'use strict';

  var params = new URLSearchParams(window.location.search);
  var state = params.get('state');
  var activeView = state === 'empty' ? 'empty' : 'default';
  var views = document.querySelectorAll('[data-view]');

  for (var i = 0; i < views.length; i += 1) {
    views[i].hidden = views[i].getAttribute('data-view') !== activeView;
  }

  document.documentElement.setAttribute('data-page-state', activeView);

  var main = document.querySelector('.mp-main');
  var rows = Array.prototype.slice.call(document.querySelectorAll('.person-row'));
  var detailLayer = document.getElementById('detail-layer');
  var detailSheet = document.getElementById('detail-sheet');
  var detailAvatar = document.getElementById('detail-avatar');
  var detailName = document.getElementById('detail-name');
  var detailRelation = document.getElementById('detail-relation');
  var detailPhone = document.getElementById('detail-phone');
  var detailAuthType = document.getElementById('detail-auth-type');
  var detailTag = document.getElementById('detail-tag');
  var confirmLayer = document.getElementById('confirm-layer');
  var confirmTitle = document.getElementById('confirm-title');
  var confirmDesc = document.getElementById('confirm-desc');
  var confirmOk = document.getElementById('confirm-ok');
  var toast = document.getElementById('toast');

  var statusTagClass = {
    approved: 'tag tag--approved',
    pending: 'tag tag--pending',
    rejected: 'tag tag--rejected'
  };
  var actionsByStatus = {
    approved: ['edit', 'pause'],
    pending: ['withdraw'],
    rejected: ['resubmit']
  };

  var currentRow = null;
  var pendingAction = null;
  var lastTrigger = null;
  var detailTimer = 0;
  var confirmTimer = 0;
  var toastTimer = 0;
  var focusTimer = 0;

  function showToast(message) {
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function openLayer(layer) {
    window.clearTimeout(layer === detailLayer ? detailTimer : confirmTimer);
    layer.hidden = false;
    void layer.offsetWidth;
    layer.classList.add('is-open');
    main.classList.add('is-locked');
  }

  function closeLayer(layer, onClosed) {
    if (!layer.classList.contains('is-open')) return;

    window.clearTimeout(layer === detailLayer ? detailTimer : confirmTimer);
    layer.classList.remove('is-open');

    var timer = window.setTimeout(function () {
      layer.hidden = true;
      if (onClosed) onClosed();
    }, 220);

    if (layer === detailLayer) detailTimer = timer;
    else confirmTimer = timer;
  }

  function setDetailActions(status) {
    var visible = actionsByStatus[status] || [];
    detailSheet.querySelectorAll('[data-detail-action]').forEach(function (button) {
      var action = button.getAttribute('data-detail-action');
      button.hidden = action !== 'delete' && visible.indexOf(action) === -1;
    });
  }

  function focusDetail() {
    var target = detailSheet.querySelector('.sheet-action:not([hidden])') ||
      detailSheet.querySelector('.sheet-close');
    window.clearTimeout(focusTimer);
    focusTimer = window.setTimeout(function () {
      if (detailLayer.classList.contains('is-open') && target) {
        target.focus({ preventScroll: true });
      }
    }, 60);
  }

  function openDetail(row) {
    if (!row) return;

    currentRow = row;
    lastTrigger = row;
    var avatar = row.querySelector('.avatar');
    var status = row.getAttribute('data-status');

    detailAvatar.textContent = avatar ? avatar.textContent : '';
    detailAvatar.className = 'avatar ' + (avatar ? avatar.className.replace('avatar ', '') : 'avatar--default');
    detailName.textContent = row.getAttribute('data-name') || '';
    detailRelation.textContent = row.getAttribute('data-relation') || '';
    detailPhone.textContent = row.getAttribute('data-phone') || '';
    detailAuthType.textContent = row.getAttribute('data-auth-type') || '长期授权';
    detailTag.textContent = row.getAttribute('data-status-label') || '';
    detailTag.className = statusTagClass[status] || 'tag';

    setDetailActions(status);
    openLayer(detailLayer);
    focusDetail();
  }

  function closeDetail() {
    closeLayer(detailLayer, function () {
      if (lastTrigger && !lastTrigger.hidden) {
        lastTrigger.focus({ preventScroll: true });
      }
    });
    main.classList.remove('is-locked');
  }

  function askConfirm(action) {
    if (!currentRow) return;

    pendingAction = action;
    var name = currentRow.getAttribute('data-name') || '该授权人';
    var isDelete = action === 'delete';

    confirmTitle.textContent = isDelete ? '删除' + name + '？' : '暂停' + name + '的接送授权？';
    confirmDesc.textContent = isDelete
      ? '删除后' + name + '将从名单移除；若今日已指定该接走人，接送码会失效。'
      : '暂停后' + name + '暂时不能接走孩子，可随时恢复。';
    confirmOk.textContent = isDelete ? '确认删除' : '确认暂停';
    confirmOk.className = isDelete
      ? 'sheet-action sheet-action--danger'
      : 'sheet-action sheet-action--primary';

    openLayer(confirmLayer);
    window.clearTimeout(focusTimer);
    focusTimer = window.setTimeout(function () {
      if (confirmLayer.classList.contains('is-open')) {
        confirmLayer.querySelector('.sheet [data-close-confirm]').focus({ preventScroll: true });
      }
    }, 60);
  }

  function closeConfirm() {
    closeLayer(confirmLayer);
    pendingAction = null;
  }

  rows.forEach(function (row) {
    row.addEventListener('click', function () {
      openDetail(row);
    });
  });

  detailLayer.addEventListener('click', function (event) {
    if (event.target.closest('[data-close-detail]')) {
      closeDetail();
    }
  });

  confirmLayer.addEventListener('click', function (event) {
    if (event.target.closest('[data-close-confirm]')) {
      closeConfirm();
    }
  });

  detailSheet.addEventListener('click', function (event) {
    var button = event.target.closest('[data-detail-action]');
    if (!button) return;

    var action = button.getAttribute('data-detail-action');
    if (action === 'edit') {
      window.location.href = '../20-pickup-authorized-create/index.html?mode=edit';
      return;
    }

    if (action === 'delete' || action === 'pause') {
      askConfirm(action);
      return;
    }

    var name = currentRow ? currentRow.getAttribute('data-name') : '';
    closeDetail();
    showToast(action === 'withdraw'
      ? '已撤回' + name + '的申请'
      : '已重新提交' + name + '的授权申请');
  });

  confirmOk.addEventListener('click', function () {
    if (!currentRow) return;

    var name = currentRow.getAttribute('data-name') || '该授权人';
    var action = pendingAction;

    closeConfirm();

    if (action === 'delete') {
      var row = currentRow;
      currentRow = null;
      closeDetail();
      row.hidden = true;
      showToast(name + '已从名单移除');
      return;
    }

    closeDetail();
    showToast('已暂停' + name + '的接送授权');
  });

  document.addEventListener('keydown', function (event) {
    if (event.key !== 'Escape') return;

    if (confirmLayer.classList.contains('is-open')) {
      event.preventDefault();
      closeConfirm();
      return;
    }

    if (detailLayer.classList.contains('is-open')) {
      event.preventDefault();
      closeDetail();
    }
  });

  if (state === 'detail' && rows.length) {
    openDetail(rows[0]);
  }
})();
