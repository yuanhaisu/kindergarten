(function () {
  var SUPPORTED_PERIODS = { month: true };
  var params = new URLSearchParams(window.location.search);
  var requested = params.get('period') || 'month';
  var period = SUPPORTED_PERIODS[requested] ? requested : 'month';

  document.documentElement.setAttribute('data-period', period);
  var app = document.getElementById('app');
  if (app) app.setAttribute('data-period', period);

  var toast = document.getElementById('toast');
  var hideTimer = null;

  function showToast(message) {
    if (!toast || !message) return;
    window.clearTimeout(hideTimer);
    toast.textContent = message;
    toast.classList.add('is-on');
    hideTimer = window.setTimeout(function () {
      toast.classList.remove('is-on');
    }, 2000);
  }

  var triggers = document.querySelectorAll('[data-toast]');
  Array.prototype.forEach.call(triggers, function (el) {
    el.addEventListener('click', function () {
      showToast(el.getAttribute('data-toast'));
    });
  });
})();
