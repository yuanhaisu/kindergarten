/* Prototype-only theme switch for the Petit Prince gallery.
 * Default (no ?theme=) = petitprince, set on <html> so the page is paper-white
 * without JS. Supported: ?theme=bright (原亮橙) | deep | ink | celestial | starry | petitprince.
 * Loaded in <head> so data-theme is set before stylesheets resolve.
 */
(function () {
  try {
    var theme = new URLSearchParams(location.search).get('theme');
    if (theme === 'bright') {
      document.documentElement.removeAttribute('data-theme');
    } else if (theme === 'deep' || theme === 'ink' || theme === 'celestial' || theme === 'starry' || theme === 'petitprince') {
      document.documentElement.setAttribute('data-theme', theme);
    }
  } catch (e) {}
})();
