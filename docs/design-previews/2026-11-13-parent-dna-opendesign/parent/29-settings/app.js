(() => {
  const updateRow = document.querySelector('[data-od-id="check-update-row"]');
  const updateStatus = document.getElementById('update-status');
  const toast = document.querySelector('[data-od-id="settings-toast"]');

  if (!updateRow || !updateStatus) return;

  let checkTimer;
  let toastTimer;
  let isChecking = false;

  const showToast = (message) => {
    if (!toast) return;
    window.clearTimeout(toastTimer);
    toast.textContent = message;
    toast.classList.add('is-visible');
    toastTimer = window.setTimeout(() => {
      toast.classList.remove('is-visible');
    }, 2000);
  };

  updateRow.addEventListener('click', () => {
    if (isChecking) return;

    isChecking = true;
    updateRow.setAttribute('aria-busy', 'true');
    updateStatus.textContent = '正在检查';
    window.clearTimeout(checkTimer);

    checkTimer = window.setTimeout(() => {
      isChecking = false;
      updateRow.removeAttribute('aria-busy');
      updateStatus.textContent = '已是最新';
      showToast('当前已是最新版本');
    }, 700);
  });
})();
