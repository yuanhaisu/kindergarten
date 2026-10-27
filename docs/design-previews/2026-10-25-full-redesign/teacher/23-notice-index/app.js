/* 通知 · 老师端交互
 * 列表筛选、详情快速查看、未读名单展开、演示状态兜底。
 * 数据字段对齐 backend/api/api-teacher/04-notices.md。
 */
(function () {
  var app = document.querySelector('.mp-app');
  var toast = document.getElementById('toast');
  var toastTimer = null;

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-show');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-show');
    }, 1800);
  }

  var NOTICES = {
    '1': {
      category: 'homework', categoryLabel: '作业',
      status: '2', statusLabel: '已发布', statusClass: 'published',
      title: '周末作业通知',
      meta: '小一班、小二班 · 10月24日 09:00',
      content: '各位家长您好，本周作业如下：\n1. 复习本周学过的儿歌；\n2. 完成手工制作《我的家》，周一带回。\n\n请家长协助孩子合理安排周末时间，谢谢配合。',
      read: 25, unread: 5, confirmed: 20, rate: 83,
      unreadNames: [
        ['乐', '乐乐妈妈', '小一班'],
        ['雨', '小雨爸爸', '小一班'],
        ['朵', '朵朵妈妈', '小二班'],
        ['安', '安安爸爸', '小二班'],
        ['禾', '禾禾妈妈', '小一班']
      ],
      note: '已读与确认数据来自家长端实时回传。'
    },
    '2': {
      category: 'activity', categoryLabel: '活动',
      status: '2', statusLabel: '已发布', statusClass: 'published',
      title: '秋游集合时间与物品清单',
      meta: '小一班 · 10月23日 16:20',
      content: '周五秋游 8:30 在幼儿园门口集合，请给孩子准备水壶、遮阳帽和轻便午餐，穿运动鞋。\n\n如遇雨天，活动顺延，另行通知。',
      read: 28, unread: 2, confirmed: 24, rate: 93,
      unreadNames: [
        ['禾', '禾禾妈妈', '小一班'],
        ['安', '安安爸爸', '小一班']
      ],
      note: '已读与确认数据来自家长端实时回传。'
    },
    '3': {
      category: 'health', categoryLabel: '健康',
      status: '1', statusLabel: '定时待发', statusClass: 'scheduled',
      title: '下周一视力筛查安排',
      meta: '小一班 · 10月26日 08:30 发送',
      content: '下周一上午进行视力筛查，请家长让孩子前一晚早点休息，当天正常入园即可。\n\n筛查结果会在完成后单独同步给家长。',
      read: 0, unread: 0, confirmed: 0, rate: 0,
      unreadNames: [],
      note: '定时发送：10月26日 08:30 自动发送，发送后开始统计已读。'
    },
    '4': {
      category: 'bring', categoryLabel: '带物',
      status: '0', statusLabel: '草稿', statusClass: 'draft',
      title: '明天请带水彩笔和围裙',
      meta: '小一班 · 最后编辑 10月22日 20:10',
      content: '明天美术课需要用到水彩笔和围裙，请家长今晚帮孩子放进书包，谢谢配合。',
      read: 0, unread: 0, confirmed: 0, rate: 0,
      unreadNames: [],
      note: '草稿仅自己可见，发布后开始统计已读。'
    },
    '5': {
      category: 'other', categoryLabel: '其他',
      status: '3', statusLabel: '已撤回', statusClass: 'revoked',
      title: '临时调整放学地点',
      meta: '小一班 · 10月21日 15:00',
      content: '因园区施工，今天放学地点临时调整到东门，请家长按班级牌有序接孩子。',
      read: 18, unread: 12, confirmed: 11, rate: 60,
      unreadNames: [],
      note: '已撤回：家长端不再展示，以上为撤回前数据。'
    }
  };

  /* ---------- 列表筛选 ---------- */
  var statusButtons = Array.prototype.slice.call(document.querySelectorAll('.seg-btn[data-status]'));
  var categoryButtons = Array.prototype.slice.call(document.querySelectorAll('.cat-btn[data-category]'));
  var cards = Array.prototype.slice.call(document.querySelectorAll('.notice-card'));
  var listCount = document.getElementById('listCount');
  var clearFilters = document.getElementById('clearFilters');
  var filterEmpty = document.getElementById('filterEmpty');
  var noticeList = document.getElementById('noticeList');
  var activeStatus = 'all';
  var activeCategory = 'all';

  function setPressed(buttons, active) {
    buttons.forEach(function (button) {
      button.setAttribute('aria-pressed', String(button === active));
    });
  }

  function applyFilters() {
    var visible = 0;
    cards.forEach(function (card) {
      var statusOk = activeStatus === 'all' || card.dataset.status === activeStatus;
      var categoryOk = activeCategory === 'all' || card.dataset.category === activeCategory;
      var ok = statusOk && categoryOk;
      card.hidden = !ok;
      if (ok) visible += 1;
    });

    if (listCount) listCount.textContent = '共 ' + visible + ' 条通知';
    if (clearFilters) clearFilters.hidden = activeStatus === 'all' && activeCategory === 'all';
    if (noticeList) noticeList.hidden = visible === 0;
    if (filterEmpty) filterEmpty.hidden = visible !== 0;
  }

  function resetFilters() {
    activeStatus = 'all';
    activeCategory = 'all';
    setPressed(statusButtons, statusButtons[0]);
    setPressed(categoryButtons, categoryButtons[0]);
    applyFilters();
  }

  statusButtons.forEach(function (button) {
    button.addEventListener('click', function () {
      activeStatus = button.dataset.status;
      setPressed(statusButtons, button);
      applyFilters();
    });
  });
  categoryButtons.forEach(function (button) {
    button.addEventListener('click', function () {
      activeCategory = button.dataset.category;
      setPressed(categoryButtons, button);
      applyFilters();
    });
  });
  if (clearFilters) clearFilters.addEventListener('click', resetFilters);
  var emptyReset = document.getElementById('emptyReset');
  if (emptyReset) emptyReset.addEventListener('click', resetFilters);

  /* ---------- 详情弹层 ---------- */
  var sheet = document.getElementById('detailSheet');
  var detailCategory = document.getElementById('detailCategory');
  var detailStatus = document.getElementById('detailStatus');
  var detailTitle = document.getElementById('detailTitle');
  var detailMeta = document.getElementById('detailMeta');
  var detailContent = document.getElementById('detailContent');
  var detailStats = document.getElementById('detailStats');
  var detailProgressWrap = document.querySelector('.sheet-progress');
  var detailRead = document.getElementById('detailRead');
  var detailUnread = document.getElementById('detailUnread');
  var detailConfirmed = document.getElementById('detailConfirmed');
  var detailProgress = document.getElementById('detailProgress');
  var detailUnreadBtn = document.getElementById('detailUnreadBtn');
  var detailRemindBtn = document.getElementById('detailRemindBtn');
  var detailEditLink = document.getElementById('detailEditLink');
  var detailUnreadList = document.getElementById('detailUnreadList');
  var detailNote = document.getElementById('detailNote');
  var currentNotice = null;

  function openDetail(id) {
    var notice = NOTICES[id];
    if (!notice) return;
    currentNotice = notice;

    detailCategory.textContent = notice.categoryLabel;
    detailCategory.className = 'cat-chip cat-' + notice.category;
    detailStatus.textContent = notice.statusLabel;
    detailStatus.className = 'status status-' + notice.statusClass;
    detailTitle.textContent = notice.title;
    detailMeta.textContent = notice.meta;
    detailContent.textContent = notice.content;

    var hasStats = notice.status === '2' || notice.status === '3';
    detailStats.hidden = !hasStats;
    detailProgressWrap.hidden = !hasStats;
    if (hasStats) {
      detailRead.textContent = String(notice.read);
      detailUnread.textContent = String(notice.unread);
      detailConfirmed.textContent = String(notice.confirmed);
      detailProgress.style.width = notice.rate + '%';
    }

    var isPublished = notice.status === '2';
    detailUnreadBtn.hidden = !isPublished;
    detailRemindBtn.hidden = !isPublished;
    detailEditLink.hidden = notice.status !== '0' && notice.status !== '1';
    detailEditLink.textContent = notice.status === '1' ? '编辑定时' : '继续编辑';
    detailUnreadList.hidden = true;
    detailUnreadList.innerHTML = '';
    detailUnreadBtn.textContent = '查看未读家长';

    detailNote.textContent = notice.note;
    sheet.classList.add('is-open');
    sheet.setAttribute('aria-hidden', 'false');
  }

  function closeDetail() {
    sheet.classList.remove('is-open');
    sheet.setAttribute('aria-hidden', 'true');
    currentNotice = null;
  }

  cards.forEach(function (card) {
    card.addEventListener('click', function (event) {
      if (event.target.closest('button, a')) return;
      openDetail(card.dataset.id);
    });
    card.addEventListener('keydown', function (event) {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault();
        openDetail(card.dataset.id);
      }
    });
  });

  Array.prototype.forEach.call(document.querySelectorAll('[data-close-detail]'), function (el) {
    el.addEventListener('click', closeDetail);
  });
  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') closeDetail();
  });

  if (detailUnreadBtn) {
    detailUnreadBtn.addEventListener('click', function () {
      if (!currentNotice) return;
      var opening = detailUnreadList.hidden;
      detailUnreadList.hidden = !opening;
      detailUnreadBtn.textContent = opening ? '收起未读家长' : '查看未读家长';
      if (!opening) return;

      var title = document.createElement('p');
      title.className = 'unread-title';
      title.textContent = '未读家长（' + currentNotice.unread + '）';
      detailUnreadList.innerHTML = '';
      detailUnreadList.appendChild(title);

      currentNotice.unreadNames.forEach(function (row) {
        var item = document.createElement('div');
        item.className = 'unread-row';
        var avatar = document.createElement('span');
        avatar.className = 'unread-avatar';
        avatar.textContent = row[0];
        var name = document.createElement('span');
        name.textContent = row[1];
        var klass = document.createElement('span');
        klass.className = 'unread-class';
        klass.textContent = row[2];
        item.appendChild(avatar);
        item.appendChild(name);
        item.appendChild(klass);
        detailUnreadList.appendChild(item);
      });
    });
  }

  if (detailRemindBtn) {
    detailRemindBtn.addEventListener('click', function () {
      if (!currentNotice) return;
      showToast('已提醒 ' + currentNotice.unread + ' 位未读家长');
    });
  }

  /* ---------- 返回 / 重试 / 状态兜底 ---------- */
  var backBtn = document.getElementById('backBtn');
  if (backBtn) {
    backBtn.addEventListener('click', function () {
      if (window.history.length > 1 && document.referrer) window.history.back();
      else window.location.href = '../01-workbench/index.html';
    });
  }

  var retryBtn = document.getElementById('retryBtn');
  if (retryBtn) {
    retryBtn.addEventListener('click', function () {
      app.dataset.state = 'loading';
      window.setTimeout(function () {
        app.dataset.state = 'normal';
      }, 700);
    });
  }

  document.addEventListener('mp:state', function (event) {
    var state = event && event.detail ? event.detail.state : '';
    if (state !== 'normal') closeDetail();
  });

  applyFilters();
})();
