(() => {
  const app = document.querySelector(".mp-app");
  if (!app) return;

  // The profile prototype has no linked secondary pages yet. Keep placeholder
  // links from jumping the internal scroll container or changing the URL hash.
  app.addEventListener("click", (event) => {
    const link = event.target.closest('a[href="#"]');
    if (!link) return;
    event.preventDefault();
  });
})();
