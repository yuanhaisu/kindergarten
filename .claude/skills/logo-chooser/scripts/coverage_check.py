"""发散覆盖检查：每条参与发散的轴，每个取值至少出现一次。

这条规则原本只写在 `axes.json → divergence_rules.coverage` 里，没有任何代码执行它。
它防的是最致命的一种失败：

    某个取值因为「看起来不完整」被默认跳过，于是正确答案根本不在选项里。

A1 就是 `figure_ground=line`。只出实心版的话，细线永远进不了候选——不是它不好，
是它根本没被摆上桌。

规则不是文档，是断言。缺一个取值就退出，不给「先跑跑看」的余地。

用法：
    python -I scripts/coverage_check.py --axis mood          # 用配置里的实际候选
    python -I scripts/coverage_check.py --axis mark_type
    python -I scripts/coverage_check.py --axis figure_ground --present line,solid
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


def load_spec() -> dict:
    return json.loads(AXES.read_text(encoding="utf-8"))


def axis_values(spec: dict, axis_id: str) -> list[str]:
    axis = next((a for a in spec["axes"] if a["id"] == axis_id), None)
    if axis is None:
        raise SystemExit(f"❌ 没有这条轴：{axis_id}（可用：{', '.join(a['id'] for a in spec['axes'])}）")
    return [v["id"] for v in axis["values"]]


def default_present(spec: dict, axis_id: str) -> list[str] | None:
    """每个站的实际候选来自哪里——不靠调用方重复声明。"""
    if axis_id == "mood":
        return [c["mood"] for c in spec["zh"]["station1_cards"]]
    if axis_id == "mark_type":
        return [t["id"] for t in spec["station3"]["types"]]
    return None


def default_cut(spec: dict, axis_id: str) -> dict:
    if axis_id == "mark_type":
        return spec["station3"].get("cut") or {}
    return {}


def check(spec: dict, axis_id: str, present: list[str],
          allow_cut: dict | None = None) -> dict:
    declared = axis_values(spec, axis_id)
    allow_cut = allow_cut or {}
    seen = list(dict.fromkeys(present))

    dupes = sorted({v for v in seen if present.count(v) > 1})
    extra = [v for v in seen if v not in declared]
    missing = [v for v in declared if v not in seen]
    undeclared = [v for v in missing if v not in allow_cut]

    return {
        "axis": axis_id, "declared": declared, "present": seen,
        "missing": missing, "undeclared": undeclared, "extra": extra, "dupes": dupes,
    }


def require(spec: dict, axis_id: str, present: list[str],
            allow_cut: dict | None = None, where: str = "") -> None:
    """不通过就退出。渲染器在出图前调用，保证候选集没有偷偷少一档。"""
    r = check(spec, axis_id, present, allow_cut)
    if not (r["undeclared"] or r["extra"] or r["dupes"]):
        return

    tag = f"{where} " if where else ""
    print(f"❌ {tag}覆盖检查失败（轴 {axis_id}）", file=sys.stderr)
    if r["undeclared"]:
        print(f"   缺少取值：{', '.join(r['undeclared'])}", file=sys.stderr)
        print("   这些取值没出现在候选里，正确答案可能因此进不了选项。", file=sys.stderr)
        print("   要么补上，要么在配置的 allow_cut 里申报理由。", file=sys.stderr)
    if r["dupes"]:
        print(f"   重复取值：{', '.join(r['dupes'])} —— 重复会挤掉别的取值的位置", file=sys.stderr)
    if r["extra"]:
        print(f"   未知取值：{', '.join(r['extra'])}", file=sys.stderr)
    raise SystemExit(1)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--axis", required=True)
    ap.add_argument("--present", help="当前候选的取值，逗号分隔；缺省时读配置里的实际候选")
    ap.add_argument("--cut", help="已申报裁掉的取值，逗号分隔；缺省时读配置")
    args = ap.parse_args()

    spec = load_spec()
    declared = axis_values(spec, args.axis)

    if args.present:
        present = [x.strip() for x in args.present.replace("，", ",").split(",") if x.strip()]
    else:
        present = default_present(spec, args.axis)
        if present is None:
            print(f"❌ 轴 {args.axis} 没有默认候选来源，请用 --present 指定")
            return 1

    if args.cut is not None:
        cut = {k: "" for k in args.cut.replace("，", ",").split(",") if k.strip()}
    else:
        cut = default_cut(spec, args.axis)

    r = check(spec, args.axis, present, cut)

    print(f"轴 {args.axis}：声明 {len(declared)} 个取值，候选里出现 {len(r['present'])} 个")
    if r["missing"]:
        cut_names = [v for v in r["missing"] if v in cut]
        if cut_names:
            print("  已申报裁掉：" + "、".join(f"{v}（{cut[v] or '无理由'}）" for v in cut_names))
    if r["undeclared"]:
        print("  ❌ 未申报的缺失：" + ", ".join(r["undeclared"]))
        return 1
    if r["dupes"]:
        print("  ❌ 重复：" + ", ".join(r["dupes"]))
        return 1
    if r["extra"]:
        print("  ❌ 未知取值：" + ", ".join(r["extra"]))
        return 1

    print("  ✅ 覆盖完整")
    return 0


if __name__ == "__main__":
    sys.exit(main())
