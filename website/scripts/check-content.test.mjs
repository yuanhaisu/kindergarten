import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const websiteRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const SCRIPT = path.join(websiteRoot, 'scripts/check-content.mjs')

const HOST = 'https://www.nurtureplanet.cn'
const PRODUCT = '养成星球'
const ICP = '湘ICP备2026054245号-1'
const ENTITY = '湘潭市岳塘区稚序软件开发工作室'
const EMAIL = 'contact@nurtureplanet.cn'
const CLASS_SHARE =
  '「班内共享」默认开启，家长可随时撤回；撤回后，其他家长看不到包含该幼儿的照片，该幼儿自己的家长仍可见。'

function html({ title, canonical, body }) {
  return [
    '<!DOCTYPE html>',
    '<html lang="zh-CN">',
    '<head>',
    '<meta charset="utf-8">',
    `<title>${title}</title>`,
    canonical ? `<link rel="canonical" href="${canonical}">` : '',
    '</head>',
    '<body>',
    body,
    '</body>',
    '</html>',
  ]
    .filter(Boolean)
    .join('\n')
}

function footer(facts = [ENTITY, ICP, EMAIL]) {
  return `<footer><p>${facts.join(' · ')}</p></footer>`
}

function baselineFiles() {
  return {
    'index.html': html({
      title: `${PRODUCT} · 幼儿园家园共育与园务管理平台`,
      canonical: `${HOST}/`,
      body: [
        `<header><nav>${PRODUCT}</nav></header>`,
        '<main><h1>孩子在园的每一天，都有据可查</h1></main>',
        `<p>${CLASS_SHARE}</p>`,
        footer(),
      ].join('\n'),
    }),
    'privacy/index.html': html({
      title: `隐私政策 · ${PRODUCT}`,
      canonical: `${HOST}/privacy`,
      body: [
        '<main><h1>隐私政策</h1>',
        '<p class="legal__meta">适用主体：' + ENTITY + '　·　生效日期：待定　·　版本：草案</p>',
        '<p>「班内共享」默认开启，家长可随时撤回；撤回后，其他家长看不到包含该幼儿的照片。</p>',
        '<h2>十三、联系我们</h2><p>邮箱：' + EMAIL + '</p></main>',
        footer(),
      ].join('\n'),
    }),
    'terms/index.html': html({
      title: `服务条款 · ${PRODUCT}`,
      canonical: `${HOST}/terms`,
      body: [
        '<main><h1>服务条款</h1>',
        '<p class="legal__meta">服务提供方：' + ENTITY + '　·　生效日期：待定　·　版本：草案</p>',
        '<p>邮箱：' + EMAIL + '</p></main>',
        footer(),
      ].join('\n'),
    }),
    '404.html': html({
      title: `页面不存在 · ${PRODUCT}`,
      body: `<header><nav>${PRODUCT}</nav></header><main><h1>这个页面不在了。</h1></main>${footer()}`,
    }),
    'robots.txt': `User-agent: *\nAllow: /\n\nSitemap: ${HOST}/sitemap.xml\n`,
    'sitemap.xml': [
      '<?xml version="1.0" encoding="UTF-8"?>',
      '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">',
      `<url><loc>${HOST}/</loc></url>`,
      `<url><loc>${HOST}/privacy</loc></url>`,
      `<url><loc>${HOST}/terms</loc></url>`,
      '</urlset>',
    ].join('\n'),
  }
}

function writeOutput(t, files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'website-content-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  for (const [relative, content] of Object.entries(files)) {
    const file = path.join(dir, relative)
    fs.mkdirSync(path.dirname(file), { recursive: true })
    fs.writeFileSync(file, content)
  }
  return dir
}

function fixture(t, mutate = () => {}) {
  const files = baselineFiles()
  mutate(files)
  return writeOutput(t, files)
}

function run(target, extraArgs = []) {
  return spawnSync(process.execPath, [SCRIPT, target, ...extraArgs], { encoding: 'utf8' })
}

test('valid built output passes', (t) => {
  const result = run(fixture(t))

  assert.equal(result.status, 0, result.stderr)
  assert.match(result.stdout, /OK/)
})

test('missing output directory fails instead of throwing', (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'website-content-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))

  const result = run(path.join(dir, 'missing'))

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /missing/)
  assert.doesNotMatch(result.stderr, /\n\s+at /)
})

test('missing ICP number on the home page names the page and the fact', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replace(ICP, '')
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /首页/)
  assert.match(result.stderr, /ICP/)
  assert.match(result.stderr, new RegExp(ICP))
})

test('missing legal entity on the privacy page names the page and the fact', (t) => {
  const dir = fixture(t, (files) => {
    files['privacy/index.html'] = files['privacy/index.html'].replaceAll(ENTITY, '')
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /隐私政策/)
  assert.match(result.stderr, /主体/)
})

test('missing contact email on the terms page names the page and the fact', (t) => {
  const dir = fixture(t, (files) => {
    files['terms/index.html'] = files['terms/index.html'].replaceAll(EMAIL, '')
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /服务条款/)
  assert.match(result.stderr, /邮箱/)
})

test('missing product name on the home page fails', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replaceAll(PRODUCT, '')
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /首页/)
  assert.match(result.stderr, /产品名/)
})

test('missing privacy page names the page', (t) => {
  const dir = fixture(t, (files) => {
    delete files['privacy/index.html']
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /隐私政策/)
  assert.match(result.stderr, /privacy\/index\.html/)
})

test('missing robots.txt names the asset', (t) => {
  const dir = fixture(t, (files) => {
    delete files['robots.txt']
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /robots\.txt/)
})

test('missing sitemap.xml names the asset', (t) => {
  const dir = fixture(t, (files) => {
    delete files['sitemap.xml']
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /sitemap\.xml/)
})

test('canonical pointing at the staging host fails', (t) => {
  const dir = fixture(t, (files) => {
    files['privacy/index.html'] = files['privacy/index.html'].replace(
      `${HOST}/privacy`,
      'https://test-www.nurtureplanet.cn/privacy',
    )
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /隐私政策/)
  assert.match(result.stderr, /canonical/)
  assert.match(result.stderr, new RegExp(HOST.replaceAll('.', '\\.')))
})

test('missing canonical on the home page fails', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replace(
      `<link rel="canonical" href="${HOST}/">`,
      '',
    )
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /首页/)
  assert.match(result.stderr, /canonical/)
})

test('sitemap using a non-production host fails', (t) => {
  const dir = fixture(t, (files) => {
    files['sitemap.xml'] = files['sitemap.xml'].replaceAll(HOST, 'https://test-www.nurtureplanet.cn')
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /sitemap\.xml/)
  assert.match(result.stderr, /生产域名|www\.nurtureplanet\.cn/)
})

test('404 page without the product brand fails', (t) => {
  const dir = fixture(t, (files) => {
    files['404.html'] = files['404.html'].replaceAll(PRODUCT, '')
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /404/)
  assert.match(result.stderr, /产品名/)
})

test('adding 立即使用 fails', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replace(
      '</main>',
      '<a class="btn btn--primary" href="#">立即使用</a></main>',
    )
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /首页/)
  assert.match(result.stderr, /立即使用/)
})

test('an actual small-program QR image fails', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replace(
      '</main>',
      '<img src="/contact/miniprogram-qr.png" alt="小程序码" width="220" height="220"></main>',
    )
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /首页/)
  assert.match(result.stderr, /小程序码素材/)
})

test('a small-program QR call-to-action fails', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replace(
      '</main>',
      '<a class="btn" href="#">扫码体验小程序</a></main>',
    )
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /首页/)
  assert.match(result.stderr, /小程序/)
})

test('the privacy third-party disclosure mentioning 小程序码 passes', (t) => {
  const dir = fixture(t, (files) => {
    files['privacy/index.html'] = files['privacy/index.html'].replace(
      '</main>',
      '<p>微信：用于登录、手机号授权与小程序码生成；会向其传输登录 code、手机号授权 code 与小程序码参数。</p></main>',
    )
  })

  const result = run(dir)

  assert.equal(result.status, 0, result.stderr)
})

test('an enterprise WeChat QR image passes', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replace(
      '</main>',
      '<img src="/contact/wecom-qr.png" alt="企业微信二维码" width="220" height="220"></main>',
    )
  })

  const result = run(dir)

  assert.equal(result.status, 0, result.stderr)
})

test('staging mode allows draft legal pages', (t) => {
  const result = run(fixture(t))

  assert.equal(result.status, 0, result.stderr)
})

test('--production fails while the privacy page is still a draft', (t) => {
  const result = run(fixture(t), ['--production'])

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /隐私政策/)
  assert.match(result.stderr, /版本：草案/)
})

test('--production fails while the terms page still has a pending effective date', (t) => {
  const dir = fixture(t, (files) => {
    files['privacy/index.html'] = files['privacy/index.html'].replace(
      '生效日期：待定　·　版本：草案',
      '生效日期：2026-12-01　·　版本：1.0',
    )
    files['terms/index.html'] = files['terms/index.html'].replace(
      '生效日期：待定　·　版本：草案',
      '生效日期：2026-12-01　·　版本：1.0',
    )
    files['terms/index.html'] = files['terms/index.html'].replace(
      '生效日期：2026-12-01　·　版本：1.0',
      '生效日期：待定　·　版本：1.0',
    )
  })

  const result = run(dir, ['--production'])

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /服务条款/)
  assert.match(result.stderr, /生效日期：待定/)
})

test('--production passes once the legal pages carry an effective date', (t) => {
  const dir = fixture(t, (files) => {
    for (const page of ['privacy/index.html', 'terms/index.html']) {
      files[page] = files[page].replace(
        '生效日期：待定　·　版本：草案',
        '生效日期：2026-12-01　·　版本：1.0',
      )
    }
  })

  const result = run(dir, ['--production'])

  assert.equal(result.status, 0, result.stderr)
})

test('dropping the class-share statement from the home page fails', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replace(`<p>${CLASS_SHARE}</p>`, '')
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /首页/)
  assert.match(result.stderr, /班内共享/)
})

test('the class-share wording may say 可随时关闭', (t) => {
  const dir = fixture(t, (files) => {
    for (const page of ['index.html', 'privacy/index.html']) {
      files[page] = files[page].replaceAll('可随时撤回', '可随时关闭')
    }
  })

  const result = run(dir)

  assert.equal(result.status, 0, result.stderr)
})

test('future no-switch class-album copy fails', (t) => {
  const dir = fixture(t, (files) => {
    files['index.html'] = files['index.html'].replace(
      '</main>',
      '<p>我们不提供家长单独关闭班级相册展示的开关。</p></main>',
    )
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /首页/)
  assert.match(result.stderr, /不提供/)
})

test('a bare 不提供开关 drift fails', (t) => {
  const dir = fixture(t, (files) => {
    files['privacy/index.html'] = files['privacy/index.html'].replace(
      '家长可随时撤回；撤回后，其他家长看不到包含该幼儿的照片。',
      '不提供开关。',
    )
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /隐私政策/)
  assert.match(result.stderr, /不提供/)
})
