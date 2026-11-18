(function () {
  "use strict";

  var MAX_EVIDENCE = 3;
  var form = document.getElementById("leaveForm");
  var startDate = document.getElementById("startDate");
  var endDate = document.getElementById("endDate");
  var startDateLabel = document.querySelector('[data-date-label="start"]');
  var endDateLabel = document.querySelector('[data-date-label="end"]');
  var endDateCol = document.getElementById("endDateCol");
  var endDateError = document.getElementById("endDateError");
  var dateOrderError = document.getElementById("dateOrderError");
  var dateLimitError = document.getElementById("dateLimitError");
  var reason = document.getElementById("leaveReason");
  var reasonShell = document.getElementById("reasonShell");
  var reasonError = document.getElementById("reasonError");
  var reasonCount = document.getElementById("reasonCount");
  var evidenceTrigger = document.getElementById("evidenceTrigger");
  var evidenceInput = document.getElementById("evidenceInput");
  var uploadTitle = document.getElementById("uploadTitle");
  var toast = document.getElementById("toast");
  var backButton = document.getElementById("backButton");
  var toastTimer;
  var params = new URLSearchParams(window.location.search);

  function formatDate(value) {
    if (!value) {
      return "请选择";
    }
    var parts = value.split("-");
    if (parts.length !== 3) {
      return "请选择";
    }
    var month = Number(parts[1]);
    var day = Number(parts[2]);
    if (!month || !day) {
      return "请选择";
    }
    return month + " 月 " + day + " 日";
  }

  function syncDateLabel(input, label) {
    label.textContent = formatDate(input.value);
  }

  function setEndDateError(active) {
    endDateCol.classList.toggle("is-invalid", active);
    endDateError.hidden = !active;
    endDate.setAttribute("aria-invalid", active ? "true" : "false");
    if (active) {
      endDate.setAttribute("aria-describedby", "endDateError");
    } else {
      endDate.removeAttribute("aria-describedby");
    }
  }

  function setReasonError(active) {
    reasonShell.classList.toggle("is-invalid", active);
    reasonError.hidden = !active;
    reason.setAttribute("aria-invalid", active ? "true" : "false");
    if (active) {
      reason.setAttribute("aria-describedby", "reasonError");
    } else {
      reason.removeAttribute("aria-describedby");
    }
  }

  function syncReasonCount() {
    reasonCount.textContent = String(reason.value.length);
  }

  function showToast(message) {
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add("is-visible");
    toastTimer = window.setTimeout(function () {
      toast.classList.remove("is-visible");
    }, 2000);
  }

  function capEvidenceFiles() {
    if (!evidenceInput.files || evidenceInput.files.length <= MAX_EVIDENCE) {
      return;
    }
    if (typeof DataTransfer === "undefined") {
      return;
    }
    var transfer = new DataTransfer();
    Array.prototype.slice.call(evidenceInput.files, 0, MAX_EVIDENCE).forEach(function (file) {
      transfer.items.add(file);
    });
    evidenceInput.files = transfer.files;
  }

  syncDateLabel(startDate, startDateLabel);
  syncDateLabel(endDate, endDateLabel);
  syncReasonCount();
  setEndDateError(false);
  setReasonError(false);

  // 从 10 请假详情「重新提交」进入时，带出原申请内容，家长可改后再提交。
  function applyResubmitPrefill() {
    if (params.get("from") !== "10") {
      return;
    }

    var type = params.get("type");
    if (type) {
      var radio = document.querySelector('input[name="leaveType"][value="' + type + '"]');
      if (radio) {
        radio.checked = true;
      }
    }

    var startValue = params.get("start");
    if (startValue) {
      startDate.value = startValue;
      syncDateLabel(startDate, startDateLabel);
    }

    var endValue = params.get("end");
    if (endValue) {
      endDate.value = endValue;
      syncDateLabel(endDate, endDateLabel);
    }

    var reasonValue = params.get("reason");
    if (reasonValue) {
      var limit = Number(reason.maxLength) || 120;
      reason.value = reasonValue.slice(0, limit);
      syncReasonCount();
    }
  }

  applyResubmitPrefill();

  if (params.get("state") === "error") {
    endDate.value = "";
    syncDateLabel(endDate, endDateLabel);
    setEndDateError(true);
    setReasonError(true);
  }

  startDate.addEventListener("change", function () {
    syncDateLabel(startDate, startDateLabel);
    dateOrderError.hidden = true;
    dateLimitError.hidden = true;
  });

  endDate.addEventListener("change", function () {
    syncDateLabel(endDate, endDateLabel);
    setEndDateError(!endDate.value);
    dateOrderError.hidden = true;
    dateLimitError.hidden = true;
  });

  document.querySelectorAll(".date-input").forEach(function (control) {
    control.addEventListener("click", function (event) {
      var input = control.querySelector(".date-native");
      if (typeof input.showPicker !== "function") {
        return;
      }
      event.preventDefault();
      try {
        input.showPicker();
      } catch (error) {
        input.focus({ preventScroll: true });
      }
    });
  });

  reason.addEventListener("input", function () {
    syncReasonCount();
    if (reason.value.trim().length > 0) {
      setReasonError(false);
    } else if (!reasonError.hidden) {
      setReasonError(true);
    }
  });

  evidenceTrigger.addEventListener("click", function () {
    evidenceInput.click();
  });

  evidenceInput.addEventListener("change", function () {
    capEvidenceFiles();
    var count = evidenceInput.files ? evidenceInput.files.length : 0;
    if (count === 0) {
      uploadTitle.textContent = "上传就医凭证（选填）";
      return;
    }
    uploadTitle.textContent = "已选择 " + count + " 张就医凭证";
  });

  form.addEventListener("submit", function (event) {
    event.preventDefault();
    var hasEndDate = Boolean(endDate.value);
    var hasReason = reason.value.trim().length > 0;
    var firstInvalid = null;

    setEndDateError(!hasEndDate);
    setReasonError(!hasReason);

    var start = startDate.value ? new Date(startDate.value + "T00:00:00") : null;
    var end = endDate.value ? new Date(endDate.value + "T00:00:00") : null;
    var today = new Date(2026, 10, 12);
    var orderInvalid = Boolean(start && end && end < start);
    var pastInvalid = Boolean(start && start < today);
    var limitInvalid = Boolean(start && end && (end - start) / 86400000 > 30);
    dateOrderError.hidden = !orderInvalid;
    dateLimitError.hidden = !(pastInvalid || limitInvalid);
    if (pastInvalid) dateLimitError.textContent = "不能选择过去的日期";
    else if (limitInvalid) dateLimitError.textContent = "一次请假最多 30 天";

    if (!hasEndDate) {
      firstInvalid = endDate;
    } else if (orderInvalid || pastInvalid) {
      firstInvalid = startDate;
    } else if (limitInvalid) {
      firstInvalid = endDate;
    } else if (!hasReason) {
      firstInvalid = reason;
    }

    if (firstInvalid) {
      window.requestAnimationFrame(function () {
        firstInvalid.focus({ preventScroll: true });
      });
      return;
    }

    showToast("请假申请已提交，老师会尽快审批");
  });

  backButton.addEventListener("click", function () {
    if (window.history.length > 1) {
      window.history.back();
    }
  });
})();
