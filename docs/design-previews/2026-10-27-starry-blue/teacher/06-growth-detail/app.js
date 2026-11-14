/* 成长记录详情（老师端）· 原型交互
 * 覆盖：照片预览弹层、AI 建议确认/拒绝、一键全班、授徽章面板、评论输入与发送、
 *      状态演示（由 _shared/shell.js 负责切换 data-state）、失败重试。
 * 只用原生 API，无外部依赖；所有提示走 toast，不用 alert。
 */
(() => {
  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  const app = $('.mp-app');
  const toastEl = $('#toast');
  let toastTimer = 0;

  function toast(text) {
    if (!toastEl) return;
    toastEl.textContent = text;
    toastEl.classList.add('is-show');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toastEl.classList.remove('is-show'), 1800);
  }

  /* ---------------- 照片预览（底部弹层模拟 uni.previewImage） ---------------- */
  const previewLayer = $('#previewLayer');
  const previewStage = $('#previewStage');
  const previewMeta = $('#previewMeta');
  const tiles = $$('.photo-open');

  // 张数按九宫格的照片位计（含加载失败的第 9 格），与「9 张 / 共 9 张」同一口径；
  // tiles 只含可预览的按钮，重试前会少算一张。
  function mediaTotal() {
    const grid = $('#mediaGrid');
    return grid ? $$('.photo-cell', grid).length : tiles.length;
  }

  function openPreview(tile) {
    if (!previewLayer || !previewStage) return;
    const index = Number(tile.dataset.preview || 0);
    const total = mediaTotal();
    // 复用同一张占位渐变，避免每张照片单独维护一份背景
    previewStage.className = 'preview-stage ph ' + (tile.classList[1] || 'ph--1');
    if (previewMeta) {
      const flag = tile.querySelector('.photo-flag');
      const suffix = flag ? ' · ' + flag.textContent.trim() : '';
      previewMeta.textContent = '第 ' + index + ' 张 / 共 ' + total + ' 张' + suffix;
    }
    previewLayer.hidden = false;
  }

  function closePreview() {
    if (previewLayer) previewLayer.hidden = true;
  }

  tiles.forEach((tile) => {
    tile.addEventListener('click', () => openPreview(tile));
  });

  const previewClose = $('#previewClose');
  const previewBackdrop = $('#previewBackdrop');
  if (previewClose) previewClose.addEventListener('click', closePreview);
  if (previewBackdrop) previewBackdrop.addEventListener('click', closePreview);

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && previewLayer && !previewLayer.hidden) closePreview();
  });

  /* ---------------- 单张照片加载失败 → 重试 ---------------- */
  const retryPhoto = $('#retryPhoto');
  if (retryPhoto) {
    retryPhoto.addEventListener('click', () => {
      toast('正在重新加载照片');
      const cell = retryPhoto.closest('.photo-cell');
      if (!cell) return;
      setTimeout(() => {
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'photo-open ph ph--9';
        btn.dataset.preview = '9';
        btn.setAttribute('aria-label', '预览第 9 张照片');
        btn.addEventListener('click', () => openPreview(btn));
        btn.style.background = 'linear-gradient(150deg, var(--accent-wash), var(--surface))';
        cell.replaceChildren(btn);
        tiles.push(btn);
        toast('照片已恢复');
      }, 700);
    });
  }

  /* ---------------- AI 识别建议 ---------------- */
  const aiList = $('#aiList');
  const aiDone = $('#aiDone');
  const aiBanner = $('#aiBanner');

  function syncAiState() {
    if (!aiList) return;
    const rows = $$('.ai-row', aiList);
    const highs = rows.filter((row) => row.querySelector('.status--ok'));
    if (aiDone) aiDone.hidden = rows.length > 0;
    if (aiBanner) aiBanner.hidden = highs.length === 0;
    const countNode = aiBanner ? aiBanner.querySelector('strong') : null;
    if (countNode) countNode.textContent = String(highs.length);
  }

  function handleSuggestion(row, action) {
    const name = row.querySelector('.ai-name');
    const label = name ? name.textContent.trim() : '这条建议';
    row.remove();
    syncAiState();
    toast(action === 'confirm' ? '已确认点名：' + label : '已忽略' + label);
  }

  if (aiList) {
    aiList.addEventListener('click', (event) => {
      const btn = event.target.closest('[data-ai-action]');
      if (!btn) return;
      const row = btn.closest('.ai-row');
      if (row) handleSuggestion(row, btn.dataset.aiAction);
    });
  }

  const confirmAll = $('#confirmAll');
  if (confirmAll) {
    confirmAll.addEventListener('click', () => {
      if (!aiList) return;
      const highs = $$('.ai-row', aiList).filter((row) => row.querySelector('.status--ok'));
      if (!highs.length) {
        toast('没有待处理的高置信建议');
        return;
      }
      highs.forEach((row) => row.remove());
      syncAiState();
      toast('已确认 ' + highs.length + ' 条高置信建议');
    });
  }

  /* ---------------- 大合影一键全班 ---------------- */
  const wholeClassBtn = $('#wholeClassBtn');
  if (wholeClassBtn) {
    wholeClassBtn.addEventListener('click', () => {
      const sub = wholeClassBtn.querySelector('.group-photo-btn__sub');
      if (sub) sub.textContent = '已点名 18 人（当日到场）';
      toast('已点名 18 人（当日到场）');
    });
  }

  /* ---------------- 授徽章 ---------------- */
  const awardTrigger = $('#awardTrigger');
  const awardPanel = $('#awardPanel');
  const awardClose = $('#awardClose');
  const awardHint = $('#awardHint');
  const awardSubmit = $('#awardSubmit');

  function setAwardOpen(open) {
    if (!awardPanel || !awardTrigger) return;
    awardPanel.hidden = !open;
    awardTrigger.setAttribute('aria-expanded', String(open));
    awardTrigger.querySelector('span').textContent = open ? '收起' : '授徽章';
    if (open) awardPanel.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }

  if (awardTrigger) awardTrigger.addEventListener('click', () => setAwardOpen(awardPanel.hidden));
  if (awardClose) awardClose.addEventListener('click', () => setAwardOpen(false));

  $$('[data-badge]').forEach((btn) => {
    btn.addEventListener('click', () => {
      $$('[data-badge]').forEach((other) => {
        const on = other === btn;
        other.classList.toggle('is-selected', on);
        other.setAttribute('aria-pressed', String(on));
      });
    });
  });

  const kidChoices = $('#kidChoices');
  if (kidChoices) {
    kidChoices.addEventListener('click', (event) => {
      const btn = event.target.closest('[data-kid]');
      if (!btn) return;
      const next = btn.getAttribute('aria-pressed') !== 'true';
      btn.setAttribute('aria-pressed', String(next));
      const picked = $$('[data-kid][aria-pressed="true"]', kidChoices).map((el) => el.dataset.kid);
      if (!awardHint) return;
      awardHint.textContent = picked.length
        ? '将授予 ' + picked.length + ' 位小朋友：' + picked.join('、')
        : '未选择孩子，将默认授予本班全部 18 人';
      awardHint.classList.toggle('is-default', picked.length === 0);
    });
  }

  const witnessCheck = $('#witnessCheck');
  if (witnessCheck) {
    witnessCheck.addEventListener('change', () => {
      toast(witnessCheck.checked ? '已关联本记录作为见证' : '已取消关联见证');
    });
  }

  if (awardSubmit) {
    awardSubmit.addEventListener('click', () => {
      const badge = $('[data-badge].is-selected');
      if (!badge) {
        toast('请先选一枚徽章');
        return;
      }
      const picked = $$('[data-kid][aria-pressed="true"]');
      const target = picked.length ? picked.length + ' 位小朋友' : '本班全部 18 人';
      setAwardOpen(false);
      toast('已授予「' + badge.dataset.badge + '」给' + target);
    });
  }

  /* ---------------- 记录操作 ---------------- */
  const editBtn = $('#editBtn');
  if (editBtn) editBtn.addEventListener('click', () => toast('打开编辑页'));

  const shareBtn = $('#shareBtn');
  if (shareBtn) shareBtn.addEventListener('click', () => toast('点击右上角分享'));

  const backBtn = $('#backBtn');
  if (backBtn) backBtn.addEventListener('click', () => toast('返回成长记录列表'));

  /* ---------------- 评论 ---------------- */
  const form = $('#commentForm');
  const input = $('#commentInput');
  const sendBtn = $('#sendBtn');
  const commentList = $('#commentList');
  const commentCount = $('#commentCount');
  const statCommentCount = $('#statCommentCount');
  const emptyWrite = $('#emptyWrite');

  function syncSend() {
    if (!sendBtn || !input) return;
    sendBtn.disabled = input.value.trim().length === 0;
  }

  // 空评论态代表「这条记录还没有任何评论」：演示评论暂时移出列表，计数随之归零；
  // 切回其它状态时原样放回。空态下发首条评论后演示评论不再回来（计数从 1 起算）。
  let stashedComments = [];

  function syncCommentEmpty(empty) {
    if (!commentList) return;
    if (empty) {
      if (stashedComments.length) return;
      stashedComments = $$('.comment', commentList);
      stashedComments.forEach((node) => node.remove());
    } else if (stashedComments.length) {
      stashedComments.forEach((node) => commentList.append(node));
      stashedComments = [];
    } else {
      return;
    }
    bumpCommentCount();
  }

  function setState(state) {
    if (!app) return;
    app.dataset.state = state;
    $$('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
    syncCommentEmpty(state === 'empty');
  }

  // 演示条由 _shared/shell.js 直接写 data-state 并广播 mp:state，这里跟着同步空态。
  document.addEventListener('mp:state', (event) => {
    const state = event.detail && event.detail.state;
    if (state) syncCommentEmpty(state === 'empty');
  });

  function bumpCommentCount() {
    const rows = commentList ? $$('.comment', commentList).length : 0;
    if (commentCount) commentCount.textContent = '（' + rows + '条）';
    if (statCommentCount) statCommentCount.textContent = String(rows);
  }

  function appendComment(text) {
    if (!commentList) return;
    const now = new Date();
    const hh = String(now.getHours()).padStart(2, '0');
    const mm = String(now.getMinutes()).padStart(2, '0');

    const li = document.createElement('li');
    li.className = 'comment';
    const avatar = document.createElement('span');
    avatar.className = 'avatar avatar--sm avatar--teacher';
    avatar.setAttribute('aria-hidden', 'true');
    avatar.textContent = '王';

    const main = document.createElement('div');
    main.className = 'comment-main';

    const top = document.createElement('div');
    top.className = 'comment-top';
    top.innerHTML =
      '<span class="comment-name">王老师</span>' +
      '<span class="role-tag role-tag--teacher">老师</span>' +
      '<time class="comment-time">' + hh + ':' + mm + '</time>';

    const body = document.createElement('p');
    body.className = 'comment-text';
    body.textContent = text;

    const actions = document.createElement('div');
    actions.className = 'comment-actions';
    actions.innerHTML =
      '<button class="link-btn like-btn" type="button" aria-pressed="false">' +
      '<span class="like-label">赞 0</span></button>' +
      '<button class="link-btn reply-btn" type="button">回复</button>';

    main.append(top, body, actions);
    li.append(avatar, main);
    commentList.append(li);
    bumpCommentCount();
  }

  if (input) {
    input.addEventListener('input', syncSend);
    input.addEventListener('keydown', (event) => {
      if (event.key === 'Enter') {
        event.preventDefault();
        if (form) form.requestSubmit();
      }
    });
  }

  if (form) {
    form.addEventListener('submit', (event) => {
      event.preventDefault();
      const text = input ? input.value.trim() : '';
      if (!text) {
        toast('请输入评论内容');
        return;
      }
      // 空评论态下发出第一条后回到正常态让新评论可见；空态本就代表没有评论，
      // 移出的演示评论不再放回，计数从 1 起算。
      if (app && app.dataset.state === 'empty') {
        stashedComments = [];
        setState('normal');
      }
      appendComment(text);
      if (input) input.value = '';
      syncSend();
      toast('评论已发布');
      window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' });
    });
  }

  if (emptyWrite) {
    emptyWrite.addEventListener('click', () => {
      if (!input) return;
      input.focus();
      input.scrollIntoView({ block: 'center', behavior: 'smooth' });
    });
  }

  /* 评论区的「赞 / 回复」：源码里两者都还是占位实现，这里只做本地反馈 */
  if (commentList) {
    commentList.addEventListener('click', (event) => {
      const like = event.target.closest('.like-btn');
      if (like) {
        const on = like.getAttribute('aria-pressed') !== 'true';
        like.setAttribute('aria-pressed', String(on));
        const label = like.querySelector('.like-label');
        if (label) label.textContent = '赞 ' + Number(like.dataset.count || 0) + (on ? 1 : 0);
        return;
      }
      const reply = event.target.closest('.reply-btn');
      if (reply) toast('回复功能开发中');
    });
  }

  /* ---------------- 加载失败重试 ---------------- */
  const retryLoad = $('#retryLoad');
  if (retryLoad) {
    retryLoad.addEventListener('click', () => {
      setState('loading');
      setTimeout(() => {
        setState('normal');
        toast('已重新加载');
      }, 900);
    });
  }

  /* 初始：骨架闪一下再进正常态，让审阅者看到加载设计 */
  if (app && !window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    setState('loading');
    setTimeout(() => setState('normal'), 900);
  }

  syncSend();
  syncAiState();
  bumpCommentCount();
})();
