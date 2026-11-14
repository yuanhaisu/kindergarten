"""重渲带文字的 PNG 导出（字形变更后）。

只处理含文字的四类：wordmark / horizontal / horizontal-reversed / stacked。
图形、徽章、app icon、favicon 不含文字，不动。
"""

import io
from pathlib import Path

import cairosvg
from PIL import Image

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2")
M = BASE / "masters"
PNG = BASE / "exports" / "png"
CREAM = (247, 243, 232)


def svg_png(path, width=None, height=None):
    kw = {}
    if width:
        kw["output_width"] = width
    if height:
        kw["output_height"] = height
    png = cairosvg.svg2png(url=str(path), **kw)
    return Image.open(io.BytesIO(png)).convert("RGBA")


def flat(im, size, bg=CREAM):
    canvas = Image.new("RGB", size, bg)
    x = (size[0] - im.width) // 2
    y = (size[1] - im.height) // 2
    canvas.paste(im, (x, y), im)
    return canvas


# wordmark：等比渲染到高 200，再居中放进 960×200
wm = svg_png(M / "wordmark.svg", height=200)
flat(wm, (960, 200)).save(PNG / "wordmark-960.png")
print("wordmark-960.png", wm.size, "→ 960x200")

# horizontal：画布本身就是 960×256
for name, out in (
    ("A1-horizontal-color.svg", "horizontal-960.png"),
    ("A1-horizontal-reversed.svg", "horizontal-reversed-960.png"),
):
    im = svg_png(M / name, width=960)
    im.convert("RGB") if False else None
    canvas = Image.new("RGB", im.size, CREAM)
    canvas.paste(im, (0, 0), im)
    canvas.save(PNG / out)
    print(out, im.size)

im = svg_png(M / "A1-horizontal-color.svg", width=480)
canvas = Image.new("RGB", im.size, CREAM)
canvas.paste(im, (0, 0), im)
canvas.save(PNG / "horizontal-480.png")
print("horizontal-480.png", im.size)

# 中英双语锁图：960 / 480，以及反白
for name, out, w in (
    ("A1-horizontal-bilingual.svg", "horizontal-bilingual-960.png", 960),
    ("A1-horizontal-bilingual.svg", "horizontal-bilingual-480.png", 480),
    ("A1-horizontal-bilingual-reversed.svg", "horizontal-bilingual-reversed-960.png", 960),
):
    im = svg_png(M / name, width=w)
    canvas = Image.new("RGB", im.size, CREAM if "reversed" not in out else (27, 42, 65))
    canvas.paste(im, (0, 0), im)
    canvas.save(PNG / out)
    print(out, im.size)

# stacked：画布 512×480，补到 512×512
st = svg_png(M / "A1-stacked-color.svg", width=512)
flat(st, (512, 512)).save(PNG / "stacked-512.png")
print("stacked-512.png", st.size, "→ 512x512")
