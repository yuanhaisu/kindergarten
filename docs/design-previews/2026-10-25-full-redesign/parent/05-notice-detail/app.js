/* 养成星球 · 家长端 · 通知详情
 * 本页交互：返回提示、附件预览提示、确认回执、失败重试、演示状态复位。
 * 演示芯片的 data-state 由 _shared/shell.js 负责，这里只监听 mp:state。
 */
(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  var toast = document.getElementById('toast');
  var confirmBtn = document.getElementById('confirm-btn');
  var confirmLabel = confirmBtn ? confirmBtn.querySelector('.confirm-btn-label') : null;
  var toastTimer = null;
  var toastClearTimer = null;
  var confirmTimer = null;

  /* 轻提示：2 秒后消失，同一元素复用 -------------------------------- */
  function showToast(message) {
    if (!toast) return;
    if (toastTimer) window.clearTimeout(toastTimer);
    if (toastClearTimer) window.clearTimeout(toastClearTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
      toastClearTimer = window.setTimeout(function () {
        toast.textContent = '';
      }, 220);
    }, 2000);
  }

  /* 确认按钮回到初始文案与可用状态 ---------------------------------- */
  function setConfirmIdle() {
    if (confirmTimer) {
      window.clearTimeout(confirmTimer);
      confirmTimer = null;
    }
    if (confirmBtn) {
      confirmBtn.disabled = false;
    }
    if (confirmLabel) {
      confirmLabel.textContent = '确认收到';
    }
  }

  function resetToNormal() {
    setConfirmIdle();
    document.body.classList.remove('is-confirmed');
  }

  /* 返回：原型不真跳页 ---------------------------------------------- */
  var back = document.querySelector('.mp-back');
  if (back) {
    back.addEventListener('click', function () {
      showToast('返回通知列表');
    });
  }

  /* 附件：预览提示 + 短暂的描边反馈 --------------------------------- */
  var thumbs = Array.prototype.slice.call(document.querySelectorAll('.att-thumb'));
  thumbs.forEach(function (thumb) {
    thumb.addEventListener('click', function () {
      thumbs.forEach(function (item) {
        item.classList.remove('is-flash');
      });
      thumb.classList.add('is-flash');
      window.setTimeout(function () {
        thumb.classList.remove('is-flash');
      }, 520);
      showToast('打开图片预览 · 原型不联网加载图片');
    });
  });

  /* 确认收到：先禁用，0.8 秒后切到已回执 ---------------------------- */
  if (confirmBtn) {
    confirmBtn.addEventListener('click', function () {
      if (confirmBtn.disabled || document.body.classList.contains('is-confirmed')) return;

      confirmBtn.disabled = true;
      if (confirmLabel) {
        confirmLabel.textContent = '确认中…';
      }
      if (confirmTimer) {
        window.clearTimeout(confirmTimer);
      }

      confirmTimer = window.setTimeout(function () {
        confirmTimer = null;
        document.body.classList.add('is-confirmed');
        confirmBtn.disabled = false;
        if (confirmLabel) {
          confirmLabel.textContent = '确认收到';
        }
        confirmBtn.blur();
        showToast('已确认收到');
      }, 800);
    });
  }

  /* 重试：回到正常态 ------------------------------------------------ */
  var retryBtn = document.querySelector('.retry-btn');
  if (retryBtn) {
    retryBtn.addEventListener('click', function () {
      resetToNormal();
      app.dataset.state = 'normal';
      document.querySelectorAll('.mp-demo-chip').forEach(function (chip) {
        chip.setAttribute('aria-pressed', String(chip.dataset.mpState === 'normal'));
      });
      showToast('已重新加载');
    });
  }

  /* 演示条切状态：清掉未完成的确认；回到正常态时整体复位 ------------ */
  document.addEventListener('mp:state', function (event) {
    setConfirmIdle();
    var state = event.detail && event.detail.state;
    // 只有「已回执」演示位自带确认态；切到仅告知/未读等其它状态时必须清掉
    // 运行时确认，否则信息型纸面的已读条会被染绿、显出并不存在的确认时间。
    if (state !== 'empty') {
      document.body.classList.remove('is-confirmed');
    }
  });
})();
