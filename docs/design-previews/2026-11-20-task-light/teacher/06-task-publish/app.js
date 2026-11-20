(function () {
  'use strict';

  var typeButtons = Array.prototype.slice.call(document.querySelectorAll('[data-type]'));
  var resourceField = document.getElementById('resource-field');
  var publish = document.getElementById('publish');
  var panel = document.getElementById('share-panel');
  var shareTitle = document.getElementById('share-title');
  var toast = document.getElementById('toast');
  var timer = 0;

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(timer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    timer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  typeButtons.forEach(function (button) {
    button.addEventListener('click', function () {
      typeButtons.forEach(function (item) {
        var on = item === button;
        item.classList.toggle('is-on', on);
        item.setAttribute('aria-pressed', String(on));
      });
      if (resourceField) resourceField.hidden = button.getAttribute('data-type') !== 'view';
    });
  });

  if (publish) {
    publish.addEventListener('click', function () {
      if (panel) panel.hidden = false;
      if (shareTitle) {
        var title = document.getElementById('task-title');
        shareTitle.textContent = title && title.value.trim() ? title.value.trim() : '亲子任务';
      }
    });
  }

  var shareGroup = document.getElementById('share-group');
  var savePoster = document.getElementById('save-poster');
  if (shareGroup) shareGroup.addEventListener('click', function () { showToast('已打开班级群分享'); });
  if (savePoster) savePoster.addEventListener('click', function () { showToast('海报已保存'); });

  var shareClose = document.getElementById('share-close');
  if (shareClose) shareClose.addEventListener('click', function () { if (panel) panel.hidden = true; });

  var viewStats = document.getElementById('view-stats');
  if (viewStats) viewStats.addEventListener('click', function () { window.location.href = '../07-task-stats/index.html'; });

  if (new URLSearchParams(window.location.search).get('state') === 'published') {
    if (panel) panel.hidden = false;
  }

  var back = document.getElementById('back');
  if (back) back.addEventListener('click', function () { window.history.back(); });
})();
