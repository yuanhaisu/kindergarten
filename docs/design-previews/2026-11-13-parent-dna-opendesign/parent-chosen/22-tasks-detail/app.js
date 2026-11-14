(function () {
  'use strict';

  var stepInputs = Array.prototype.slice.call(document.querySelectorAll('.check-input'));
  var stepCount = document.getElementById('step-count');
  var segments = Array.prototype.slice.call(document.querySelectorAll('.steps-progress__seg'));
  var note = document.getElementById('checkin-note');
  var counter = document.getElementById('note-counter');
  var submitBtn = document.getElementById('submit-btn');
  var timeLive = document.getElementById('time-left-live');
  var timeDone = document.getElementById('time-left-done');
  var addPhoto = document.getElementById('add-photo');
  var photoGrid = document.getElementById('photo-grid');
  var toast = document.getElementById('toast');
  var toastTimer = null;

  function refreshSteps() {
    var done = 0;
    stepInputs.forEach(function (input) {
      input.closest('.check-row').classList.toggle('is-done', input.checked);
      if (input.checked) done += 1;
    });
    stepCount.textContent = String(done);
    segments.forEach(function (seg, index) {
      seg.classList.toggle('is-done', index < done);
    });
  }

  function refreshCounter() {
    var length = note.value.length;
    counter.textContent = length + ' / 200';
    counter.classList.toggle('is-near', length >= 180);
  }

  function showToast(message) {
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function markSubmitted(withToast) {
    submitBtn.disabled = true;
    submitBtn.textContent = '今天已提交';
    timeLive.hidden = true;
    timeDone.hidden = false;
    if (withToast) showToast('已提交，老师晚上会看到');
  }

  stepInputs.forEach(function (input) {
    input.addEventListener('change', refreshSteps);
  });

  note.addEventListener('input', refreshCounter);

  submitBtn.addEventListener('click', function () {
    if (submitBtn.disabled) return;
    markSubmitted(true);
  });

  addPhoto.addEventListener('click', function () {
    var tile = document.createElement('span');
    tile.className = 'photo photo--soft';
    tile.setAttribute('role', 'img');
    tile.setAttribute('aria-label', '已添加的照片');
    photoGrid.insertBefore(tile, addPhoto);
    addPhoto.hidden = true;
    showToast('照片已添加');
  });

  refreshSteps();
  refreshCounter();

  if (new URLSearchParams(window.location.search).get('state') === 'done') {
    markSubmitted(false);
  }
})();
