(function () {
  'use strict';
  var empty = document.getElementById('view-empty');
  var isEmpty = new URLSearchParams(window.location.search).get('state') === 'empty';
  if (isEmpty && empty) {
    empty.hidden = false;
    var today = document.querySelector('[data-od-id="today-card"]');
    var section = document.querySelector('[data-od-id="task-section"]');
    var note = document.querySelector('.light-note');
    if (today) today.hidden = true;
    if (section) section.hidden = true;
    if (note) note.hidden = true;
  }

  var back = document.getElementById('back-btn');
  if (back) {
    back.addEventListener('click', function () {
      window.location.href = '../01-home/index.html';
    });
  }
})();
