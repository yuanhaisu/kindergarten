/* 22 亲子任务 landing：完成型一键完成；观看型按资源进度记录。
 * ?state=pending | view | view-done | viewer | unbound | child | ended；默认已记录。
 * 不上传任何家长媒体；资源查看不再由单次点击直接完成。
 */
(function () {
  'use strict';

  var stateCard = document.getElementById('state-card');
  var doneState = document.getElementById('state-done');
  var pendingState = document.getElementById('state-pending');
  var unboundState = document.getElementById('state-unbound');
  var endedState = document.getElementById('state-ended');
  var resourceCard = document.getElementById('resource-card');
  var completeButton = document.getElementById('complete-btn');
  var resourceButton = document.getElementById('resource-btn');
  var progress = document.getElementById('resource-progress');
  var doneCopy = document.getElementById('done-copy');
  var footnote = document.getElementById('light-footnote');
  var backButton = document.getElementById('back-btn');
  var enrollLink = document.getElementById('enroll-link');
  var undoComplete = document.getElementById('undo-complete');
  var bindChild = document.getElementById('bind-child');
  var childSheet = document.getElementById('child-sheet');
  var childClose = document.getElementById('child-close');
  var pageChild = document.getElementById('page-child');
  var unboundTitle = document.getElementById('unbound-title');
  var unboundCopy = document.getElementById('unbound-copy');
  var unboundNote = document.getElementById('unbound-note');
  var enrollNote = document.getElementById('enroll-note');
  var taskLead = document.getElementById('task-lead');
  var viewerLayer = document.getElementById('viewer-layer');
  var viewerClose = document.getElementById('viewer-close');
  var viewerPlay = document.getElementById('viewer-play');
  var viewerFill = document.getElementById('viewer-fill');
  var viewerElapsed = document.getElementById('viewer-elapsed');
  var viewerStatus = document.getElementById('viewer-status');
  var toast = document.getElementById('toast');
  var toastTimer = 0;
  var playbackTimer = 0;
  var viewerProgress = 0;
  var viewerDone = false;

  if (!stateCard || !doneState || !pendingState || !unboundState || !endedState || !resourceCard) return;

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
    [doneState, pendingState, unboundState, endedState].forEach(function (el) {
      el.hidden = el !== target;
    });
  }

  function formatTime(seconds) {
    var whole = Math.max(0, Math.round(seconds));
    var minutes = Math.floor(whole / 60);
    var rest = whole % 60;
    return minutes + ':' + (rest < 10 ? '0' : '') + rest;
  }

  function setViewerProgress(percent, quiet) {
    var value = Math.max(0, Math.min(100, percent));
    viewerProgress = value;
    if (viewerFill) viewerFill.style.width = value + '%';
    if (viewerElapsed) viewerElapsed.textContent = formatTime((value / 100) * 150);
    if (value >= 90) markViewDone(quiet);
  }

  function markViewDone(quiet) {
    if (viewerDone) return;
    viewerDone = true;
    if (progress) progress.style.width = '100%';
    if (resourceButton) {
      resourceButton.textContent = '已记录';
      resourceButton.disabled = true;
    }
    if (viewerStatus) {
      viewerStatus.textContent = '已记录';
      viewerStatus.hidden = false;
    }
    if (viewerPlay) {
      viewerPlay.textContent = '已看完';
      viewerPlay.disabled = true;
    }
    if (footnote) footnote.textContent = '已记录 · 老师会在群里看到。';
    if (!quiet) showToast('已记录，老师会在群里看到');
  }

  function openViewer() {
    if (!viewerLayer) return;
    viewerLayer.hidden = false;
    if (viewerStatus) viewerStatus.hidden = !viewerDone;
    if (viewerPlay) {
      viewerPlay.disabled = viewerDone;
      viewerPlay.textContent = viewerDone ? '已看完' : (viewerProgress > 0 ? '继续播放' : '播放');
    }
    setViewerProgress(viewerProgress);
  }

  function closeViewer() {
    if (playbackTimer) {
      window.clearInterval(playbackTimer);
      playbackTimer = 0;
    }
    if (viewerLayer) viewerLayer.hidden = true;
    if (viewerPlay && !viewerDone) {
      viewerPlay.disabled = false;
      viewerPlay.textContent = viewerProgress > 0 ? '继续播放' : '播放';
    }
  }

  function startPlayback() {
    if (viewerDone || playbackTimer) return;
    var start = viewerProgress / 100;
    if (start >= 1) return;
    var duration = 6000 * (1 - start);
    var startedAt = Date.now();
    if (viewerPlay) {
      viewerPlay.disabled = true;
      viewerPlay.textContent = '播放中…';
    }
    playbackTimer = window.setInterval(function () {
      var ratio = Math.min(1, (Date.now() - startedAt) / duration);
      var percent = (start + (1 - start) * ratio) * 100;
      setViewerProgress(percent);
      if (ratio >= 1) {
        window.clearInterval(playbackTimer);
        playbackTimer = 0;
        if (viewerPlay && !viewerDone) {
          viewerPlay.disabled = true;
          viewerPlay.textContent = '已看完';
        }
      }
    }, 100);
  }

  function setTaskLead(child) {
    if (!taskLead) return;
    var who = child || '孩子';
    taskLead.innerHTML = '晚饭后问问' + who + '今天在幼儿园最开心的事，听他讲讲当时的心情。<br /><span class="hl">不用引导答案</span>，他愿意讲就好。';
  }

  function enterChildMode() {
    if (pageChild) pageChild.textContent = '选择孩子';
    if (unboundTitle) unboundTitle.textContent = '选择孩子';
    if (unboundCopy) unboundCopy.textContent = '这次任务记录给哪个孩子？';
    if (bindChild) bindChild.textContent = '选择孩子';
    if (unboundNote) unboundNote.textContent = '只影响本次任务；默认孩子可以在「我的」里切换。';
    if (enrollNote) enrollNote.hidden = true;
    if (childSheet) childSheet.hidden = false;
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

  if (completeButton) {
    completeButton.addEventListener('click', function () {
      showOnly(doneState);
      if (doneCopy) doneCopy.textContent = '今天 20:05 · 老师会在群里看到';
      if (footnote) footnote.textContent = '完成记录只用于班级统计。';
      showToast('已记录，老师会在群里看到');
    });
  }

  if (undoComplete) {
    undoComplete.addEventListener('click', function () {
      showOnly(pendingState);
      if (footnote) footnote.textContent = '记录后老师会在群里看到。';
      showToast('已撤销，可以重新记录');
    });
  }

  if (resourceButton) {
    resourceButton.addEventListener('click', openViewer);
  }

  if (viewerClose) viewerClose.addEventListener('click', closeViewer);
  if (viewerPlay) viewerPlay.addEventListener('click', startPlayback);

  if (backButton) {
    backButton.addEventListener('click', function () {
      window.location.href = '../21-tasks/index.html';
    });
  }

  if (bindChild) {
    bindChild.addEventListener('click', enterChildMode);
  }

  if (childClose) {
    childClose.addEventListener('click', function () {
      if (childSheet) childSheet.hidden = true;
    });
  }

  Array.prototype.slice.call(document.querySelectorAll('.child-option')).forEach(function (option) {
    option.addEventListener('click', function () {
      var child = option.getAttribute('data-child');
      var className = option.getAttribute('data-class');
      if (pageChild) pageChild.textContent = child + ' · ' + className;
      setTaskLead(child);
      if (childSheet) childSheet.hidden = true;
      stateCard.hidden = false;
      showOnly(pendingState);
      if (footnote) footnote.textContent = '记录后老师会在群里看到。';
      showToast('已选择' + child);
    });
  });

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
    if (pageChild) pageChild.textContent = '未绑定孩子';
    setTaskLead('');
    if (footnote) footnote.textContent = '绑定只用于记录任务完成情况。';
  } else if (state === 'child') {
    showOnly(unboundState);
    setTaskLead('');
    enterChildMode();
    if (footnote) footnote.textContent = '绑定后选择本次任务记录的孩子。';
  } else if (state === 'ended') {
    showOnly(endedState);
    stateCard.hidden = false;
    resourceCard.hidden = true;
    if (footnote) footnote.hidden = true;
  } else if (state === 'view' || state === 'view-done' || state === 'viewer') {
    setViewTask();
    if (state === 'view-done') {
      viewerDone = false;
      setViewerProgress(100, true);
    } else if (state === 'viewer') {
      openViewer();
    }
  }
})();
