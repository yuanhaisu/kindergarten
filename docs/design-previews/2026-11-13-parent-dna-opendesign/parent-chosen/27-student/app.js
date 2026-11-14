(() => {
  const toast = document.querySelector("[data-toast]");
  let hideTimer = 0;
  let clearTimer = 0;

  const showToast = (message) => {
    if (!toast) return;
    window.clearTimeout(hideTimer);
    window.clearTimeout(clearTimer);
    toast.textContent = message;
    toast.classList.add("is-visible");
    hideTimer = window.setTimeout(() => {
      toast.classList.remove("is-visible");
      clearTimer = window.setTimeout(() => {
        toast.textContent = "";
      }, 200);
    }, 2000);
  };

  document.querySelectorAll("[data-action]").forEach((button) => {
    button.addEventListener("click", () => {
      const action = button.getAttribute("data-action");
      if (action === "back") {
        showToast("返回「我的」");
      } else if (action === "edit") {
        showToast("资料修改请联系班级老师");
      }
    });
  });

  document.querySelectorAll("[data-call]").forEach((button) => {
    button.addEventListener("click", () => {
      const contact = button.getAttribute("data-call");
      showToast(`正在拨打 ${contact}`);
    });
  });
})();
