<script setup lang="ts">
defineProps<{ error: { statusCode: number; statusMessage?: string } }>()
</script>

<template>
  <div class="site">
    <SiteNav />

    <main class="legal">
      <div class="container legal__inner">
        <p class="legal__meta">{{ error.statusCode }}</p>
        <h1>这个页面不在了。</h1>
        <p>链接可能已经失效，或者地址拼错了。</p>
        <p class="legal__action">
          <NuxtLink to="/" class="btn btn--primary">回到首页</NuxtLink>
        </p>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>
