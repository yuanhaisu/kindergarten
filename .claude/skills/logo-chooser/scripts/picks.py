"""选型记录：picks.json 是唯一真相源，brief.md 由它派生。

为什么要派生而不是追加：

    上游一改，下游的结论就过期了。追加式的 brief 无法表达「这条已经不算数」，
    于是文档里会同时躺着两个互相矛盾的结论，而读者不知道信哪个。

依赖关系是显式的，不是「所有下游都失效」：

    站 1 ──┐
           ├──> 站 3
    站 2 ──┘

改站 1 不影响站 2 的句子选择，但会让站 3 的类型结论过期。

三种状态：
    active      当前有效
    superseded  同站有更晚的选择，这条进了历史
    stale       上游改过，这条结论不再成立
"""

from __future__ import annotations

import json
from datetime import datetime
from pathlib import Path

# 谁依赖谁。新增站时必须在这里登记，否则上游改动不会让下游失效。
DEPENDS_ON: dict[int, set[int]] = {1: set(), 2: set(), 3: {1, 2}, 4: {3}}

TITLE = {1: "站 1 气质", 2: "站 2 意象", 3: "站 3 类型", 4: "站 4 构造"}
ORDER = [1, 2, 3, 4]


def load(path: Path) -> dict:
    if not path.exists():
        return {"brand": "", "history": []}
    return normalize(json.loads(path.read_text(encoding="utf-8")))


def save(path: Path, data: dict) -> None:
    path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def normalize(data: dict) -> dict:
    """补齐缺字段：seq（单调序号）与 status。

    ⚠️ 版本标识必须用 seq，不能用 at：同一秒内可以发生多次选择，
    秒级时间戳会撞在一起，导致「上游变了」被误判成「没变」。
    """
    hist = data.setdefault("history", [])
    for i, h in enumerate(hist, 1):
        h.setdefault("seq", i)
        if h.get("status"):
            continue
        h["status"] = "superseded" if any(
            o.get("station") == h.get("station") and o.get("at", "") > h.get("at", "")
            for o in hist
        ) else "active"
    return data


def next_seq(data: dict) -> int:
    return max((h.get("seq", 0) for h in data.get("history", [])), default=0) + 1


def dependents(station: int) -> set[int]:
    return {s for s, deps in DEPENDS_ON.items() if station in deps}


def snapshot_upstream(data: dict, station: int) -> dict[str, int]:
    """记下这次选择所依据的上游版本（seq）——没有它，restore 就没法判断结论是否还成立。"""
    out: dict[str, int] = {}
    for dep in sorted(DEPENDS_ON.get(station, set())):
        e = active(data, dep)
        if e:
            out[str(dep)] = e["seq"]
    return out


def restore(data: dict, station: int, attempt: int, at: datetime) -> dict:
    """把该站第 attempt 次的选择恢复为当前。

    恢复一条「按旧上游得出」的结论是不合法的，所以先核对 based_on：
    上游变了就拒绝，并说明该先恢复谁、还是该重做。
    """
    hist = data.get("history", [])
    entries = [h for h in hist if h.get("station") == station]

    if not entries:
        return {"ok": False, "error": f"站 {station} 还没有任何记录"}
    if not 1 <= attempt <= len(entries):
        return {"ok": False, "error": f"站 {station} 只有 {len(entries)} 次记录，没有第 {attempt} 次"}

    target = entries[attempt - 1]
    if target.get("status") == "active":
        return {"ok": True, "changed": False, "target": target, "dirty": []}

    deps = DEPENDS_ON.get(station, set())
    if deps:
        based = target.get("based_on")
        if not based:
            return {"ok": False, "error": (
                f"站 {station} 依赖站 {', '.join(map(str, sorted(deps)))}，"
                f"但这条记录没有记下上游版本，无法判断是否还成立；请重做而不是恢复"
            )}
        for dep, expected in based.items():
            cur = active(data, int(dep))
            # 老记录存的是 at 字符串，新记录存 seq；两种都要能对上
            cur_id = None if cur is None else (
                cur.get("seq") if isinstance(expected, int) else cur.get("at")
            )
            if cur_id != expected:
                cur_desc = "（无有效记录）" if cur is None else f"{_fmt_time(cur.get('at', ''))}（#{cur.get('seq')}）"
                exp_desc = f"#{expected}" if isinstance(expected, int) else expected
                return {"ok": False, "error": (
                    f"站 {dep} 当前有效的是 {cur_desc}，"
                    f"而这条结论是按 {exp_desc} 的站 {dep} 得出的。\n"
                    f"   直接恢复会得到一个自相矛盾的记录。先恢复站 {dep}，或重做站 {station}。"
                )}

    stamp = at.strftime("%Y-%m-%d %H:%M")
    dirty: list[int] = []
    for h in hist:
        if h is target:
            h["status"] = "active"
            h["restored_at"] = at.isoformat(timespec="seconds")
        elif h.get("station") == station:
            h["status"] = "superseded"
        elif h.get("station") in dependents(station) and h.get("status") == "active":
            h["status"] = "stale"
            h["stale_reason"] = f"站 {station} 于 {stamp} 回退到第 {attempt} 次"
            dirty.append(h["station"])

    return {"ok": True, "changed": True, "target": target, "dirty": sorted(set(dirty))}


def active(data: dict, station: int) -> dict | None:
    """该站当前有效的记录。全被标成 stale 时返回 None——那时必须重选。"""
    for h in reversed(data.get("history", [])):
        if h.get("station") == station and h.get("status") == "active":
            return h
    return None


def record(data: dict, station: int, *, at: datetime, candidates: str,
           picks: list[dict], chosen: list[str], rejected: list[str],
           locked: str, note: str | None,
           exception: str | None = None) -> tuple[dict, list[str]]:
    """追加一次选择，并把受影响的下游标成 stale。返回 (新条目, 被标脏的说明)。"""
    hist = data.setdefault("history", [])
    stamp = at.strftime("%Y-%m-%d %H:%M")
    dirtied: list[str] = []
    based_on = snapshot_upstream(data, station)

    for h in hist:
        if h.get("status") != "active":
            continue
        if h.get("station") == station:
            h["status"] = "superseded"
        elif h.get("station") in dependents(station):
            h["status"] = "stale"
            h["stale_reason"] = f"站 {station} 于 {stamp} 重新选择"
            dirtied.append(h.get("station"))

    entry = {
        "seq": next_seq(data),
        "station": station,
        "at": at.isoformat(timespec="seconds"),
        "status": "active",
        "candidates": candidates,
        "picks": picks,
        "chosen": chosen,
        "rejected": rejected,
        "locked": locked,
        "note": note,
        "exception": exception,
        "based_on": based_on,
    }
    hist.append(entry)
    return entry, sorted(set(dirtied))


def _fmt_time(iso: str) -> str:
    try:
        return datetime.fromisoformat(iso).strftime("%Y-%m-%d %H:%M")
    except ValueError:
        return iso


def render_brief(data: dict) -> str:
    """整篇从 history 重新生成。任何一条记录的状态变化都会立刻反映出来。"""
    lines = [f"# {data.get('brand', '')} · 选型记录", ""]

    for st in ORDER:
        entries = [h for h in data.get("history", []) if h.get("station") == st]
        if not entries:
            continue

        lines += [f"## {TITLE[st]}", ""]
        for i, h in enumerate(entries, 1):
            if h.get("status") == "stale":
                mark = f"　⚠️ 已过期（{h.get('stale_reason', '上游已改')}）——需要重做"
            elif h.get("status") == "superseded":
                mark = "　（已被同站更晚的选择取代）"
            else:
                mark = ""
            if h.get("restored_at") and h.get("status") == "active":
                mark += f"　↩ 于 {_fmt_time(h['restored_at'])} 回退到此"

            lines.append(f"### 第 {i} 次 · {_fmt_time(h.get('at', ''))}{mark}")
            lines.append("")
            lines.append(f"- 候选：{h.get('candidates', '—')}")
            lines.append(f"- 选中：{'；'.join(h.get('chosen', []))}")
            lines.append(f"- 未选：{'、'.join(h.get('rejected', []))}")
            lines.append(f"- 锁定：{h.get('locked', '—')}")
            if h.get("exception"):
                lines.append(f"- **破例**：{h['exception']}")
            if h.get("note"):
                lines.append(f"- 原话：{h['note']}")
            lines.append("")

    stale = [h for h in data.get("history", []) if h.get("status") == "stale"]
    if stale:
        lines += [
            "---",
            "",
            "> 有记录被标记为过期：上游改过之后，下面的结论不再成立。",
            "> 重新走一遍对应站，或者明确接受旧结论——但不能当作没发生过。",
            "",
        ]

    return "\n".join(lines).rstrip() + "\n"
