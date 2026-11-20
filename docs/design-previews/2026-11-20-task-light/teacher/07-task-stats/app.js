(function () {
  'use strict';

  var names = ['乐乐', '朵朵', '小宇', '辰辰', '安然', '果果', '阳阳', '豆豆'];
  var rows = Array.prototype.slice.call(document.querySelectorAll('.student-row'));
  var input = document.getElementById('chat-input');
  var importButton = document.getElementById('import-btn');
  var result = document.getElementById('import-result');
  var resultCopy = document.getElementById('result-copy');
  var undoButton = document.getElementById('undo-import');
  var statDone = document.getElementById('stat-done');
  var statParent = document.getElementById('stat-parent');
  var statChat = document.getElementById('stat-chat');
  var rosterNote = document.querySelector('.roster-note');
  var toast = document.getElementById('toast');
  var lastMarked = [];
  var toastTimer = 0;

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function countSources() {
    var parent = 0;
    var chat = 0;
    var none = 0;
    rows.forEach(function (row) {
      var source = row.getAttribute('data-source');
      if (source === 'parent') parent += 1;
      else if (source === 'chat') chat += 1;
      else none += 1;
    });
    if (statDone) statDone.textContent = String(parent + chat);
    if (statParent) statParent.textContent = String(parent);
    if (statChat) statChat.textContent = String(chat);
    if (rosterNote) {
      rosterNote.textContent = '已记录：家长完成 ' + parent + ' · 聊天记录 ' + chat + ' · 未完成 ' + none;
    }
    return { parent: parent, chat: chat, none: none };
  }

  function rowFor(name) {
    return rows.find(function (row) {
      return row.getAttribute('data-student') === name;
    });
  }

  function applyImport() {
    var text = input ? input.value : '';
    var marked = [];
    names.forEach(function (name) {
      var row = rowFor(name);
      if (!row || !text.includes(name)) return;
      if (row.getAttribute('data-source') !== 'none') return;
      row.setAttribute('data-source', 'chat');
      var status = row.querySelector('.student-status');
      if (status) status.textContent = '聊天记录';
      marked.push(name);
    });

    lastMarked = marked;
    countSources();

    if (result) result.hidden = false;
    if (resultCopy) {
      resultCopy.textContent = marked.length
        ? '已自动标记 ' + marked.length + ' 人：' + marked.join('、')
        : '没有识别到新的孩子';
    }
    if (undoButton) undoButton.hidden = marked.length === 0;
    if (marked.length) showToast('已根据群聊记录自动标记 ' + marked.length + ' 人');
  }

  function undoImport() {
    lastMarked.forEach(function (name) {
      var row = rowFor(name);
      if (!row) return;
      row.setAttribute('data-source', 'none');
      var status = row.querySelector('.student-status');
      if (status) status.textContent = '未完成';
    });
    lastMarked = [];
    countSources();
    if (result) result.hidden = true;
    showToast('已撤销本次导入');
  }

  if (importButton) importButton.addEventListener('click', applyImport);
  if (undoButton) undoButton.addEventListener('click', undoImport);

  // 粘贴后自动识别；按钮保留为显式入口。
  if (input) {
    input.addEventListener('paste', function () {
      window.setTimeout(applyImport, 80);
    });
  }

  countSources();
  if (new URLSearchParams(window.location.search).get('state') === 'marked') applyImport();

  var back = document.getElementById('back');
  if (back) back.addEventListener('click', function () { window.history.back(); });
})();
