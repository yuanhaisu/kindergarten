/* 班级日历（老师端）原型交互
 * 真日期数学重建月历；四态由共享 shell 的 mp-demo 驱动（mp:state 事件）。 */
(() => {
  const app = document.querySelector('.mp-app');
  const grid = document.getElementById('calGrid');
  const monthLabel = document.getElementById('monthLabel');
  const monthSub = document.getElementById('monthSub');
  const legend = document.getElementById('calLegend');
  const dayDate = document.getElementById('dayDate');
  const dayCount = document.getElementById('dayCount');
  const eventList = document.getElementById('eventList');
  const dayEmpty = document.getElementById('dayEmpty');
  const emptyTitle = document.getElementById('emptyTitle');
  const toast = document.getElementById('toast');

  const TODAY = new Date(2026, 9, 26);
  const WEEK = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'];
  const PIN_SVG =
    '<svg width="13" height="13" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" aria-hidden="true"><path d="M8 14s4.5-4.1 4.5-7.5a4.5 4.5 0 1 0-9 0C3.5 9.9 8 14 8 14Z"/><circle cx="8" cy="6.4" r="1.6"/></svg>';

  // 字典 calendar_event_type 的内置 5 类；色走 token 对（见 styles.css [data-tone]）。
  const TYPES = {
    notice: { label: '通知', tone: 'info' },
    activity: { label: '活动', tone: 'success' },
    holiday: { label: '假期', tone: 'accent' },
    meeting: { label: '家长会', tone: 'muted' },
    checkup: { label: '体检', tone: 'danger' },
  };

  // 接口 /teacher/calendar/events 的真实字段形状。
  const EVENTS = [
    { id: 1, title: '国庆假期', description: '', event_type: 'holiday', scope: 'kindergarten', status: 'published', event_date: '2026-10-01', start_time: '', end_time: '', location: '', is_all_day: true },
    { id: 2, title: '国庆假期', description: '', event_type: 'holiday', scope: 'kindergarten', status: 'published', event_date: '2026-10-02', start_time: '', end_time: '', location: '', is_all_day: true },
    { id: 3, title: '秋季流感疫苗接种', description: '请家长当天按时送孩子入园，接种前避免空腹。', event_type: 'notice', scope: 'class', status: 'published', event_date: '2026-10-08', start_time: '', end_time: '', location: '保健室', is_all_day: true },
    { id: 4, title: '家长开放日', description: '半天开放，家长可进班观摩集体活动与区域游戏。', event_type: 'meeting', scope: 'class', status: 'published', event_date: '2026-10-14', start_time: '09:00', end_time: '11:00', location: '小一班教室', is_all_day: false },
    { id: 5, title: '秋游', description: '全园秋游，请为孩子准备水壶、遮阳帽和轻便午餐。', event_type: 'activity', scope: 'kindergarten', status: 'published', event_date: '2026-10-16', start_time: '', end_time: '', location: '城市植物园', is_all_day: true },
    { id: 6, title: '消防安全演练', description: '全园同步进行，听到警报后按疏散路线撤离到操场。', event_type: 'activity', scope: 'kindergarten', status: 'published', event_date: '2026-10-21', start_time: '10:00', end_time: '10:40', location: '操场', is_all_day: false },
    { id: 7, title: '秋季运动会彩排', description: '各班按方阵入场，练习接力与亲子项目走位，请提前十分钟带队到操场集合。', event_type: 'activity', scope: 'class', status: 'published', event_date: '2026-10-26', start_time: '09:00', end_time: '10:30', location: '大操场', is_all_day: false },
    { id: 8, title: '班级家长会', description: '沟通本月成长重点与运动会安排，请家长提前十分钟到场签到。', event_type: 'meeting', scope: 'class', status: 'published', event_date: '2026-10-26', start_time: '16:00', end_time: '17:00', location: '小一班教室', is_all_day: false },
    { id: 9, title: '视力筛查', description: '由区妇幼保健院入园筛查，请家长当天准时送孩子入园。', event_type: 'checkup', scope: 'class', status: 'published', event_date: '2026-10-27', start_time: '', end_time: '', location: '保健室', is_all_day: true },
    { id: 10, title: '万圣节变装游园', description: '变装游园与讨糖游戏，请家长协助准备简单装扮。', event_type: 'activity', scope: 'class', status: 'draft', event_date: '2026-10-29', start_time: '15:30', end_time: '16:30', location: '小一班教室', is_all_day: false },
    { id: 11, title: '重阳节敬老活动', description: '邀请爷爷奶奶进班，孩子表演节目并送上手工礼物。', event_type: 'activity', scope: 'class', status: 'published', event_date: '2026-10-30', start_time: '09:30', end_time: '10:30', location: '多功能厅', is_all_day: false },
    { id: 12, title: '秋季亲子运动会', description: '全园亲子运动会，请家长穿运动鞋参加。', event_type: 'activity', scope: 'kindergarten', status: 'published', event_date: '2026-11-03', start_time: '09:00', end_time: '11:30', location: '大操场', is_all_day: false },
    { id: 13, title: '家长开放日', description: '半天开放，家长可进班观摩集体活动与区域游戏。', event_type: 'meeting', scope: 'class', status: 'published', event_date: '2026-11-11', start_time: '09:00', end_time: '11:00', location: '小一班教室', is_all_day: false },
  ];

  let viewYear = TODAY.getFullYear();
  let viewMonth = TODAY.getMonth() + 1;
  let selectedDay = TODAY.getDate();
  let demoState = 'normal';
  let toastTimer = 0;

  const pad2 = (n) => String(n).padStart(2, '0');
  const dateKey = (y, m, d) => `${y}-${pad2(m)}-${pad2(d)}`;
  const daysInMonth = (y, m) => new Date(y, m, 0).getDate();

  function sortEvents(a, b) {
    if (a.is_all_day !== b.is_all_day) return a.is_all_day ? -1 : 1;
    return (a.start_time || '99:99').localeCompare(b.start_time || '99:99');
  }
  function eventsOn(key) {
    return EVENTS.filter((e) => e.event_date === key).sort(sortEvents);
  }
  function isViewingCurrentMonth() {
    return viewYear === TODAY.getFullYear() && viewMonth === TODAY.getMonth() + 1;
  }

  function buildCells() {
    const firstWeekday = new Date(viewYear, viewMonth - 1, 1).getDay();
    const dim = daysInMonth(viewYear, viewMonth);
    const prevDim = new Date(viewYear, viewMonth - 1, 0).getDate();
    const cells = [];
    for (let i = firstWeekday - 1; i >= 0; i -= 1) cells.push({ day: prevDim - i, type: 'prev' });
    for (let d = 1; d <= dim; d += 1) cells.push({ day: d, type: 'current' });
    const remaining = Math.min(42 - cells.length, 14);
    for (let d = 1; d <= remaining; d += 1) cells.push({ day: d, type: 'next' });
    return cells;
  }

  function renderMonth() {
    monthLabel.textContent = `${viewYear}年${viewMonth}月`;
    // 加载/失败态的数据并不可信：不先报出具体数量，否则与下方的骨架屏、错误块正面矛盾。
    if (demoState === 'loading') {
      monthSub.textContent = '正在加载…';
      return;
    }
    if (demoState === 'error') {
      monthSub.textContent = '活动加载失败';
      return;
    }
    const count = EVENTS.filter((e) => e.event_date.startsWith(`${viewYear}-${pad2(viewMonth)}`)).length;
    monthSub.textContent = demoState === 'empty' || count === 0 ? '本月暂无活动' : `本月 ${count} 个活动`;
  }

  // 失败态下月历已被错误块替换，月份条只保留可读的月份：箭头禁用，避免「月份变了但错误块原样」。
  function syncMonthNav() {
    const blocked = demoState === 'error';
    document.getElementById('prevMonth').disabled = blocked;
    document.getElementById('nextMonth').disabled = blocked;
  }

  function renderGrid() {
    const frag = document.createDocumentFragment();
    for (const cell of buildCells()) {
      const isCurrent = cell.type === 'current';
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'cell';
      if (!isCurrent) btn.classList.add('is-outside');
      const isToday = isCurrent && isViewingCurrentMonth() && cell.day === TODAY.getDate();
      if (isToday) btn.classList.add('is-today');
      const isSelected = isCurrent && cell.day === selectedDay;
      if (isSelected) btn.classList.add('is-selected');

      const dayEvents = isCurrent ? eventsOn(dateKey(viewYear, viewMonth, cell.day)) : [];
      const num = document.createElement('span');
      num.className = 'cell-num';
      num.textContent = String(cell.day);
      const dots = document.createElement('span');
      dots.className = 'cell-dots';
      for (const ev of dayEvents.slice(0, 3)) {
        const dot = document.createElement('i');
        dot.className = 'cell-dot';
        dot.dataset.tone = (TYPES[ev.event_type] || {}).tone || 'muted';
        dots.appendChild(dot);
      }
      btn.append(num, dots);

      if (isCurrent) {
        btn.dataset.day = String(cell.day);
        btn.setAttribute('aria-pressed', String(isSelected));
        const label = `${viewMonth}月${cell.day}日${isToday ? '，今天' : ''}${dayEvents.length ? `，${dayEvents.length} 个活动` : ''}`;
        btn.setAttribute('aria-label', label);
      } else {
        btn.disabled = true;
        btn.tabIndex = -1;
        btn.setAttribute('aria-hidden', 'true');
      }
      frag.appendChild(btn);
    }
    grid.replaceChildren(frag);
  }

  function renderLegend() {
    const frag = document.createDocumentFragment();
    Object.values(TYPES).forEach((meta) => {
      const item = document.createElement('span');
      item.className = 'legend-item';
      const dot = document.createElement('i');
      dot.className = 'legend-dot';
      dot.dataset.tone = meta.tone;
      const text = document.createElement('span');
      text.textContent = meta.label;
      item.append(dot, text);
      frag.appendChild(item);
    });
    legend.replaceChildren(frag);
  }

  function buildEventCard(ev) {
    const meta = TYPES[ev.event_type] || { label: ev.event_type_label || ev.event_type, tone: 'muted' };
    const card = document.createElement('article');
    card.className = 'event-card';

    const top = document.createElement('div');
    top.className = 'event-top';
    const icon = document.createElement('span');
    icon.className = 'event-icon';
    icon.dataset.tone = meta.tone;
    icon.textContent = meta.label.charAt(0);
    icon.setAttribute('aria-hidden', 'true');

    const body = document.createElement('div');
    body.className = 'event-body';
    const titleRow = document.createElement('div');
    titleRow.className = 'event-title-row';
    const title = document.createElement('h3');
    title.className = 'event-title';
    title.textContent = ev.title;
    titleRow.appendChild(title);
    // 与源码一致：草稿、全园都用标题旁的小标签；本班是默认值，不额外标注。
    if (ev.status === 'draft') {
      const chip = document.createElement('span');
      chip.className = 'chip is-draft';
      chip.textContent = '草稿';
      titleRow.appendChild(chip);
    }
    if (ev.scope === 'kindergarten') {
      const chip = document.createElement('span');
      chip.className = 'chip is-garden';
      chip.textContent = '全园';
      titleRow.appendChild(chip);
    }

    const metaRow = document.createElement('div');
    metaRow.className = 'event-meta';
    const typeEl = document.createElement('span');
    typeEl.className = 'event-type';
    typeEl.dataset.tone = meta.tone;
    typeEl.textContent = ev.event_type_label || meta.label;
    metaRow.appendChild(typeEl);
    const timeText = ev.is_all_day ? '全天' : [ev.start_time, ev.end_time].filter(Boolean).join('–');
    if (timeText) {
      const sep = document.createElement('span');
      sep.className = 'meta-sep';
      sep.textContent = '·';
      sep.setAttribute('aria-hidden', 'true');
      const time = document.createElement('span');
      time.className = 'event-time';
      time.textContent = timeText;
      metaRow.append(sep, time);
    }
    body.append(titleRow, metaRow);
    top.append(icon, body);
    card.appendChild(top);

    if (ev.description) {
      const desc = document.createElement('p');
      desc.className = 'event-desc';
      desc.textContent = ev.description;
      card.appendChild(desc);
    }

    const foot = document.createElement('div');
    foot.className = 'event-foot';
    if (ev.location) {
      const loc = document.createElement('span');
      loc.className = 'foot-item';
      loc.innerHTML = PIN_SVG;
      const locText = document.createElement('span');
      locText.textContent = ev.location;
      loc.appendChild(locText);
      foot.appendChild(loc);
    }

    // 园级事件只读：源码 v-if="event.scope !== 'kindergarten'" 才显示编辑。
    if (ev.scope !== 'kindergarten') {
      const edit = document.createElement('button');
      edit.type = 'button';
      edit.className = 'event-edit';
      edit.textContent = '编辑';
      edit.addEventListener('click', () => showToast(`已打开「${ev.title}」详情`));
      foot.appendChild(edit);
    }
    // 无地点且不可编辑时（如全园假期）不渲染空的底栏，避免卡片尾部留出一段空白。
    if (foot.childElementCount) card.appendChild(foot);
    else card.classList.add('has-no-foot');
    return card;
  }

  function renderDay() {
    const list = eventsOn(dateKey(viewYear, viewMonth, selectedDay));
    const weekday = WEEK[new Date(viewYear, viewMonth - 1, selectedDay).getDay()];
    dayDate.textContent = `${viewMonth}月${selectedDay}日 ${weekday}`;
    dayCount.textContent = demoState === 'empty' || !list.length ? '暂无活动' : `${list.length} 个活动`;
    emptyTitle.textContent = `${viewMonth}月${selectedDay}日暂无活动安排`;

    const frag = document.createDocumentFragment();
    list.forEach((ev) => frag.appendChild(buildEventCard(ev)));
    eventList.replaceChildren(frag);

    const noEvents = demoState === 'empty' || list.length === 0;
    eventList.hidden = noEvents;
    dayEmpty.hidden = !noEvents;
  }

  function renderAll() {
    renderMonth();
    renderGrid();
    renderDay();
  }

  function shiftMonth(delta) {
    const next = new Date(viewYear, viewMonth - 1 + delta, 1);
    viewYear = next.getFullYear();
    viewMonth = next.getMonth() + 1;
    const dim = daysInMonth(viewYear, viewMonth);
    if (isViewingCurrentMonth()) selectedDay = TODAY.getDate();
    else if (selectedDay > dim) selectedDay = dim;
    renderAll();
  }

  function showToast(text) {
    toast.textContent = text;
    toast.classList.add('is-on');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => toast.classList.remove('is-on'), 1800);
  }

  function applyState(state) {
    demoState = state;
    grid.classList.toggle('is-empty', state === 'empty');
    syncMonthNav();
    renderMonth();
    renderDay();
  }

  function setState(state) {
    app.dataset.state = state;
    document.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === state));
    });
    applyState(state);
  }

  document.getElementById('prevMonth').addEventListener('click', () => shiftMonth(-1));
  document.getElementById('nextMonth').addEventListener('click', () => shiftMonth(1));
  document.getElementById('addBtn').addEventListener('click', () => showToast('打开新建活动'));
  document.getElementById('emptyAddBtn').addEventListener('click', () => showToast('打开新建活动'));
  document.getElementById('retryBtn').addEventListener('click', () => {
    setState('normal');
    showToast('已重新加载');
  });

  grid.addEventListener('click', (event) => {
    const btn = event.target.closest('.cell');
    if (!btn || btn.disabled || !btn.dataset.day) return;
    selectedDay = Number(btn.dataset.day);
    renderGrid();
    renderDay();
  });

  document.addEventListener('mp:state', (event) => {
    applyState((event.detail && event.detail.state) || 'normal');
  });

  // 骨架屏网格由 JS 补齐（35 个占位），避免 HTML 里堆 35 个空节点。
  const skGrid = document.getElementById('skGrid');
  for (let i = 0; i < 35; i += 1) {
    const cell = document.createElement('span');
    cell.className = 'sk-cell';
    skGrid.appendChild(cell);
  }

  renderLegend();
  syncMonthNav();
  renderAll();
})();
