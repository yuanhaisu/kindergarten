"""全路径覆盖断言：把每一站实际摆出来的取值并起来，对轴表查一遍。

为什么需要这个，而不是已有的 `coverage_check.py`：

    `coverage_check` 是**逐站**的。
    `typeset` 只查 mood，`matrix` 只查 mark_type，
    `construct` 只查「这一轮声明的取值都出现了」，**从不检查声明的比轴表少了多少**。

    于是站 4 声明 `geometry: [circle, square]` 也能通过——
    另外 6 个取值永远不会出现，而没有任何一条断言会响。

实测（2026-11-10）：47 个取值里出现 20 个，缺 27 个，其中 22 个没有申报理由。

三种状态，判定不同：

    出现     在站 1 / 站 3 / 站 4 的稿子里真的摆出来过
    申报     `axes.json → path_cut` 里写明为什么这一档不能用
    待站 4   站 4 还没收口，这一档可能出现在后续轮次——不是错误，是待办
    缺       站 4 已收口，却既没出现也没申报 → **退出码 1**

用法：
    python -I scripts/coverage_path.py --brand 养成星球          # 报告，不判死
    python -I scripts/coverage_path.py --brand 养成星球 --strict  # 收口后必须通过
    python -I scripts/coverage_path.py --no-run                  # 只看配置层的覆盖
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

SKILL = Path(__file__).resolve().parent.parent
AXES = SKILL / "axes.json"
RUNS = SKILL / "runs"


def load_spec() -> dict:
    return json.loads(AXES.read_text(encoding="utf-8"))


def axis_values(spec: dict, axis_id: str) -> list[str]:
    axis = next((a for a in spec["axes"] if a["id"] == axis_id), None)
    if axis is None:
        raise SystemExit(f"❌ 没有这条轴：{axis_id}")
    return [v["id"] for v in axis["values"]]


def cut_for(spec: dict, axis_id: str) -> dict[str, str]:
    """裁掉的取值及其理由。

    `mark_type` 的裁剪写在 `station3.cut`（站 3 自己要用），这里一并认。
    其余轴写在新加的 `path_cut`。两处都认，避免同一份理由维护两遍。
    """
    cut: dict[str, str] = {}
    if axis_id == "mark_type":
        cut.update(spec.get("station3", {}).get("cut") or {})
    cut.update((spec.get("path_cut") or {}).get(axis_id) or {})
    return cut


def load_rounds(run_dir: Path) -> list[dict]:
    """读站 4 每一轮的声明与稿子。`round-4.out.json` 是渲染产物，跳过。"""
    if not run_dir or not run_dir.exists():
        return []
    rounds = []
    for p in sorted(run_dir.glob("round-4*.json")):
        if ".out" in p.name:
            continue
        try:
            rounds.append(json.loads(p.read_text(encoding="utf-8")))
        except (json.JSONDecodeError, OSError) as e:
            print(f"⚠️  读不了 {p.name}：{e}", file=sys.stderr)
    return rounds


def station4_closed(spec: dict, run_dir: Path | None) -> bool:
    """站 4 是否已经收口——`picks.json` 里有 station 4 的 active 记录。

    没读完站 4 的品牌，缺的取值只是「还没摆」，不是「永远不会摆」。
    """
    if not run_dir:
        return False
    p = run_dir / "picks.json"
    if not p.exists():
        return False
    try:
        data = json.loads(p.read_text(encoding="utf-8"))
    except (json.JSONDecodeError, OSError):
        return False
    return any(h.get("station") == 4 and h.get("status") == "active"
               for h in data.get("history", []))


def collect(spec: dict, rounds: list[dict]) -> dict[str, dict[str, set[str]]]:
    """每个轴 → {取值: 出处集合}。只收**真的摆上桌**的东西，不收设计意图。"""
    seen: dict[str, dict[str, set[str]]] = {}

    def put(axis: str, value: str, where: str) -> None:
        seen.setdefault(axis, {}).setdefault(value, set()).add(where)

    for c in spec["zh"]["station1_cards"]:
        put("mood", c["mood"], "站1")
        # 卡片上的 type_style 是可选的：只有真的对应某一档时才标，不硬凑
        if c.get("type_style"):
            put("type_style", c["type_style"], "站1")

    for t in spec["station3"]["types"]:
        put("mark_type", t["id"], "站3")

    for rnd in rounds:
        tag = f"站4·{rnd.get('round', '?')}"
        for d in rnd.get("drafts") or []:
            for axis, value in (d.get("axes") or {}).items():
                put(axis, value, tag)

    return seen


def build_report(spec: dict, seen: dict, closed: bool) -> list[dict]:
    rows = []
    for axis in spec["axes"]:
        aid = axis["id"]
        declared = [v["id"] for v in axis["values"]]
        cut = cut_for(spec, aid)
        got = seen.get(aid, {})

        present = [v for v in declared if v in got]
        declared_cut = [v for v in declared if v not in got and v in cut]
        unknown = [v for v in got if v not in declared]
        rest = [v for v in declared if v not in got and v not in cut]

        # 站 4 收口了还缺 = 错误；没收口 = 待办
        missing = rest if closed else []
        pending = [] if closed else rest

        rows.append({
            "axis": aid, "declared": declared,
            "present": present, "cut": declared_cut,
            "pending": pending, "missing": missing, "unknown": unknown,
            "sources": got,
        })
    return rows


def print_report(rows: list[dict], spec: dict, closed: bool, brand: str) -> int:
    print(f"全路径覆盖 · {brand or '（未指定品牌）'}"
          f"　站4 {'已收口' if closed else '未收口（缺的只是待办）'}")
    print("─" * 78)
    print(f"{'轴':<14}{'取值':>4}{'出现':>5}{'申报':>5}{'待站4':>6}{'缺':>4}   说明")

    bad = 0
    for r in rows:
        flag = "✅" if not r["missing"] and not r["unknown"] else "❌"
        note = ""
        if r["unknown"]:
            note = f"未知取值：{', '.join(r['unknown'])}"
        elif r["missing"]:
            note = "既没出现也没申报"
        elif r["pending"]:
            note = f"待站4：{', '.join(r['pending'][:4])}" + ("…" if len(r["pending"]) > 4 else "")
        print(f"{r['axis']:<14}{len(r['declared']):>4}{len(r['present']):>5}"
              f"{len(r['cut']):>5}{len(r['pending']):>6}{len(r['missing']):>4}   {flag} {note}")
        if r["missing"] or r["unknown"]:
            bad += 1

    total_decl = sum(len(r["declared"]) for r in rows)
    total_got = sum(len(r["present"]) for r in rows)
    total_cut = sum(len(r["cut"]) for r in rows)
    total_pending = sum(len(r["pending"]) for r in rows)
    total_missing = sum(len(r["missing"]) for r in rows)
    print("─" * 78)
    print(f"合计 {total_decl} 个取值：出现 {total_got} · 申报裁掉 {total_cut}"
          f" · 待站4 {total_pending} · 缺 {total_missing}")

    if total_missing:
        print()
        print("❌ 站 4 已收口，但下面这些取值既没摆出来，也没申报理由：")
        for r in rows:
            for v in r["missing"]:
                print(f"   {r['axis']}.{v}")
        print()
        print("   要么在后续轮次里摆出来，要么在 axes.json → path_cut 写明为什么不能用。")
        print("   缺一个取值，正确答案就可能根本不在桌上——这条规则本来就是这么写的。")
        return 1

    if total_pending:
        print()
        print("ℹ️  站 4 还没收口，上面的「待站4」是待办，不是失败。")
        print("   收口后跑 --strict，它们必须变成「出现」或「申报」。")

    return 0


def unused_upstream(run_dir: Path) -> tuple[list[str], list[str]]:
    """站 3 的 active 格子里，哪些没有被任何一轮站 4 稿子用到。

    2026-11-11 站 4 建在 S07 上、而站 3 选的是 S01/S04，漂了两天没人发现。
    2026-11-13 同样的事又发生一次：站 3 的 S01 和 S04 都 active，
    五个轮次的 from 全是 S01，16 张稿里没有一张来自 S04。
    记录状态机管不了这个——它只维护「已记录的东西」。
    """
    p = run_dir / "picks.json"
    if not p.exists():
        return [], []
    data = json.loads(p.read_text(encoding="utf-8"))

    active = None
    for r in reversed(data.get("history", [])):
        if r.get("station") == 3 and r.get("status") == "active":
            active = r
            break
    if not active:
        return [], []

    cells = []
    for line in active.get("chosen") or []:
        m = re.match(r"(S\d+-[a-z]+|wordmark)", line)
        if m:
            cells.append(m.group(1))

    used = {r.get("from") for r in load_rounds(run_dir) if r.get("from")}
    return cells, [c for c in cells if c not in used]


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--brand", default="养成星球")
    ap.add_argument("--no-run", action="store_true",
                    help="不读任何 run 目录，只看配置层（站 1 + 站 3）")
    ap.add_argument("--strict", action="store_true",
                    help="有待办也判失败——站 4 收口后应当用这个")
    args = ap.parse_args()

    spec = load_spec()
    run_dir = None if args.no_run else RUNS / args.brand

    rounds = load_rounds(run_dir) if run_dir else []
    closed = station4_closed(spec, run_dir)
    if args.strict:
        closed = True

    seen = collect(spec, rounds)
    rows = build_report(spec, seen, closed)
    code = print_report(rows, spec, closed, args.brand)

    cells, unused = unused_upstream(run_dir) if run_dir else ([], [])
    if unused:
        print(f"\n⚠️  站 3 选了 {', '.join(cells)}，站 4 却没有一轮从 {', '.join(unused)} 出发")
        print("   用户挑 3 张，是在半个决策空间里挑。")
        code = code or 1

    return code


if __name__ == "__main__":
    sys.exit(main())
