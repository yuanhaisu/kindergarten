(() => {
  "use strict";

  const app = document.querySelector(".mp-app");
  const main = document.querySelector(".mp-main");
  const toast = document.getElementById("toast");
  const leaveCta = document.querySelector('[data-od-id="leave-detail-cta"]');
  const peopleCard = document.querySelector(".people-card");

  if (!app || !main || !toast) {
    return;
  }

  // 本变体只有「请假中」一种状态；?state=leave 与默认打开一致。
  document.body.dataset.state = "leave";
  app.dataset.state = "leave";

  let toastTimer = 0;

  function showToast(message) {
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add("is-visible");
    toastTimer = window.setTimeout(() => {
      toast.classList.remove("is-visible");
    }, 2000);
  }

  if (leaveCta) {
    leaveCta.addEventListener("click", () => {
      showToast("请假详情 · 待审批");
    });
  }

  if (peopleCard) {
    peopleCard.addEventListener("click", () => {
      showToast("可接送人 · 外婆待审核");
    });
  }

  document.addEventListener("click", (event) => {
    if (!(event.target instanceof Element)) {
      return;
    }

    const link = event.target.closest('a[href="#"]');
    if (!link) {
      return;
    }

    event.preventDefault();
    if (link.classList.contains("tab") && link.dataset.tab === "pickup") {
      main.scrollTop = 0;
    }
  });
})();
