"""侧栏实测：徽章在 #1E293B 上的 32 / 36 / 40px 真实像素（放大 6 倍）。"""

import io
from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2")
SIDEBAR = (30, 41, 59)      # #1E293B
HOVER = (51, 65, 85)
FONT = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
SVG = BASE / "masters" / "A1-badge-cream.svg"


def ft(s, w=400):
    f = ImageFont.truetype(FONT, s)
    try:
        f.set_variation_by_axes([w])
    except Exception:
        pass
    return f


def render(size):
    png = cairosvg.svg2png(url=str(SVG), output_width=size, output_height=size)
    return Image.open(io.BytesIO(png)).convert("RGBA")


SIZES = [32, 36, 40]
SCALE = 5
COL = 260
W = 40 + COL * len(SIZES)
H = 60 + 60 + max(SIZES) * SCALE + 50
sheet = Image.new("RGB", (W, H), SIDEBAR)
d = ImageDraw.Draw(sheet)

for i, s in enumerate(SIZES):
    x = 40 + i * COL
    d.text((x, 26), f"{s}px", font=ft(12), fill=(226, 232, 240))
    sheet.paste(render(s), (x, 60), render(s))
    d.text((x + 50, 60), "← 真实", font=ft(10), fill=(148, 163, 184))
    big = render(s).resize((s * SCALE, s * SCALE), Image.NEAREST)
    sheet.paste(big, (x, 110), big)

d.text((40, H - 32), "#1E293B 侧栏底色上：上方为真实尺寸，下方为该尺寸的 5 倍像素", font=ft(11), fill=(148, 163, 184))

out = BASE / "exports" / "sidebar-check.png"
sheet.save(out)
print(out)
