/* Prototype navigation wiring for the chosen parent pages.
   Content and visual states stay in each page; this file only turns
   existing affordances into real links. */
(function () {
  'use strict';

  var path = window.location.pathname.replace(/\\/g, '/');
  var slug = (path.match(/parent-chosen\/([^/]+)\//) || [])[1] || '';
  var TAB = {
    '首页': '../01-home/index.html',
    '成长': '../11-growth/index.html',
    '接送': '../17-pickup/index.html',
    '我的': '../31-profile/index.html'
  };

  var pageRoutes = {
    '01-home': {
      '[data-od-id="quick-leave"]': '../08-leave-index/index.html',
      '[data-od-id="quick-notifications"]': '../06-notice-index/index.html',
      '[data-od-id="all-updates"]': '../06-notice-index/index.html'
    },
    '06-notice-index': {
      '.notice-card': '../07-notice-detail/index.html',
      '[data-od-id="notice-back"]': '../01-home/index.html'
    },
    '08-leave-index': {
      '.record-card': '../10-leave-detail/index.html',
      '[data-od-id="leave-new"]': '../09-leave-create/index.html',
      '[data-od-id="leave-back"]': '../01-home/index.html'
    },
    '15-album-index': {
      '.album-card': '../16-album-detail/index.html',
      '[data-od-id="back-button"]': '../11-growth/index.html'
    },
    '17-pickup': {
      '[data-od-id="pickup-primary-cta"]': '../18-pickup-qrcode/index.html',
      '[data-od-id="pickup-code-shortcut"]': '../18-pickup-qrcode/index.html',
      '[data-od-id="authorized-pickup-card"]': '../19-pickup-authorized/index.html'
    },
    '19-pickup-authorized': {
      '[data-od-id="add-authorizer"]': '../20-pickup-authorized-create/index.html',
      '[data-od-id="add-authorizer-empty"]': '../20-pickup-authorized-create/index.html',
      '[data-od-id="back-to-pickup"]': '../17-pickup/index.html'
    },
    '21-tasks': {
      '.task-row': '../22-tasks-detail/index.html',
      '[data-od-id="today-card"]': '../22-tasks-detail/index.html',
      '[data-od-id="class-wall-row"]': '../23-tasks-wall/index.html',
      '[data-od-id="back-button"]': '../01-home/index.html'
    },
    '31-profile': {
      '[data-od-id="service-child-file"]': '../27-student/index.html',
      '[data-od-id="service-class-info"]': '../25-class/index.html',
      '[data-od-id="service-achievements"]': '../24-points/index.html',
      '[data-od-id="service-feedback"]': '../28-feedback/index.html',
      '[data-od-id="setting-messages"]': '../30-messages/index.html',
      '[data-od-id="setting-notifications"]': '../29-settings/index.html',
      '[data-od-id="setting-privacy"]': '../29-settings/index.html'
    }
  };

  function go(href) {
    if (href) window.location.href = href;
  }

  function tabFor(target) {
    var tab = target.closest && target.closest('nav .tab, nav .nav-item');
    if (!tab) return null;
    var label = (tab.textContent || '').trim();
    var href = TAB[label];
    if (!href) return null;
    if (path.indexOf('/' + href.split('/')[1] + '/') > -1) return null;
    return href;
  }

  document.addEventListener('click', function (event) {
    var target = event.target;
    if (!(target instanceof Element)) return;

    var tabHref = tabFor(target);
    if (tabHref) {
      event.preventDefault();
      go(tabHref);
      return;
    }

    var routes = pageRoutes[slug];
    if (!routes) return;
    for (var selector in routes) {
      if (!Object.prototype.hasOwnProperty.call(routes, selector)) continue;
      var node = target.closest(selector);
      if (node) {
        event.preventDefault();
        event.stopPropagation();
        go(routes[selector]);
        return;
      }
    }
  }, true);

  // Make the static tabs keyboard-addressable too.
  Array.prototype.forEach.call(document.querySelectorAll('nav .tab, nav .nav-item'), function (tab) {
    var label = (tab.textContent || '').trim();
    if (!TAB[label]) return;
    if (tab.tagName === 'A' && (!tab.getAttribute('href') || tab.getAttribute('href') === '#')) {
      tab.setAttribute('href', TAB[label]);
    }
  });
})();
