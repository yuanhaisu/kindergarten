#!/usr/bin/env python3
"""
click.py — 通过 CDP 在小程序视图层做真实点击（激活窗口 + 查元素 + 点击，单进程完成）

为什么需要它：
  `wechat_automator` 的 tap 走 daemon 的视图层通道，该通道会间歇性挂死
  （120s 超时、element_info 同样超时），而 evaluate / CDP 截图正常。
  挂死时无法做任何真实交互验证。

  与 `wechat_navigate` 有本质区别：后者直接调 IDE 的路由 API（reLaunch /
  switchTab），完全绕过页面业务代码；本工具派发的是 `Input.dispatchMouseEvent`，
  走完整的 DOM 事件管线，**会触发 Vue 的 @click 处理器**——与真机点击同路径。

用法：
  python scripts/click.py --title 家长端 ".record-card"
  python scripts/click.py --title 老师端 ".grant-btn" --port 9223
  python scripts/click.py --title 家长端 ".record-card" --nth 2     # 第 3 个匹配（0-based）

  --title 命中「家长/老师」时自动切 CDP 端口（9222/9223），避免点错实例。

退出码：0 成功 / 1 失败
"""

import argparse
import asyncio
import ctypes
import json
import os
import sys
import urllib.request

WS_URL_FMT = "ws://127.0.0.1:{port}/devtools/page/{target_id}"
CDP_PORT = 9222
TITLE_PORT = {"家长": 9222, "老师": 9223, "parent": 9222, "teacher": 9223}


def focus_ide(title: str = "") -> bool:
    ps1 = os.path.join(os.path.dirname(os.path.abspath(__file__)), "focus-ide.ps1")
    cmd = ["powershell", "-ExecutionPolicy", "Bypass", "-File", ps1]
    if title:
        cmd += ["-Title", title]
    try:
        import subprocess
        return subprocess.run(cmd, capture_output=True, timeout=15).returncode == 0
    except Exception:
        return False


def find_pageframe_target(port: int) -> dict | None:
    with urllib.request.urlopen(f"http://127.0.0.1:{port}/json/list", timeout=5) as r:
        targets = json.loads(r.read())
    for t in targets:
        if t.get("type") == "webview" and "__pageframe__" in t.get("url", ""):
            return t
    return None


async def click(ws_url: str, selector: str, nth: int) -> dict:
    import websockets

    async with websockets.connect(ws_url, open_timeout=10) as ws:
        # 1) 取元素中心坐标（Runtime.evaluate 走视图层 DOM）
        expr = f"""
        (() => {{
          const els = document.querySelectorAll({json.dumps(selector)});
          if (!els.length) return JSON.stringify({{err: 'no match', n: 0}});
          const i = Math.min({nth}, els.length - 1);
          const r = els[i].getBoundingClientRect();
          return JSON.stringify({{
            x: r.left + r.width / 2,
            y: r.top + r.height / 2,
            w: r.width, h: r.height,
            n: els.length, idx: i,
            tag: els[i].tagName
          }});
        }})()
        """
        await ws.send(json.dumps({
            "id": 1,
            "method": "Runtime.evaluate",
            "params": {"expression": expr, "returnByValue": True},
        }))
        raw = await asyncio.wait_for(ws.recv(), timeout=15)
        resp = json.loads(raw)
        val = resp.get("result", {}).get("result", {}).get("value")
        if not val:
            raise RuntimeError("evaluate failed: " + json.dumps(resp)[:200])
        info = json.loads(val)
        if info.get("err"):
            raise RuntimeError(f"selector {selector!r} matched 0 elements")
        x, y = info["x"], info["y"]

        # 2) 派发真实鼠标事件序列（与真机触摸同路径进入事件管线）
        for typ in ("mouseMoved", "mousePressed", "mouseReleased"):
            await ws.send(json.dumps({
                "id": 2,
                "method": "Input.dispatchMouseEvent",
                "params": {
                    "type": typ,
                    "x": x, "y": y,
                    "button": "left",
                    "clickCount": 1,
                },
            }))
            await asyncio.wait_for(ws.recv(), timeout=15)

        return info


def main() -> int:
    ap = argparse.ArgumentParser(description="CDP real-click on mini-program element")
    ap.add_argument("selector", help="CSS selector")
    ap.add_argument("--title", default="", help="窗口标题子串（双 IDE 时指定）")
    ap.add_argument("--port", type=int, default=None, help="CDP 端口（默认按 --title 推断）")
    ap.add_argument("--target-id", default=None, help="指定 CDP target（默认取 __pageframe__）")
    ap.add_argument("--nth", type=int, default=0, help="第几个匹配（0-based，默认第一个）")
    args = ap.parse_args()

    port = args.port
    if port is None:
        port = CDP_PORT
        for kw, p in TITLE_PORT.items():
            if kw in args.title:
                port = p
                break

    if not focus_ide(args.title):
        print("WARN: could not confirm IDE window in foreground")

    if args.target_id:
        target_id = args.target_id
    else:
        t = find_pageframe_target(port)
        if not t:
            print("FAIL: no __pageframe__ target on CDP port", port)
            return 1
        target_id = t["id"]
        print(f"target: {t['url']}")

    ws_url = WS_URL_FMT.format(port=port, target_id=target_id)
    try:
        info = asyncio.run(click(ws_url, args.selector, args.nth))
    except Exception as e:
        print("FAIL:", e)
        return 1

    print("OK: clicked %s [%d/%d] at (%.0f,%.0f) %dx%d" % (
        info.get("tag"), info.get("idx", 0) + 1, info.get("n"),
        info["x"], info["y"], info.get("w", 0), info.get("h", 0)))
    return 0


if __name__ == "__main__":
    sys.exit(main())
