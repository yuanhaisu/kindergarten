(() => {
  "use strict";

  const release = (target) => target.classList.remove("is-pressed");

  document.querySelectorAll(".tap-target").forEach((target) => {
    target.addEventListener("pointerdown", (event) => {
      if (event.pointerType === "mouse" && event.button !== 0) return;
      target.classList.add("is-pressed");
    });

    ["pointerup", "pointercancel", "pointerleave", "blur"].forEach((type) => {
      target.addEventListener(type, () => release(target));
    });
  });
})();
