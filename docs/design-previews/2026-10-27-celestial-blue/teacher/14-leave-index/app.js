/* 请假审批 · 老师端原型交互
 * 数据结构对齐 GET /teacher/leaves 的真实字段（applicant / student / review_remark 等）。
 * 今天是 2026-10-26，示例数据与统计口径保持一致（待审批 3 + 已同意 4 + 已拒绝 1 = 8）。
 */
(() => {
  "use strict";

  const root = document.querySelector(".mp-app");
  const page = document.getElementById("page");
  const list = document.getElementById("leaveList");
  const filterBar = document.getElementById("filterBar");
  const listTitle = document.getElementById("listTitle");
  const listSub = document.getElementById("listSub");
  const emptyTitle = document.getElementById("emptyTitle");
  const emptySub = document.getElementById("emptySub");
  const emptyAction = document.getElementById("emptyAction");
  const toastEl = document.getElementById("toast");
  const dialogLayer = document.getElementById("rejectDialog");
  const reasonInput = document.getElementById("rejectReason");
  const reasonCount = document.getElementById("rejectCount");
  const rejectConfirm = document.getElementById("rejectConfirm");
  const statNodes = {
    pending: document.getElementById("statPending"),
    approved: document.getElementById("statApproved"),
    rejected: document.getElementById("statRejected"),
    total: document.getElementById("statTotal"),
  };
  const countNodes = {};
  filterBar.querySelectorAll(".filter-count").forEach((node) => {
    countNodes[node.dataset.count] = node;
  });

  const NOW = new Date(2026, 9, 26, 12, 0, 0);
  const NOW_STAMP = "2026-10-26T12:00";
  const FILTER_LABEL = { all: "全部", pending: "待审批", approved: "已同意", rejected: "已拒绝" };
  const STATUS_LABEL = { 0: "待审批", 1: "已同意", 2: "已拒绝" };

  /* 对应契约里的 status：0 待审批 / 1 已同意 / 2 已拒绝 */
  const records = [
    { id: 1, studentName: "乐乐", applicant: "乐乐妈妈", reason: "感冒发烧，在家休息两天", startDate: "2026-10-27", endDate: "2026-10-28", status: 0, createdAt: "2026-10-26T09:12", reviewedAt: "", reviewRemark: "" },
    { id: 2, studentName: "小雨", applicant: "小雨爸爸", reason: "家中有事，需带孩子回老家", startDate: "2026-10-26", endDate: "2026-10-30", status: 0, createdAt: "2026-10-25T20:15", reviewedAt: "", reviewRemark: "" },
    { id: 3, studentName: "朵朵", applicant: "朵朵奶奶", reason: "咳嗽加重，去医院复查", startDate: "2026-10-26", endDate: "2026-10-26", status: 0, createdAt: "2026-10-26T07:40", reviewedAt: "", reviewRemark: "" },
    { id: 4, studentName: "乐乐", applicant: "乐乐妈妈", reason: "接种疫苗后低烧，需观察一天", startDate: "2026-10-20", endDate: "2026-10-21", status: 1, createdAt: "2026-10-19T17:05", reviewedAt: "2026-10-19T18:22", reviewRemark: "同意，注意休息，返园前告知体温" },
    { id: 5, studentName: "小雨", applicant: "小雨妈妈", reason: "参加亲子活动", startDate: "2026-10-15", endDate: "2026-10-15", status: 1, createdAt: "2026-10-14T15:20", reviewedAt: "2026-10-14T16:05", reviewRemark: "" },
    { id: 6, studentName: "朵朵", applicant: "朵朵爸爸", reason: "回老家探亲", startDate: "2026-10-09", endDate: "2026-10-10", status: 1, createdAt: "2026-10-08T18:30", reviewedAt: "2026-10-08T19:40", reviewRemark: "同意，路上注意安全" },
    { id: 7, studentName: "乐乐", applicant: "乐乐爸爸", reason: "周一早高峰堵车，晚到半天", startDate: "2026-10-13", endDate: "2026-10-13", status: 1, createdAt: "2026-10-12T20:02", reviewedAt: "2026-10-12T20:40", reviewRemark: "" },
    { id: 8, studentName: "小雨", applicant: "小雨爸爸", reason: "外出旅游", startDate: "2026-10-12", endDate: "2026-10-18", status: 2, createdAt: "2026-10-11T21:10", reviewedAt: "2026-10-12T10:18", reviewRemark: "请假天数过长，建议缩短到 2 天以内" },
  ];

  let state = root.dataset.state || "normal";
  let filter = "all";
  let rejectTargetId = null;
  let lastRejectTrigger = null;
  let toastTimer = 0;
  const busy = new Set();

  /* ---------- 小工具 ---------- */

  const pad2 = (n) => String(n).padStart(2, "0");

  function parseDate(iso) {
    const parts = iso.split("T")[0].split("-").map(Number);
    return { y: parts[0], m: parts[1], d: parts[2] };
  }

  function parseTime(iso) {
    const t = iso.split("T")[1];
    return t || "00:00";
  }

  function dayDiff(a, b) {
    return Math.round((Date.UTC(b.y, b.m - 1, b.d) - Date.UTC(a.y, a.m - 1, a.d)) / 86400000);
  }

  function formatStamp(iso) {
    if (!iso) return "";
    const date = parseDate(iso);
    const time = parseTime(iso);
    const today = { y: NOW.getFullYear(), m: NOW.getMonth() + 1, d: NOW.getDate() };
    const gap = dayDiff(date, today);
    if (gap === 0) return "今天 " + time;
    if (gap === 1) return "昨天 " + time;
    if (date.y !== today.y) return date.y + "年" + date.m + "月" + date.d + "日";
    return date.m + "月" + date.d + "日 " + time;
  }

  function formatDay(iso) {
    if (!iso) return "";
    const date = parseDate(iso);
    const today = { y: NOW.getFullYear(), m: NOW.getMonth() + 1, d: NOW.getDate() };
    const gap = dayDiff(date, today);
    if (gap === 0) return "今天";
    if (gap === 1) return "昨天";
    return date.m + "月" + date.d + "日";
  }

  function formatRange(item) {
    const start = parseDate(item.startDate);
    const end = parseDate(item.endDate);
    const days = dayDiff(start, end) + 1;
    const sameDay = start.y === end.y && start.m === end.m && start.d === end.d;
    const head = sameDay
      ? start.m + "月" + start.d + "日"
      : start.m + "月" + start.d + "日–" + end.m + "月" + end.d + "日";
    return head + " · 共 " + days + " 天";
  }

  function escapeHtml(text) {
    return String(text == null ? "" : text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  function toast(message) {
    toastEl.textContent = message;
    toastEl.classList.add("is-visible");
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => toastEl.classList.remove("is-visible"), 1800);
  }

  /* ---------- 卡片模板 ---------- */

  function factsMarkup(item) {
    const rows = [
      '<div class="fact"><dt>请假时间</dt><dd>' + escapeHtml(formatRange(item)) + "</dd></div>",
      '<div class="fact"><dt>事由</dt><dd>' + escapeHtml(item.reason) + "</dd></div>",
    ];
    if (item.status === 2) {
      rows.push(
        '<div class="fact is-reject"><dt>驳回理由</dt><dd>' +
          escapeHtml(item.reviewRemark || "未填写理由") +
          "</dd></div>"
      );
    }
    return '<dl class="facts">' + rows.join("") + "</dl>";
  }

  function avatarMarkup(item, history) {
    const initial = item.studentName.slice(0, 1);
    return '<span class="avatar" aria-hidden="true">' + escapeHtml(initial) + "</span>";
  }

  function cardMarkup(item) {
    const history = item.status !== 0;
    const pillClass = item.status === 0 ? "is-pending" : item.status === 1 ? "is-approved" : "is-rejected";
    const timeText = history
      ? "审批 " + formatDay(item.reviewedAt)
      : formatStamp(item.createdAt);

    let actions = "";
    if (!history) {
      const isBusy = busy.has(item.id);
      actions =
        '<div class="card-actions">' +
        '<button class="btn-reject" type="button" data-act="reject" data-id="' + item.id + '"' +
        (isBusy ? " disabled" : "") + ">驳回</button>" +
        '<button class="btn-approve" type="button" data-act="approve" data-id="' + item.id + '"' +
        (isBusy ? " disabled" : "") + ">" +
        (isBusy ? '<span class="btn-spinner" aria-hidden="true"></span>处理中…' : "同意") +
        "</button></div>";
    }

    let trace = "";
    if (history) {
      trace =
        '<div class="trace-hint"><span>审批留痕</span>' +
        '<svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="m5 3 4 4-4 4"/></svg>' +
        "</div>" +
        '<div class="trace" hidden>' +
        '<div class="trace-row"><span>审批时间</span><span>' + escapeHtml(formatStamp(item.reviewedAt)) + "</span></div>" +
        '<div class="trace-row"><span>审批意见</span><span>' + escapeHtml(item.reviewRemark || "未填写审批意见") + "</span></div>" +
        "</div>";
    }

    return (
      '<article class="leave-card ' + (history ? "is-history" : "is-pending") + '" data-id="' + item.id + '">' +
      '<div class="card-head">' +
      avatarMarkup(item, history) +
      '<div class="who"><div class="name-row">' +
      '<h3 class="student-name">' + escapeHtml(item.studentName) + "</h3>" +
      '<span class="status-pill ' + pillClass + '">' + STATUS_LABEL[item.status] + "</span>" +
      "</div>" +
      '<p class="applicant">' + escapeHtml(item.applicant) + " 提交</p></div>" +
      '<span class="card-time">' + escapeHtml(timeText) + "</span>" +
      "</div>" +
      factsMarkup(item) +
      actions +
      trace +
      (history
        ? '<button class="card-hit" type="button" data-act="toggle" data-id="' + item.id +
          '" aria-expanded="false" aria-label="展开 ' + escapeHtml(item.studentName) + ' 的审批留痕"></button>'
        : "") +
      "</article>"
    );
  }

  /* ---------- 统计 / 计数 ---------- */

  function tally() {
    const result = { pending: 0, approved: 0, rejected: 0, total: records.length };
    records.forEach((item) => {
      if (item.status === 0) result.pending += 1;
      else if (item.status === 1) result.approved += 1;
      else result.rejected += 1;
    });
    return result;
  }

  function setStatsDash() {
    Object.keys(statNodes).forEach((key) => {
      statNodes[key].textContent = "—";
    });
  }

  function setStatsZeros() {
    Object.keys(statNodes).forEach((key) => {
      statNodes[key].textContent = "0";
    });
  }

  function setCountsZeros() {
    Object.keys(countNodes).forEach((key) => {
      countNodes[key].textContent = "0";
    });
  }

  function updateStats() {
    const t = tally();
    statNodes.pending.textContent = String(t.pending);
    statNodes.approved.textContent = String(t.approved);
    statNodes.rejected.textContent = String(t.rejected);
    statNodes.total.textContent = String(t.total);
    countNodes.all.textContent = String(t.total);
    countNodes.pending.textContent = String(t.pending);
    countNodes.approved.textContent = String(t.approved);
    countNodes.rejected.textContent = String(t.rejected);
  }

  /* ---------- 渲染 ---------- */

  function filteredRecords() {
    if (filter === "all") return records.slice();
    const wanted = filter === "pending" ? 0 : filter === "approved" ? 1 : 2;
    return records.filter((item) => item.status === wanted);
  }

  /* 单列合并后待审批必须留在第一屏：先按状态（待审批在前），再按提交时间倒序 */
  function sortForDisplay(items) {
    return items.sort((a, b) => {
      const pendingFirst = (a.status === 0 ? 0 : 1) - (b.status === 0 ? 0 : 1);
      return pendingFirst !== 0 ? pendingFirst : b.createdAt.localeCompare(a.createdAt);
    });
  }

  function applyEmptyCopy(kind) {
    if (kind === "demo") {
      emptyTitle.textContent = "暂无请假申请";
      emptySub.textContent = "家长提交后会出现在这里，下拉可刷新";
      emptyAction.textContent = "刷新列表";
    } else if (filter === "pending") {
      emptyTitle.textContent = "暂无待审批申请";
      emptySub.textContent = "全部处理完了，新的提交会出现在这里";
      emptyAction.textContent = "查看全部";
    } else if (filter === "approved") {
      emptyTitle.textContent = "还没有已同意的申请";
      emptySub.textContent = "审批通过后会归档到这里";
      emptyAction.textContent = "查看全部";
    } else if (filter === "rejected") {
      emptyTitle.textContent = "还没有已拒绝的申请";
      emptySub.textContent = "驳回记录会连同理由保留在这里";
      emptyAction.textContent = "查看全部";
    } else {
      emptyTitle.textContent = "暂无请假申请";
      emptySub.textContent = "家长提交后会出现在这里，下拉可刷新";
      emptyAction.textContent = "刷新列表";
    }
  }

  function render() {
    const visible = sortForDisplay(filteredRecords());
    list.innerHTML = visible.map(cardMarkup).join("");

    listTitle.textContent = filter === "all" ? "最近 20 条" : FILTER_LABEL[filter];
    listSub.textContent =
      "共 " + visible.length + " 条 · " +
      (filter === "all" ? "待审批优先，再按提交时间倒序" : "按提交时间倒序");

    const demoEmpty = state === "empty";
    const showEmpty = demoEmpty || (visible.length === 0 && state !== "loading" && state !== "error");
    page.classList.toggle("is-empty", showEmpty);
    if (showEmpty) applyEmptyCopy(demoEmpty ? "demo" : filter);

    if (state === "loading" || state === "error") setStatsDash();
    else if (state === "empty") {
      setStatsZeros();
      setCountsZeros();
    } else updateStats();
  }

  function applyState(next) {
    state = next;
    const locked = state === "loading" || state === "error" || state === "empty";
    filterBar.classList.toggle("is-disabled", locked);
    filterBar.querySelectorAll(".filter-btn").forEach((btn) => {
      btn.setAttribute("aria-disabled", String(locked));
    });
    render();
  }

  document.addEventListener("mp:state", (event) => {
    const next = (event.detail && event.detail.state) || root.dataset.state || "normal";
    applyState(next);
  });

  /* ---------- 分段筛选 ---------- */

  filterBar.addEventListener("click", (event) => {
    const btn = event.target.closest(".filter-btn");
    if (!btn) return;
    if (filterBar.classList.contains("is-disabled")) return;
    filter = btn.dataset.filter || "all";
    filterBar.querySelectorAll(".filter-btn").forEach((item) => {
      const active = item === btn;
      item.classList.toggle("is-active", active);
      item.setAttribute("aria-pressed", String(active));
    });
    render();
  });

  /* ---------- 同意 / 驳回 ---------- */

  function findRecord(id) {
    return records.find((item) => item.id === id);
  }

  function openRejectDialog(id, trigger) {
    const record = findRecord(id);
    if (!record || record.status !== 0) return;
    rejectTargetId = id;
    lastRejectTrigger = trigger || null;
    reasonInput.value = "";
    reasonCount.textContent = "0";
    rejectConfirm.disabled = true;
    rejectConfirm.setAttribute("aria-disabled", "true");
    dialogLayer.hidden = false;
    window.requestAnimationFrame(() => reasonInput.focus());
  }

  function closeRejectDialog() {
    dialogLayer.hidden = true;
    rejectTargetId = null;
    const focusTarget =
      lastRejectTrigger && lastRejectTrigger.isConnected
        ? lastRejectTrigger
        : filterBar.querySelector(".filter-btn.is-active");
    lastRejectTrigger = null;
    if (focusTarget) focusTarget.focus();
  }

  function approve(id) {
    const record = findRecord(id);
    if (!record || record.status !== 0 || busy.has(id)) return;
    busy.add(id);
    render();
    window.setTimeout(() => {
      busy.delete(id);
      record.status = 1;
      record.reviewedAt = NOW_STAMP;
      record.reviewRemark = "";
      render();
      toast("已同意");
    }, 900);
  }

  function reject(reason) {
    const id = rejectTargetId;
    const record = id == null ? null : findRecord(id);
    if (!record || record.status !== 0) {
      closeRejectDialog();
      return;
    }
    record.status = 2;
    record.reviewedAt = NOW_STAMP;
    record.reviewRemark = reason;
    closeRejectDialog();
    render();
    toast("已拒绝");
  }

  list.addEventListener("click", (event) => {
    const btn = event.target.closest("button[data-act]");
    if (!btn) return;
    const id = Number(btn.dataset.id);
    const act = btn.dataset.act;

    if (act === "approve") {
      approve(id);
      return;
    }
    if (act === "reject") {
      openRejectDialog(id, btn);
      return;
    }
    if (act === "toggle") {
      const card = btn.closest(".leave-card");
      const trace = card ? card.querySelector(".trace") : null;
      if (!card || !trace) return;
      const expanded = btn.getAttribute("aria-expanded") === "true";
      const nameNode = card.querySelector(".student-name");
      const name = nameNode ? nameNode.textContent : "该记录";
      btn.setAttribute("aria-expanded", String(!expanded));
      btn.setAttribute("aria-label", (expanded ? "收起 " : "展开 ") + name + " 的审批留痕");
      card.classList.toggle("is-expanded", !expanded);
      trace.hidden = expanded;
    }
  });

  /* ---------- 对话框 ---------- */

  dialogLayer.querySelectorAll("[data-close-dialog]").forEach((node) => {
    node.addEventListener("click", closeRejectDialog);
  });

  reasonInput.addEventListener("input", () => {
    const length = reasonInput.value.length;
    reasonCount.textContent = String(length);
    const empty = reasonInput.value.trim().length === 0;
    rejectConfirm.disabled = empty;
    rejectConfirm.setAttribute("aria-disabled", String(empty));
  });

  rejectConfirm.addEventListener("click", () => {
    const reason = reasonInput.value.trim();
    if (!reason) {
      toast("请填写拒绝原因");
      reasonInput.focus();
      return;
    }
    reject(reason);
  });

  document.addEventListener("keydown", (event) => {
    if (dialogLayer.hidden) return;
    if (event.key === "Escape") {
      event.preventDefault();
      closeRejectDialog();
      return;
    }
    if (event.key !== "Tab") return;
    const focusables = [reasonInput, dialogLayer.querySelector(".dialog-cancel"), rejectConfirm].filter(
      (node) => node && !node.disabled
    );
    if (focusables.length === 0) return;
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  });

  /* ---------- 空态 / 失败态 / 返回 ---------- */

  emptyAction.addEventListener("click", () => {
    if (state === "empty") {
      document.querySelector('.mp-demo-chip[data-mp-state="normal"]').click();
      toast("已刷新");
      return;
    }
    if (filter === "all") {
      document.querySelector('.mp-demo-chip[data-mp-state="normal"]').click();
      toast("已刷新");
      return;
    }
    filter = "all";
    filterBar.querySelectorAll(".filter-btn").forEach((btn) => {
      const active = btn.dataset.filter === "all";
      btn.classList.toggle("is-active", active);
      btn.setAttribute("aria-pressed", String(active));
    });
    render();
  });

  document.getElementById("retryBtn").addEventListener("click", () => {
    document.querySelector('.mp-demo-chip[data-mp-state="normal"]').click();
    toast("已重新加载");
  });

  document.getElementById("backBtn").addEventListener("click", () => {
    toast("返回老师工作台");
  });

  render();
})();
