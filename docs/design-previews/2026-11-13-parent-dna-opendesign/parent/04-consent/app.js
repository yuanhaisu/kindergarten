(function () {
  "use strict";

  var rows = Array.prototype.slice.call(document.querySelectorAll(".consent-input"));
  var submit = document.getElementById("consent-submit");
  var decline = document.getElementById("consent-decline");
  var toast = document.getElementById("consent-toast");
  var toastTimer = null;

  function syncSubmit() {
    var allChecked = rows.every(function (input) {
      return input.checked;
    });
    submit.disabled = !allChecked;
  }

  function showToast(message) {
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add("is-visible");
    toastTimer = window.setTimeout(function () {
      toast.classList.remove("is-visible");
    }, 2000);
  }

  rows.forEach(function (input) {
    input.addEventListener("change", syncSubmit);
  });

  submit.addEventListener("click", function () {
    if (submit.disabled) return;
    showToast("已同意三份约定");
  });

  decline.addEventListener("click", function () {
    showToast("未同意任何授权，你可以随时再来确认");
  });

  var state = new URLSearchParams(window.location.search).get("state");
  if (state === "partial") {
    var minor = document.getElementById("consent-minor");
    if (minor) minor.checked = false;
  }

  syncSubmit();
})();
