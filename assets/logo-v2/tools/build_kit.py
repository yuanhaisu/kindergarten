from pathlib import Path

from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont
from fontTools.pens.svgPathPen import SVGPathPen

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\assets\logo-v2")
MASTERS = BASE / "masters"
APP = BASE / "exports" / "app-icon"
WEB = BASE / "exports" / "web"
PNG = BASE / "exports" / "png"
for d in (MASTERS, APP, WEB, PNG):
    d.mkdir(parents=True, exist_ok=True)

INK = "#1B2A41"
ACCENT = "#E8A23A"
CREAM = "#F7F3E8"
FONT_PATH = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
TEXT = "养成星球"

font = TTFont(FONT_PATH)
font = instantiateVariableFont(font, {"wght": 700}, inplace=False)
cmap = font.getBestCmap()
glyphs = font.getGlyphSet()
hmtx = font["hmtx"]
upm = font["head"].unitsPerEm


from fontTools.pens.boundsPen import BoundsPen


def _metrics(ch):
    name = cmap[ord(ch)]
    bpen = BoundsPen(glyphs)
    glyphs[name].draw(bpen)
    x0, y0, x1, y1 = bpen.bounds
    pen = SVGPathPen(glyphs)
    glyphs[name].draw(pen)
    return dict(x0=x0, bw=(x1 - x0), area=(x1 - x0) * (y1 - y0), d=pen.getCommands())


# 字面率校正：四个字的墨面积并不相等（「星」比其余小 12%），
# 等比缩放到同一视觉面积，再按墨迹边缘排距，不用 em 框前进宽度。
_M = {ch: _metrics(ch) for ch in TEXT}
_TARGET = max(m["area"] for m in _M.values())
for _m in _M.values():
    _m["s"] = (_TARGET / _m["area"]) ** 0.5
    _m["w"] = _m["bw"] * _m["s"]


def build_glyphs(font_size, ink_gap):
    """ink_gap 为字体单位下的墨迹间距。返回 (items, total_width, 1.0)。"""
    k = font_size / upm
    x = 0.0
    items = []
    for ch in TEXT:
        m = _M[ch]
        items.append((x - m["x0"] * m["s"] * k, m["d"], m["s"] * k))
        x += m["w"] * k + ink_gap * k
    total = x - ink_gap * k
    return items, total, 1.0


def glyph_group(items, _scale, x_offset, baseline, fill):
    parts = []
    for dx, d, s in items:
        parts.append(
            f'<g transform="translate({x_offset + dx:.2f} {baseline}) scale({s:.6f} {-s:.6f})">'
            f'<path d="{d}" fill="{fill}"/></g>'
        )
    return "\n  ".join(parts)


def symbol(arc_color, dot_color, tx=0, ty=0, scale=1.0):
    return f'''<g transform="translate({tx} {ty}) scale({scale})">
    <path d="M 128 44 A 84 84 0 1 1 44 128" fill="none" stroke="{arc_color}" stroke-width="22" stroke-linecap="round"/>
    <circle cx="80" cy="80" r="18" fill="{dot_color}"/>
  </g>'''


def write(path, content):
    path.write_text(content, encoding="utf-8")
    print(path)


# wordmark alone
items, wm_width, wm_scale = build_glyphs(88, 150)
write(
    MASTERS / "wordmark.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {wm_width + 40:.0f} 200" width="{wm_width + 40:.0f}" height="200" role="img" aria-labelledby="title">
  <title id="title">养成星球 wordmark</title>
  {glyph_group(items, wm_scale, 20, 150, INK)}
</svg>
''',
)

# symbol variants
write(
    MASTERS / "A1-symbol-color.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 A1 symbol</title>
  {symbol(INK, ACCENT)}
</svg>
''',
)
write(
    MASTERS / "A1-symbol-mono.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 A1 symbol mono</title>
  {symbol(INK, INK)}
</svg>
''',
)
write(
    MASTERS / "A1-symbol-reversed.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 A1 symbol reversed</title>
  {symbol(CREAM, ACCENT)}
</svg>
''',
)
write(
    MASTERS / "A1-symbol-reversed-mono.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 A1 symbol reversed mono</title>
  {symbol(CREAM, CREAM)}
</svg>
''',
)

# horizontal lockup
wm_items, wm_width, wm_scale = build_glyphs(88, 150)
write(
    MASTERS / "A1-horizontal-color.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 256" width="960" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 horizontal</title>
  {symbol(INK, ACCENT, 140.5, 28, 0.78)}
  {glyph_group(wm_items, wm_scale, 410.5, 160, INK)}
</svg>
''',
)
write(
    MASTERS / "A1-horizontal-reversed.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 256" width="960" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 horizontal reversed</title>
  {symbol(CREAM, ACCENT, 140.5, 28, 0.78)}
  {glyph_group(wm_items, wm_scale, 410.5, 160, CREAM)}
</svg>
''',
)

# stacked lockup
st_items, st_width, st_scale = build_glyphs(56, 145)
write(
    MASTERS / "A1-stacked-color.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512" role="img" aria-labelledby="title">
  <title id="title">养成星球 stacked</title>
  {symbol(INK, ACCENT, 146, 60, 0.86)}
  {glyph_group(st_items, st_scale, (512 - st_width) / 2, 400, INK)}
</svg>
''',
)

# badges
write(
    MASTERS / "A1-badge-cream.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 cream badge</title>
  <circle cx="128" cy="128" r="124" fill="{CREAM}"/>
  <circle cx="128" cy="128" r="121" fill="none" stroke="{INK}" stroke-width="2.5"/>
  {symbol(INK, ACCENT)}
</svg>
''',
)
write(
    MASTERS / "A1-badge-ink.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 ink badge</title>
  <circle cx="128" cy="128" r="124" fill="{INK}"/>
  <circle cx="128" cy="128" r="121" fill="none" stroke="{CREAM}" stroke-width="2.5"/>
  {symbol(CREAM, ACCENT)}
</svg>
''',
)

# app icon: cream tile + symbol; badge alternate
write(
    APP / "app-icon.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 app icon</title>
  <rect width="256" height="256" rx="56" fill="{CREAM}"/>
  {symbol(INK, ACCENT)}
</svg>
''',
)
write(
    APP / "app-icon-badge.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 app icon badge</title>
  <rect width="256" height="256" rx="56" fill="{CREAM}"/>
  <circle cx="128" cy="128" r="118" fill="{CREAM}"/>
  <circle cx="128" cy="128" r="115" fill="none" stroke="{INK}" stroke-width="2.5"/>
  {symbol(INK, ACCENT, 0, 0, 0.92)}
</svg>
''',
)

# favicon source: same as app icon symbol
write(
    WEB / "favicon.svg",
    f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">养成星球 favicon</title>
  <rect width="256" height="256" rx="56" fill="{CREAM}"/>
  {symbol(INK, ACCENT)}
</svg>
''',
)

write(
    WEB / "site.webmanifest",
    '''{
  "name": "养成星球",
  "icons": [
    {"src": "/icon-192.png", "sizes": "192x192", "type": "image/png"},
    {"src": "/icon-512.png", "sizes": "512x512", "type": "image/png"},
    {"src": "/maskable-512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"}
  ],
  "theme_color": "#F7F3E8",
  "background_color": "#F7F3E8",
  "display": "standalone",
  "start_url": "/"
}
''',
)
write(
    WEB / "head-snippet.html",
    '''<link rel="icon" href="/favicon.ico" sizes="48x48">
<link rel="icon" href="/favicon.svg" type="image/svg+xml">
<link rel="apple-touch-icon" href="/apple-touch-icon.png">
<link rel="manifest" href="/site.webmanifest">
<meta name="theme-color" content="#F7F3E8">
''',
)
