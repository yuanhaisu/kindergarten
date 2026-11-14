(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  if (!app) return;

  const MONTHS = ['2026-08', '2026-09', '2026-10', '2026-11', '2026-12'];

  const monthTitle = document.getElementById('monthTitle');
  const monthCount = document.getElementById('monthCount');
  const monthDot = document.getElementById('monthDot');
  const monthRange = document.getElementById('monthRange');
  const monthLive = document.getElementById('monthLive');
  const prevBtn = document.getElementById('prevMonth');
  const nextBtn = document.getElementById('nextMonth');
  const panels = Array.from(document.querySelectorAll('[data-month-panel]'));
  const monthEmpty = document.querySelector('[data-month-empty]');
  const prevFromEmpty = document.getElementById('prevFromEmpty');
  const backBtn = document.getElementById('backBtn');
  const emptyRefresh = document.getElementById('emptyRefresh');
  const retryBtn = document.getElementById('retryBtn');
  const retryLabel = document.getElementById('retryLabel');
  const toast = document.getElementById('toast');

  const pv = document.getElementById('preview');
  const pvSheet = document.getElementById('previewSheet');
  const pvTitle = document.getElementById('pvTitle');
  const pvArt = document.getElementById('pvArt');
  const pvStage = document.getElementById('pvStage');
  const pvPrev = document.getElementById('pvPrev');
  const pvNext = document.getElementById('pvNext');
  const pvMeta = document.getElementById('pvMeta');

  const initialPanel = panels.find((panel) => !panel.hidden);
  const initialKey = initialPanel ? initialPanel.dataset.monthPanel : '2026-10';
  let monthIndex = Math.max(0, MONTHS.indexOf(initialKey));

  let previewOpen = false;
  let previewPhotos = [];
  let previewIndex = 0;
  let previewDate = '';
  let previewTrigger = null;

  let toastTimer = 0;
  let toastClearTimer = 0;
  let retryTimer = 0;

  const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)');

  function formatMonth(key) {
    const parts = key.split('-');
    return `${parts[0]}年${Number(parts[1])}月`;
  }

  function getPanel(key) {
    return panels.find((panel) => panel.dataset.monthPanel === key) || null;
  }

  function monthInfo(key) {
    const panel = getPanel(key);
    if (!panel) return { count: 0, range: '' };
    const days = Array.from(panel.querySelectorAll('.day-date'));
    const count = panel.querySelectorAll('.ph').length;
    const range = days.length > 1
      ? `${days[days.length - 1].textContent.trim()} - ${days[0].textContent.trim()}`
      : (days[0] ? days[0].textContent.trim() : '');
    return { count, range };
  }

  function setArrow(button, disabled) {
    if (!button) return;
    button.disabled = disabled;
    button.setAttribute('aria-disabled', String(disabled));
  }

  function syncArrows() {
    const state = app.dataset.state || 'normal';
    // 整页空态（还没有照片）没有月份可翻：月份条已隐藏，箭头同步冻结
    const locked = state === 'loading' || state === 'error' || state === 'empty';
    setArrow(prevBtn, locked || monthIndex <= 0);
    setArrow(nextBtn, locked || monthIndex >= MONTHS.length - 1);
  }

  function renderMonth(announce) {
    const key = MONTHS[monthIndex];
    const info = monthInfo(key);
    const hasRange = info.count > 0 && Boolean(info.range);

    if (monthTitle) monthTitle.textContent = formatMonth(key);
    if (monthCount) monthCount.textContent = `${info.count} 张照片`;
    if (monthRange) {
      monthRange.textContent = info.range;
      monthRange.hidden = !hasRange;
    }
    if (monthDot) monthDot.hidden = !hasRange;

    panels.forEach((panel) => {
      panel.hidden = panel.dataset.monthPanel !== key;
    });

    if (monthEmpty) monthEmpty.hidden = info.count > 0;
    if (prevFromEmpty) prevFromEmpty.hidden = monthIndex <= 0;

    if (announce && monthLive) {
      monthLive.textContent = info.count
        ? `${formatMonth(key)}，共 ${info.count} 张照片`
        : `${formatMonth(key)}，本月暂无照片`;
    }

    syncArrows();
  }

  function goMonth(delta, options = {}) {
    // 整页空态下没有月份轴（monthbar 隐藏），键盘 / 程序化换月一律拦下
    if ((app.dataset.state || 'normal') === 'empty') return;

    const target = Math.max(0, Math.min(MONTHS.length - 1, monthIndex + delta));
    if (target === monthIndex) return;

    monthIndex = target;
    renderMonth(!options.toast);

    if (options.toast) {
      showToast(`已切换到 ${formatMonth(MONTHS[monthIndex])}`);
    }

    window.scrollTo({ top: 0, left: 0, behavior: 'auto' });
  }

  function showToast(message) {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    window.clearTimeout(toastClearTimer);
    toast.classList.remove('is-visible');
    toast.textContent = '';
    void toast.offsetWidth;
    toast.textContent = message;
    toast.classList.add('is-visible');

    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastClearTimer = window.setTimeout(() => {
        toast.textContent = '';
      }, 320);
    }, 2000);
  }

  function setDemoState(state) {
    const chip = document.querySelector(`.mp-demo-chip[data-mp-state="${state}"]`);
    if (chip) chip.click();

    if (app.dataset.state !== state) {
      app.dataset.state = state;
      document.dispatchEvent(new CustomEvent('mp:state', { detail: { state } }));
    }
  }

  function resetRetryButton() {
    if (!retryBtn) return;
    retryBtn.disabled = false;
    retryBtn.removeAttribute('aria-busy');
    retryBtn.classList.remove('is-busy');
    if (retryLabel) retryLabel.textContent = '重试';
  }

  function updatePreview(animate) {
    const total = previewPhotos.length;
    if (!total) return;

    const photo = previewPhotos[previewIndex];
    const tone = photo.dataset.tone || 'sky';
    const comp = photo.classList.contains('comp-2')
      ? 2
      : photo.classList.contains('comp-3')
        ? 3
        : 1;

    if (pvTitle) {
      pvTitle.textContent = `${previewDate} · 第 ${previewIndex + 1} / ${total} 张`;
    }

    if (pvArt) {
      pvArt.className = `pv-art art art--${tone} comp-${comp}`;
      pvArt.setAttribute('aria-label', `${previewDate}第 ${previewIndex + 1} 张照片`);
    }

    const multiple = total > 1;
    if (pvPrev) pvPrev.hidden = !multiple;
    if (pvNext) pvNext.hidden = !multiple;
    if (pvStage) pvStage.classList.toggle('is-single', !multiple);
    if (pvMeta) {
      pvMeta.textContent = multiple
        ? `共 ${total} 张 · 左右切换查看`
        : `共 ${total} 张`;
    }

    if (animate && pvArt && !reduceMotion.matches) {
      void pvArt.offsetWidth;
      pvArt.classList.add('is-swap');
    }
  }

  function openPreview(trigger) {
    const group = trigger.closest('.day');
    if (!group) return;

    const photos = Array.from(group.querySelectorAll('.ph'));
    const index = photos.indexOf(trigger);
    if (index < 0) return;

    const dateEl = group.querySelector('.day-date');
    previewDate = dateEl ? dateEl.textContent.trim() : '';
    previewPhotos = photos;
    previewIndex = index;
    previewTrigger = trigger;
    previewOpen = true;

    updatePreview(false);
    if (pv) {
      pv.classList.add('is-open');
      pv.setAttribute('aria-hidden', 'false');
    }
    document.body.classList.add('is-locked');
    if (pvSheet) pvSheet.focus({ preventScroll: true });
  }

  function stepPreview(delta) {
    const total = previewPhotos.length;
    if (total < 2) return;
    previewIndex = (previewIndex + delta + total) % total;
    updatePreview(true);
  }

  function closePreview(restoreFocus) {
    if (!previewOpen) return;
    previewOpen = false;

    if (pv) {
      pv.classList.remove('is-open');
      pv.setAttribute('aria-hidden', 'true');
    }
    document.body.classList.remove('is-locked');

    const trigger = previewTrigger;
    previewTrigger = null;
    previewPhotos = [];
    previewIndex = 0;

    if (restoreFocus && trigger && trigger.isConnected && trigger.offsetParent !== null) {
      trigger.focus({ preventScroll: true });
      return;
    }

    const chip = document.querySelector('.mp-demo-chip[aria-pressed="true"]');
    if (chip) chip.focus({ preventScroll: true });
  }

  if (prevBtn) prevBtn.addEventListener('click', () => goMonth(-1));
  if (nextBtn) nextBtn.addEventListener('click', () => goMonth(1));

  if (prevFromEmpty) {
    prevFromEmpty.addEventListener('click', () => goMonth(-1, { toast: true }));
  }

  if (backBtn) {
    backBtn.addEventListener('click', () => showToast('已返回成长时光'));
  }

  if (emptyRefresh) {
    emptyRefresh.addEventListener('click', () => {
      setDemoState('normal');
      showToast('已刷新');
    });
  }

  if (retryBtn) {
    retryBtn.addEventListener('click', () => {
      if (retryBtn.disabled) return;
      window.clearTimeout(retryTimer);
      retryBtn.disabled = true;
      retryBtn.setAttribute('aria-busy', 'true');
      retryBtn.classList.add('is-busy');
      if (retryLabel) retryLabel.textContent = '重试中…';

      retryTimer = window.setTimeout(() => {
        retryTimer = 0;
        resetRetryButton();
        if (app.dataset.state === 'error') {
          setDemoState('normal');
          showToast('已重新加载');
        }
      }, 900);
    });
  }

  document.querySelectorAll('.ph').forEach((photo) => {
    photo.addEventListener('click', () => openPreview(photo));
  });

  if (pvPrev) pvPrev.addEventListener('click', () => stepPreview(-1));
  if (pvNext) pvNext.addEventListener('click', () => stepPreview(1));

  document.querySelectorAll('[data-close="preview"]').forEach((element) => {
    element.addEventListener('click', () => closePreview(true));
  });

  document.addEventListener('keydown', (event) => {
    if (!previewOpen) return;

    if (event.key === 'Escape') {
      event.preventDefault();
      closePreview(true);
      return;
    }

    if (event.key !== 'Tab') return;

    const items = [pvPrev, pvNext, document.getElementById('pvClose')]
      .filter((item) => item && !item.hidden);
    if (!items.length) {
      event.preventDefault();
      if (pvSheet) pvSheet.focus();
      return;
    }

    const first = items[0];
    const last = items[items.length - 1];
    const active = document.activeElement;
    const inside = pvSheet && active && pvSheet.contains(active);

    if (!inside) {
      event.preventDefault();
      (event.shiftKey ? last : first).focus();
      return;
    }

    if (event.shiftKey) {
      if (active === first || active === pvSheet) {
        event.preventDefault();
        last.focus();
      }
    } else if (active === last) {
      event.preventDefault();
      first.focus();
    }
  });

  document.addEventListener('mp:state', () => {
    const state = app.dataset.state || 'normal';
    syncArrows();

    if (state !== 'normal' && previewOpen) closePreview(false);

    if (state !== 'error') {
      window.clearTimeout(retryTimer);
      retryTimer = 0;
      resetRetryButton();
    }
  });

  renderMonth(false);
})();
