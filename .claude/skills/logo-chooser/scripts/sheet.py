#!/usr/bin/env python3
"""把站 4 的稿子拼成一张对比图：每张同时显示大图和 16px 版本。

为什么两张一起给：轴上的差异在 256px 看得很清楚，在 16px 可能根本不存在。
`identity` 的原话是「16 张只有 4 种读法」——那就得把 16px 并排摆出来，
让人自己看见哪些其实同一张。

不合格的稿子标红，并把原因写在下面。
"""
from __future__ import annotations

import argparse
import io
import json
import sys
from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw, ImageFont

SKILL = Path(__file__).resolve().parent.parent
FONT_CANDIDATES = [
    "C:/Windows/Fonts/msyh.ttc",
    "C:/Windows/Fonts/simhei.ttf",
    "arial.ttf",
]


def _font(size: int):
    for p in FONT_CANDIDATES:
        try:
            return ImageFont.truetype(p, size)
        except OSError:
            continue
    return ImageFont.load_default()


def render(svg_text: str, size: int) -> Image.Image:
    png = cairosvg.svg2png(bytestring=svg_text.encode(),
                           output_width=size, output_height=size)
    return Image.open(io.BytesIO(png)).convert("RGBA")


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--brand", default="养成星球")
    ap.add_argument("--round", help="只出一轮（如 4.2）；不给就出全部")
    ap.add_argument("--out", type=Path)
    args = ap.parse_args()

    run_dir = SKILL / "runs" / args.brand
    if args.round:
        srcs = [run_dir / f"round-4.{args.round}.out.json"]
    else:
        srcs = sorted(run_dir.glob("round-4.*.out.json"))
    if not srcs:
        print("❌ 没有任何 round-4.*.out.json，先跑 construct.py")
        return 1

    cells = []
    for src in srcs:
        if not src.exists():
            print(f"❌ 找不到 {src}")
            return 1
        rnd = json.loads(src.read_text(encoding="utf-8"))
        bad = set(rnd.get("failed") or [])
        why = {d["id"]: "；".join(d.get("reasons") or [])
               for d in (rnd.get("failed_detail") or [])}
        for d in rnd["drafts"]:
            svg = (run_dir / "drafts" / d["file"]).read_text(encoding="utf-8")
            cells.append({
                "label": f"{rnd['round']} {d['id']}",
                "file": d["file"],
                "big": render(svg, 160),
                "small": render(svg, 16).resize((48, 48), Image.NEAREST),
                "bad": d["id"] in bad,
                "why": why.get(d["id"], ""),
            })

    COLS, CW, CH = 6, 182, 214
    nrows = (len(cells) + COLS - 1) // COLS
    sheet = Image.new("RGB", (COLS * CW, nrows * CH), "white")
    draw = ImageDraw.Draw(sheet)
    f13, f10 = _font(13), _font(10)

    for i, c in enumerate(cells):
        x, y = (i % COLS) * CW, (i // COLS) * CH
        sheet.paste(c["big"], (x + 10, y + 8), c["big"])
        sheet.paste(c["small"], (x + 118, y + 118), c["small"])
        if c["bad"]:
            draw.rectangle([x + 8, y + 6, x + 170, y + 170], outline="#d00", width=3)
        draw.text((x + 10, y + 174), c["label"], fill="#d00" if c["bad"] else "#222", font=f13)
        if c["bad"]:
            line = c["why"][:26]
            draw.text((x + 10, y + 191), line, fill="#d00", font=f10)

    out = args.out or (run_dir / "_check" / "station4-all.png")
    out.parent.mkdir(parents=True, exist_ok=True)
    sheet.save(out)
    print(f"✅ {out}  ({len(cells)} 张 · 红框 = 硬检查不合格)")
    print("   每格：左 160px 大图 · 右下 48px 是同一张缩到 16px 再放大")
    return 0


if __name__ == "__main__":
    sys.exit(main())
