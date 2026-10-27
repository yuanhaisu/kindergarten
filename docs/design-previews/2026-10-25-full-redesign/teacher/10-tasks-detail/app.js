(() => {
  const app = document.querySelector(".mp-app");
  if (!app) return;
  const page = document.querySelector(".mp-main.page");
  if (!page) return;

  const $ = (sel, root = document) => root.querySelector(sel);
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel));

  /* ---------- toast ---------- */

  const toastEl = $("#toast");
  let toastTimer = null;

  function toast(message) {
    if (!toastEl) return;
    toastEl.textContent = message;
    toastEl.hidden = false;
    window.requestAnimationFrame(() => toastEl.classList.add("is-show"));
    window.clearTimeout(toastTimer);
    toastTimer = window.setTimeout(() => {
      toastEl.classList.remove("is-show");
      window.setTimeout(() => { toastEl.hidden = true; }, 220);
    }, 1800);
  }

  /* ---------- 分段与筛选 ---------- */

  const tabButtons = $$("[data-tab]");
  const lists = $$("[data-list]");
  const filterChips = $$("[data-review-filter]");
  const filterEmpty = $("#filterEmpty");
  const filterJump = $("#filterJump");
  let activeTab = "pending";
  let reviewFilter = "all";

  function setTab(tab) {
    activeTab = tab;
    tabButtons.forEach((btn) => {
      const on = btn.dataset.tab === tab;
      btn.classList.toggle("is-active", on);
      btn.setAttribute("aria-selected", String(on));
    });
    lists.forEach((list) => { list.hidden = list.dataset.list !== tab; });
    const batch = $("#batchStrip");
    if (batch) batch.hidden = tab !== "pending";
    applyFilter();
  }

  function applyFilter() {
    const list = lists.find((item) => item.dataset.list === activeTab);
    if (!list) return;

    let visible = 0;
    $$(".checkin", list).forEach((card) => {
      const match = reviewFilter === "all" || card.dataset.review === reviewFilter;
      card.hidden = !match;
      if (match) visible += 1;
    });

    const other = lists.find((item) => item.dataset.list !== activeTab);
    let otherVisible = 0;
    if (other) {
      otherVisible = $$(".checkin", other).filter((card) => (
        reviewFilter === "all" || card.dataset.review === reviewFilter
      )).length;
    }

    if (filterEmpty) {
      filterEmpty.hidden = visible !== 0;
      if (visible === 0 && filterJump) {
        filterJump.hidden = otherVisible === 0;
        filterJump.textContent = otherVisible
          ? `切到「${activeTab === "pending" ? "已回应" : "待回应"}」看 ${otherVisible} 条`
          : "";
      }
    }

    updateReviewCounts();
  }

  function updateReviewCounts() {
    const counts = { all: 0, pending: 0, approved: 0, revoked: 0 };
    $$(".checkin").forEach((card) => {
      counts.all += 1;
      const key = card.dataset.review;
      if (key in counts) counts[key] += 1;
    });
    $$("[data-review-count]").forEach((el) => {
      el.textContent = String(counts[el.dataset.reviewCount] || 0);
    });
  }

  tabButtons.forEach((btn) => {
    btn.addEventListener("click", () => setTab(btn.dataset.tab));
  });

  filterChips.forEach((chip) => {
    chip.addEventListener("click", () => {
      reviewFilter = chip.dataset.reviewFilter;
      filterChips.forEach((item) => {
        const on = item === chip;
        item.classList.toggle("is-active", on);
        item.setAttribute("aria-pressed", String(on));
      });
      applyFilter();
    });
  });

  if (filterJump) {
    filterJump.addEventListener("click", () => {
      setTab(activeTab === "pending" ? "responded" : "pending");
    });
  }

  /* ---------- 审核状态 ---------- */

  const idleHtml = new WeakMap();

  function lockCard(card, activeBtn, label) {
    $$("button", card).forEach((btn) => { btn.disabled = true; });
    if (!activeBtn) return;
    idleHtml.set(activeBtn, activeBtn.innerHTML);
    activeBtn.innerHTML = `<span class="spinner" aria-hidden="true"></span>${label}`;
  }

  function unlockCard(card) {
    $$("button", card).forEach((btn) => {
      btn.disabled = false;
      if (idleHtml.has(btn)) {
        btn.innerHTML = idleHtml.get(btn);
        idleHtml.delete(btn);
      }
    });
  }

  function setReview(card, status) {
    if (!card) return;
    card.dataset.review = status;
    const block = $(".review-block", card);
    if (block) block.dataset.status = status;
  }

  $$("[data-scope]").forEach((scope) => {
    scope.addEventListener("click", () => {
      const on = scope.getAttribute("aria-checked") !== "true";
      scope.setAttribute("aria-checked", String(on));
      const card = scope.closest(".checkin");
      const approve = card ? $("[data-approve]", card) : null;
      if (approve) approve.disabled = !on;
    });
  });

  $$("[data-approve]").forEach((btn) => {
    btn.addEventListener("click", () => {
      if (btn.disabled) return;
      const card = btn.closest(".checkin");
      lockCard(card, btn, "批准中…");
      window.setTimeout(() => {
        setReview(card, "approved");
        unlockCard(card);
        toast("已批准公开");
        applyFilter();
      }, 800);
    });
  });

  /* ---------- 居中确认框 ---------- */

  const dialog = $("#confirmDialog");
  const dialogTitle = $("#dialogTitle");
  const dialogBody = $("#dialogBody");
  const dialogConfirm = $("#dialogConfirm");
  const publicSwitch = $("#publicSwitch");
  let dialogAction = null;
  // 公开开关在编辑弹层里只暂存，点「保存」才落到页面；取消/关闭弹层不改动已公开状态。
  let pendingPublic = null;

  function closeConfirm() {
    if (!dialog) return;
    dialog.hidden = true;
    dialogAction = null;
  }

  function openConfirm(kind, card) {
    if (!dialog) return;

    if (kind === "reject") {
      dialogTitle.textContent = "拒绝公开申请？";
      dialogBody.textContent = "拒绝后家长需要重新申请，这条内容不会出现在班级墙。";
      dialogConfirm.textContent = "拒绝";
      dialogAction = () => {
        const btn = $("[data-reject]", card);
        lockCard(card, btn, "处理中…");
        window.setTimeout(() => {
          setReview(card, "revoked");
          unlockCard(card);
          toast("已拒绝公开申请");
          applyFilter();
        }, 800);
      };
    } else if (kind === "revoke") {
      dialogTitle.textContent = "撤回公开？";
      dialogBody.textContent = "撤回后本班家长将看不到这条内容，家长需要重新申请。";
      dialogConfirm.textContent = "确认撤回";
      dialogAction = () => {
        const btn = $("[data-revoke]", card);
        lockCard(card, btn, "处理中…");
        window.setTimeout(() => {
          setReview(card, "revoked");
          unlockCard(card);
          toast("已撤回公开");
          applyFilter();
        }, 800);
      };
    } else {
      dialogTitle.textContent = "关闭公开？";
      dialogBody.textContent = "关闭并保存后，已公开内容将暂不对本班展示，家长需要重新申请才能公开。";
      dialogConfirm.textContent = "确认关闭";
      dialogAction = () => {
        pendingPublic = false;
        if (publicSwitch) publicSwitch.setAttribute("aria-checked", "false");
        toast("已关闭公开，保存后生效");
      };
    }

    dialog.hidden = false;
    if (dialogConfirm) dialogConfirm.focus({ preventScroll: true });
  }

  $$("[data-reject]").forEach((btn) => {
    btn.addEventListener("click", () => openConfirm("reject", btn.closest(".checkin")));
  });

  $$("[data-revoke]").forEach((btn) => {
    btn.addEventListener("click", () => openConfirm("revoke", btn.closest(".checkin")));
  });

  if (dialogConfirm) {
    dialogConfirm.addEventListener("click", () => {
      const action = dialogAction;
      closeConfirm();
      if (action) action();
    });
  }

  $$("[data-dialog-cancel]").forEach((btn) => {
    btn.addEventListener("click", closeConfirm);
  });

  if (publicSwitch) {
    publicSwitch.addEventListener("click", () => {
      const on = publicSwitch.getAttribute("aria-checked") === "true";
      if (on) {
        openConfirm("publicOff", null);
      } else {
        pendingPublic = true;
        publicSwitch.setAttribute("aria-checked", "true");
        toast("已开启公开，保存后生效");
      }
    });
  }

  /* ---------- 弹层 ---------- */

  function openSheet(id) {
    const sheet = document.getElementById(id);
    if (!sheet) return;
    sheet.hidden = false;
    const focusTarget = $(".sheet-close", sheet);
    if (focusTarget) focusTarget.focus({ preventScroll: true });
  }

  function closeSheet(id) {
    const sheet = document.getElementById(id);
    if (sheet) sheet.hidden = true;
  }

  $$("[data-close]").forEach((btn) => {
    btn.addEventListener("click", () => closeSheet(btn.dataset.close));
  });

  /* ---------- 照片预览 ---------- */

  $$("[data-media]").forEach((tile) => {
    tile.addEventListener("click", () => {
      const title = $("#mediaTitle");
      const sub = $("#mediaSub");
      const preview = $("#mediaPreview");
      if (title) title.textContent = `${tile.dataset.student}的照片`;
      if (sub) sub.textContent = `${tile.dataset.time} · 家长提交`;
      if (preview) preview.dataset.tone = tile.dataset.tone || "warm";
      openSheet("mediaSheet");
    });
  });

  /* ---------- 编辑任务 ---------- */

  const titleInput = $("#taskTitleInput");
  const descInput = $("#taskDescInput");
  const linkInput = $("#taskLinkInput");

  function openEdit() {
    if (titleInput) titleInput.value = $(".hero-title").textContent.trim();
    if (descInput) descInput.value = $(".hero-desc").textContent.trim();
    if (linkInput) linkInput.value = $(".hero-learning-text").textContent.trim();
    pendingPublic = page.dataset.public !== "off";
    if (publicSwitch) {
      publicSwitch.setAttribute("aria-checked", String(pendingPublic));
    }
    openSheet("editSheet");
  }

  const editTaskBtn = $("#editTaskBtn");
  if (editTaskBtn) editTaskBtn.addEventListener("click", openEdit);

  const saveTaskBtn = $("#saveTaskBtn");
  if (saveTaskBtn) {
    saveTaskBtn.addEventListener("click", () => {
      if (!titleInput || !descInput || !linkInput) return;
      const title = titleInput.value.trim();
      if (!title) {
        toast("任务标题不能为空");
        titleInput.focus();
        return;
      }
      // 源码 saveEdit 要求标题与学习关联同时非空（detail.vue:512-515）。
      const learningLink = linkInput.value.trim();
      if (!learningLink) {
        toast("学习关联不能为空");
        linkInput.focus();
        return;
      }
      $(".hero-title").textContent = title;
      $(".hero-desc").textContent = descInput.value.trim();
      $(".hero-learning-text").textContent = learningLink;
      // 暂存的公开开关只在保存时生效，与源码「先改 edit.allow_public、保存才提交」一致。
      const nextPublic = pendingPublic === null ? page.dataset.public !== "off" : pendingPublic;
      page.dataset.public = nextPublic ? "on" : "off";
      if (publicSwitch) publicSwitch.setAttribute("aria-checked", String(nextPublic));
      closeSheet("editSheet");
      toast("已保存");
    });
  }

  /* ---------- 生成小程序码 ---------- */

  function generateShare() {
    const cell = $("#shareTaskBtn");
    const value = $("#shareValue");
    if (!cell || !value || cell.disabled) return;
    cell.disabled = true;
    value.textContent = "生成中…";
    window.setTimeout(() => {
      cell.disabled = false;
      value.textContent = "转发班级群";
      toast("已保存到相册，可转发班级群");
    }, 900);
  }

  const shareTaskBtn = $("#shareTaskBtn");
  if (shareTaskBtn) shareTaskBtn.addEventListener("click", generateShare);

  const emptyShareBtn = $("#emptyShareBtn");
  if (emptyShareBtn) {
    emptyShareBtn.addEventListener("click", () => {
      if (emptyShareBtn.disabled) return;
      const original = emptyShareBtn.textContent;
      emptyShareBtn.disabled = true;
      emptyShareBtn.textContent = "生成中…";
      window.setTimeout(() => {
        emptyShareBtn.disabled = false;
        emptyShareBtn.textContent = original;
        toast("已保存到相册，可转发班级群");
      }, 900);
    });
  }

  const emptyRefreshBtn = $("#emptyRefreshBtn");
  if (emptyRefreshBtn) emptyRefreshBtn.addEventListener("click", () => toast("已刷新"));

  /* ---------- 批量回应 ---------- */

  const batchBtn = $("#batchBtn");
  if (batchBtn) {
    batchBtn.addEventListener("click", () => {
      if (batchBtn.disabled) return;
      const original = batchBtn.innerHTML;
      batchBtn.disabled = true;
      batchBtn.innerHTML = '<span class="spinner" aria-hidden="true"></span><span class="batch-label">汇总中…</span>';
      window.setTimeout(() => {
        batchBtn.disabled = false;
        batchBtn.innerHTML = original;
        toast("已回应 3 条");
      }, 900);
    });
  }

  /* ---------- 录音（原型） ---------- */

  let recording = null;

  $$("[data-record]").forEach((btn) => {
    btn.addEventListener("click", () => {
      if (recording && recording.btn !== btn) {
        toast("请先停止当前录音");
        return;
      }

      if (recording) {
        const seconds = Math.max(1, Math.round((Date.now() - recording.started) / 1000));
        window.clearInterval(recording.timer);
        recording.btn.classList.remove("is-recording");
        recording.btn.textContent = "录音";
        recording.btn.setAttribute("aria-label", "录音回应");
        recording = null;
        toast(`语音已发送（${seconds} 秒）`);
        return;
      }

      btn.classList.add("is-recording");
      btn.setAttribute("aria-label", "停止录音");
      const started = Date.now();
      const render = () => {
        const seconds = Math.floor((Date.now() - started) / 1000);
        const mm = Math.floor(seconds / 60);
        const ss = String(seconds % 60).padStart(2, "0");
        btn.innerHTML = `<span class="rec-dot" aria-hidden="true"></span><span>停止 ${mm}:${ss}</span>`;
      };
      render();
      const timer = window.setInterval(render, 500);
      recording = { btn, started, timer };
    });
  });

  /* ---------- 语音播放（原型） ---------- */

  $$("[data-voice]").forEach((btn) => {
    let timer = null;
    btn.addEventListener("click", () => {
      if (timer) return;
      const label = $(".voice-label", btn);
      if (!label) return;
      const original = label.textContent;
      label.textContent = "正在播放…";
      btn.classList.add("is-playing");
      timer = window.setTimeout(() => {
        label.textContent = original;
        btn.classList.remove("is-playing");
        timer = null;
      }, 1400);
    });
  });

  /* ---------- 补充回应 ---------- */

  $$("[data-supplement-toggle]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const block = btn.closest(".response-block");
      if (!block) return;
      const panel = $(".supplement-panel", block);
      if (!panel) return;
      const willOpen = panel.hidden;
      panel.hidden = !willOpen;
      $$("[data-supplement-toggle]", block).forEach((item) => {
        item.setAttribute("aria-expanded", String(willOpen));
      });
    });
  });

  /* ---------- 通用 toast 按钮 ---------- */

  $$("[data-toast]").forEach((btn) => {
    btn.addEventListener("click", () => toast(btn.dataset.toast));
  });

  /* ---------- 四态 ---------- */

  const NORMAL_STATS = { total: "28", submitted: "6", pending: "3", responded: "3" };
  const EMPTY_STATS = { total: "28", submitted: "0", pending: "0", responded: "0" };

  function applyStats(state) {
    const stats = state === "empty" ? EMPTY_STATS : NORMAL_STATS;
    $$("[data-stat]").forEach((el) => {
      el.textContent = stats[el.dataset.stat] || "0";
    });
    const pendingCount = $('[data-tab-count="pending"]');
    const respondedCount = $('[data-tab-count="responded"]');
    if (pendingCount) pendingCount.textContent = stats.pending;
    if (respondedCount) respondedCount.textContent = stats.responded;
  }

  function setState(state) {
    app.dataset.state = state;
    $$(".mp-demo-chip").forEach((chip) => {
      chip.setAttribute("aria-pressed", String(chip.dataset.mpState === state));
    });
    applyStats(state);
  }

  document.addEventListener("mp:state", (event) => {
    applyStats(event.detail.state);
  });

  const retryBtn = $("#retryBtn");
  if (retryBtn) retryBtn.addEventListener("click", () => setState("normal"));

  const backBtn = $("#backBtn");
  if (backBtn) backBtn.addEventListener("click", () => toast("返回亲子任务列表"));

  /* ---------- 键盘 ---------- */

  document.addEventListener("keydown", (event) => {
    if (event.key !== "Escape") return;
    if (dialog && !dialog.hidden) {
      closeConfirm();
      return;
    }
    const open = $$(".sheet-layer").find((layer) => !layer.hidden);
    if (open) closeSheet(open.id);
  });

  /* ---------- 初始化 ---------- */

  setTab("pending");
  applyStats(app.dataset.state || "normal");
  applyFilter();
})();
