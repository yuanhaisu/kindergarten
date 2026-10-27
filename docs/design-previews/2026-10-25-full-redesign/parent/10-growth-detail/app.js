(() => {
  const app = document.querySelector('.mp-app');
  if (!app) return;

  const $ = (selector) => document.querySelector(selector);
  const $$ = (selector) => Array.from(document.querySelectorAll(selector));

  const dom = {
    photoGrid: $('#photoGrid'),
    photoEmpty: $('#photoEmpty'),
    showAllPhotos: $('#showAllPhotos'),
    likeBtn: $('#likeBtn'),
    likeCount: $('#likeCount'),
    focusComment: $('#focusComment'),
    commentsList: $('#commentList'),
    writeFirst: $('#writeFirst'),
    composer: $('#composer'),
    form: $('#commentForm'),
    input: $('#commentInput'),
    send: $('#sendComment'),
    cancelReply: $('#cancelReply'),
    replyName: $('#replyName'),
    retryBtn: $('#retryBtn'),
    back: $('.mp-back'),
    sheet: $('#photoSheet'),
    sheetScrim: $('#sheetScrim'),
    sheetClose: $('#closePreview'),
    previewStage: $('#previewStage'),
    previewCount: $('#previewCount'),
    previewNav: $('#previewNav'),
    prevPhoto: $('#prevPhoto'),
    nextPhoto: $('#nextPhoto'),
    toast: $('#toast')
  };

  const photos = $$('.photo');
  const filters = $$('.filter-chip');
  const countEls = $$('[data-comment-count]');
  let liked = false;
  let likes = 12;
  let commentCount = 4;
  let localSeq = 0;
  let activeFilter = 'mine';
  let replyTarget = null;
  let previewTrigger = null;
  let previewList = [];
  let previewIndex = 0;
  let swipeStartX = null;
  let toastTimer = 0;
  let retryTimer = 0;

  const stateOf = () => app.dataset.state || 'normal';
  const isBlocked = () => stateOf() === 'loading' || stateOf() === 'error';

  function updateSendState() {
    if (!dom.input || !dom.send) return;
    const blocked = isBlocked();
    const empty = !dom.input.value.trim();
    dom.input.disabled = blocked;
    dom.send.disabled = blocked || empty;
    dom.send.setAttribute('aria-disabled', String(blocked || empty));
  }

  function syncInteractiveState() {
    const blocked = isBlocked();
    if (dom.likeBtn) dom.likeBtn.setAttribute('aria-disabled', String(blocked));
    filters.forEach((chip) => chip.setAttribute('aria-disabled', String(blocked)));
    photos.forEach((photo) => photo.setAttribute('aria-disabled', String(blocked)));
    if (dom.focusComment) dom.focusComment.setAttribute('aria-disabled', String(blocked));
    updateSendState();
  }

  function previewFocusables() {
    return [dom.sheetClose, dom.prevPhoto, dom.nextPhoto].filter((el) => el && !el.disabled && !el.closest('[hidden]'));
  }

  function focusPreview() {
    if (!dom.sheet || dom.sheet.dataset.open !== 'true') return;
    const items = previewFocusables();
    if (items.length) items[0].focus({ preventScroll: true });
  }

  function trapPreviewFocus(event) {
    const items = previewFocusables();
    if (!items.length) return;
    event.preventDefault();
    const current = items.indexOf(document.activeElement);
    const step = event.shiftKey ? -1 : 1;
    const next = current < 0 ? 0 : (current + step + items.length) % items.length;
    items[next].focus({ preventScroll: true });
  }

  function renderPreview() {
    const photo = previewList[previewIndex];
    if (!photo || !dom.previewStage) return;
    const art = photo.querySelector('.photo-art');
    if (!art) return;
    dom.previewStage.replaceChildren(art.cloneNode(true));
    if (dom.previewCount) dom.previewCount.textContent = `第 ${previewIndex + 1} 张 / 共 ${previewList.length} 张`;
    const single = previewList.length < 2;
    if (dom.previewNav) dom.previewNav.hidden = single;
    if (dom.prevPhoto) dom.prevPhoto.disabled = single || previewIndex === 0;
    if (dom.nextPhoto) dom.nextPhoto.disabled = single || previewIndex === previewList.length - 1;
  }

  function stepPreview(delta) {
    if (!dom.sheet || dom.sheet.dataset.open !== 'true') return;
    const next = previewIndex + delta;
    if (next < 0 || next >= previewList.length) return;
    previewIndex = next;
    renderPreview();
    const active = document.activeElement;
    // 焦点落在被禁用的翻页键上时，浏览器会把焦点丢回 body，这里收回弹层
    if (!active || active === document.body || active.disabled) focusPreview();
  }

  function openPreview(trigger, list) {
    if (!dom.sheet || !dom.previewStage || isBlocked()) return;
    const index = list.indexOf(trigger);
    if (index < 0) return;
    previewList = list.slice();
    previewIndex = index;
    previewTrigger = trigger;
    renderPreview();
    dom.sheet.dataset.open = 'true';
    dom.sheet.setAttribute('aria-hidden', 'false');
    document.body.classList.add('sheet-open');
    window.requestAnimationFrame(focusPreview);
  }

  function closePreview(restoreFocus) {
    if (!dom.sheet || dom.sheet.dataset.open !== 'true') return;
    dom.sheet.dataset.open = 'false';
    dom.sheet.setAttribute('aria-hidden', 'true');
    document.body.classList.remove('sheet-open');
    previewList = [];
    previewIndex = 0;
    const trigger = previewTrigger;
    previewTrigger = null;
    if (restoreFocus && trigger && !isBlocked()) {
      trigger.focus({ preventScroll: true });
    }
  }

  function setReplyTarget(comment) {
    if (!dom.composer || !dom.replyName || !dom.input) return;
    const author = comment.querySelector('.comment-author');
    const name = author ? author.textContent.trim() : '这条评论';
    replyTarget = { id: comment.dataset.commentId || '', el: comment, name };
    dom.composer.dataset.reply = 'true';
    dom.replyName.textContent = name;
    document.body.classList.add('reply-open');
    dom.input.setAttribute('aria-label', `回复 ${name}`);
    if (!isBlocked()) dom.input.focus();
  }

  function cancelReply(focusInput) {
    replyTarget = null;
    if (dom.composer) dom.composer.dataset.reply = 'false';
    if (dom.replyName) dom.replyName.textContent = '';
    if (dom.input) dom.input.setAttribute('aria-label', '写评论');
    document.body.classList.remove('reply-open');
    if (focusInput && !isBlocked() && dom.input) dom.input.focus();
  }

  function updateCommentCounts() {
    countEls.forEach((el) => { el.textContent = String(commentCount); });
  }

  function makeComment(options) {
    const article = document.createElement('article');
    article.className = options.isReply ? 'comment reply-comment' : 'comment';
    article.dataset.commentId = options.id;

    const head = document.createElement('div');
    head.className = 'comment-head';

    const author = document.createElement('span');
    author.className = 'comment-author';
    author.textContent = options.name;
    head.appendChild(author);

    if (options.relation) {
      const relation = document.createElement('span');
      relation.className = 'relation-tag';
      relation.textContent = options.relation;
      head.appendChild(relation);
    }

    const time = document.createElement('span');
    time.className = 'comment-time';
    time.textContent = options.time;
    head.appendChild(time);

    const text = document.createElement('p');
    text.className = 'comment-text';
    text.textContent = options.text;

    const foot = document.createElement('div');
    foot.className = 'comment-foot';
    const reply = document.createElement('button');
    reply.type = 'button';
    reply.className = 'reply-btn';
    reply.textContent = '回复';
    foot.appendChild(reply);

    article.append(head, text, foot);
    return article;
  }

  function addTopLevelComment(text) {
    if (!dom.commentsList) return;
    const comment = makeComment({
      id: `local-${++localSeq}`,
      name: '乐乐妈妈',
      relation: '妈妈',
      time: '刚刚',
      text,
      isReply: false
    });
    dom.commentsList.prepend(comment);
    commentCount += 1;
    updateCommentCounts();
  }

  function addReply(text) {
    if (!replyTarget || !replyTarget.el) return;
    const target = replyTarget;
    let thread = null;
    const parent = target.el.parentElement;
    if (parent && parent.classList.contains('reply-thread')) {
      thread = parent;
    } else {
      thread = Array.from(target.el.children).find((child) => child.classList.contains('reply-thread'));
      if (!thread) {
        thread = document.createElement('div');
        thread.className = 'reply-thread';
        target.el.appendChild(thread);
      }
    }
    const reply = makeComment({
      id: `local-${++localSeq}`,
      name: '乐乐妈妈',
      relation: '妈妈',
      time: '刚刚',
      text,
      isReply: true
    });
    thread.appendChild(reply);
  }

  function applyFilter(mode) {
    if (isBlocked()) return;
    activeFilter = mode === 'all' ? 'all' : 'mine';
    filters.forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.filter === activeFilter));
    });
    photos.forEach((photo) => {
      photo.hidden = !(activeFilter === 'all' || photo.dataset.own === 'true');
    });
    if (dom.photoEmpty) {
      dom.photoEmpty.hidden = photos.some((photo) => !photo.hidden);
    }
  }

  function showToast(message) {
    if (!dom.toast) return;
    window.clearTimeout(toastTimer);
    dom.toast.textContent = message;
    dom.toast.classList.add('is-visible');
    toastTimer = window.setTimeout(() => {
      dom.toast.classList.remove('is-visible');
    }, 1800);
  }

  function handleStateChange() {
    if (isBlocked()) closePreview(false);
    if (isBlocked() || stateOf() === 'empty') cancelReply(false);
    syncInteractiveState();
  }

  function selectState(state) {
    const chip = document.querySelector(`.mp-demo-chip[data-mp-state="${state}"]`);
    if (chip) {
      chip.click();
    } else {
      app.dataset.state = state;
      handleStateChange();
    }
  }

  if (dom.likeBtn) {
    dom.likeBtn.addEventListener('click', () => {
      if (isBlocked()) return;
      liked = !liked;
      likes += liked ? 1 : -1;
      dom.likeBtn.classList.toggle('is-liked', liked);
      dom.likeBtn.setAttribute('aria-pressed', String(liked));
      dom.likeBtn.setAttribute('aria-label', `${liked ? '取消点赞' : '点赞'}，${likes} 个赞`);
      if (dom.likeCount) dom.likeCount.textContent = String(likes);
    });
  }

  filters.forEach((chip) => {
    chip.addEventListener('click', () => {
      if (isBlocked()) return;
      applyFilter(chip.dataset.filter);
    });
  });

  if (dom.showAllPhotos) {
    dom.showAllPhotos.addEventListener('click', () => applyFilter('all'));
  }

  if (dom.photoGrid) {
    dom.photoGrid.addEventListener('click', (event) => {
      const photo = event.target.closest('.photo');
      if (!photo || photo.hidden || isBlocked()) return;
      if (photo.getAttribute('aria-disabled') === 'true') return;
      const visible = photos.filter((item) => !item.hidden);
      openPreview(photo, visible);
    });
  }

  if (dom.commentsList) {
    dom.commentsList.addEventListener('click', (event) => {
      const button = event.target.closest('.reply-btn');
      if (!button || isBlocked()) return;
      const comment = button.closest('.comment');
      if (!comment) return;
      setReplyTarget(comment);
    });
  }

  if (dom.cancelReply) {
    dom.cancelReply.addEventListener('click', () => cancelReply(true));
  }

  if (dom.focusComment) {
    dom.focusComment.addEventListener('click', () => {
      if (isBlocked()) return;
      dom.input.focus();
    });
  }

  if (dom.writeFirst) {
    dom.writeFirst.addEventListener('click', () => {
      if (isBlocked()) return;
      dom.input.focus();
    });
  }

  if (dom.input) {
    dom.input.addEventListener('input', updateSendState);
  }

  if (dom.form) {
    dom.form.addEventListener('submit', (event) => {
      event.preventDefault();
      if (isBlocked()) return;
      const text = dom.input.value.trim();
      if (!text) {
        updateSendState();
        return;
      }
      const target = replyTarget;
      if (stateOf() === 'empty') {
        if (dom.commentsList) dom.commentsList.replaceChildren();
        commentCount = 0;
        updateCommentCounts();
        selectState('normal');
      }
      if (target) addReply(text);
      else addTopLevelComment(text);
      dom.input.value = '';
      cancelReply(false);
      updateSendState();
      showToast('评论成功');
    });
  }

  if (dom.retryBtn) {
    dom.retryBtn.addEventListener('click', () => {
      if (stateOf() !== 'error') return;
      window.clearTimeout(retryTimer);
      selectState('loading');
      retryTimer = window.setTimeout(() => {
        if (stateOf() === 'loading') selectState('normal');
      }, 900);
    });
  }

  if (dom.sheetClose) {
    dom.sheetClose.addEventListener('click', () => closePreview(true));
  }
  if (dom.sheetScrim) {
    dom.sheetScrim.addEventListener('click', () => closePreview(true));
  }
  if (dom.prevPhoto) {
    dom.prevPhoto.addEventListener('click', () => stepPreview(-1));
  }
  if (dom.nextPhoto) {
    dom.nextPhoto.addEventListener('click', () => stepPreview(1));
  }
  if (dom.previewStage) {
    dom.previewStage.addEventListener('touchstart', (event) => {
      swipeStartX = event.touches[0] ? event.touches[0].clientX : null;
    }, { passive: true });
    dom.previewStage.addEventListener('touchend', (event) => {
      if (swipeStartX === null) return;
      const touch = event.changedTouches[0];
      const dx = touch ? touch.clientX - swipeStartX : 0;
      swipeStartX = null;
      if (Math.abs(dx) < 36) return;
      stepPreview(dx < 0 ? 1 : -1);
    }, { passive: true });
  }
  if (dom.back) {
    dom.back.addEventListener('click', () => showToast('返回成长时光'));
  }
  document.addEventListener('keydown', (event) => {
    if (!dom.sheet || dom.sheet.dataset.open !== 'true') return;
    if (event.key === 'Escape') {
      event.preventDefault();
      closePreview(true);
    } else if (event.key === 'ArrowLeft') {
      event.preventDefault();
      stepPreview(-1);
    } else if (event.key === 'ArrowRight') {
      event.preventDefault();
      stepPreview(1);
    } else if (event.key === 'Tab') {
      trapPreviewFocus(event);
    }
  });

  document.addEventListener('mp:state', (event) => {
    const state = event.detail && event.detail.state;
    if (state !== 'loading') {
      window.clearTimeout(retryTimer);
      retryTimer = 0;
    }
    handleStateChange();
  });

  updateCommentCounts();
  applyFilter('mine');
  syncInteractiveState();
})();
