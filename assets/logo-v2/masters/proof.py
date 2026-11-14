import io
from pathlib import Path

import cairosvg
from PIL import Image

BASE = Path(__file__).parent
CREAM = (247, 243, 232)
INK = (27, 42, 65)


def render(name, width, bg=CREAM):
    png = cairosvg.svg2png(url=str(BASE / name), output_width=width)
    im = Image.open(io.BytesIO(png)).convert("RGBA")
    flat = Image.new("RGB", im.size, bg)
    flat.paste(im, (0, 0), im)
    return flat


a = render("A1-horizontal-bilingual.svg", 700)
b = render("A1-horizontal-bilingual-reversed.svg", 700, INK)
c = render("A1-horizontal-color.svg", 700)

W = 740
H = a.height + b.height + c.height + 90
sheet = Image.new("RGB", (W, H), CREAM)
sheet.paste(a, (20, 20))
sheet.paste(b, (20, 30 + a.height))
sheet.paste(c, (20, 40 + a.height + b.height))
out = BASE / "bilingual-proof.png"
sheet.save(out)
print(out, sheet.size)
