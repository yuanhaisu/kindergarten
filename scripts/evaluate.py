#!/usr/bin/env python3
"""
evaluate.py — 在小程序「逻辑层」(appservice/mainframe) 执行 JS 并打印结果

为什么需要它：
  shot.py / click.py 走的是 __pageframe__【视图层】，只能碰 DOM；
  而 wx.login() / uni.setStorageSync() 等能力住在【逻辑层】上下文里。

  逻辑层下发的调用与真机同路径（IDE 的 appservice 就是小程序 JS 引擎），
  因此可用于「清 token → 触发重登」这类需要真实业务代码跑一遍的验证。
  ⚠️ 与 wechat_navigate 本质不同：后者直接调路由 API、绕过业务代码。

用法：
  python scripts/evaluate.py --title 老师端 "wx.getStorageSync('token')"
  python scripts/evaluate.py --title 老师端 "JSON.stringify(wx.getSystemInfoSync())"
  python scripts/evaluate.py --title 家长端 --expr-file expr.js

  --title 命中「家长/老师」时自动切 CDP 端口（9222/9223），与 shot.py / click.py 一致。
退出码：0 成功 / 1 失败
"""

import argparse
import asyncio
import json
import os
import sys
import urllib.request

# Windows 控制台默认 GBK，直出中文会 UnicodeEncodeError
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8", errors="replace")

WS_URL_FMT = "ws://127.0.0.1:{port}/devtools/page/{target_id}"
CDP_PORT = 9222
TITLE_PORT = {"家长": 9222, "老师": 9223, "parent": 9222, "teacher": 9223}


def focus_ide(title: str = "") -> bool:
    import subprocess
    ps1 = os.path.join(os.path.dirname(os.path.abspath(__file__)), "focus-ide.ps1")
    cmd = ["powershell", "-ExecutionPolicy", "Bypass", "-File", ps1]
    if title:
        cmd += ["-Title", title]
    try:
        return subprocess.run(cmd, capture_output=True, timeout=15).returncode == 0
    except Exception:
        return False


def find_appservice_target(port: int) -> dict | None:
    """appservice/mainframe 是逻辑层；id 随 IDE 重载变化，每次现取。"""
    with urllib.request.urlopen(f"http://127.0.0.1:{port}/json/list", timeout=5) as r:
        targets = json.loads(r.read())
    for t in targets:
        if t.get("type") == "webview" and "appservice/mainframe" in t.get("url", ""):
            return t
    return None


async def evaluate(ws_url: str, expr: str, wait_promise: bool, timeout_s: int) -> dict:
    import websockets

    async with websockets.connect(ws_url, open_timeout=10) as ws:
        await ws.send(json.dumps({
            "id": 1,
            "method": "Runtime.evaluate",
            "params": {
                "expression": expr,
                "returnByValue": True,
                "awaitPromise": wait_promise,
            },
        }))
        raw = await asyncio.wait_for(ws.recv(), timeout=timeout_s)

    resp = json.loads(raw)
    res = resp.get("result", {})
    if "exceptionDetails" in res:
        raise RuntimeError("JS exception: " + json.dumps(res["exceptionDetails"], ensure_ascii=False)[:600])
    return res.get("result", {})


def main() -> int:
    ap = argparse.ArgumentParser(description="Evaluate JS in mini-program appservice context via CDP")
    ap.add_argument("expr", nargs="?", default=None, help="要执行的 JS 表达式")
    ap.add_argument("--expr-file", default=None, help="从文件读取表达式（避免 shell 转义问题）")
    ap.add_argument("--title", default="", help="窗口标题子串（双 IDE 时指定）")
    ap.add_argument("--port", type=int, default=None, help="CDP 端口（默认按 --title 推断）")
    ap.add_argument("--target-id", default=None, help="指定 CDP target（默认取 appservice/mainframe）")
    ap.add_argument("--no-focus", action="store_true", help="跳过窗口聚焦（纯读取可跳过）")
    ap.add_argument("--await-promise", dest="await_promise", action="store_true", help="等待 Promise 结果")
    ap.add_argument("--timeout", type=int, default=20, help="等待结果的秒数（默认 20）")
    args = ap.parse_args()

    expr = args.expr
    if args.expr_file:
        with open(args.expr_file, encoding="utf-8") as f:
            expr = f.read()
    if not expr:
        print("FAIL: no expression given (positional or --expr-file)")
        return 1

    port = args.port
    if port is None:
        port = CDP_PORT
        for kw, p in TITLE_PORT.items():
            if kw in args.title:
                port = p
                break

    if not args.no_focus:
        if not focus_ide(args.title):
            print("WARN: could not confirm IDE window in foreground")

    if args.target_id:
        target_id = args.target_id
    else:
        t = find_appservice_target(port)
        if not t:
            print("FAIL: no appservice/mainframe target on CDP port", port)
            return 1
        target_id = t["id"]

    ws_url = WS_URL_FMT.format(port=port, target_id=target_id)
    try:
        result = asyncio.run(evaluate(ws_url, expr, args.await_promise, args.timeout))
    except Exception as e:
        print("FAIL:", e)
        return 1

    t = result.get("type")
    v = result.get("value")
    if isinstance(v, (dict, list)):
        print(json.dumps(v, ensure_ascii=False, indent=2))
    else:
        print(f"[{t}] {v}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
