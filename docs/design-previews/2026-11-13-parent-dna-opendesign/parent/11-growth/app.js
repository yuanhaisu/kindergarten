(function () {
  'use strict';

  var root = document.documentElement;

  try {
    var params = new URLSearchParams(window.location.search);
    if (params.get('state') === 'empty') {
      root.setAttribute('data-view', 'empty');
    }
  } catch (e) {
    /* no query support — keep the default feed */
  }

  function ready(fn) {
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', fn);
    } else {
      fn();
    }
  }

  ready(function () {
    var filters = Array.prototype.slice.call(document.querySelectorAll('[data-filter]'));
    var records = Array.prototype.slice.call(document.querySelectorAll('[data-record]'));

    filters.forEach(function (button) {
      button.addEventListener('click', function () {
        filters.forEach(function (item) {
          item.setAttribute('aria-pressed', item === button ? 'true' : 'false');
        });

        var key = button.getAttribute('data-filter');
        records.forEach(function (card) {
          var visible =
            key === 'all' ||
            key === 'month' ||
            (key === 'photos' && card.getAttribute('data-has-photos') === 'true') ||
            (key === 'note' && card.getAttribute('data-has-note') === 'true');
          card.hidden = !visible;
        });
      });
    });

    Array.prototype.forEach.call(document.querySelectorAll('.like-btn'), function (button) {
      button.addEventListener('click', function () {
        var liked = button.getAttribute('aria-pressed') === 'true';
        var next = !liked;
        var countNode = button.querySelector('.like-count');
        var count = parseInt(countNode.textContent, 10) || 0;

        count = Math.max(0, count + (next ? 1 : -1));
        button.setAttribute('aria-pressed', next ? 'true' : 'false');
        countNode.textContent = String(count);
        button.setAttribute(
          'aria-label',
          (next ? '取消点赞' : '点赞') + '，当前 ' + count + ' 人喜欢'
        );
      });
    });
  });
})();
