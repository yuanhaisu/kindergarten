"""字面率校正：等视觉面积 + 等墨距。

规则：
1. 每个字等比缩放到同一个墨迹面积（以四字中最大面积为基准，只放大不缩小）；
2. 缩放以基线为准，保持字形不「浮起来」；
3. 排版不用字体自带的前进宽度，改用墨迹边缘 + 固定墨距，保证四个字之间的
   视觉间隔一致（原来的 tracking 是按 em 框算的，墨迹小的字会显得间距更大）。
"""

import io
from pathlib import Path

import cairosvg
from fontTools.pens.boundsPen import BoundsPen
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2\masters")
FONT_PATH = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
FONT_UI = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
TEXT = "养成星球"
INK = "#1B2A41"
AMBER = "#E8A23A"
CREAM = (247, 243, 232)
INK_RGB = (27, 42, 65)
MUTED = (120, 130, 145)

font = instantiateVariableFont(TTFont(FONT_PATH), {"wght": 700}, inplace=False)
cmap = font.getBestCmap()
gs = font.getGlyphSet()
upm = font["head"].unitsPerEm


def ui(size, weight=400):
    f = ImageFont.truetype(FONT_UI, size)
    try:
        f.set_variation_by_axes([weight])
    except Exception:
        pass
    return f


# ---- 测量 ----
metrics = []
for ch in TEXT:
    name = cmap[ord(ch)]
    bpen = BoundsPen(gs)
    gs[name].draw(bpen)
    x0, y0, x1, y1 = bpen.bounds
    area = (x1 - x0) * (y1 - y0) / (upm * upm)
    spen = SVGPathPen(gs)
    gs[name].draw(spen)
    metrics.append(dict(ch=ch, x0=x0, y0=y0, x1=x1, y1=y1, area=area, d=spen.getCommands()))

target_area = max(m["area"] for m in metrics)
for m in metrics:
    m["s"] = (target_area / m["area"]) ** 0.5
    m["w"] = (m["x1"] - m["x0"]) * m["s"]

GAP = 150.0  # 字体单位下的墨距（约合 88px 字号时的 13px）
for m in metrics:
    print(f'{m["ch"]}  面积 {m["area"]:.3f}  缩放 {m["s"]:.4f}  墨宽 {m["w"]:.0f}')

total_w = sum(m["w"] for m in metrics) + GAP * (len(metrics) - 1)
print(f"总墨宽 {total_w:.0f} 字体单位")


def optical_svg(size, tracking_equiv_gap=GAP):
    """size 为字号；按等面积缩放后，用 GAP 作为墨距。"""
    k = size / upm
    x = 0.0
    parts = []
    for m in metrics:
        tx = x - m["x0"] * m["s"] * k
        parts.append(
            f'<g transform="translate({tx:.2f} {size}) scale({m["s"] * k:.6f} {-m["s"] * k:.6f})">'
            f'<path d="{m["d"]}" fill="{INK}"/></g>'
        )
        x += m["w"] * k + tracking_equiv_gap * k
    width = x - tracking_equiv_gap * k
    return "".join(parts), width


body, w = optical_svg(88)
svg = f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {w:.2f} 128">{body}</svg>'
(BASE / "wordmark-optical.svg").write_text(svg, encoding="utf-8")


def draw(svg_text, w, h=256, out_w=900):
    png = cairosvg.svg2png(bytestring=svg_text.encode(), output_width=out_w)
    im = Image.open(io.BytesIO(png)).convert("RGBA")
    flat = Image.new("RGB", im.size, CREAM)
    flat.paste(im, (0, 0), im)
    return flat


# ---- 对比图 ----
old = (BASE / "wordmark.svg").read_text(encoding="utf-8")
new = svg

a = draw(old, 0, 200, 832)   # 原字标 416×200 的 2 倍
b = draw(new, 0, 0, 752)     # 校正版 376×128 的 2 倍

old32 = Image.open(io.BytesIO(cairosvg.svg2png(
    bytestring=old.encode(), output_width=151))).convert("RGBA")
new32 = Image.open(io.BytesIO(cairosvg.svg2png(
    bytestring=new.encode(), output_width=137))).convert("RGBA")


def flat(im):
    f = Image.new("RGB", im.size, CREAM)
    f.paste(im, (0, 0), im)
    return f


o32 = flat(old32).resize((old32.width * 3, old32.height * 3), Image.NEAREST)
n32 = flat(new32).resize((new32.width * 3, new32.height * 3), Image.NEAREST)

W = 1000
H = 60 + a.height + 46 + b.height + 50 + o32.height + 40
sheet = Image.new("RGB", (W, H), CREAM)
d = ImageDraw.Draw(sheet)
d.text((30, 22), "字面率校正 · 前 / 后", font=ui(22, 700), fill=INK_RGB)
d.text((30, 54), "上：原来（Noto 原始字面 + em 框字距）　下：等视觉面积 + 等墨距", font=ui(12), fill=MUTED)

y = 84
d.text((30, y), "校正前", font=ui(11), fill=MUTED)
sheet.paste(a, (30, y + 20))
y2 = y + 20 + a.height + 18
d.text((30, y2), "校正后", font=ui(11), fill=MUTED)
sheet.paste(b, (30, y2 + 20))

# 32px 真实像素（3 倍最近邻）
y3 = y2 + 20 + b.height + 14
d.text((30, y3), "32px 真实像素（×3）", font=ui(11), fill=MUTED)
y32 = y3 + 18
sheet.paste(o32, (30, y32))
ny = y32 + (o32.height - n32.height) // 2   # 两图高度不同，按垂直中心对齐
sheet.paste(n32, (30 + o32.width + 40, ny))
d.text((30, y32 + o32.height + 2), "前", font=ui(10), fill=MUTED)
d.text((30 + o32.width + 40, ny + n32.height + 2), "后", font=ui(10), fill=MUTED)

out = BASE / "optical-proof.png"
sheet.save(out)
print(out)
