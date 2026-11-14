"""按 filter 从库里筛出候选 exemplar，替代手挑。

轴表的 filter 是可判定的谓词；exemplar 就应该由它推导出来，
而不是凭印象填。手挑的 exemplar 会和 filter 打架——这正是
validate_axes.py 报出的那 13 处。

排序偏好（构造学习的角度，不是审美排序）：
  1. exemplary=1 优先
  2. 无渐变、无位图优先（能看清几何）
  3. 颜色数少的优先
  4. 方形画幅优先（符号形；字标类反过来偏好横排）

用法：
    python -I scripts/suggest_exemplars.py                 # 全部取值，各列 5 个候选
    python -I scripts/suggest_exemplars.py --axis mood     # 只看一条轴
    python -I scripts/suggest_exemplars.py --json          # 机器可读
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

FIELD = {
    "type": "mark_type",
    "mood": "mood",
    "type_style": "type_style",
    "geometry": "geometry",
    "technique": "techniques",
    "subject": "subject",
}
SET_FIELDS = {"mood", "geometry", "techniques"}


def matches(entry: dict, flt: dict) -> bool:
    for key, want in flt.items():
        if key == "has_subject":
            if bool((entry.get("subject") or "").strip()) != bool(want):
                return False
            continue
        if key == "technique_not":
            if set(entry.get("techniques") or []) & set(want):
                return False
            continue

        field = FIELD.get(key, key)
        actual = entry.get(field)

        if key == "type":
            allowed = {s.strip() for s in str(want).split(",")}
            if actual not in allowed:
                return False
        elif field in SET_FIELDS:
            if want not in set(actual or []):
                return False
        elif actual != want:
            return False
    return True


def score(entry: dict) -> float:
    s = 0.0
    if entry.get("exemplary"):
        s += 100
    if not entry.get("gradients"):
        s += 20
    if not entry.get("has_mask") and not entry.get("has_filter"):
        s += 10
    s += max(0, 8 - (entry.get("n_colors") or 8)) * 2
    # 符号类偏好方形；字标类偏好横向
    aspect = entry.get("aspect_class")
    if entry.get("mark_type") in {"wordmark", "lettermark", "combination"}:
        s += 8 if aspect == "horizontal" else 0
    else:
        s += 8 if aspect in {"square", "vertical"} else 0
    # 细节越少越容易学构造
    s -= min((entry.get("anchors") or 0) / 40.0, 20)
    return s


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--axis")
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--limit", type=int, default=5)
    args = ap.parse_args()

    spec = json.loads(AXES.read_text(encoding="utf-8"))
    catalog = json.loads(Path(spec["library"]["catalog"]).read_text(encoding="utf-8"))
    have = {e["file"] for e in catalog}

    out: dict[str, list[str]] = {}

    for axis in spec["axes"]:
        if args.axis and axis["id"] != args.axis:
            continue
        for val in axis["values"]:
            flt = val.get("filter") or {}
            pool = [e for e in catalog if matches(e, flt)]
            pool.sort(key=score, reverse=True)
            names = [e["file"] for e in pool if e["file"] in have][: args.limit]
            out[f"{axis['id']}.{val['id']}"] = names

            if args.json:
                continue

            print(f"\n## {axis['id']}.{val['id']}  （命中 {len(pool)}）")
            cur = val.get("exemplars") or []
            for n in names:
                mark = "  " if n in cur else "→ "
                print(f" {mark}{n}")
            stale = [c for c in cur if c not in names]
            if stale:
                print(f"   （现表里不在候选前列：{', '.join(stale)}）")

    if args.json:
        print(json.dumps(out, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
