(() => {
  "use strict";

  const main = document.querySelector(".mp-main");

  // 早餐优先展示今日食谱；食谱缺失时回退到老师记录的 meal_label 文案。
  const params = new URLSearchParams(window.location.search);
  if (params.get("state") === "meal-fallback") {
    const mealLine = document.getElementById("meal-line");
    const mealSource = document.getElementById("meal-source");
    if (mealLine) {
      mealLine.textContent = "早餐吃得很好。";
    }
    if (mealSource) {
      mealSource.textContent = "老师记录 · 10:20 更新";
    }
  }

  /* Placeholder links stay put; without this they would jump the scroll
   * container or change the URL hash. */
  document.addEventListener("click", (event) => {
    if (!(event.target instanceof Element)) {
      return;
    }

    const link = event.target.closest('a[href="#"]');
    if (!link) {
      return;
    }

    event.preventDefault();
    if (link.classList.contains("tab") && link.dataset.tab === "home" && main) {
      main.scrollTop = 0;
    }
  });
})();
