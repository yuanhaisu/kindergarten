#!/usr/bin/env python3
"""
shot.py — 一键截图（激活 IDE 窗口 + CDP 截图，单进程内完成）

为什么需要它：
  微信开发者工具（nwjs 版）的截图依赖合成层，IDE 窗口必须在前台。
  窗口被覆盖时 Page.captureScreenshot 会挂死不返回。
  而「先激活窗口、再调 MCP 截图」分两步走不通 —— 中间的任何终端操作
  都会把焦点抢回去。所以激活与截图必须在同一进程内一次完成。

用法：
  python scripts/shot.py                    # 截当前页面，存 _verify/shot-<ts>.png
  python scripts/shot.py out.png            # 指定输出
  python scripts/shot.py --target-id <ID>   # 指定 CDP target（默认取 __pageframe__）

⚠️ 双 IDE 并存（见 scripts/dual-ide.ps1）**必须**用 --title 指定目标窗口，
   否则 focus-ide.ps1 会聚焦到另一个实例，目标仍被遮挡 → 截图挂死：
  python scripts/shot.py --title 家长端 out.png   # 家长端（CDP 9222）
  python scripts/shot.py --title 老师端 out.png   # 老师端（CDP 9223）

退出码：0 成功 / 1 失败
"""

import argparse
import asyncio
import base64
import ctypes
import json
import os
import sys
import time
import urllib.request

WS_URL_FMT = "ws://127.0.0.1:{port}/devtools/page/{target_id}"
CDP_PORT = 9222
# 双 IDE 端口分配（与 scripts/dual-ide.ps1 一致）。
# --title 命中这些关键词时自动切换 CDP 端口，避免「聚焦了老师端、却从家长端截图」。
TITLE_PORT = {
    "家长": 9222,
    "老师": 9223,
    "parent": 9222,
    "teacher": 9223,
}

user32 = ctypes.windll.user32


def focus_ide(title: str = "") -> bool:
    """把微信开发者工具窗口置前。

    调用已验证可靠的 focus-ide.ps1（Get-Process + MainWindowHandle），
    而非 EnumWindows —— 后者匹配中文标题不可靠，会让激活静默失败，
    于是截图卡死在合成层上。

    双 IDE 并存时必须传 title（如「家长端」/「老师端」）：
    否则脚本取到的可能是另一个实例，目标窗口仍被遮挡 → captureScreenshot 挂死。
    """
    ps1 = os.path.join(os.path.dirname(os.path.abspath(__file__)), "focus-ide.ps1")
    cmd = ["powershell", "-ExecutionPolicy", "Bypass", "-File", ps1]
    if title:
        cmd += ["-Title", title]
    try:
        import subprocess
        r = subprocess.run(cmd, capture_output=True, timeout=15)
        return r.returncode == 0
    except Exception:
        return False


def find_pageframe_target(port: int) -> dict | None:
    """从 CDP 取 __pageframe__ 那个 webview target（id 会随 IDE 重载变化）。"""
    with urllib.request.urlopen(f"http://127.0.0.1:{port}/json/list", timeout=5) as r:
        targets = json.loads(r.read())
    for t in targets:
        if t.get("type") == "webview" and "__pageframe__" in t.get("url", ""):
            return t
    return None


async def capture(ws_url: str, out_path: str) -> dict:
    import websockets

    t0 = time.time()
    async with websockets.connect(ws_url, open_timeout=10) as ws:
        await ws.send(json.dumps({
            "id": 1,
            "method": "Page.captureScreenshot",
            "params": {"format": "png"},
        }))
        raw = await asyncio.wait_for(ws.recv(), timeout=15)

    resp = json.loads(raw)
    data = resp.get("result", {}).get("data")
    if not data:
        raise RuntimeError("captureScreenshot returned no data: " + json.dumps(resp)[:200])

    os.makedirs(os.path.dirname(os.path.abspath(out_path)) or ".", exist_ok=True)
    with open(out_path, "wb") as f:
        f.write(base64.b64decode(data))

    return {
        "path": os.path.abspath(out_path),
        "bytes": len(data),
        "seconds": round(time.time() - t0, 2),
    }


def main() -> int:
    ap = argparse.ArgumentParser(description="Focus WeChat DevTools and capture screenshot")
    ap.add_argument("output", nargs="?", default=None, help="output PNG path")
    ap.add_argument("--target-id", default=None, help="CDP target id (default: auto __pageframe__)")
    ap.add_argument("--port", type=int, default=None, help="CDP port（默认按 --title 推断，否则 9222）")
    ap.add_argument("--title", default="", help="窗口标题子串，双 IDE 时指定目标实例（如「家长端」）")
    args = ap.parse_args()

    ts = time.strftime("%Y%m%d_%H%M%S")
    out = args.output or os.path.join("_verify", f"shot-{ts}.png")

    # 端口解析：显式 --port 优先；否则按 --title 推断；再否则默认 9222。
    # 不推断的话会出现「聚焦了 A 窗口、却截了 B 的画面」——两个实例画面可能都是 52KB 级，肉眼难辨。
    port = args.port
    if port is None:
        port = CDP_PORT
        for kw, p in TITLE_PORT.items():
            if kw in args.title:
                port = p
                break
        if args.title and port == CDP_PORT and not any(k in args.title for k in TITLE_PORT):
            print(f"WARN: --title {args.title!r} 未命中已知关键词，仍用 CDP {port}；"
                  f"若截错实例请显式传 --port")

    # 1) 激活窗口 —— 必须在截图前，且不能被别的命令打断
    if not focus_ide(args.title):
        print("WARN: could not confirm IDE window in foreground; screenshot may hang")

    # 2) 取 target（id 会变，每次现取）
    if args.target_id:
        target_id = args.target_id
    else:
        t = find_pageframe_target(port)
        if not t:
            print("FAIL: no __pageframe__ target on CDP port", port)
            return 1
        target_id = t["id"]

    ws_url = WS_URL_FMT.format(port=port, target_id=target_id)

    # 3) 截图
    try:
        info = asyncio.run(capture(ws_url, out))
    except Exception as e:
        print("FAIL:", e)
        return 1

    print("OK: %s  (%d bytes, %ss)" % (info["path"], info["bytes"], info["seconds"]))
    return 0


if __name__ == "__main__":
    sys.exit(main())
