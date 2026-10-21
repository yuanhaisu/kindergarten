#!/usr/bin/env python3
"""cdp.py — 微信开发者工具 CDP 通用助手（双实例：9222 家长端 / 9223 老师端）

为什么需要它（与既有脚本的关系）：
  - `evaluate.py` / `click.py` 只认单实例的默认端口、且每次现取 target 不做重试，
    页面切换（reLaunch）导致 pageframe target 重建时会短暂取不到 → 误报失败。
    本脚本统一支持 `--port`，并对 target 做短重试。
  - `shot.py` 仍保留用于**截图**（截图需 IDE 窗口前台，无法并行、易挂死）。
    本脚本刻意**不做截图**：结构断言一律走 DOM evaluate（截图还会抓到陈旧合成帧）。
  - Git Bash（MSYS）会把 `/pages/x` 参数改写成 `C:/Program Files/Git/pages/x`，
    本脚本对 route 参数做归一化修复，调用方无需加 MSYS_NO_PATHCONV。

子命令：
  stack   --port P                         打印当前页面栈（逻辑层）
  nav     --port P --route /pages/x        逻辑层 wx.reLaunch 到指定路由，打印结果
  eval    --port P --expr "<js>" [--layer logic|view]
                                           执行 JS 并打印结果（视图层自动挑 __pageframe__/pages/*，
                                           排除 instanceframe 残影；逻辑层挑 appservice/mainframe）
  watch   --port P --route /pages/x --seconds 6
                                           导航并采集 N 秒的 console error/warning 与未捕获异常
                                           （已过滤 WeChatLib/域名校验等 IDE 噪音，输出 JSON）
  click   --port P --selector "<css>" [--nth 0]
                                           视图层**真实点击**（Input.dispatchMouseEvent 走完整事件
                                           管线，会触发 @click——与真机同路径，不同于 wechat_navigate
                                           直接调路由 API 绕过业务代码）

典型用法（并行双端扫描见 .claude/skills/dual-ide-verification）：
  python scripts/cdp.py watch --port 9223 --route /pages/growth/index --seconds 6
  python scripts/cdp.py eval  --port 9222 --expr "JSON.stringify({txt: document.body.innerText.slice(0,80)})"
  python scripts/cdp.py click --port 9223 --selector ".app-button"   # 登录按钮

约定：只读浏览为主；`click` 是真实交互，用于需要触发业务代码的验证（如登录恢复）。
"""
import argparse
import asyncio
import json
import sys
import time
import urllib.request

sys.stdout.reconfigure(encoding="utf-8", errors="replace")


def targets(port):
    with urllib.request.urlopen(f"http://127.0.0.1:{port}/json/list", timeout=5) as r:
        return json.loads(r.read())


def norm_route(route):
    """修 Git Bash(MSYS) 路径改写：/pages/x 常被转成 C:/Program Files/Git/pages/x。"""
    import re
    m = re.match(r"^[A-Za-z]:[/\\]Program Files[/\\]Git[/\\](.*)$", route)
    if m:
        route = "/" + m.group(1)
    if not route.startswith("/"):
        route = "/" + route
    return route


def pick(port, layer, retries=6, host=""):
    """target 会随页面切换重建；短重试避免踩到重建窗口。

    host：同一个 CDP 端口上可能挂着多个项目窗口（同一实例开了两份项目时），
    每个窗口共享一个内部端口（URL 形如 http://127.0.0.1:<host>/appservice/...）。
    传 host 可钉选某个窗口，避免 pageframe 与 appservice 来自不同窗口。
    不传时：优先取“当前有页面栈”的窗口不可靠，按遍历序取第一个。
    """
    last = None
    for _ in range(retries):
        last = _pick_once(port, layer, host)
        if last:
            return last
        time.sleep(0.6)
    return last


def _pick_once(port, layer, host=""):
    for t in targets(port):
        u = t.get("url", "")
        if t.get("type") != "webview":
            continue
        if host and f":{host}/" not in u:
            continue
        if layer == "logic":
            if "appservice/mainframe" in u:
                return t
        else:
            if "__pageframe__" in u and "/pages/" in u and "instanceframe" not in u:
                return t
    return None


async def _eval(ws_url, expr, timeout=15):
    import websockets
    async with websockets.connect(ws_url, open_timeout=8) as ws:
        await ws.send(json.dumps({"id": 1, "method": "Runtime.evaluate",
                                  "params": {"expression": expr, "returnByValue": True}}))
        raw = await asyncio.wait_for(ws.recv(), timeout=timeout)
        r = json.loads(raw)
        if "exceptionDetails" in r.get("result", {}):
            return {"exc": json.dumps(r["result"]["exceptionDetails"], ensure_ascii=False)[:400]}
        res = r.get("result", {}).get("result", {})
        return res.get("value", res.get("description", "<no value>"))


def pick_or_die(port, layer, host=""):
    t = pick(port, layer, host=host)
    if not t:
        print(json.dumps({"ok": False, "err": f"no {layer} target on {port}"}))
        sys.exit(1)
    return t


def cmd_stack(a):
    t = pick_or_die(a.port, "logic", a.host)
    v = asyncio.run(_eval(f"ws://127.0.0.1:{a.port}/devtools/page/{t['id']}",
                          "JSON.stringify(getCurrentPages().map(p=>p.route+'|'+JSON.stringify(p.options||{})))"))
    print(v)


def cmd_nav(a):
    a.route = norm_route(a.route)
    t = pick_or_die(a.port, "logic", a.host)
    expr = f"wx.reLaunch({{url: {json.dumps(a.route)}}}); 'navigating'"
    asyncio.run(_eval(f"ws://127.0.0.1:{a.port}/devtools/page/{t['id']}", expr))
    time.sleep(a.wait)
    v = asyncio.run(_eval(f"ws://127.0.0.1:{a.port}/devtools/page/{t['id']}",
                          "JSON.stringify(getCurrentPages().map(p=>p.route))"))
    print(json.dumps({"ok": True, "route": a.route, "stack": v}, ensure_ascii=False))


def cmd_eval(a):
    layer = a.layer
    t = pick_or_die(a.port, layer, a.host)
    v = asyncio.run(_eval(f"ws://127.0.0.1:{a.port}/devtools/page/{t['id']}", a.expr))
    if isinstance(v, (dict, list)):
        print(json.dumps(v, ensure_ascii=False))
    else:
        print(v)


async def _watch(a, t):
    import websockets
    errors = []
    async with websockets.connect(f"ws://127.0.0.1:{a.port}/devtools/page/{t['id']}", open_timeout=8) as ws:
        await ws.send(json.dumps({"id": 1, "method": "Runtime.enable"}))
        await asyncio.sleep(0.3)
        if a.route:
            await ws.send(json.dumps({"id": 2, "method": "Runtime.evaluate",
                                      "params": {"expression": f"wx.reLaunch({{url: {json.dumps(norm_route(a.route))}}}); 'go'"}}))
        end = time.time() + a.seconds
        while time.time() < end:
            try:
                raw = await asyncio.wait_for(ws.recv(), timeout=max(0.2, end - time.time()))
            except asyncio.TimeoutError:
                break
            m = json.loads(raw)
            meth = m.get("method", "")
            if meth == "Runtime.exceptionThrown":
                d = m["params"].get("exceptionDetails", {})
                errors.append(("exception", str(d.get("text", ""))[:200]))
            elif meth == "Runtime.consoleAPICalled":
                p = m["params"]
                if p.get("type") in ("error", "warning"):
                    txt = " ".join(str(x.get("value", "")) for x in p.get("args", []))[:200]
                    # 过滤 IDE 自身噪音与空消息（采集层系统性噪音）
                    if not txt.strip():
                        continue
                    if any(k in txt for k in ("WeChatLib", "HarmonyOS", "getSystemInfo", "关于上述警告",
                                              "__subPageFrameEndTime__", "autoTestInfo", "文章推荐",
                                              "配置中关闭合法域名", "未校验合法域名", "TLS 版本以及 HTTPS")):
                        continue
                    errors.append((p.get("type"), txt))
    return errors


def cmd_watch(a):
    t = pick_or_die(a.port, "logic", a.host)
    errs = asyncio.run(_watch(a, t))
    print(json.dumps({"ok": True, "route": norm_route(a.route) if a.route else "", "errors": errs}, ensure_ascii=False))


def cmd_click(a):
    """视图层真实点击（Input.dispatchMouseEvent，走完整事件管线，触发 @click）。"""
    t = pick_or_die(a.port, "view", a.host)

    async def _click():
        import websockets
        async with websockets.connect(f"ws://127.0.0.1:{a.port}/devtools/page/{t['id']}", open_timeout=10) as ws:
            expr = f"""
            (() => {{
              const els = document.querySelectorAll({json.dumps(a.selector)});
              if (!els.length) return JSON.stringify({{err: 'no match'}});
              const i = Math.min({a.nth}, els.length - 1);
              const r = els[i].getBoundingClientRect();
              return JSON.stringify({{x: r.left + r.width/2, y: r.top + r.height/2, n: els.length, text: (els[i].innerText||'').slice(0,50)}});
            }})()
            """
            await ws.send(json.dumps({"id": 1, "method": "Runtime.evaluate", "params": {"expression": expr, "returnByValue": True}}))
            raw = await asyncio.wait_for(ws.recv(), timeout=15)
            info = json.loads(json.loads(raw)["result"]["result"]["value"])
            if info.get("err"):
                return info
            for typ in ("mouseMoved", "mousePressed", "mouseReleased"):
                await ws.send(json.dumps({"id": 2, "method": "Input.dispatchMouseEvent",
                                          "params": {"type": typ, "x": info["x"], "y": info["y"], "button": "left", "clickCount": 1}}))
                await asyncio.wait_for(ws.recv(), timeout=15)
            return info
    print(json.dumps(asyncio.run(_click()), ensure_ascii=False))


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("stack"); p.add_argument("--port", type=int, required=True); p.add_argument("--host", default="", help="钉选项目窗口（URL 内部端口，如 39326）"); p.set_defaults(fn=cmd_stack)
    p = sub.add_parser("nav"); p.add_argument("--port", type=int, required=True); p.add_argument("--host", default=""); p.add_argument("--route", required=True); p.add_argument("--wait", type=float, default=3.5); p.set_defaults(fn=cmd_nav)
    p = sub.add_parser("eval"); p.add_argument("--port", type=int, required=True); p.add_argument("--host", default=""); p.add_argument("--expr", required=True); p.add_argument("--layer", choices=["logic", "view"], default="view"); p.set_defaults(fn=cmd_eval)
    p = sub.add_parser("watch"); p.add_argument("--port", type=int, required=True); p.add_argument("--host", default=""); p.add_argument("--route", default=""); p.add_argument("--seconds", type=float, default=6); p.set_defaults(fn=cmd_watch)
    p = sub.add_parser("click"); p.add_argument("--port", type=int, required=True); p.add_argument("--host", default=""); p.add_argument("--selector", required=True); p.add_argument("--nth", type=int, default=0); p.set_defaults(fn=cmd_click)

    a = ap.parse_args()
    a.fn(a)


if __name__ == "__main__":
    main()
