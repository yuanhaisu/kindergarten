from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\docs\research\2026-11-02-kindergarten-logo-landscape\directions")
REND = BASE / "renders-color"
REND.mkdir(parents=True, exist_ok=True)

SYMBOLS = ["A1-p1", "A1-p2", "A2-p1", "A2-p2"]
SIZES = [512, 144, 81, 32, 16]
LOCKUPS = ["A1-p1-lockup", "A2-p1-lockup"]

for name in SYMBOLS:
    for size in SIZES:
        cairosvg.svg2png(
            url=str(BASE / f"{name}.svg"),
            write_to=str(REND / f"{name}-{size}.png"),
            output_width=size,
            output_height=size,
        )
for name in LOCKUPS:
    cairosvg.svg2png(
        url=str(BASE / f"{name}.svg"),
        write_to=str(REND / f"{name}-960.png"),
        output_width=960,
        output_height=256,
    )

W, H = 1700, 1560
M = 70
COL = 760
GAP = 60
BG = "#F3EFE6"
INK = "#1B2A41"
MUTED = "#5B5B5B"
LINE = "#D8D0C1"

font_title = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 42)
font_sub = ImageFont.truetype(r"C:\Windows\Fonts\msyh.ttc", 20)
font_label = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 24)
font_note = ImageFont.truetype(r"C:\Windows\Fonts\msyh.ttc", 18)
font_meta = ImageFont.truetype(r"C:\Windows\Fonts\msyh.ttc", 15)


def fit(path, size, resample=Image.LANCZOS):
    im = Image.open(path).convert("RGBA")
    bg = Image.new("RGBA", im.size, (243, 239, 230, 255))
    bg.alpha_composite(im)
    return bg.convert("RGB").resize((size, size), resample)


def center_text(draw, text, font, fill, box):
    x0, y0, x1, y1 = box
    bbox = draw.textbbox((0, 0), text, font=font)
    tw, th = bbox[2] - bbox[0], bbox[3] - bbox[1]
    draw.text((x0 + (x1 - x0 - tw) / 2, y0 + (y1 - y0 - th) / 2 - bbox[1]), text, font=font, fill=fill)


img = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(img)
draw.text((M, 36), "A1 vs A2 · 两套色板", font=font_title, fill=INK)
draw.text((M, 92), "上：色板 1 深墨 + 琥珀；中：色板 2 暖墨 + 珊瑚；下：横排字标（字体占位）+ 尺寸阶梯", font=font_sub, fill=MUTED)

cols = [
    ("A1 · 弧巢", "A1-p1", "A1-p2", "A1-p1-lockup"),
    ("A2 · 双弧守护", "A2-p1", "A2-p2", "A2-p1-lockup"),
]

for i, (title, p1, p2, lock) in enumerate(cols):
    x = M + i * (COL + GAP)
    center_text(draw, title, font_label, INK, (x, 130, x + COL, 170))
    # palette symbols
    for row, (name, label) in enumerate([(p1, "色板 1 · 深墨 + 琥珀"), (p2, "色板 2 · 暖墨 + 珊瑚")]):
        y = 190 + row * 410
        im = fit(REND / f"{name}-512.png", 360)
        img.paste(im, (x + (COL - 360) // 2, y))
        center_text(draw, label, font_note, MUTED, (x, y + 370, x + COL, y + 400))
    # lockup
    ly = 1040
    lockim = Image.open(REND / f"{lock}-960.png").convert("RGBA")
    bg = Image.new("RGBA", lockim.size, (243, 239, 230, 255))
    bg.alpha_composite(lockim)
    lockim = bg.convert("RGB").resize((700, 187), Image.LANCZOS)
    img.paste(lockim, (x + (COL - 700) // 2, ly))
    center_text(draw, "横排字标 · 字体占位", font_note, MUTED, (x, ly + 192, x + COL, ly + 218))
    # sizes
    sy = 1270
    sx = x + (COL - (144 + 81 + 32 + 16 + 3 * 24)) // 2
    for s in [144, 81, 32, 16]:
        im = fit(REND / f"{p1}-{s}.png", s)
        img.paste(im, (sx, sy + (144 - s) // 2))
        draw.rectangle((sx - 1, sy + (144 - s) // 2 - 1, sx + s, sy + (144 - s) // 2 + s), outline=LINE, width=1)
        center_text(draw, f"{s}px", font_meta, MUTED, (sx - 10, sy + 150, sx + s + 10, sy + 172))
        sx += s + 24

bar = (M, 1450, W - M, 1530)
draw.rounded_rectangle(bar, radius=18, fill="#1B2A41")
draw.text((M + 24, 1464), "推荐：A2 双弧守护 + 色板 1", font=font_label, fill="#F7F3E8")
draw.text(
    (M + 24, 1500),
    "A2 在 16 / 32px 更稳，双弧的「守护」比 A1 的 C 弧 + 点更明确；色板 1 更专业，色板 2 更亲和。字标为占位，选中后做定制中文字形。",
    font=font_note,
    fill="#E8E2D0",
)

out = BASE / "compare-A1-A2-color.png"
img.save(out, "PNG")
print(out)
