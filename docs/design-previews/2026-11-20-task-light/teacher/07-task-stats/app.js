(function () {
  'use strict';

  var names = ['乐乐', '朵朵', '小宇', '辰辰', '安然', '果果', '阳阳', '豆豆'];
  var rows = Array.prototype.slice.call(document.querySelectorAll('.student-row'));
  var input = document.getElementById('chat-input');
  var importButton = document.getElementById('import-btn');
  var result = document.getElementById('import-result');
  var resultCopy = document.getElementById('result-copy');
  var undoButton = document.getElementById('undo-import');
  var reviewBox = document.getElementById('import-review');
  var reviewRows = document.getElementById('review-rows');
  var unmatched = document.getElementById('import-unmatched');
  var statDone = document.getElementById('stat-done');
  var statParent = document.getElementById('stat-parent');
  var statChat = document.getElementById('stat-chat');
  var rosterNote = document.querySelector('.roster-note');
  var toast = document.getElementById('toast');
  var lastMarked = [];
  var lastNegative = [];
  var reviewItems = [];
  var toastTimer = 0;

  var positivePattern = /完成|做了|打卡|看完|看了|好了|画完|读完|整理好/;
  var negativePattern = /请假|没做|未完成|没完成|来不及|不做|缺席|补做|明天/;
  var ambiguousPattern = /收到|好的|知道了|ok|OK|👌/;

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }

  function rowFor(name) {
    return rows.find(function (row) {
      return row.getAttribute('data-student') === name;
    });
  }

  function statusFor(row) {
    return row.querySelector('.student-status');
  }

  function isDone(row) {
    var source = row.getAttribute('data-source');
    return source === 'parent' || source === 'chat' || source === 'manual';
  }

  function setRow(row, source, status, flag) {
    if (!row) return;
    row.setAttribute('data-source', source);
    if (flag) row.setAttribute('data-flag', flag);
    else row.removeAttribute('data-flag');
    var statusNode = statusFor(row);
    if (statusNode) statusNode.textContent = status;
  }

  function countSources() {
    var parent = 0;
    var chat = 0;
    var manual = 0;
    var none = 0;
    var leave = 0;
    rows.forEach(function (row) {
      var source = row.getAttribute('data-source');
      if (source === 'parent') parent += 1;
      else if (source === 'chat') chat += 1;
      else if (source === 'manual') manual += 1;
      else none += 1;
      if (row.getAttribute('data-flag') === 'leave') leave += 1;
    });
    if (statDone) statDone.textContent = String(parent + chat + manual);
    if (statParent) statParent.textContent = String(parent);
    if (statChat) statChat.textContent = String(chat);
    if (rosterNote) {
      rosterNote.textContent = '已记录：家长完成 ' + parent + ' · 聊天记录 ' + chat +
        ' · 手动 ' + manual + ' · 未完成 ' + none +
        (leave ? '（含请假 ' + leave + '）' : '') +
        '。识别有误？点孩子可手动标记或取消。';
    }
    return { parent: parent, chat: chat, manual: manual, none: none, leave: leave };
  }

  function classify(line) {
    if (negativePattern.test(line)) return 'negative';
    if (positivePattern.test(line)) return 'positive';
    if (ambiguousPattern.test(line)) return 'ambiguous';
    return 'none';
  }

  function classifyLines(lines) {
    var kinds = lines.map(classify);
    if (kinds.indexOf('positive') !== -1) return 'positive';
    if (kinds.indexOf('negative') !== -1) return 'negative';
    if (kinds.indexOf('ambiguous') !== -1) return 'ambiguous';
    return 'none';
  }

  function updateImportSummary() {
    if (!result) return;
    var parts = [];
    if (lastMarked.length) parts.push('已标记 ' + lastMarked.length);
    if (reviewItems.length) parts.push('待确认 ' + reviewItems.length);
    if (lastNegative.length) parts.push('未完成 ' + lastNegative.length);
    if (resultCopy) {
      resultCopy.textContent = parts.length ? parts.join(' · ') : '没有识别到完成消息';
    }
    if (undoButton) undoButton.hidden = lastMarked.length === 0 && lastNegative.length === 0;
  }

  function renderUnmatched(unmatchedNames) {
    if (!unmatched) return;
    if (!unmatchedNames.length) {
      unmatched.hidden = true;
      unmatched.textContent = '';
      return;
    }
    var shown = unmatchedNames.slice(0, 6).join('、');
    if (unmatchedNames.length > 6) shown += ' 等 ' + unmatchedNames.length + ' 人';
    unmatched.hidden = false;
    unmatched.textContent = '未提及 ' + unmatchedNames.length + ' 人：' + shown;
  }

  function renderReview() {
    if (!reviewBox || !reviewRows) return;
    reviewRows.textContent = '';
    if (!reviewItems.length) {
      reviewBox.hidden = true;
      return;
    }
    reviewBox.hidden = false;
    reviewItems.forEach(function (item) {
      var row = document.createElement('div');
      row.className = 'review-row';

      var copy = document.createElement('span');
      copy.className = 'review-copy';
      var name = document.createElement('strong');
      name.textContent = item.name;
      var quote = document.createElement('span');
      quote.textContent = '“' + item.line + '”';
      copy.appendChild(name);
      copy.appendChild(quote);

      var actions = document.createElement('span');
      actions.className = 'review-actions';

      var yes = document.createElement('button');
      yes.type = 'button';
      yes.className = 'review-btn review-btn--on';
      yes.textContent = '算完成';
      yes.addEventListener('click', function () { acceptReview(item.name); });

      var no = document.createElement('button');
      no.type = 'button';
      no.className = 'review-btn';
      no.textContent = '不算';
      no.addEventListener('click', function () { rejectReview(item.name); });

      actions.appendChild(yes);
      actions.appendChild(no);
      row.appendChild(copy);
      row.appendChild(actions);
      reviewRows.appendChild(row);
    });
  }

  function removeReview(name) {
    reviewItems = reviewItems.filter(function (item) { return item.name !== name; });
    renderReview();
  }

  function acceptReview(name) {
    var row = rowFor(name);
    if (row) setRow(row, 'chat', '聊天记录');
    if (lastMarked.indexOf(name) === -1) lastMarked.push(name);
    removeReview(name);
    countSources();
    updateImportSummary();
    showToast('已把' + name + '算作完成');
  }

  function rejectReview(name) {
    var row = rowFor(name);
    if (row) setRow(row, 'none', '未完成');
    removeReview(name);
    countSources();
    updateImportSummary();
    showToast('已保留' + name + '未完成');
  }

  function toggleRow(row) {
    if (!row) return;
    var name = row.getAttribute('data-student') || '';
    if (isDone(row)) {
      setRow(row, 'none', '未完成');
      showToast('已取消' + name + '的完成标记');
    } else {
      setRow(row, 'manual', '手动标记');
      showToast('已手动标记' + name);
    }
    countSources();
  }

  function applyImport() {
    var text = input ? input.value.trim() : '';
    if (!text) {
      if (result) result.hidden = true;
      if (reviewBox) reviewBox.hidden = true;
      if (unmatched) unmatched.hidden = true;
      showToast('先粘贴群聊文字');
      return;
    }

    var lines = text.split(/\n+/).map(function (line) { return line.trim(); }).filter(Boolean);
    var marked = [];
    var negative = [];
    var pending = [];
    var mentioned = [];

    names.forEach(function (name) {
      var row = rowFor(name);
      if (!row) return;
      var nameLines = lines.filter(function (line) { return line.indexOf(name) !== -1; });
      if (!nameLines.length) return;
      mentioned.push(name);
      if (isDone(row)) return;

      var kind = classifyLines(nameLines);
      if (kind === 'positive') {
        setRow(row, 'chat', '聊天记录');
        marked.push(name);
      } else if (kind === 'negative') {
        var leaveLine = nameLines.find(function (item) { return item.indexOf('请假') !== -1; });
        if (leaveLine) setRow(row, 'none', '请假', 'leave');
        else setRow(row, 'none', '未完成');
        negative.push(name);
      } else if (kind === 'ambiguous') {
        var line = nameLines.find(function (item) { return classify(item) === 'ambiguous'; }) || nameLines[0];
        pending.push({ name: name, line: line });
      }
    });

    lastMarked = marked;
    lastNegative = negative;
    reviewItems = pending;
    countSources();
    renderReview();
    renderUnmatched(names.filter(function (name) {
      var row = rowFor(name);
      return row && mentioned.indexOf(name) === -1 && !isDone(row) &&
        row.getAttribute('data-flag') !== 'leave';
    }));
    if (result) result.hidden = false;
    updateImportSummary();
    if (marked.length) showToast('已根据群聊记录自动标记 ' + marked.length + ' 人');
    else showToast('没有找到明确的完成消息，请确认或手动调整');
  }

  function undoImport() {
    lastMarked.forEach(function (name) {
      var row = rowFor(name);
      if (!row) return;
      setRow(row, 'none', '未完成');
    });
    lastNegative.forEach(function (name) {
      var row = rowFor(name);
      if (!row) return;
      setRow(row, 'none', '未完成');
    });
    lastMarked = [];
    lastNegative = [];
    reviewItems = [];
    renderReview();
    renderUnmatched([]);
    countSources();
    if (result) result.hidden = true;
    showToast('已撤销本次导入');
  }

  rows.forEach(function (row) {
    row.setAttribute('role', 'button');
    row.setAttribute('tabindex', '0');
    row.setAttribute('aria-label', (row.getAttribute('data-student') || '孩子') + '，点击手动调整完成状态');
    row.addEventListener('click', function () { toggleRow(row); });
    row.addEventListener('keydown', function (event) {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        toggleRow(row);
      }
    });
  });

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
