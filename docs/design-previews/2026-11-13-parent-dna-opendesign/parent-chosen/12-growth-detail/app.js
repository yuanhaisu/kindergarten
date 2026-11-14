(function () {
  'use strict';

  var likeBtn = document.getElementById('like-btn');
  var likeLabel = document.getElementById('like-label');
  var likeCountEl = document.getElementById('like-count');
  var commentBtn = document.getElementById('comment-btn');
  var barCountEl = document.getElementById('bar-comment-count');
  var headCountEl = document.getElementById('comment-count');
  var list = document.getElementById('comment-list');
  var main = document.querySelector('.mp-main');

  var scrim = document.getElementById('scrim');
  var sheet = document.getElementById('sheet');
  var closeBtn = document.getElementById('sheet-close');
  var input = document.getElementById('comment-input');
  var sendBtn = document.getElementById('send-comment');
  var charCountEl = document.getElementById('char-count');
  var toast = document.getElementById('toast');

  var liked = false;
  var likes = 8;
  var commentTotal = 2;
  var lastFocused = null;
  var toastTimer = 0;

  /* ---------- 点赞 ---------- */

  likeBtn.addEventListener('click', function () {
    liked = !liked;
    likes += liked ? 1 : -1;
    likeBtn.classList.toggle('is-liked', liked);
    likeBtn.setAttribute('aria-pressed', liked ? 'true' : 'false');
    likeLabel.textContent = liked ? '已赞' : '点赞';
    likeCountEl.textContent = String(likes);
  });

  /* ---------- 抽屉开合 ---------- */

  function openSheet() {
    lastFocused = document.activeElement;
    sheet.classList.add('is-open');
    scrim.classList.add('is-open');
    window.requestAnimationFrame(function () {
      input.focus({ preventScroll: true });
    });
  }

  function closeSheet() {
    if (!sheet.classList.contains('is-open')) return;
    sheet.classList.remove('is-open');
    scrim.classList.remove('is-open');
    if (lastFocused && typeof lastFocused.focus === 'function') {
      lastFocused.focus({ preventScroll: true });
    }
  }

  commentBtn.addEventListener('click', openSheet);
  closeBtn.addEventListener('click', closeSheet);
  scrim.addEventListener('click', closeSheet);

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') closeSheet();
  });

  // 焦点收在抽屉内
  sheet.addEventListener('keydown', function (e) {
    if (e.key !== 'Tab' || !sheet.classList.contains('is-open')) return;
    var nodes = Array.prototype.filter.call(
      sheet.querySelectorAll('button, textarea'),
      function (el) { return !el.disabled; }
    );
    if (!nodes.length) return;
    var first = nodes[0];
    var last = nodes[nodes.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  });

  /* ---------- 输入与发布 ---------- */

  input.addEventListener('input', function () {
    charCountEl.textContent = String(input.value.length);
    sendBtn.disabled = input.value.trim().length === 0;
  });

  function appendComment(text) {
    var li = document.createElement('li');
    li.className = 'comment';

    var avatar = document.createElement('span');
    avatar.className = 'avatar avatar--clay';
    avatar.setAttribute('aria-hidden', 'true');
    avatar.textContent = '妈';

    var body = document.createElement('div');
    body.className = 'comment__body';

    var meta = document.createElement('div');
    meta.className = 'comment__meta';

    var name = document.createElement('span');
    name.className = 'comment__name';
    name.textContent = '妈妈';

    var time = document.createElement('span');
    time.className = 'comment__time';
    time.textContent = '刚刚';

    var p = document.createElement('p');
    p.className = 'comment__text';
    p.textContent = text;

    meta.appendChild(name);
    meta.appendChild(time);
    body.appendChild(meta);
    body.appendChild(p);
    li.appendChild(avatar);
    li.appendChild(body);
    list.appendChild(li);
  }

  sendBtn.addEventListener('click', function () {
    var text = input.value.trim();
    if (!text) return;

    appendComment(text);
    commentTotal += 1;
    headCountEl.textContent = commentTotal + ' 条';
    barCountEl.textContent = String(commentTotal);

    closeSheet();
    input.value = '';
    charCountEl.textContent = '0';
    sendBtn.disabled = true;
    showToast('已发送');

    window.setTimeout(function () {
      main.scrollTop = main.scrollHeight;
    }, 320);
  });

  /* ---------- Toast ---------- */

  function showToast(message) {
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 2000);
  }
})();
