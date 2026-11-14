(function () {
  var toast = document.getElementById('toast');
  var hideTimer = null;
  var swapTimer = null;

  function showToast(message) {
    if (!toast || !message) return;
    window.clearTimeout(hideTimer);
    toast.textContent = message;
    toast.classList.add('is-on');
    hideTimer = window.setTimeout(function () {
      toast.classList.remove('is-on');
    }, 2000);
  }

  function runAction(el) {
    var message = el.getAttribute('data-toast');
    var done = el.getAttribute('data-toast-done');
    var delay = parseInt(el.getAttribute('data-toast-delay'), 10);

    window.clearTimeout(swapTimer);
    showToast(message);

    if (done) {
      swapTimer = window.setTimeout(function () {
        showToast(done);
      }, isNaN(delay) ? 1200 : delay);
    }
  }

  var triggers = document.querySelectorAll('[data-toast]');
  Array.prototype.forEach.call(triggers, function (el) {
    el.addEventListener('click', function () {
      runAction(el);
    });
  });
})();
