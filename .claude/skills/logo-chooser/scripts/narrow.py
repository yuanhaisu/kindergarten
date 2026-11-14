"""收敛与记账：把「指认结果」变成可复现的记录。

两站共用一套记账，但校验规则不同：

    站 1 气质  选 2–3 张，不能全在同一（字族 × 重量档）
    站 2 意象  选 3 句，且必须来自至少 2 个不同的词

两条规则的共同点：**都在防「其实只有一个想法」。**
站 1 的三张全挑重的，站 2 的三句全出自同一个词，本质是同一种塌缩。

> 原文档里站 1 写「必须来自 2 个不同 mood 分支」、站 2 写「必须是 3 个不同物件」，
> 两条都是空转的：站 1 每张卡本来就是一个 mood，站 2 每个物件本来就不重名。
> 规则必须能真的拦下东西，否则等于没有。

用法：
    python -I scripts/narrow.py --pick 04,05 --note "沉静像园长看的样子"
    python -I scripts/narrow.py --station 2 --pick S01,S07,S12

产物：
    runs/<品牌>/picks.json   机读，保留全部历史
    runs/<品牌>/brief.md     人读，追加式，回退不丢上下文
"""

from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

SKILL = Path(__file__).resolve().parent.parent
AXES = SKILL / "axes.json"

sys.path.insert(0, str(Path(__file__).resolve().parent))
import picks  # noqa: E402
from coverage_check import require  # noqa: E402



def family(card: dict) -> str:
    return "serif" if "serif" in card["font"] else "sans"


def make_tier(tiers: list[dict]):
    """重量档阈值只存在于 axes.json，不在脚本里硬编码。"""
    def tier(wght: float) -> str:
        for t in tiers:
            if wght >= t.get("min", float("-inf")) and wght <= t.get("max", float("inf")):
                return t["id"]
        return tiers[-1]["id"]
    return tier


def split_ids(raw: str) -> list[str]:
    return [x for x in raw.replace("，", ",").replace(" ", "").split(",") if x]


def record(run_dir: Path, brand: str, station: int, pick_rows: list[dict],
           note: str | None, candidates: str, chosen: list[str],
           rejected: list[str], locked: str,
           exception: str | None = None) -> int:
    now = datetime.now()
    path = run_dir / "picks.json"

    data = picks.load(path)
    data["brand"] = brand
    picks.record(
        data, station, at=now, candidates=candidates, picks=pick_rows,
        chosen=chosen, rejected=rejected, locked=locked, note=note,
        exception=exception,
    )
    picks.save(path, data)

    attempt = sum(1 for h in data["history"] if h["station"] == station)
    brief = run_dir / "brief.md"
    brief.write_text(picks.render_brief(data), encoding="utf-8")

    print(f"✅ 站 {station} 第 {attempt} 次选择已记录")
    for line in chosen:
        print("   " + line)

    dirtied = [h for h in data["history"]
               if h["status"] == "stale" and h.get("stale_reason", "").endswith(
                   f"站 {station} 于 {now.strftime('%Y-%m-%d %H:%M')} 重新选择")]
    if dirtied:
        names = "、".join(sorted({f"站 {h['station']}" for h in dirtied}))
        print(f"\n⚠️  {names} 的记录已标记为过期（上游改了），需要重做")

    print(f"\n   {path}")
    print(f"   {brief}")
    return 0


def station1(args, run_dir: Path) -> int:
    spec = json.loads(AXES.read_text(encoding="utf-8"))
    cards = spec["zh"]["station1_cards"]
    total = len(cards)
    tier = make_tier(spec["zh"]["weight_tiers"])
    tier_zh = {t["id"]: t["zh"] for t in spec["zh"]["weight_tiers"]}

    try:
        ids = [int(x) for x in split_ids(args.pick)]
    except ValueError:
        print(f"❌ 编号必须是数字：{args.pick}")
        return 1

    if len(set(ids)) != len(ids):
        print("❌ 有重复编号")
        return 1
    bad = [i for i in ids if not 1 <= i <= total]
    if bad:
        print(f"❌ 编号超出范围（1–{total}）：{bad}")
        return 1
    if not 2 <= len(ids) <= 3:
        print(f"❌ 站 1 必须选 2–3 张，你选了 {len(ids)} 张")
        return 1

    ids.sort()
    picked = [(i, cards[i - 1]) for i in ids]

    combos = {(family(c), tier(c["wght"])) for _, c in picked}
    if len(combos) == 1:
        fam, t = next(iter(combos))
        print("❌ 这几张是同一个分量，等于只选了一次。")
        print(f"   {fam} / {tier_zh[t]}：{'、'.join(c['zh'] for _, c in picked)}")
        print("   追问用户：如果只能留这一张，你愿意再接受一张什么分量的？")
        return 1

    chosen = [
        f"{i:02d} {c['zh']} · wght {c['wght']} · 字距 {c.get('tracking', 0):+.0%}"
        for i, c in picked
    ]
    rejected = [f"{i:02d}" for i in range(1, total + 1) if i not in ids]
    locked = (f"mood = {', '.join(c['mood'] for _, c in picked)}；"
              f"type_style = {', '.join(sorted({family(c) for _, c in picked}))}")

    return record(
        run_dir, args.brand, 1,
        pick_rows=[{
            "id": f"{i:02d}", "mood": c["mood"], "zh": c["zh"],
            "font": c["font"], "wght": c["wght"],
            "tracking": c.get("tracking", 0.0),
            "family": family(c), "tier": tier(c["wght"]),
        } for i, c in picked],
        note=args.note,
        candidates=f"01–{total:02d}（`runs/{args.brand}/round-1.html`）",
        chosen=chosen, rejected=rejected, locked=locked,
    )


def station2(args, run_dir: Path) -> int:
    src = run_dir / "round-2.json"
    if not src.exists():
        print(f"❌ 找不到 {src}，先跑 scripts/draft.py")
        return 1

    bank = json.loads(src.read_text(encoding="utf-8"))
    sentences = {s["id"]: s for s in bank["sentences"]}
    ids = split_ids(args.pick)

    unknown = [i for i in ids if i not in sentences]
    if unknown:
        print(f"❌ 没有这些编号：{unknown}")
        return 1
    if len(set(ids)) != len(ids):
        print("❌ 有重复编号")
        return 1
    if len(ids) != 3:
        print(f"❌ 站 2 必须选 3 句，你选了 {len(ids)} 句")
        return 1

    words = {sentences[i]["word"] for i in ids}
    if len(words) == 1:
        w = next(iter(words))
        print("❌ 三句都在同一个词下，这其实是同一个想法。")
        print(f"   「{w}」：{' / '.join(sentences[i]['object'] for i in ids)}")
        print("   追问用户：如果只能留这一句，你愿意再从别的词里挑一句吗？")
        return 1

    ids.sort()
    chosen = [f"{i} {sentences[i]['sentence']}" for i in ids]
    rejected = [i for i in sentences if i not in ids]
    locked = (f"words = {', '.join(sorted(words))}；"
              f"objects = {', '.join(sentences[i]['object'] for i in ids)}")

    return record(
        run_dir, args.brand, 2,
        pick_rows=[{
            "id": i, "word": sentences[i]["word"], "object": sentences[i]["object"],
            "sentence": sentences[i]["sentence"],
            "shape": sentences[i]["shape"],
            "relation": sentences[i]["relation"],
            "join": sentences[i]["join"],
        } for i in ids],
        note=args.note,
        candidates=f"{len(sentences)} 句（`runs/{args.brand}/round-2.html`）",
        chosen=chosen, rejected=rejected, locked=locked,
    )


def station3(args, run_dir: Path) -> int:
    matrix_path = run_dir / "round-3.json"
    if not matrix_path.exists():
        print(f"❌ 找不到 {matrix_path}，先跑 scripts/matrix.py")
        return 1
    matrix = json.loads(matrix_path.read_text(encoding="utf-8"))
    by_id = {s["id"]: s for s in matrix["sentences"]}
    n_sent = len(by_id)
    variant = matrix.get("variant", "full")

    spec = json.loads(AXES.read_text(encoding="utf-8"))
    require(spec, "mark_type", [t["id"] for t in spec["station3"]["types"]],
            allow_cut=spec["station3"].get("cut"), where="站 3 · 类型")
    types = {t["id"]: t for t in spec["station3"]["types"]}

    ids = split_ids(args.pick)
    parsed: list[tuple[str, str]] = []
    for cid in ids:
        # 全局卡：字标不绑句子，编号就是类型本身
        if cid in types:
            if types[cid]["symbol"]:
                print(f"❌ {cid} 必须带句子编号（如 S01-{cid}）")
                return 1
            parsed.append(("", cid))
            continue
        sid, _, t = cid.rpartition("-")
        if not sid or sid not in by_id:
            print(f"❌ 不认识的句子编号：{cid}")
            return 1
        if t not in types:
            print(f"❌ 不认识的类型：{t}（可用：{', '.join(types)}）")
            return 1
        if not types[t]["symbol"]:
            print(f"❌ {t} 不绑句子，直接写 {t} 即可")
            return 1
        parsed.append((sid, t))

    if len(set(parsed)) != len(parsed):
        print("❌ 有重复的格子")
        return 1
    if not 2 <= len(parsed) <= 3:
        print(f"❌ 站 3 必须选 2–3 格，你选了 {len(parsed)} 格")
        return 1

    picked_types = {t for _, t in parsed}
    exception = None
    if len(picked_types) == 1:
        t = next(iter(picked_types))
        reason = (args.allow_single_type or "").strip()
        if not reason:
            print("❌ 都在同一种类型里，那样没有在比较类型。")
            print(f"   {types[t]['zh']}：{'、'.join(sid for sid, _ in parsed)}")
            print("   追问用户：如果只能留这一格，你愿意换一种类型再看一次吗？")
            print("   如果「同一种类型」本身就是答案，用 --allow-single-type 写清理由再记。")
            return 1
        exception = f"只有一种类型（{types[t]['zh']}）：{reason}"
        print(f"⚠️  破例记录：{exception}")

    def cell_id(sid: str, t: str) -> str:
        return f"{sid}-{t}" if sid else t

    def sentence_of(sid: str) -> str:
        return by_id[sid]["sentence"] if sid else "（只看名字，不绑任何想法）"

    # 字标不绑句子，页面上只有一张全局卡。格子数必须按渲染器的实际布局算：
    # matrix.py 每行只出 symbol 列，再单独加一张全局卡，不是 n_sent * n_types。
    n_symbol = sum(1 for t in types.values() if t["symbol"])
    n_global = len(types) - n_symbol
    n_cells = n_symbol * n_sent + n_global

    chosen = [f"{cell_id(sid, t)} {types[t]['zh']} · {sentence_of(sid)}"
              for sid, t in parsed]
    locked = (f"variant = {variant}；"
              f"mark_type = {', '.join(sorted(picked_types))}；"
              f"cells = {', '.join(cell_id(s, t) for s, t in parsed)}")

    return record(
        run_dir, args.brand, 3,
        pick_rows=[{
            "id": cell_id(sid, t), "sentence_id": sid or None, "type": t,
            "type_zh": types[t]["zh"],
            "sentence": sentence_of(sid),
            "object": by_id[sid]["object"] if sid else None,
        } for sid, t in parsed],
        note=args.note,
        exception=exception,
        candidates=(f"{n_cells} 格 = {n_sent} 句 × {n_symbol} 类型"
                    f" + {n_global} 张全局卡（`runs/{args.brand}/round-3.html`）"),
        chosen=chosen,
        rejected=[f"其余 {n_cells - len(parsed)} 格"],
        locked=locked,
    )


def do_restore(args, run_dir: Path) -> int:
    path = run_dir / "picks.json"
    if not path.exists():
        print("❌ 还没有任何选择记录，没有可回退的版本")
        return 1

    data = picks.load(path)
    data["brand"] = args.brand
    res = picks.restore(data, args.station, args.restore, datetime.now())

    if not res["ok"]:
        print(f"❌ {res['error']}")
        return 1

    if not res["changed"]:
        print(f"ℹ️  站 {args.station} 第 {args.restore} 次本来就是当前选择，没有变化")
        return 0

    picks.save(path, data)
    (run_dir / "brief.md").write_text(picks.render_brief(data), encoding="utf-8")

    print(f"↩ 已回退：站 {args.station} 第 {args.restore} 次")
    for line in res["target"].get("chosen", []):
        print("   " + line)
    if res["dirty"]:
        names = "、".join(f"站 {s}" for s in res["dirty"])
        print(f"\n⚠️  {names} 的记录已标记为过期（上游回退了），需要重做")
    print(f"\n   {path}")
    print(f"   {run_dir / 'brief.md'}")
    return 0


def station4(args, run_dir: Path) -> int:
    # 每一轮有自己的 out 文件。旧版所有轮次共写 round-4.out.json 且互相覆盖，
    # 于是只有最后一轮能被选——前面几轮的稿子还在磁盘上，但入口没了。
    if args.round:
        src = run_dir / f"round-4.{args.round}.out.json"
        if not src.exists():
            print(f"❌ 找不到 {src}，先跑 scripts/construct.py --round {args.round}")
            return 1
    else:
        outs = sorted(run_dir.glob("round-4.*.out.json"))
        if not outs:
            print("❌ 找不到任何 round-4.*.out.json，先跑 scripts/construct.py")
            return 1
        src = outs[-1]

    rnd = json.loads(src.read_text(encoding="utf-8"))
    drafts = {d["id"]: d for d in rnd["drafts"]}
    ids = split_ids(args.pick)
    total = len(drafts)

    unknown = [i for i in ids if i not in drafts]
    if unknown:
        print(f"❌ {src.name} 里没有这些稿子：{unknown}（这轮是 {sorted(drafts)}）")
        return 1
    if len(set(ids)) != len(ids):
        print("❌ 有重复编号")
        return 1

    # 相对规则：留 ≤40% 且 ≤3，至少 2 张。
    # 旧规则是「必须恰好 3 张」，于是只有 2 张的轮次在数学上永远收不了口——
    # 站 4 卡住不是没人选，是选不了。
    cap = max(2, min(3, round(total * 0.4)))
    if not 2 <= len(ids) <= cap:
        print(f"❌ {src.name} 有 {total} 张，要留 2–{cap} 张（不超过 40%），你选了 {len(ids)} 张")
        return 1
    # 不合格的稿子不能被选。它的问题已经量出来了——形状不是自称的那个、
    # 两块粘住、间隙不够——让用户选它，等于让他选一个已知有问题的东西。
    bad_picks = [i for i in ids if i in rnd.get("failed", [])]
    if bad_picks:
        print(f"❌ 这些稿子没过硬检查，不能选：{'、'.join(bad_picks)}")
        for row in rnd.get("failed_detail") or []:
            if row["id"] in bad_picks:
                print(f"   {row['id']}：" + "；".join(row["reasons"]))
        print("   要么修好再选，要么这一轮作废重画。")
        return 1

    # 不做多样性校验：每张稿子的坐标组合本来就是唯一的，
    # 任何「至少两种取值」的规则都会自动成立。写上去只是装饰。

    chosen = [f"{i} {drafts[i]['axes']}" for i in ids]
    locked = (f"round = {rnd.get('round')}；"
              f"vary = {', '.join(rnd.get('vary') or {})}；"
              f"drafts = {', '.join(ids)}")

    return record(
        run_dir, args.brand, 4,
        pick_rows=[{
            "id": i, "file": drafts[i]["file"], "axes": drafts[i]["axes"],
            "note": drafts[i].get("note"), "failed": i in rnd.get("failed", []),
        } for i in ids],
        note=args.note,
        candidates=f"{len(drafts)} 张（`runs/{args.brand}/round-4-{rnd.get('round')}.html`）",
        chosen=chosen,
        rejected=[f"其余 {len(drafts) - len(ids)} 张"],
        locked=locked,
    )


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--brand", default="养成星球")
    ap.add_argument("--station", type=int, choices=[1, 2, 3, 4], default=1)
    ap.add_argument("--pick", help="卡片/格子编号，逗号分隔")
    ap.add_argument("--restore", type=int, metavar="N",
                    help="把该站第 N 次的选择恢复为当前（不重新选）")
    ap.add_argument("--note")
    ap.add_argument("--round", metavar="R",
                    help="站 4 指定从哪一轮收敛（默认取最新一轮）；如 --round 4.2")
    ap.add_argument("--allow-single-type", metavar="理由",
                    help="站 3 允许所有格子同类型，但必须写理由（会记进 brief）")
    args = ap.parse_args()

    run_dir = SKILL / "runs" / args.brand
    run_dir.mkdir(parents=True, exist_ok=True)

    if args.restore is not None:
        return do_restore(args, run_dir)
    if not args.pick:
        print("❌ 用 --pick 做选择，或用 --restore N 回退")
        return 1

    return {1: station1, 2: station2, 3: station3, 4: station4}[args.station](args, run_dir)


if __name__ == "__main__":
    sys.exit(main())
