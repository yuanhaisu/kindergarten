"""校验 axes.json 的每一条取值都能被真实作品指认。

一条轴取值只有在同时满足下面三点时才算成立：
  1. exemplar 文件真实存在
  2. exemplar 在 catalog 里有元数据
  3. 元数据与 filter 声明一致

第 3 条是重点：轴表里的 filter 是"谓词描述"，不是注册表。
如果 exemplar 自己都不满足这个谓词，这条取值就是编的。

用法：
    python -I scripts/validate_axes.py [--fix-report out.md]
退出码 0 = 全部通过；1 = 存在不成立取值。
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

# Windows 控制台默认 GBK，中文与符号会直接抛 UnicodeEncodeError
for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

SKILL = Path(__file__).resolve().parent.parent
AXES = SKILL / "axes.json"

# axes.json 的 filter 键 → catalog 字段
FIELD = {
    "type": "mark_type",
    "mood": "mood",
    "type_style": "type_style",
    "geometry": "geometry",
    "technique": "techniques",
    "subject": "subject",
}

# 值是集合而不是标量的字段
SET_FIELDS = {"mood", "geometry", "techniques"}


def _get(entry: dict, key: str):
    return entry.get(FIELD.get(key, key))


def matches(entry: dict, flt: dict) -> list[str]:
    """返回不满足的条目；空列表 = 通过。"""
    bad: list[str] = []

    for key, want in flt.items():
        if key == "has_subject":
            has = bool((entry.get("subject") or "").strip())
            if has != bool(want):
                bad.append(f"has_subject={want} 实际={has}")
            continue

        if key == "technique_not":
            techs = set(entry.get("techniques") or [])
            hit = techs & set(want)
            if hit:
                bad.append(f"technique_not={want} 实际含有 {sorted(hit)}")
            continue

        field = FIELD.get(key, key)
        actual = entry.get(field)

        if key == "type":
            allowed = {s.strip() for s in str(want).split(",")}
            if actual not in allowed:
                bad.append(f"{key}={want} 实际={actual}")
            continue

        if field in SET_FIELDS:
            actual_set = set(actual or [])
            if want not in actual_set:
                bad.append(f"{key}={want} 实际={sorted(actual_set)}")
            continue

        if actual != want:
            bad.append(f"{key}={want} 实际={actual}")

    return bad


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--fix-report", type=Path, help="把失败项写成 markdown")
    args = ap.parse_args()

    spec = json.loads(AXES.read_text(encoding="utf-8"))
    lib = (SKILL / spec["library"]["root"]).resolve() if not Path(spec["library"]["root"]).is_absolute() else Path(spec["library"]["root"])
    catalog = json.loads(Path(spec["library"]["catalog"]).read_text(encoding="utf-8"))
    by_file = {e["file"]: e for e in catalog}

    failures: list[str] = []
    total = 0

    for axis in spec["axes"]:
        aid = axis["id"]
        for val in axis["values"]:
            vid = val["id"]
            exs = val.get("exemplars") or []
            total += 1

            if len(exs) != 3:
                failures.append(f"`{aid}.{vid}`：exemplar 数量是 {len(exs)}，要求 3")

            for name in exs:
                path = lib / name
                if not path.exists():
                    failures.append(f"`{aid}.{vid}`：文件不存在 `{name}`")
                    continue
                entry = by_file.get(name)
                if entry is None:
                    failures.append(f"`{aid}.{vid}`：`{name}` 不在 catalog 中")
                    continue
                bad = matches(entry, val.get("filter") or {})
                if bad:
                    failures.append(f"`{aid}.{vid}` ← `{name}`：" + "；".join(bad))

    print(f"轴取值 {total} 条，库文件 {len(catalog)} 个\n")

    if not failures:
        print("✅ 全部通过：每条取值都有 3 个真实作品指认，且元数据与 filter 一致。")
        return 0

    print(f"❌ {len(failures)} 处不成立：\n")
    for f in failures:
        print(" -", f)

    if args.fix_report:
        lines = ["# axes.json 校验失败项", ""]
        lines += [f"- {f}" for f in failures]
        args.fix_report.write_text("\n".join(lines) + "\n", encoding="utf-8")
        print(f"\n已写入 {args.fix_report}")

    return 1


if __name__ == "__main__":
    sys.exit(main())
