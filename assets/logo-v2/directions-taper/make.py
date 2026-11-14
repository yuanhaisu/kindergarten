"""A1 的第三种可能：弧从等宽改成有粗细变化。

等宽弧读作「工业环」；有笔锋的弧读作「一笔」。知识星球是绝对等宽的，
所以这一步不是改形，是改「它是什么东西」。

中线仍是半径 84 的圆，从 270° 顺时针到 180°；宽度沿参数 t 平滑变化，
两端保留圆头。外沿/内沿用 Hermite→三次贝塞尔拟合，锚点保持在低位。
"""

import io
import math
from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2")
HERE = BASE / "directions-taper"
HERE.mkdir(parents=True, exist_ok=True)
FONT = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
CREAM = (247, 243, 232)
INK = (27, 42, 65)
MUTED = (120, 130, 145)
AMBER = (232, 162, 58)
INK_HEX = "#1B2A41"
AMBER_HEX = "#E8A23A"

CX = CY = 128.0
R = 84.0
TH0, SWEEP = 270.0, 270.0  # 顺时针，跨 270°；终点角 = 540°（等价 180°）
TH1 = TH0 + SWEEP
SEG = 10


def theta(t):
    return math.radians(TH0 + SWEEP * t)


def half_width(t, h0, h1, power=1.5):
    return h1 + (h0 - h1) * (1 - t) ** power


def edge_points(h0, h1, side, n):
    """side=+1 外沿，-1 内沿。返回 [(t, point, tangent)]。"""
    out = []
    dth = math.radians(SWEEP)
    for i in range(n + 1):
        t = i / n
        th = theta(t)
        h = half_width(t, h0, h1)
        r = R + side * h
        x = CX + r * math.cos(th)
        y = CY + r * math.sin(th)
        dh = -1.5 * (h0 - h1) * (1 - t) ** 0.5  # d h / d t
        dx = side * dh * math.cos(th) - r * dth * math.sin(th)
        dy = side * dh * math.sin(th) + r * dth * math.cos(th)
        out.append((t, (x, y), (dx, dy)))
    return out


def bezier_edge(points, t0, t1, reverse=False):
    pts = points if not reverse else list(reversed(points))
    d = []
    for i in range(SEG):
        a = pts[i]
        b = pts[i + 1]
        dt = (b[0] - a[0]) / 3.0
        p0, p1 = a[1], b[1]
        c1 = (p0[0] + a[2][0] * dt, p0[1] + a[2][1] * dt)
        c2 = (p1[0] - b[2][0] * dt, p1[1] - b[2][1] * dt)
        d.append(f"C {c1[0]:.2f} {c1[1]:.2f} {c2[0]:.2f} {c2[1]:.2f} {p1[0]:.2f} {p1[1]:.2f}")
    return " ".join(d)


def cap(center_angle, h, from_inner=False, steps=10):
    """端点圆头：沿中心线端点画半圆，朝行进方向鼓出。"""
    th = math.radians(center_angle)
    cx = CX + R * math.cos(th)
    cy = CY + R * math.sin(th)
    pts = []
    for i in range(steps + 1):
        a = th + math.pi + math.pi * (i / steps) if from_inner else th + math.pi * (i / steps)
        pts.append(f"{cx + h * math.cos(a):.2f} {cy + h * math.sin(a):.2f}")
    return "L " + " L ".join(pts)


def taper_path(h0, h1):
    outer = edge_points(h0, h1, +1, SEG)
    inner = edge_points(h0, h1, -1, SEG)
    start = outer[0][1]
    d = [f"M {start[0]:.2f} {start[1]:.2f}"]
    d.append(bezier_edge(outer, 0, 1))
    d.append(cap(TH1, half_width(1, h0, h1), False))
    d.append(bezier_edge(inner, 0, 1, reverse=True))
    d.append(cap(TH0, half_width(0, h0, h1), True))
    d.append("Z")
    return " ".join(d)


CASES = [
    ("taper-a", 12.0, 4.5, "起笔粗（上），收笔细（左）"),
    ("taper-b", 4.5, 12.0, "起笔细（上），收笔粗（左）"),
]

for key, h0, h1, note in CASES:
    d = taper_path(h0, h1)
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
  <path d="{d}" fill="{INK_HEX}"/>
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
    ("A1 · 等宽", BASE / "masters" / "A1-symbol-color.svg", "一条工业环"),
    ("a · 上粗左细", HERE / "taper-a.svg", "笔锋朝点收"),
    ("b · 上细左粗", HERE / "taper-b.svg", "重量压在左下"),
    ("知识星球", BASE / "directions-child" / "zsxq.png", "绝对等宽的环"),
]

COL_W = 280
PAD = 44
TOP = 150
BIG = 230
LADDER = [81, 48, 32, 16]
W = PAD * 2 + COL_W * len(COLS)
H = TOP + BIG + 92 + sum(LADDER) + 34 * len(LADDER) + 40

sheet = Image.new("RGB", (W, H), CREAM)
d = ImageDraw.Draw(sheet)
d.text((PAD, 34), "有笔锋的弧", font=ft(27, 700), fill=INK)
d.text((PAD, 78), "同样的位置、同样的点，只有弧从等宽变成一笔。", font=ft(14), fill=MUTED)

zs = Image.open(BASE / "directions-child" / "zsxq.png").convert("RGB")
for i, (name, path, note) in enumerate(COLS):
    x = PAD + i * COL_W
    is_img = str(path).lower().endswith(".png")
    big = zs.resize((BIG, BIG), Image.LANCZOS) if is_img else render(path, BIG)
    sheet.paste(big, (x + (COL_W - BIG) // 2, TOP))
    d.text((x + 8, TOP + BIG + 14), name, font=ft(17, 700), fill=INK)
    d.text((x + 8, TOP + BIG + 40), note, font=ft(12), fill=(160, 110, 40))

    y = TOP + BIG + 92
    for s in LADDER:
        im = zs.resize((s, s), Image.LANCZOS) if is_img else render(path, s)
        sheet.paste(im, (x + 8, y))
        d.text((x + 14 + s, y + s // 2 - 8), str(s), font=ft(11), fill=MUTED)
        y += s + 34

out = HERE / "compare.png"
sheet.save(out)
print(out)
