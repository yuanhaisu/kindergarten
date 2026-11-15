import test from 'node:test'
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const SCRIPT = path.join(repoRoot, 'scripts/check-admin-static.mjs')

/**
 * 造一个临时产物目录。所有测试都通过 CLI 退出码与输出断言行为，
 * 不碰脚本内部实现 —— 检查逻辑重构时测试不应该跟着改。
 */
function makeOutput(t, { withIndex = true, assets = { 'entry.js': 'console.log(1)\n', 'style.css': 'body{}\n' } } = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'admin-static-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))

  if (withIndex) {
    fs.writeFileSync(
      path.join(dir, 'index.html'),
      '<!doctype html><html><body><div id="__nuxt"></div></body></html>\n',
    )
  }
  fs.mkdirSync(path.join(dir, '_nuxt'), { recursive: true })
  for (const [name, content] of Object.entries(assets)) {
    const file = path.join(dir, '_nuxt', name)
    fs.mkdirSync(path.dirname(file), { recursive: true })
    fs.writeFileSync(file, content)
  }
  return dir
}

function run(dir) {
  return spawnSync(process.execPath, [SCRIPT, dir], { encoding: 'utf8' })
}

test('valid static output passes with a summary', (t) => {
  const dir = makeOutput(t, {
    assets: {
      'entry.js': 'console.log(1)\n',
      'style.css': 'body{}\n',
      'chunks/vendor.js': 'console.log(2)\n',
    },
  })

  const result = run(dir)

  assert.equal(result.status, 0, result.stderr)
  assert.match(result.stdout, /OK/)
  assert.match(result.stdout, /index\.html/)
  assert.match(result.stdout, /3 js\/css/)
})

test('missing index.html fails', (t) => {
  const dir = makeOutput(t, { withIndex: false })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /missing-entry/)
  assert.match(result.stderr, /index\.html/)
})

test('remote font hosts fail and name the offending file', (t) => {
  const dir = makeOutput(t, {
    assets: {
      'entry.js': "const css = 'https://fonts.googleapis.com/css2?family=Inter'\n",
      'style.css': "@import url('https://fonts.gstatic.com/s/inter.css');\n",
    },
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /remote-font/)
  assert.match(result.stderr, /fonts\.googleapis\.com/)
  assert.match(result.stderr, /entry\.js/)
  assert.match(result.stderr, /fonts\.gstatic\.com/)
  assert.match(result.stderr, /style\.css/)
})

test('http://localhost fails', (t) => {
  const dir = makeOutput(t)
  fs.appendFileSync(path.join(dir, 'index.html'), "<script>window.__API__ = 'http://localhost:8080/api/v1'</script>\n")

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /localhost-url/)
  assert.match(result.stderr, /index\.html/)
})

test('http://127.0.0.1 fails', (t) => {
  const dir = makeOutput(t, { assets: { 'style.css': 'body{background:url(http://127.0.0.1:8080/a.png)}\n' } })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /localhost-url/)
  assert.match(result.stderr, /style\.css/)
})

test('empty _nuxt directory fails', (t) => {
  const dir = makeOutput(t, { assets: {} })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /missing-assets/)
})

test('missing _nuxt directory fails', (t) => {
  const dir = makeOutput(t)
  fs.rmSync(path.join(dir, '_nuxt'), { recursive: true, force: true })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /missing-assets/)
})

test('missing output directory fails without a stack trace', (t) => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'admin-static-'))
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }))
  const missing = path.join(dir, 'does-not-exist')

  const result = run(missing)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /does-not-exist/)
  assert.doesNotMatch(result.stderr, /\n\s+at /)
})

test('vendored axios localhost fallback is allowed', (t) => {
  // axios 浏览器适配器：没有 window.location 时（如 Worker）用它解析相对 URL
  const dir = makeOutput(t, {
    assets: {
      'entry.js': 'const Rn=Se&&window.location.href||"http://localhost",gn={};\n',
    },
  })

  const result = run(dir)

  assert.equal(result.status, 0, result.stderr)
})

test('real localhost URL next to the axios fallback still fails', (t) => {
  const dir = makeOutput(t, {
    assets: {
      'entry.js': 'const Rn=window.location.href||"http://localhost";fetch("http://localhost:8080/api/v1");\n',
    },
  })

  const result = run(dir)

  assert.notEqual(result.status, 0)
  assert.match(result.stderr, /localhost-url/)
  assert.match(result.stderr, /entry\.js:1/)
})
