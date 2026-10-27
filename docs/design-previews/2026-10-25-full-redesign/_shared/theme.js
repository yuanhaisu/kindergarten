/* Prototype-only theme switch.
 * Default (no ?theme=) = bright. Supported: ?theme=deep | ?theme=ink | ?theme=bright.
 * Loaded in <head> so data-theme is set before stylesheets resolve.
 */
(function () {
  try {
    var theme = new URLSearchParams(location.search).get('theme');
    if (theme === 'deep' || theme === 'ink') {
      document.documentElement.setAttribute('data-theme', theme);
    } else if (theme === 'bright') {
      document.documentElement.removeAttribute('data-theme');
    }
  } catch (e) {}
})();
