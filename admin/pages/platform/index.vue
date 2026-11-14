<template>
  <div class="text-text-muted">正在进入平台控制台…</div>
</template>

<script setup lang="ts">
definePageMeta({ layout: 'platform' })
await navigateTo('/platform/kindergartens', { replace: true })
</script>
