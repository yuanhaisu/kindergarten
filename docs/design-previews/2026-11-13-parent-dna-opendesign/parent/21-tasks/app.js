/* 亲子任务 · 预览态切换
 * 只处理：?state=empty / 筛选 chip / 打卡完成反馈 / 班级墙提示
 */
(function () {
  'use strict';

  var params = new URLSearchParams(window.location.search);
  var emptyMode = params.get('state') === 'empty';
  var viewDefault = document.getElementById('view-default');
  var viewEmpty = document.getElementById('view-empty');

  viewDefault.hidden = emptyMode;
  viewEmpty.hidden = !emptyMode;

  /* ---------- 筛选 ---------- */

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

    taskCard.hidden = visible === 0;
    filterEmpty.hidden = visible !== 0;
    taskCount.textContent = '共 ' + visible + ' 项';

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

  /* ---------- toast ---------- */

  var toast = document.getElementById('toast');
  var toastTimer = null;

  function showToast(message) {
    toast.textContent = message;
    toast.classList.add('is-on');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-on');
    }, 2000);
  }

  Array.prototype.slice.call(document.querySelectorAll('[data-wall]')).forEach(function (el) {
    el.addEventListener('click', function () {
      showToast('班级墙 · 今天 18 位小朋友完成了任务');
    });
  });

  /* ---------- 去打卡 ---------- */

  var checkinBtn = document.getElementById('checkin-btn');
  var todayCard = document.querySelector('.today-card');
  var streak = document.getElementById('streak');
  var streakThird = document.getElementById('streak-third');
  var todayMeta = document.getElementById('today-meta');
  var ghostIdle = document.getElementById('ghost-idle');
  var ghostDone = document.getElementById('ghost-done');
  var checked = false;

  checkinBtn.addEventListener('click', function () {
    if (checked) { return; }
    checked = true;

    todayCard.classList.add('is-checked');
    streakThird.classList.remove('is-off');
    streakThird.classList.add('is-on');
    streak.setAttribute('aria-label', '本周已完成 3 天');
    todayMeta.textContent = '本周 3 / 3 天 · 今天已打卡';

    ghostIdle.hidden = true;
    ghostDone.hidden = false;
    checkinBtn.disabled = true;

    showToast('打卡成功 · +10 积分');
  });
}());
