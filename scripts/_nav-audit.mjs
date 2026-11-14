import fs from 'node:fs'
import path from 'node:path'

function walk(dir) {
  let out = []
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name)
    if (entry.isDirectory()) out = out.concat(walk(p))
    else if (/\.(vue|js)$/.test(entry.name)) out.push(p)
  }
  return out
}

for (const surface of ['teacher', 'parent']) {
  const root = path.join(surface, 'src')
  const files = walk(root)
  const pages = JSON.parse(fs.readFileSync(path.join(root, 'pages.json'), 'utf8')).pages.map((p) => '/' + p.path)
  const edges = []
  for (const f of files) {
    const text = fs.readFileSync(f, 'utf8')
    const rel = f.split(path.sep).join('/').replace(surface + '/src/', '')
    const literals = []
    for (const m of text.matchAll(/uni\.(?:navigateTo|switchTab|redirectTo|reLaunch)\s*\(\s*\{[^}]*?url:\s*["'`]([^"'`]+)["'`]/g)) literals.push(m[1])
    for (const m of text.matchAll(/(?:path|url):\s*["'`](\/pages\/[^"'`?]+)/g)) literals.push(m[1])
    for (const m of text.matchAll(/:\s*["'`](\/pages\/[^"'`?]+)["'`]/g)) literals.push(m[1])
    for (const l of literals) {
      if (pages.includes(l)) edges.push([rel, l])
    }
  }
  const inbound = Object.fromEntries(pages.map((p) => [p, []]))
  for (const [from, to] of edges) if (inbound[to]) inbound[to].push(from)
  console.log(`\n=== ${surface} pages with ZERO inbound nav ===`)
  for (const p of pages) if (inbound[p].length === 0) console.log(p)
  console.log(`=== ${surface} inbound counts ===`)
  Object.entries(inbound)
    .sort((a, b) => a[1].length - b[1].length)
    .forEach(([p, fs]) => console.log(fs.length, p, fs.slice(0, 3).join(',')))
}
