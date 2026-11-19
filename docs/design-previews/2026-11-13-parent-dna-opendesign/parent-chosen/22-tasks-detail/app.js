/* 22 任务详情：默认展示已打卡；?state=unopened 展示未打开状态和打卡交互。 */
(function () {
  'use strict';

  var pending = document.getElementById('checkin-pending');
  var complete = document.getElementById('checkin-complete');
  var openButton = document.getElementById('checkin-open');
  var form = document.getElementById('checkin-form');
  var submitButton = document.getElementById('checkin-submit');
  var status = document.getElementById('checkin-status');
  var weekMeta = document.getElementById('week-meta');
  var photo = document.getElementById('checkin-photo');

  if (!pending || !complete || !openButton || !form || !submitButton || !status || !weekMeta) return;

  function openCheckin() {
    form.hidden = false;
    openButton.hidden = true;
    var input = document.getElementById('checkin-input');
    if (input) input.focus();
  }

  function submitCheckin() {
    pending.hidden = true;
    complete.hidden = false;
    status.textContent = '已完成 · 今天 20:05';
    weekMeta.textContent = '本周 3 / 3 天 · 今天已打卡';
    if (photo) photo.hidden = true;
  }

  openButton.addEventListener('click', openCheckin);
  submitButton.addEventListener('click', submitCheckin);

  var state = new URLSearchParams(window.location.search).get('state');
  if (state === 'unopened' || state === 'unopened-open') {
    complete.hidden = true;
    pending.hidden = false;
  }
  if (state === 'unopened-open') openCheckin();
})();
