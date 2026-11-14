(function () {
  "use strict";

  var consentRow = document.getElementById("consent-row");
  var consentInput = document.getElementById("consent-checkbox");
  var consentError = document.getElementById("consent-error");
  var wechatButton = document.getElementById("wechat-login");
  var wechatLabel = document.getElementById("wechat-login-label");

  if (!consentRow || !consentInput || !consentError || !wechatButton || !wechatLabel) {
    return;
  }

  var pending = false;

  function setConsentError(visible) {
    consentError.hidden = !visible;
    consentRow.classList.toggle("is-invalid", visible);
    consentInput.setAttribute("aria-invalid", visible ? "true" : "false");
  }

  // The whole consent row is tappable; links and the native checkbox keep their own behavior.
  consentRow.addEventListener("click", function (event) {
    if (event.target.closest("a, input, label")) {
      return;
    }

    consentInput.checked = !consentInput.checked;
    consentInput.dispatchEvent(new Event("change", { bubbles: true }));
  });

  consentInput.addEventListener("change", function () {
    if (consentInput.checked) {
      setConsentError(false);
    }
  });

  wechatButton.addEventListener("click", function () {
    if (pending) {
      return;
    }

    if (!consentInput.checked) {
      setConsentError(true);
      consentInput.focus({ preventScroll: true });
      return;
    }

    pending = true;
    setConsentError(false);
    wechatButton.setAttribute("aria-busy", "true");
    wechatLabel.textContent = "正在唤起微信…";

    window.setTimeout(function () {
      pending = false;
      wechatButton.removeAttribute("aria-busy");
      wechatLabel.textContent = "微信一键登录";
    }, 1400);
  });

  // Preview hook: open with ?state=need-consent to inspect the consent error state.
  if (new URLSearchParams(window.location.search).get("state") === "need-consent") {
    consentInput.checked = false;
    setConsentError(true);
  }
})();
