<script setup lang="ts">
import { site } from '~/content/site'

const year = new Date().getFullYear()
</script>

<template>
  <footer class="footer">
    <div class="container">
      <div class="footer__row">
        <div>
          <img class="footer__logo" src="/brand/horizontal-color.svg" alt="养成星球" />
          <p class="footer__entity">{{ site.legalEntity || '主体名称 · 待补充' }}</p>
        </div>

        <nav class="footer__links" aria-label="页脚导航">
          <NuxtLink to="/privacy">隐私政策</NuxtLink>
          <NuxtLink to="/terms">服务条款</NuxtLink>
          <NuxtLink to="/#contact">联系我们</NuxtLink>
        </nav>

        <div class="footer__meta">
          <span>{{ site.icp || 'ICP 备案号 · 待补充' }}</span>
          <span>© {{ year }} {{ site.name }}</span>
        </div>
      </div>
    </div>
  </footer>
</template>
