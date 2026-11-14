(function () {
  'use strict';

  var toast = document.getElementById('toast');
  var hideTimer = null;

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(hideTimer);
    hideTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function bindAction(id, message) {
    var el = document.querySelector('[data-od-id="' + id + '"]');
    if (!el) return;
    el.addEventListener('click', function () {
      showToast(message);
    });
  }

  bindAction('btn-save-all', '32 张照片已保存到相册');
  bindAction('btn-share', '分享卡片已生成');
})();
