(function () {
  "use strict";

  const main = document.querySelector(".mp-main");
  const chips = Array.from(document.querySelectorAll(".chip"));
  const cards = Array.from(document.querySelectorAll(".wall-card"));

  const layer = document.getElementById("sheet-layer");
  const sheetPhoto = document.getElementById("sheet-photo");
  const sheetName = document.getElementById("sheet-name");
  const sheetTask = document.getElementById("sheet-task");
  const sheetMeta = document.getElementById("sheet-meta");
  const closeBtn = document.getElementById("sheet-close");
  const mask = layer.querySelector(".sheet-mask");

  let lastCard = null;
  let closeTimer = 0;

  function applyFilter(filter) {
    chips.forEach(function (chip) {
      const on = chip.dataset.filter === filter;
      chip.classList.toggle("is-on", on);
      chip.setAttribute("aria-pressed", on ? "true" : "false");
    });

    cards.forEach(function (card) {
      const isMine = card.dataset.mine === "true";
      const show = filter === "all" || (filter === "mine" && isMine);
      card.hidden = !show;
    });
  }

  function openSheet(card) {
    window.clearTimeout(closeTimer);
    lastCard = card;

    sheetPhoto.className =
      "photo sheet-photo tone--" +
      card.dataset.tone +
      (card.dataset.flip === "true" ? " photo--flip" : "");
    sheetName.textContent = card.dataset.name;
    sheetTask.textContent = card.dataset.task;
    sheetMeta.textContent = "小一班 · 今天 " + card.dataset.time + " 已打卡";

    layer.hidden = false;
    main.classList.add("is-locked");
    main.setAttribute("inert", "");

    window.requestAnimationFrame(function () {
      window.requestAnimationFrame(function () {
        layer.classList.add("is-open");
      });
    });

    closeBtn.focus({ preventScroll: true });
  }

  function closeSheet() {
    if (layer.hidden) return;

    layer.classList.remove("is-open");
    main.classList.remove("is-locked");
    main.removeAttribute("inert");

    closeTimer = window.setTimeout(function () {
      layer.hidden = true;
      if (lastCard) lastCard.focus({ preventScroll: true });
    }, 210);
  }

  chips.forEach(function (chip) {
    chip.addEventListener("click", function () {
      applyFilter(chip.dataset.filter);
    });
  });

  cards.forEach(function (card) {
    card.addEventListener("click", function () {
      openSheet(card);
    });
  });

  closeBtn.addEventListener("click", closeSheet);
  mask.addEventListener("click", closeSheet);

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" && !layer.hidden) closeSheet();
  });
})();
