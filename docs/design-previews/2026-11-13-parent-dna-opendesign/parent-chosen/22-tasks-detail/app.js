/* 22 任务详情：待完成 → 展开打卡 → 已提交（等老师回应）。
 * 图片与文字都选填；状态由 ?state=pending / pending-open 驱动，兼容旧的 unopened 参数。
 */
(function () {
  'use strict';

  var pending = document.getElementById('checkin-pending');
  var complete = document.getElementById('checkin-complete');
  var openButton = document.getElementById('checkin-open');
  var cancelButton = document.getElementById('checkin-cancel');
  var form = document.getElementById('checkin-form');
  var submitButton = document.getElementById('checkin-submit');
  var status = document.getElementById('checkin-status');
  var hint = document.getElementById('checkin-hint');
  var currentText = document.getElementById('current-checkin-text');
  var photo = document.getElementById('checkin-photo');
  var photoPreview = document.getElementById('checkin-photo-preview');
  var addPhoto = document.getElementById('checkin-add-photo');
  var addPhotoLabel = document.getElementById('checkin-add-photo-label');
  var input = document.getElementById('checkin-input');
  var count = document.getElementById('checkin-count');
  var historySection = document.getElementById('submission-history');
  var historyCount = historySection ? historySection.querySelector('.section-count') : null;
  var toast = document.getElementById('toast');
  var backButton = document.getElementById('back-btn');
  var toastTimer = 0;

  if (!pending || !complete || !openButton || !form || !submitButton || !status || !hint || !currentText || !historySection || !historyCount) return;

  function setExpanded(value) {
    openButton.setAttribute('aria-expanded', String(value));
  }

  function openCheckin() {
    form.hidden = false;
    openButton.hidden = true;
    setExpanded(true);
    if (input) input.focus();
  }

  function cancelCheckin() {
    form.hidden = true;
    openButton.hidden = false;
    setExpanded(false);
    openButton.focus();
  }

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2200);
  }

  function togglePhoto() {
    if (!photoPreview || !addPhotoLabel) return;
    if (photoPreview.querySelector('.photo')) {
      photoPreview.innerHTML = '';
      addPhotoLabel.textContent = '添加照片';
      return;
    }
    photoPreview.innerHTML = '<span class="photo photo--sage" role="img" aria-label="已添加的照片"></span>';
    addPhotoLabel.textContent = '移除照片';
  }

  function addHistory(note) {
    var first = historySection.querySelector('.submission');
    var article = document.createElement('article');
    article.className = 'submission';
    article.setAttribute('data-od-id', 'submission-today');

    var head = document.createElement('div');
    head.className = 'submission-head';
    var date = document.createElement('span');
    date.className = 'submission-date';
    date.textContent = '今天 · 11 月 12 日';
    var tag = document.createElement('span');
    tag.className = 'tag tag--pending';
    tag.textContent = '等老师回应';
    head.appendChild(date);
    head.appendChild(tag);

    var noteEl = document.createElement('p');
    noteEl.className = 'submission-note';
    noteEl.textContent = note;

    article.appendChild(head);
    article.appendChild(noteEl);
    historySection.insertBefore(article, first || null);
  }

  function submitCheckin() {
    var note = input ? input.value.trim() : '';
    var hasPhoto = !!(photoPreview && photoPreview.querySelector('.photo'));
    var displayNote = note || '（未填写文字）';

    pending.hidden = true;
    complete.hidden = false;
    status.textContent = '已提交 · 今天 20:05';
    hint.textContent = '已提交，等老师回应。';
    currentText.textContent = displayNote;
    if (photo) photo.hidden = !hasPhoto;

    addHistory(displayNote);
    var total = parseInt(historyCount.textContent.replace(/\D/g, ''), 10) || 0;
    historyCount.textContent = '共 ' + (total + 1) + ' 条';
    showToast('已提交，等老师回应');
  }

  openButton.addEventListener('click', openCheckin);
  if (cancelButton) cancelButton.addEventListener('click', cancelCheckin);
  if (addPhoto) addPhoto.addEventListener('click', togglePhoto);
  submitButton.addEventListener('click', submitCheckin);

  if (input && count) {
    input.addEventListener('input', function () {
      count.textContent = input.value.length + ' / 200';
    });
  }

  if (backButton) {
    backButton.addEventListener('click', function () {
      window.location.href = '../21-tasks/index.html';
    });
  }

  var state = new URLSearchParams(window.location.search).get('state');
  if (state === 'pending' || state === 'unopened' || state === 'pending-open' || state === 'unopened-open') {
    complete.hidden = true;
    pending.hidden = false;
  }
  if (state === 'pending-open' || state === 'unopened-open') openCheckin();
})();
