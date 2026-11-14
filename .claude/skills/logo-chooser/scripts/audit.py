"""站 5 验收：全自动，不问用户任何问题。

只写「过 / 不过 + 原因」，**不写分数**——分数会让人讨论分数，而不是讨论问题。

六项里四项能测，两项测不了：

  1 16px 认得出主体        ✅ 实测（survive 的同一套：连通块 / 洞 / 墨占比）
  2 纯黑成立               ✅ 解析 fill / stroke，数颜色
  3 反白后结构不变         ✅ 渲到深底上，比连通块与洞
  4 无 text / 位图 / 滤镜  ✅ 解析标签与属性
  5 与 5 个同行并排最先被记住  ❌ 需要同行样本，人工
  6 一句话说得清是什么        ❌ 语义判断，人工

第 3 项只能测「结构有没有塌」。「反白看起来偏重」是光学错觉，像素数一样——
所以这里不假装能量出「该减细 5–8%」，那是构造阶段的判断。
"""

from __future__ import annotations

import argparse
import io
import json
import re
import sys
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

import cairosvg
from PIL import Image

SKILL = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))
import coverage_path  # noqa: E402
import picks  # noqa: E402

BOX = 96
DARK = (30, 41, 59)      # #1E293B，侧栏底色
VB = re.compile(r'viewBox="([-\d.]+)[ ,]+([-\d.]+)[ ,]+([-\d.]+)[ ,]+([-\d.]+)"')
BLACKISH = {"#000", "#000000", "black", "currentcolor", "none", ""}


def _grid(svg: str, box: int, bg: str, ink_dark: bool) -> list[list[bool]]:
    m = VB.search(svg)
    if not m:
        raise SystemExit("SVG 缺少 viewBox")
    _, _, w, h = (float(v) for v in m.groups())
    scale = box / max(w, h)
    ow, oh = max(1, round(w * scale)), max(1, round(h * scale))
    png = cairosvg.svg2png(bytestring=svg.encode("utf-8"),
                           output_width=ow, output_height=oh, background_color=bg)
    im = Image.open(io.BytesIO(png)).convert("L")
    canvas = Image.new("L", (box, box), 255 if ink_dark else 0)
    canvas.paste(im, ((box - ow) // 2, (box - oh) // 2))
    px = canvas.load()
    if ink_dark:
        return [[px[x, y] < 128 for x in range(box)] for y in range(box)]
    return [[px[x, y] > 128 for x in range(box)] for y in range(box)]


def _components(g, box):
    seen = [[False] * box for _ in range(box)]
    out = []
    for y in range(box):
        for x in range(box):
            if not g[y][x] or seen[y][x]:
                continue
            stack, n = [(y, x)], 0
            seen[y][x] = True
            while stack:
                cy, cx = stack.pop()
                n += 1
                for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    ny, nx = cy + dy, cx + dx
                    if 0 <= ny < box and 0 <= nx < box and g[ny][nx] and not seen[ny][nx]:
                        seen[ny][nx] = True
                        stack.append((ny, nx))
            out.append(n)
    total = sum(out)
    floor = max(2, total * 0.005)
    return sum(1 for n in out if n >= floor), total


def _holes(g, box):
    seen = [[False] * box for _ in range(box)]

    def flood(sy, sx):
        stack, n = [(sy, sx)], 0
        seen[sy][sx] = True
        while stack:
            cy, cx = stack.pop()
            n += 1
            for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                ny, nx = cy + dy, cx + dx
                if 0 <= ny < box and 0 <= nx < box and not g[ny][nx] and not seen[ny][nx]:
                    seen[ny][nx] = True
                    stack.append((ny, nx))
        return n

    for i in range(box):
        for y, x in ((0, i), (box - 1, i), (i, 0), (i, box - 1)):
            if not g[y][x] and not seen[y][x]:
                flood(y, x)
    count = 0
    for y in range(box):
        for x in range(box):
            if not g[y][x] and not seen[y][x] and flood(y, x) >= 2:
                count += 1
    return count


def _shape(svg: str, box: int, bg: str, ink_dark: bool):
    g = _grid(svg, box, bg, ink_dark)
    return {"components": _components(g, box)[0], "holes": _holes(g, box),
            "ink": _components(g, box)[1] / (box * box)}


def colors_in(svg: str) -> set[str]:
    found = set()
    for m in re.finditer(r'(?:fill|stroke|stop-color)\s*=\s*"([^"]+)"', svg):
        found.add(m.group(1).strip().lower())
    return found


def check_file(path: Path) -> list[tuple[str, str, str]]:
    """返回 [(状态, 检查项, 说明)]，状态为 过 / 不过 / 人工。"""
    svg = path.read_text(encoding="utf-8")
    rows: list[tuple[str, str, str]] = []

    # 1. 16px
    small = _shape(svg, 16, "white", True)
    big = _shape(svg, 96, "white", True)
    lost = []
    if small["components"] < big["components"]:
        lost.append(f"连通块 {big['components']} → {small['components']}")
    if small["holes"] != big["holes"]:
        lost.append(f"洞 {big['holes']} → {small['holes']}")
    if small["ink"] < 0.06:
        lost.append(f"墨只有 {small['ink']:.0%}")
    if not lost:
        rows.append(("过", "16px 认得出主体", f"{small['components']} 块 · {small['holes']} 洞 · 墨 {small['ink']:.0%}"))
    else:
        rows.append(("不过", "16px 认得出主体", "；".join(lost) + "——做简化版或删细节"))

    # 先算清两件事，再按规格顺序出条目
    cs = colors_in(svg) - BLACKISH
    bad = []
    if re.search(r"<text[\s>]", svg):
        bad.append("<text>")
    if re.search(r"<image[\s>]", svg):
        bad.append("<image>")
    if re.search(r"<filter[\s>]|\bfilter\s*=", svg):
        bad.append("filter")
    if re.search(r"<mask[\s>]", svg):
        bad.append("<mask>")

    # 2. 纯黑
    if not cs:
        rows.append(("过", "纯黑成立", "只有一种墨色"))
    else:
        rows.append(("不过", "纯黑成立", f"{len(cs)} 种颜色：{', '.join(sorted(cs))}——需要单色版"))

    # 3. 反白：只有干净的单色轮廓稿才验得了。彩色或含文字时，
    #    颜色替换本身就会失败，硬验会得出「结构变了」这种假结论。
    if cs or bad:
        why = []
        if cs:
            why.append("彩色")
        if bad:
            why.append("文字/位图")
        rows.append(("不过", "反白后结构不变", f"含{ '、'.join(why) }，先出单色轮廓版再验"))
    else:
        rev_ink = svg
        for a, b in (("#000000", "#FFFFFF"), ("#000", "#FFFFFF"),
                     ("black", "#FFFFFF"), ("currentColor", "#FFFFFF")):
            rev_ink = rev_ink.replace(a, b)
        rev = _shape(rev_ink, 96, "#1E293B", False)
        if rev["components"] == big["components"] and rev["holes"] == big["holes"]:
            rows.append(("过", "反白后结构不变", f"{rev['components']} 块 · {rev['holes']} 洞"))
        else:
            rows.append(("不过", "反白后结构不变",
                         f"正常 {big['components']} 块 {big['holes']} 洞，"
                         f"反白 {rev['components']} 块 {rev['holes']} 洞"))

    # 4. 禁用元素
    if not bad:
        rows.append(("过", "无 text / 位图 / 滤镜", "标签干净"))
    else:
        rows.append(("不过", "无 text / 位图 / 滤镜", "含 " + "、".join(bad) + "——必须轮廓化"))

    rows.append(("人工", "与 5 个同行并排最先被记住", "需要同行样本"))
    rows.append(("人工", "一句话说得清是什么", "语义判断"))
    return rows


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--brand", default="养成星球")
    ap.add_argument("--files", nargs="*", help="要验收的 SVG；默认读站 4 的选择")
    ap.add_argument("--out", type=Path)
    args = ap.parse_args()

    run_dir = SKILL / "runs" / args.brand
    if args.files:
        paths = [Path(f) for f in args.files]
    else:
        data = picks.load(run_dir / "picks.json")
        entry = picks.active(data, 4)
        if entry is None:
            print("❌ 没有有效的站 4 选择；用 --files 指定，或先跑 narrow.py --station 4")
            return 1
        paths = [run_dir / "drafts" / p["file"] for p in entry["picks"]]

    missing = [p for p in paths if not p.exists()]
    if missing:
        print("❌ 文件不存在：" + ", ".join(str(p) for p in missing))
        return 1

    lines = [f"# {args.brand} · 验收报告", "", "> 只写「过 / 不过 + 原因」。不写分数。", ""]
    any_fail = False

    for p in paths:
        rows = check_file(p)
        failed = [r for r in rows if r[0] == "不过"]
        any_fail = any_fail or bool(failed)
        head = "**全部通过**" if not failed else f"**{len(failed)} 项不过**"
        lines += [f"## {p.name} — {head}", ""]
        for status, name, detail in rows:
            lines.append(f"- `{status}` {name} —— {detail}")
        lines.append("")
        print(f"{p.name}: {'过' if not failed else '不过 %d 项' % len(failed)}")

    # 站 5 在站 4 收口之后跑，所以覆盖按 strict 判：缺一个就是不通过。
    # 用 --files 做单文件体检时不查——那是另一件事。
    if not args.files:
        spec = coverage_path.load_spec()
        cov = coverage_path.build_report(
            spec,
            coverage_path.collect(spec, coverage_path.load_rounds(run_dir)),
            True,
        )
        missing = [(r["axis"], v) for r in cov for v in r["missing"]]
        got = sum(len(r["present"]) for r in cov)
        cut = sum(len(r["cut"]) for r in cov)
        total = sum(len(r["declared"]) for r in cov)

        head = "**全部通过**" if not missing else f"**{len(missing)} 个取值不过**"
        lines += [f"## 全路径覆盖 — {head}", ""]
        lines.append(f"- 轴表 {total} 个取值：出现 {got} · 申报裁掉 {cut} · 缺 {len(missing)}")
        for axis, value in missing:
            lines.append(f"- `不过` {axis}.{value} —— 既没在路径上出现，也没在 `path_cut` 申报理由")
        lines.append("")
        any_fail = any_fail or bool(missing)
        print(f"全路径覆盖: {'过' if not missing else f'不过 {len(missing)} 个取值'}")

    # `--files` 是临时体检，不写回品牌目录。
    # 踩过：一次单文件复查把 runs/<品牌>/audit.md 冲成了只剩一行结果，
    # 而那份报告没法重新生成（站 4 没有记录，--files 是唯一入口）。
    if args.out is None and args.files:
        print()
        print("\n".join(lines).rstrip())
        return 1 if any_fail else 0

    out = args.out or (run_dir / "audit.md")
    out.write_text("\n".join(lines).rstrip() + "\n", encoding="utf-8")

    print(f"\n{out}")
    return 1 if any_fail else 0


if __name__ == "__main__":
    sys.exit(main())
