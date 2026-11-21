// 家长端不得出现「其他园所目录」（#176）。
//
// 扫码进入的目标园只能通过两种来源解析：
//   1. GET /common/user/kindergartens —— 当前账号自己的成员园（#169）
//   2. GET /public/kindergartens/:code —— 按不可枚举的公开码取单园品牌
// 任何全量园所列表/搜索接口都会把家长导向别的园，属于越界能力。
// 这里扫描源码字符串，新增任何 /kindergartens 端点都必须先通过本断言。
import test from "node:test"
import assert from "node:assert/strict"
import { readdirSync, readFileSync } from "node:fs"
import { dirname, join, resolve } from "node:path"
import { fileURLToPath } from "node:url"

const SRC_DIR = resolve(dirname(fileURLToPath(import.meta.url)), "..", "src")
const SOURCE_EXTENSIONS = new Set([".js", ".mjs", ".vue"])

// 允许的形态：自己的成员园、按码取单园、前端路由页（不是接口）。
const ALLOWED_PREFIXES = ["/common/user/kindergartens", "/public/kindergartens/"]
const ALLOWED_ROUTES = new Set(["/pages/kindergartens/index"])

function listSourceFiles(dir) {
  const files = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const path = join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...listSourceFiles(path))
      continue
    }
    const dot = entry.name.lastIndexOf(".")
    if (dot > 0 && SOURCE_EXTENSIONS.has(entry.name.slice(dot))) files.push(path)
  }
  return files
}

// 只取以 / 开头、含 kindergartens 的字符串片段；模板串里的 ${...} 截断到 $ 为止。
const KINDERGARTEN_URL_RE = /\/[\w\-./${}%]*kindergartens[\w\-./${}%]*/g

test("家长端源码不引用任何全量园所目录接口", () => {
  const offenders = []
  for (const file of listSourceFiles(SRC_DIR)) {
    const source = readFileSync(file, "utf8")
    for (const match of source.matchAll(KINDERGARTEN_URL_RE)) {
      const url = match[0]
      if (ALLOWED_ROUTES.has(url)) continue
      if (ALLOWED_PREFIXES.some((prefix) => url.startsWith(prefix))) continue
      offenders.push(`${file}: ${url}`)
    }
  }
  assert.deepEqual(offenders, [], "只允许自己的成员园与按公开码取单园；不得出现其他园所目录")
})
