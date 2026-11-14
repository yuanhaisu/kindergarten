(function () {
  'use strict';

  var state = new URLSearchParams(window.location.search).get('state');
  var activeView = state === 'empty' ? 'empty' : 'default';
  var views = document.querySelectorAll('[data-view]');

  for (var i = 0; i < views.length; i += 1) {
    views[i].hidden = views[i].getAttribute('data-view') !== activeView;
  }

  document.documentElement.setAttribute('data-page-state', activeView);
})();
