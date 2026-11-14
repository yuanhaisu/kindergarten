<script setup lang="ts">
import { site } from '~/content/site'

const canonical = `${site.siteUrl}/terms`
const description = '养成星球软件服务的适用范围、账号与园所责任、服务变更与争议解决。'

useHead({
  title: '服务条款 · 养成星球',
  link: [{ rel: 'canonical', href: canonical }],
  meta: [
    { name: 'description', content: description },
    { property: 'og:url', content: canonical },
    { property: 'og:title', content: '服务条款 · 养成星球' },
    { property: 'og:description', content: description },
    { property: 'og:site_name', content: site.name },
  ],
})

const entity = computed(() => site.legalEntity || '（主体名称待补充）')
</script>

<template>
  <div class="site">
    <SiteNav />

    <main class="legal">
      <div class="container legal__inner">
        <NuxtLink to="/" class="legal__back">← 返回首页</NuxtLink>

        <h1>服务条款</h1>
        <p class="legal__meta">
          服务提供方：{{ entity }}　·　生效日期：待定　·　版本：草案
        </p>

        <h2>一、服务内容</h2>
        <p>
          养成星球向幼儿园提供家园共育与园务管理软件服务，
          包括教师端、家长端微信小程序与园所管理后台。
        </p>
        <p>
          本服务面向幼儿园（园所）及其教师、家长；幼儿不作为账号主体。
        </p>

        <h2>二、账号与使用</h2>
        <ul>
          <li>园所负责为其教师开通账号，并对账号在园所内的使用负责。</li>
          <li>家长账号由家长本人使用，不得转让。</li>
          <li>你应妥善保管账号信息，发现异常应及时通知我们。</li>
        </ul>

        <h2>三、园所责任</h2>
        <ul>
          <li>
            园所应确保其录入与发布的幼儿信息、照片已取得监护人同意，
            特别是身份证号、健康信息与人脸照片等敏感个人信息。
          </li>
          <li>园所应按其内部制度使用平台功能，不得将平台用于与园务无关的用途。</li>
        </ul>

        <h2>四、我们的责任</h2>
        <ul>
          <li>我们按约定提供软件服务，并采取合理措施保障服务的连续性与数据安全。</li>
          <li>
            因不可抗力、网络故障、第三方服务异常等非我方原因导致的服务中断，
            我们不承担由此产生的间接损失。
          </li>
        </ul>

        <h2>五、知识产权</h2>
        <p>
          平台的软件、界面与文档归服务提供方所有。园所与用户上传的内容归其各自权利人所有。
        </p>

        <h2>六、付费与订阅</h2>
        <p>
          具体功能模块、套餐、授权期限与费用，以双方另行签署的订阅合同或订单为准。
          本条款与该合同不一致时，以该合同为准。
        </p>

        <h2>七、服务变更与终止</h2>
        <ul>
          <li>我们可能根据业务需要更新功能；重大变更会提前通知。</li>
          <li>园所或用户违反本条款时，我们可暂停或终止相应服务。</li>
        </ul>

        <h2>八、通知</h2>
        <p>
          我们可通过站内消息、小程序、邮件或本页面公告向你发送通知；
          涉及服务变更、费用或责任的重要通知，会以显著方式提示。
        </p>

        <h2>九、争议解决</h2>
        <p>
          本条款适用中华人民共和国法律。争议由服务提供方所在地有管辖权的人民法院管辖。
        </p>

        <h2>十、联系我们</h2>
        <p>主体：{{ entity }}</p>
        <p>邮箱：{{ site.contact.email || '（待补充）' }}</p>

        <p v-if="site.legalDraft" class="legal__draft">
          本文本为初稿，正式上线前将由法务复核，并按备案主体信息补全。
        </p>
      </div>
    </main>

    <SiteFooter />
  </div>
</template>
