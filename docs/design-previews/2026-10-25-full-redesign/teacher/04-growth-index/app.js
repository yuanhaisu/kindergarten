(function () {
  'use strict';

  var TODAY = '2026-10-24';

  function previews(n) {
    return Array.from({ length: n }, function () {
      return { thumb_url: '', type: 'image' };
    });
  }

  var RECORDS = [
    {
      id: 1024,
      date: '2026-10-24',
      content: '今天的美术课小朋友们都太棒了！乐乐画的太阳是橙色的，还说要把画带回家给妈妈看。朵朵第一次自己调出了绿色，特别开心。',
      performance_tags: [],
      media_preview: previews(3),
      likes_count: 12,
      comments_count: 4,
      related_students_count: 28,
      pending_suggestions_count: 2,
      sync_to_feed: true,
      target_type: 1,
      creator: { nickname: '王老师' },
      created_at: '2026-10-24 11:20',
      classKey: 'small1'
    },
    {
      id: 1023,
      date: '2026-10-24',
      content: '今天小雨主动把积木分给了新来的小朋友，午睡也比上周快了很多。',
      performance_tags: ['主动分享', '午睡进步'],
      media_preview: previews(1),
      likes_count: 3,
      comments_count: 1,
      related_students_count: 0,
      pending_suggestions_count: 0,
      sync_to_feed: true,
      target_type: 2,
      creator: { nickname: '王老师' },
      created_at: '2026-10-24 09:05',
      classKey: 'small1'
    },
    {
      id: 1022,
      date: '2026-10-23',
      content: '户外活动时间我们玩了老鹰捉小鸡，孩子们跑得满头大汗，回教室都喝了一大杯水。',
      performance_tags: [],
      media_preview: [],
      likes_count: 20,
      comments_count: 6,
      related_students_count: 28,
      pending_suggestions_count: 0,
      sync_to_feed: true,
      target_type: 1,
      creator: { nickname: '李老师' },
      created_at: '2026-10-23 16:40',
      classKey: 'small1'
    },
    {
      id: 1021,
      date: '2026-10-21',
      content: '秋游照片来啦，孩子们在草地上捡了好多落叶，回来做成了一棵大树。',
      performance_tags: [],
      media_preview: previews(4),
      likes_count: 31,
      comments_count: 9,
      related_students_count: 28,
      pending_suggestions_count: 1,
      sync_to_feed: true,
      target_type: 1,
      creator: { nickname: '王老师' },
      created_at: '2026-10-21 15:10',
      classKey: 'small1'
    },
    {
      id: 930,
      date: '2026-09-30',
      content: '朵朵今天自己吃完了一整碗饭，还举手说要添一点。',
      performance_tags: ['自己吃饭'],
      media_preview: previews(2),
      likes_count: 0,
      comments_count: 0,
      related_students_count: 0,
      pending_suggestions_count: 0,
      sync_to_feed: false,
      target_type: 2,
      creator: { nickname: '王老师' },
      created_at: '2026-09-30 12:05',
      classKey: 'small1'
    },
    {
      id: 926,
      date: '2026-09-26',
      content: '九月集体生日会，我们一起唱了生日歌，吃了小蛋糕。',
      performance_tags: [],
      media_preview: previews(9),
      likes_count: 26,
      comments_count: 11,
      related_students_count: 28,
      pending_suggestions_count: 0,
      sync_to_feed: true,
      target_type: 1,
      creator: { nickname: '王老师' },
      created_at: '2026-09-26 10:35',
      classKey: 'small1'
    },
    {
      id: 922,
      date: '2026-09-22',
      content: '开学第一个月，孩子们已经能自己排队洗手了。',
      performance_tags: [],
      media_preview: [],
      likes_count: 15,
      comments_count: 2,
      related_students_count: 28,
      pending_suggestions_count: 0,
      sync_to_feed: true,
      target_type: 1,
      creator: { nickname: '李老师' },
      created_at: '2026-09-22 09:15',
      classKey: 'small1'
    }
  ];

  var CLASS_OPTIONS = [
    { value: 'all', label: '全部班级' },
    { value: 'small1', label: '小一班' },
    { value: 'mid2', label: '中二班' }
  ];

  var TIME_OPTIONS = [
    { value: 'all', label: '全部时间', chip: '全部时间' },
    { value: '2026-10', label: '本月（10月）', chip: '10月' },
    { value: '2026-09', label: '上月（9月）', chip: '上月（9月）' }
  ];

  var app = document.getElementById('app');
  var pageScroll = document.getElementById('pageScroll');
  var listRegion = document.getElementById('listRegion');
  var listFoot = document.getElementById('listFoot');
  var statToday = document.getElementById('statToday');
  var statPending = document.getElementById('statPending');
  var statScope = document.getElementById('statScope');
  var publishBtn = document.getElementById('publishBtn');
  var classChip = document.getElementById('classChip');
  var classChipText = document.getElementById('classChipText');
  var timeChip = document.getElementById('timeChip');
  var timeChipText = document.getElementById('timeChipText');
  var searchShell = document.getElementById('searchShell');
  var searchInput = document.getElementById('searchInput');
  var searchClear = document.getElementById('searchClear');
  var sheetLayer = document.getElementById('sheetLayer');
  var sheetScrim = document.getElementById('sheetScrim');
  var sheetClose = document.getElementById('sheetClose');
  var sheetTitle = document.getElementById('sheetTitle');
  var sheetOptions = document.getElementById('sheetOptions');
  var toast = document.getElementById('toast');
  var emptyText = document.getElementById('emptyText');
  var emptyHint = document.getElementById('emptyHint');
  var emptyAction = document.getElementById('emptyAction');
  var stateEmpty = document.getElementById('stateEmpty');
  var retryBtn = document.getElementById('retryBtn');

  var state = {
    filters: { cls: 'all', time: 'all' },
    query: ''
  };
  var currentSheet = null;
  var lastFocused = null;
  var sheetTimer = null;
  var toastTimer = null;

  var ICON_CHECK = '<svg class="option-check" width="18" height="18" viewBox="0 0 18 18" fill="none" aria-hidden="true"><path d="m3.5 9.3 3.4 3.4 7.6-8" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"></path></svg>';
  var ICON_HEART = '<svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true"><path d="M7 12.1S2 9.1 2 5.7A2.8 2.8 0 0 1 7 3.8a2.8 2.8 0 0 1 5 1.9c0 3.4-5 6.4-5 6.4Z" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"></path></svg>';
  var ICON_COMMENT = '<svg width="14" height="14" viewBox="0 0 14 14" fill="none" aria-hidden="true"><path d="M2 6.2A4 4 0 0 1 6 2.2h2a4 4 0 0 1 4 4v.8a4 4 0 0 1-4 4H6.9l-2.8 1.7v-2A4 4 0 0 1 2 7V6.2Z" stroke="currentColor" stroke-width="1.3" stroke-linejoin="round"></path></svg>';

  function escapeHtml(value) {
    return String(value).replace(/[&<>"']/g, function (ch) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch];
    });
  }

  function computeStats() {
    var today = RECORDS.filter(function (r) { return r.date === TODAY; }).length;
    var pending = RECORDS.filter(function (r) {
      return r.pending_suggestions_count > 0 || (r.target_type === 2 && r.sync_to_feed === false);
    }).length;
    return { today: today, pending: pending };
  }

  function paintStats(today, pending) {
    statToday.textContent = String(today);
    statPending.textContent = String(pending);
  }

  function paintStatsForNormal() {
    var stats = computeStats();
    paintStats(stats.today, stats.pending);
  }

  // 统计口径固定为已加载的一页（与源码 todayCount/pendingCount 一致），不受筛选/搜索影响；
  // 筛选生效时显式标注，避免「未找到匹配的记录」与「今日已记 2」看起来互相矛盾。
  function paintStatScope(scoped) {
    statScope.textContent = scoped ? '仅统计已加载 · 不受筛选影响' : '仅统计已加载';
  }

  function setControlsDisabled(disabled) {
    publishBtn.disabled = disabled;
    publishBtn.setAttribute('aria-disabled', String(disabled));
    searchInput.disabled = disabled;
    searchShell.classList.toggle('is-disabled', disabled);
    searchShell.setAttribute('aria-disabled', String(disabled));
    [classChip, timeChip].forEach(function (chip) {
      chip.disabled = disabled;
      chip.setAttribute('aria-disabled', String(disabled));
    });
    if (disabled) {
      searchClear.hidden = true;
    }
  }

  function updateChips() {
    var cls = CLASS_OPTIONS.filter(function (o) { return o.value === state.filters.cls; })[0] || CLASS_OPTIONS[0];
    var time = TIME_OPTIONS.filter(function (o) { return o.value === state.filters.time; })[0] || TIME_OPTIONS[0];
    classChipText.textContent = cls.label;
    timeChipText.textContent = time.chip;
    classChip.classList.toggle('is-active', state.filters.cls !== 'all');
    timeChip.classList.toggle('is-active', state.filters.time !== 'all');
    classChip.setAttribute('aria-label', '班级筛选：' + cls.label);
    timeChip.setAttribute('aria-label', '时间筛选：' + time.label);
  }

  function groupRecords(rows) {
    var groups = [];
    rows.forEach(function (record) {
      var key = record.date.slice(0, 7);
      var group = groups.filter(function (g) { return g.key === key; })[0];
      if (!group) {
        group = { key: key, records: [] };
        groups.push(group);
      }
      group.records.push(record);
    });
    return groups;
  }

  function renderMedia(record) {
    var items = record.media_preview || [];
    var count = items.length;
    if (!count) return '';

    var layout = 'media-one';
    var tiles = '';
    var i;

    if (count === 1) {
      tiles = '<div class="media-tile" aria-hidden="true"></div>';
    } else if (count <= 3) {
      layout = 'media-row cols-' + count;
      for (i = 0; i < count; i += 1) {
        tiles += '<div class="media-tile" aria-hidden="true"></div>';
      }
    } else {
      layout = 'media-grid';
      for (i = 0; i < 4; i += 1) {
        var hidden = count - 4;
        var more = i === 3 && hidden > 0
          ? '<span class="media-more">+' + hidden + '</span>'
          : '';
        tiles += '<div class="media-tile" aria-hidden="true">' + more + '</div>';
      }
    }

    return '<div class="card-media ' + layout + '" role="img" aria-label="附件预览，共 ' + count + ' 张">' +
      tiles +
      '<span class="media-count">' + count + ' 张</span>' +
      '</div>';
  }

  function renderRecord(record) {
    var name = escapeHtml(record.creator.nickname);
    var initial = escapeHtml(Array.from(record.creator.nickname)[0] || '师');
    var media = renderMedia(record);
    var tags = '';

    if (record.target_type === 2 && record.performance_tags.length) {
      tags = '<p class="tag-line"><span>' +
        record.performance_tags.map(escapeHtml).join(' · ') +
        '</span></p>';
    }

    var left = '';
    if (record.target_type === 1) {
      left += '<span class="related-text">相关 ' + record.related_students_count + ' 位小朋友</span>';
    } else if (record.sync_to_feed) {
      left += '<span class="badge badge-success"><span class="badge-dot" aria-hidden="true"></span>已同步给家长</span>';
    } else {
      left += '<span class="badge badge-warning"><span class="badge-dot" aria-hidden="true"></span>未同步给家长</span>';
    }

    if (record.pending_suggestions_count > 0) {
      left += '<span class="badge badge-info" aria-label="AI 建议 ' + record.pending_suggestions_count + ' 条待确认">' +
        '<span class="badge-dot" aria-hidden="true"></span>AI建议 <strong>' +
        record.pending_suggestions_count + '</strong> 待确认</span>';
    }

    return '<article class="record-card" role="button" tabindex="0" data-id="' + record.id + '"' +
      ' data-od-id="growth-record-' + record.id + '"' +
      ' aria-label="查看 ' + name + ' ' + record.date + ' 的成长记录">' +
      '<div class="card-inner">' +
        media +
        '<div class="author-row">' +
          '<span class="avatar" aria-hidden="true">' + initial + '</span>' +
          '<div class="author-main"><span class="author-name">' + name + '</span></div>' +
          '<time class="record-date" datetime="' + record.date + '">' + record.date.slice(5) + '</time>' +
        '</div>' +
        tags +
        '<p class="card-content">' + escapeHtml(record.content) + '</p>' +
        '<div class="card-foot">' +
          '<div class="foot-left">' + left + '</div>' +
          '<div class="foot-counts">' +
            '<span class="count-item" aria-label="' + record.likes_count + ' 个赞">' + ICON_HEART +
              '<span aria-hidden="true">' + record.likes_count + '</span></span>' +
            '<span class="count-item" aria-label="' + record.comments_count + ' 条评论">' + ICON_COMMENT +
              '<span aria-hidden="true">' + record.comments_count + '</span></span>' +
          '</div>' +
        '</div>' +
      '</div>' +
    '</article>';
  }

  function filteredRows() {
    var rows = RECORDS.filter(function (record) {
      var classMatch = state.filters.cls === 'all' || record.classKey === state.filters.cls;
      var timeMatch = state.filters.time === 'all' || record.date.slice(0, 7) === state.filters.time;
      return classMatch && timeMatch;
    });

    var query = state.query.trim().toLowerCase();
    if (query) {
      rows = rows.filter(function (record) {
        return record.content.toLowerCase().indexOf(query) !== -1 ||
          record.creator.nickname.toLowerCase().indexOf(query) !== -1;
      });
    }

    return rows.sort(function (a, b) {
      return b.date.localeCompare(a.date) || b.created_at.localeCompare(a.created_at);
    });
  }

  function renderListHtml(rows) {
    return groupRecords(rows).map(function (group) {
      var parts = group.key.split('-');
      var monthLabel = parts[0] + '年' + Number(parts[1]) + '月';
      return '<section class="month-group" aria-label="' + monthLabel + '">' +
        '<div class="month-head">' +
          '<h3 class="month-title">' + monthLabel + '</h3>' +
          '<span class="month-count">（' + group.records.length + '条记录）</span>' +
        '</div>' +
        '<div class="card-stack">' + group.records.map(renderRecord).join('') + '</div>' +
      '</section>';
    }).join('');
  }

  // 空态一屏三用：服务端无记录（plain）/ 筛选无结果（filter）/ 搜索无结果（search）。
  // 可见性由 CSS 按 .mp-app[data-state] 决定，这里只换文案与动作。
  function setEmptyContent(kind) {
    var isFilter = kind === 'filter';
    var isSearch = kind === 'search';
    emptyText.textContent = kind ? '未找到匹配的记录' : '还没有成长记录，从今天开始吧';
    emptyHint.hidden = !kind;
    emptyHint.textContent = isFilter
      ? '当前筛选范围内没有记录，换一个班级或时间试试。'
      : (isSearch ? '换一个关键词，或清除搜索查看全部记录。' : '');
    emptyAction.textContent = kind ? (isFilter ? '清除筛选' : '清除搜索') : '发布第一条';
    emptyAction.dataset.action = kind ? (isFilter ? 'clear-filters' : 'clear-search') : 'publish-first';
    emptyAction.className = kind ? 'btn-ghost' : 'btn-gradient';
  }

  function renderNormal() {
    setControlsDisabled(false);
    searchClear.hidden = state.query === '';
    paintStatsForNormal();
    updateChips();

    var rows = filteredRows();
    var filtersActive = state.filters.cls !== 'all' || state.filters.time !== 'all';
    var hasQuery = state.query.trim() !== '';
    var scoped = filtersActive || hasQuery;

    paintStatScope(scoped);
    // 页脚如实反映「加载了多少」与「当前显示多少」；无筛选时才提下拉刷新
    listFoot.textContent = scoped
      ? '已加载 ' + RECORDS.length + ' 条 · 当前显示 ' + rows.length + ' 条'
      : '已显示最近 ' + RECORDS.length + ' 条 · 下拉刷新';

    if (!rows.length) {
      listRegion.innerHTML = '';
      setEmptyContent(hasQuery ? 'search' : (filtersActive ? 'filter' : 'plain'));
      // 服务端空（empty）与筛选/搜索无结果（noresult）分开：后者仍保留发布入口
      app.dataset.state = scoped ? 'noresult' : 'empty';
      return;
    }

    setEmptyContent('');
    app.dataset.state = 'normal';
    listRegion.innerHTML = renderListHtml(rows);
  }

  // 演示芯片切换时同步统计与控件状态；页面自身渲染不走这里，避免覆盖真实数据。
  function syncState(name) {
    closeSheetNow();
    if (name === 'empty') {
      setControlsDisabled(false);
      paintStats(0, 0);
      paintStatScope(false);
      setEmptyContent('');
    } else if (name === 'loading' || name === 'error') {
      setControlsDisabled(true);
      paintStats('—', '—');
    } else {
      // 「正常」= 干净的全量列表，不留上一轮的筛选/搜索残局
      clearSearchState();
      resetFilters();
      renderNormal();
    }
    resetScroll();
  }

  function resetScroll() {
    window.scrollTo(0, 0);
    pageScroll.scrollTop = 0;
  }

  function resetFilters() {
    state.filters = { cls: 'all', time: 'all' };
  }

  function clearSearchState() {
    state.query = '';
    searchInput.value = '';
    searchClear.hidden = true;
  }

  function updateSheetAria() {
    var classOpen = currentSheet === 'class';
    var timeOpen = currentSheet === 'time';
    classChip.setAttribute('aria-expanded', String(classOpen));
    timeChip.setAttribute('aria-expanded', String(timeOpen));
  }

  function buildSheet(kind) {
    var options = kind === 'class' ? CLASS_OPTIONS : TIME_OPTIONS;
    var selectedValue = kind === 'class' ? state.filters.cls : state.filters.time;
    sheetTitle.textContent = kind === 'class' ? '选择班级' : '选择时间';
    sheetOptions.innerHTML = options.map(function (option) {
      var selected = option.value === selectedValue;
      return '<button class="sheet-option' + (selected ? ' is-selected' : '') + '"' +
        ' type="button" data-value="' + option.value + '"' +
        ' aria-pressed="' + String(selected) + '">' +
        '<span>' + option.label + '</span>' +
        ICON_CHECK +
      '</button>';
    }).join('');
  }

  function openSheet(kind) {
    // 无结果落到空态时仍要能改筛选，所以只看控件是否禁用，不看 view
    if (classChip.disabled || timeChip.disabled) return;
    clearTimeout(sheetTimer);
    lastFocused = document.activeElement;
    currentSheet = kind;
    buildSheet(kind);
    sheetLayer.hidden = false;
    updateSheetAria();
    requestAnimationFrame(function () {
      sheetLayer.classList.add('is-open');
      app.classList.add('sheet-open');
      sheetClose.focus({ preventScroll: true });
    });
  }

  function closeSheet() {
    if (!currentSheet || sheetLayer.hidden) return;
    sheetLayer.classList.remove('is-open');
    app.classList.remove('sheet-open');
    currentSheet = null;
    updateSheetAria();
    clearTimeout(sheetTimer);
    sheetTimer = setTimeout(function () {
      sheetLayer.hidden = true;
      if (lastFocused && document.contains(lastFocused)) {
        lastFocused.focus({ preventScroll: true });
      }
    }, 220);
  }

  function closeSheetNow() {
    clearTimeout(sheetTimer);
    sheetLayer.classList.remove('is-open');
    sheetLayer.hidden = true;
    currentSheet = null;
    app.classList.remove('sheet-open');
    updateSheetAria();
  }

  function showToast(message) {
    clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = setTimeout(function () {
      toast.classList.remove('is-visible');
    }, 1800);
  }

  function openRecord(card) {
    card.classList.add('is-pressed');
    setTimeout(function () {
      card.classList.remove('is-pressed');
    }, 220);
    showToast('已打开记录详情');
  }

  publishBtn.addEventListener('click', function () {
    if (publishBtn.disabled) return;
    showToast('已进入发布页');
  });

  classChip.addEventListener('click', function () { openSheet('class'); });
  timeChip.addEventListener('click', function () { openSheet('time'); });
  sheetScrim.addEventListener('click', closeSheet);
  sheetClose.addEventListener('click', closeSheet);

  sheetOptions.addEventListener('click', function (event) {
    var button = event.target.closest('.sheet-option');
    if (!button) return;
    var value = button.dataset.value;
    if (currentSheet === 'class') {
      state.filters.cls = value;
    } else if (currentSheet === 'time') {
      state.filters.time = value;
    }
    closeSheet();
    renderNormal();
  });

  searchInput.addEventListener('input', function () {
    if (searchInput.disabled) return;
    state.query = searchInput.value;
    searchClear.hidden = state.query === '';
    renderNormal();
  });

  searchClear.addEventListener('click', function () {
    if (searchInput.disabled) return;
    clearSearchState();
    renderNormal();
    searchInput.focus({ preventScroll: true });
  });

  function runAction(name) {
    if (name === 'retry') {
      renderNormal();
      showToast('已重新加载');
    } else if (name === 'clear-filters') {
      resetFilters();
      renderNormal();
    } else if (name === 'clear-search') {
      clearSearchState();
      renderNormal();
      searchInput.focus({ preventScroll: true });
    } else if (name === 'publish-first') {
      showToast('已进入发布页');
    }
  }

  listRegion.addEventListener('click', function (event) {
    var card = event.target.closest('.record-card');
    if (card) openRecord(card);
  });

  // 空态/错误态是外壳下的静态块，动作委托在这两个容器上
  [stateEmpty, retryBtn.parentElement].forEach(function (container) {
    container.addEventListener('click', function (event) {
      var action = event.target.closest('[data-action]');
      if (action) runAction(action.dataset.action);
    });
  });

  // shell.js 点演示芯片后广播 mp:state，这里同步统计/控件/空态文案
  document.addEventListener('mp:state', function (event) {
    syncState((event.detail && event.detail.state) || 'normal');
  });

  listRegion.addEventListener('keydown', function (event) {
    var card = event.target.closest('.record-card');
    if (!card) return;
    if (event.key === 'Enter' || event.key === ' ' || event.key === 'Spacebar') {
      event.preventDefault();
      openRecord(card);
    }
  });

  document.addEventListener('keydown', function (event) {
    if (!sheetLayer.hidden) {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeSheet();
        return;
      }
      if (event.key === 'Tab') {
        var focusable = Array.from(sheetLayer.querySelectorAll('button:not(:disabled)'));
        if (!focusable.length) return;
        var first = focusable[0];
        var last = focusable[focusable.length - 1];
        if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last.focus();
        } else if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first.focus();
        }
      }
      return;
    }
  });

  document.getElementById('backBtn').addEventListener('click', function () {
    showToast('已返回上一页');
  });

  updateChips();
  renderNormal();
  resetScroll();
})();
