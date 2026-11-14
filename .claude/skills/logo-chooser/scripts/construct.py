"""站 4 构造：沿轴窄化出黑稿，并挡住不合格的稿子。

这个脚本**不画形**。它做三件事：

  1. 按「固定 1–2 条轴、只动 1–2 条轴」算出这一轮该出哪几组坐标
  2. 把每一张稿量一遍，挡住不合格的（层级 / 呼吸）
  3. 出可指认的网格，记录选择

四条关系不变量里，只有两条能从像素独立测出来：

  层级  主元素面积 ≥ 次元素 3 倍      → 可以量（连通块面积比）
  呼吸  最小间隙 ≥ 主元素短边 1/12    → 可以量（连通块之间的距离）

另外两条测不了，只能由画的人保证：

  锚定  附属元素落在主元素的半径 / 切线 / 角平分线上   → 需要知道「哪条线」
  过渡  两形相接处夹角 ≥ 45°                          → 需要求曲线交点

**测不了的就不假装测了**，页面上标「人工核对」。这比给一个假分数强。

稿子必须申报坐标。不申报就没法算覆盖，也没法证明这一轮真的只动了 1–2 条轴。

用法：
    python -I scripts/construct.py --brand 养成星球 --round 4.1
"""

from __future__ import annotations

import argparse
import hashlib
import html
import json
import math
import sys
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

import cairosvg
from PIL import Image
import io
import re

SKILL = Path(__file__).resolve().parent.parent
AXES = SKILL / "axes.json"

BOX = 256          # 测量分辨率
MIN_HIERARCHY = 3.0
MIN_DRAFTS = 3     # 非对决轮至少 3 张：一条轴只有 3 个值时，3 张就是穷举
BREATH_RATIO = 1 / 12
VB = re.compile(r'viewBox="([-\d.]+)[ ,]+([-\d.]+)[ ,]+([-\d.]+)[ ,]+([-\d.]+)"')

SELECTION_JS = """
(function () {
  const cells = Array.from(document.querySelectorAll('.cell'));
  const picked = new Set();
  const out = document.getElementById('out');
  const count = document.getElementById('count');
  const hint = document.getElementById('hint');
  const copyBtn = document.getElementById('copy');

  function render() {
    cells.forEach(c => c.classList.toggle('picked', picked.has(c.dataset.id)));
    const ids = Array.from(picked).sort();
    out.value = ids.join(',');
    count.textContent = ids.length;
    hint.textContent = ids.length < 3 ? '选 3 张' : '可以了 —— 把编号发回对话';
    hint.classList.toggle('ok', ids.length === 3);
  }

  cells.forEach(c => c.addEventListener('click', () => {
    const id = c.dataset.id;
    if (picked.has(id)) picked.delete(id);
    else if (picked.size < 3) picked.add(id);
    render();
  }));

  out.addEventListener('click', () => out.select());
  copyBtn.addEventListener('click', async () => {
    out.select();
    try { await navigator.clipboard.writeText(out.value); }
    catch (e) { try { document.execCommand('copy'); } catch (_) {} }
    const t = copyBtn.textContent;
    copyBtn.textContent = '已复制';
    setTimeout(() => { copyBtn.textContent = t; }, 1200);
  });

  render();
})();
"""


def _grid(svg_text: str) -> list[list[bool]]:
    m = VB.search(svg_text)
    if not m:
        raise SystemExit("SVG 缺少 viewBox")
    _, _, w, h = (float(v) for v in m.groups())
    scale = BOX / max(w, h)
    ow, oh = max(1, round(w * scale)), max(1, round(h * scale))
    png = cairosvg.svg2png(bytestring=svg_text.encode("utf-8"),
                           output_width=ow, output_height=oh, background_color="white")
    im = Image.open(io.BytesIO(png)).convert("L")
    canvas = Image.new("L", (BOX, BOX), 255)
    canvas.paste(im, ((BOX - ow) // 2, (BOX - oh) // 2))
    px = canvas.load()
    return [[px[x, y] < 128 for x in range(BOX)] for y in range(BOX)]


def _components(g: list[list[bool]]):
    seen = [[False] * BOX for _ in range(BOX)]
    comps = []
    for y in range(BOX):
        for x in range(BOX):
            if not g[y][x] or seen[y][x]:
                continue
            stack, pix = [(y, x)], []
            seen[y][x] = True
            while stack:
                cy, cx = stack.pop()
                pix.append((cy, cx))
                for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1)):
                    ny, nx = cy + dy, cx + dx
                    if 0 <= ny < BOX and 0 <= nx < BOX and g[ny][nx] and not seen[ny][nx]:
                        seen[ny][nx] = True
                        stack.append((ny, nx))
            ys = [p[0] for p in pix]
            xs = [p[1] for p in pix]
            comps.append({
                "area": len(pix), "pixels": pix,
                "bbox": (min(xs), min(ys), max(xs), max(ys)),
            })
    comps.sort(key=lambda c: c["area"], reverse=True)
    return comps


def _min_gap(a: dict, b: dict) -> float:
    """两连通块之间的最短距离（像素，切比雪夫近似用 8 邻域 BFS）。"""
    INF = 10 ** 9
    dist = [[INF] * BOX for _ in range(BOX)]
    stack = []
    for y, x in a["pixels"]:
        dist[y][x] = 0
        stack.append((y, x))
    head = 0
    while head < len(stack):
        cy, cx = stack[head]
        head += 1
        d = dist[cy][cx]
        for dy, dx in ((1, 0), (-1, 0), (0, 1), (0, -1), (1, 1), (1, -1), (-1, 1), (-1, -1)):
            ny, nx = cy + dy, cx + dx
            if 0 <= ny < BOX and 0 <= nx < BOX and dist[ny][nx] > d + 1:
                dist[ny][nx] = d + 1
                stack.append((ny, nx))
    return min(dist[y][x] for y, x in b["pixels"])


def analyze(svg_text: str, expected: int | None = None, framed: bool = False,
            radial: bool = False) -> dict:
    """量一张黑稿。

    `framed=True` 用于带容器的稿子：容器是框，不是「次元素」。
    不跳过的话，框和主体面积相当（1.8:1），层级规则必然判不合格——
    而那条规则想抓的是「主次读不出来」，不是「有个外框」。

    `radial=True` 用于放射构成：三个等大的形按定义就是 1:1，
    层级规则的前提（存在主次）根本不成立。呼吸仍然要量——等大的形更怕挤。
    """
    g = _grid(svg_text)
    comps = _components(g)
    if not comps:
        return {"ok": False, "reasons": ["图是空的"], "hierarchy": None, "gap": None}

    reasons = []
    # 元素数对不上，说明两块粘住了或细节丢了——先报这一条，后面的比值就没意义了
    if expected is not None and len(comps) != expected:
        reasons.append(f"元素数 {len(comps)} ≠ 申报的 {expected}"
                       + ("，两块粘在一起了" if len(comps) < expected else "，多出了碎块"))

    main = comps[0]
    others = comps[1:]
    hierarchy = None
    gap = None

    if others and not framed:
        second = others[0]["area"]
        hierarchy = main["area"] / max(second, 1)
        if hierarchy < MIN_HIERARCHY and not radial:
            reasons.append(f"层级 {hierarchy:.1f}:1 < {MIN_HIERARCHY}:1，主次读不出来")

        gaps = [_min_gap(main, o) for o in others]
        gap = min(gaps)
        x0, y0, x1, y1 = main["bbox"]
        short_side = min(x1 - x0, y1 - y0)
        need = short_side * BREATH_RATIO
        if gap < need:
            reasons.append(f"呼吸 {gap:.0f}px < 阈值 {need:.0f}px（主元素短边 1/12），看着粘住了")

    return {"ok": not reasons, "reasons": reasons,
            "hierarchy": hierarchy, "gap": gap, "components": len(comps),
            "expected": expected, "framed": framed}


# ---------- 形状校验：这张稿是不是它自称的那个形状 ----------
# analyze() 量的是元素之间的比例关系，量不到身份。
# 4.2-hexagon 的实测内角是 106°/148°，4.5-radial 三个标志只有平移没有旋转，
# 两张都通过了 analyze——因为注释里的意图是唯一的审计线，
# 而那条线由画稿的人自己维护。写「radial」，它就成了 radial。


def _polygon_points(d: str) -> list[tuple[float, float]]:
    """抓 path 里 M/L 的折点。曲线段不参与多边形内角计算。"""
    return [(float(x), float(y))
            for _, x, y in re.findall(r"([ML])\s*([\d.]+)[ ,]\s*([\d.]+)", d)]


def _interior_angles(pts: list[tuple[float, float]]) -> list[float]:
    """开口折线只算中间点：首尾是缝口端点，不是多边形顶点。"""
    out = []
    for i in range(1, len(pts) - 1):
        ax, ay = pts[i - 1]
        bx, by = pts[i]
        cx, cy = pts[i + 1]
        v1 = (ax - bx, ay - by)
        v2 = (cx - bx, cy - by)
        n1, n2 = math.hypot(*v1), math.hypot(*v2)
        if n1 < 1e-6 or n2 < 1e-6:
            continue
        cos = max(-1.0, min(1.0, (v1[0] * v2[0] + v1[1] * v2[1]) / (n1 * n2)))
        out.append(math.degrees(math.acos(cos)))
    return out


def _rotate_mismatch(g: list[list[bool]], turns: int) -> float:
    """把栅格图绕中心旋转 360/turns 度，返回对不上的像素比例。"""
    h, w = len(g), len(g[0])
    a = 2 * math.pi / turns
    cos_a, sin_a = math.cos(a), math.sin(a)
    cx, cy = (w - 1) / 2, (h - 1) / 2
    diff = 0
    for y in range(h):
        for x in range(w):
            dx, dy = x - cx, y - cy
            sx = round(cx + dx * cos_a - dy * sin_a)
            sy = round(cy + dx * sin_a + dy * cos_a)
            if not (0 <= sx < w and 0 <= sy < h) or g[y][x] != g[sy][sx]:
                diff += 1
    return diff / (w * h)


def shape_check(svg_text: str, axes: dict) -> list[str]:
    """校验「这张稿是不是它自称的那个形状」。analyze 量关系，这里量身份。"""
    reasons: list[str] = []

    if axes.get("geometry") == "hexagon":
        ds = re.findall(r'\sd="([^"]+)"', svg_text)
        pts = _polygon_points(ds[0]) if ds else []
        angles = _interior_angles(pts)
        if len(angles) < 3:
            reasons.append("hexagon：找不到足够折点，无法确认是六边形")
        else:
            off = [a for a in angles if abs(a - 120) > 2]
            if off:
                shown = "、".join(f"{a:.0f}°" for a in angles)
                reasons.append(f"hexagon：内角应为 120°，实测 {shown}——这不是正六边形")

    if axes.get("symmetry") == "radial":
        g = _grid(svg_text)
        worst = min((_rotate_mismatch(g, n), n) for n in (2, 3, 4, 6))
        if worst[0] > 0.02:
            reasons.append(
                f"radial：按 360/n 旋转后最小也有 {worst[0]:.1%} 像素对不上——"
                f"这不是放射对称，只是把几个图形摆成一圈"
            )

    return reasons


def load_spec() -> dict:
    return json.loads(AXES.read_text(encoding="utf-8"))


def upstream_cells(run_dir: Path) -> list[str]:
    """站 3 的 active 记录里选中的全部格子。

    站 4 的稿子必须从这些格子里长出来。S04 曾经在站 3 是 active，
    但五个轮次的 from 全是 S01——挑 3 张，是在半个决策空间里挑。
    """
    p = run_dir / "picks.json"
    if not p.exists():
        return []
    data = json.loads(p.read_text(encoding="utf-8"))
    for r in reversed(data.get("history", [])):
        if r.get("station") == 3 and r.get("status") == "active":
            out = []
            for line in r.get("chosen") or []:
                m = re.match(r"(S\d+-[a-z]+|wordmark)", line)
                if m:
                    out.append(m.group(1))
            return out
    return []


def axis_values(spec: dict, axis_id: str) -> list[str]:
    axis = next(a for a in spec["axes"] if a["id"] == axis_id)
    return [v["id"] for v in axis["values"]]


def validate_round(spec: dict, rnd: dict) -> list[str]:
    errs: list[str] = []
    fixed, vary = rnd.get("fixed") or {}, rnd.get("vary") or {}
    drafts = rnd.get("drafts") or []

    if not drafts:
        errs.append("这一轮没有任何稿子")
    elif rnd.get("duel"):
        # 对决轮：轴只剩两个值可摆（其余值已在别的轮次出现过），2 张就是穷举。
        if len(drafts) != 2:
            errs.append(f"标了 duel 的轮次恰好 2 张，这一轮有 {len(drafts)} 张")
    elif len(drafts) < MIN_DRAFTS:
        errs.append(
            f"一轮至少 {MIN_DRAFTS} 张，这一轮只有 {len(drafts)} 张。"
            f"如果这一轮本来就是二选一，在 round-4.x.json 里写 \"duel\": true，"
            f"并在 note 里说清另一个取值在哪一轮出现过。"
        )
    if len(fixed) + len(vary) == 0:
        errs.append("没有声明任何轴")
    if len(vary) > 2:
        errs.append(f"一轮只许动 1–2 条轴，声明了 {len(vary)} 条：{', '.join(vary)}")

    for axis_id, vals in list(fixed.items()) + list(vary.items()):
        declared = axis_values(spec, axis_id)
        for v in (vals if isinstance(vals, list) else [vals]):
            if v not in declared:
                errs.append(f"{axis_id} 没有取值 {v}")

    # 每个稿子的坐标必须落在这一轮声明的范围内，且组合唯一
    seen = set()
    for d in drafts:
        ax = d.get("axes") or {}
        for axis_id in list(fixed) + list(vary):
            if axis_id not in ax:
                errs.append(f"{d.get('id')} 缺坐标 {axis_id}")
        for axis_id, v in fixed.items():
            if ax.get(axis_id) != v:
                errs.append(f"{d.get('id')} 的 {axis_id}={ax.get(axis_id)}，但这一轮固定 {v}")
        for axis_id, allowed in vary.items():
            if ax.get(axis_id) not in allowed:
                errs.append(f"{d.get('id')} 的 {axis_id}={ax.get(axis_id)} 不在本轮允许值 {allowed}")
        key = tuple(sorted(ax.items()))
        if key in seen:
            errs.append(f"{d.get('id')} 与前面的稿子坐标完全相同")
        seen.add(key)

    # 变化轴的每个取值至少出现一次——缺一个，那一档就永远选不到
    for axis_id, allowed in vary.items():
        present = {d.get("axes", {}).get(axis_id) for d in drafts}
        missing = [v for v in allowed if v not in present]
        if missing:
            errs.append(f"{axis_id} 的取值没出现：{', '.join(missing)}（这条来自 coverage 规则）")

    return errs


def build_page(rnd: dict, drafts: list[dict], run_dir: Path) -> str:
    fixed, vary = rnd.get("fixed") or {}, rnd.get("vary") or {}
    axes_shown = list(fixed) + list(vary)

    cards = []
    for d in drafts:
        svg = (run_dir / "drafts" / d["file"]).read_text(encoding="utf-8").strip()
        if svg.startswith("<?xml"):
            svg = svg.split("?>", 1)[1].strip()
        r = d["_check"]

        chips = "".join(
            f'<span class="{"fixed" if a in fixed else "vary"}">{a}={d["axes"][a]}</span>'
            for a in axes_shown
        )
        if r["ok"]:
            verdict = '<span class="pass">层级 / 呼吸 通过</span>'
        else:
            verdict = "".join(f'<span class="fail">{html.escape(x)}</span>' for x in r["reasons"])

        cards.append(f"""
    <button class="cell" data-id="{d['id']}" data-fail="{0 if r['ok'] else 1}">
      <div class="tick">✓</div>
      <div class="stage">{svg}</div>
      <div class="axes">{chips}</div>
      <div class="verdict">{verdict}</div>
      <div class="manual">锚定 / 过渡：人工核对</div>
    </button>""")

    return f"""<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(rnd.get('brand',''))} · 站 4 构造 {html.escape(rnd.get('round',''))}</title>
<style>
  :root {{ --ink:#16181D; --muted:#6B6F76; --line:#E5DFD2; --paper:#FAF8F3; --card:#fff; }}
  * {{ box-sizing: border-box; }}
  body {{
    margin:0; padding:44px 24px 116px; background:var(--paper); color:var(--ink);
    font-family:-apple-system,"Segoe UI","Microsoft YaHei",sans-serif; -webkit-font-smoothing:antialiased;
  }}
  header {{ max-width:1180px; margin:0 auto 28px; }}
  h1 {{ margin:0 0 10px; font-size:22px; font-weight:600; }}
  header p {{ margin:0 0 6px; font-size:13px; line-height:1.75; color:var(--muted); max-width:720px; }}
  header b {{ color:var(--ink); font-weight:600; }}
  .round {{
    display:inline-block; margin-top:6px; padding:4px 10px; border-radius:6px;
    background:#F1EDE3; color:#8C8578; font-size:11.5px;
  }}
  .grid {{
    max-width:1180px; margin:0 auto;
    display:grid; grid-template-columns:repeat(auto-fit, minmax(260px, 1fr)); gap:14px;
  }}
  .cell {{
    position:relative; background:var(--card); border:1px solid var(--line);
    border-radius:16px; padding:18px 16px 14px; cursor:pointer;
    display:flex; flex-direction:column; align-items:center; gap:10px;
    font:inherit; color:inherit; transition:border-color .15s, box-shadow .15s;
  }}
  .cell:hover {{ border-color:#CFC7B6; }}
  .cell.picked {{ border-color:var(--ink); box-shadow:0 0 0 1px var(--ink); }}
  .cell[data-fail="1"] {{ background:#FDFBF7; }}
  .tick {{
    position:absolute; top:8px; right:8px; width:18px; height:18px; border-radius:50%;
    background:var(--ink); color:#fff; font-size:10px; display:none;
    align-items:center; justify-content:center;
  }}
  .cell.picked .tick {{ display:flex; }}
  .stage {{ height:120px; display:flex; align-items:center; justify-content:center; }}
  .stage svg {{ height:120px; width:auto; max-width:100%; }}
  .axes {{ display:flex; flex-wrap:wrap; gap:5px; justify-content:center; }}
  .axes span {{ font-size:10.5px; padding:2px 7px; border-radius:5px; }}
  .axes .fixed {{ background:#EFEAE0; color:#9A948A; }}
  .axes .vary {{ background:#E4EFE6; color:#3F6B4A; }}
  .verdict {{ font-size:10.5px; display:flex; flex-wrap:wrap; gap:5px; justify-content:center; }}
  .verdict .pass {{ color:#3F6B4A; }}
  .verdict .fail {{ color:#B45309; }}
  .manual {{ font-size:10px; color:#C0B9AC; }}
  .bar {{
    position:fixed; left:0; right:0; bottom:0; z-index:10;
    background:rgba(250,248,243,.95); backdrop-filter:blur(10px);
    border-top:1px solid var(--line); padding:14px 24px;
  }}
  .bar .inner {{ max-width:1180px; margin:0 auto; display:flex; align-items:center; gap:14px; flex-wrap:wrap; }}
  .bar .num {{ font-size:13px; color:var(--muted); white-space:nowrap; }}
  .bar .num b {{ color:var(--ink); font-size:15px; font-variant-numeric:tabular-nums; }}
  .bar .hint {{ font-size:12.5px; color:var(--muted); flex:1; min-width:180px; }}
  .bar .hint.ok {{ color:#3F6B4A; }}
  .bar input {{
    width:170px; font-family:ui-monospace,Consolas,monospace; font-size:12.5px;
    padding:8px 10px; text-align:center; border:1px solid var(--line); border-radius:8px; background:#fff;
  }}
  .bar button {{
    padding:8px 14px; font-size:12.5px; border-radius:8px;
    border:1px solid var(--ink); background:var(--ink); color:#fff; cursor:pointer; white-space:nowrap;
  }}
</style>
</head>
<body>
<header>
  <h1>{html.escape(rnd.get('brand',''))}</h1>
  <p>
    <b>站 4 · 构造 {html.escape(rnd.get('round',''))}。</b>
    这一轮只动 <b>{len(vary)}</b> 条轴，其余锁定。选 <b>3 张</b>你愿意再看一眼的。
  </p>
  <p>层级和呼吸是量出来的；锚定和过渡要你自己看——脚本不假装能测。</p>
  <div class="round">固定：{', '.join(f'{k}={v}' for k, v in fixed.items()) or '—'}　·　变化：{', '.join(vary) or '—'}</div>
</header>
<div class="grid">{''.join(cards)}
</div>
<div class="bar">
  <div class="inner">
    <span class="num">已选 <b id="count">0</b> / 3</span>
    <span class="hint" id="hint">选 3 张</span>
    <input id="out" readonly value="" placeholder="——">
    <button id="copy">复制编号</button>
  </div>
</div>
<script>{SELECTION_JS}</script>
</body>
</html>
"""


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--brand", default="养成星球")
    ap.add_argument("--round", default="4.1")
    ap.add_argument("--out", type=Path)
    args = ap.parse_args()

    run_dir = SKILL / "runs" / args.brand
    spec = load_spec()

    spec_path = run_dir / f"round-4.{args.round.split('.')[-1]}.json"
    if not spec_path.exists():
        spec_path = run_dir / "round-4.json"
    if not spec_path.exists():
        print(f"❌ 找不到这一轮的声明：{run_dir / 'round-4.json'}")
        print("   站 4 的稿子需要人工/AI 画，并申报坐标。")
        return 1

    rnd = json.loads(spec_path.read_text(encoding="utf-8"))
    rnd.setdefault("brand", args.brand)
    errs = validate_round(spec, rnd)

    upstream = upstream_cells(run_dir)
    if upstream and rnd.get("from") not in upstream:
        errs.append(
            f"from={rnd.get('from')} 不在站 3 的 active 里：{', '.join(upstream)}"
            f"——站 4 的稿子必须从站 3 选中的格子里长出来"
        )

    if errs:
        print(f"❌ 这一轮有 {len(errs)} 处问题：")
        for e in errs:
            print("   -", e)
        return 1

    drafts = rnd["drafts"]
    seen_paths: dict[str, str] = {}
    for d in drafts:
        p = run_dir / "drafts" / d["file"]
        if not p.exists():
            print(f"❌ 缺稿子：{p}")
            return 1
        svg = p.read_text(encoding="utf-8")
        d["_check"] = analyze(
            svg,
            d.get("elements"),
            framed=(d.get("axes", {}).get("container") not in (None, "none")),
            radial=(d.get("axes", {}).get("symmetry") == "radial"),
        )
        d["_check"]["reasons"] += shape_check(svg, d.get("axes", {}))

        # 同一张图存两次：两个不同问题的答案长得一样，用户会读成「这就是标」。
        # 只比 path 会漏掉「同样的芽、不同的容器」，所以整段规范化后再哈希。
        body = re.sub(r"<title>.*?</title>|<!--.*?-->|<svg[^>]*>|</svg>", "", svg, flags=re.S)
        sig = hashlib.sha256(re.sub(r"\s+", " ", body).strip().encode()).hexdigest()[:12]
        if sig in seen_paths:
            d["_check"]["reasons"].append(
                f"路径与 {seen_paths[sig]} 完全相同——同一张图存了两次"
            )
        else:
            seen_paths[sig] = d["id"]
        d["_check"]["ok"] = not d["_check"]["reasons"]

    bad = [d for d in drafts if not d["_check"]["ok"]]
    for d in bad:
        print(f"⚠️  {d['id']} 不合格：" + "；".join(d["_check"]["reasons"]))

    out = args.out or (run_dir / f"round-4-{rnd.get('round','1')}.html")
    out.write_text(build_page(rnd, drafts, run_dir), encoding="utf-8")

    # 每一轮写自己的 out 文件。旧版统一写 round-4.out.json 且每轮覆盖，
    # 结果 narrow.py 只看得见最后一轮——前面几轮的稿子还在磁盘上，但选不了。
    out_json = run_dir / f"round-4.{rnd.get('round')}.out.json"
    out_json.write_text(json.dumps({
        "brand": rnd.get("brand"), "round": rnd.get("round"),
        "from": rnd.get("from"),
        "fixed": rnd.get("fixed"), "vary": rnd.get("vary"),
        "drafts": [{k: v for k, v in d.items() if k != "_check"} for d in drafts],
        "failed": [d["id"] for d in bad],
        "failed_detail": [{"id": d["id"], "reasons": d["_check"]["reasons"]} for d in bad],
    }, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    print(f"✅ {out}")
    print(f"   {len(drafts)} 张稿 · {len(bad)} 张不合格（仍会显示，但标红）")
    print(f"   声明已写入 {out_json.name}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
