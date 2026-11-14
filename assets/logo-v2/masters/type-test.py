"""「养成星球」字形参数对比：字重 / 字距 / 衬线对照。

全部用 Noto（SIL OFL，可商用可修改），每行都配真实图形，看的是锁图里的平衡，
不是孤立看字。
"""

import io
from pathlib import Path

import cairosvg
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2\masters")
SANS = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
SERIF = r"C:\Windows\Fonts\NotoSerifSC-VF.ttf"
FONT_UI = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
INK = "#1B2A41"
AMBER = "#E8A23A"
CREAM = (247, 243, 232)
INK_RGB = (27, 42, 65)
MUTED = (120, 130, 145)
TEXT = "养成星球"


def ui(size, weight=400):
    f = ImageFont.truetype(FONT_UI, size)
    try:
        f.set_variation_by_axes([weight])
    except Exception:
        pass
    return f


def glyph_run(path, weight, size, tracking):
    font = instantiateVariableFont(TTFont(path), {"wght": weight}, inplace=False)
    cmap = font.getBestCmap()
    gs = font.getGlyphSet()
    hmtx = font["hmtx"]
    upm = font["head"].unitsPerEm
    scale = size / upm
    x = 0.0
    items = []
    for ch in TEXT:
        name = cmap[ord(ch)]
        pen = SVGPathPen(gs)
        gs[name].draw(pen)
        items.append((x, pen.getCommands()))
        x += hmtx[name][0] * scale + tracking
    return items, x - tracking, scale


def lockup(path, weight, size, tracking, w=1000, h=256):
    items, tw, sc = glyph_run(path, weight, size, tracking)
    TXT_X = 410.5
    body = (
        f'<g transform="translate(140.5 28) scale(0.78)">'
        f'<path d="M 128 44 A 84 84 0 1 1 44 128" fill="none" stroke="{INK}" stroke-width="22" stroke-linecap="round"/>'
        f'<circle cx="80" cy="80" r="18" fill="{AMBER}"/></g>'
    )
    for x, d in items:
        body += (
            f'<g transform="translate({TXT_X + x:.2f} 160) scale({sc:.6f} {-sc:.6f})">'
            f'<path d="{d}" fill="{INK}"/></g>'
        )
    svg = f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w} {h}">{body}</svg>'
    png = cairosvg.svg2png(bytestring=svg.encode(), output_width=1000)
    return Image.open(io.BytesIO(png)).convert("RGBA")


ROWS = [
    ("wght 500  字重更轻，与弧的对比拉大", (SANS, 500, 88, 8)),
    ("wght 600", (SANS, 600, 88, 8)),
    ("wght 700  当前", (SANS, 700, 88, 8)),
    ("wght 800  与弧更接近，整体更沉", (SANS, 800, 88, 8)),
    ("wght 700 · 字距 0  更紧", (SANS, 700, 88, 0)),
    ("wght 700 · 字距 16  更松", (SANS, 700, 88, 16)),
    ("Noto Serif 700  衬线对照（与 brief 相反）", (SERIF, 700, 88, 8)),
]

PAD = 28
LOCK_W = 720
LOCK_H = int(256 * LOCK_W / 1000)
ROW_H = LOCK_H + 26
W = LOCK_W + PAD * 2
H = PAD + 64 + ROW_H * len(ROWS) + 24
sheet = Image.new("RGB", (W, H), CREAM)
d = ImageDraw.Draw(sheet)
d.text((PAD, 22), "养成星球 · 字形参数对比", font=ui(24, 700), fill=INK_RGB)
d.text((PAD, 54), "同一款 Noto（OFL，可商用可改），只变字重与字距。每行都配真实图形。", font=ui(12), fill=MUTED)

for i, (label, args) in enumerate(ROWS):
    y = PAD + 64 + i * ROW_H
    im = lockup(*args)
    h = int(im.height * LOCK_W / im.width)
    im = im.resize((LOCK_W, h), Image.LANCZOS)
    flat = Image.new("RGB", im.size, CREAM)
    flat.paste(im, (0, 0), im)
    sheet.paste(flat, (PAD, y))
    d.text((PAD, y + h + 4), label, font=ui(11), fill=MUTED)

out = BASE / "type-test.png"
sheet.save(out)
print(out, sheet.size)
