(function () {
  "use strict";

  var toast = document.querySelector(".toast");
  var toastTimer = null;

  function hideToast() {
    if (!toast) return;
    toast.classList.remove("is-visible");
    if (toastTimer) {
      window.clearTimeout(toastTimer);
      toastTimer = null;
    }
  }

  function showToast(message) {
    if (!toast) return;

    toast.textContent = message;
    toast.classList.add("is-visible");

    if (toastTimer) {
      window.clearTimeout(toastTimer);
    }

    toastTimer = window.setTimeout(function () {
      toast.classList.remove("is-visible");
      toastTimer = null;
    }, 2000);
  }

  var backButton = document.querySelector('[data-action="back"]');
  if (backButton) {
    backButton.addEventListener("click", function () {
      var insidePreview = window.parent !== window;

      if (!insidePreview && window.history.length > 1) {
        window.history.back();
        return;
      }

      showToast("返回上一页");
    });
  }

  var callButtons = document.querySelectorAll("[data-call]");
  Array.prototype.forEach.call(callButtons, function (button) {
    button.addEventListener("click", function () {
      var teacher = button.getAttribute("data-call");
      if (!teacher) return;
      showToast("正在呼叫" + teacher);
    });
  });

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape") {
      hideToast();
    }
  });
})();
