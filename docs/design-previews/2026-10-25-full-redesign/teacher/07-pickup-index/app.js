(function () {
  'use strict';

  var CLASS_NAME = '小一班';

  /* 数据取自 GET /teacher/pickup/overview 与 /students 的真实字段：
   * status / arrive_time / authorized_persons / is_overdue / pickup_deadline
   * / last_remark / is_authorized_override。全班 10 人，数字自洽：
   * arrived(9) = picked_up(3) + not_picked_up(6)。
   * 契约 students 无接走时间/接走人，卡片只展示授权接送人。
   * STUDENTS 是可变的演示副本，切回「正常」chip 会重置为 INITIAL。 */
  var STUDENTS = [
    {
      name: '乐乐', status: 'not_picked_up', arrive: '08:12', overdue: true, deadline: '17:30',
      persons: [
        { name: '李梅', relation: '妈妈', phone: '138****6688' },
        { name: '王强', relation: '爸爸', phone: '139****2233' }
      ]
    },
    {
      name: '小雨', status: 'not_picked_up', arrive: '08:05', override: true,
      remark: '邻居代接，已电话确认',
      persons: [{ name: '张敏', relation: '妈妈', phone: '137****5566' }]
    },
    {
      name: '朵朵', status: 'not_picked_up', arrive: '07:58',
      persons: []
    },
    {
      name: '果果', status: 'not_picked_up', arrive: '08:20', overdue: true, deadline: '17:30',
      persons: [
        { name: '刘芳', relation: '奶奶', phone: '136****1122' },
        { name: '刘明', relation: '爸爸', phone: '158****3344' }
      ]
    },
    {
      name: '豆豆', status: 'not_picked_up', arrive: '08:30',
      persons: [{ name: '陈静', relation: '妈妈', phone: '135****7788' }]
    },
    {
      name: '米米', status: 'not_picked_up', arrive: '08:41',
      persons: [{ name: '赵霞', relation: '妈妈', phone: '133****9900' }]
    },
    {
      name: '团团', status: 'picked_up', arrive: '08:02',
      persons: [{ name: '李梅', relation: '妈妈', phone: '138****6688' }]
    },
    {
      name: '安安', status: 'picked_up', arrive: '07:55',
      persons: [{ name: '王强', relation: '爸爸', phone: '139****2233' }]
    },
    {
      name: '妞妞', status: 'picked_up', arrive: '08:18',
      persons: [{ name: '刘芳', relation: '奶奶', phone: '136****1122' }]
    },
    {
      name: '天天', status: 'not_arrived', arrive: '',
      persons: [{ name: '孙丽', relation: '妈妈', phone: '132****4455' }]
    }
  ];

  // 初始快照：切换演示 chip 回到「正常」时重置，避免一次登记后演示无法重复
  var INITIAL = JSON.parse(JSON.stringify(STUDENTS));

  // 演示用的入园时间，固定值，避免整页数字随真实时钟漂移
  var DEMO_ARRIVE = '16:40';

  var STATUS_TEXT = { not_arrived: '未到园', not_picked_up: '已入园', picked_up: '已接走' };
  var FILTER_LABEL = { not_picked_up: '未接走', all: '全部', arrived: '已入园', picked_up: '已接走' };

  var app = document.querySelector('.mp-app');
  var listEl = document.getElementById('studentList');
  var listTitle = document.getElementById('listTitle');
  var filterEmpty = document.getElementById('filterEmpty');
  var overdueStrip = document.getElementById('overdueStrip');
  var overdueText = document.getElementById('overdueText');
  var toastEl = document.getElementById('toast');
  var toastTimer = null;

  /* ---------- 基础 ---------- */

  function toast(message) {
    toastEl.textContent = message;
    toastEl.classList.add('is-on');
    if (toastTimer) { clearTimeout(toastTimer); }
    toastTimer = setTimeout(function () { toastEl.classList.remove('is-on'); }, 2000);
  }

  function openLayer(el) {
    el.hidden = false;
    document.body.style.overflow = 'hidden';
  }

  function closeLayer(el) {
    el.hidden = true;
    if (!document.querySelector('.sheet-layer:not([hidden])')) {
      document.body.style.overflow = '';
    }
  }

  function closeAllLayers() {
    Array.prototype.forEach.call(document.querySelectorAll('.sheet-layer'), function (el) {
      el.hidden = true;
    });
    document.body.style.overflow = '';
    clearScanTimer();
  }

  function chip(state) {
    var target = document.querySelector('.mp-demo-chip[data-mp-state="' + state + '"]');
    if (target) { target.click(); }
    else { app.dataset.state = state; }
  }

  /* ---------- 学生卡片 ---------- */

  function pill(label, variant) {
    return '<span class="s-pill s-pill--' + variant + '">' + label + '</span>';
  }

  function actionsHtml(s) {
    if (s.status === 'picked_up') {
      return '<button class="a-btn a-btn--quiet" type="button" data-act="pickup" data-name="' + s.name + '">再接走</button>';
    }
    if (s.status === 'not_arrived') {
      // 后端规则：未登记入园不能直接接走，把约束前置成禁用态
      return '<button class="a-btn a-btn--success" type="button" data-act="dropoff" data-name="' + s.name + '">入园</button>' +
             '<button class="a-btn a-btn--off" type="button" aria-disabled="true" title="需先登记入园">接走</button>';
    }
    var out = '<button class="a-btn a-btn--primary" type="button" data-act="pickup" data-name="' + s.name + '">接走</button>';
    if (s.overdue) {
      out += '<button class="a-btn a-btn--danger" type="button" data-act="notify" data-name="' + s.name + '">通知</button>';
    }
    return out;
  }

  function cardHtml(s) {
    // 滞留只对「尚未接走」成立；已接走的学生不能同时挂「滞留」pill 和红边框
    var pendingOverdue = s.overdue && s.status === 'not_picked_up';
    var cls = 'student-card';
    if (pendingOverdue) { cls += ' student-card--overdue'; }
    if (s.status === 'picked_up') { cls += ' student-card--done'; }

    var pills = '';
    if (s.status === 'not_arrived') { pills += pill('未到园', 'wait'); }
    if (s.status === 'not_picked_up') { pills += pill('待接走', 'pending'); }
    if (s.status === 'picked_up') { pills += pill('已接走', 'done'); }
    if (pendingOverdue) { pills += pill('滞留', 'overdue'); }
    if (s.override) { pills += pill('例外', 'override'); }

    // 契约不返回接走时间/接走人，所有行统一展示授权接送人
    var auth = s.persons.length
      ? s.persons.map(function (p) { return p.name + '（' + p.relation + '）'; }).join('、')
      : '暂无';
    var lines = '<p class="s-auth">授权接送人：' + auth + '</p>';

    if (s.status === 'not_arrived') {
      lines += '<p class="s-meta">今日未登记入园</p>';
    } else if (s.status === 'not_picked_up') {
      lines += '<p class="s-meta">入园 ' + s.arrive + '</p>';
    }
    if (pendingOverdue) {
      lines += '<p class="s-alert">放学 ' + s.deadline + '，请尽快联系家长</p>';
    }
    if (s.remark) {
      lines += '<p class="s-remark">上次登记：' + s.remark + '</p>';
    }

    return '<article class="' + cls + '" data-status="' + s.status + '" data-name="' + s.name + '">' +
      '<span class="s-avatar" aria-hidden="true">' + s.name.charAt(0) + '</span>' +
      '<div class="s-main">' +
        '<div class="s-name-row"><h3 class="s-name">' + s.name + '</h3>' + pills + '</div>' +
        lines +
      '</div>' +
      '<div class="s-actions">' + actionsHtml(s) + '</div>' +
    '</article>';
  }

  function renderList() {
    listEl.innerHTML = STUDENTS.map(cardHtml).join('');
  }

  function countStatus(status) {
    return STUDENTS.filter(function (s) { return s.status === status; }).length;
  }

  /* 概览四项与滞留条都从同一份 STUDENTS 派生，登记后不会与列表打架。
   * 异常 = 当日最后一次接走走了授权例外通道的学生数。 */
  function renderSummary() {
    document.getElementById('statNotArrived').textContent = countStatus('not_arrived');
    document.getElementById('statPending').textContent = countStatus('not_picked_up');
    document.getElementById('statPicked').textContent = countStatus('picked_up');
    document.getElementById('statAbnormal').textContent =
      STUDENTS.filter(function (s) { return s.override; }).length;

    var overdueLeft = STUDENTS.filter(function (s) {
      return s.status === 'not_picked_up' && s.overdue;
    });
    overdueStrip.hidden = overdueLeft.length === 0;
    if (overdueLeft.length) {
      overdueText.textContent = overdueLeft.length + ' 名孩子已过放学时间 ' +
        (overdueLeft[0].deadline || '17:30') + ' 仍在园';
    }
  }

  /* 登记类动作的统一收口：改完 STUDENTS 后重画列表、筛选计数与概览 */
  function refreshView() {
    renderSummary();
    renderList();
    applyFilter(activeFilter);
  }

  /* ---------- 筛选 ---------- */

  var activeFilter = 'not_picked_up';

  function matches(s, filter) {
    if (filter === 'all') { return true; }
    if (filter === 'arrived') { return s.status === 'not_picked_up' || s.status === 'picked_up'; }
    return s.status === filter;
  }

  function applyFilter(filter) {
    activeFilter = filter;
    var shown = 0;
    Array.prototype.forEach.call(listEl.children, function (card) {
      var keep = matches(STUDENTS.filter(function (s) { return s.name === card.dataset.name; })[0], filter);
      card.hidden = !keep;
      if (keep) { shown += 1; }
    });
    listTitle.textContent = FILTER_LABEL[filter] + '学生（' + shown + '人）';
    filterEmpty.hidden = shown !== 0;
    Array.prototype.forEach.call(document.querySelectorAll('.filter-btn'), function (btn) {
      var on = btn.dataset.status === filter;
      btn.classList.toggle('is-active', on);
      btn.setAttribute('aria-pressed', String(on));
    });
  }

  Array.prototype.forEach.call(document.querySelectorAll('.filter-btn'), function (btn) {
    btn.addEventListener('click', function () { applyFilter(btn.dataset.status); });
  });

  /* ---------- 列表动作 ---------- */

  listEl.addEventListener('click', function (event) {
    var btn = event.target.closest('[data-act]');
    if (!btn) { return; }
    var name = btn.getAttribute('data-name');
    var student = STUDENTS.filter(function (s) { return s.name === name; })[0];
    if (!student) { return; }

    if (btn.getAttribute('aria-disabled') === 'true') { return; }

    var act = btn.getAttribute('data-act');
    if (act === 'pickup') {
      openManual(student, 'authorized');
      return;
    }
    if (act === 'dropoff') {
      // 源码在 action 后调用 loadPickup()：这里就地改状态，卡片、筛选计数与概览一起更新
      student.status = 'not_picked_up';
      if (!student.arrive) { student.arrive = DEMO_ARRIVE; }
      refreshView();
      toast(student.name + ' 已入园');
      return;
    }
    if (act === 'notify') {
      btn.textContent = '已通知';
      btn.className = 'a-btn a-btn--notified';
      btn.setAttribute('aria-disabled', 'true');
      toast('已通知 ' + student.name + ' 家长');
    }
  });

  /* ---------- 选择学生 ---------- */

  var pickerSheet = document.getElementById('studentSheet');
  var pickerEl = document.getElementById('studentPicker');

  function renderPicker() {
    pickerEl.innerHTML = STUDENTS.map(function (s) {
      // 与列表「未到园不能接走」的规则保持一致：未入园的行标注原因并不可选
      var off = s.status === 'not_arrived';
      return '<button class="picker-row' + (off ? ' picker-row--off' : '') + '" type="button" data-name="' + s.name + '"' +
          (off ? ' aria-disabled="true"' : '') + '>' +
        '<span class="picker-name">' + s.name + '</span>' +
        '<span class="picker-status">' +
          (off ? '需先入园' : STATUS_TEXT[s.status] + (s.overdue ? ' · 滞留' : '')) +
        '</span>' +
      '</button>';
    }).join('');
  }

  pickerEl.addEventListener('click', function (event) {
    var row = event.target.closest('.picker-row');
    if (!row) { return; }
    if (row.getAttribute('aria-disabled') === 'true') {
      toast(row.dataset.name + ' 今日尚未入园，请先登记入园');
      return;
    }
    var student = STUDENTS.filter(function (s) { return s.name === row.dataset.name; })[0];
    closeLayer(pickerSheet);
    openManual(student, 'authorized');
  });

  document.getElementById('manualEntry').addEventListener('click', function () {
    renderPicker();
    openLayer(pickerSheet);
  });

  /* ---------- 手动登记 ---------- */

  var manualSheet = document.getElementById('manualSheet');
  var manualTitle = document.getElementById('manualTitle');
  var manualSub = document.getElementById('manualSub');
  var personList = document.getElementById('personList');
  var noAuthorized = document.getElementById('noAuthorized');
  var overridePanel = document.getElementById('overridePanel');
  var authorizedPanel = document.getElementById('authorizedPanel');
  var overrideName = document.getElementById('overrideName');
  var reasonInput = document.getElementById('reasonInput');
  var confirmBtn = document.getElementById('confirmManual');

  var manualStudent = null;
  var manualMode = 'authorized';
  var selectedPid = null;

  function personHtml(p, index) {
    return '<button class="person-card" type="button" data-pid="' + index + '">' +
      '<span class="person-radio" aria-hidden="true"></span>' +
      '<span class="person-avatar" aria-hidden="true">' + p.name.charAt(0) + '</span>' +
      '<span class="person-info">' +
        '<span class="person-name">' + p.name + '　' + p.relation + '</span>' +
        '<span class="person-relation">' + p.phone + '</span>' +
      '</span>' +
    '</button>';
  }

  function renderPersons(student) {
    selectedPid = null;
    if (!student.persons.length) {
      personList.innerHTML = '';
      noAuthorized.hidden = false;
      return;
    }
    noAuthorized.hidden = true;
    personList.innerHTML = student.persons.map(personHtml).join('');
  }

  function syncModePanels() {
    authorizedPanel.hidden = manualMode !== 'authorized';
    overridePanel.hidden = manualMode !== 'override';
    Array.prototype.forEach.call(document.querySelectorAll('.mode-btn'), function (btn) {
      var on = btn.dataset.mode === manualMode;
      btn.classList.toggle('is-active', on);
      btn.setAttribute('aria-selected', String(on));
    });
  }

  function syncConfirm() {
    var ok = manualMode === 'authorized'
      ? selectedPid !== null
      : overrideName.value.trim() !== '';
    confirmBtn.setAttribute('aria-disabled', ok ? 'false' : 'true');
  }

  function openManual(student, mode) {
    manualStudent = student;
    manualMode = mode || 'authorized';
    manualTitle.textContent = '手动登记 · ' + student.name;
    manualSub.textContent = CLASS_NAME + ' · ' + STATUS_TEXT[student.status];
    overrideName.value = '';
    reasonInput.value = '';
    renderPersons(student);
    syncModePanels();
    syncConfirm();
    openLayer(manualSheet);
  }

  personList.addEventListener('click', function (event) {
    var card = event.target.closest('.person-card');
    if (!card) { return; }
    selectedPid = Number(card.dataset.pid);
    Array.prototype.forEach.call(personList.children, function (el) {
      var on = el === card;
      el.classList.toggle('is-selected', on);
      var radio = el.querySelector('.person-radio');
      if (radio) {
        radio.innerHTML = on ? '<span class="person-dot"></span>' : '';
      }
    });
    syncConfirm();
  });

  Array.prototype.forEach.call(document.querySelectorAll('.mode-btn'), function (btn) {
    btn.addEventListener('click', function () {
      manualMode = btn.dataset.mode;
      selectedPid = null;
      overrideName.value = '';
      syncModePanels();
      Array.prototype.forEach.call(personList.children, function (el) {
        el.classList.remove('is-selected');
        var radio = el.querySelector('.person-radio');
        if (radio) { radio.innerHTML = ''; }
      });
      syncConfirm();
    });
  });

  overrideName.addEventListener('input', syncConfirm);

  confirmBtn.addEventListener('click', function () {
    if (confirmBtn.getAttribute('aria-disabled') === 'true') {
      toast(manualMode === 'authorized' ? '请先选择一位授权接送人' : '请填写例外接送人姓名');
      return;
    }
    var reason = reasonInput.value.trim();
    // 契约：重复接走（当日第 2+ 次）未填理由时后端拒绝，源码 mapPickupError 映射成同一条提示
    if (manualStudent.status === 'picked_up' && !reason) {
      toast('重复接走需填写理由');
      return;
    }
    var isOverride = manualMode === 'override';
    // 源码成功后 loadPickup()：这里就地落库到演示数据（is_authorized_override / last_remark 同口径）
    manualStudent.status = 'picked_up';
    manualStudent.override = isOverride;
    if (reason) { manualStudent.remark = reason; }
    closeLayer(manualSheet);
    refreshView();
    toast(isOverride
      ? '已按例外登记 · ' + manualStudent.name
      : manualStudent.name + ' 登记成功');
  });

  /* ---------- 扫码核验 ---------- */

  var scanSheet = document.getElementById('scanSheet');
  var scanLoading = document.getElementById('scanLoading');
  var scanFailure = document.getElementById('scanFailure');
  var scanSuccess = document.getElementById('scanSuccess');
  var scanTimer = null;

  function clearScanTimer() {
    if (scanTimer) { clearTimeout(scanTimer); scanTimer = null; }
  }

  function scanPhase(phase) {
    scanLoading.hidden = phase !== 'loading';
    scanFailure.hidden = phase !== 'failure';
    scanSuccess.hidden = phase !== 'success';
  }

  function openScan() {
    scanPhase('loading');
    openLayer(scanSheet);
    clearScanTimer();
    scanTimer = setTimeout(function () { scanPhase('failure'); }, 700);
  }

  document.getElementById('scanEntry').addEventListener('click', openScan);

  document.getElementById('rescanBtn').addEventListener('click', function () {
    scanPhase('loading');
    clearScanTimer();
    scanTimer = setTimeout(function () { scanPhase('success'); }, 500);
  });

  document.getElementById('scanOverrideBtn').addEventListener('click', function () {
    var target = STUDENTS.filter(function (s) { return s.name === '朵朵'; })[0];
    closeLayer(scanSheet);
    openManual(target, 'override');
  });

  document.getElementById('scanDoneBtn').addEventListener('click', function () {
    var target = STUDENTS.filter(function (s) { return s.name === '乐乐'; })[0];
    closeLayer(scanSheet);
    // 扫码响应只有 student / person / type_label，接走人仍在授权集合内 → 非例外
    if (target && target.status !== 'picked_up') {
      target.status = 'picked_up';
      target.override = false;
      refreshView();
    }
    toast('乐乐 已接走');
  });

  /* ---------- 其它入口 ---------- */

  document.getElementById('recordsEntry').addEventListener('click', function () {
    toast('已打开 接送记录');
  });
  document.getElementById('emptyRecordsBtn').addEventListener('click', function () {
    toast('已打开 接送记录');
  });
  document.getElementById('backBtn').addEventListener('click', function () {
    toast('返回老师工作台');
  });

  document.getElementById('retryBtn').addEventListener('click', function () {
    chip('normal');
    toast('接送数据已更新');
  });

  /* ---------- 弹层关闭 ---------- */

  Array.prototype.forEach.call(document.querySelectorAll('[data-close]'), function (btn) {
    btn.addEventListener('click', function () {
      var layer = document.getElementById(btn.getAttribute('data-close'));
      if (layer) { closeLayer(layer); }
    });
  });

  document.addEventListener('keydown', function (event) {
    if (event.key !== 'Escape') { return; }
    var open = document.querySelector('.sheet-layer:not([hidden])');
    if (open) { closeLayer(open); }
  });

  /* 切到 空/加载/失败 时收起弹层，并回滚临时态（单一状态出口是演示 chip）；
   * 回到「正常」时把演示数据重置回初始快照，保证登记演示可以重复。 */
  document.addEventListener('mp:state', function () {
    closeAllLayers();
    Array.prototype.forEach.call(document.querySelectorAll('.a-btn--notified'), function (btn) {
      btn.className = 'a-btn a-btn--danger';
      btn.textContent = '通知';
      btn.removeAttribute('aria-disabled');
    });
    if (app.dataset.state === 'normal') {
      STUDENTS = JSON.parse(JSON.stringify(INITIAL));
      refreshView();
    }
  });

  /* ---------- 初始渲染 ---------- */

  renderSummary();
  renderList();
  applyFilter('not_picked_up');
})();
