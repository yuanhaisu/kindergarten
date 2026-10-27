/* 添加成长记录 · 原型交互
 * 外壳（状态演示条）由 ../../_shared/shell.js 负责，这里只接管页面内容与弹层。
 */
(() => {
  const app = document.querySelector('.mp-app');
  if (!app) return;

  const $ = (id) => document.getElementById(id);

  const dayDate = $('dayDate');
  const daySub = $('daySub');
  const dayPrev = $('dayPrev');
  const dayNext = $('dayNext');
  const obsCard = $('obsCard');
  const studentCard = $('studentCard');
  const pickedBox = $('pickedBox');
  const avatars = $('avatars');
  const pickedText = $('pickedText');
  const pickCount = $('pickCount');
  const editPick = $('editPick');
  const openPickEmpty = $('openPickEmpty');
  const mediaGrid = $('mediaGrid');
  const mediaCount = $('mediaCount');
  const namedTip = $('namedTip');
  const descList = $('descList');
  const batchBtn = $('batchBtn');
  const batchTools = $('batchTools');
  const selectAllPhotos = $('selectAllPhotos');
  const pickStudentsBtn = $('pickStudentsBtn');
  const cancelBatchBtn = $('cancelBatchBtn');
  const addMediaBtn = $('addMedia');
  const contentInput = $('contentInput');
  const charCount = $('charCount');
  const badges = $('badges');
  const badgeCard = $('badgeCard');
  const grantBox = $('grantBox');
  const grantStudents = $('grantStudents');
  const witness = $('witness');
  const scopeText = $('scopeText');
  const syncRow = $('syncRow');
  const syncSw = $('syncSw');
  const dockReason = $('dockReason');
  const publishBtn = $('publishBtn');
  const errRetry = $('errRetry');
  const toast = $('toast');
  const sheetLayer = $('sheetLayer');
  const sheetTitle = $('sheetTitle');
  const sheetDesc = $('sheetDesc');
  const sheetList = $('sheetList');
  const searchInput = $('searchInput');
  const selectAllBtn = $('selectAll');
  const confirmPick = $('confirmPick');

  const ROSTER = ['乐乐', '小雨', '朵朵', '小满', '果果', '豆豆', '安安', '小北', '一一', '妞妞',
    '壮壮', '希希', '圆圆', '木木', '丫丫', '天天', '星星', '贝贝', '九月', '六六',
    '点点', '淘淘', '糖糖', '小禾', '阿布', '团团', '米朵', '向南'];
  const TAGS = ['乐于助人', '积极发言', '遵守纪律', '动手能力强', '专注力强',
    '有礼貌', '爱干净', '团结友善', '勇于尝试'];
  const SAMPLE = '今天上午带孩子们去后院捡了落叶，回来做了树叶拓印。乐乐专注地试了三种颜色，小雨主动把胶棒分给旁边的小朋友。';

  const state = {
    mode: 'class',
    edit: false,
    selected: new Set(['乐乐', '小雨', '朵朵', '小满', '果果']),
    media: [],
    submitting: false,
  };

  let sheetMode = 'pick';
  let sheetTarget = null;
  let batchMode = false;
  let batchSelected = new Set();
  let temp = new Set();
  let dayOffset = 0;
  let toastTimer = null;

  /* ── 通用小件 ─────────────────────────────────────── */

  function showToast(text) {
    toast.textContent = text;
    toast.hidden = false;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => { toast.hidden = true; }, 1800);
  }

  function setState(next) {
    app.dataset.state = next;
    document.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.setAttribute('aria-pressed', String(chip.dataset.mpState === next));
    });
  }

  function artSvg(kind) {
    return `<span class="art ${kind}" aria-hidden="true"><i></i><i></i><i></i></span>`;
  }

  function artFor(index) {
    return ['art-leaf', 'art-clay', 'art-yard'][index % 3];
  }

  /* ── 标签 ─────────────────────────────────────────── */

  function buildTags(host, group) {
    if (!host) return;
    host.innerHTML = '';
    TAGS.forEach((tag) => {
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'tag' + (['动手能力强', '乐于助人'].includes(tag) ? ' is-on' : '');
      btn.dataset.tag = tag;
      btn.dataset.group = group;
      btn.setAttribute('aria-pressed', String(btn.classList.contains('is-on')));
      btn.textContent = tag;
      host.appendChild(btn);
    });
  }

  document.addEventListener('click', (event) => {
    const tag = event.target.closest('.tag');
    if (!tag) return;
    const on = !tag.classList.contains('is-on');
    document.querySelectorAll(`.tag[data-tag="${tag.dataset.tag}"]`).forEach((el) => {
      el.classList.toggle('is-on', on);
      el.setAttribute('aria-pressed', String(on));
    });
  });

  /* ── 单选组（表情 / chip）─────────────────────────── */

  function bindGroups(root) {
    root.addEventListener('click', (event) => {
      const btn = event.target.closest('.rate, .chip[data-value]');
      if (!btn) return;
      const group = btn.parentElement;
      if (!group) return;
      const wasOn = btn.classList.contains('is-on');
      group.querySelectorAll('.rate, .chip[data-value]').forEach((el) => {
        el.classList.remove('is-on');
        el.setAttribute('aria-pressed', 'false');
      });
      if (!wasOn) {
        btn.classList.add('is-on');
        btn.setAttribute('aria-pressed', 'true');
      }
    });
  }

  /* ── 日期 ─────────────────────────────────────────── */

  const DAYS = [
    { date: '10月24日 周六', sub: '2026年 · 今天' },
    { date: '10月23日 周五', sub: '2026年 · 昨天' },
    { date: '10月22日 周四', sub: '2026年 · 前天' },
  ];

  function renderDay() {
    dayDate.textContent = DAYS[dayOffset].date;
    daySub.textContent = DAYS[dayOffset].sub;
    dayPrev.disabled = state.edit || dayOffset >= DAYS.length - 1;
    dayNext.disabled = state.edit || dayOffset === 0;
  }

  dayPrev.addEventListener('click', () => {
    if (dayOffset < DAYS.length - 1) { dayOffset += 1; renderDay(); }
  });
  dayNext.addEventListener('click', () => {
    if (dayOffset > 0) { dayOffset -= 1; renderDay(); }
  });

  /* ── 记录对象 ─────────────────────────────────────── */

  function setMode(mode) {
    state.mode = mode;
    document.querySelectorAll('.mode-btn').forEach((btn) => {
      const on = btn.dataset.mode === mode;
      btn.classList.toggle('is-on', on);
      btn.setAttribute('aria-selected', String(on));
      btn.disabled = state.edit;
    });
    obsCard.hidden = mode !== 'class';
    studentCard.hidden = mode !== 'student';
    // 编辑态可见范围锁定（源码 ?id= 分支）：不同步班级动态、不可切记录对象
    syncRow.hidden = state.edit || mode !== 'class';
    refreshDock();
  }

  document.querySelectorAll('.mode-btn').forEach((btn) => {
    btn.addEventListener('click', () => setMode(btn.dataset.mode));
  });

  /* ── 已选幼儿 ─────────────────────────────────────── */

  function renderPicked() {
    const names = [...state.selected];
    pickCount.textContent = `已选 ${names.length} 人`;
    avatars.innerHTML = '';
    names.slice(0, 3).forEach((name) => {
      const span = document.createElement('span');
      span.className = 'ava';
      span.textContent = name.slice(0, 1);
      avatars.appendChild(span);
    });
    if (names.length > 3) {
      const more = document.createElement('span');
      more.className = 'ava ava--more';
      more.textContent = `+${names.length - 3}`;
      avatars.appendChild(more);
    }
    if (!names.length) {
      pickedText.textContent = '还没有选择幼儿';
    } else if (names.length <= 3) {
      pickedText.textContent = names.join('、');
    } else {
      pickedText.textContent = `${names.slice(0, 3).join('、')} 等 ${names.length} 人`;
    }
    // 摘要 / 空态按真实人数互斥，不再由演示条 data-state 决定
    pickedBox.hidden = names.length === 0;
    pickEmpty.hidden = names.length > 0;
  }

  /* ── 照片 ─────────────────────────────────────────── */

  function namedPhotoCount() {
    return state.media.filter((m) => m.kind === 'done' && m.students && m.students.length).length;
  }

  function mediaBadge(item) {
    if (item.students && item.students.length) return `点名 ${item.students.length}`;
    if (item.suggestions) return `AI 建议 ${item.suggestions}`;
    return '待点名';
  }

  function doneIndexes() {
    return state.media.map((m, i) => (m.kind === 'done' ? i : -1)).filter((i) => i >= 0);
  }

  function allPhotosSelected() {
    const idx = doneIndexes();
    return idx.length > 0 && idx.every((i) => batchSelected.has(i));
  }

  function renderMediaBar() {
    if (batchMode) {
      namedTip.textContent = `已选 ${batchSelected.size} 张照片`;
      selectAllPhotos.textContent = allPhotosSelected() ? '取消全选' : '全选';
    } else {
      namedTip.textContent = `已点名 ${namedPhotoCount()} 张`;
    }
  }

  function renderMedia() {
    mediaGrid.innerHTML = '';
    state.media.forEach((item, index) => {
      const tile = document.createElement('div');
      tile.className = 'tile';

      if (item.kind === 'uploading') {
        tile.innerHTML = `${artSvg(artFor(index))}
          <span class="tile-mask"><span class="spin"></span><span>${item.pct}%</span></span>`;
      } else if (item.kind === 'failed') {
        tile.innerHTML = `${artSvg(artFor(index))}
          <button class="tile-mask is-failed" type="button" aria-label="上传失败，点击重试">上传失败<br>点击重试</button>`;
        tile.querySelector('.tile-mask').addEventListener('click', () => retryUpload(index));
      } else {
        tile.classList.toggle('is-picked', batchMode && batchSelected.has(index));
        tile.innerHTML = `${artSvg(artFor(index))}
          <button class="tile-badge" type="button" aria-label="编辑这张照片的点名">${mediaBadge(item)}</button>
          ${batchMode
            ? '<span class="tile-pick" aria-hidden="true">✓</span>'
            : '<button class="tile-x" type="button" aria-label="删除这张照片"><span>✕</span></button>'}`;
        if (batchMode) {
          tile.addEventListener('click', () => toggleBatchSelect(index));
        } else {
          // 整张瓦片（含角标）都是单张点名入口，与源码 @click="openSingleNaming(i)" 一致
          tile.addEventListener('click', () => openSingle(index));
          tile.querySelector('.tile-x').addEventListener('click', (event) => {
            event.stopPropagation();
            state.media.splice(index, 1);
            renderMedia();
            refreshDock();
          });
        }
      }
      mediaGrid.appendChild(tile);
    });

    const add = document.createElement('button');
    add.type = 'button';
    add.className = 'tile tile-add';
    add.id = 'addMedia';
    add.innerHTML = '<span class="plus" aria-hidden="true">＋</span><span>添加</span>';
    add.addEventListener('click', addMedia);
    mediaGrid.appendChild(add);

    const done = state.media.filter((m) => m.kind === 'done').length;
    mediaCount.textContent = `${state.media.length} / 20`;
    descList.hidden = done === 0;
    renderMediaBar();
  }

  function addMedia() {
    if (state.media.some((m) => m.kind === 'uploading')) {
      showToast('还有照片在上传中');
      return;
    }
    state.media.push({ kind: 'uploading', pct: 0 });
    renderMedia();
    refreshDock();
    const index = state.media.length - 1;
    const tick = setInterval(() => {
      const item = state.media[index];
      if (!item || item.kind !== 'uploading') { clearInterval(tick); return; }
      item.pct = Math.min(100, item.pct + 11);
      if (item.pct >= 100) {
        item.kind = 'done';
        item.students = [];
        item.suggestions = 0;
        clearInterval(tick);
      }
      renderMedia();
      refreshDock();
    }, 200);
  }

  function retryUpload(index) {
    const item = state.media[index];
    if (!item) return;
    item.kind = 'uploading';
    item.pct = 0;
    renderMedia();
    refreshDock();
    const tick = setInterval(() => {
      if (item.kind !== 'uploading') { clearInterval(tick); return; }
      item.pct = Math.min(100, item.pct + 14);
      if (item.pct >= 100) {
        item.kind = 'done';
        item.students = [];
        item.suggestions = 0;
        clearInterval(tick);
      }
      renderMedia();
      refreshDock();
    }, 180);
  }

  /* ── 底部栏 ───────────────────────────────────────── */

  function refreshDock() {
    const uploading = state.media.filter((m) => m.kind === 'uploading').length;
    const failed = state.media.filter((m) => m.kind === 'failed').length;
    const reasons = [];
    if (state.mode === 'student' && state.selected.size === 0) reasons.push('请先选择至少一位幼儿');
    if (!contentInput.value.trim()) reasons.push('请输入记录内容');
    if (uploading) reasons.push(`${uploading} 张照片上传中，完成后可发布`);
    if (failed) reasons.push(`有 ${failed} 张照片上传失败，请重试或移除`);

    const disabled = reasons.length > 0 || state.submitting;
    publishBtn.disabled = disabled;
    dockReason.hidden = reasons.length === 0;
    dockReason.textContent = reasons.join(' · ');
    if (state.edit) {
      // 编辑态可见范围只读（源码 ?id= 分支）：只陈述结果，不给开关
      scopeText.textContent = state.mode === 'student'
        ? '该记录仅被选中的幼儿家长可见'
        : '该记录对全班家长可见';
    } else {
      scopeText.textContent = state.mode === 'student'
        ? (state.selected.size ? `仅 ${state.selected.size} 位幼儿家长可见` : '仅选中的幼儿家长可见')
        : '小一班全体家长可见';
    }

    if (state.submitting) {
      publishBtn.textContent = state.edit ? '保存中…' : '发布中…';
    } else if (app.dataset.state === 'error') {
      publishBtn.textContent = state.edit ? '重新保存' : '重新发布';
    } else if (state.edit) {
      publishBtn.textContent = '保存修改';
    } else {
      publishBtn.textContent = state.mode === 'student' ? '发布个体记录' : '发布班级记录';
    }
  }

  contentInput.addEventListener('input', () => {
    charCount.textContent = `${contentInput.value.length} 字`;
    refreshDock();
  });

  syncSw.addEventListener('click', () => {
    const next = syncSw.getAttribute('aria-checked') !== 'true';
    syncSw.setAttribute('aria-checked', String(next));
    showToast(next ? '已开启同步到班级动态' : '已关闭同步到班级动态');
  });

  publishBtn.addEventListener('click', () => {
    if (publishBtn.disabled || state.submitting) return;
    state.submitting = true;
    refreshDock();
    setTimeout(() => {
      state.submitting = false;
      if (app.dataset.state === 'error') {
        setState('normal');
        showToast('已重新发布');
      } else if (state.edit) {
        showToast('已保存修改');
      } else {
        showToast(state.mode === 'student' ? '已发布给 5 位幼儿家长' : '已发布到小一班');
      }
      refreshDock();
    }, 800);
  });

  errRetry.addEventListener('click', () => {
    setState('normal');
    showToast('正在重试…');
    refreshDock();
  });

  /* ── 徽章 ─────────────────────────────────────────── */

  badges.addEventListener('click', (event) => {
    const chip = event.target.closest('.badge-chip');
    if (!chip) return;
    const on = !chip.classList.contains('is-on');
    badges.querySelectorAll('.badge-chip').forEach((el) => {
      el.classList.remove('is-on');
      el.setAttribute('aria-pressed', 'false');
    });
    if (on) {
      chip.classList.add('is-on');
      chip.setAttribute('aria-pressed', 'true');
    }
    grantBox.hidden = !on;
  });

  grantStudents.addEventListener('click', (event) => {
    const chip = event.target.closest('.chip');
    if (!chip) return;
    const on = !chip.classList.contains('is-on');
    chip.classList.toggle('is-on', on);
    chip.setAttribute('aria-pressed', String(on));
  });

  witness.addEventListener('click', () => {
    const next = witness.getAttribute('aria-pressed') !== 'true';
    witness.setAttribute('aria-pressed', String(next));
  });

  /* ── 幼儿弹层 ─────────────────────────────────────── */

  function renderSheetList() {
    const query = searchInput.value.trim();
    const list = query ? ROSTER.filter((n) => n.includes(query)) : ROSTER;
    sheetList.innerHTML = '';
    if (!list.length) {
      const empty = document.createElement('p');
      empty.className = 'sheet-empty';
      empty.textContent = '没有找到匹配的幼儿';
      sheetList.appendChild(empty);
    }
    list.forEach((name) => {
      const row = document.createElement('button');
      row.type = 'button';
      row.className = 'srow';
      row.setAttribute('aria-pressed', String(temp.has(name)));
      row.innerHTML = `<span class="savatar" aria-hidden="true">${name.slice(0, 1)}</span>
        <span class="sname">${name}</span>
        <span class="scheck" aria-hidden="true">✓</span>`;
      row.addEventListener('click', () => {
        if (temp.has(name)) temp.delete(name); else temp.add(name);
        row.setAttribute('aria-pressed', String(temp.has(name)));
        updateSheetFoot();
      });
      sheetList.appendChild(row);
    });
    updateSheetFoot();
  }

  function updateSheetFoot() {
    const all = ROSTER.every((n) => temp.has(n));
    selectAllBtn.textContent = all ? '取消全选' : '全选';
    confirmPick.textContent = sheetMode === 'batch'
      ? `追加到 ${batchSelected.size} 张照片（${temp.size} 人）`
      : `确定（已选 ${temp.size} 人）`;
    sheetDesc.textContent = sheetMode === 'batch'
      ? `勾选的孩子会追加到所选 ${batchSelected.size} 张照片，已有名单保留（新增 ${temp.size} 人）`
      : sheetMode === 'single'
        ? '勾选这张照片里的孩子（AI 建议可一并确认）'
        : `小一班 · 共 ${ROSTER.length} 位 · 已选 ${temp.size} 人`;
  }

  function openSheet(mode) {
    sheetMode = mode;
    if (mode === 'pick') {
      temp = new Set(state.selected);
    } else if (mode === 'single') {
      temp = new Set((state.media[sheetTarget] && state.media[sheetTarget].students) || []);
    } else {
      temp = new Set(); // 批量 = 追加语义，从空开始，不覆盖各自已有点名
    }
    sheetTitle.textContent = mode === 'pick' ? '选择幼儿' : mode === 'single' ? '照片点名' : '批量点名';
    searchInput.value = '';
    renderSheetList();
    sheetLayer.hidden = false;
    document.body.style.overflow = 'hidden';
  }

  function closeSheet() {
    sheetLayer.hidden = true;
    document.body.style.overflow = '';
  }

  editPick.addEventListener('click', () => openSheet('pick'));
  openPickEmpty.addEventListener('click', () => openSheet('pick'));

  /* 点名：批量 = 先勾照片再选学生（对应源码 startBatch → openBatchNaming 两段） */

  function openSingle(index) {
    const item = state.media[index];
    if (!item || item.kind !== 'done') return;
    sheetTarget = index;
    openSheet('single');
  }

  function toggleBatchSelect(index) {
    const item = state.media[index];
    if (!item || item.kind !== 'done') return;
    if (batchSelected.has(index)) batchSelected.delete(index);
    else batchSelected.add(index);
    renderMedia();
  }

  function setBatchMode(on) {
    batchMode = on;
    batchSelected = new Set();
    batchBtn.hidden = on;
    batchTools.hidden = !on;
    renderMedia();
  }

  batchBtn.addEventListener('click', () => {
    if (!state.media.length) { showToast('先添加照片，再点名'); return; }
    if (!doneIndexes().length) { showToast('等照片上传完成，再点名'); return; }
    setBatchMode(true);
  });

  selectAllPhotos.addEventListener('click', () => {
    batchSelected = allPhotosSelected() ? new Set() : new Set(doneIndexes());
    renderMedia();
  });

  pickStudentsBtn.addEventListener('click', () => {
    if (!batchSelected.size) { showToast('请先选择照片'); return; }
    openSheet('batch');
  });

  cancelBatchBtn.addEventListener('click', () => setBatchMode(false));

  searchInput.addEventListener('input', renderSheetList);

  selectAllBtn.addEventListener('click', () => {
    const query = searchInput.value.trim();
    const list = query ? ROSTER.filter((n) => n.includes(query)) : ROSTER;
    const all = list.every((n) => temp.has(n));
    list.forEach((n) => { if (all) temp.delete(n); else temp.add(n); });
    renderSheetList();
  });

  confirmPick.addEventListener('click', () => {
    if (sheetMode === 'pick') {
      state.selected = new Set(temp);
      renderPicked();
    } else if (sheetMode === 'single') {
      // 单张 = 编辑这张照片的完整名单（加/减皆可）
      const item = state.media[sheetTarget];
      if (item) item.students = [...temp];
      renderMedia();
      showToast(temp.size ? `已点名 ${temp.size} 位幼儿` : '已清空这张照片的点名');
    } else {
      // 批量 = 追加：与各照片已有点名取并集，重复应用不叠加
      if (!temp.size) {
        closeSheet();
        showToast('未选择新增孩子，未做改动');
        return;
      }
      const targets = [...batchSelected];
      targets.forEach((index) => {
        const item = state.media[index];
        if (!item) return;
        const merged = new Set(item.students || []);
        temp.forEach((name) => merged.add(name));
        item.students = [...merged];
        item.suggestions = 0;
      });
      setBatchMode(false);
      showToast(`已为 ${targets.length} 张照片追加 ${temp.size} 位幼儿`);
    }
    closeSheet();
    refreshDock();
  });

  sheetLayer.addEventListener('click', (event) => {
    if (event.target.closest('[data-close="sheet"]')) closeSheet();
  });

  /* ── 状态切换 ─────────────────────────────────────── */

  function baseMedia() {
    return [
      { kind: 'done', students: ['乐乐', '小雨', '朵朵', '小满', '果果', '豆豆'], suggestions: 0 },
      { kind: 'done', students: [], suggestions: 3 },
    ];
  }

  function applyState(next) {
    state.edit = next === 'edit';
    if (next === 'empty') {
      state.selected = new Set();
      state.media = [];
      contentInput.value = '';
      setMode('student');
    } else if (next === 'loading') {
      state.media = [...baseMedia(), { kind: 'uploading', pct: 62 }];
      contentInput.value = SAMPLE;
      setMode('class');
    } else if (next === 'error') {
      state.media = baseMedia();
      contentInput.value = SAMPLE;
      setMode('class');
    } else {
      // normal / edit 共用一张表单；编辑态只是日期与可见范围锁定、徽章卡隐藏
      state.media = baseMedia();
      contentInput.value = SAMPLE;
      state.selected = new Set(['乐乐', '小雨', '朵朵', '小满', '果果']);
      setMode('class');
    }
    badgeCard.hidden = state.edit;
    batchMode = false;
    batchSelected = new Set();
    batchBtn.hidden = false;
    batchTools.hidden = true;
    renderDay();
    charCount.textContent = `${contentInput.value.length} 字`;
    renderPicked();
    renderMedia();
    refreshDock();
  }

  document.addEventListener('mp:state', (event) => {
    applyState(event.detail && event.detail.state ? event.detail.state : 'normal');
  });

  /* ── boot ─────────────────────────────────────────── */

  buildTags($('tagsClass'), 'perfClass');
  buildTags($('tagsStudent'), 'perfStudent');
  bindGroups(document.querySelector('.page'));
  renderDay();
  applyState(app.dataset.state || 'normal');

  $('navBack').addEventListener('click', () => showToast('返回成长记录'));
})();
