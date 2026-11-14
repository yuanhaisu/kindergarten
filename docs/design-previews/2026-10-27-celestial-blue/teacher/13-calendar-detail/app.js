/* 活动详情（老师端）— 原型交互。
 * 状态由共享演示条写入 .mp-app[data-state]，本文件负责内容、弹层与轻提示。
 */
(function () {
  'use strict';

  var app = document.querySelector('.mp-app');
  if (!app) return;

  function byId(id) { return document.getElementById(id); }

  var ui = {
    typeBadge: byId('typeBadge'),
    typeMono: byId('typeMono'),
    typeLabel: byId('typeLabel'),
    statusBadge: byId('statusBadge'),
    statusText: byId('statusText'),
    title: byId('eventTitle'),
    draftNote: byId('draftNote'),
    desc: byId('eventDesc'),
    descEmpty: byId('descEmpty'),
    scopeNote: byId('scopeNote'),
    infoDate: byId('infoDate'),
    infoTime: byId('infoTime'),
    infoLocation: byId('infoLocation'),
    infoScope: byId('infoScope'),
    actions: byId('bottomActions'),
    previewBtn: byId('previewBtn'),
    editBtn: byId('editBtn'),
    deleteBtn: byId('deleteBtn'),
    retryBtn: byId('retryBtn'),
    descAddBtn: byId('descAddBtn'),
    sheetLayer: byId('sheetLayer'),
    sheetMask: byId('sheetMask'),
    sheetClose: byId('sheetClose'),
    sheetDraft: byId('sheetDraft'),
    parentTitle: byId('parentTitle'),
    parentType: byId('parentType'),
    parentMeta: byId('parentMeta'),
    parentDesc: byId('parentDesc'),
    dialogLayer: byId('dialogLayer'),
    dialogMask: byId('dialogMask'),
    dialogCancel: byId('dialogCancel'),
    dialogConfirm: byId('dialogConfirm'),
    toast: byId('toast')
  };

  /* ---- 演示数据（真实字段，不添加源码没有的信息） ---------------------- */
  var BASE_EVENT = {
    typeName: '活动',
    typeChar: '活',
    tone: 'success',
    status: 'published',
    scope: 'class',
    title: '秋季运动会彩排',
    date: '2026-10-26',
    weekday: '周一',
    allDay: false,
    start: '09:00',
    end: '10:30',
    location: '大操场',
    description: '各班按方阵入场，练习接力与亲子项目走位。请提前十分钟带队到操场集合，孩子穿运动鞋、深色长裤，水杯写上名字。\n\n如遇下雨，彩排改到多功能厅，按年级分批进行，具体时间另行通知。'
  };

  var EVENTS = {
    normal: BASE_EVENT,
    draft: Object.assign({}, BASE_EVENT, { status: 'draft' }),
    empty: {
      typeName: '体检',
      typeChar: '体',
      tone: 'danger',
      status: 'published',
      scope: 'class',
      title: '视力筛查',
      date: '2026-10-26',
      weekday: '周一',
      allDay: true,
      start: '',
      end: '',
      location: '保健室',
      description: ''
    },
    kindergarten: {
      typeName: '活动',
      typeChar: '活',
      tone: 'success',
      status: 'published',
      scope: 'kindergarten',
      title: '秋游',
      date: '2026-10-26',
      weekday: '周一',
      allDay: true,
      start: '',
      end: '',
      location: '城市植物园',
      description: '全园秋游日，请为孩子准备轻便背包、水壶与遮阳帽。集合与返园时间以班级通知为准。'
    }
  };

  var DATA_STATES = ['normal', 'empty', 'draft', 'kindergarten'];

  /* ---- 文本格式化 ------------------------------------------------------ */
  function formatDate(event) {
    var parts = String(event.date).split('-');
    if (parts.length !== 3) return event.date + ' ' + event.weekday;
    return Number(parts[0]) + '年' + Number(parts[1]) + '月' + Number(parts[2]) + '日 ' + event.weekday;
  }

  function formatTime(event, compact) {
    var joiner = compact ? '-' : ' - ';
    if (event.allDay) return '全天';
    if (event.start && event.end) return event.start + joiner + event.end;
    return event.start || event.end || '未设置';
  }

  function splitParagraphs(text) {
    if (!text) return [];
    return String(text).split(/\n\s*\n/).map(function (part) {
      return part.trim();
    }).filter(Boolean);
  }

  /* ---- 渲染活动内容 ---------------------------------------------------- */
  function render(event) {
    ui.typeBadge.dataset.tone = event.tone;
    ui.typeMono.textContent = event.typeChar;
    ui.typeLabel.textContent = event.typeName;

    ui.statusBadge.dataset.status = event.status;
    ui.statusText.textContent = event.status === 'draft' ? '草稿（家长不可见）' : '已发布';

    ui.title.textContent = event.title;
    ui.draftNote.hidden = event.status !== 'draft';

    var paragraphs = splitParagraphs(event.description);
    ui.desc.textContent = '';
    paragraphs.forEach(function (text) {
      var p = document.createElement('p');
      p.textContent = text;
      ui.desc.appendChild(p);
    });
    ui.desc.hidden = paragraphs.length === 0;
    ui.descEmpty.hidden = paragraphs.length !== 0;

    ui.scopeNote.hidden = event.scope !== 'kindergarten';

    ui.infoDate.textContent = formatDate(event);
    ui.infoTime.textContent = formatTime(event, false);
    ui.infoLocation.textContent = event.location || '未设置';
    ui.infoScope.textContent = event.scope === 'kindergarten' ? '全园' : '本班';

    ui.actions.hidden = event.scope === 'kindergarten';

    ui.sheetDraft.hidden = event.status !== 'draft';
    ui.parentTitle.textContent = event.title;
    ui.parentType.textContent = event.typeName;
    ui.parentType.dataset.tone = event.tone;

    var meta = [event.date, formatTime(event, true), event.location].filter(Boolean);
    ui.parentMeta.textContent = meta.join(' · ');

    var parentText = paragraphs.join(' ');
    ui.parentDesc.textContent = parentText;
    ui.parentDesc.hidden = parentText.length === 0;
  }

  /* ---- 轻提示 ---------------------------------------------------------- */
  var toastTimer = null;
  function toast(message) {
    ui.toast.textContent = message;
    ui.toast.classList.add('is-on');
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(function () {
      ui.toast.classList.remove('is-on');
      toastTimer = null;
    }, 2200);
  }

  /* ---- 弹层 ------------------------------------------------------------ */
  var reduceMotion = !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
  var SHEET_CLOSE_MS = reduceMotion ? 0 : 200;
  var DIALOG_CLOSE_MS = reduceMotion ? 0 : 160;

  var sheetTimer = null;
  var dialogTimer = null;
  var sheetReturnFocus = null;
  var dialogReturnFocus = null;

  function lockBody() {
    var open = ui.sheetLayer.classList.contains('is-open') ||
      ui.dialogLayer.classList.contains('is-open');
    document.body.classList.toggle('is-modal-open', open);
  }

  function focusSafely(element) {
    if (!element || typeof element.focus !== 'function') return;
    try { element.focus({ preventScroll: true }); } catch (error) { /* noop */ }
  }

  function openSheet() {
    sheetReturnFocus = document.activeElement;
    if (sheetTimer) { clearTimeout(sheetTimer); sheetTimer = null; }
    ui.sheetLayer.classList.remove('is-closing');
    ui.sheetLayer.classList.add('is-open');
    lockBody();
    focusSafely(ui.sheetClose);
  }

  function closeSheet(immediate) {
    if (!ui.sheetLayer.classList.contains('is-open')) return;
    if (sheetTimer) { clearTimeout(sheetTimer); sheetTimer = null; }

    if (immediate) {
      ui.sheetLayer.classList.remove('is-open', 'is-closing');
      lockBody();
      return;
    }

    ui.sheetLayer.classList.add('is-closing');
    sheetTimer = setTimeout(function () {
      ui.sheetLayer.classList.remove('is-open', 'is-closing');
      sheetTimer = null;
      lockBody();
      focusSafely(sheetReturnFocus);
    }, SHEET_CLOSE_MS);
  }

  function openDialog() {
    dialogReturnFocus = document.activeElement;
    if (dialogTimer) { clearTimeout(dialogTimer); dialogTimer = null; }
    ui.dialogLayer.classList.remove('is-closing');
    ui.dialogLayer.classList.add('is-open');
    lockBody();
    focusSafely(ui.dialogCancel);
  }

  function closeDialog(immediate) {
    if (!ui.dialogLayer.classList.contains('is-open')) return;
    if (dialogTimer) { clearTimeout(dialogTimer); dialogTimer = null; }

    if (immediate) {
      ui.dialogLayer.classList.remove('is-open', 'is-closing');
      lockBody();
      return;
    }

    ui.dialogLayer.classList.add('is-closing');
    dialogTimer = setTimeout(function () {
      ui.dialogLayer.classList.remove('is-open', 'is-closing');
      dialogTimer = null;
      lockBody();
      focusSafely(dialogReturnFocus);
    }, DIALOG_CLOSE_MS);
  }

  function closeOverlaysNow() {
    if (sheetTimer) { clearTimeout(sheetTimer); sheetTimer = null; }
    if (dialogTimer) { clearTimeout(dialogTimer); dialogTimer = null; }
    ui.sheetLayer.classList.remove('is-open', 'is-closing');
    ui.dialogLayer.classList.remove('is-open', 'is-closing');
    lockBody();
  }

  /* ---- 状态切换 -------------------------------------------------------- */
  function setState(state) {
    closeOverlaysNow();
    if (DATA_STATES.indexOf(state) !== -1) render(EVENTS[state]);
  }

  function activateState(state) {
    app.dataset.state = state;
    document.querySelectorAll('.mp-demo-chip').forEach(function (chip) {
      chip.setAttribute('aria-pressed', String(chip.getAttribute('data-mp-state') === state));
    });
    setState(state);
  }

  document.addEventListener('mp:state', function (event) {
    var state = event.detail && event.detail.state;
    if (state) setState(state);
  });

  /* ---- 绑定交互 -------------------------------------------------------- */
  ui.previewBtn.addEventListener('click', openSheet);
  ui.sheetClose.addEventListener('click', function () { closeSheet(false); });
  ui.sheetMask.addEventListener('click', function () { closeSheet(false); });

  ui.editBtn.addEventListener('click', function () { toast('打开编辑活动'); });
  ui.descAddBtn.addEventListener('click', function () { toast('打开编辑活动'); });

  ui.deleteBtn.addEventListener('click', openDialog);
  ui.dialogCancel.addEventListener('click', function () { closeDialog(false); });
  ui.dialogMask.addEventListener('click', function () { closeDialog(false); });
  ui.dialogConfirm.addEventListener('click', function () {
    closeDialog(false);
    toast('已删除，即将返回班级日历');
  });

  ui.retryBtn.addEventListener('click', function () {
    activateState('normal');
    toast('已重新加载');
  });

  var backBtn = document.querySelector('.mp-back');
  if (backBtn) {
    backBtn.addEventListener('click', function () { toast('返回班级日历'); });
  }

  document.addEventListener('keydown', function (event) {
    if (event.key !== 'Escape') return;
    if (ui.dialogLayer.classList.contains('is-open')) {
      closeDialog(false);
    } else if (ui.sheetLayer.classList.contains('is-open')) {
      closeSheet(false);
    }
  });

  /* ---- 初始化 ---------------------------------------------------------- */
  setState(app.dataset.state || 'normal');
})();
