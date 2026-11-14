/* 设置 · 简化提案 — 交互与文档正文 */
(() => {
  'use strict';

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];
  const KEY = 'nurture-parent-settings-v4';

  /* ---------- 设置状态 ---------- */
  const state = { school: true, teacher: true, quiet: true, quietStart: '20:00', quietEnd: '07:30' };
  try {
    const saved = JSON.parse(localStorage.getItem(KEY) || 'null');
    if (saved && typeof saved === 'object') {
      ['school', 'teacher', 'quiet'].forEach((k) => {
        if (typeof saved[k] === 'boolean') state[k] = saved[k];
      });
      ['quietStart', 'quietEnd'].forEach((k) => {
        if (/^\d{2}:\d{2}$/.test(saved[k])) state[k] = saved[k];
      });
    }
  } catch (_) { /* 隐私模式下不持久化 */ }
  const persist = () => {
    try { localStorage.setItem(KEY, JSON.stringify(state)); } catch (_) { /* 忽略 */ }
  };

  const app = $('#app');
  const settingsView = $('#view-settings');
  const docView = $('#view-doc');
  const docMain = $('#doc-main');
  const layer = $('#sheet-layer');
  const toastEl = $('#toast');
  const schoolTag = $('#row-school-tag');
  const teacherTag = $('#row-teacher-tag');
  const quietValue = $('#row-quiet-value');
  const updateStatus = $('#update-status');

  const paintSettings = () => {
    [[schoolTag, state.school], [teacherTag, state.teacher]].forEach(([el, on]) => {
      el.textContent = on ? '已开启' : '已关闭';
      el.classList.toggle('state-tag--off', !on);
    });
    quietValue.textContent = state.quiet ? `${state.quietStart}–${state.quietEnd}` : '未开启';
  };
  paintSettings();

  /* ---------- 轻提示 ---------- */
  let toastTimer = 0;
  const toast = (message) => {
    toastEl.textContent = message;
    toastEl.classList.add('is-visible');
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => toastEl.classList.remove('is-visible'), 2000);
  };
  const hideToast = () => {
    clearTimeout(toastTimer);
    toastEl.classList.remove('is-visible');
  };

  /* ---------- 底部面板 ---------- */
  const sheets = $$('.sheet');
  let currentSheet = null;
  let lastFocus = null;
  let closeTimer = 0;

  const setInert = (on) => {
    if ('inert' in settingsView) {
      settingsView.inert = on;
      docView.inert = on;
    }
  };

  const openSheet = (sheet, trigger) => {
    if (!sheet) return;
    hideToast();
    clearTimeout(closeTimer);
    lastFocus = trigger || document.activeElement;
    sheets.forEach((s) => { if (s !== sheet) s.hidden = true; });
    sheet.hidden = false;
    layer.hidden = false;
    app.classList.add('is-sheet-open');
    setInert(true);
    void layer.offsetWidth;
    layer.classList.add('is-open');
    currentSheet = sheet;
    sheet.focus({ preventScroll: true });
  };

  const closeSheet = (restore = true, done = null) => {
    if (!currentSheet) return;
    const sheet = currentSheet;
    currentSheet = null;
    layer.classList.remove('is-open');
    app.classList.remove('is-sheet-open');
    setInert(false);
    clearTimeout(closeTimer);
    closeTimer = setTimeout(() => {
      sheet.hidden = true;
      layer.hidden = true;
      if (restore && lastFocus) lastFocus.focus({ preventScroll: true });
      if (done) done();
    }, 210);
  };

  document.addEventListener('keydown', (e) => {
    if (!currentSheet) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      closeSheet();
      return;
    }
    if (e.key !== 'Tab') return;
    const focusables = $$('button:not([disabled]),input:not([disabled]),[href],[tabindex]:not([tabindex="-1"])', currentSheet)
      .filter((el) => el.offsetParent !== null);
    if (!focusables.length) {
      e.preventDefault();
      currentSheet.focus();
      return;
    }
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    if (e.shiftKey && (document.activeElement === first || document.activeElement === currentSheet)) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  });

  /* ---------- 消息通知 ---------- */
  const notifySheet = $('#sheet-notify');
  const notifyTitle = $('#notify-sheet-title');
  const notifyDesc = $('#notify-sheet-desc');
  const notifyLabel = $('#notify-switch-label');
  const notifySwitch = $('#notify-switch');
  const NOTIFY_COPY = {
    school: {
      title: '园所通知',
      desc: '园所公告、活动与审批结果会通过微信服务通知送达。关闭后仍可在消息页查看。',
      label: '接收园所通知'
    },
    teacher: {
      title: '老师消息',
      desc: '老师的私信与回复会通过微信服务通知提醒。关闭后仍可在消息页查看。',
      label: '接收老师消息'
    }
  };
  let notifyKind = 'school';
  let notifyDraft = true;

  const openNotify = (kind, trigger) => {
    notifyKind = kind;
    notifyDraft = state[kind];
    const copy = NOTIFY_COPY[kind];
    notifyTitle.textContent = copy.title;
    notifyDesc.textContent = copy.desc;
    notifyLabel.textContent = copy.label;
    notifySwitch.setAttribute('aria-checked', String(notifyDraft));
    openSheet(notifySheet, trigger);
  };
  notifySwitch.addEventListener('click', () => {
    notifyDraft = !notifyDraft;
    notifySwitch.setAttribute('aria-checked', String(notifyDraft));
  });
  $('#notify-save').addEventListener('click', () => {
    state[notifyKind] = notifyDraft;
    persist();
    paintSettings();
    closeSheet(true, () => toast(notifyDraft ? '通知已开启' : '通知已关闭'));
  });

  /* ---------- 免打扰时段 ---------- */
  const quietSheet = $('#sheet-quiet');
  const quietSwitch = $('#quiet-switch');
  const startInput = $('#quiet-start');
  const endInput = $('#quiet-end');
  const quietError = $('#quiet-error');
  const presets = $$('.preset', $('#quiet-presets'));
  let quietDraft = true;

  const paintQuiet = () => {
    quietSwitch.setAttribute('aria-checked', String(quietDraft));
    startInput.disabled = !quietDraft;
    endInput.disabled = !quietDraft;
    presets.forEach((p) => {
      p.disabled = !quietDraft;
      p.setAttribute('aria-pressed', String(
        p.dataset.start === startInput.value && p.dataset.end === endInput.value
      ));
    });
  };
  const openQuiet = (trigger) => {
    quietDraft = state.quiet;
    startInput.value = state.quietStart;
    endInput.value = state.quietEnd;
    quietError.hidden = true;
    startInput.removeAttribute('aria-invalid');
    endInput.removeAttribute('aria-invalid');
    paintQuiet();
    openSheet(quietSheet, trigger);
  };
  const quietErrorShow = (message, input) => {
    quietError.textContent = message;
    quietError.hidden = false;
    input.setAttribute('aria-invalid', 'true');
    input.focus({ preventScroll: true });
  };
  quietSwitch.addEventListener('click', () => {
    quietDraft = !quietDraft;
    quietError.hidden = true;
    paintQuiet();
  });
  [startInput, endInput].forEach((input) => {
    input.addEventListener('input', () => {
      quietError.hidden = true;
      input.removeAttribute('aria-invalid');
      paintQuiet();
    });
  });
  presets.forEach((p) => {
    p.addEventListener('click', () => {
      if (p.disabled) return;
      startInput.value = p.dataset.start;
      endInput.value = p.dataset.end;
      quietError.hidden = true;
      paintQuiet();
    });
  });
  $('#quiet-save').addEventListener('click', () => {
    const start = startInput.value;
    const end = endInput.value;
    if (quietDraft && (!start || !end)) {
      quietErrorShow('请选择开始和结束时间', !start ? startInput : endInput);
      return;
    }
    if (quietDraft && start === end) {
      quietErrorShow('开始和结束时间不能相同', startInput);
      return;
    }
    state.quiet = quietDraft;
    if (start && end) {
      state.quietStart = start;
      state.quietEnd = end;
    }
    persist();
    paintSettings();
    closeSheet(true, () => toast(quietDraft ? `免打扰已设为 ${state.quietStart}–${state.quietEnd}` : '免打扰已关闭'));
  });

  /* ---------- 注销账号 ---------- */
  const ack = $('#delete-ack');
  const deleteSubmit = $('#delete-submit');
  const deleteStatus = $('#delete-status');
  const resetDelete = () => {
    ack.checked = false;
    deleteSubmit.disabled = true;
    deleteStatus.hidden = true;
  };
  ack.addEventListener('change', () => {
    deleteSubmit.disabled = !ack.checked;
  });
  deleteSubmit.addEventListener('click', () => {
    if (!ack.checked) return;
    deleteStatus.textContent = 'Phase 2 接口未接入 · 本次未提交';
    deleteStatus.hidden = false;
    deleteSubmit.disabled = true;
  });

  /* ---------- 检查更新 ---------- */
  let updateTimer = 0;
  const checkUpdate = (row) => {
    if (row.disabled) return;
    row.disabled = true;
    updateStatus.textContent = '正在检查';
    clearTimeout(updateTimer);
    updateTimer = setTimeout(() => {
      updateStatus.textContent = '已是最新';
      row.disabled = false;
      if (!currentSheet && docView.hidden) toast('已是最新版本');
    }, 1400);
  };
  updateStatus.setAttribute('aria-live', 'polite');

  /* ---------- 文档 ---------- */
  const DOCS = {
    policy: {
      title: '隐私政策',
      meta: '更新于 2026 年 11 月 1 日 · 生效于 2026 年 11 月 8 日',
      lede: '我们尽量说得简单：哪些信息会被收集、用在哪里、保存多久，以及你有哪些选择。',
      sections: [
        {
          t: '我们收集哪些信息',
          p: [
            '为了完成家园沟通，我们会收集三类信息：你主动填写的账号信息，例如手机号与孩子的关系；园所提供的孩子信息，例如姓名、班级、入园日期与接送授权人；使用过程中产生的记录，例如通知已读、请假、接送与成长记录。',
            '我们不会收集与上述目的无关的信息，也<span class="hl">不会读取你的通讯录、相册或位置</span>。'
          ]
        },
        {
          t: '我们如何使用',
          p: [
            '这些信息只用于四件事：把园所通知送到你手上、核验接送人、记录孩子的在园成长，以及在必要时与你联系。',
            '我们<span class="hl">不会把信息用于广告投放</span>，也不会用于与家园沟通无关的推荐。'
          ]
        },
        {
          t: '存储与保护',
          p: [
            '信息存储在中国境内的服务器，传输过程使用加密连接。园所与平台按最小必要原则分配查看权限，操作记录可追溯。',
            '账号信息在孩子离园后<span class="hl">保存 3 年</span>，到期后删除或匿名化处理；法律法规另有要求的，按法定期限保存。'
          ]
        },
        {
          t: '共享与披露',
          p: [
            '我们<span class="hl">不出售、不出租</span>你和孩子的信息。仅在两种情况下共享：一是向孩子所在园所提供其教学与照护必需的信息；二是应司法机关或法律法规的明确要求。',
            '使用第三方服务时，我们只提供完成服务所需的最少信息，并要求其承担保密义务。'
          ]
        },
        {
          t: '家长的权利',
          p: [
            '你可以查看、更正孩子的信息，撤回对未成年人信息处理的授权，或申请注销账号。撤回授权后，与孩子相关的家园沟通功能将不可用。',
            '在「我的 · 隐私与协议」中可以随时查看本政策，也可以直接联系我们提出请求。'
          ]
        },
        {
          t: '联系我们',
          p: [
            '如对本政策有疑问，或希望行使上述权利，请发送邮件至 <span class="mail">contact@nurtureplanet.cn</span>，或通过「意见反馈」联系我们。',
            '我们会在收到请求后的<span class="hl hl--marigold">15 个工作日内答复</span>。'
          ]
        }
      ],
      foot: '本政策适用于养成星球家长端。园所另有约定的，以园所与家长签署的入园协议为准。'
    },
    terms: {
      title: '用户协议',
      meta: '更新于 2026 年 11 月 1 日 · 生效于 2026 年 11 月 8 日',
      lede: '这份约定说明你可以怎样使用养成星球，以及我们在服务中各自承担的责任。',
      sections: [
        {
          t: '服务范围',
          p: [
            '养成星球家长端由平台提供技术服务，园所通过它发布通知、处理请假与接送，并记录孩子的在园成长。具体教学与照护安排由园所负责。',
            '家长端面向已入园孩子的家长与监护人，<span class="hl">不向未满 14 周岁的孩子单独开放</span>。'
          ]
        },
        {
          t: '账号与使用',
          p: [
            '请使用本人手机号登录，并只邀请已获授权的家庭成员共同使用。账号仅限本人使用，不得转让、出借或用于与家园沟通无关的用途。',
            '发现异常登录或账号被他人使用时，请立即联系班主任，或通过「意见反馈」告知我们。'
          ]
        },
        {
          t: '内容与行为',
          p: [
            '园所发布的通知与孩子的在园记录，归园所与相应家庭所有。未经同意，<span class="hl">请勿截图、转发或公开</span>涉及其他孩子与家庭的内容。',
            '不得利用本服务发布违法、侵权、骚扰或干扰园所正常教学的信息。'
          ]
        },
        {
          t: '服务变更与中断',
          p: [
            '因系统维护、升级或不可抗力，服务可能短暂中断，我们会尽量提前告知。园所停止使用本服务时，相关功能会随之关闭。',
            '我们可能根据业务与法规变化更新本协议；涉及你重要权益的变更，会通过园所通知或页面提示说明。'
          ]
        },
        {
          t: '注销与终止',
          p: [
            '你可以随时申请注销账号。注销后，账号数据与孩子的在园记录会按《隐私政策》删除或匿名化，且无法恢复。',
            '如你违反本协议或法律法规，我们可暂停或终止提供服务，并依法保留相关记录。'
          ]
        },
        {
          t: '联系我们',
          p: [
            '如对本协议有疑问，请发送邮件至 <span class="mail">contact@nurtureplanet.cn</span>，或通过「意见反馈」联系我们。',
            '我们会在收到请求后的<span class="hl hl--marigold">15 个工作日内答复</span>。'
          ]
        }
      ],
      foot: '本协议与《隐私政策》共同构成你与养成星球之间的约定。'
    }
  };

  let docFromApp = false;
  let lastDocTrigger = null;

  const renderDoc = (kind) => {
    const doc = DOCS[kind];
    docMain.innerHTML = `
      <header class="doc-head">
        <h1 class="doc-title" id="doc-title">${doc.title}</h1>
        <p class="doc-meta">${doc.meta}</p>
        <p class="doc-lede">${doc.lede}</p>
      </header>
      ${doc.sections.map((section, index) => `
        <section class="doc-section">
          <div class="section-head"><span class="section-no">${String(index + 1).padStart(2, '0')}</span></div>
          <h2 class="section-title">${section.t}</h2>
          ${section.p.map((paragraph) => `<p>${paragraph}</p>`).join('')}
        </section>`).join('')}
      <footer class="doc-foot"><p>${doc.foot}</p></footer>`;
    docMain.scrollTop = 0;
    docMain.tabIndex = -1;
    docMain.focus({ preventScroll: true });
    document.title = `${doc.title} · 养成星球`;
  };

  const route = () => {
    const kind = location.hash === '#/policy' ? 'policy'
      : location.hash === '#/terms' ? 'terms' : null;
    if (kind) {
      hideToast();
      if (docView.hidden) {
        settingsView.hidden = true;
        docView.hidden = false;
      }
      renderDoc(kind);
    } else {
      docFromApp = false;
      if (!docView.hidden) {
        docView.hidden = true;
        settingsView.hidden = false;
        docMain.innerHTML = '';
        document.title = '设置 · 养成星球';
        const trigger = lastDocTrigger;
        lastDocTrigger = null;
        if (trigger) trigger.focus({ preventScroll: true });
      }
    }
  };

  const openDoc = (kind, trigger) => {
    lastDocTrigger = trigger || null;
    docFromApp = true;
    const target = `#/${kind}`;
    if (location.hash === target) route();
    else location.hash = target;
  };

  const backFromDoc = () => {
    if (docFromApp && history.length > 1) {
      history.back();
    } else {
      history.replaceState(null, '', location.pathname + location.search);
      route();
    }
  };

  /* ---------- 动作分发 ---------- */
  document.addEventListener('click', (e) => {
    const target = e.target.closest('[data-action]');
    if (!target) return;
    switch (target.dataset.action) {
      case 'back-settings':
        toast('已在设置首页');
        break;
      case 'back-doc':
        backFromDoc();
        break;
      case 'close-sheet':
        closeSheet();
        break;
      case 'notify-school':
        openNotify('school', target);
        break;
      case 'notify-teacher':
        openNotify('teacher', target);
        break;
      case 'quiet':
        openQuiet(target);
        break;
      case 'child':
        openSheet($('#sheet-child'), target);
        break;
      case 'delete':
        resetDelete();
        openSheet($('#sheet-delete'), target);
        break;
      case 'logout':
        openSheet($('#sheet-logout'), target);
        break;
      case 'policy':
        openDoc('policy', target);
        break;
      case 'terms':
        openDoc('terms', target);
        break;
      case 'open-policy':
        closeSheet(false);
        openDoc('policy', $('[data-action="policy"]'));
        break;
      case 'check-update':
        checkUpdate(target);
        break;
      default:
        break;
    }
  });

  $('#logout-confirm').addEventListener('click', () => {
    closeSheet(true, () => toast('已退出登录'));
  });

  window.addEventListener('hashchange', route);
  route();
})();
