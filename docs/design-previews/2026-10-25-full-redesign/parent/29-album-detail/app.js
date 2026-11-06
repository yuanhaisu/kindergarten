// 相册详情：返回 + 照片轻提示。四态由 _shared/shell.js 的演示 chip 接管。
(function () {
  const back = document.getElementById('backButton');
  if (back) back.addEventListener('click', () => history.back());

  document.querySelectorAll('.photo-tile').forEach((tile) => {
    tile.addEventListener('click', () => {
      const label = tile.getAttribute('aria-label') || '照片';
      tile.animate(
        [{ transform: 'scale(1)' }, { transform: 'scale(.96)' }, { transform: 'scale(1)' }],
        { duration: 180, easing: 'ease-out' },
      );
      tile.title = label;
    });
  });
})();
