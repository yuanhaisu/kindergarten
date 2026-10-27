(function () {
  'use strict';

  var TODAY = '2026-10-26';
  var TYPE_LABELS = {
    checkup: '体检',
    notice: '通知',
    activity: '活动',
    holiday: '假期',
    meeting: '家长会'
  };
  var WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
  var ESCAPE_MAP = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };

  var proto = document.getElementById('proto');
  var agendaGroups = document.getElementById('agendaGroups');
  var countValue = document.getElementById('countValue');
  var eventSheet = document.getElementById('eventSheet');
  var sheetOverlay = document.getElementById('sheetOverlay');
  var sheetTitle = document.getElementById('sheetTitle');
  var closeSheetBtn = document.getElementById('closeSheetBtn');
  var editNote = document.getElementById('editNote');
  var editNoteText = document.getElementById('editNoteText');
  var eventForm = document.getElementById('eventForm');
  var eventTitle = document.getElementById('eventTitle');
  var eventDescription = document.getElementById('eventDescription');
  var eventLocation = document.getElementById('eventLocation');
  var typeSwitch = document.getElementById('typeSwitch');
  var typeButtons = Array.prototype.slice.call(typeSwitch.querySelectorAll('.type-option'));
  var eventDateField = document.getElementById('eventDateField');
  var eventDateValue = document.getElementById('eventDateValue');
  var startTimeField = document.getElementById('startTimeField');
  var startTimeValue = document.getElementById('startTimeValue');
  var endTimeField = document.getElementById('endTimeField');
  var endTimeValue = document.getElementById('endTimeValue');
  var notifyRow = document.getElementById('notifyRow');
  var notifySwitch = document.getElementById('notifySwitch');
  var saveDraftBtn = document.getElementById('saveDraftBtn');
  var publishNowBtn = document.getElementById('publishNowBtn');
  var cancelEditBtn = document.getElementById('cancelEditBtn');
  var saveEditBtn = document.getElementById('saveEditBtn');
  var sheetFooter = eventSheet.querySelector('.sheet-footer');
  var confirmSheet = document.getElementById('confirmSheet');
  var confirmOverlay = document.getElementById('confirmOverlay');
  var changeList = document.getElementById('changeList');
  var changeNotifySwitch = document.getElementById('changeNotifySwitch');
  var cancelConfirmBtn = document.getElementById('cancelConfirmBtn');
  var saveConfirmBtn = document.getElementById('saveConfirmBtn');
  var textDialog = document.getElementById('textDialog');
  var textDialogOverlay = document.getElementById('textDialogOverlay');
  var textDialogTitle = document.getElementById('textDialogTitle');
  var textDialogHint = document.getElementById('textDialogHint');
  var textDialogInput = document.getElementById('textDialogInput');
  var textDialogCancel = document.getElementById('textDialogCancel');
  var textDialogConfirm = document.getElementById('textDialogConfirm');
  var deleteDialog = document.getElementById('deleteDialog');
  var deleteOverlay = document.getElementById('deleteOverlay');
  var deleteBody = document.getElementById('deleteBody');
  var cancelDeleteBtn = document.getElementById('cancelDeleteBtn');
  var confirmDeleteBtn = document.getElementById('confirmDeleteBtn');
  var toast = document.getElementById('toast');
  var openPublishBtn = document.getElementById('openPublishBtn');
  var emptyPublishBtn = document.getElementById('emptyPublishBtn');
  var reloadEventsBtn = document.getElementById('reloadEventsBtn');

  var events = [
    {
      id: 'evt-checkup-all',
      title: '全园体检',
      event_type: 'checkup',
      description: '身高体重、视力与口腔检查；请家长当天给孩子穿宽松衣物，早餐照常。',
      event_date: '2026-10-26',
      start_time: '09:00',
      end_time: '11:00',
      location: '医务室',
      is_all_day: false,
      notify: true,
      status: 'published'
    },
    {
      id: 'evt-makeup-checkup',
      title: '小班补检',
      event_type: 'checkup',
      description: '上周请假未检的幼儿补检，班主任陪同前往。',
      event_date: '2026-10-26',
      start_time: '14:30',
      end_time: '',
      location: '医务室',
      is_all_day: false,
      notify: true,
      status: 'draft'
    },
    {
      id: 'evt-vision-retest',
      title: '视力筛查复查',
      event_type: 'checkup',
      description: '初筛结果异常的幼儿复查，请家长留意班级通知。',
      event_date: '2026-10-29',
      start_time: '09:30',
      end_time: '10:30',
      location: '保健室',
      is_all_day: false,
      notify: true,
      status: 'published'
    },
    {
      id: 'evt-health-talk',
      title: '秋冬季健康讲座',
      event_type: 'activity',
      description: '面向全园家长的流感预防与家庭护理讲座，可线上同步观看。',
      event_date: '2026-11-03',
      start_time: '15:00',
      end_time: '16:00',
      location: '多功能厅',
      is_all_day: false,
      notify: true,
      status: 'published'
    },
    {
      id: 'evt-dental-check',
      title: '口腔检查',
      event_type: 'checkup',
      description: '检查已完成，结果已反馈给各班老师。',
      event_date: '2026-10-20',
      start_time: '10:00',
      end_time: '11:30',
      location: '医务室',
      is_all_day: false,
      notify: true,
      status: 'published'
    }
  ];

  var selectedDate = null;
  var collapsedDates = new Set();
  var sheetMode = null;
  var editingId = null;
  var editingBaseline = null;
  var pendingEdit = null;
  var deleteTargetId = null;
  var textDialogField = null;
  var submitting = false;
  var toastTimer = 0;
  var toastHideTimer = 0;
  var focusMemory = new WeakMap();

  var formState = {
    event_type: 'checkup',
    event_date: '',
    start_time: '09:00',
    end_time: ''
  };

  function escapeHtml(value) {
    return String(value == null ? '' : value).replace(/[&<>"']/g, function (character) {
      return ESCAPE_MAP[character];
    });
  }

  function dateInfo(dateString) {
    var parts = String(dateString || '').split('-').map(Number);
    var year = parts[0];
    var month = parts[1];
    var day = parts[2];
    var parsed = new Date(year, month - 1, day, 12, 0, 0);
    return {
      year: year,
      month: month,
      day: day,
      weekday: WEEKDAYS[parsed.getDay()],
      monthLabel: year + '年' + month + '月',
      dateLabel: month + '月' + String(day).padStart(2, '0') + '日 ' + WEEKDAYS[parsed.getDay()]
    };
  }

  function eventTimeLabel(event) {
    if (event.is_all_day) return '全天';
    if (!event.start_time) return '未设';
    return event.end_time ? event.start_time + '–' + event.end_time : event.start_time;
  }

  function findEvent(id) {
    return events.find(function (event) { return event.id === id; });
  }

  function updateCount() {
    if (countValue) countValue.textContent = String(events.length);
  }

  function sortedEventsForDate(date) {
    return events.filter(function (event) {
      return event.event_date === date;
    }).sort(function (a, b) {
      if (a.is_all_day !== b.is_all_day) return a.is_all_day ? -1 : 1;
      return String(a.start_time || '').localeCompare(String(b.start_time || ''));
    });
  }

  function eventCardMarkup(event) {
    var id = escapeHtml(event.id);
    var title = escapeHtml(event.title);
    var typeLabel = escapeHtml(TYPE_LABELS[event.event_type] || '通知');
    var time = escapeHtml(eventTimeLabel(event));
    var description = event.description
      ? '<span class="event-desc">' + escapeHtml(event.description) + '</span>'
      : '';
    var place = event.location
      ? '<span class="event-place">' +
          '<svg class="place-icon" viewBox="0 0 20 20" fill="none" aria-hidden="true">' +
            '<path d="M10 17.25s5.25-4.5 5.25-9.1a5.25 5.25 0 1 0-10.5 0c0 4.6 5.25 9.1 5.25 9.1Z" stroke="currentColor" stroke-width="1.5" />' +
            '<circle cx="10" cy="8" r="1.75" stroke="currentColor" stroke-width="1.5" />' +
          '</svg>' +
          '<span>' + escapeHtml(event.location) + '</span>' +
        '</span>'
      : '';
    var draftTag = event.status === 'draft' ? '<span class="draft-tag">草稿</span>' : '';

    return '<article class="event-card' + (event.status === 'draft' ? ' is-draft' : '') + '" data-od-id="event-card-' + id + '">' +
      '<button class="event-open" type="button" data-action="open-event" data-id="' + id + '" aria-label="打开 ' + title + '">' +
        '<span class="event-top">' +
          '<span class="type-tag">' + typeLabel + '</span>' +
          draftTag +
          '<span class="event-time">' + time + '</span>' +
        '</span>' +
        '<span class="event-title">' + title + '</span>' +
        description +
        '<span class="event-meta">' +
          place +
          '<span class="scope-tag">全园</span>' +
        '</span>' +
      '</button>' +
      '<div class="event-actions">' +
        '<button class="event-action" type="button" data-action="edit-event" data-id="' + id + '" aria-label="编辑' + title + '">编辑</button>' +
        '<button class="event-action danger" type="button" data-action="delete-event" data-id="' + id + '" aria-label="删除' + title + '">删除</button>' +
      '</div>' +
    '</article>';
  }

  function dayGroupMarkup(date, eventList, isPast) {
    var info = dateInfo(date);
    var isToday = date === TODAY;
    var isSelected = selectedDate === date;
    var isCollapsed = collapsedDates.has(date);
    var groupClasses = ['agenda-group'];
    if (isToday) groupClasses.push('is-today');
    if (isSelected) groupClasses.push('is-selected');
    if (isCollapsed) groupClasses.push('is-collapsed');
    if (isPast) groupClasses.push('is-past');

    var label = isToday ? '今天 · ' + info.weekday : info.dateLabel;
    var eventsId = 'day-events-' + date.replace(/-/g, '');
    var cards = eventList.map(eventCardMarkup).join('');

    return '<section class="' + groupClasses.join(' ') + '" data-date="' + escapeHtml(date) + '" data-od-id="day-group-' + escapeHtml(date) + '">' +
      '<div class="month-row"><span class="month-tag">' + escapeHtml(info.monthLabel) + '</span></div>' +
      '<div class="day-head">' +
        '<button class="day-select" type="button" data-action="toggle-day" data-date="' + escapeHtml(date) + '" aria-pressed="' + String(isSelected) + '">' +
          '<span class="date-badge">' + info.day + '</span>' +
          '<span class="day-copy">' +
            '<span class="day-title">' + escapeHtml(label) + '</span>' +
            '<span class="day-sub">' + eventList.length + ' 场</span>' +
          '</span>' +
        '</button>' +
        '<button class="day-toggle" type="button" data-action="toggle-collapse" data-date="' + escapeHtml(date) + '" aria-expanded="' + String(!isCollapsed) + '" aria-controls="' + eventsId + '" aria-label="' + (isCollapsed ? '展开' : '收起') + escapeHtml(info.dateLabel) + '的事件">' +
          '<svg class="day-chevron" viewBox="0 0 20 20" fill="none" aria-hidden="true">' +
            '<path d="M5.5 8 10 12.5 14.5 8" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" />' +
          '</svg>' +
        '</button>' +
      '</div>' +
      '<div class="day-events" id="' + eventsId + '">' + cards + '</div>' +
    '</section>';
  }

  function renderGroups() {
    var dates = Array.from(new Set(events.map(function (event) { return event.event_date; }).filter(Boolean)));
    var upcoming = dates.filter(function (date) { return date >= TODAY; }).sort();
    var past = dates.filter(function (date) { return date < TODAY; }).sort(function (a, b) { return b.localeCompare(a); });
    var html = '';

    if (selectedDate && dates.indexOf(selectedDate) === -1) selectedDate = null;

    upcoming.forEach(function (date) {
      html += dayGroupMarkup(date, sortedEventsForDate(date), false);
    });

    if (past.length) {
      html += '<div class="past-divider" role="separator"><span>已结束</span></div>';
      past.forEach(function (date) {
        html += dayGroupMarkup(date, sortedEventsForDate(date), true);
      });
    }

    agendaGroups.innerHTML = html;
    updateCount();
  }

  function showToast(message) {
    window.clearTimeout(toastTimer);
    window.clearTimeout(toastHideTimer);
    toast.textContent = message;
    toast.hidden = false;
    toast.classList.remove('is-visible');
    void toast.offsetWidth;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(function () {
      toast.classList.remove('is-visible');
      toastHideTimer = window.setTimeout(function () {
        toast.hidden = true;
      }, 180);
    }, 2000);
  }

  function isLayerVisible(layer) {
    return !!layer && !layer.hidden;
  }

  function refreshBodyLock() {
    var openLayer = document.querySelector('.overlay:not([hidden]), .modal-overlay:not([hidden])');
    document.body.classList.toggle('layer-open', !!openLayer);
  }

  function openLayer(layer, overlay, focusTarget) {
    focusMemory.set(layer, document.activeElement);
    overlay.hidden = false;
    layer.hidden = false;
    document.body.classList.add('layer-open');
    var target = focusTarget || layer;
    window.requestAnimationFrame(function () {
      if (!layer.hidden && target && typeof target.focus === 'function') {
        target.focus({ preventScroll: true });
      }
    });
  }

  function closeLayer(layer, overlay) {
    layer.hidden = true;
    overlay.hidden = true;
    refreshBodyLock();
    var target = focusMemory.get(layer);
    focusMemory.delete(layer);
    if (target && typeof target.focus === 'function' && document.contains(target)) {
      try { target.focus({ preventScroll: true }); } catch (error) { /* no-op */ }
    }
  }

  function getOpenLayer() {
    var layers = [eventSheet, confirmSheet, textDialog, deleteDialog];
    for (var index = layers.length - 1; index >= 0; index -= 1) {
      if (isLayerVisible(layers[index])) return layers[index];
    }
    return null;
  }

  function getFocusable(layer) {
    var nodes = layer.querySelectorAll('button:not([disabled]), input:not([disabled]), textarea:not([disabled]), select:not([disabled]), a[href], [tabindex]:not([tabindex="-1"])');
    return Array.prototype.filter.call(nodes, function (node) {
      return !node.hidden && (node.offsetWidth > 0 || node.offsetHeight > 0 || node.getClientRects().length > 0);
    });
  }

  function setSwitch(button, isOn) {
    button.setAttribute('aria-checked', isOn ? 'true' : 'false');
  }

  function isSwitchOn(button) {
    return button.getAttribute('aria-checked') === 'true';
  }

  function setType(type) {
    var nextType = TYPE_LABELS[type] ? type : 'checkup';
    formState.event_type = nextType;
    typeButtons.forEach(function (button) {
      var active = button.dataset.type === nextType;
      button.classList.toggle('is-selected', active);
      button.setAttribute('aria-checked', String(active));
      button.tabIndex = active ? 0 : -1;
    });
  }

  function renderDateValue() {
    var hasDate = !!formState.event_date;
    eventDateValue.textContent = hasDate ? formState.event_date : '请选择日期';
    eventDateValue.classList.toggle('is-placeholder', !hasDate);
  }

  function renderTimeValues() {
    startTimeValue.textContent = formState.start_time || '09:00';
    startTimeValue.classList.toggle('is-placeholder', !formState.start_time);
    endTimeValue.textContent = formState.end_time || '未设';
    endTimeValue.classList.toggle('is-placeholder', !formState.end_time);
  }

  function resetSubmitButtons() {
    [saveDraftBtn, publishNowBtn].forEach(function (button) {
      button.disabled = false;
      button.removeAttribute('aria-disabled');
    });
    saveDraftBtn.textContent = '存为草稿';
    publishNowBtn.textContent = '立即发布';
    eventForm.removeAttribute('aria-busy');
    submitting = false;
  }

  function setFooterMode(mode) {
    var isEdit = mode === 'edit';
    // 编辑态保留「存为草稿」，与源码一致：发布过的事件也能退回草稿。
    saveDraftBtn.hidden = false;
    publishNowBtn.hidden = isEdit;
    cancelEditBtn.hidden = !isEdit;
    saveEditBtn.hidden = !isEdit;
    sheetFooter.classList.toggle('is-edit', isEdit);
  }

  function openCreateSheet() {
    if (submitting) return;
    sheetMode = 'create';
    editingId = null;
    editingBaseline = null;
    sheetTitle.textContent = '发布园级事件';
    eventForm.reset();
    eventTitle.value = '';
    eventDescription.value = '';
    eventLocation.value = '';
    formState.event_date = '';
    formState.start_time = '09:00';
    formState.end_time = '';
    setType('checkup');
    renderDateValue();
    renderTimeValues();
    setSwitch(notifySwitch, true);
    notifyRow.hidden = false;
    editNote.hidden = true;
    setFooterMode('create');
    resetSubmitButtons();
    eventSheet.querySelector('.sheet-scroll').scrollTop = 0;
    openLayer(eventSheet, sheetOverlay, eventSheet);
  }

  function openEditSheet(event) {
    if (!event || submitting) return;
    sheetMode = 'edit';
    editingId = event.id;
    editingBaseline = {
      title: event.title,
      description: event.description,
      event_type: event.event_type,
      event_date: event.event_date,
      start_time: event.start_time,
      end_time: event.end_time,
      location: event.location,
      is_all_day: event.is_all_day,
      notify: event.notify
    };
    sheetTitle.textContent = '编辑园级事件';
    eventTitle.value = event.title;
    eventDescription.value = event.description;
    eventLocation.value = event.location;
    formState.event_date = event.event_date;
    formState.start_time = event.start_time || '09:00';
    formState.end_time = event.end_time || '';
    setType(event.event_type);
    renderDateValue();
    renderTimeValues();
    // 草稿编辑走表单内开关（源码 startEdit 将 notify 置 false），已发布事件才隐藏开关。
    setSwitch(notifySwitch, event.status === 'draft' ? false : event.notify !== false);
    notifyRow.hidden = event.status === 'published';
    editNote.hidden = false;
    editNoteText.textContent = event.status === 'draft'
      ? '该事件仍为草稿——保存发布后，全园家长即可在日历中看到。'
      : '该事件已发布——保存时将让你确认是否通知家长（按变更内容预置建议）。';
    setFooterMode('edit');
    resetSubmitButtons();
    eventSheet.querySelector('.sheet-scroll').scrollTop = 0;
    openLayer(eventSheet, sheetOverlay, eventSheet);
  }

  function closeSheet() {
    if (submitting) return;
    if (eventSheet.hidden) return;
    closeLayer(eventSheet, sheetOverlay);
    sheetMode = null;
    editingId = null;
    editingBaseline = null;
    resetSubmitButtons();
  }

  function configureTextDialog(field) {
    textDialogField = field;
    textDialogInput.maxLength = field === 'date' ? 10 : 5;
    if (field === 'date') {
      textDialogTitle.textContent = '事件日期';
      textDialogHint.textContent = '格式 YYYY-MM-DD，留空保持原值';
      textDialogInput.placeholder = '2026-10-26';
      textDialogInput.value = formState.event_date;
    } else if (field === 'start') {
      textDialogTitle.textContent = '开始时间';
      textDialogHint.textContent = '24 小时制，如 09:00';
      textDialogInput.placeholder = '09:00';
      textDialogInput.value = formState.start_time;
    } else {
      textDialogTitle.textContent = '结束时间';
      textDialogHint.textContent = '24 小时制，留空保持原值';
      textDialogInput.placeholder = '如 11:00';
      textDialogInput.value = formState.end_time;
    }
  }

  function openTextDialog(field) {
    configureTextDialog(field);
    openLayer(textDialog, textDialogOverlay, textDialogInput);
    window.requestAnimationFrame(function () {
      if (!textDialog.hidden) {
        textDialogInput.focus({ preventScroll: true });
        textDialogInput.select();
      }
    });
  }

  function closeTextDialog() {
    if (textDialog.hidden) return;
    closeLayer(textDialog, textDialogOverlay);
    textDialogField = null;
  }

  function isValidDateString(value) {
    var match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
    if (!match) return false;
    var year = Number(match[1]);
    var month = Number(match[2]);
    var day = Number(match[3]);
    var date = new Date(year, month - 1, day, 12, 0, 0);
    return date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day;
  }

  function isValidTimeString(value) {
    return /^([01]\d|2[0-3]):[0-5]\d$/.test(value);
  }

  function commitTextDialog() {
    var value = textDialogInput.value.trim();
    if (!value) {
      closeTextDialog();
      return;
    }
    if (textDialogField === 'date') {
      if (!isValidDateString(value)) {
        showToast('请按 YYYY-MM-DD 格式填写日期');
        return;
      }
      formState.event_date = value;
    } else {
      if (!isValidTimeString(value)) {
        showToast('请按 HH:MM 格式填写时间');
        return;
      }
      if (textDialogField === 'start') formState.start_time = value;
      if (textDialogField === 'end') formState.end_time = value;
    }
    renderDateValue();
    renderTimeValues();
    closeTextDialog();
  }

  function readFormData() {
    var title = eventTitle.value.trim();
    if (!title) {
      showToast('请填写事件标题');
      eventTitle.focus({ preventScroll: true });
      return null;
    }
    if (!formState.event_date) {
      showToast('请选择事件日期');
      eventDateField.focus({ preventScroll: true });
      return null;
    }
    var current = editingId ? findEvent(editingId) : null;
    // 已发布事件是否通知家长只在变更确认弹层里决定（saveConfirm 会覆盖该值）；
    // 草稿编辑与新建一样读取表单内开关。
    var publishedEdit = sheetMode === 'edit' && !!current && current.status === 'published';
    return {
      title: title,
      event_type: formState.event_type,
      description: eventDescription.value.trim(),
      event_date: formState.event_date,
      start_time: formState.start_time || '09:00',
      end_time: formState.end_time,
      location: eventLocation.value.trim(),
      is_all_day: editingBaseline ? editingBaseline.is_all_day : false,
      notify: publishedEdit && editingBaseline ? editingBaseline.notify : isSwitchOn(notifySwitch)
    };
  }

  function handleCreateSubmit(kind) {
    if (submitting) return;
    var data = readFormData();
    if (!data) return;
    var pressedButton = kind === 'draft' ? saveDraftBtn : publishNowBtn;

    submitting = true;
    [saveDraftBtn, publishNowBtn].forEach(function (button) {
      button.disabled = true;
      button.setAttribute('aria-disabled', 'true');
    });
    saveDraftBtn.textContent = '存为草稿';
    publishNowBtn.textContent = '立即发布';
    pressedButton.textContent = kind === 'draft' ? '保存中…' : '发布中…';
    eventForm.setAttribute('aria-busy', 'true');
    proto.dataset.state = 'disabled';

    window.setTimeout(function () {
      events.push({
        id: 'evt-' + Date.now(),
        title: data.title,
        event_type: data.event_type,
        description: data.description,
        event_date: data.event_date,
        start_time: data.start_time,
        end_time: data.end_time,
        location: data.location,
        is_all_day: false,
        notify: data.notify,
        status: kind === 'draft' ? 'draft' : 'published'
      });
      proto.dataset.state = events.length ? 'normal' : 'empty';
      renderGroups();
      submitting = false;
      resetSubmitButtons();
      closeSheet();
      showToast(kind === 'draft' ? '已存草稿' : '发布成功');
    }, 800);
  }

  function buildChanges(baseline, data) {
    var changes = [];
    if (data.event_date !== baseline.event_date) {
      changes.push({ label: '日期', value: dateInfo(data.event_date).dateLabel });
    }
    if (data.start_time !== baseline.start_time || data.end_time !== baseline.end_time) {
      var timeValue = data.end_time ? data.start_time + '–' + data.end_time : data.start_time;
      changes.push({ label: '时间', value: timeValue });
    }
    if (data.location !== baseline.location) {
      changes.push({ label: '地点', value: data.location || '未填写' });
    }
    if (data.title !== baseline.title) changes.push({ label: '标题已更新' });
    if (data.description !== baseline.description) changes.push({ label: '内容已更新' });
    if (data.event_type !== baseline.event_type) changes.push({ label: '类型已更新' });
    return changes;
  }

  function changeAffectsTimeOrPlace(baseline, data) {
    return data.event_date !== baseline.event_date ||
      data.start_time !== baseline.start_time ||
      data.end_time !== baseline.end_time ||
      data.location !== baseline.location;
  }

  function populateChangeSheet(changes, suggestNotify) {
    changeList.innerHTML = changes.map(function (change) {
      var value = change.value
        ? '<span class="change-value">' + escapeHtml(change.value) + '</span>'
        : '';
      return '<div class="change-row">' +
        '<span class="change-bullet" aria-hidden="true"></span>' +
        '<span class="change-label">' + escapeHtml(change.label) + '</span>' +
        value +
      '</div>';
    }).join('');
    setSwitch(changeNotifySwitch, suggestNotify);
  }

  function applyEventChanges(event, data) {
    event.title = data.title;
    event.event_type = data.event_type;
    event.description = data.description;
    event.event_date = data.event_date;
    event.start_time = data.start_time;
    event.end_time = data.end_time;
    event.location = data.location;
    event.is_all_day = data.is_all_day;
    event.notify = data.notify;
  }

  function handleEditSave() {
    if (submitting || !editingId || !editingBaseline) return;
    var event = findEvent(editingId);
    if (!event) return;
    var data = readFormData();
    if (!data) return;
    var changes = buildChanges(editingBaseline, data);

    if (event.status === 'published' && changes.length) {
      pendingEdit = { id: editingId, data: data, changes: changes };
      populateChangeSheet(changes, changeAffectsTimeOrPlace(editingBaseline, data));
      openLayer(confirmSheet, confirmOverlay, confirmSheet);
      return;
    }

    applyEventChanges(event, data);
    event.status = 'published';
    proto.dataset.state = 'normal';
    renderGroups();
    closeSheet();
    showToast('已保存');
  }

  // 编辑态「存为草稿」：直接以 draft 保存，不弹变更确认、不推送家长（对齐源码 submitDraft）。
  function handleEditDraftSave() {
    if (submitting || !editingId || !editingBaseline) return;
    var event = findEvent(editingId);
    if (!event) return;
    var data = readFormData();
    if (!data) return;
    data.notify = false;
    applyEventChanges(event, data);
    event.status = 'draft';
    renderGroups();
    closeSheet();
    showToast('已存草稿');
  }

  function cancelConfirm() {
    if (confirmSheet.hidden) return;
    pendingEdit = null;
    closeLayer(confirmSheet, confirmOverlay);
  }

  function saveConfirm() {
    if (!pendingEdit) return;
    var event = findEvent(pendingEdit.id);
    if (!event) {
      pendingEdit = null;
      closeLayer(confirmSheet, confirmOverlay);
      return;
    }
    // 「发布变更通知」开关是唯一决定是否推送家长的地方，必须写回后再应用。
    pendingEdit.data.notify = isSwitchOn(changeNotifySwitch);
    applyEventChanges(event, pendingEdit.data);
    event.status = 'published';
    pendingEdit = null;
    proto.dataset.state = 'normal';
    renderGroups();
    closeLayer(confirmSheet, confirmOverlay);
    closeSheet();
    showToast('已保存');
  }

  function closeDeleteDialog() {
    if (deleteDialog.hidden) return;
    deleteTargetId = null;
    closeLayer(deleteDialog, deleteOverlay);
  }

  function openDeleteDialog(id) {
    var event = findEvent(id);
    if (!event) return;
    deleteTargetId = id;
    deleteBody.textContent = '确定删除「' + event.title + '」吗？删除后全园家长不再可见。';
    openLayer(deleteDialog, deleteOverlay, cancelDeleteBtn);
  }

  function confirmDelete() {
    if (!deleteTargetId) return;
    events = events.filter(function (event) { return event.id !== deleteTargetId; });
    deleteTargetId = null;
    closeLayer(deleteDialog, deleteOverlay);
    proto.dataset.state = events.length ? 'normal' : 'empty';
    renderGroups();
    showToast('已删除');
  }

  function reloadEvents() {
    proto.dataset.state = events.length ? 'normal' : 'empty';
    renderGroups();
    showToast('已重新加载');
  }

  // 选中与折叠分开：选中只负责「今天 vs 选中」的视觉语义（实心圆），
  // 折叠交给右侧 chevron。合并成一个点击会让用户刚选中某天就看到事件消失。
  function toggleDaySelect(button) {
    var group = button.closest('.agenda-group');
    if (!group) return;
    var date = button.dataset.date;
    var willSelect = selectedDate !== date;

    var previous = agendaGroups.querySelector('.agenda-group.is-selected');
    if (previous && previous !== group) {
      previous.classList.remove('is-selected');
      var previousButton = previous.querySelector('.day-select');
      if (previousButton) previousButton.setAttribute('aria-pressed', 'false');
    }

    selectedDate = willSelect ? date : null;
    group.classList.toggle('is-selected', willSelect);
    button.setAttribute('aria-pressed', String(willSelect));
    if (willSelect) {
      // 选中即展开：否则实心圆标着的是一组看不见的事件。
      collapsedDates.delete(date);
      var toggle = group.querySelector('.day-toggle');
      if (toggle) {
        toggle.setAttribute('aria-expanded', 'true');
        toggle.setAttribute('aria-label', '收起' + dateInfo(date).dateLabel + '的事件');
      }
      group.classList.remove('is-collapsed');
    }
  }

  function toggleCollapse(button) {
    var group = button.closest('.agenda-group');
    if (!group) return;
    var date = button.dataset.date;
    var willExpand = button.getAttribute('aria-expanded') !== 'true';
    button.setAttribute('aria-expanded', String(willExpand));
    button.setAttribute('aria-label', (willExpand ? '收起' : '展开') + dateInfo(date).dateLabel + '的事件');
    group.classList.toggle('is-collapsed', !willExpand);
    if (willExpand) collapsedDates.delete(date);
    else collapsedDates.add(date);
  }

  // Event delegation for the agenda.
  agendaGroups.addEventListener('click', function (event) {
    var actionTarget = event.target.closest('[data-action]');
    if (!actionTarget || !agendaGroups.contains(actionTarget)) return;
    var action = actionTarget.getAttribute('data-action');

    if (action === 'toggle-day') {
      toggleDaySelect(actionTarget);
      return;
    }
    if (action === 'toggle-collapse') {
      toggleCollapse(actionTarget);
      return;
    }

    var id = actionTarget.getAttribute('data-id');
    var item = id ? findEvent(id) : null;
    if (!item) return;

    if (action === 'open-event') {
      showToast('已打开 ' + item.title);
    } else if (action === 'edit-event') {
      openEditSheet(item);
    } else if (action === 'delete-event') {
      openDeleteDialog(id);
    }
  });

  // Toolbar and empty/error actions.
  openPublishBtn.addEventListener('click', openCreateSheet);
  emptyPublishBtn.addEventListener('click', openCreateSheet);
  reloadEventsBtn.addEventListener('click', reloadEvents);
  closeSheetBtn.addEventListener('click', closeSheet);
  sheetOverlay.addEventListener('click', closeSheet);
  cancelEditBtn.addEventListener('click', closeSheet);
  saveDraftBtn.addEventListener('click', function () {
    if (sheetMode === 'edit') handleEditDraftSave();
    else handleCreateSubmit('draft');
  });
  publishNowBtn.addEventListener('click', function () { handleCreateSubmit('publish'); });
  saveEditBtn.addEventListener('click', handleEditSave);

  // Type radio group.
  typeSwitch.addEventListener('click', function (event) {
    var option = event.target.closest('.type-option');
    if (!option) return;
    setType(option.dataset.type);
  });

  typeSwitch.addEventListener('keydown', function (event) {
    var current = event.target.closest('.type-option');
    if (!current) return;
    var currentIndex = typeButtons.indexOf(current);
    var nextIndex = null;
    if (event.key === 'ArrowRight' || event.key === 'ArrowDown') nextIndex = (currentIndex + 1) % typeButtons.length;
    if (event.key === 'ArrowLeft' || event.key === 'ArrowUp') nextIndex = (currentIndex - 1 + typeButtons.length) % typeButtons.length;
    if (event.key === 'Home') nextIndex = 0;
    if (event.key === 'End') nextIndex = typeButtons.length - 1;
    if (nextIndex === null) return;
    event.preventDefault();
    setType(typeButtons[nextIndex].dataset.type);
    typeButtons[nextIndex].focus();
  });

  // Switches.
  [notifySwitch, changeNotifySwitch].forEach(function (button) {
    button.addEventListener('click', function () {
      setSwitch(button, !isSwitchOn(button));
    });
  });

  // Editable native-style dialogs for date and time.
  eventDateField.addEventListener('click', function () { openTextDialog('date'); });
  startTimeField.addEventListener('click', function () { openTextDialog('start'); });
  endTimeField.addEventListener('click', function () { openTextDialog('end'); });
  textDialogCancel.addEventListener('click', closeTextDialog);
  textDialogConfirm.addEventListener('click', commitTextDialog);
  textDialogOverlay.addEventListener('click', closeTextDialog);
  textDialogInput.addEventListener('keydown', function (event) {
    if (event.key === 'Enter') {
      event.preventDefault();
      commitTextDialog();
    }
  });

  // Confirmation sheet.
  cancelConfirmBtn.addEventListener('click', cancelConfirm);
  saveConfirmBtn.addEventListener('click', saveConfirm);
  confirmOverlay.addEventListener('click', cancelConfirm);

  // Delete alert.
  cancelDeleteBtn.addEventListener('click', closeDeleteDialog);
  confirmDeleteBtn.addEventListener('click', confirmDelete);
  deleteOverlay.addEventListener('click', closeDeleteDialog);

  eventForm.addEventListener('submit', function (event) { event.preventDefault(); });

  // Global escape and focus trap.
  document.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
      if (!textDialog.hidden) {
        event.preventDefault();
        closeTextDialog();
      } else if (!deleteDialog.hidden) {
        event.preventDefault();
        closeDeleteDialog();
      } else if (!confirmSheet.hidden) {
        event.preventDefault();
        cancelConfirm();
      } else if (!eventSheet.hidden) {
        event.preventDefault();
        closeSheet();
      }
      return;
    }

    if (event.key !== 'Tab') return;
    var layer = getOpenLayer();
    if (!layer) return;
    var focusable = getFocusable(layer);
    if (!focusable.length) {
      event.preventDefault();
      layer.focus({ preventScroll: true });
      return;
    }
    var first = focusable[0];
    var last = focusable[focusable.length - 1];
    var active = document.activeElement;
    if (event.shiftKey && (active === first || active === layer || !layer.contains(active))) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && active === last) {
      event.preventDefault();
      first.focus();
    }
  });

  // Host-injected state demos: surface a toast when the list enters error.
  var lastObservedState = proto.dataset.state;
  var stateObserver = new MutationObserver(function () {
    var state = proto.dataset.state;
    if (state === 'error' && lastObservedState !== 'error') showToast('加载事件失败');
    lastObservedState = state;
  });
  stateObserver.observe(proto, { attributes: true, attributeFilter: ['data-state'] });

  setType('checkup');
  renderGroups();
  if (proto.dataset.state === 'error') showToast('加载事件失败');
})();
