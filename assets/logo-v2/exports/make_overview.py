from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2")
OUT = BASE / "exports" / "kit-overview.png"

W, H = 1700, 1280
BG = "#E9E4D8"
INK = "#1B2A41"
ACCENT = "#E8A23A"
CREAM = "#F7F3E8"
MUTED = "#5B5B5B"
LINE = "#CFC7B8"

font_title = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 42)
font_sub = ImageFont.truetype(r"C:\Windows\Fonts\msyh.ttc", 19)
font_label = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 20)
font_note = ImageFont.truetype(r"C:\Windows\Fonts\msyh.ttc", 15)


def load(path, size=None, resample=Image.LANCZOS):
    im = Image.open(path).convert("RGBA")
    bg = Image.new("RGBA", im.size, (233, 228, 216, 255))
    bg.alpha_composite(im)
    im = bg.convert("RGB")
    if size:
        im = im.resize(size, resample)
    return im


def center_text(draw, text, font, fill, box):
    x0, y0, x1, y1 = box
    bbox = draw.textbbox((0, 0), text, font=font)
    tw, th = bbox[2] - bbox[0], bbox[3] - bbox[1]
    draw.text((x0 + (x1 - x0 - tw) / 2, y0 + (y1 - y0 - th) / 2 - bbox[1]), text, font=font, fill=fill)


img = Image.new("RGB", (W, H), BG)
draw = ImageDraw.Draw(img)
draw.text((60, 30), "养成星球 · Logo Kit", font=font_title, fill=INK)
draw.text((60, 84), "A1 弧巢 + 色板 1（深墨 / 琥珀 / 奶油）· 未接入三端", font=font_sub, fill=MUTED)

# Row 1
img.paste(load(BASE / "exports/png/horizontal-960.png", (800, 213)), (60, 130))
center_text(draw, "横排主锁图", font_note, MUTED, (60, 350, 860, 374))
img.paste(load(BASE / "exports/png/stacked-512.png", (300, 300)), (900, 130))
center_text(draw, "竖排锁图", font_note, MUTED, (900, 440, 1200, 464))
img.paste(load(BASE / "exports/png/badge-cream-512.png", (300, 300)), (1240, 130))
center_text(draw, "奶油徽章", font_note, MUTED, (1240, 440, 1540, 464))

# Row 2
img.paste(load(BASE / "exports/png/badge-ink-512.png", (300, 300)), (60, 500))
center_text(draw, "深墨徽章", font_note, MUTED, (60, 810, 360, 834))
img.paste(load(BASE / "exports/app-icon/app-icon-512.png", (300, 300)), (400, 500))
center_text(draw, "App icon", font_note, MUTED, (400, 810, 700, 834))
img.paste(load(BASE / "exports/png/symbol-512.png", (300, 300)), (740, 500))
center_text(draw, "图形 · 彩色", font_note, MUTED, (740, 810, 1040, 834))
img.paste(load(BASE / "exports/png/symbol-reversed-512.png", (300, 300)), (1080, 500))
center_text(draw, "图形 · 反白", font_note, MUTED, (1080, 810, 1380, 834))
img.paste(load(BASE / "exports/png/symbol-mono-512.png", (200, 200)), (1440, 550))
center_text(draw, "单色", font_note, MUTED, (1400, 770, 1680, 794))

# Row 3 size ladder + favicon + wordmark
sx = 70
sy = 880
for s in [144, 81, 32, 16]:
    im = load(BASE / f"exports/png/symbol-{s}.png", (s, s))
    img.paste(im, (sx, sy + (144 - s) // 2))
    draw.rectangle((sx - 1, sy + (144 - s) // 2 - 1, sx + s, sy + (144 - s) // 2 + s), outline=LINE, width=1)
    center_text(draw, f"{s}px", font_note, MUTED, (sx - 10, sy + 150, sx + s + 10, sy + 172))
    sx += s + 24

fx = 500
for s in [32, 16]:
    im = load(BASE / f"exports/web/favicon-{s}.png", (s, s))
    img.paste(im, (fx, sy + (144 - s) // 2))
    center_text(draw, f"{s}", font_note, MUTED, (fx - 10, sy + 150, fx + s + 10, sy + 172))
    fx += s + 50
center_text(draw, "favicon", font_note, MUTED, (490, sy + 174, 700, sy + 196))

wm = load(BASE / "exports/png/wordmark-960.png", (600, 125))
img.paste(wm, (800, sy + 10))
center_text(draw, "字标 · Noto Sans SC Bold（已轮廓化）", font_note, MUTED, (780, sy + 150, 1420, sy + 174))

# Footer
bar = (60, 1120, W - 60, 1230)
draw.rounded_rectangle(bar, radius=18, fill=INK)
draw.text((84, 1138), "A1 + P1 · 深墨 #1B2A41 / 琥珀 #E8A23A / 奶油 #F7F3E8", font=font_label, fill=CREAM)
draw.text(
    (84, 1174),
    "字标基于 Noto Sans SC Bold（SIL OFL）轮廓化；正式商用建议做定制字形。符号为 A1 原始几何，未描摹任何竞品。",
    font=font_note,
    fill="#E8E2D0",
)
draw.text(
    (84, 1198),
    "旧 assets/logo.* 仍在线上；本 kit 尚未接入 teacher / parent / admin。",
    font=font_note,
    fill="#E8E2D0",
)

img.save(OUT, "PNG")
print(OUT)
