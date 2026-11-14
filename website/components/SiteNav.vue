<template>
  <header class="nav">
    <div class="container nav__inner">
      <NuxtLink to="/" class="nav__brand" aria-label="养成星球首页">
        <img src="/brand/horizontal-color.svg" alt="养成星球" width="960" height="256" />
      </NuxtLink>

      <nav class="nav__links" aria-label="页面导航">
        <NuxtLink
          to="/#product"
          :class="{ 'is-active': activeSection === 'product' }"
          :aria-current="activeSection === 'product' ? 'location' : undefined"
        >产品</NuxtLink>
        <NuxtLink
          to="/#privacy"
          :class="{ 'is-active': activeSection === 'privacy' }"
          :aria-current="activeSection === 'privacy' ? 'location' : undefined"
        >隐私</NuxtLink>
        <NuxtLink
          to="/#contact"
          :class="{ 'is-active': activeSection === 'contact' }"
          :aria-current="activeSection === 'contact' ? 'location' : undefined"
        >联系</NuxtLink>
      </nav>

      <NuxtLink to="/#contact" class="btn btn--primary btn--sm nav__cta">预约演示</NuxtLink>
    </div>
  </header>
</template>

<script setup lang="ts">
const route = useRoute()
const activeSection = ref('')
let observer: IntersectionObserver | null = null

// 只有首页有这些区块；法务页上保持无高亮。
const sectionIds = ['product', 'privacy', 'contact']

function teardown() {
  observer?.disconnect()
  observer = null
}

function setup() {
  teardown()
  activeSection.value = ''
  if (route.path !== '/' || typeof IntersectionObserver === 'undefined') return

  const targets = sectionIds
    .map((id) => document.getElementById(id))
    .filter((el): el is HTMLElement => Boolean(el))
  if (!targets.length) return

  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) activeSection.value = entry.target.id
      }
    },
    // 视口中线附近的一条窄带：同一时刻至多一个区块落在带内。
    // 落在两个区块的间隙时保留上一次命中，避免滚动途中高亮闪断。
    { rootMargin: '-45% 0px -45% 0px', threshold: 0 },
  )
  targets.forEach((el) => observer!.observe(el))
}

onMounted(() => nextTick(setup))
watch(() => route.path, () => nextTick(setup))
onBeforeUnmount(teardown)
</script>
