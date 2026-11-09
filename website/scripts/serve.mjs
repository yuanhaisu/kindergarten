// 本地预览 .output/public。Python 的 http.server 在 Windows 上会把 .js 发成 text/plain，
// 导致 hydration 不执行；这里显式给出正确的 MIME。
import http from 'node:http'
import fs from 'node:fs'
import path from 'node:path'

const ROOT = path.resolve(import.meta.dirname, '../.output/public')
const PORT = Number(process.env.PORT || 4174)

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.webp': 'image/webp',
  '.ico': 'image/x-icon',
  '.webmanifest': 'application/manifest+json',
  '.woff2': 'font/woff2',
}

const server = http.createServer((req, res) => {
  const urlPath = decodeURIComponent((req.url || '/').split('?')[0])
  let file = path.join(ROOT, urlPath === '/' ? 'index.html' : urlPath)

  if (!file.startsWith(ROOT)) {
    res.writeHead(403).end('forbidden')
    return
  }

  if (fs.existsSync(file) && fs.statSync(file).isDirectory()) {
    file = path.join(file, 'index.html')
  }
  if (!fs.existsSync(file)) {
    const fallback = path.join(ROOT, '404.html')
    if (fs.existsSync(fallback)) {
      res.writeHead(404, { 'content-type': MIME['.html'] })
      fs.createReadStream(fallback).pipe(res)
      return
    }
    res.writeHead(404).end('not found')
    return
  }

  res.writeHead(200, {
    'content-type': MIME[path.extname(file)] || 'application/octet-stream',
    'cache-control': 'no-store',
  })
  fs.createReadStream(file).pipe(res)
})

server.listen(PORT, '127.0.0.1', () => {
  console.log(`website preview: http://127.0.0.1:${PORT}/`)
})
