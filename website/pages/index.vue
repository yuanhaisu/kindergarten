<script setup lang="ts">
import { contact, hero, modules, plain, privacy, record, site } from '~/content/site'

const canonical = `${site.siteUrl}/`
const structuredData = {
  '@context': 'https://schema.org',
  '@graph': [
    {
      '@type': 'Organization',
      '@id': `${site.siteUrl}/#organization`,
      name: site.name,
      legalName: site.legalEntity,
      url: canonical,
      logo: `${site.siteUrl}/icon-512.png`,
      email: site.contact.email,
    },
    {
      '@type': 'WebSite',
      '@id': `${site.siteUrl}/#website`,
      url: canonical,
      name: site.name,
      inLanguage: 'zh-CN',
      publisher: { '@id': `${site.siteUrl}/#organization` },
    },
    {
      '@type': 'SoftwareApplication',
      name: site.name,
      applicationCategory: 'BusinessApplication',
      operatingSystem: 'Web, 微信小程序',
      description: '面向民办幼儿园的家园共育与园务管理平台。',
      url: canonical,
      publisher: { '@id': `${site.siteUrl}/#organization` },
    },
  ],
}

useHead({
  link: [{ rel: 'canonical', href: canonical }],
  meta: [
    { property: 'og:url', content: canonical },
    { property: 'og:site_name', content: site.name },
  ],
  script: [{ type: 'application/ld+json', innerHTML: JSON.stringify(structuredData) }],
})

const mailtoHref = `mailto:${site.contact.email}?subject=${encodeURIComponent(contact.emailSubject)}`
</script>

<template>
  <div class="site">
    <SiteNav />

    <main>
      <section class="hero">
        <div class="container hero__inner">
          <div class="hero__copy">
            <h1 class="hero__title rise">
              <span class="hero__title-line">{{ hero.title }}，</span>
              <span class="hero__title-line">
                <span class="mark"><span>{{ hero.titleAccent }}</span></span>。
              </span>
            </h1>

            <p class="hero__lead rise rise--1">{{ hero.lead }}</p>

            <div class="hero__actions rise rise--2">
              <a class="btn btn--primary" href="#contact">{{ hero.primary }}</a>
              <a class="btn btn--ghost" href="#day">{{ hero.secondary }}</a>
            </div>

            <p class="hero__note note rise rise--3">{{ hero.note }}</p>
          </div>

          <ScreenWall />
        </div>
      </section>

      <section class="record">
        <div class="container">
          <h2 class="section-title">{{ record.heading }}</h2>
          <p class="section-lead">{{ record.lead }}</p>

          <ol class="record__steps">
            <li v-for="step in record.steps" :key="step.label" class="record__step">
              <span class="record__dot" aria-hidden="true" />
              <span class="record__time">{{ step.time }}</span>
              <span class="record__label">{{ step.label }}</span>
              <span class="record__detail">{{ step.detail }}</span>
            </li>
          </ol>

          <p class="note record__note">{{ record.note }}</p>
        </div>
      </section>

      <DayScroller />

      <section id="product" class="modules">
        <div class="container">
          <h2 class="section-title">{{ modules.heading }}</h2>
          <p class="section-lead">{{ modules.lead }}</p>

          <dl class="modules__sheet">
            <div v-for="row in modules.rows" :key="row.role" class="modules__row">
              <dt>{{ row.role }}</dt>
              <dd>{{ row.items }}</dd>
            </div>
          </dl>

          <p class="note modules__note">{{ modules.note }}</p>
        </div>
      </section>

      <section id="privacy" class="privacy">
        <div class="container">
          <div class="privacy__grid">
            <div>
              <h2 class="section-title">{{ privacy.heading }}</h2>
              <p class="section-lead">{{ privacy.lead }}</p>
              <p class="note privacy__note">{{ privacy.note }}</p>
            </div>

            <table class="privacy__table">
              <caption class="note">可见范围</caption>
              <thead>
                <tr>
                  <th scope="col">内容</th>
                  <th v-for="column in privacy.columns" :key="column" scope="col">
                    {{ column }}
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="row in privacy.rows" :key="row.label">
                  <th scope="row">{{ row.label }}</th>
                  <td v-for="(value, index) in row.values" :key="index">{{ value }}</td>
                </tr>
              </tbody>
            </table>

            <dl class="privacy__stack">
              <div v-for="row in privacy.rows" :key="row.label" class="privacy__stack-row">
                <dt>{{ row.label }}</dt>
                <dd v-for="(value, index) in row.values" :key="index">
                  <span class="privacy__stack-key">{{ privacy.columns[index] }}</span>
                  <span>{{ value }}</span>
                </dd>
              </div>
            </dl>
          </div>
        </div>
      </section>

      <section class="plain">
        <div class="container plain__inner">
          <div>
            <h2 class="plain__title">{{ plain.heading }}</h2>
            <p class="plain__lead">{{ plain.lead }}</p>
          </div>
          <ul class="plain__list">
            <li v-for="item in plain.items" :key="item">{{ item }}</li>
          </ul>
        </div>
      </section>

      <section id="contact" class="contact">
        <div class="container contact__grid">
          <div>
            <h2 class="section-title">{{ contact.heading }}</h2>
            <p class="section-lead">{{ contact.lead }}</p>
            <p class="note contact__note">{{ contact.note }}</p>
          </div>

          <div class="contact__card">
            <p class="contact__label">邮件联系</p>
            <a v-if="site.contact.email" class="contact__mail" :href="mailtoHref">
              {{ site.contact.email }}
            </a>
            <p class="contact__hint">{{ contact.emailHint }}</p>

            <div v-if="site.contact.wecomQr" class="qr contact__qr">
              <img :src="site.contact.wecomQr" alt="企业微信二维码" width="220" height="220" />
            </div>
          </div>
        </div>
      </section>
    </main>

    <SiteFooter />
  </div>
</template>
