(function () {
  'use strict';

  /* ?variant=signup（默认）| signup-closed（可选截止后态）
     ?state=normal | confirm | confirmed 为普通通知三态 */
  var variant = 'signup';
  var normalState = null;
  try {
    var search = new URLSearchParams(window.location.search);
    if (search.get('variant') === 'signup-closed') variant = 'signup-closed';
    var requestedState = search.get('state');
    if (requestedState === 'normal' || requestedState === 'confirm' || requestedState === 'confirmed') {
      normalState = requestedState;
    }
  } catch (err) {
    variant = 'signup';
    normalState = null;
  }
  document.documentElement.setAttribute('data-variant', normalState ? 'normal' : variant);

  var app = document.getElementById('app');
  var pageBar = document.querySelector('.page-bar');
  var main = document.getElementById('main');
  var actionBar = document.getElementById('action-bar');

  var cta = document.getElementById('signup-cta');
  var ctaLabel = document.getElementById('signup-cta-label');
  var cancelBtn = document.getElementById('cancel-signup');
  var signedNote = document.getElementById('signed-note');
  var deadlineText = document.getElementById('deadline-text');

  var overline = document.getElementById('notice-overline');
  var titleEl = document.getElementById('notice-title');
  var authorAvatar = document.getElementById('author-avatar');
  var authorName = document.getElementById('author-name');
  var authorRole = document.getElementById('author-role');
  var signupBody = document.getElementById('signup-body');
  var normalBody = document.getElementById('normal-body');
  var noticeHelp = document.getElementById('notice-help');
  var noticeStatus = document.getElementById('notice-status');
  var confirmNoticeBtn = document.getElementById('confirm-notice');
  var proposalTag = document.querySelector('.proposal-tag');
  var proposalNote = document.querySelector('.notice-foot__proposal');
  var deadlineCard = document.querySelector('.deadline');
  var attachmentCard = document.querySelector('.attachment');

  var layer = document.getElementById('signup-sheet-layer');
  var scrim = document.getElementById('sheet-scrim');
  var panel = document.getElementById('signup-sheet');
  var confirmBtn = document.getElementById('confirm-signup');
  var dismissBtn = document.getElementById('dismiss-sheet');
  var saveBtn = document.querySelector('[data-od-id="save-attachment"]');

  var toastEl = document.getElementById('toast');

  var isClosed = variant === 'signup-closed';
  var signedUp = false;
  var isOpen = false;
  var isClosing = false;
  var toastTimer = null;

  var prefersReduced = !!(window.matchMedia &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches);
  var closeDelay = prefersReduced ? 0 : 200;

  function showToast(message) {
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toastEl.classList.remove('is-visible');
    }, 2000);
  }

  var NORMAL_NOTICES = {
    normal: {
      title: '流感疫苗接种安排',
      overline: '园所通知 · 今天 09:40',
      avatar: '王',
      name: '王老师',
      role: '小一班 · 配班',
      status: '无需确认',
      help: '如有疑问，可以在班级群里直接问王老师。',
      paragraphs: [
        '各位家长：',
        '今天上午社区卫生服务中心到园为孩子们接种流感疫苗。<span class="hl">已交回执的孩子都已完成接种</span>，接种后留观 30 分钟，没有出现不适。',
        '还没交回执的家长，请明天让孩子把回执带来，我们会统一补交。',
        '如果孩子今天回家后有发热或精神不好，请及时告诉老师。'
      ]
    },
    confirm: {
      title: '本月家长会安排',
      overline: '园所通知 · 11 月 10 日',
      avatar: '李',
      name: '李老师',
      role: '小一班 · 班主任',
      status: '需确认',
      help: '如有疑问，可以在班级群里直接问李老师。',
      paragraphs: [
        '各位家长：',
        '11 月 21 日（周六）上午 10:00，在小一班教室召开本学期家长会，介绍课程安排和冬季保健。',
        '会议预计一小时，<span class="hl">请提前 10 分钟签到</span>。每家一位家长参加即可。',
        '如果当天不能来，请提前告诉李老师。'
      ]
    },
    confirmed: {
      title: '本月家长会安排',
      overline: '园所通知 · 11 月 10 日',
      avatar: '李',
      name: '李老师',
      role: '小一班 · 班主任',
      status: '已确认 · 11 月 12 日 10:20',
      help: '如有疑问，可以在班级群里直接问李老师。',
      paragraphs: [
        '各位家长：',
        '11 月 21 日（周六）上午 10:00，在小一班教室召开本学期家长会，介绍课程安排和冬季保健。',
        '会议预计一小时，<span class="hl">请提前 10 分钟签到</span>。每家一位家长参加即可。',
        '如果当天不能来，请提前告诉李老师。'
      ]
    }
  };

  function applyNormalState(state) {
    var data = NORMAL_NOTICES[state];
    if (!data) return;

    document.title = data.title + ' · 通知';
    overline.textContent = data.overline;
    titleEl.textContent = data.title;
    authorAvatar.textContent = data.avatar;
    authorName.textContent = data.name;
    authorRole.textContent = data.role;
    normalBody.innerHTML = data.paragraphs.map(function (text) {
      return '<p>' + text + '</p>';
    }).join('');
    signupBody.hidden = true;
    normalBody.hidden = false;
    noticeHelp.textContent = data.help;
    noticeStatus.textContent = data.status;
    noticeStatus.hidden = false;

    proposalTag.hidden = true;
    proposalNote.hidden = true;
    deadlineCard.hidden = true;
    attachmentCard.hidden = true;
    cta.hidden = true;
    cancelBtn.hidden = true;
    signedNote.hidden = true;
    layer.hidden = true;

    confirmNoticeBtn.hidden = state !== 'confirm';
  }

  function confirmNotice() {
    if (normalState !== 'confirm') return;
    normalState = 'confirmed';
    noticeStatus.textContent = NORMAL_NOTICES.confirmed.status;
    confirmNoticeBtn.hidden = true;
    showToast('已确认参加');
  }

  function setBackgroundInert(on) {
    [pageBar, main, actionBar].forEach(function (el) {
      if (!el) return;
      if (on) el.setAttribute('inert', '');
      else el.removeAttribute('inert');
    });
  }

  function focusEl(el) {
    if (!el) return;
    try {
      el.focus({ preventScroll: true });
    } catch (err) {
      el.focus();
    }
  }

  function setSignedUp(next) {
    signedUp = next;
    signedNote.hidden = !next;
    cancelBtn.hidden = !next;
    cta.disabled = next || isClosed;
    cta.classList.toggle('is-signed', next);

    if (next) {
      ctaLabel.textContent = '已报名';
      cta.removeAttribute('aria-haspopup');
    } else {
      ctaLabel.textContent = isClosed ? '报名已截止' : '我要报名';
      if (!isClosed) cta.setAttribute('aria-haspopup', 'dialog');
    }
  }

  function openSheet() {
    if (signedUp || isClosed || isOpen) return;

    window.clearTimeout(toastTimer);
    toastEl.classList.remove('is-visible');

    isOpen = true;
    isClosing = false;
    setBackgroundInert(true);
    app.classList.add('is-sheet-open');

    layer.hidden = false;
    void layer.offsetHeight;
    layer.classList.add('is-open');
    focusEl(panel);
  }

  function closeSheet(afterClose) {
    if (!isOpen || isClosing) return;

    isClosing = true;
    layer.classList.remove('is-open');
    app.classList.remove('is-sheet-open');

    window.setTimeout(function () {
      layer.hidden = true;
      isOpen = false;
      isClosing = false;
      setBackgroundInert(false);
      if (typeof afterClose === 'function') afterClose();
    }, closeDelay);
  }

  function dismissSheet() {
    closeSheet(function () {
      focusEl(cta);
    });
  }

  function applyClosedState() {
    isClosed = true;
    deadlineText.innerHTML = '报名已于<strong>本周四 18:00</strong>截止，不再补报。';
    cta.disabled = true;
    cta.classList.add('is-closed');
    cta.removeAttribute('aria-haspopup');
    ctaLabel.textContent = '报名已截止';
  }

  cta.addEventListener('click', function () {
    if (signedUp || isClosed) return;
    openSheet();
  });

  confirmBtn.addEventListener('click', function () {
    if (isClosing) return;

    closeSheet(function () {
      setSignedUp(true);
      showToast('报名成功');
      focusEl(cancelBtn);
    });
  });

  dismissBtn.addEventListener('click', dismissSheet);
  scrim.addEventListener('click', dismissSheet);

  cancelBtn.addEventListener('click', function () {
    if (!signedUp) return;
    setSignedUp(false);
    showToast('已取消报名');
    focusEl(cta);
  });

  if (saveBtn) {
    saveBtn.addEventListener('click', function () {
      showToast('活动海报已保存');
    });
  }

  layer.addEventListener('keydown', function (e) {
    if (!isOpen) return;

    if (e.key === 'Escape') {
      if (isClosing) return;
      e.preventDefault();
      dismissSheet();
      return;
    }

    if (e.key !== 'Tab' || isClosing) return;

    var focusables = Array.prototype.slice.call(
      panel.querySelectorAll('button:not([disabled])')
    );
    if (!focusables.length) return;

    var first = focusables[0];
    var last = focusables[focusables.length - 1];
    var active = document.activeElement;

    if (active === panel) {
      e.preventDefault();
      focusEl(e.shiftKey ? last : first);
      return;
    }

    if (e.shiftKey && active === first) {
      e.preventDefault();
      focusEl(last);
    } else if (!e.shiftKey && active === last) {
      e.preventDefault();
      focusEl(first);
    }
  });

  setSignedUp(false);
  if (isClosed) applyClosedState();
  if (normalState) applyNormalState(normalState);
  confirmNoticeBtn.addEventListener('click', confirmNotice);
})();
