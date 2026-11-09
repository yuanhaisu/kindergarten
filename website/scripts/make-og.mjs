// 生成 1200x630 的分享卡片。文字用系统中文栈，避免引入 web font。
// 产物 public/og.png 的生成方式记录在此，便于换文案后重出。
import fs from 'node:fs'
import path from 'node:path'
import { Resvg } from '@resvg/resvg-js'

const OUT = path.resolve(import.meta.dirname, '../public/og.png')

const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
  <rect width="1200" height="630" fill="#F7F3E8"/>
  <g transform="translate(80 76) scale(0.22)">
    <path d="M 128 44 A 84 84 0 1 1 44 128" fill="none" stroke="#1B2A41" stroke-width="22" stroke-linecap="round"/>
    <circle cx="80" cy="80" r="18" fill="#E8A23A"/>
  </g>
  <text x="152" y="112" font-family="'Microsoft YaHei','PingFang SC','Noto Sans SC',sans-serif" font-size="30" font-weight="600" fill="#1B2A41">养成星球</text>
  <text x="152" y="142" font-family="'Microsoft YaHei','PingFang SC','Noto Sans SC',sans-serif" font-size="16" letter-spacing="3" fill="#566275">NURTURE PLANET</text>

  <text x="80" y="320" font-family="'Microsoft YaHei','PingFang SC','Noto Sans SC',sans-serif" font-size="64" font-weight="700" fill="#1B2A41">孩子在园的每一天，</text>
  <text x="80" y="406" font-family="'Microsoft YaHei','PingFang SC','Noto Sans SC',sans-serif" font-size="64" font-weight="700" fill="#1B2A41">都有据可查。</text>
  <rect x="80" y="432" width="336" height="18" rx="6" fill="#E8A23A"/>

  <text x="80" y="520" font-family="'Microsoft YaHei','PingFang SC','Noto Sans SC',sans-serif" font-size="22" fill="#566275">通知 · 成长记录 · 请假 · 接送 · 园务管理</text>
  <line x1="80" y1="566" x2="1120" y2="566" stroke="#DED5C4" stroke-width="1"/>
  <text x="80" y="600" font-family="'Microsoft YaHei','PingFang SC','Noto Sans SC',sans-serif" font-size="16" fill="#566275">幼儿园家园共育与园务管理平台</text>
</svg>`

const resvg = new Resvg(svg, {
  fitTo: { mode: 'width', value: 1200 },
  font: { loadSystemFonts: true, defaultFontFamily: 'Microsoft YaHei' },
})

fs.writeFileSync(OUT, resvg.render().asPng())
console.log(`og.png  ${(fs.statSync(OUT).size / 1024) | 0}KB`)
