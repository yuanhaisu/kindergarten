#!/usr/bin/env node
/**
 * sync-ui.js — 把 shared-ui/ 的组件分发到两个小程序的 components/
 *
 * 与 design token 流水线同思路：单一来源（shared-ui/）+ 构建期分发。
 * 两端 import 路径保持 `@/components/X.vue`，不改 uni-app 构建配置。
 *
 * 用法：node scripts/sync-ui.js            （predev/prebuild 自动调用）
 */

const fs = require("fs")
const path = require("path")

const ROOT = path.resolve(__dirname, "..")
const SRC = path.join(ROOT, "shared-ui")
const TARGETS = [
  path.join(ROOT, "teacher", "src", "components"),
  path.join(ROOT, "parent", "src", "components"),
]

function main() {
  if (!fs.existsSync(SRC)) {
    console.error("FAIL: shared-ui/ not found")
    process.exit(1)
  }

  const files = fs.readdirSync(SRC).filter((f) => f.endsWith(".vue"))
  if (files.length === 0) {
    console.error("FAIL: shared-ui/ has no .vue components")
    process.exit(1)
  }

  for (const dir of TARGETS) {
    fs.mkdirSync(dir, { recursive: true })
    for (const f of files) {
      fs.copyFileSync(path.join(SRC, f), path.join(dir, f))
    }
    console.log(`synced ${files.length} components -> ${path.relative(ROOT, dir)}`)
  }
}

main()
