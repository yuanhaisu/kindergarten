/* Prototype-only theme switch for the 星空蓝 gallery.
 * Default (no ?theme=) = starry, set on <html> so the page is blue without JS.
 * Supported: ?theme=bright (原亮橙) | ?theme=deep | ?theme=ink | ?theme=celestial | ?theme=starry.
 * Loaded in <head> so data-theme is set before stylesheets resolve.
 */
(function () {
  try {
    var theme = new URLSearchParams(location.search).get('theme');
    if (theme === 'bright') {
      document.documentElement.removeAttribute('data-theme');
    } else if (theme === 'deep' || theme === 'ink' || theme === 'celestial' || theme === 'starry') {
      document.documentElement.setAttribute('data-theme', theme);
    }
  } catch (e) {}
})();
