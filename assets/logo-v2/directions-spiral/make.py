"""最后一根杠杆：弧的半径沿途收缩，变成一道螺旋。

螺旋不是环——它没有「同一个圆」，所以不可能和知识星球的环撞。
语义上螺旋是壳 / 生长，也贴「养成」。宽度仍等宽，点仍在缺口里。
"""

import io
import math
from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2")
HERE = BASE / "directions-spiral"
HERE.mkdir(parents=True, exist_ok=True)
FONT = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
CREAM = (247, 243, 232)
INK = (27, 42, 65)
MUTED = (120, 130, 145)
AMBER = (232, 162, 58)
INK_HEX = "#1B2A41"
AMBER_HEX = "#E8A23A"

CX = CY = 128.0
TH0, SWEEP = 270.0, 270.0
H = 11.0  # 半宽
SEG = 10


def radius(t, r0, r1):
    return r0 + (r1 - r0) * t


def edge(r0, r1, side, n):
    out = []
    dth = math.radians(SWEEP)
    dr = r1 - r0
    for i in range(n + 1):
        t = i / n
        th = math.radians(TH0 + SWEEP * t)
        r = radius(t, r0, r1) + side * H
        x = CX + r * math.cos(th)
        y = CY + r * math.sin(th)
        dx = dr * math.cos(th) - r * dth * math.sin(th)
        dy = dr * math.sin(th) + r * dth * math.cos(th)
        out.append((t, (x, y), (dx, dy)))
    return out


def bez(points, reverse=False):
    pts = points if not reverse else list(reversed(points))
    d = []
    for i in range(SEG):
        a, b = pts[i], pts[i + 1]
        dt = (b[0] - a[0]) / 3.0
        p0, p1 = a[1], b[1]
        c1 = (p0[0] + a[2][0] * dt, p0[1] + a[2][1] * dt)
        c2 = (p1[0] - b[2][0] * dt, p1[1] - b[2][1] * dt)
        d.append(f"C {c1[0]:.2f} {c1[1]:.2f} {c2[0]:.2f} {c2[1]:.2f} {p1[0]:.2f} {p1[1]:.2f}")
    return " ".join(d)


def cap(deg, r, from_inner=False, steps=10):
    th = math.radians(deg)
    cx = CX + r * math.cos(th)
    cy = CY + r * math.sin(th)
    pts = []
    for i in range(steps + 1):
        a = th + math.pi + math.pi * (i / steps) if from_inner else th + math.pi * (i / steps)
        pts.append(f"{cx + H * math.cos(a):.2f} {cy + H * math.sin(a):.2f}")
    return "L " + " L ".join(pts)


def spiral_path(r0, r1):
    o = edge(r0, r1, +1, SEG)
    i = edge(r0, r1, -1, SEG)
    s = o[0][1]
    d = [f"M {s[0]:.2f} {s[1]:.2f}", bez(o)]
    d.append(cap(TH0 + SWEEP, r1, False))
    d.append(bez(i, reverse=True))
    d.append(cap(TH0, r0, True))
    d.append("Z")
    return " ".join(d)


CASES = [
    ("s1-soft", 90.0, 74.0, "半径 90→74，收得轻"),
    ("s2-deep", 94.0, 66.0, "半径 94→66，收得重"),
]

for key, r0, r1, note in CASES:
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
  <path d="{spiral_path(r0, r1)}" fill="{INK_HEX}"/>
  <circle cx="80" cy="80" r="18" fill="{AMBER_HEX}"/>
</svg>
'''
    p = HERE / f"{key}.svg"
    p.write_text(svg, encoding="utf-8")
    print(p, note)


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
    ("A1 · 正圆弧", BASE / "masters" / "A1-symbol-color.svg", "同一个圆"),
    ("s1 · 轻螺旋", HERE / "s1-soft.svg", "半径收 16"),
    ("s2 · 深螺旋", HERE / "s2-deep.svg", "半径收 28"),
    ("A2 · 双弧", BASE / "directions-a2" / "a2-r22.svg", "两道弧 + 点"),
    ("知识星球", BASE / "directions-child" / "zsxq.png", "闭合环 + 点"),
]

COL_W = 240
PAD = 40
TOP = 145
BIG = 210
LADDER = [81, 48, 32, 16]
W = PAD * 2 + COL_W * len(COLS)
Hh = TOP + BIG + 88 + sum(LADDER) + 34 * len(LADDER) + 40

sheet = Image.new("RGB", (W, Hh), CREAM)
d = ImageDraw.Draw(sheet)
d.text((PAD, 32), "螺旋：弧的半径沿途收缩", font=ft(27, 700), fill=INK)
d.text((PAD, 76), "螺旋没有「同一个圆」，所以不可能读成环。语义上是壳 / 生长。", font=ft(14), fill=MUTED)

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
