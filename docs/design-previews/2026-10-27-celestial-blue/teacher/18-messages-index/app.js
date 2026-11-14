(function () {
  'use strict';

  /*
   * 原型说明：当前源码的 mapMessage 会把 isRead 固定为 true、未读数为 0。
   * 设计稿要求展示未读与“全部已读”，这里先用 unread 字段还原视觉状态，
   * 待接口真正下发未读数据后替换。
   */

  var app = document.getElementById('messagesApp');
  var inboxList = document.getElementById('inboxList');
  var listSummary = document.getElementById('listSummary');
  var markAllBtn = document.getElementById('markAllBtn');
  var categoryEmpty = document.getElementById('categoryEmpty');
  var toast = document.getElementById('toast');
  var backBtn = document.getElementById('backBtn');
  var refreshEmptyBtn = document.getElementById('refreshEmptyBtn');
  var retryBtn = document.getElementById('retryBtn');
  var main = document.querySelector('.mp-main');
  var filterBtns = Array.prototype.slice.call(document.querySelectorAll('[data-filter]'));
  var demoChips = Array.prototype.slice.call(document.querySelectorAll('[data-mp-state]'));
  if (!app || !inboxList || !listSummary || !markAllBtn) return;

  var TYPE_LABEL = {
    notice: '通知',
    leave: '审批',
    system: '系统'
  };

  var messages = [
    {
      id: 'm1',
      type: 'leave',
      title: '乐乐妈妈提交的请假申请',
      content: '乐乐：感冒发烧，在家休息两天（10月27日 至 10月28日）',
      created_at: '2026-10-26T12:00:00+08:00',
      timeLabel: '刚刚',
      unread: true,
      extra: { status_label: '待审批', status_tone: 'pending' }
    },
    {
      id: 'm2',
      type: 'notice',
      title: '秋游亲子活动通知',
      content: '10月30日（周五）8:30 园门口集合，请为乐乐准备水壶和午餐',
      created_at: '2026-10-26T11:48:00+08:00',
      timeLabel: '12分钟前',
      unread: true,
      extra: {
        category_label: '活动',
        status_label: '已发布',
        read_count: 9,
        recipient_count: 28
      }
    },
    {
      id: 'm3',
      type: 'notice',
      title: '本周亲子任务已发布',
      content: '共 3 项：整理玩具、亲子阅读、跳绳练习，周日 20:00 前完成',
      created_at: '2026-10-26T11:00:00+08:00',
      timeLabel: '1小时前',
      unread: false,
      extra: {
        category_label: '作业',
        status_label: '已发布',
        read_count: 18,
        recipient_count: 28
      }
    },
    {
      id: 'm4',
      type: 'leave',
      title: '朵朵爸爸提交的请假申请',
      content: '朵朵：家中有事，需请假一天（10月26日 至 10月26日）',
      created_at: '2026-10-26T09:00:00+08:00',
      timeLabel: '3小时前',
      unread: true,
      extra: { status_label: '待审批', status_tone: 'pending' }
    },
    {
      id: 'm5',
      type: 'system',
      title: '系统维护通知',
      content: '10月27日 22:00–24:00 系统升级，期间消息推送可能延迟',
      created_at: '2026-10-25T18:20:00+08:00',
      timeLabel: '昨天 18:20',
      unread: false,
      extra: {}
    },
    {
      id: 'm6',
      type: 'notice',
      title: '传染病预防提醒',
      content: '近期流感高发，请提醒孩子勤洗手，发热幼儿暂缓返园',
      created_at: '2026-10-25T09:05:00+08:00',
      timeLabel: '昨天 09:05',
      unread: false,
      extra: {
        category_label: '健康',
        status_label: '已发布',
        read_count: 26,
        recipient_count: 28
      }
    },
    {
      id: 'm7',
      type: 'leave',
      title: '小雨妈妈提交的请假申请',
      content: '小雨：接种疫苗后低烧，需观察一天（10月20日 至 10月21日）',
      created_at: '2026-10-20T16:40:00+08:00',
      timeLabel: '10月20日',
      unread: false,
      extra: { status_label: '已同意', status_tone: 'approved' }
    },
    {
      id: 'm8',
      type: 'system',
      title: '账号安全提醒',
      content: '你的账号于 10月19日 在一台新设备上登录，如非本人操作请及时修改密码',
      created_at: '2026-10-19T08:12:00+08:00',
      timeLabel: '10月19日',
      unread: false,
      extra: {}
    },
    {
      id: 'm9',
      type: 'notice',
      title: '家长会通知',
      content: '10月18日（周六）9:30 在小二班教室召开家长会，请准时参加',
      created_at: '2026-10-17T17:30:00+08:00',
      timeLabel: '10月17日',
      unread: false,
      extra: {
        category_label: '活动',
        status_label: '已发布',
        read_count: 27,
        recipient_count: 28
      }
    },
    {
      id: 'm10',
      type: 'notice',
      title: '秋季运动会报名',
      content: '10月25日前在班级群接龙报名，项目：短跑、跳远、亲子接力',
      created_at: '2026-10-15T10:20:00+08:00',
      timeLabel: '10月15日',
      unread: false,
      extra: {
        category_label: '活动',
        status_label: '已发布',
        read_count: 21,
        recipient_count: 28
      }
    },
    {
      id: 'm11',
      type: 'system',
      title: '成长记录导出完成',
      content: '你导出的 9 月成长记录已生成，可在电脑端下载',
      created_at: '2026-10-14T14:00:00+08:00',
      timeLabel: '10月14日',
      unread: true,
      extra: {}
    },
    {
      id: 'm12',
      type: 'leave',
      title: '乐乐爸爸提交的请假申请',
      content: '乐乐：周一早高峰堵车，晚到半天（10月13日 至 10月13日）',
      created_at: '2026-10-12T07:50:00+08:00',
      timeLabel: '10月12日',
      unread: false,
      extra: { status_label: '已同意', status_tone: 'approved' }
    }
  ];

  var currentFilter = 'all';
  var toastTimer = 0;

  function escapeHtml(value) {
    var map = {
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;'
    };
    return String(value).replace(/[&<>"']/g, function (ch) {
      return map[ch];
    });
  }

  function clipText(value, max) {
    var text = String(value);
    return text.length > max ? text.slice(0, max) + '…' : text;
  }

  function messagesFor(filter) {
    if (filter === 'all') return messages;
    return messages.filter(function (item) {
      return item.type === filter;
    });
  }

  function countUnread(list) {
    return list.reduce(function (total, item) {
      return total + (item.unread ? 1 : 0);
    }, 0);
  }

  function iconTone(item) {
    if (item.type === 'notice') return 'type-icon--notice';
    if (item.type === 'system') return 'type-icon--system';
    if (item.extra.status_tone === 'pending') return 'type-icon--leave-pending';
    if (item.extra.status_tone === 'approved') return 'type-icon--leave-approved';
    return 'type-icon--leave-muted';
  }

  function iconName(item) {
    if (item.type === 'notice') return 'i-notice';
    if (item.type === 'leave') return 'i-leave';
    return 'i-system';
  }

  function metaMarkup(item) {
    var badge = '<span class="type-badge">' + TYPE_LABEL[item.type] + '</span>';
    var detail = '';
    if (item.type === 'notice') {
      detail =
        '<span class="meta-detail">' +
        escapeHtml(item.extra.category_label) +
        ' · ' +
        escapeHtml(item.extra.status_label) +
        ' · 已读 ' +
        escapeHtml(item.extra.read_count + '/' + item.extra.recipient_count) +
        '</span>';
    } else if (item.type === 'leave') {
      detail =
        '<span class="status-text status-text--' +
        escapeHtml(item.extra.status_tone) +
        '">' +
        escapeHtml(item.extra.status_label) +
        '</span>';
    }
    return '<span class="message-meta">' + badge + detail + '</span>';
  }

  function rowMarkup(item) {
    var dot = item.unread ? '<span class="unread-dot" aria-hidden="true"></span>' : '';
    return (
      '<div class="message-cell" role="listitem">' +
      '<button class="message-row" type="button" data-message-id="' +
      escapeHtml(item.id) +
      '" data-unread="' +
      String(item.unread) +
      '" aria-label="' +
      escapeHtml(item.title) +
      '" data-od-id="message-' +
      escapeHtml(item.id) +
      '">' +
      '<span class="message-icon ' +
      iconTone(item) +
      '" aria-hidden="true">' +
      '<svg class="ico" width="20" height="20"><use href="#' +
      iconName(item) +
      '"></use></svg>' +
      dot +
      '</span>' +
      '<span class="message-body">' +
      '<span class="message-head">' +
      '<span class="message-title">' +
      escapeHtml(item.title) +
      '</span>' +
      '<span class="message-time">' +
      escapeHtml(item.timeLabel) +
      '</span>' +
      '</span>' +
      '<span class="message-summary">' +
      escapeHtml(clipText(item.content, 60)) +
      '</span>' +
      metaMarkup(item) +
      '</span>' +
      '<svg class="ico message-chevron" width="12" height="12" aria-hidden="true"><use href="#i-chevron"></use></svg>' +
      '</button>' +
      '</div>'
    );
  }

  function renderList() {
    var visible = messagesFor(currentFilter);
    inboxList.innerHTML = visible.map(rowMarkup).join('');
    inboxList.setAttribute('aria-busy', 'false');
    if (categoryEmpty) categoryEmpty.hidden = visible.length > 0;
  }

  function syncFilterButtons() {
    filterBtns.forEach(function (btn) {
      var key = btn.getAttribute('data-filter');
      btn.setAttribute('aria-pressed', String(key === currentFilter));
      var countEl = btn.querySelector('.seg-count');
      if (countEl) countEl.textContent = String(messagesFor(key).length);
    });
  }

  function updateMeta() {
    var visible = messagesFor(currentFilter);
    var unread = countUnread(visible);
    listSummary.textContent = '共 ' + visible.length + ' 条 · ' + unread + ' 条未读';
    // 「全部已读」是全局动作（对齐源码 markAllRead），禁用与否看全量未读，
    // 不能只看当前分类——否则在已读完的分类下会误禁用。
    var unavailable = countUnread(messages) === 0;
    markAllBtn.disabled = unavailable;
    markAllBtn.setAttribute('aria-disabled', String(unavailable));
    syncFilterButtons();
  }

  function showToast(message) {
    if (!toast) return;
    toast.textContent = message;
    toast.classList.add('is-show');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-show');
    }, 1800);
  }

  function scrollToTop() {
    // 页面滚动的是窗口（mp-main 不是滚动容器），只清 main.scrollTop 不会生效。
    window.scrollTo(0, 0);
  }

  function syncDemoChips(state) {
    demoChips.forEach(function (chip) {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === state));
    });
  }

  function setState(state, closeDemo) {
    app.setAttribute('data-state', state);
    syncDemoChips(state);
    scrollToTop();
    if (closeDemo && window.OdShell && typeof window.OdShell.closeDemo === 'function') {
      window.OdShell.closeDemo();
    }
  }

  function setFilter(key) {
    currentFilter = key;
    renderList();
    updateMeta();
    scrollToTop();
  }

  filterBtns.forEach(function (btn) {
    btn.addEventListener('click', function () {
      setFilter(btn.getAttribute('data-filter'));
    });
  });

  inboxList.addEventListener('click', function (event) {
    var row = event.target.closest ? event.target.closest('.message-row') : null;
    if (!row) return;
    var id = row.getAttribute('data-message-id');
    var item = messages.filter(function (message) {
      return message.id === id;
    })[0];
    if (!item) return;

    if (item.unread) {
      item.unread = false;
      row.setAttribute('data-unread', 'false');
      var dot = row.querySelector('.unread-dot');
      if (dot) dot.remove();
      updateMeta();
    }
    showToast(item.type === 'leave' ? '已打开请假审批' : '查看：' + item.title);
  });

  markAllBtn.addEventListener('click', function () {
    // 守卫与禁用条件同口径：都看全量未读。若这里改看当前分类，
    // 在「本类已读完但别类还有未读」时分段上按钮可点却静默无反应。
    if (countUnread(messages) === 0) return;
    messages.forEach(function (item) {
      item.unread = false;
    });
    Array.prototype.forEach.call(inboxList.querySelectorAll('.message-row'), function (row) {
      row.setAttribute('data-unread', 'false');
      var dot = row.querySelector('.unread-dot');
      if (dot) dot.remove();
    });
    updateMeta();
    showToast('已全部标为已读');
  });

  if (backBtn) {
    backBtn.addEventListener('click', function () {
      showToast('返回上一页');
    });
  }

  if (refreshEmptyBtn) {
    refreshEmptyBtn.addEventListener('click', function () {
      setState('normal', false);
    });
  }

  if (retryBtn) {
    retryBtn.addEventListener('click', function () {
      setState('normal', false);
    });
  }

  demoChips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      setState(chip.getAttribute('data-mp-state'), true);
    });
  });

  renderList();
  updateMeta();
  syncDemoChips('normal');
})();
