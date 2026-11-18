(() => {
  "use strict";

  const button = document.getElementById("wechat-login");
  const buttonLabel = document.getElementById("wechat-label");
  const row = document.getElementById("consent-row");
  const checkbox = document.getElementById("consent-checkbox");
  const error = document.getElementById("consent-error");

  if (!button || !buttonLabel || !row || !checkbox || !error) {
    return;
  }

  let busy = false;

  function setErrorVisible(visible) {
    error.hidden = !visible;
    row.classList.toggle("is-invalid", visible);
    checkbox.setAttribute("aria-invalid", visible ? "true" : "false");
  }

  checkbox.addEventListener("change", () => {
    if (checkbox.checked) {
      setErrorVisible(false);
    }
  });

  row.addEventListener("click", (event) => {
    if (busy) {
      return;
    }
    if (event.target instanceof Element && event.target.closest("a, label, input")) {
      return;
    }
    checkbox.checked = !checkbox.checked;
    checkbox.dispatchEvent(new Event("change", { bubbles: true }));
  });

  button.addEventListener("click", () => {
    if (busy) {
      return;
    }

    if (!checkbox.checked) {
      setErrorVisible(true);
      checkbox.focus({ preventScroll: true });
      return;
    }

    setErrorVisible(false);
    busy = true;
    button.setAttribute("aria-busy", "true");
    buttonLabel.textContent = "正在唤起微信…";

    window.setTimeout(() => {
      window.location.href = "../../../parent-chosen/04-consent/index.html";
    }, 900);
  });
})();
