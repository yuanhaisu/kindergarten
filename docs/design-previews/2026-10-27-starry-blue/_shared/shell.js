/* 养成星球 · 小程序外壳交互（原型专用）
 * 只负责：折叠/展开状态演示条，把选中的状态写到 .mp-app[data-state]。
 * 页面自身交互写在页面自己的 app.js 里。
 */
(() => {
  const app = document.querySelector('.mp-app');
  if (!app) return;

  document.querySelectorAll('.mp-demo').forEach((demo) => {
    const toggle = demo.querySelector('.mp-demo-toggle');
    const setOpen = (open) => {
      demo.dataset.open = String(open);
      if (toggle) toggle.setAttribute('aria-expanded', String(open));
    };
    if (toggle) toggle.addEventListener('click', () => setOpen(demo.dataset.open !== 'true'));

    demo.querySelectorAll('.mp-demo-chip').forEach((chip) => {
      chip.addEventListener('click', () => {
        const state = chip.dataset.mpState || 'normal';
        demo.querySelectorAll('.mp-demo-chip').forEach((c) => {
          c.setAttribute('aria-pressed', String(c === chip));
        });
        app.dataset.state = state;
        document.dispatchEvent(new CustomEvent('mp:state', { detail: { state } }));
      });
    });
  });
})();
