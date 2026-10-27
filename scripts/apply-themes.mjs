#!/usr/bin/env node
/**
 * Wire the opt-in theme layer into every prototype page.
 *
 * - inserts _shared/theme.js in <head> (before CSS)
 * - inserts _shared/themes.css after shell.css
 * - switches gradient-surface text from --on-brand to --on-accent
 */
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const batch = path.join(repoRoot, 'docs', 'design-previews', '2026-10-25-full-redesign')

let pages = 0
let links = 0
let scripts = 0
let rules = 0

for (const surface of ['teacher', 'parent']) {
  for (const dir of fs.readdirSync(path.join(batch, surface))) {
    const pageDir = path.join(batch, surface, dir)
    const htmlPath = path.join(pageDir, 'index.html')
    if (!fs.existsSync(htmlPath)) continue
    pages += 1

    let html = fs.readFileSync(htmlPath, 'utf8')
    if (!html.includes('_shared/theme.js')) {
      html = html.replace(/<head(\s[^>]*)?>/i, (m) => `${m}\n  <script src="../../_shared/theme.js"></script>`)
      scripts += 1
    }
    if (!html.includes('_shared/themes.css')) {
      html = html.replace(
        /(<link[^>]*href="\.\.\/\.\.\/_shared\/shell\.css"[^>]*>)/i,
        (m) => `${m}\n  <link rel="stylesheet" href="../../_shared/themes.css">`,
      )
      links += 1
    }
    fs.writeFileSync(htmlPath, html, 'utf8')

    const cssPath = path.join(pageDir, 'styles.css')
    let css = fs.readFileSync(cssPath, 'utf8')
    let n = 0
    css = css.replace(/([^{}]*)\{([^{}]*)\}/g, (m, selector, body) => {
      if (
        /background(?:-color)?:\s*var\(--accent-gradient\)/.test(body) &&
        /color:\s*var\(--on-brand\)/.test(body)
      ) {
        body = body.replace(/color:\s*var\(--on-brand\)/g, 'color: var(--on-accent)')
        n += 1
      }
      return `${selector}{${body}}`
    })
    if (n) {
      fs.writeFileSync(cssPath, css, 'utf8')
      rules += n
    }
  }
}

console.log(JSON.stringify({ pages, scripts, links, rules }))
