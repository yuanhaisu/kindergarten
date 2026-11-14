// 从 docs/design-previews 的原型长图裁剪出官网使用的手机端首屏片段。
// 原型图是 2x 分辨率（780px 宽 = 390pt），官网按 390pt 宽显示，因此保留原图不失真。
import fs from 'node:fs'
import path from 'node:path'
import { PNG } from 'pngjs'

const ROOT = path.resolve(import.meta.dirname, '../..')
const PREVIEW = path.join(ROOT, 'docs/design-previews/2026-10-25-full-redesign')
const AUDIT = path.join(ROOT, 'docs/design-previews/audit-2026-10-25')
const OUT = path.resolve(import.meta.dirname, '../public/screens')

const JOBS = [
  { src: path.join(PREVIEW, 'teacher/01-workbench/preview.png'), out: 'teacher-workbench.png', crop: 1400 },
  { src: path.join(PREVIEW, 'parent/01-home/preview.png'), out: 'parent-home.png', crop: 1400 },
  { src: path.join(PREVIEW, 'teacher/20-doctor-checkin/preview.png'), out: 'teacher-checkin.png', crop: 1400 },
  { src: path.join(PREVIEW, 'teacher/23-notice-index/preview.png'), out: 'teacher-notice.png', crop: 1400 },
  { src: path.join(PREVIEW, 'parent/09-growth-index/preview.png'), out: 'parent-growth.png', crop: 1400 },
  { src: path.join(PREVIEW, 'teacher/14-leave-index/preview.png'), out: 'teacher-leave.png', crop: 1400 },
  { src: path.join(PREVIEW, 'parent/13-pickup-index/preview.png'), out: 'parent-pickup.png', crop: 1400 },
  // 后台截图本身就是 1440x900，原样使用。
  { src: path.join(AUDIT, 'admin-ep-dashboard.png'), out: 'admin-dashboard.png', crop: null },
]

fs.mkdirSync(OUT, { recursive: true })

for (const job of JOBS) {
  if (!fs.existsSync(job.src)) {
    console.error(`missing source: ${job.src}`)
    process.exitCode = 1
    continue
  }
  const png = PNG.sync.read(fs.readFileSync(job.src))
  const height = job.crop ? Math.min(job.crop, png.height) : png.height
  const out = new PNG({ width: png.width, height })
  PNG.bitblt(png, out, 0, 0, png.width, height, 0, 0)
  fs.writeFileSync(path.join(OUT, job.out), PNG.sync.write(out))
  console.log(`${job.out}  ${png.width}x${height}  ${(fs.statSync(path.join(OUT, job.out)).size / 1024) | 0}KB`)
}
