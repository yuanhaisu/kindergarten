import test from "node:test"
import assert from "node:assert/strict"
import { readFileSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

import { resolveClientScope } from "../src/utils/client-scope.mjs"

// 构建注入（#163）：产物必须自带客户端 AppID 与客户端版本，
// 登录（#175）才能把 app_id 报给后端做客户端注册表校验。
//
// Node 无法求值 import.meta.env，所以配置断言读文件文本；
// 解析逻辑单独放在 client-scope.mjs 里用纯函数测试。

const root = join(dirname(fileURLToPath(import.meta.url)), "..")
const EXPECTED_APP_ID = "wxdf989efff8796c53"

function read(rel) {
  return readFileSync(join(root, rel), "utf8")
}

function envValue(content, key) {
  const match = content.match(new RegExp(`^${key}=(.*)$`, "m"))
  return match ? match[1].trim() : undefined
}

const modes = ["development", "test", "production"]

test("三种构建模式的 .env 都注入同一个客户端 AppID", () => {
  for (const mode of modes) {
    const content = read(`.env.${mode}`)
    assert.equal(envValue(content, "VITE_APP_ID"), EXPECTED_APP_ID, `.env.${mode} 缺少正确的 VITE_APP_ID`)
  }
})

test("VITE_APP_ID 与 manifest 指向同一个小程序", () => {
  const manifest = JSON.parse(read("src/manifest.json"))
  assert.equal(manifest["mp-weixin"].appid, EXPECTED_APP_ID)
})

test("三种构建模式的客户端版本一致且与 package.json 同步", () => {
  const version = JSON.parse(read("package.json")).version
  for (const mode of modes) {
    const content = read(`.env.${mode}`)
    assert.equal(envValue(content, "VITE_CLIENT_VERSION"), version, `.env.${mode} 的 VITE_CLIENT_VERSION 与 package.json 不一致`)
  }
})

test("resolveClientScope 优先使用 Vite env，缺省回落到构建内置值", () => {
  assert.deepEqual(
    resolveClientScope({ VITE_APP_ID: "wx_from_env", VITE_CLIENT_VERSION: "9.8.7" }, {
      fallbackAppId: EXPECTED_APP_ID,
      fallbackVersion: "0.1.0"
    }),
    { appId: "wx_from_env", clientVersion: "9.8.7" }
  )

  assert.deepEqual(
    resolveClientScope({}, { fallbackAppId: EXPECTED_APP_ID, fallbackVersion: "0.1.0" }),
    { appId: EXPECTED_APP_ID, clientVersion: "0.1.0" }
  )
})
