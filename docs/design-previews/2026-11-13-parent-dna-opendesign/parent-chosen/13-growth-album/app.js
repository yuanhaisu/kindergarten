(() => {
  const filters = Array.from(document.querySelectorAll(".filter-chip"));
  const months = Array.from(document.querySelectorAll(".album-month"));
  const main = document.querySelector(".mp-main");

  if (!filters.length || !months.length) return;

  const validFilters = new Set(filters.map((button) => button.dataset.filter));
  const requestedFilter = new URLSearchParams(window.location.search).get("filter");
  let activeFilter = validFilters.has(requestedFilter) ? requestedFilter : "all";

  function applyFilter(nextFilter) {
    activeFilter = nextFilter;

    filters.forEach((button) => {
      button.setAttribute(
        "aria-pressed",
        String(button.dataset.filter === activeFilter)
      );
    });

    months.forEach((month) => {
      let visibleTiles = 0;

      month.querySelectorAll(".album-tile").forEach((tile) => {
        const matches =
          activeFilter === "all" || tile.dataset.kind === activeFilter;
        tile.hidden = !matches;
        if (matches) visibleTiles += 1;
      });

      month.hidden = visibleTiles === 0;
    });

    if (main) main.scrollTop = 0;
  }

  filters.forEach((button) => {
    button.addEventListener("click", () => {
      const nextFilter = button.dataset.filter;
      if (!nextFilter || nextFilter === activeFilter) return;
      applyFilter(nextFilter);
    });
  });

  applyFilter(activeFilter);
})();
