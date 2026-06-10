<template>
  <div class="admin-shell min-h-screen flex bg-slate-100">
    <Sidebar />
    <div class="flex flex-1 flex-col min-w-0">
      <Topbar />
      <main class="flex-1 overflow-auto p-6">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts"></script>
