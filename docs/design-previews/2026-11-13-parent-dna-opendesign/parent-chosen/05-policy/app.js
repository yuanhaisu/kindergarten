(function () {
  "use strict";

  var scroller = document.getElementById("policy-scroll");
  if (!scroller) return;

  var params = new URLSearchParams(window.location.search);
  var state = params.get("state") || "";
  var from = params.get("from");
  var docs = ["privacy", "minor", "terms"];
  var titles = {
    privacy: "隐私政策",
    minor: "未成年人个人信息保护规则",
    terms: "用户协议",
  };
  var tabs = Array.prototype.slice.call(document.querySelectorAll(".doc-tab"));
  var panels = Array.prototype.slice.call(document.querySelectorAll(".doc-panel"));
  var backButton = document.getElementById("back-button");

  // 05 从 02 的协议链接、04 的「查看」或 29 的隐私设置进入；默认返回同意页。
  var backTarget = from === "login"
    ? "../02-login/index.html"
    : from === "settings"
      ? "../29-settings/index.html"
      : "../04-consent/index.html";

  function selectDoc(doc, updateUrl) {
    if (docs.indexOf(doc) < 0) doc = "privacy";

    tabs.forEach(function (tab) {
      var active = tab.getAttribute("data-doc") === doc;
      tab.classList.toggle("is-active", active);
      tab.setAttribute("aria-selected", active ? "true" : "false");
      tab.tabIndex = active ? 0 : -1;
    });

    panels.forEach(function (panel) {
      panel.hidden = panel.id !== "panel-" + doc;
    });

    document.title = titles[doc] + " · 养成星球家长端";
    scroller.scrollTop = 0;

    if (updateUrl) {
      var next = "?doc=" + encodeURIComponent(doc) + (from ? "&from=" + encodeURIComponent(from) : "");
      window.history.replaceState(null, "", next);
    }
  }

  tabs.forEach(function (tab) {
    tab.addEventListener("click", function () {
      selectDoc(tab.getAttribute("data-doc"), true);
    });
  });

  if (backButton) {
    backButton.addEventListener("click", function () {
      window.location.href = backTarget;
    });
  }

  function jumpTo(id) {
    var target = document.getElementById(id);
    if (!target) return;
    scroller.scrollTop = Math.max(target.offsetTop - 8, 0);
  }

  // 初始文档：04 / 02 传 ?doc=，预览用 ?state=doc-<doc> 或 #<doc>。
  var hash = (window.location.hash || "").replace("#", "");
  var requested = params.get("doc");
  var previewDoc = state.indexOf("doc-") === 0 ? state.slice(4) : state;
  var initial = "privacy";

  if (docs.indexOf(requested) >= 0) initial = requested;
  else if (docs.indexOf(hash) >= 0) initial = hash;
  else if (docs.indexOf(previewDoc) >= 0) initial = previewDoc;

  selectDoc(initial, false);

  window.requestAnimationFrame(function () {
    if (state === "end") {
      scroller.scrollTop = scroller.scrollHeight;
    } else if (state === "contact") {
      jumpTo("policy-contact");
    }
  });
})();
