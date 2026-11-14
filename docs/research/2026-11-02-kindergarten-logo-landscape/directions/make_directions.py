from pathlib import Path

OUT = Path(r"D:\yuanhaisu\personal_projects\kindergarten\docs\research\2026-11-02-kindergarten-logo-landscape\directions")
OUT.mkdir(parents=True, exist_ok=True)

INK = "#111111"


def star(cx, cy, r):
    k = r * 0.18
    return (
        f"M{cx} {cy - r} Q{cx + k} {cy - k} {cx + r} {cy} "
        f"Q{cx + k} {cy + k} {cx} {cy + r} "
        f"Q{cx - k} {cy + k} {cx - r} {cy} "
        f"Q{cx - k} {cy - k} {cx} {cy - r} Z"
    )


def svg_square(title, body):
    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256" role="img" aria-labelledby="title">
  <title id="title">{title}</title>
  {body}
</svg>
'''


def svg_wide(title, body):
    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 960 256" width="960" height="256" role="img" aria-labelledby="title">
  <title id="title">{title}</title>
  {body}
</svg>
'''


# A1 弧巢：非对称 C 弧 + 缺口里的种子
a1 = svg_square(
    "A1 弧巢",
    f'''<path d="M 128 44 A 84 84 0 1 1 44 128" fill="none" stroke="{INK}" stroke-width="22" stroke-linecap="round"/>
  <circle cx="80" cy="80" r="18" fill="{INK}"/>''',
)

# A2 双弧守护：一对括号弧 + 中心星
a2 = svg_square(
    "A2 双弧守护",
    f'''<path d="M 96 56 A 72 72 0 0 0 96 200" fill="none" stroke="{INK}" stroke-width="18" stroke-linecap="round"/>
  <path d="M 160 56 A 72 72 0 0 1 160 200" fill="none" stroke="{INK}" stroke-width="18" stroke-linecap="round"/>
  <path d="{star(128, 128, 20)}" fill="{INK}"/>''',
)

# A3 环 + 星：环留一个缺口，星从缺口长出来
a3 = svg_square(
    "A3 环 + 星",
    f'''<path d="M 170 55 A 84 84 0 1 1 86 55" fill="none" stroke="{INK}" stroke-width="22" stroke-linecap="round"/>
  <path d="{star(128, 36, 16)}" fill="{INK}"/>''',
)

# B1 三步成星：三根递增高圆角柱 + 星
bars = "".join(
    f'<rect x="{x}" y="{y}" width="28" height="{h}" rx="14" fill="{INK}"/>'
    for x, y, h in [(56, 160, 40), (104, 132, 68), (152, 96, 104)]
)
b1 = svg_square(
    "B1 三步成星",
    f'''<g transform="rotate(-8 128 160)">{bars}</g>
  <path d="{star(208, 80, 20)}" fill="{INK}"/>''',
)

# B2 星轨刻度：3/4 轨道弧 + 刻度 + 星
b2 = svg_square(
    "B2 星轨刻度",
    f'''<path d="M 128 40 A 88 88 0 1 1 40 128" fill="none" stroke="{INK}" stroke-width="18" stroke-linecap="round"/>
  <path d="M 205 108 L 232 108 M 128 205 L 128 232 M 64 190 L 46 208" stroke="{INK}" stroke-width="12" stroke-linecap="round"/>
  <path d="{star(24, 108, 18)}" fill="{INK}"/>''',
)

# B3 向上折线：三级台阶 + 星
b3 = svg_square(
    "B3 向上折线",
    f'''<path d="M 48 192 H 80 Q 88 192 88 184 V 152 H 120 Q 128 152 128 144 V 104 H 160 Q 168 104 168 96 V 56 H 184"
        fill="none" stroke="{INK}" stroke-width="18" stroke-linecap="round" stroke-linejoin="round"/>
  <path d="{star(208, 48, 18)}" fill="{INK}"/>''',
)

FONT = "Microsoft YaHei, Source Han Sans SC, Noto Sans CJK SC, sans-serif"

# C1 字标 + 星点：完整中文字标，「星」上方一颗星
c1 = svg_wide(
    "C1 字标 + 星点",
    f'''<text x="480" y="168" text-anchor="middle" font-family="{FONT}" font-size="96" font-weight="700"
        letter-spacing="8" fill="{INK}">养成星球</text>
  <path d="{star(532, 56, 15)}" fill="{INK}"/>''',
)

# C2 「星」字标：单字 + 右上星点
c2 = svg_square(
    "C2 「星」字标",
    f'''<text x="122" y="202" text-anchor="middle" font-family="{FONT}" font-size="178" font-weight="700"
        fill="{INK}">星</text>
  <path d="{star(202, 58, 14)}" fill="{INK}"/>''',
)

# C3 竖排字标：两行中文 + 星点
c3 = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512" role="img" aria-labelledby="title">
  <title id="title">C3 竖排字标</title>
  <path d="{star(256, 92, 18)}" fill="{INK}"/>
  <text x="256" y="236" text-anchor="middle" font-family="{FONT}" font-size="110" font-weight="700"
        letter-spacing="10" fill="{INK}">养成</text>
  <text x="256" y="366" text-anchor="middle" font-family="{FONT}" font-size="110" font-weight="700"
        letter-spacing="10" fill="{INK}">星球</text>
</svg>
'''

files = {
    "A1-arc-nest.svg": a1,
    "A2-double-arc.svg": a2,
    "A3-arc-orbit.svg": a3,
    "B1-three-steps.svg": b1,
    "B2-orbit-ticks.svg": b2,
    "B3-rising-steps.svg": b3,
    "C1-wordmark-star.svg": c1,
    "C2-xing-monogram.svg": c2,
    "C3-stacked-wordmark.svg": c3,
}

for name, content in files.items():
    path = OUT / name
    path.write_text(content, encoding="utf-8")
    print(path)
