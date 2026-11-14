"""A1 换成椭圆弧：读法从「环」变成「轨道」。

椭圆用参数方程取点，再整体旋转；缺口位置沿用 A1（参数 180°–270°），
点放在缺口中线、约 0.78 半径处。
"""

import io
import math
from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2")
HERE = BASE / "directions-ellipse"
HERE.mkdir(parents=True, exist_ok=True)
FONT = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
CREAM = (247, 243, 232)
INK = (27, 42, 65)
MUTED = (120, 130, 145)
AMBER = (232, 162, 58)
INK_HEX = "#1B2A41"
AMBER_HEX = "#E8A23A"

CX = CY = 128.0


def point(rx, ry, phi, deg):
    a = math.radians(deg)
    x = rx * math.cos(a)
    y = ry * math.sin(a)
    p = math.radians(phi)
    return (CX + x * math.cos(p) - y * math.sin(p),
            CY + x * math.sin(p) + y * math.cos(p))


def arc(rx, ry, phi, d1, d2):
    x1, y1 = point(rx, ry, phi, d1)
    x2, y2 = point(rx, ry, phi, d2)
    delta = (d2 - d1) % 360
    large = 1 if delta > 180 else 0
    return f"M {x1:.2f} {y1:.2f} A {rx} {ry} {phi} {large} 1 {x2:.2f} {y2:.2f}"


def dot_at(rx, ry, phi, deg, f=0.78):
    x, y = point(rx, ry, phi, deg)
    return CX + (x - CX) * f, CY + (y - CY) * f


CASES = [
    ("e1-flat", 106, 76, 0, 270, 540, 225, 18, "横椭圆，缺口左上"),
    ("e2-tilt", 104, 78, -20, 270, 540, 225, 18, "倾斜 -20°，轨道感"),
    ("e3-open", 106, 76, 0, 290, 530, 230, 18, "弧缩到 240°，口更大"),
]

# e4：真正读作「轨道」的做法——细的闭合椭圆 + 一颗贴在轨道上的点
_e4_dot = point(108, 64, -16, -38)
(HERE / "e4-orbit.svg").write_text(
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
  <ellipse cx="128" cy="128" rx="108" ry="64" transform="rotate(-16 128 128)" fill="none" stroke="{INK_HEX}" stroke-width="8"/>
  <circle cx="{_e4_dot[0]:.2f}" cy="{_e4_dot[1]:.2f}" r="19" fill="{AMBER_HEX}"/>
</svg>
''',
    encoding="utf-8",
)

for key, rx, ry, phi, d1, d2, da, dr, note in CASES:
    dx, dy = dot_at(rx, ry, phi, da)
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
  <path d="{arc(rx, ry, phi, d1, d2)}" fill="none" stroke="{INK_HEX}" stroke-width="22" stroke-linecap="round"/>
  <circle cx="{dx:.2f}" cy="{dy:.2f}" r="{dr}" fill="{AMBER_HEX}"/>
</svg>
'''
    p = HERE / f"{key}.svg"
    p.write_text(svg, encoding="utf-8")
    print(p, note, f"dot=({dx:.1f},{dy:.1f})")


def ft(size, weight=400):
    f = ImageFont.truetype(FONT, size)
    try:
        f.set_variation_by_axes([weight])
    except Exception:
        pass
    return f


def render(path, size):
    png = cairosvg.svg2png(url=str(path), output_width=size, output_height=size)
    im = Image.open(io.BytesIO(png)).convert("RGBA")
    flat = Image.new("RGB", im.size, CREAM)
    flat.paste(im, (0, 0), im)
    return flat


COLS = [
    ("A1 · 正圆", BASE / "masters" / "A1-symbol-color.svg", "缺口 90°"),
    ("e1 · 横椭圆", HERE / "e1-flat.svg", "轨道，缺口左上"),
    ("e2 · 倾斜", HERE / "e2-tilt.svg", "-20°，轨道感最强"),
    ("e3 · 椭圆开大口", HERE / "e3-open.svg", "弧 240°"),
    ("e4 · 真轨道", HERE / "e4-orbit.svg", "细闭合椭圆 + 点贴在上面"),
    ("知识星球", BASE / "directions-child" / "zsxq.png", "正圆，近乎闭合"),
]

COL_W = 240
PAD = 40
TOP = 145
BIG = 214
LADDER = [81, 48, 32, 16]
W = PAD * 2 + COL_W * len(COLS)
H = TOP + BIG + 88 + sum(LADDER) + 34 * len(LADDER) + 40

sheet = Image.new("RGB", (W, H), CREAM)
d = ImageDraw.Draw(sheet)
d.text((PAD, 32), "圆弧换成椭圆", font=ft(27, 700), fill=INK)
d.text((PAD, 76), "同一个点、同一个缺口位置，只把正圆拉成椭圆。", font=ft(14), fill=MUTED)

zs = Image.open(BASE / "directions-child" / "zsxq.png").convert("RGB")
for i, (name, path, note) in enumerate(COLS):
    x = PAD + i * COL_W
    is_img = str(path).lower().endswith(".png")
    big = zs.resize((BIG, BIG), Image.LANCZOS) if is_img else render(path, BIG)
    sheet.paste(big, (x + (COL_W - BIG) // 2, TOP))
    d.text((x + 6, TOP + BIG + 12), name, font=ft(15, 700), fill=INK)
    d.text((x + 6, TOP + BIG + 36), note, font=ft(11), fill=(160, 110, 40))

    y = TOP + BIG + 88
    for s in LADDER:
        im = zs.resize((s, s), Image.LANCZOS) if is_img else render(path, s)
        sheet.paste(im, (x + 6, y))
        d.text((x + 12 + s, y + s // 2 - 8), str(s), font=ft(10), fill=MUTED)
        y += s + 32

out = HERE / "compare.png"
sheet.save(out)
print(out)
