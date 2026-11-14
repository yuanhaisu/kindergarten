from pathlib import Path

OUT = Path(r"D:\yuanhaisu\personal_projects\kindergarten\docs\research\2026-11-02-kindergarten-logo-landscape\directions")
OUT.mkdir(parents=True, exist_ok=True)

FONT = "Microsoft YaHei, Source Han Sans SC, Noto Sans CJK SC, sans-serif"

PALETTES = {
    "p1": {"ink": "#1B2A41", "accent": "#E8A23A", "cream": "#F7F3E8", "name": "深墨 + 琥珀"},
    "p2": {"ink": "#2B2A33", "accent": "#E2725B", "cream": "#FBF6EE", "name": "暖墨 + 珊瑚"},
}


def star(cx, cy, r):
    k = r * 0.18
    return (
        f"M{cx} {cy - r} Q{cx + k} {cy - k} {cx + r} {cy} "
        f"Q{cx + k} {cy + k} {cx} {cy + r} "
        f"Q{cx - k} {cy + k} {cx - r} {cy} "
        f"Q{cx - k} {cy - k} {cx} {cy - r} Z"
    )


def symbol_a1(ink, accent):
    return f'''<path d="M 128 44 A 84 84 0 1 1 44 128" fill="none" stroke="{ink}" stroke-width="22" stroke-linecap="round"/>
  <circle cx="80" cy="80" r="18" fill="{accent}"/>'''


def symbol_a2(ink, accent):
    return f'''<path d="M 96 56 A 72 72 0 0 0 96 200" fill="none" stroke="{ink}" stroke-width="18" stroke-linecap="round"/>
  <path d="M 160 56 A 72 72 0 0 1 160 200" fill="none" stroke="{ink}" stroke-width="18" stroke-linecap="round"/>
  <path d="{star(128, 128, 20)}" fill="{accent}"/>'''


def square(name, body):
    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">{name}</title>
  {body}
</svg>
'''


def lockup(name, body):
    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 256" width="960" height="256" role="img" aria-labelledby="title">
  <title id="title">{name}</title>
  <g transform="translate(90 28) scale(0.78)">{body}</g>
  <text x="360" y="158" font-family="{FONT}" font-size="84" font-weight="700" letter-spacing="6" fill="{INK_PLACEHOLDER}">养成星球</text>
</svg>
'''


files = {}
for key, pal in PALETTES.items():
    ink, accent = pal["ink"], pal["accent"]
    for concept, body in [("a1", symbol_a1(ink, accent)), ("a2", symbol_a2(ink, accent))]:
        files[f"{concept.upper()}-{key}.svg"] = square(f"{concept.upper()} {pal['name']}", body)
        lock = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 256" width="960" height="256" role="img" aria-labelledby="title">
  <title id="title">{concept.upper()} {pal['name']} 横排</title>
  <g transform="translate(90 28) scale(0.78)">{body}</g>
  <text x="360" y="158" font-family="{FONT}" font-size="84" font-weight="700" letter-spacing="6" fill="{ink}">养成星球</text>
</svg>
'''
        files[f"{concept.upper()}-{key}-lockup.svg"] = lock

for name, content in files.items():
    path = OUT / name
    path.write_text(content, encoding="utf-8")
    print(path)
