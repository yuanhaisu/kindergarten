// 班级相册列表：范围切换 + 返回。四态由 _shared/shell.js 的演示 chip 接管。
(function () {
  const back = document.getElementById('backButton');
  if (back) back.addEventListener('click', () => history.back());

  document.querySelectorAll('.scope-btn').forEach((button) => {
    button.addEventListener('click', () => {
      document.querySelectorAll('.scope-btn').forEach((item) => {
        const active = item === button;
        item.classList.toggle('is-active', active);
        item.setAttribute('aria-selected', String(active));
      });
    });
  });

  const more = document.querySelector('.load-more-btn');
  if (more) {
    more.addEventListener('click', () => {
      more.textContent = '已经到底了';
      more.disabled = true;
      more.style.opacity = '.6';
    });
  }
})();
