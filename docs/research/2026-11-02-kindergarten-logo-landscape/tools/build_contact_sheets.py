import csv
import json
from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\docs\research\2026-11-02-kindergarten-logo-landscape")
CSV_PATH = BASE.parent / "2026-11-02-kindergarten-logo-landscape.csv"
REPORT = BASE / "download-report.json"
TMP = BASE / "render-tmp"
TMP.mkdir(parents=True, exist_ok=True)

LAYER_TITLES = {
    "L1": "L1 · 中国家园共育 / 园务管理 SaaS",
    "L2": "L2 · 中国幼儿园品牌",
    "L3": "L3 · 中国幼教内容 / 课程 / 玩具",
    "L4": "L4 · 国际同类产品",
}
LAYER_SUBS = {
    "L1": "直接竞品层：家长与老师每天打开的 App / 小程序",
    "L2": "园所品牌层：家长对「幼儿园」本身的视觉预期",
    "L3": "内容 / 玩具层：童趣、色彩、角色语言的来源",
    "L4": "国际对照层：更克制、更企业化的另一条路",
}

W = 1720
M = 50
COLS = 5
GAP = 24
TW, TH = 300, 200
LABEL_H = 52
ROW_H = TH + LABEL_H + 24
HEADER = 120
FOOTER = 56
BG = "#FBF9F3"
NAVY = "#0B173D"
MUTED = "#5B5B5B"
LINE = "#D8D0C1"
CHECK_A = "#F7F3EA"
CHECK_B = "#ECE6D9"

font_title = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 34)
font_sub = ImageFont.truetype(r"C:\Windows\Fonts\msyh.ttc", 19)
font_brand = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 18)
font_cat = ImageFont.truetype(r"C:\Windows\Fonts\msyh.ttc", 14)
font_meta = ImageFont.truetype(r"C:\Windows\Fonts\msyh.ttc", 14)
font_gap = ImageFont.truetype(r"C:\Windows\Fonts\msyhbd.ttc", 20)


def fit_text(text, font, max_width):
    draw = ImageDraw.Draw(Image.new("RGB", (1, 1)))
    if draw.textlength(text, font=font) <= max_width:
        return text
    while text and draw.textlength(text + "…", font=font) > max_width:
        text = text[:-1]
    return text + "…"


def checkerboard(w, h):
    tile = Image.new("RGB", (w, h), CHECK_A)
    d = ImageDraw.Draw(tile)
    step = 10
    for y in range(0, h, step):
        for x in range(0, w, step):
            if ((x // step) + (y // step)) % 2 == 0:
                d.rectangle((x, y, x + step - 1, y + step - 1), fill=CHECK_B)
    return tile


def load_logo(path):
    if path.suffix.lower() == ".svg":
        rendered = TMP / (path.stem + ".png")
        cairosvg.svg2png(url=str(path), write_to=str(rendered), output_width=900)
        im = Image.open(rendered).convert("RGBA")
    else:
        im = Image.open(path).convert("RGBA")
    im.thumbnail((TW - 30, TH - 24), Image.LANCZOS)
    return im


def main():
    rows = list(csv.DictReader(CSV_PATH.open(encoding="utf-8-sig")))
    report = json.loads(REPORT.read_text(encoding="utf-8"))
    by_key = {(r["layer"], r["brand"]): r for r in report}

    for layer in ["L1", "L2", "L3", "L4"]:
        items = [r for r in rows if r["layer"] == layer]
        n = len(items)
        rows_n = (n + COLS - 1) // COLS
        H = HEADER + rows_n * ROW_H + FOOTER
        img = Image.new("RGB", (W, H), BG)
        draw = ImageDraw.Draw(img)

        ok_count = sum(1 for r in items if by_key.get((r["layer"], r["brand"]), {}).get("ok"))
        draw.text((M, 34), LAYER_TITLES[layer], font=font_title, fill=NAVY)
        draw.text(
            (M, 80),
            f"{n} 个样本 · {ok_count} 个有图 · {n - ok_count} 个缺失 / 失败 · {LAYER_SUBS[layer]}",
            font=font_sub,
            fill=MUTED,
        )

        for i, row in enumerate(items):
            c, r = i % COLS, i // COLS
            x = M + c * (TW + GAP)
            y = HEADER + r * ROW_H
            info = by_key.get((row["layer"], row["brand"]), {})
            tile = checkerboard(TW, TH)
            td = ImageDraw.Draw(tile)

            file = info.get("file")
            if info.get("ok") and file and (BASE / file).exists():
                try:
                    logo = load_logo(BASE / file)
                    tile.paste(logo, ((TW - logo.width) // 2, (TH - logo.height) // 2), logo)
                except Exception as exc:
                    td.rectangle((0, 0, TW - 1, TH - 1), outline=LINE, width=2)
                    msg = "渲染失败"
                    tw = td.textlength(msg, font=font_gap)
                    td.text(((TW - tw) / 2, TH / 2 - 12), msg, font=font_gap, fill=MUTED)
                    _ = exc
            else:
                td.rectangle((0, 0, TW - 1, TH - 1), outline=LINE, width=2)
                msg = "未找到" if not info.get("url") else "下载失败"
                tw = td.textlength(msg, font=font_gap)
                td.text(((TW - tw) / 2, TH / 2 - 12), msg, font=font_gap, fill=MUTED)

            img.paste(tile, (x, y))
            draw.rectangle((x - 1, y - 1, x + TW, y + TH), outline=LINE, width=1)
            draw.text((x + 2, y + TH + 8), fit_text(row["brand"], font_brand, TW), font=font_brand, fill=NAVY)
            draw.text((x + 2, y + TH + 30), fit_text(row["category"], font_cat, TW), font=font_cat, fill=MUTED)

        draw.text(
            (M, H - FOOTER + 12),
            "研究用 contact sheet · logo 版权归各品牌所有 · 不用于描摹或生产",
            font=font_meta,
            fill=MUTED,
        )
        out = BASE / f"contact-{layer}.png"
        img.save(out, "PNG")
        print(out)


if __name__ == "__main__":
    main()
