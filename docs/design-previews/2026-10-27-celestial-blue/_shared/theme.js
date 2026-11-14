/* Prototype-only theme switch for the 星海蓝 gallery.
 * Default (no ?theme=) = celestial, set on <html> so the page is blue without JS.
 * Supported: ?theme=bright (原亮橙) | ?theme=deep | ?theme=ink | ?theme=celestial.
 * Loaded in <head> so data-theme is set before stylesheets resolve.
 */
(function () {
  try {
    var theme = new URLSearchParams(location.search).get('theme');
    if (theme === 'bright') {
      document.documentElement.removeAttribute('data-theme');
    } else if (theme === 'deep' || theme === 'ink' || theme === 'celestial') {
      document.documentElement.setAttribute('data-theme', theme);
    }
  } catch (e) {}
})();
