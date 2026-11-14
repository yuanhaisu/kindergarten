"""A1 中英双语锁图：中文在上、英文在下，左对齐起笔。

沿用 A1 横排的间距常数（symbol tx=140.5 ty=28 s=0.78），
字形仍是 Noto Sans SC 抽路径，不含 <text>。
"""

from pathlib import Path

from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2")
MASTERS = BASE / "masters"
FONT_PATH = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"

INK = "#1B2A41"
AMBER = "#E8A23A"
CREAM = "#F7F3E8"

CN = "养成星球"
EN = "Nurture Planet"

font = instantiateVariableFont(TTFont(FONT_PATH), {"wght": 700}, inplace=False)
cmap = font.getBestCmap()
glyphs = font.getGlyphSet()
hmtx = font["hmtx"]
upm = font["head"].unitsPerEm


from fontTools.pens.boundsPen import BoundsPen


def run(text, size, tracking):
    scale = size / upm
    x = 0.0
    items = []
    for ch in text:
        if ch == " ":
            x += hmtx[cmap[ord(" ")]][0] * scale + tracking
            continue
        name = cmap[ord(ch)]
        pen = SVGPathPen(glyphs)
        glyphs[name].draw(pen)
        items.append((x, pen.getCommands()))
        x += hmtx[name][0] * scale + tracking
    return items, x - tracking, scale


# 中文字面率校正：等视觉面积 + 等墨距（与 build_kit.py 同一套规则）
def _cn_metrics(ch):
    name = cmap[ord(ch)]
    bpen = BoundsPen(glyphs)
    glyphs[name].draw(bpen)
    x0, y0, x1, y1 = bpen.bounds
    pen = SVGPathPen(glyphs)
    glyphs[name].draw(pen)
    return dict(x0=x0, bw=x1 - x0, area=(x1 - x0) * (y1 - y0), d=pen.getCommands())


_M = {ch: _cn_metrics(ch) for ch in CN}
_T = max(m["area"] for m in _M.values())
for _m in _M.values():
    _m["s"] = (_T / _m["area"]) ** 0.5
    _m["w"] = _m["bw"] * _m["s"]


def run_cn(size, ink_gap):
    k = size / upm
    x = 0.0
    items = []
    for ch in CN:
        m = _M[ch]
        items.append((x - m["x0"] * m["s"] * k, m["d"], m["s"] * k))
        x += m["w"] * k + ink_gap * k
    return items, x - ink_gap * k


def group_scaled(items, x0, baseline, fill):
    return "\n  ".join(
        f'<g transform="translate({x0 + dx:.2f} {baseline}) scale({s:.6f} {-s:.6f})">'
        f'<path d="{d}" fill="{fill}"/></g>'
        for dx, d, s in items
    )


def group(items, scale, x0, baseline, fill):
    return "\n  ".join(
        f'<g transform="translate({x0 + x:.2f} {baseline}) scale({scale:.6f} {-scale:.6f})">'
        f'<path d="{d}" fill="{fill}"/></g>'
        for x, d in items
    )


cn_items, cn_w = run_cn(88, 150)
en_items, en_w, en_scale = run(EN, 32, 1.2)

# 与既有 A1-horizontal-color.svg 完全一致的间距常数（文字起点 410.5，画布 960）
SYM_X, SYM_Y, SYM_S = 140.5, 28.0, 0.78
TXT_X = 410.5
CN_BASE, EN_BASE = 132.0, 178.0
W = 960.0
H = 256

SYMBOL = f'''<path d="M 128 44 A 84 84 0 1 1 44 128" fill="none" stroke="{{ink}}" stroke-width="22" stroke-linecap="round"/>
  <circle cx="80" cy="80" r="18" fill="{AMBER}"/>'''


def build(name, ink, title):
    body = (
        f'<g transform="translate({SYM_X} {SYM_Y}) scale({SYM_S})">'
        + SYMBOL.format(ink=ink)
        + "</g>\n  "
        + group_scaled(cn_items, TXT_X, CN_BASE, ink)
        + "\n  "
        + group(en_items, en_scale, TXT_X + 2, EN_BASE, ink)
    )
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {W:.2f} {H}" width="{W:.2f}" height="{H}" role="img" aria-labelledby="t">
  <title id="t">{title}</title>
  {body}
</svg>
'''
    (MASTERS / name).write_text(svg, encoding="utf-8")
    print(MASTERS / name, f"{W:.0f}x{H}")


build("A1-horizontal-bilingual.svg", INK, "养成星球 Nurture Planet")
build("A1-horizontal-bilingual-reversed.svg", CREAM, "养成星球 Nurture Planet reversed")
