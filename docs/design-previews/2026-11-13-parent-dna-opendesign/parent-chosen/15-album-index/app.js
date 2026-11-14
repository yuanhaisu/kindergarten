(function () {
  "use strict";

  var app = document.querySelector(".mp-app");
  if (!app) return;

  var state = new URLSearchParams(window.location.search).get("state");
  app.setAttribute("data-state", state === "empty" ? "empty" : "default");
})();
