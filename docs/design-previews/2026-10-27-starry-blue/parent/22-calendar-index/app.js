(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  var els = {
    page: document.querySelector('.page'),
    changesCard: document.getElementById('changesCard'),
    changeList: document.getElementById('changeList'),
    changesToggle: document.getElementById('changesToggle'),
    changesToggleText: document.getElementById('changesToggleText'),
    changesCount: document.getElementById('changesCount'),
    monthTitle: document.getElementById('monthTitle'),
    monthSub: document.getElementById('monthSub'),
    prevMonth: document.getElementById('prevMonth'),
    nextMonth: document.getElementById('nextMonth'),
    todayBtn: document.getElementById('todayBtn'),
    calendarCard: document.getElementById('calendarCard'),
    calendarLabel: document.getElementById('calendarLabel'),
    dayGrid: document.getElementById('dayGrid'),
    calendarLegend: document.getElementById('calendarLegend'),
    eventsDate: document.getElementById('eventsDate'),
    eventsCount: document.getElementById('eventsCount'),
    eventList: document.getElementById('eventList'),
    dayEmpty: document.getElementById('dayEmpty'),
    dayEmptyTitle: document.getElementById('dayEmptyTitle'),
    monthEmpty: document.getElementById('monthEmpty'),
    monthEmptyTitle: document.getElementById('monthEmptyTitle'),
    backToday: document.getElementById('backToday'),
    skeletonGrid: document.getElementById('skeletonGrid'),
    eventSkeletons: document.getElementById('eventSkeletons'),
    retryCalendar: document.getElementById('retryCalendar'),
    liveStatus: document.getElementById('liveStatus')
  };
  if (!els.dayGrid || !els.eventList) return;

  var TODAY = { y: 2026, m: 9, d: 26 };
  var TODAY_ISO = '2026-10-26';
  var DAY_NAMES = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
  var TYPE_ORDER = ['notice', 'activity', 'holiday', 'meeting', 'checkup'];
  var TYPE_INFO = {
    notice: { label: '通知', char: '通' },
    activity: { label: '活动', char: '活' },
    holiday: { label: '假期', char: '假' },
    meeting: { label: '家长会', char: '会' },
    checkup: { label: '体检', char: '检' }
  };
  var VALID_STATES = { normal: true, empty: true, loading: true, error: true };

  var EVENTS = [
    {
      id: 'national-day',
      title: '国庆假期',
      description: '国庆假期，园所按法定节假日安排放假。请留意返园时间，假期后按时送孩子回园。如有出行计划，请注意旅途安全，并关注天气变化。',
      event_type: 'holiday',
      event_type_label: '假期',
      event_date: '2026-10-01',
      start_time: null,
      end_time: null,
      location: '',
      is_all_day: true,
      change_summary: ''
    },
    {
      id: 'flu-vaccine',
      title: '秋季流感疫苗接种',
      description: '区妇幼保健院入园为孩子们接种秋季流感疫苗，请家长提前确认孩子的健康状况，并留意班级群里的接种提醒。接种当天请让孩子穿宽松衣物，便于露出上臂。',
      event_type: 'notice',
      event_type_label: '通知',
      event_date: '2026-10-08',
      start_time: null,
      end_time: null,
      location: '保健室',
      is_all_day: true,
      change_summary: ''
    },
    {
      id: 'parent-open-day',
      title: '家长开放日',
      description: '邀请家长走进班级，观摩晨圈、集体活动与户外游戏，近距离了解孩子在园的生活节奏。请提前十分钟到园签到，活动期间将手机调至静音。',
      event_type: 'meeting',
      event_type_label: '家长会',
      event_date: '2026-10-14',
      start_time: '09:00',
      end_time: '11:00',
      location: '小一班教室',
      is_all_day: false,
      change_summary: ''
    },
    {
      id: 'autumn-outing',
      title: '秋游',
      description: '小班亲子秋游，请为孩子准备轻便午餐、饮用水和遮阳帽，按班级通知的时间和地点集合。家长全程陪同，并请为孩子准备一套备用衣物。',
      event_type: 'activity',
      event_type_label: '活动',
      event_date: '2026-10-16',
      start_time: null,
      end_time: null,
      location: '城市植物园',
      is_all_day: true,
      change_summary: ''
    },
    {
      id: 'fire-drill',
      title: '消防安全演练',
      description: '全园开展消防安全疏散演练，帮助孩子熟悉逃生路线与集合地点。当天正常入园，建议穿轻便运动鞋。演练期间请家长留意班级群通知。',
      event_type: 'activity',
      event_type_label: '活动',
      event_date: '2026-10-21',
      start_time: '10:00',
      end_time: '10:40',
      location: '操场',
      is_all_day: false,
      change_summary: ''
    },
    {
      id: 'sports-rehearsal',
      title: '秋季运动会彩排',
      description: '各班按方阵入场，练习接力与亲子项目的走位和交接。请提前十分钟送孩子到园，穿好运动鞋并带好水壶；家长志愿者请按班级群通知到场。',
      event_type: 'activity',
      event_type_label: '活动',
      event_date: '2026-10-26',
      start_time: '09:00',
      end_time: '10:30',
      location: '大操场',
      is_all_day: false,
      change_summary: '时间 10:00 → 10:30'
    },
    {
      id: 'class-meeting',
      title: '班级家长会',
      description: '沟通本月孩子的成长重点、在园表现与秋季运动会安排，请家长提前十分钟到场签到。会议期间请将手机调至静音，如无法到场请提前向老师请假。',
      event_type: 'meeting',
      event_type_label: '家长会',
      event_date: '2026-10-26',
      start_time: '16:00',
      end_time: '17:00',
      location: '小一班教室',
      is_all_day: false,
      change_summary: '地点 多功能厅 → 小一班教室'
    },
    {
      id: 'vision-screening',
      title: '视力筛查',
      description: '区妇幼保健院入园为孩子们进行视力筛查，请当天准时送孩子入园。如近期眼睛不适，请提前告知班级老师；筛查结果将由老师单独反馈。',
      event_type: 'checkup',
      event_type_label: '体检',
      event_date: '2026-10-27',
      start_time: null,
      end_time: null,
      location: '医务室',
      is_all_day: true,
      change_summary: '地点 保健室 → 医务室'
    },
    {
      id: 'double-ninth',
      title: '重阳节敬老活动',
      description: '邀请爷爷奶奶走进班级，观看孩子们的节目表演，一起做手工、品重阳糕，感受敬老爱老的传统节日氛围。请为孩子准备一句节日祝福。',
      event_type: 'activity',
      event_type_label: '活动',
      event_date: '2026-10-30',
      start_time: '09:30',
      end_time: '10:30',
      location: '多功能厅',
      is_all_day: false,
      change_summary: '时间 15:30 → 09:30'
    },
    {
      id: 'family-sports-day',
      title: '秋季亲子运动会',
      description: '全园亲子运动会，请家长和孩子穿好运动装、运动鞋，按班级方阵入场，并为孩子准备充足饮用水。请按班级通知提前到场热身。',
      event_type: 'activity',
      event_type_label: '活动',
      event_date: '2026-11-03',
      start_time: '09:00',
      end_time: '11:30',
      location: '大操场',
      is_all_day: false,
      change_summary: ''
    },
    {
      id: 'parent-open-day-nov',
      title: '家长开放日',
      description: '邀请家长进班观摩半日活动，了解孩子在园的学习、游戏与同伴交往。请提前十分钟到园签到，活动期间请将手机调至静音。',
      event_type: 'meeting',
      event_type_label: '家长会',
      event_date: '2026-11-11',
      start_time: '09:00',
      end_time: '11:00',
      location: '',
      is_all_day: false,
      change_summary: ''
    }
  ];

  var state = app.getAttribute('data-state') || 'normal';
  if (!VALID_STATES[state]) state = 'normal';
  var view = { y: TODAY.y, m: TODAY.m };
  var selected = TODAY_ISO;
  var changesOpen = false;
  var expanded = {};
  var flashTimer = null;
  var flashingCard = null;
  var reduceMotion = !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);

  function pad2(n) {
    return n < 10 ? '0' + n : String(n);
  }

  function isoOf(y, m, d) {
    return y + '-' + pad2(m + 1) + '-' + pad2(d);
  }

  function parseISO(iso) {
    var parts = String(iso).split('-');
    return { y: Number(parts[0]), m: Number(parts[1]) - 1, d: Number(parts[2]) };
  }

  function daysInMonth(y, m) {
    return new Date(y, m + 1, 0).getDate();
  }

  function dayOfWeek(iso) {
    var p = parseISO(iso);
    return new Date(p.y, p.m, p.d).getDay();
  }

  function shortDate(iso) {
    var p = parseISO(iso);
    return (p.m + 1) + '月' + p.d + '日';
  }

  function longDate(iso) {
    return shortDate(iso) + ' ' + DAY_NAMES[dayOfWeek(iso)];
  }

  function esc(value) {
    return String(value == null ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function compareEvents(a, b) {
    if (a.event_date !== b.event_date) return a.event_date < b.event_date ? -1 : 1;
    if (a.is_all_day !== b.is_all_day) return a.is_all_day ? -1 : 1;
    var at = a.start_time || '00:00';
    var bt = b.start_time || '00:00';
    if (at !== bt) return at < bt ? -1 : 1;
    return a.title.localeCompare(b.title, 'zh-CN');
  }

  function eventsInMonth(y, m) {
    var prefix = y + '-' + pad2(m + 1);
    return EVENTS.filter(function (item) {
      return item.event_date.slice(0, 7) === prefix;
    }).sort(compareEvents);
  }

  function eventsOn(iso) {
    return EVENTS.filter(function (item) {
      return item.event_date === iso;
    }).sort(compareEvents);
  }

  function findEvent(id) {
    for (var i = 0; i < EVENTS.length; i += 1) {
      if (EVENTS[i].id === id) return EVENTS[i];
    }
    return null;
  }

  function timeLabel(item) {
    if (item.is_all_day) return '全天';
    if (item.start_time && item.end_time) return item.start_time + '–' + item.end_time;
    return item.start_time || item.end_time || '全天';
  }

  function currentMonthEvents() {
    if (state === 'empty') return [];
    return eventsInMonth(view.y, view.m);
  }

  function syncChips() {
    var chips = document.querySelectorAll('.mp-demo-chip');
    Array.prototype.forEach.call(chips, function (chip) {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === state));
    });
  }

  function renderToolbar() {
    var monthEvents = currentMonthEvents();
    var isCurrent = view.y === TODAY.y && view.m === TODAY.m;
    els.monthTitle.textContent = view.y + '年' + (view.m + 1) + '月';
    if (state === 'loading') {
      els.monthSub.textContent = '正在加载本月活动';
    } else if (state === 'error') {
      els.monthSub.textContent = '本月活动未加载';
    } else {
      els.monthSub.textContent = monthEvents.length ? '本月 ' + monthEvents.length + ' 个活动' : '本月暂无活动';
    }
    // 只有「已在今天」才无可跳之处；当前月选中其他日期时仍应能一键回到今天。
    els.todayBtn.disabled = isCurrent && selected === TODAY_ISO;
  }

  function changeItemHTML(item) {
    return '<button class="change-item" type="button" data-change-event="' + item.id + '" data-od-id="change-item-' + item.id + '" aria-label="查看' + esc(longDate(item.event_date)) + '的' + esc(item.title) + '变更">' +
      '<span class="change-dot" aria-hidden="true"></span>' +
      '<span class="change-copy">' +
        '<span class="change-title">' + esc(item.title) + '</span>' +
        '<span class="change-diff">' + esc(shortDate(item.event_date)) + ' · ' + esc(item.change_summary) + '</span>' +
      '</span>' +
      '<svg class="change-chevron" width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m4.5 3 3 3-3 3"/></svg>' +
    '</button>';
  }

  function renderChanges() {
    // 变更窗口由服务端按「近 7 天修订且尚未发生」下发，与当前查看的月份无关。
    // 只按视图月份过滤会让隔壁月的调整被静默吞掉，而家长恰恰可能按旧时间行动。
    var items = state === 'normal' ? EVENTS.filter(function (item) {
      return !!item.change_summary;
    }).sort(compareEvents) : [];
    if (!items.length) {
      els.changesCard.hidden = true;
      return;
    }

    els.changesCard.hidden = false;
    els.changesCount.textContent = items.length + ' 条';
    var visible = changesOpen ? items : items.slice(0, 3);
    els.changeList.innerHTML = visible.map(changeItemHTML).join('');
    var extra = Math.max(0, items.length - 3);
    var showToggle = changesOpen || extra > 0;
    els.changesToggle.hidden = !showToggle;
    els.changesToggle.setAttribute('aria-expanded', changesOpen ? 'true' : 'false');
    els.changesToggleText.textContent = changesOpen ? '收起' : '另有 ' + extra + ' 条';
  }

  function renderLegend(monthEvents) {
    if (state !== 'normal' || !monthEvents.length) {
      els.calendarLegend.hidden = true;
      els.calendarLegend.innerHTML = '';
      return;
    }
    var present = {};
    monthEvents.forEach(function (item) { present[item.event_type] = true; });
    var types = TYPE_ORDER.filter(function (type) { return present[type]; });
    if (!types.length) {
      els.calendarLegend.hidden = true;
      els.calendarLegend.innerHTML = '';
      return;
    }
    els.calendarLegend.innerHTML = types.map(function (type) {
      var info = TYPE_INFO[type];
      return '<span class="legend-item"><span class="type-dot type-dot--' + type + '" aria-hidden="true"></span>' + esc(info.label) + '</span>';
    }).join('');
    els.calendarLegend.hidden = false;
  }

  function renderGrid() {
    var firstDay = new Date(view.y, view.m, 1).getDay();
    var monthEvents = currentMonthEvents();
    var html = '';

    for (var i = 0; i < 42; i += 1) {
      var date = new Date(view.y, view.m, 1 - firstDay + i);
      var y = date.getFullYear();
      var m = date.getMonth();
      var d = date.getDate();
      var iso = isoOf(y, m, d);
      var isCurrentMonth = y === view.y && m === view.m;
      var isToday = iso === TODAY_ISO;
      var isSelected = isCurrentMonth && iso === selected;
      var dayEvents = isCurrentMonth && state === 'normal' ? eventsOn(iso) : [];
      var classes = 'day';
      var attrs = '';
      var label = (m + 1) + '月' + d + '日 ' + DAY_NAMES[date.getDay()];

      if (!isCurrentMonth) {
        classes += ' is-outside';
        attrs = ' tabindex="-1" aria-disabled="true"';
        label += '，非本月日期';
      } else {
        if (isSelected) classes += ' is-selected';
        if (isToday) classes += ' is-today';
        attrs = ' data-date="' + iso + '" aria-pressed="' + (isSelected ? 'true' : 'false') + '" data-od-id="calendar-day-' + iso + '"';
        if (isToday) {
          attrs += ' aria-current="date"';
          label += '，今天';
        }
        if (dayEvents.length) label += '，' + dayEvents.length + ' 个活动';
      }

      var dots = dayEvents.slice(0, 3).map(function (item) {
        return '<span class="type-dot type-dot--' + item.event_type + '" aria-hidden="true"></span>';
      }).join('');

      html += '<button class="' + classes + '" type="button"' + (isCurrentMonth ? '' : ' disabled') + ' aria-label="' + esc(label) + '"' + attrs + '>' +
        '<span class="day-num">' + d + '</span>' +
        '<span class="day-dots" aria-hidden="true">' + dots + '</span>' +
      '</button>';
    }

    els.dayGrid.innerHTML = html;
    els.dayGrid.setAttribute('aria-label', view.y + '年' + (view.m + 1) + '月日期');
    els.calendarLabel.textContent = view.y + '年' + (view.m + 1) + '月月历';
    renderLegend(monthEvents);
  }

  function updateSelection() {
    var nodes = els.dayGrid.querySelectorAll('.day[data-date]');
    Array.prototype.forEach.call(nodes, function (node) {
      var on = node.getAttribute('data-date') === selected;
      node.classList.toggle('is-selected', on);
      node.setAttribute('aria-pressed', on ? 'true' : 'false');
    });
  }

  function eventCardHTML(item) {
    var type = item.event_type;
    var info = TYPE_INFO[type] || TYPE_INFO.activity;
    var label = item.event_type_label || info.label;
    var open = expanded[item.id] === true;
    var titleId = 'event-title-' + item.id;
    var metaId = 'event-meta-' + item.id;
    var descId = 'event-desc-' + item.id;
    var changeId = 'event-change-' + item.id;
    var describedBy = metaId + ' ' + descId + (item.change_summary ? ' ' + changeId : '');
    var change = item.change_summary
      ? '<p class="event-change" id="' + changeId + '"><span class="change-label">安排调整</span><span>' + esc(item.change_summary) + '</span></p>'
      : '';
    var location = item.location
      ? '<span class="event-loc"><svg width="12" height="12" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 14s4.6-4.3 4.6-8A4.6 4.6 0 0 0 3.4 6c0 3.7 4.6 8 4.6 8Z"/><circle cx="8" cy="6" r="1.7"/></svg>' + esc(item.location) + '</span>'
      : '';

    return '<article class="event-card evt-' + type + (open ? ' is-expanded' : '') + '" data-event-id="' + item.id + '" data-od-id="event-card-' + item.id + '" data-expandable="true" role="button" tabindex="0" aria-expanded="' + open + '" aria-controls="' + descId + '" aria-labelledby="' + titleId + '" aria-describedby="' + describedBy + '">' +
      '<div class="event-head">' +
        '<span class="event-icon" aria-hidden="true">' + esc(info.char) + '</span>' +
        '<div class="event-summary">' +
          '<div class="event-title-row">' +
            '<h3 class="event-title" id="' + titleId + '">' + esc(item.title) + '</h3>' +
          '</div>' +
          '<p class="event-meta" id="' + metaId + '"><span class="meta-type">' + esc(label) + '</span><span class="meta-sep" aria-hidden="true">·</span><span class="event-time">' + esc(timeLabel(item)) + '</span></p>' +
        '</div>' +
      '</div>' +
      '<p class="event-desc" id="' + descId + '">' + esc(item.description) + '</p>' +
      change +
      '<div class="event-foot">' + location +
        '<span class="expand-hint" aria-hidden="true"><span class="expand-label">' + (open ? '收起' : '展开全文') + '</span><svg class="expand-caret" width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="m3 4.5 3 3 3-3"/></svg></span>' +
      '</div>' +
    '</article>';
  }

  function renderDay() {
    var p = parseISO(selected);
    var isLoading = state === 'loading';
    var monthEvents = currentMonthEvents();
    var dayEvents = state === 'normal' ? eventsOn(selected) : [];
    var noMonthEvents = state === 'normal' && monthEvents.length === 0;
    var showMonthEmpty = state === 'empty' || noMonthEvents;
    var showDayEmpty = state === 'normal' && !noMonthEvents && dayEvents.length === 0;

    els.eventsDate.textContent = shortDate(selected) + ' ' + DAY_NAMES[dayOfWeek(selected)];
    els.eventsCount.textContent = isLoading
      ? '加载中'
      : (state === 'error' ? '—' : (dayEvents.length ? dayEvents.length + ' 个活动' : '暂无活动'));
    els.eventList.innerHTML = dayEvents.map(eventCardHTML).join('');
    els.eventList.hidden = dayEvents.length === 0;
    els.dayEmpty.hidden = !showDayEmpty;
    els.monthEmpty.hidden = !showMonthEmpty;
    els.dayEmptyTitle.textContent = shortDate(selected) + '暂无活动安排';
    els.monthEmptyTitle.textContent = (p.m + 1) + '月暂无活动安排';

    if (isLoading) {
      els.dayEmpty.hidden = true;
      els.monthEmpty.hidden = true;
    }
  }

  function renderAll() {
    app.setAttribute('data-state', state);
    if (els.page) els.page.setAttribute('aria-busy', state === 'loading' ? 'true' : 'false');
    renderToolbar();
    renderChanges();
    renderGrid();
    renderDay();
  }

  function buildSkeletons() {
    var cells = '';
    for (var i = 0; i < 42; i += 1) {
      cells += '<span class="sk-day"><span class="sk-orb"></span></span>';
    }
    if (els.skeletonGrid) els.skeletonGrid.innerHTML = cells;

    var card = '<div class="event-skeleton">' +
      '<div class="skeleton-head"><span class="sk-icon"></span><span class="sk-lines"><span class="sk-line sk-title"></span><span class="sk-line sk-meta"></span></span></div>' +
      '<span class="sk-line sk-copy"></span>' +
      '<span class="sk-line sk-copy sk-copy-short"></span>' +
      '<div class="sk-foot"><span class="sk-foot-line"></span><span class="sk-foot-line"></span></div>' +
    '</div>';
    if (els.eventSkeletons) els.eventSkeletons.innerHTML = card + card + card;
  }

  function applyMode(next) {
    if (!VALID_STATES[next]) return;
    var previous = state;
    state = next;

    if (state === 'empty') {
      view = { y: 2026, m: 11 };
      selected = isoOf(2026, 11, 26);
    } else if (state === 'normal' && previous === 'empty') {
      view = { y: TODAY.y, m: TODAY.m };
      selected = TODAY_ISO;
    }

    if (state !== 'normal') changesOpen = false;
    expanded = {};
    syncChips();
    renderAll();
  }

  function goToday() {
    state = 'normal';
    app.setAttribute('data-state', 'normal');
    view = { y: TODAY.y, m: TODAY.m };
    selected = TODAY_ISO;
    changesOpen = false;
    expanded = {};
    syncChips();
    renderAll();
    els.liveStatus.textContent = '已回到今天，' + longDate(selected);
  }

  function selectDate(iso) {
    if (iso === selected) {
      renderDay();
      return;
    }
    selected = iso;
    expanded = {};
    updateSelection();
    renderToolbar();
    renderDay();
  }

  function shiftMonth(delta) {
    var first = new Date(view.y, view.m + delta, 1);
    var previousDay = parseISO(selected).d;
    view = { y: first.getFullYear(), m: first.getMonth() };
    selected = (view.y === TODAY.y && view.m === TODAY.m)
      ? TODAY_ISO
      : isoOf(view.y, view.m, Math.min(previousDay, daysInMonth(view.y, view.m)));
    changesOpen = false;
    expanded = {};
    renderAll();
    els.liveStatus.textContent = '已切换到' + view.y + '年' + (view.m + 1) + '月';
  }

  function toggleCard(card) {
    var id = card.getAttribute('data-event-id');
    var open = card.getAttribute('aria-expanded') !== 'true';
    expanded[id] = open;
    card.classList.toggle('is-expanded', open);
    card.setAttribute('aria-expanded', open ? 'true' : 'false');
    var label = card.querySelector('.expand-label');
    if (label) label.textContent = open ? '收起' : '展开全文';
  }

  function flashCard(card) {
    if (flashTimer) window.clearTimeout(flashTimer);
    if (flashingCard) flashingCard.classList.remove('is-flash');
    flashingCard = card;
    card.classList.add('is-flash');
    flashTimer = window.setTimeout(function () {
      card.classList.remove('is-flash');
      if (flashingCard === card) flashingCard = null;
    }, 2400);
  }

  function jumpToEvent(id) {
    var item = findEvent(id);
    if (!item) return;
    var p = parseISO(item.event_date);
    view = { y: p.y, m: p.m };
    selected = item.event_date;
    state = 'normal';
    app.setAttribute('data-state', 'normal');
    expanded = {};
    syncChips();
    renderAll();

    window.requestAnimationFrame(function () {
      var card = els.eventList.querySelector('[data-event-id="' + item.id + '"]');
      if (!card) return;
      var top = window.pageYOffset + card.getBoundingClientRect().top - 76;
      window.scrollTo({ top: Math.max(0, top), behavior: reduceMotion ? 'auto' : 'smooth' });
      flashCard(card);
      try {
        card.focus({ preventScroll: true });
      } catch (err) {
        card.focus();
      }
      els.liveStatus.textContent = '已定位到' + longDate(item.event_date) + '的' + item.title;
    });
  }

  function closestFrom(node, selector) {
    while (node && node !== document) {
      if (node.nodeType === 1 && node.matches && node.matches(selector)) return node;
      node = node.parentNode;
    }
    return null;
  }

  els.prevMonth.addEventListener('click', function () { shiftMonth(-1); });
  els.nextMonth.addEventListener('click', function () { shiftMonth(1); });
  els.todayBtn.addEventListener('click', function () {
    if (!els.todayBtn.disabled) goToday();
  });
  els.backToday.addEventListener('click', goToday);
  els.retryCalendar.addEventListener('click', function () {
    state = 'normal';
    app.setAttribute('data-state', 'normal');
    syncChips();
    renderAll();
    els.liveStatus.textContent = '本月活动已重新加载';
  });

  els.changesToggle.addEventListener('click', function () {
    changesOpen = !changesOpen;
    renderChanges();
  });

  els.changeList.addEventListener('click', function (event) {
    var row = closestFrom(event.target, '.change-item');
    if (!row || !els.changeList.contains(row)) return;
    jumpToEvent(row.getAttribute('data-change-event'));
  });

  els.dayGrid.addEventListener('click', function (event) {
    var cell = closestFrom(event.target, '.day');
    if (!cell || cell.disabled || !els.dayGrid.contains(cell)) return;
    var iso = cell.getAttribute('data-date');
    if (iso) selectDate(iso);
  });

  els.eventList.addEventListener('click', function (event) {
    var card = closestFrom(event.target, '.event-card');
    if (!card || !els.eventList.contains(card)) return;
    toggleCard(card);
  });

  els.eventList.addEventListener('keydown', function (event) {
    if (event.key !== 'Enter' && event.key !== ' ') return;
    var card = closestFrom(event.target, '.event-card');
    if (!card || !els.eventList.contains(card)) return;
    event.preventDefault();
    toggleCard(card);
  });

  document.addEventListener('mp:state', function (event) {
    if (!event.detail || !event.detail.state) return;
    applyMode(event.detail.state);
  });

  buildSkeletons();
  syncChips();
  renderAll();
})();
