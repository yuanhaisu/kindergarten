(function () {
  'use strict';

  var MAX_PHOTOS = 3;
  var MAX_DESCRIPTION = 500;

  var form = document.getElementById('feedback-form');
  var chips = Array.prototype.slice.call(document.querySelectorAll('[data-feedback-type]'));
  var description = document.getElementById('feedback-description');
  var counter = document.getElementById('feedback-counter');
  var descriptionError = document.getElementById('description-error');
  var uploadRow = document.getElementById('upload-row');
  var uploadInput = document.getElementById('photo-input');
  var toast = document.getElementById('toast');
  var toastText = toast.querySelector('.toast__text');

  var photos = [];
  var toastTimer = null;
  var toastExitTimer = null;

  var plusIcon = [
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"',
    ' stroke-linecap="round" aria-hidden="true"><path d="M12 5.5v13M5.5 12h13"/></svg>'
  ].join('');

  var closeIcon = [
    '<svg viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6"',
    ' stroke-linecap="round" aria-hidden="true"><path d="M2.5 2.5l7 7M9.5 2.5l-7 7"/></svg>'
  ].join('');

  function setToast(message) {
    window.clearTimeout(toastTimer);
    window.clearTimeout(toastExitTimer);

    toast.classList.remove('is-hiding');
    toastText.textContent = message;
    toast.classList.add('is-visible');

    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
      toast.classList.add('is-hiding');
      toastExitTimer = window.setTimeout(function () {
        toast.classList.remove('is-hiding');
      }, 150);
    }, 2000);
  }

  function updateCounter() {
    var count = Math.min(description.value.length, MAX_DESCRIPTION);
    counter.textContent = count + ' / ' + MAX_DESCRIPTION;
  }

  function showDescriptionError() {
    description.classList.add('is-invalid');
    description.setAttribute('aria-invalid', 'true');
    descriptionError.hidden = false;
  }

  function clearDescriptionError() {
    description.classList.remove('is-invalid');
    description.removeAttribute('aria-invalid');
    descriptionError.hidden = true;
  }

  chips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      chips.forEach(function (item) {
        var isSelected = item === chip;
        item.classList.toggle('is-selected', isSelected);
        item.setAttribute('aria-pressed', isSelected ? 'true' : 'false');
      });
    });
  });

  description.addEventListener('input', function () {
    updateCounter();
    if (description.value.trim()) {
      clearDescriptionError();
    }
  });

  function addUploadSlot(label) {
    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'upload-slot';
    button.setAttribute('data-upload-trigger', '');
    button.setAttribute('aria-label', label);
    button.innerHTML = plusIcon;
    uploadRow.appendChild(button);
  }

  function renderUploads() {
    uploadRow.innerHTML = '';

    photos.forEach(function (photo, index) {
      var thumb = document.createElement('div');
      thumb.className = 'upload-thumb';

      var image = document.createElement('img');
      image.src = photo.url;
      image.alt = '已选照片 ' + (index + 1);
      thumb.appendChild(image);

      var remove = document.createElement('button');
      remove.type = 'button';
      remove.className = 'upload-thumb__remove';
      remove.setAttribute('aria-label', '移除第 ' + (index + 1) + ' 张照片');
      remove.innerHTML = closeIcon;
      remove.addEventListener('click', function () {
        window.URL.revokeObjectURL(photo.url);
        photos.splice(index, 1);
        renderUploads();
      });
      thumb.appendChild(remove);

      uploadRow.appendChild(thumb);
    });

    var openSlots = photos.length === 0 ? 2 : Math.max(0, MAX_PHOTOS - photos.length);

    for (var i = 0; i < openSlots; i += 1) {
      addUploadSlot(photos.length === 0 && i === 0 ? '上传照片' : '再添加一张照片');
    }
  }

  uploadRow.addEventListener('click', function (event) {
    var trigger = event.target.closest('[data-upload-trigger]');
    if (!trigger || !uploadRow.contains(trigger)) {
      return;
    }
    uploadInput.click();
  });

  uploadInput.addEventListener('change', function () {
    var selectedFiles = Array.prototype.slice.call(uploadInput.files || []);
    var available = Math.max(0, MAX_PHOTOS - photos.length);
    var images = selectedFiles.filter(function (file) {
      return file.type.indexOf('image/') === 0;
    });

    if (selectedFiles.length > 0 && images.length === 0) {
      setToast('请选择图片文件');
    }

    if (images.length > available) {
      setToast('最多上传 ' + MAX_PHOTOS + ' 张照片');
    }

    images.slice(0, available).forEach(function (file) {
      photos.push({
        url: window.URL.createObjectURL(file),
        name: file.name
      });
    });

    uploadInput.value = '';
    renderUploads();
  });

  form.addEventListener('submit', function (event) {
    event.preventDefault();

    if (!description.value.trim()) {
      showDescriptionError();
      description.focus();
      return;
    }

    clearDescriptionError();
    setToast('反馈已发送给园所管理员');
  });

  window.addEventListener('beforeunload', function () {
    photos.forEach(function (photo) {
      window.URL.revokeObjectURL(photo.url);
    });
  });

  updateCounter();
})();
