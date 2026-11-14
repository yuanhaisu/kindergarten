#!/usr/bin/env node
/**
 * 生产构建产物的 project.config.json 强制开启合法域名校验。
 *
 * 为什么不在 manifest 里直接写 true：dev 构建也读 manifest，urlCheck=true 会让
 * 本地 127.0.0.1 直接报「不在合法域名列表」，本地调试全废。
 * 这里只改 dist/build 产物，dev 产物保持可调试。
 *
 * 用法：node scripts/ensure-urlcheck.mjs <dist/mp-weixin 目录>
 */
import { readFileSync, writeFileSync } from "node:fs"
import { resolve } from "node:path"

const dist = process.argv[2]
if (!dist) {
  console.error("usage: node scripts/ensure-urlcheck.mjs <dist/mp-weixin>")
  process.exit(1)
}

const file = resolve(dist, "project.config.json")
let cfg
try {
  cfg = JSON.parse(readFileSync(file, "utf8"))
} catch (err) {
  console.error(`[ensure-urlcheck] cannot read ${file}: ${err.message}`)
  process.exit(1)
}

cfg.setting = cfg.setting || {}
if (cfg.setting.urlCheck === true) {
  console.log(`[ensure-urlcheck] already true: ${file}`)
  process.exit(0)
}
cfg.setting.urlCheck = true
writeFileSync(file, JSON.stringify(cfg, null, 2) + "\n")
console.log(`[ensure-urlcheck] set urlCheck=true: ${file}`)
