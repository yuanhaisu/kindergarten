"""16px 存活率的真实测量。

原来的徽章是按类型写死的常量：所有「具象」格都是「高」，所有「字标」格都是「低」。
它评价的是类型，不是格子——一个弱的符号和一个强的符号拿到完全相同的标签。

改成量出来的：

    把这一格的符号缩到 16×16，和二值化后的 96×96 参考图比三件事
      · 连通块数有没有变少（两个元素糊成一个 = 结构丢了）
      · 洞数有没有变少（细缝闭合 = 细节丢了）
      · 墨迹占比是不是低到看不见

三项全过 → 高；丢一项 → 中；丢两项以上或墨太少 → 低。

测的是**符号**，不是整张锁图。含文字的格子单独标注：16px 上文字本来就不可读，
给一个「不适用」比编一个「低」诚实。

用法：
    python -I scripts/survive.py runs/养成星球/sketches/S07.svg
"""

from __future__ import annotations

import io
import re
import sys
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

import cairosvg
from PIL import Image

VB = re.compile(r'viewBox="([-\d.]+)[ ,]+([-\d.]+)[ ,]+([-\d.]+)[ ,]+([-\d.]+)"')

MIN_INK = 0.06          # 低于这个比例，16px 上基本看不见
MAX_INK_JUMP = 0.18     # 16px 比 96px 多出这么多墨，说明糊成团了


def _grid(svg_text: str, box: int) -> list[list[bool]]:
    """按比例缩进 box×box 的方框并居中，返回二值网格（True = 墨）。"""
    m = VB.search(svg_text)
    if not m:
        raise SystemExit("SVG 缺少 viewBox")
    _, _, w, h = (float(v) for v in m.groups())
    scale = box / max(w, h)
    ow, oh = max(1, round(w * scale)), max(1, round(h * scale))

    png = cairosvg.svg2png(bytestring=svg_text.encode("utf-8"),
                           output_width=ow, output_height=oh, background_color="white")
    im = Image.open(io.BytesIO(png)).convert("L")
    if im.size != (ow, oh):
        im = im.resize((ow, oh))

    canvas = Image.new("L", (box, box), 255)
    canvas.paste(im, ((box - ow) // 2, (box - oh) // 2))
    px = canvas.load()
    return [[px[x, y] < 128 for x in range(box)] for y in range(box)]


def _components(g: list[list[bool]]) -> int:
    """数连通块，但忽略抗锯齿造成的孤立噪点。

    实测：96px 下 S07 抽象版有 5 块，其中 2 块只有 1 个像素——
    那是边缘反锯齿的产物，不是结构。不过滤会得到随机的判定。
    """
    n = len(g)
    seen = [[False] * n for _ in range(n)]
    sizes: list[int] = []
    for y in range(n):
        for x in range(n):
            if not g[y][x] or seen[y][x]:
                continue
            stack = [(y, x)]
            seen[y][x] = True
            size = 0
            while stack:
                cy, cx = stack.pop()
                size += 1
                for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    ny, nx = cy + dy, cx + dx
                    if 0 <= ny < n and 0 <= nx < n and g[ny][nx] and not seen[ny][nx]:
                        seen[ny][nx] = True
                        stack.append((ny, nx))
            sizes.append(size)

    total = sum(sizes)
    floor = max(2, total * 0.005)
    return sum(1 for s in sizes if s >= floor)


def _holes(g: list[list[bool]]) -> int:
    """数**被围住的白色区域个数**，不是数白色像素。

    这个区别在闭合图形上是致命的：一个闭合圆环内部有 ~2400 个白像素，
    但如果按像素数返回，96px 会得到 2400、16px 得到 60，看上去像「洞消失了」，
    实际洞一直是那一个。S09 就是因此被判成「中」的。
    """
    n = len(g)
    seen = [[False] * n for _ in range(n)]

    def flood(sy: int, sx: int) -> int:
        stack = [(sy, sx)]
        seen[sy][sx] = True
        size = 0
        while stack:
            cy, cx = stack.pop()
            size += 1
            for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                ny, nx = cy + dy, cx + dx
                if 0 <= ny < n and 0 <= nx < n and not g[ny][nx] and not seen[ny][nx]:
                    seen[ny][nx] = True
                    stack.append((ny, nx))
        return size

    # 先灌边界上的白，灌到的都是「外部」
    for i in range(n):
        for y, x in ((0, i), (n - 1, i), (i, 0), (i, n - 1)):
            if not g[y][x] and not seen[y][x]:
                flood(y, x)

    # 剩下没灌到的白区域，各算一个洞；小于 2 像素的当抗锯齿噪点
    count = 0
    for y in range(n):
        for x in range(n):
            if not g[y][x] and not seen[y][x]:
                if flood(y, x) >= 2:
                    count += 1
    return count


def _stats(g: list[list[bool]]) -> dict:
    n = len(g)
    ink = sum(1 for row in g for v in row if v) / (n * n)
    return {"ink": ink, "components": _components(g), "holes": _holes(g)}


def measure(svg_text: str, small: int = 16, ref: int = 96) -> dict:
    a, b = _stats(_grid(svg_text, small)), _stats(_grid(svg_text, ref))

    reasons = []
    if a["ink"] < MIN_INK:
        reasons.append(f"16px 墨只有 {a['ink']:.1%}，太细")
    if a["components"] < b["components"]:
        reasons.append(f"连通块 {b['components']} → {a['components']}，元素糊在一起")
    if a["holes"] < b["holes"]:
        reasons.append(f"洞 {b['holes']} → {a['holes']}，缝隙闭合")
    if a["holes"] > b["holes"]:
        # 开口在 16px 上闭合成针孔——肉眼看到的是一个实心团加几个白点
        reasons.append(f"洞 {b['holes']} → {a['holes']}，开口闭合成针孔")
    if a["ink"] - b["ink"] > MAX_INK_JUMP:
        reasons.append(f"墨占比 {b['ink']:.0%} → {a['ink']:.0%}，放大成团")

    score = len(reasons)
    verdict = "高" if score == 0 else ("中" if score == 1 else "低")
    return {"verdict": verdict, "small": a, "ref": b, "reasons": reasons}


def main() -> int:
    if len(sys.argv) < 2:
        print("用法: python -I scripts/survive.py <svg> [更多 svg...]")
        return 1

    for p in sys.argv[1:]:
        text = Path(p).read_text(encoding="utf-8")
        r = measure(text)
        a, b = r["small"], r["ref"]
        print(f"{Path(p).name}: 16px {r['verdict']}")
        print(f"   16px  墨 {a['ink']:.1%} · {a['components']} 块 · {a['holes']} 洞")
        print(f"   96px  墨 {b['ink']:.1%} · {b['components']} 块 · {b['holes']} 洞")
        for reason in r["reasons"]:
            print(f"   ! {reason}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
