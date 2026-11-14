import csv
import json
from pathlib import Path

import cairosvg
from PIL import Image, ImageDraw, ImageFont

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\docs\research\2026-11-02-keyword-logos")
CSV_PATH = BASE.parent / "2026-11-02-keyword-logos.csv"
REPORT = BASE / "download-report.json"
TMP = BASE / "render-tmp"
TMP.mkdir(parents=True, exist_ok=True)

TITLES = {
    "星球": "星球 · 语义参考",
    "养成": "养成 · 语义参考",
    "成长": "成长 · 语义参考",
}
SUBS = {
    "星球": "15 个产品 / 项目 · 名称含「星球」或以星球 / 宇宙为核心符号",
    "养成": "15 个产品 / 项目 · 养成游戏 / 虚拟宠物 / 陪伴 / 习惯成瘾",
    "成长": "15 个产品 / 项目 · 成长记录 / 里程碑 / 儿童与个人成长",
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
    by_key = {(r["keyword"], r["brand"]): r for r in report}

    for keyword in ["星球", "养成", "成长"]:
        items = [r for r in rows if r["keyword"] == keyword]
        n = len(items)
        rows_n = (n + COLS - 1) // COLS
        H = HEADER + rows_n * ROW_H + FOOTER
        img = Image.new("RGB", (W, H), BG)
        draw = ImageDraw.Draw(img)

        ok_count = sum(1 for r in items if by_key.get((r["keyword"], r["brand"]), {}).get("ok"))
        draw.text((M, 34), TITLES[keyword], font=font_title, fill=NAVY)
        draw.text((M, 80), f"{n} 个 · {ok_count} 个有图 · {SUBS[keyword]}", font=font_sub, fill=MUTED)

        for i, row in enumerate(items):
            c, r = i % COLS, i // COLS
            x = M + c * (TW + GAP)
            y = HEADER + r * ROW_H
            info = by_key.get((row["keyword"], row["brand"]), {})
            tile = checkerboard(TW, TH)
            td = ImageDraw.Draw(tile)

            file = info.get("file")
            if info.get("ok") and file and (BASE / file).exists():
                try:
                    logo = load_logo(BASE / file)
                    tile.paste(logo, ((TW - logo.width) // 2, (TH - logo.height) // 2), logo)
                except Exception:
                    td.rectangle((0, 0, TW - 1, TH - 1), outline=LINE, width=2)
                    msg = "渲染失败"
                    tw = td.textlength(msg, font=font_gap)
                    td.text(((TW - tw) / 2, TH / 2 - 12), msg, font=font_gap, fill=MUTED)
            else:
                td.rectangle((0, 0, TW - 1, TH - 1), outline=LINE, width=2)
                msg = "未找到" if not info.get("url") else "下载失败"
                tw = td.textlength(msg, font=font_gap)
                td.text(((TW - tw) / 2, TH / 2 - 12), msg, font=font_gap, fill=MUTED)

            img.paste(tile, (x, y))
            draw.rectangle((x - 1, y - 1, x + TW, y + TH), outline=LINE, width=1)
            draw.text((x + 2, y + TH + 8), fit_text(row["brand"], font_brand, TW), font=font_brand, fill=NAVY)
            label = f"{row['product_type']} · {row['market']}"
            draw.text((x + 2, y + TH + 30), fit_text(label, font_cat, TW), font=font_cat, fill=MUTED)

        draw.text(
            (M, H - FOOTER + 12),
            "语义参考 contact sheet · logo 版权归各品牌所有 · 不用于描摹或生产",
            font=font_meta,
            fill=MUTED,
        )
        out = BASE / f"contact-{keyword}.png"
        img.save(out, "PNG")
        print(out)


if __name__ == "__main__":
    main()
