/* 22 亲子任务 landing：完成型一键完成；观看型按资源进度记录。
 * ?state=pending | view | view-done | unbound；默认已记录。
 * 不上传任何家长媒体。
 */
(function () {
  'use strict';

  var stateCard = document.getElementById('state-card');
  var doneState = document.getElementById('state-done');
  var pendingState = document.getElementById('state-pending');
  var unboundState = document.getElementById('state-unbound');
  var resourceCard = document.getElementById('resource-card');
  var completeButton = document.getElementById('complete-btn');
  var resourceButton = document.getElementById('resource-btn');
  var progress = document.getElementById('resource-progress');
  var doneCopy = document.getElementById('done-copy');
  var footnote = document.getElementById('light-footnote');
  var backButton = document.getElementById('back-btn');
  var enrollLink = document.getElementById('enroll-link');
  var toast = document.getElementById('toast');
  var toastTimer = 0;

  if (!stateCard || !doneState || !pendingState || !unboundState || !resourceCard) return;

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2200);
  }

  function showOnly(target) {
    [doneState, pendingState, unboundState].forEach(function (el) {
      el.hidden = el !== target;
    });
  }

  function setViewTask() {
    var title = document.getElementById('task-title');
    var lead = document.getElementById('task-lead');
    var deadline = document.getElementById('task-deadline');
    var teacher = document.getElementById('task-teacher');
    var type = document.getElementById('task-type');
    if (title) title.innerHTML = '一起看秋天落叶';
    if (lead) lead.textContent = '和乐乐一起看 2 分钟短片，看看秋天树叶为什么会落下来。';
    if (deadline) deadline.textContent = '截止 11 月 24 日';
    if (teacher) teacher.textContent = '李老师';
    if (type) type.textContent = '观看型';
    stateCard.hidden = true;
    resourceCard.hidden = false;
    if (footnote) footnote.textContent = '观看进度会自动记录，不需要上传照片。';
  }

  function markViewDone() {
    if (progress) progress.style.width = '100%';
    if (resourceButton) {
      resourceButton.textContent = '已记录';
      resourceButton.disabled = true;
    }
    if (footnote) footnote.textContent = '已记录 · 老师会在群里看到。';
    showToast('已记录，老师会在群里看到');
  }

  if (completeButton) {
    completeButton.addEventListener('click', function () {
      showOnly(doneState);
      if (doneCopy) doneCopy.textContent = '今天 20:05 · 老师会在群里看到';
      if (footnote) footnote.textContent = '完成记录只用于班级统计。';
      showToast('已记录，老师会在群里看到');
    });
  }

  if (resourceButton) {
    resourceButton.addEventListener('click', markViewDone);
  }

  if (backButton) {
    backButton.addEventListener('click', function () {
      window.location.href = '../21-tasks/index.html';
    });
  }

  if (enrollLink) {
    enrollLink.addEventListener('click', function (event) {
      event.preventDefault();
      showToast('招生咨询请关注园所公众号');
    });
  }

  var state = new URLSearchParams(window.location.search).get('state');
  if (state === 'pending') {
    showOnly(pendingState);
    if (footnote) footnote.textContent = '记录后老师会在群里看到。';
  } else if (state === 'unbound') {
    showOnly(unboundState);
    if (footnote) footnote.textContent = '绑定只用于记录任务完成情况。';
  } else if (state === 'view' || state === 'view-done') {
    setViewTask();
    if (state === 'view-done') markViewDone();
  }
})();
