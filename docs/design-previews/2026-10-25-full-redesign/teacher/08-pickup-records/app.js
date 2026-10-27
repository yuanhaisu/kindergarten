(function () {
  'use strict';

  /* ---------- 数据常量 ---------- */

  var TODAY = '2026-10-26';
  var CLASS_NAME = '小一班';
  var WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];

  var DATE_WINDOW = [
    '2026-10-26',
    '2026-10-25',
    '2026-10-24',
    '2026-10-23',
    '2026-10-22',
    '2026-10-21',
    '2026-10-20'
  ];

  var STUDENTS = [
    { name: '乐乐', guardian: '张敏（妈妈）' },
    { name: '小雨', guardian: '赵静（妈妈）' },
    { name: '朵朵', guardian: '陈静（妈妈）' },
    { name: '果果', guardian: '孙磊（爸爸）' },
    { name: '豆豆', guardian: '周敏（妈妈）' },
    { name: '米米', guardian: '吴涛（爸爸）' },
    { name: '团团', guardian: '李梅（妈妈）' },
    { name: '安安', guardian: '王强（爸爸）' },
    { name: '妞妞', guardian: '刘芳（奶奶）' },
    { name: '天天', guardian: '何静（妈妈）' }
  ];

  var TODAY_RECORDS = [
    {
      time: '17:12', student: '朵朵', type: 'pickup', method: 'manual',
      person: '张伟（舅舅）', override: true, anomaly: false,
      remark: '临时委托舅舅接走，已电话核实母亲', anomalyNote: ''
    },
    {
      time: '17:05', student: '团团', type: 'pickup', method: 'scan',
      person: '李梅（妈妈）', override: false, anomaly: true,
      remark: '家长称孩子发烧，需提前接走', anomalyNote: '提前接走（17:05，早于 17:30 放学）'
    },
    {
      time: '16:52', student: '安安', type: 'pickup', method: 'manual',
      person: '王强（爸爸）', override: false, anomaly: false,
      remark: '', anomalyNote: ''
    },
    {
      time: '16:38', student: '妞妞', type: 'pickup', method: 'scan',
      person: '刘芳（奶奶）', override: false, anomaly: false,
      remark: '', anomalyNote: ''
    },
    { time: '08:41', student: '米米', type: 'dropoff', method: 'scan', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' },
    { time: '08:30', student: '豆豆', type: 'dropoff', method: 'scan', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' },
    { time: '08:20', student: '果果', type: 'dropoff', method: 'manual', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' },
    { time: '08:18', student: '妞妞', type: 'dropoff', method: 'scan', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' },
    { time: '08:12', student: '乐乐', type: 'dropoff', method: 'scan', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' },
    { time: '08:05', student: '小雨', type: 'dropoff', method: 'manual', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' },
    { time: '08:02', student: '团团', type: 'dropoff', method: 'scan', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' },
    { time: '07:58', student: '朵朵', type: 'dropoff', method: 'manual', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' },
    { time: '07:55', student: '安安', type: 'dropoff', method: 'scan', person: '', override: false, anomaly: false, remark: '', anomalyNote: '' }
  ];

  var HISTORY_DROP_TIMES = ['07:52', '07:58', '08:03', '08:08', '08:12', '08:18', '08:22', '08:27', '08:33', '08:41'];
  var HISTORY_PICK_TIMES = ['16:34', '16:41', '16:47', '16:52', '16:58', '17:03', '17:09', '17:14', '17:20', '17:26'];

  var ICON_SCAN = '<svg width="11" height="11" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M1.5 4V2.5a1 1 0 0 1 1-1H4M8 1.5h1.5a1 1 0 0 1 1 1V4M10.5 8v1.5a1 1 0 0 1-1 1H8M4 10.5H2.5a1 1 0 0 1-1-1V8"/><path d="M1.5 6h9"/></svg>';
  var ICON_HAND = '<svg width="11" height="11" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4.1 5.6V3.2a.8.8 0 0 1 1.6 0v2.2"/><path d="M5.7 5.4V2.7a.8.8 0 0 1 1.6 0v2.6"/><path d="M7.3 5.5V3.5a.8.8 0 0 1 1.6 0v4c0 1.9-1.3 3.2-3.1 3.2-1.4 0-2.3-.6-2.9-1.8l-1-1.8a.9.9 0 0 1 1.5-.9l.8 1.1"/></svg>';
  var ICON_CHECK = '<svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m3 8.4 3.3 3.3L13 4.9"/></svg>';

  var ESCAPE_MAP = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };

  /* ---------- 工具函数 ---------- */

  function pad2(value) {
    return value < 10 ? '0' + value : String(value);
  }

  function parseISO(iso) {
    var parts = iso.split('-');
    return new Date(Number(parts[0]), Number(parts[1]) - 1, Number(parts[2]));
  }

  function isoOf(date) {
    return date.getFullYear() + '-' + pad2(date.getMonth() + 1) + '-' + pad2(date.getDate());
  }

  function shiftDate(iso, offset) {
    var date = parseISO(iso);
    date.setDate(date.getDate() + offset);
    return isoOf(date);
  }

  function monthDay(iso) {
    var date = parseISO(iso);
    return (date.getMonth() + 1) + '月' + date.getDate() + '日';
  }

  function weekdayOf(iso) {
    return WEEKDAYS[parseISO(iso).getDay()];
  }

  function isWeekend(iso) {
    var day = parseISO(iso).getDay();
    return day === 0 || day === 6;
  }

  function initialOf(name) {
    return String(name).slice(0, 1);
  }

  function escapeHtml(value) {
    return String(value).replace(/[&<>"']/g, function (character) {
      return ESCAPE_MAP[character];
    });
  }

  function typeLabel(type) {
    return type === 'pickup' ? '接走' : '送入';
  }

  function methodLabel(method) {
    return method === 'scan' ? '扫码' : '手动';
  }

  function buildHistory() {
    var list = [];

    HISTORY_DROP_TIMES.forEach(function (time, index) {
      var student = STUDENTS[index];
      list.push({
        time: time,
        student: student.name,
        type: 'dropoff',
        method: index % 2 === 0 ? 'scan' : 'manual',
        person: '',
        override: false,
        anomaly: false,
        remark: '',
        anomalyNote: ''
      });
    });

    HISTORY_PICK_TIMES.forEach(function (time, index) {
      var student = STUDENTS[index];
      list.push({
        time: time,
        student: student.name,
        type: 'pickup',
        method: index % 2 === 0 ? 'scan' : 'manual',
        person: student.guardian,
        override: false,
        anomaly: false,
        remark: '',
        anomalyNote: ''
      });
    });

    list.sort(function (a, b) {
      return b.time.localeCompare(a.time);
    });

    return list;
  }

  var historyCache = {};

  function recordsForDate(iso) {
    if (iso === TODAY) return TODAY_RECORDS;
    if (isWeekend(iso)) return [];
    if (!historyCache[iso]) historyCache[iso] = buildHistory();
    return historyCache[iso];
  }

  /* ---------- DOM 引用 ---------- */

  var app = document.querySelector('.mp-app');
  var backBtn = document.getElementById('backBtn');
  var prevDay = document.getElementById('prevDay');
  var nextDay = document.getElementById('nextDay');
  var dateBtn = document.getElementById('dateBtn');
  var dateMain = document.getElementById('dateMain');
  var dateWeek = document.getElementById('dateWeek');
  var todayPill = document.getElementById('todayPill');
  var sumTotal = document.getElementById('sumTotal');
  var sumPickup = document.getElementById('sumPickup');
  var sumDropoff = document.getElementById('sumDropoff');
  var groupDate = document.getElementById('groupDate');
  var groupCount = document.getElementById('groupCount');
  var recordList = document.getElementById('recordList');
  var emptyDesc = document.getElementById('emptyDesc');
  var errorDesc = document.getElementById('errorDesc');
  var emptyTodayBtn = document.getElementById('emptyTodayBtn');
  var retryBtn = document.getElementById('retryBtn');
  var toast = document.getElementById('toast');
  var dateSheet = document.getElementById('dateSheet');
  var detailSheet = document.getElementById('detailSheet');
  var dateList = document.getElementById('dateList');
  var detailSub = document.getElementById('detailSub');
  var detailBody = document.getElementById('detailBody');
  var demoChips = Array.prototype.slice.call(document.querySelectorAll('.mp-demo-chip'));

  /* ---------- 状态机 ---------- */

  var state = { date: TODAY, phase: 'idle' };
  var currentRecords = [];
  var loadTimer = null;
  var toastTimer = null;
  var lastFocused = null;

  function effectiveState() {
    if (state.phase === 'loading') return 'loading';
    if (state.phase === 'error') return 'error';
    return currentRecords.length ? 'normal' : 'empty';
  }

  function syncChips(current) {
    demoChips.forEach(function (chip) {
      chip.setAttribute('aria-pressed', chip.getAttribute('data-mp-state') === current ? 'true' : 'false');
    });
  }

  function sync() {
    var list = recordsForDate(state.date);
    currentRecords = list;

    updateDateChrome();

    if (state.phase !== 'loading') {
      updateSummary(list);
      renderGroup(list);
      renderList(list);
      updateDescriptions();
    }

    var next = effectiveState();
    app.setAttribute('data-state', next);
    syncChips(next);
  }

  function updateDateChrome() {
    dateMain.textContent = monthDay(state.date);
    dateWeek.textContent = weekdayOf(state.date);
    todayPill.hidden = state.date !== TODAY;
    nextDay.disabled = state.date >= TODAY;
  }

  function updateSummary(list) {
    var pickups = 0;
    list.forEach(function (record) {
      if (record.type === 'pickup') pickups += 1;
    });

    sumTotal.textContent = String(list.length);
    sumPickup.textContent = String(pickups);
    sumDropoff.textContent = String(list.length - pickups);
  }

  function updateDescriptions() {
    var label = monthDay(state.date) + ' ' + weekdayOf(state.date);
    emptyDesc.textContent = isWeekend(state.date) ? label + ' · 全园休息' : '换个日期看看，或下拉刷新';
    errorDesc.textContent = '网络异常，没取到 ' + monthDay(state.date) + ' 的接送记录，请重试';
  }

  /* ---------- 页面渲染 ---------- */

  function renderGroup(list) {
    groupDate.textContent = monthDay(state.date) + ' ' + weekdayOf(state.date);
    groupCount.textContent = '共 ' + list.length + ' 条';
  }

  function renderList(list) {
    recordList.innerHTML = list.map(buildRow).join('');
  }

  function buildRow(record, index) {
    var isPickup = record.type === 'pickup';
    var methodModifier = record.method === 'scan' ? 'scan' : 'manual';
    var methodIcon = record.method === 'scan' ? ICON_SCAN : ICON_HAND;
    var meta = '<span class="tl-type tl-type--' + record.type + '">' + typeLabel(record.type) + '</span>';

    if (isPickup && record.person) {
      meta += '<span class="tl-dotsep" aria-hidden="true">·</span>';
      meta += '<span class="tl-person">' + escapeHtml(record.person) + '</span>';
    }

    if (record.override) {
      meta += '<span class="tl-flag tl-flag--override">例外</span>';
    }

    if (record.anomaly) {
      meta += '<span class="tl-flag tl-flag--danger">异常</span>';
    }

    var remark = record.remark
      ? '<span class="tl-remark"><span class="tl-remark-label">理由</span>' + escapeHtml(record.remark) + '</span>'
      : '';

    return '<button class="tl-row' + (record.anomaly ? ' tl-row--danger' : '') + '" type="button" data-idx="' + index + '" data-od-id="pickup-record-' + index + '" aria-label="查看 ' + escapeHtml(record.student) + ' ' + record.time + ' 的接送记录">'
      + '<span class="tl-time">' + record.time + '</span>'
      + '<span class="tl-rail" aria-hidden="true"><span class="tl-dot tl-dot--' + record.type + '"></span></span>'
      + '<span class="tl-body">'
      + '<span class="tl-top">'
      + '<span class="tl-avatar" aria-hidden="true">' + escapeHtml(initialOf(record.student)) + '</span>'
      + '<span class="tl-name">' + escapeHtml(record.student) + '</span>'
      + '<span class="tl-class">' + CLASS_NAME + '</span>'
      + '<span class="tl-method tl-method--' + methodModifier + '">' + methodIcon + methodLabel(record.method) + '</span>'
      + '</span>'
      + '<span class="tl-meta">' + meta + '</span>'
      + remark
      + '</span>'
      + '</button>';
  }

  /* ---------- 弹层 ---------- */

  function getOpenSheetId() {
    if (dateSheet && !dateSheet.hidden) return 'dateSheet';
    if (detailSheet && !detailSheet.hidden) return 'detailSheet';
    return '';
  }

  function openSheet(id, trigger) {
    var layer = document.getElementById(id);
    if (!layer || !layer.hidden) return;
    if (getOpenSheetId()) return;

    lastFocused = trigger || document.activeElement;
    if (id === 'dateSheet') dateBtn.setAttribute('aria-expanded', 'true');

    layer.hidden = false;

    var closeButton = layer.querySelector('.sheet-close');
    if (closeButton) closeButton.focus({ preventScroll: true });
  }

  function closeSheet(id) {
    var layer = document.getElementById(id);
    if (!layer || layer.hidden) return;

    layer.hidden = true;
    if (id === 'dateSheet') dateBtn.setAttribute('aria-expanded', 'false');

    if (lastFocused && document.contains(lastFocused)) {
      lastFocused.focus({ preventScroll: true });
    }
    lastFocused = null;
  }

  function renderDateSheet() {
    dateList.innerHTML = DATE_WINDOW.map(function (iso) {
      var count = recordsForDate(iso).length;
      var selected = iso === state.date;
      var isToday = iso === TODAY;
      var side = '';

      if (isToday) {
        side += '<span class="today-pill">今天</span>';
      }
      side += '<span class="date-option-count">' + (count ? count + ' 条' : '无记录') + '</span>';
      if (selected) {
        side += '<span class="date-option-check" aria-hidden="true">' + ICON_CHECK + '</span>';
      }

      var accessibleLabel = monthDay(iso) + ' ' + weekdayOf(iso)
        + (isToday ? '，今天' : '')
        + '，' + (count ? count + ' 条记录' : '无记录');

      return '<button class="date-option' + (selected ? ' is-selected' : '') + '" type="button" data-date="' + iso + '" data-od-id="date-option-' + iso + '"'
        + (selected ? ' aria-current="date"' : '')
        + ' aria-label="' + escapeHtml(accessibleLabel) + '">'
        + '<span class="date-option-main">'
        + '<span class="date-option-day">' + monthDay(iso) + '</span>'
        + '<span class="date-option-week">' + weekdayOf(iso) + '</span>'
        + '</span>'
        + '<span class="date-option-side">' + side + '</span>'
        + '</button>';
    }).join('');
  }

  function openDateSheet() {
    renderDateSheet();
    dateList.scrollTop = 0;
    openSheet('dateSheet', dateBtn);
  }

  function buildDetail(record) {
    var isPickup = record.type === 'pickup';
    var flags = '';

    if (record.override) {
      flags += '<span class="tl-flag tl-flag--override">例外</span>';
    }
    if (record.anomaly) {
      flags += '<span class="tl-flag tl-flag--danger">异常</span>';
    }

    var rows = '';
    rows += detailRow('接送类型', typeLabel(record.type));
    rows += detailRow('记录时间', monthDay(state.date) + ' ' + weekdayOf(state.date) + ' ' + record.time);

    if (isPickup && record.person) {
      rows += detailRow('接送人', record.person);
    }

    rows += detailRow('核验方式', methodLabel(record.method));

    if (isPickup) {
      rows += detailRow('授权状态', record.override ? '例外放行 · 非授权接送人' : '有效授权接送人');
    }

    if (record.anomaly && record.anomalyNote) {
      rows += detailRow('异常备注', record.anomalyNote);
    }

    if (record.remark) {
      rows += detailRow('理由', record.remark);
    }

    return '<div class="detail-hero">'
      + '<span class="detail-avatar" aria-hidden="true">' + escapeHtml(initialOf(record.student)) + '</span>'
      + '<div class="detail-hero-main">'
      + '<div class="detail-name-row">'
      + '<span class="detail-name">' + escapeHtml(record.student) + '</span>'
      + flags
      + '</div>'
      + '<p class="detail-subline">' + CLASS_NAME + ' · ' + typeLabel(record.type) + '</p>'
      + '</div>'
      + '<span class="detail-time">' + record.time + '</span>'
      + '</div>'
      + '<dl class="detail-list">' + rows + '</dl>';
  }

  function detailRow(label, value) {
    return '<div class="detail-row"><dt>' + label + '</dt><dd>' + escapeHtml(value) + '</dd></div>';
  }

  function openDetail(index, trigger) {
    var record = currentRecords[index];
    if (!record) return;

    detailSub.textContent = monthDay(state.date) + ' ' + weekdayOf(state.date) + ' · ' + record.time;
    detailBody.innerHTML = buildDetail(record);
    detailBody.scrollTop = 0;
    openSheet('detailSheet', trigger);
  }

  /* ---------- Toast 与状态切换 ---------- */

  function showToast(message) {
    clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-on');
    toastTimer = setTimeout(function () {
      toast.classList.remove('is-on');
    }, 1800);
  }

  function loadInto(date) {
    clearTimeout(loadTimer);
    if (typeof date === 'string' && date) state.date = date;

    state.phase = 'loading';
    sync();

    loadTimer = setTimeout(function () {
      state.phase = 'idle';
      sync();
    }, 380);
  }

  function applyDemoState(name) {
    if (!name) return;

    clearTimeout(loadTimer);

    if (name === 'normal') {
      state.date = TODAY;
      state.phase = 'idle';
    } else if (name === 'empty') {
      state.date = '2026-10-25';
      state.phase = 'idle';
    } else if (name === 'loading') {
      state.date = TODAY;
      state.phase = 'loading';
    } else if (name === 'error') {
      state.date = TODAY;
      state.phase = 'error';
    } else {
      return;
    }

    sync();
  }

  /* ---------- 事件绑定 ---------- */

  backBtn.addEventListener('click', function () {
    showToast('已返回接送管理');
  });

  prevDay.addEventListener('click', function () {
    loadInto(shiftDate(state.date, -1));
  });

  nextDay.addEventListener('click', function () {
    if (state.date >= TODAY) return;
    var next = shiftDate(state.date, 1);
    if (next > TODAY) return;
    loadInto(next);
  });

  dateBtn.addEventListener('click', function () {
    openDateSheet();
  });

  emptyTodayBtn.addEventListener('click', function () {
    loadInto(TODAY);
  });

  retryBtn.addEventListener('click', function () {
    loadInto(null);
  });

  recordList.addEventListener('click', function (event) {
    var row = event.target.closest('.tl-row');
    if (!row || !recordList.contains(row)) return;

    var index = parseInt(row.getAttribute('data-idx'), 10);
    if (isNaN(index) || !currentRecords[index]) return;
    openDetail(index, row);
  });

  dateList.addEventListener('click', function (event) {
    var option = event.target.closest('.date-option');
    if (!option) return;

    var date = option.getAttribute('data-date');
    if (!date) return;

    closeSheet('dateSheet');
    loadInto(date);
  });

  document.addEventListener('click', function (event) {
    var closer = event.target.closest ? event.target.closest('[data-close]') : null;
    if (!closer) return;
    closeSheet(closer.getAttribute('data-close'));
  });

  document.addEventListener('keydown', function (event) {
    var openId = getOpenSheetId();
    if (!openId) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      closeSheet(openId);
      return;
    }

    if (event.key !== 'Tab') return;

    var sheet = document.querySelector('#' + openId + ' .sheet');
    if (!sheet) return;

    var focusables = Array.prototype.slice.call(
      sheet.querySelectorAll('button:not(:disabled):not([tabindex="-1"])')
    );
    if (!focusables.length) return;

    var first = focusables[0];
    var last = focusables[focusables.length - 1];
    var active = document.activeElement;

    if (event.shiftKey) {
      if (active === first || !sheet.contains(active)) {
        event.preventDefault();
        last.focus({ preventScroll: true });
      }
    } else if (active === last || !sheet.contains(active)) {
      event.preventDefault();
      first.focus({ preventScroll: true });
    }
  });

  demoChips.forEach(function (chip) {
    chip.addEventListener('click', function () {
      applyDemoState(chip.getAttribute('data-mp-state'));
    });
  });

  document.addEventListener('mp:state', function (event) {
    var detail = event.detail;
    var next = typeof detail === 'string' ? detail : detail && detail.state;
    if (typeof next === 'string') applyDemoState(next);
  });

  /* ---------- 启动 ---------- */

  sync();
})();
