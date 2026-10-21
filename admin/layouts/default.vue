<template>
  <div class="admin-shell min-h-screen flex bg-slate-100">
    <Sidebar />
    <div class="flex flex-1 flex-col min-w-0">
      <Topbar />
      <UrgencyBanner />
      <main class="flex-1 overflow-auto p-6">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
const pickupAlert = usePickupAlertStore()

// 壳层常驻的滞留信号：进入后台即开始轮询，退出后台布局即停止
onMounted(() => pickupAlert.start())
onUnmounted(() => pickupAlert.stop())
</script>
