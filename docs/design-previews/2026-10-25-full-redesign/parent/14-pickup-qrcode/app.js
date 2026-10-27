/* 接送二维码 · 页面交互（原型）
 * 二维码为确定性伪随机绘制的装饰图形，不可扫描。
 */
(() => {
  'use strict';

  const app = document.querySelector('.mp-app');
  const page = document.querySelector('.page');
  const qrSvg = document.querySelector('.qr-svg');
  const backBtn = document.querySelector('.mp-back');
  const refreshBtn = document.getElementById('refreshQr');
  const changeBtn = document.getElementById('changePerson');
  const designateBtn = document.getElementById('designatePerson');
  const designateFromError = document.getElementById('designateFromError');
  const retryBtn = document.getElementById('retryLoad');
  const actionStack = document.querySelector('.action-stack');
  const toast = document.getElementById('toast');
  const chips = Array.from(document.querySelectorAll('.mp-demo-chip'));

  if (!app || !page || !qrSvg || !refreshBtn || !retryBtn || !toast) return;

  const SVG_NS = 'http://www.w3.org/2000/svg';
  const STATES = ['normal', 'expired', 'empty', 'loading', 'error'];
  const REFRESH_LABEL = '刷新二维码';
  const RETRY_LABEL = '重试';
  const QR_SEED = 20261026;

  let refreshTimer = 0;
  let retryTimer = 0;
  let toastTimer = 0;

  /* 工具 ----------------------------------------------------------------- */

  function svgNode(tag, attrs) {
    const node = document.createElementNS(SVG_NS, tag);
    Object.keys(attrs).forEach((key) => node.setAttribute(key, attrs[key]));
    return node;
  }

  function setLabel(button, text) {
    const label = button && button.querySelector('.btn-label');
    if (label) label.textContent = text;
  }

  function makeRandom(seed) {
    let state = seed >>> 0;
    return function random() {
      state = (Math.imul(state, 1664525) + 1013904223) >>> 0;
      return state / 4294967296;
    };
  }

  /* 二维码占位 ----------------------------------------------------------- */

  function renderQR() {
    const random = makeRandom(QR_SEED);
    qrSvg.textContent = '';

    qrSvg.appendChild(
      svgNode('rect', {
        class: 'qr-base',
        x: 0,
        y: 0,
        width: 200,
        height: 200
      })
    );

    const code = svgNode('g', { class: 'qr-code' });

    function addFinder(x, y) {
      code.appendChild(
        svgNode('rect', { class: 'qr-dark', x: x, y: y, width: 70, height: 70 })
      );
      code.appendChild(
        svgNode('rect', { class: 'qr-base', x: x + 10, y: y + 10, width: 50, height: 50 })
      );
      code.appendChild(
        svgNode('rect', { class: 'qr-dark', x: x + 20, y: y + 20, width: 30, height: 30 })
      );
    }

    addFinder(0, 0);
    addFinder(130, 0);
    addFinder(0, 130);

    for (let row = 0; row < 20; row += 1) {
      for (let col = 0; col < 20; col += 1) {
        const value = random();
        const inFinder =
          (col <= 7 && row <= 7) ||
          (col >= 12 && row <= 7) ||
          (col <= 7 && row >= 12);
        const x = col * 10 + 5;
        const y = row * 10 + 5;
        const nearLogo = Math.hypot(x - 100, y - 100) < 27;
        if (inFinder || nearLogo) continue;
        if (value > 0.46) continue;
        code.appendChild(
          svgNode('rect', {
            class: 'qr-mod qr-dark',
            x: col * 10,
            y: row * 10,
            width: 10,
            height: 10
          })
        );
      }
    }

    qrSvg.appendChild(code);

    const logo = svgNode('g', { class: 'qr-logo' });
    logo.appendChild(
      svgNode('circle', {
        class: 'qr-logo-plate',
        cx: 100,
        cy: 100,
        r: 20
      })
    );

    const mark = svgNode('g', { transform: 'translate(100 100)' });
    mark.appendChild(
      svgNode('circle', { class: 'qr-logo-planet', cx: 0, cy: 0, r: 5.2 })
    );
    mark.appendChild(
      svgNode('ellipse', {
        class: 'qr-logo-ring',
        cx: 0,
        cy: 0,
        rx: 9.4,
        ry: 3.4,
        transform: 'rotate(-18)'
      })
    );
    logo.appendChild(mark);
    qrSvg.appendChild(logo);
  }

  /* 状态 ----------------------------------------------------------------- */

  function syncChips(state) {
    chips.forEach((chip) => {
      chip.setAttribute(
        'aria-pressed',
        String(chip.getAttribute('data-mp-state') === state)
      );
    });
  }

  function resetPending() {
    window.clearTimeout(refreshTimer);
    window.clearTimeout(retryTimer);
    refreshTimer = 0;
    retryTimer = 0;

    page.setAttribute('data-refreshing', 'false');

    refreshBtn.disabled = false;
    refreshBtn.removeAttribute('aria-busy');
    setLabel(refreshBtn, REFRESH_LABEL);

    retryBtn.disabled = false;
    retryBtn.removeAttribute('aria-busy');
    setLabel(retryBtn, RETRY_LABEL);

    if (actionStack) actionStack.removeAttribute('inert');
  }

  function applyState(state) {
    const next = STATES.indexOf(state) >= 0 ? state : 'normal';
    resetPending();
    app.setAttribute('data-state', next);
    syncChips(next);
    if (actionStack) actionStack.toggleAttribute('inert', next === 'loading');
  }

  function showToast(message) {
    toast.textContent = message;
    toast.classList.add('is-visible');
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
      toastTimer = 0;
    }, 2000);
  }

  /* 返回：原型不真跳页 ----------------------------------------------------- */

  if (backBtn) {
    backBtn.addEventListener('click', () => {
      showToast('返回接送记录');
    });
  }

  /* 刷新 ----------------------------------------------------------------- */

  refreshBtn.addEventListener('click', () => {
    const state = app.getAttribute('data-state');
    if (refreshBtn.disabled) return;
    if (state !== 'normal' && state !== 'expired') return;

    window.clearTimeout(refreshTimer);
    page.setAttribute('data-refreshing', 'true');
    refreshBtn.disabled = true;
    refreshBtn.setAttribute('aria-busy', 'true');
    setLabel(refreshBtn, '刷新中…');

    refreshTimer = window.setTimeout(() => {
      refreshTimer = 0;
      renderQR();
      applyState('normal');
      showToast('二维码已刷新');
    }, 1400);
  });

  /* 更换 / 指定接走人 ----------------------------------------------------- */

  [changeBtn, designateBtn, designateFromError].forEach((button) => {
    if (!button) return;
    button.addEventListener('click', () => {
      showToast('已打开授权接送人');
    });
  });

  /* 重试 ----------------------------------------------------------------- */

  retryBtn.addEventListener('click', () => {
    if (retryBtn.disabled) return;

    window.clearTimeout(retryTimer);
    retryBtn.disabled = true;
    retryBtn.setAttribute('aria-busy', 'true');
    setLabel(retryBtn, '重试中…');

    retryTimer = window.setTimeout(() => {
      retryTimer = 0;
      applyState('normal');
      showToast('已重新加载');
    }, 1000);
  });

  /* 演示芯片联动 ---------------------------------------------------------- */

  document.addEventListener('mp:state', (event) => {
    const state = event && event.detail ? event.detail.state : '';
    applyState(state);
  });

  /* 初始化 ---------------------------------------------------------------- */

  renderQR();
  applyState(app.getAttribute('data-state') || 'normal');
})();
