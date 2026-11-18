/* 亲子任务 · 任务积分提案（V2）
 * 预览态：?variant=points / ?state=empty
 * 只做本地交互，不引入框架、不发网络请求。
 * 提案：需 06-task 恢复 points/streak。
 */
(function () {
  'use strict';

  var params = new URLSearchParams(window.location.search);
  var app = document.getElementById('app');
  var emptyMode = params.get('state') === 'empty';

  if (app && params.get('variant') === 'points') {
    app.dataset.variant = 'points';
  }

  var viewDefault = document.getElementById('view-default');
  var viewEmpty = document.getElementById('view-empty');

  if (viewDefault && viewEmpty) {
    viewDefault.hidden = emptyMode;
    viewEmpty.hidden = !emptyMode;
  }

  /* ---------- toast ---------- */

  var toast = document.getElementById('toast');
  var toastTimer = null;

  function showToast(message) {
    if (!toast) { return; }
    toast.textContent = message;
    toast.classList.add('is-on');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-on');
    }, 2000);
  }

  /* ---------- filter chips ---------- */

  var chips = Array.prototype.slice.call(document.querySelectorAll('.chip'));
  var rows = Array.prototype.slice.call(document.querySelectorAll('.task-row'));
  var taskCard = document.getElementById('task-card');
  var filterEmpty = document.getElementById('filter-empty');
  var taskCount = document.getElementById('task-count');

  function applyFilter(filter) {
    var visible = 0;

    rows.forEach(function (row) {
      var show = filter === 'all' || row.getAttribute('data-status') === filter;
      row.hidden = !show;
      if (show) { visible += 1; }
    });

    if (taskCard) { taskCard.hidden = visible === 0; }
    if (filterEmpty) { filterEmpty.hidden = visible !== 0; }
    if (taskCount) { taskCount.textContent = '共 ' + visible + ' 项'; }

    chips.forEach(function (chip) {
      var active = chip.getAttribute('data-filter') === filter;
      chip.classList.toggle('is-active', active);
      chip.setAttribute('aria-pressed', active ? 'true' : 'false');
    });
  }

  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      applyFilter(chip.getAttribute('data-filter'));
    });
  });

  applyFilter('pending');

  /* ---------- class wall / back ---------- */

  Array.prototype.slice.call(document.querySelectorAll('[data-wall]')).forEach(function (el) {
    el.addEventListener('click', function () {
      showToast('班级墙 · 今天 8 位小朋友完成了任务');
    });
  });

  var backBtn = document.getElementById('back-btn');
  if (backBtn) {
    backBtn.addEventListener('click', function () {
      showToast('返回首页');
    });
  }
}());
