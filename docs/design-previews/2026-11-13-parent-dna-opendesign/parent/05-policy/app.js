(function () {
  'use strict';

  var scroller = document.getElementById('policy-scroll');
  if (!scroller) return;

  var params = new URLSearchParams(window.location.search);
  var state = params.get('state');

  function jumpTo(id) {
    var target = document.getElementById(id);
    if (!target) return;
    scroller.scrollTop = Math.max(target.offsetTop - 8, 0);
  }

  function applyState() {
    if (state === 'end') {
      scroller.scrollTop = scroller.scrollHeight;
      return;
    }
    if (state === 'contact') {
      jumpTo('section-contact');
    }
  }

  window.requestAnimationFrame(applyState);
})();
