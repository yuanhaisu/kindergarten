(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  var toastEl = document.getElementById('toast');
  var toastTimer = null;

  function toast(message) {
    toastEl.textContent = message;
    toastEl.classList.add('is-on');
    if (toastTimer) { clearTimeout(toastTimer); }
    toastTimer = setTimeout(function () {
      toastEl.classList.remove('is-on');
    }, 2000);
  }

  /* ---------- 遮罩层 ---------- */
  var lastFocus = null;

  function openOverlay(el, focusEl) {
    lastFocus = document.activeElement;
    el.classList.add('is-open');
    if (focusEl) {
      setTimeout(function () { focusEl.focus(); }, 120);
    }
  }

  function closeOverlay(el) {
    el.classList.remove('is-open');
    if (lastFocus && typeof lastFocus.focus === 'function') {
      try { lastFocus.focus({ preventScroll: true }); } catch (e) { lastFocus.focus(); }
    }
  }

  Array.prototype.forEach.call(document.querySelectorAll('[data-close]'), function (btn) {
    btn.addEventListener('click', function () {
      var target = document.getElementById(btn.getAttribute('data-close'));
      if (target) { closeOverlay(target); }
    });
  });

  Array.prototype.forEach.call(document.querySelectorAll('.overlay'), function (overlay) {
    overlay.addEventListener('click', function (event) {
      if (event.target === overlay) { closeOverlay(overlay); }
    });
  });

  /* ---------- 通用入口 toast ---------- */
  Array.prototype.forEach.call(document.querySelectorAll('[data-open]'), function (btn) {
    btn.addEventListener('click', function () {
      toast('已打开 ' + btn.getAttribute('data-open'));
    });
  });

  document.getElementById('bell').addEventListener('click', function () {
    toast('已打开 消息中心');
  });

  document.getElementById('view-all').addEventListener('click', function () {
    toast('已打开 更多');
  });

  /* ---------- 演示条切换状态时回滚页面内的临时态 ---------- */
  var approvedRows = [];

  function resetApprovals() {
    approvedRows.forEach(function (btn) {
      btn.removeAttribute('aria-disabled');
      btn.textContent = '同意';
    });
    approvedRows = [];
  }

  // 状态本身由 _shared/shell.js 写到 .mp-app[data-state]，页面只做自己的清理。
  document.addEventListener('mp:state', resetApprovals);

  document.addEventListener('keydown', function (event) {
    if (event.key !== 'Escape') { return; }
    Array.prototype.forEach.call(document.querySelectorAll('.overlay.is-open'), function (overlay) {
      closeOverlay(overlay);
    });
  });

  /* ---------- 待办：同意 / 拒绝 / 转发 / 去记录 ---------- */
  var rejectOverlay = document.getElementById('reject-overlay');
  var rejectInput = document.getElementById('reject-input');
  var rejectConfirm = document.getElementById('reject-confirm');
  var shareOverlay = document.getElementById('share-overlay');

  document.addEventListener('click', function (event) {
    var btn = event.target.closest ? event.target.closest('[data-action]') : null;
    if (!btn) { return; }

    var action = btn.getAttribute('data-action');

    if (action === 'approve') {
      if (btn.getAttribute('aria-disabled') === 'true') { return; }
      btn.textContent = '已同意';
      btn.setAttribute('aria-disabled', 'true');
      approvedRows.push(btn);
      toast('已同意请假申请');
      return;
    }

    if (action === 'reject') {
      openOverlay(rejectOverlay, rejectInput);
      return;
    }

    if (action === 'share') {
      openOverlay(shareOverlay, shareOverlay.querySelector('.sheet-item'));
      return;
    }

    if (action === 'go_record') {
      toast('已打开 添加成长记录');
    }
  });

  rejectConfirm.addEventListener('click', function () {
    if (rejectInput.value.trim() === '') {
      toast('请填写拒绝原因');
      rejectInput.focus();
      return;
    }
    rejectInput.value = '';
    closeOverlay(rejectOverlay);
    toast('已拒绝请假申请');
  });

  Array.prototype.forEach.call(shareOverlay.querySelectorAll('[data-share]'), function (item) {
    item.addEventListener('click', function () {
      var where = item.getAttribute('data-share');
      closeOverlay(shareOverlay);
      toast(where === 'group' ? '已分享到班级群' : '已转发给微信好友');
    });
  });

  /* ---------- 空态 / 错误态动作 ---------- */
  document.getElementById('empty-record').addEventListener('click', function () {
    toast('已打开 添加成长记录');
  });

  document.getElementById('retry').addEventListener('click', function () {
    // 走演示 chip 而不是直接改 data-state，保证 chip 的选中态同步（单一状态出口）。
    var normalChip = document.querySelector('.mp-demo-chip[data-mp-state="normal"]');
    if (normalChip) {
      normalChip.click();
    } else {
      app.setAttribute('data-state', 'normal');
      resetApprovals();
    }
    toast('班级数据已更新');
  });
})();
