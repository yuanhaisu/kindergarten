"""开工前自检：每一站的原料齐不齐。

为什么要有这个：

    这套工具不能让人独立完成。站 2 的 15 句要人写，站 3 的示意要人画。
    如果不提前说，用户会在站 2 卡住，然后以为工具坏了。

它把「你需要提供什么」变成一张可执行的清单，而不是文档里的一句提醒。

用法：
    python -I scripts/preflight.py --brand 养成星球
    python -I scripts/preflight.py --brand 新品牌
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

SKILL = Path(__file__).resolve().parent.parent
AXES = SKILL / "axes.json"

sys.path.insert(0, str(Path(__file__).resolve().parent))
import draft  # noqa: E402
import picks  # noqa: E402
import coverage_path  # noqa: E402
from coverage_check import check  # noqa: E402


def mark(level: str) -> str:
    return {"ok": "✅", "warn": "⚠️ ", "block": "⛔", "todo": "▫️ "}[level]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--brand", default="养成星球")
    args = ap.parse_args()

    run_dir = SKILL / "runs" / args.brand
    spec = json.loads(AXES.read_text(encoding="utf-8"))
    rows: list[tuple[str, str, list[str]]] = []
    fatal = False

    # ---- 站 1 ----
    fonts = spec["zh"]["fonts"]
    missing_fonts = [f["zh"] for f in fonts if not Path(f["file"]).exists()]
    mood_cov = check(spec, "mood", [c["mood"] for c in spec["zh"]["station1_cards"]])
    if missing_fonts:
        rows.append(("站 1 气质", "block", [f"缺字体文件：{', '.join(missing_fonts)}"]))
        fatal = True
    elif mood_cov["undeclared"]:
        rows.append(("站 1 气质", "block", [f"覆盖缺口：{', '.join(mood_cov['undeclared'])}"]))
        fatal = True
    else:
        rows.append(("站 1 气质", "ok", [f"{len(spec['zh']['station1_cards'])} 张卡，8 个气质全覆盖"]))

    # ---- 站 2 ----
    words_path = run_dir / "words.json"
    sentences: list[dict] = []
    if not words_path.exists():
        rows.append(("站 2 意象", "block", [
            f"缺词库：runs/{args.brand}/words.json",
            "15 句「这是一个 ___」需要人写，脚本不生成",
        ]))
    else:
        bank = json.loads(words_path.read_text(encoding="utf-8"))
        sentences, failures = draft.validate(bank)
        if failures:
            rows.append(("站 2 意象", "block", [f"{len(failures)} 处不合格"] + failures[:4]))
        else:
            rows.append(("站 2 意象", "ok", [f"{len(sentences)} 句，校验通过"]))

    # ---- 站 3 ----
    if not sentences:
        rows.append(("站 3 类型", "todo", ["等站 2 有词库后再检查示意"]))
    else:
        # 站 3 只摆「只用图形」和「图形 + 名字」，每句一份示意就够。
        # 抽象版属于站 4 的构造轴，这里不再要求。
        missing = [s["id"] for s in sentences
                   if not (run_dir / "sketches" / f"{s['id']}.svg").exists()]
        if missing:
            rows.append(("站 3 类型", "block", [
                f"缺 {len(missing)} 张示意：{', '.join(missing)}",
                "示意要从句子变成形，需要人画（每句一份）",
            ]))
        else:
            rows.append(("站 3 类型", "ok", [f"{len(sentences)} 句都有示意"]))

    # ---- 站 4 ----
    specs = [p for p in sorted(run_dir.glob("round-4*.json"))
             if not p.name.endswith(".out.json")]
    if not specs:
        rows.append(("站 4 构造", "todo", ["还没开始——按轮次画黑稿，写 round-4.x.json"]))
    else:
        rows.append(("站 4 构造", "ok",
                     [f"{p.name}：{len(json.loads(p.read_text(encoding='utf-8')).get('drafts') or [])} 张稿"
                      for p in specs]))

    # ---- 全路径覆盖 ----
    # 逐站检查不等于路径检查：站 4 声明 geometry 只出两档也能过单站校验，
    # 但那一轮之外的值就永远不会出现。这里把它们并起来看。
    cp = coverage_path.build_report(
        spec,
        coverage_path.collect(spec, coverage_path.load_rounds(run_dir)),
        coverage_path.station4_closed(spec, run_dir),
    )
    cp_got = sum(len(r["present"]) for r in cp)
    cp_cut = sum(len(r["cut"]) for r in cp)
    cp_pend = sum(len(r["pending"]) for r in cp)
    cp_miss = sum(len(r["missing"]) for r in cp)
    if cp_miss:
        rows.append(("全路径覆盖", "block", [
            f"{cp_miss} 个取值既没出现也没申报",
            "跑 python -I scripts/coverage_path.py 看是哪几个",
        ]))
        fatal = True
    elif cp_pend:
        rows.append(("全路径覆盖", "todo", [
            f"出现 {cp_got} · 申报裁掉 {cp_cut} · 站 4 还要摆 {cp_pend} 个取值",
            "收口前用 --strict：缺一个就过不去",
        ]))
    else:
        rows.append(("全路径覆盖", "ok", [f"出现 {cp_got} · 申报裁掉 {cp_cut}"]))

    # ---- 站 5 ----
    data = picks.load(run_dir / "picks.json")
    if picks.active(data, 4):
        rows.append(("站 5 验收", "ok", ["可跑：python -I scripts/audit.py"]))
    else:
        rows.append(("站 5 验收", "todo", ["等站 4 有选择"]))

    # ---- 输出 ----
    print(f"\n{args.brand} · 开工自检\n")
    for name, level, details in rows:
        print(f"  {mark(level)} {name}")
        for d in details:
            print(f"       {d}")

    blocked = [r for r in rows if r[1] == "block"]
    print()
    if blocked:
        print("  不能完整跑通。你需要补的是：")
        print("    · 站 2 的 15 句意象（人写）")
        print("    · 站 3 的示意图（人画，每句 1–2 张）")
        print("  这些是创作，不是工具能生成的——但每一个判断仍然由你做。")
    else:
        print("  原料齐全。开始：")
        print(f"    python -I scripts/typeset.py --brand {args.brand}")

    return 1 if fatal else 0


if __name__ == "__main__":
    sys.exit(main())
