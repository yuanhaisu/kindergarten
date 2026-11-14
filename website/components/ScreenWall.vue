<template>
  <div class="wall" role="group" aria-label="养成星球产品界面示意">
    <figure class="wall__item wall__item--teacher">
      <div class="phone">
        <img
          src="/screens/teacher-workbench.webp"
          alt="产品界面示意：老师端工作台"
          width="780"
          height="1400"
          fetchpriority="high"
        />
      </div>
      <figcaption>老师端 · 工作台</figcaption>
    </figure>

    <figure class="wall__item wall__item--parent">
      <div class="phone">
        <img
          src="/screens/parent-home.webp"
          alt="产品界面示意：家长端首页"
          width="780"
          height="1400"
          loading="lazy"
          decoding="async"
        />
      </div>
      <figcaption>家长端 · 首页</figcaption>
    </figure>

    <figure class="wall__item wall__item--admin">
      <div class="browser">
        <div class="browser__bar" aria-hidden="true">
          <span class="browser__dot" />
          <span class="browser__dot" />
          <span class="browser__dot" />
          <span class="browser__url">园所后台 · 数据看板</span>
        </div>
        <img
          src="/screens/admin-dashboard.webp"
          alt="产品界面示意：园所后台数据看板"
          width="1440"
          height="900"
          loading="lazy"
          decoding="async"
        />
      </div>
      <figcaption>园所后台 · 数据看板</figcaption>
    </figure>
  </div>
</template>
