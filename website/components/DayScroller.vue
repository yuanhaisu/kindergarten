<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { day } from '~/content/site'

const active = ref(0)
let observer: IntersectionObserver | null = null

onMounted(() => {
  const items = Array.from(document.querySelectorAll<HTMLElement>('[data-moment]'))
  if (!items.length || typeof IntersectionObserver === 'undefined') return

  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (!entry.isIntersecting) continue
        const index = Number((entry.target as HTMLElement).dataset.moment)
        if (!Number.isNaN(index)) active.value = index
      }
    },
    // 只有越过视口中线的那一段才算当前时刻，避免相邻两段来回抢。
    { rootMargin: '-45% 0px -45% 0px', threshold: 0 },
  )

  items.forEach((item) => observer?.observe(item))
})

onBeforeUnmount(() => observer?.disconnect())
</script>

<template>
  <section id="day" class="day">
    <div class="container">
      <header class="day__head">
        <h2 class="section-title">{{ day.heading }}</h2>
        <p class="section-lead">{{ day.lead }}</p>
      </header>

      <div class="day__stage">
        <div class="day__phone phone" aria-hidden="true">
          <div class="day__frames">
            <img
              v-for="(moment, index) in day.moments"
              :key="moment.title"
              :src="moment.image"
              :class="{ 'is-active': active === index }"
              :loading="index === 0 ? 'eager' : 'lazy'"
              width="780"
              height="1400"
              alt=""
            />
          </div>
        </div>

        <ol class="day__moments">
          <li
            v-for="(moment, index) in day.moments"
            :key="moment.title"
            class="day__moment"
            :class="{ 'is-active': active === index }"
            :data-moment="index"
          >
            <time class="day__time">{{ moment.time }}</time>
            <h3>{{ moment.title }}</h3>
            <p>{{ moment.body }}</p>
            <img
              class="day__inline"
              :src="moment.image"
              :alt="moment.alt"
              width="780"
              height="1400"
              loading="lazy"
            />
          </li>
        </ol>
      </div>

      <p class="note day__note">{{ day.note }}</p>
    </div>
  </section>
</template>
