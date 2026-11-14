(function () {
  var params = new URLSearchParams(window.location.search);
  var requested = (params.get("state") || "").toLowerCase();
  var state = requested === "away" || requested === "done" ? requested : "in";

  document.body.dataset.state = state;
})();
