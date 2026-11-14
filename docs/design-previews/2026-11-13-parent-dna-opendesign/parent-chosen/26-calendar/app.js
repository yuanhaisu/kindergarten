(function () {
  "use strict";

  var TODAY = "2026-11-12";
  var WEEKDAYS = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"];

  var dayInfo = {
    "2026-11-06": { kind: "holiday", count: 1 },
    "2026-11-12": { kind: "mixed", count: 2, detailed: true },
    "2026-11-13": { kind: "activity", count: 1 },
    "2026-11-14": { kind: "activity", count: 1 },
    "2026-11-18": { kind: "meeting", count: 1 },
    "2026-11-24": { kind: "activity", count: 1 }
  };

  var dayButtons = Array.prototype.slice.call(document.querySelectorAll(".day[data-date]"));
  var agendaTitle = document.getElementById("agenda-title");
  var agendaMeta = document.getElementById("agenda-meta");
  var agendaCard = document.getElementById("agenda-card");
  var announcement = document.getElementById("selection-announcement");
  var todayButton = document.getElementById("today-button");
  var defaultAgendaMarkup = agendaCard.innerHTML;

  function parseDate(dateKey) {
    var parts = dateKey.split("-").map(Number);
    return new Date(parts[0], parts[1] - 1, parts[2]);
  }

  function weekdayOf(dateKey) {
    return WEEKDAYS[parseDate(dateKey).getDay()];
  }

  function readableDate(dateKey) {
    var parts = dateKey.split("-").map(Number);
    return parts[1] + " 月 " + parts[2] + " 日";
  }

  function summaryFor(info) {
    if (!info) {
      return '<p class="day-summary__text">这一天没有安排。</p>';
    }

    if (info.kind === "holiday") {
      return '<p class="day-summary__text">这一天是<span class="hl hl--terracotta">假期</span>。</p>';
    }

    if (info.kind === "meeting") {
      return '<p class="day-summary__text">这一天有<span class="hl hl--sage">家长会</span>。</p>';
    }

    return '<p class="day-summary__text">这一天有 1 个<span class="hl hl--marigold">活动</span>。</p>';
  }

  function metaFor(dateKey, info) {
    var date = readableDate(dateKey);

    if (!info) {
      return date + " · 无安排";
    }

    if (info.kind === "holiday") {
      return date + " · 假期";
    }

    return date + " · " + info.count + " 个安排";
  }

  function renderDay(dateKey) {
    var info = dayInfo[dateKey];

    agendaTitle.textContent = dateKey === TODAY ? "今天" : weekdayOf(dateKey);
    agendaMeta.textContent = metaFor(dateKey, info);

    if (info && info.detailed) {
      agendaCard.className = "card agenda-card";
      agendaCard.innerHTML = defaultAgendaMarkup;
    } else {
      agendaCard.className = "card agenda-card agenda-card--summary";
      agendaCard.innerHTML =
        '<div class="day-summary">' +
        '<p class="day-summary__eyebrow">当日安排</p>' +
        summaryFor(info) +
        "</div>";
    }
  }

  function selectDay(dateKey, shouldAnnounce) {
    dayButtons.forEach(function (button) {
      var isSelected = button.getAttribute("data-date") === dateKey;
      button.classList.toggle("is-selected", isSelected);
      button.setAttribute("aria-pressed", String(isSelected));
    });

    renderDay(dateKey);

    if (shouldAnnounce && announcement) {
      var info = dayInfo[dateKey];
      var countText = !info
        ? "无安排"
        : info.kind === "holiday"
          ? "假期"
          : info.count + " 个安排";
      announcement.textContent =
        "已选择 " + readableDate(dateKey) + "，" + weekdayOf(dateKey) + "，" + countText;
    }
  }

  dayButtons.forEach(function (button) {
    button.addEventListener("click", function () {
      selectDay(button.getAttribute("data-date"), true);
    });
  });

  if (todayButton) {
    todayButton.addEventListener("click", function () {
      selectDay(TODAY, true);
    });
  }
})();
