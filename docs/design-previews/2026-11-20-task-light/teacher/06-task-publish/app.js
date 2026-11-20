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
  var posterPanel = document.getElementById('poster-panel');
  var posterTitle = document.getElementById('poster-title');
  var posterSave = document.getElementById('poster-save');
  var posterClose = document.getElementById('poster-close');

  function openPoster() {
    if (posterPanel) posterPanel.hidden = false;
    if (posterTitle && shareTitle) posterTitle.textContent = shareTitle.textContent;
  }

  if (shareGroup) shareGroup.addEventListener('click', function () { showToast('已打开班级群分享'); });
  if (savePoster) savePoster.addEventListener('click', openPoster);
  if (posterSave) posterSave.addEventListener('click', function () { showToast('海报已保存到相册（含小程序码）'); });
  if (posterClose) posterClose.addEventListener('click', function () { if (posterPanel) posterPanel.hidden = true; });

  var shareClose = document.getElementById('share-close');
  if (shareClose) {
    shareClose.addEventListener('click', function () {
      if (panel) panel.hidden = true;
      if (posterPanel) posterPanel.hidden = true;
    });
  }

  var viewStats = document.getElementById('view-stats');
  if (viewStats) viewStats.addEventListener('click', function () { window.location.href = '../07-task-stats/index.html'; });

  var state = new URLSearchParams(window.location.search).get('state');
  if (state === 'published' || state === 'poster') {
    if (panel) panel.hidden = false;
    if (state === 'poster') openPoster();
  }

  var back = document.getElementById('back');
  if (back) back.addEventListener('click', function () { window.history.back(); });
})();
