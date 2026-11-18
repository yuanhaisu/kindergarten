(() => {
  "use strict";

  const loginMain = document.getElementById("login-main");
  const consentRow = document.getElementById("consent-row");
  const consentInput = document.getElementById("consent-checkbox");
  const consentError = document.getElementById("consent-error");
  const wechatButton = document.getElementById("wechat-login");
  const wechatLabel = document.getElementById("wechat-login-label");
  const phoneButton = document.getElementById("phone-login");
  const toast = document.getElementById("toast");

  const legalLayer = document.getElementById("legal-layer");
  const legalTitle = document.getElementById("legal-title");
  const legalBody = document.getElementById("legal-body");
  const phoneLayer = document.getElementById("phone-layer");
  const phoneForm = document.getElementById("phone-form");
  const phoneInput = document.getElementById("phone-input");
  const codeInput = document.getElementById("code-input");
  const phoneField = document.getElementById("phone-field");
  const codeField = document.getElementById("code-field");
  const phoneError = document.getElementById("phone-error");
  const phoneLede = document.getElementById("phone-lede");
  const sendCodeButton = document.getElementById("send-code");

  if (
    !loginMain || !consentRow || !consentInput || !consentError ||
    !wechatButton || !wechatLabel || !phoneButton || !toast ||
    !legalLayer || !legalTitle || !legalBody || !phoneLayer || !phoneForm ||
    !phoneInput || !codeInput || !phoneField || !codeField || !phoneError ||
    !phoneLede || !sendCodeButton
  ) {
    return;
  }

  const legalCopy = {
    terms: {
      title: "用户协议",
      paragraphs: [
        "使用养成星球家长端，即表示你同意：仅在已绑定的孩子范围内使用家园沟通、成长记录与接送服务；不将账号转借他人；遵守孩子所在园所发布的具体规定。",
        "本页为协议要点摘要，完整协议以园所正式发布版本为准。"
      ]
    },
    privacy: {
      title: "隐私政策",
      paragraphs: [
        "为了完成家园沟通与在园照护，我们会在必要范围内使用你的账号信息、孩子所在园所提供的信息，以及通知、请假、接送和成长记录等服务数据。",
        "我们不出售、不出租你和孩子的信息。你可以联系园所老师，查询、更正或申请删除相关信息。"
      ]
    }
  };

  const layers = [legalLayer, phoneLayer];
  const layerTriggers = new WeakMap();
  const layerCloseTimers = new WeakMap();
  let activeLayer = null;
  let toastTimer = 0;
  let wechatPending = false;
  let codeSent = false;

  function setConsentError(visible, moveFocus) {
    consentError.hidden = !visible;
    consentRow.classList.toggle("is-invalid", visible);
    consentInput.setAttribute("aria-invalid", visible ? "true" : "false");
    if (visible && moveFocus) {
      consentInput.focus({ preventScroll: true });
    }
  }

  function ensureConsent() {
    if (consentInput.checked) {
      return true;
    }
    setConsentError(true, true);
    return false;
  }

  function showToast(message) {
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.remove("is-visible");
    void toast.offsetWidth;
    toast.classList.add("is-visible");
    toastTimer = window.setTimeout(() => {
      toast.classList.remove("is-visible");
    }, 2000);
  }

  function getFocusable(layer) {
    const selector = "button:not([disabled]), [href], input:not([disabled]), [tabindex]:not([tabindex='-1'])";
    return Array.from(layer.querySelectorAll(selector)).filter((node) => {
      return !node.hasAttribute("hidden") && node.getClientRects().length > 0;
    });
  }

  function openLayer(layer, trigger, initialFocus) {
    if (layer === activeLayer) {
      return;
    }
    if (activeLayer) {
      closeLayer(activeLayer, false);
    }

    const closeTimer = layerCloseTimers.get(layer);
    if (closeTimer) {
      window.clearTimeout(closeTimer);
      layerCloseTimers.delete(layer);
    }

    activeLayer = layer;
    layerTriggers.set(layer, trigger);
    layer.hidden = false;
    layer.setAttribute("aria-hidden", "false");
    loginMain.setAttribute("inert", "");

    window.requestAnimationFrame(() => {
      layer.classList.add("is-open");
      const focusTarget = initialFocus || layer.querySelector(".sheet-close") || layer;
      focusTarget.focus({ preventScroll: true });
    });
  }

  function closeLayer(layer, restoreFocus = true) {
    if (!layer || layer.hidden) {
      return;
    }

    const trigger = layerTriggers.get(layer);
    layer.classList.remove("is-open");
    layer.setAttribute("aria-hidden", "true");
    loginMain.removeAttribute("inert");

    if (activeLayer === layer) {
      activeLayer = null;
    }

    window.clearTimeout(layerCloseTimers.get(layer));
    const closeTimer = window.setTimeout(() => {
      if (!layer.classList.contains("is-open")) {
        layer.hidden = true;
      }
      layerCloseTimers.delete(layer);
    }, 220);
    layerCloseTimers.set(layer, closeTimer);

    if (restoreFocus && trigger && trigger.isConnected) {
      trigger.focus({ preventScroll: true });
    }
  }

  function openLegal(kind, trigger) {
    const entry = legalCopy[kind] || legalCopy.terms;
    legalTitle.textContent = entry.title;

    const paragraphs = entry.paragraphs.map((text) => {
      const paragraph = document.createElement("p");
      paragraph.textContent = text;
      return paragraph;
    });
    legalBody.replaceChildren(...paragraphs);

    openLayer(legalLayer, trigger, legalLayer.querySelector(".sheet-close"));
  }

  function setFormError(message, field) {
    phoneError.textContent = message || "";
    phoneError.hidden = !message;
    [phoneField, codeField].forEach((wrapper) => {
      wrapper.classList.remove("is-invalid");
      wrapper.querySelector("input").setAttribute("aria-invalid", "false");
    });

    if (message && field) {
      field.classList.add("is-invalid");
      field.querySelector("input").setAttribute("aria-invalid", "true");
    }
  }

  function resetPhoneForm() {
    phoneForm.reset();
    codeSent = false;
    sendCodeButton.textContent = "获取验证码";
    phoneLede.textContent = "验证码将发送至该手机号。";
    setFormError("", null);
  }

  function openPhone(trigger) {
    resetPhoneForm();
    openLayer(phoneLayer, trigger, phoneInput);
  }

  function normalizePhone() {
    return phoneInput.value.replace(/\D/g, "").slice(0, 11);
  }

  function normalizeCode() {
    return codeInput.value.replace(/\D/g, "").slice(0, 6);
  }

  consentInput.addEventListener("change", () => {
    if (consentInput.checked) {
      setConsentError(false, false);
    }
  });

  consentRow.addEventListener("click", (event) => {
    if (event.target.closest("a, input")) {
      return;
    }
    consentInput.checked = !consentInput.checked;
    consentInput.dispatchEvent(new Event("change", { bubbles: true }));
  });

  document.querySelectorAll("[data-legal]").forEach((link) => {
    link.addEventListener("click", (event) => {
      event.preventDefault();
      event.stopPropagation();
      openLegal(link.dataset.legal, link);
    });
  });

  wechatButton.addEventListener("click", () => {
    if (wechatPending || !ensureConsent()) {
      return;
    }

    wechatPending = true;
    wechatButton.setAttribute("aria-busy", "true");
    wechatLabel.textContent = "正在唤起微信…";

    window.setTimeout(() => {
      wechatPending = false;
      wechatButton.removeAttribute("aria-busy");
      wechatLabel.textContent = "微信一键登录";
      showToast("微信授权成功");
    }, 1200);
  });

  phoneButton.addEventListener("click", () => {
    if (!ensureConsent()) {
      return;
    }
    openPhone(phoneButton);
  });

  phoneInput.addEventListener("input", () => {
    phoneInput.value = normalizePhone();
    if (phoneError.textContent) {
      setFormError("", null);
    }
  });

  codeInput.addEventListener("input", () => {
    codeInput.value = normalizeCode();
    if (phoneError.textContent) {
      setFormError("", null);
    }
  });

  sendCodeButton.addEventListener("click", () => {
    const phone = normalizePhone();
    if (!/^1[3-9]\d{9}$/.test(phone)) {
      setFormError("请输入 11 位手机号", phoneField);
      phoneInput.focus({ preventScroll: true });
      return;
    }

    setFormError("", null);
    codeSent = true;
    sendCodeButton.textContent = "重新发送";
    phoneLede.textContent = "验证码已发送，请查收短信。";
    codeInput.focus({ preventScroll: true });
  });

  phoneForm.addEventListener("submit", (event) => {
    event.preventDefault();
    const phone = normalizePhone();
    const code = normalizeCode();

    if (!/^1[3-9]\d{9}$/.test(phone)) {
      setFormError("请输入 11 位手机号", phoneField);
      phoneInput.focus({ preventScroll: true });
      return;
    }
    if (!codeSent) {
      setFormError("请先获取验证码", codeField);
      codeInput.focus({ preventScroll: true });
      return;
    }
    if (!/^\d{6}$/.test(code)) {
      setFormError("请输入 6 位验证码", codeField);
      codeInput.focus({ preventScroll: true });
      return;
    }

    setFormError("", null);
    closeLayer(phoneLayer);
    showToast("登录成功");
  });

  layers.forEach((layer) => {
    layer.querySelectorAll("[data-close]").forEach((button) => {
      button.addEventListener("click", () => closeLayer(layer));
    });
  });

  document.addEventListener("keydown", (event) => {
    if (!activeLayer) {
      return;
    }

    if (event.key === "Escape") {
      event.preventDefault();
      closeLayer(activeLayer);
      return;
    }

    if (event.key !== "Tab") {
      return;
    }

    const focusable = getFocusable(activeLayer);
    if (!focusable.length) {
      return;
    }

    const first = focusable[0];
    const last = focusable[focusable.length - 1];

    if (!activeLayer.contains(document.activeElement)) {
      event.preventDefault();
      (event.shiftKey ? last : first).focus();
      return;
    }
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });

  if (new URLSearchParams(window.location.search).get("state") === "need-consent") {
    setConsentError(true, false);
  }
})();
