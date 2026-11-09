<template>
  <header class="nav">
    <div class="container nav__inner">
      <NuxtLink to="/" class="nav__brand" aria-label="养成星球首页">
        <img src="/brand/horizontal-color.svg" alt="养成星球" width="960" height="256" />
      </NuxtLink>

      <nav class="nav__links" aria-label="页面导航">
        <NuxtLink to="/#product">产品</NuxtLink>
        <NuxtLink to="/#privacy">隐私</NuxtLink>
        <NuxtLink to="/#contact">联系</NuxtLink>
      </nav>

      <NuxtLink to="/#contact" class="btn btn--primary btn--sm nav__cta">预约演示</NuxtLink>
    </div>
  </header>
</template>
