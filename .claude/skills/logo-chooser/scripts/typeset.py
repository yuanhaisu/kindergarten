"""站 1 渲染器：同名试排。

把品牌名按「mood × 字重」排 8 次，每次出一个轮廓化的 SVG，配 3 个该 mood 的
真实 exemplar 缩略图，拼成一张可指认的卡片页。

不做的事：
  - 不画图形（站 3 之后才轮到 logo-design）
  - 不选字（只按 axes.json → zh.station1_cards 排）
  - 不用 <text>（探索稿允许，但轮廓化是习惯；后续站直接用同一份几何）

中文字面率约 0.9，直接把 font-size 当字高会偏大，所以先按墨迹框做等面积
校正，再按等墨迹间隙布局——和 assets/logo-v2 里 done 的字标同一套算法。

用法：
    python -I scripts/typeset.py                       # 默认跑 axes.json 里配置的品牌
    python -I scripts/typeset.py --brand 养成星球
    python -I scripts/typeset.py --brand 养成星球 --out ../../runs/demo/round-1.html
"""

from __future__ import annotations

import argparse
import html
import json
import shutil
import sys
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

from fontTools.pens.boundsPen import BoundsPen
from fontTools.pens.svgPathPen import SVGPathPen
from fontTools.ttLib import TTFont

sys.path.insert(0, str(Path(__file__).resolve().parent))
from coverage_check import require  # noqa: E402

SKILL = Path(__file__).resolve().parent.parent
AXES = SKILL / "axes.json"

SIZE = 100.0          # 内部排版尺寸，最后靠 CSS 缩放
BASE_INK_GAP = 0.10   # em，中文正常的字间墨迹间隙
OPTICAL_CLAMP = (0.86, 1.14)


def is_cjk(ch: str) -> bool:
    o = ord(ch)
    return 0x3400 <= o <= 0x9FFF or 0xF900 <= o <= 0xFAFF


def family(card: dict) -> str:
    return "serif" if "serif" in card["font"] else "sans"


def make_tier(tiers: list[dict]):
    """重量档阈值只在 axes.json 里，narrow.py 读同一份，避免两处各写一遍。"""
    def tier(wght: float) -> str:
        for t in tiers:
            if wght >= t.get("min", float("-inf")) and wght <= t.get("max", float("inf")):
                return t["id"]
        return tiers[-1]["id"]
    return tier


SELECTION_JS = """
(function () {
  const cards = Array.from(document.querySelectorAll('.card'));
  const picked = new Set();
  const out = document.getElementById('out');
  const count = document.getElementById('count');
  const hint = document.getElementById('hint');
  const copyBtn = document.getElementById('copy');

  const combo = c => c.dataset.family + '|' + c.dataset.tier;

  function render() {
    cards.forEach(c => c.classList.toggle('picked', picked.has(c.dataset.id)));
    const ids = Array.from(picked).sort();
    out.value = ids.join(',');
    count.textContent = ids.length;

    let msg = '至少 2 张，最多 3 张';
    let warn = false;
    if (ids.length >= 2) {
      const combos = new Set(cards.filter(c => picked.has(c.dataset.id)).map(combo));
      if (combos.size === 1) {
        msg = '这几张是同一个分量，换一张不同轻重的';
        warn = true;
      } else {
        msg = '可以了 —— 把编号发回对话';
      }
    }
    hint.textContent = msg;
    hint.classList.toggle('warn', warn);
  }

  cards.forEach(c => c.addEventListener('click', () => {
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


class Face:
    """一个字重下的字体视图。用 getGlyphSet(location=) 做变体，不整体实例化。"""

    def __init__(self, path: str | Path, wght: float):
        self.font = TTFont(str(path), lazy=True)
        self.upm = self.font["head"].unitsPerEm
        self.cmap = self.font.getBestCmap()
        self.glyphs = self.font.getGlyphSet(location={"wght": wght})
        self.hmtx = self.font["hmtx"]

    def ink(self, ch: str) -> dict:
        name = self.cmap.get(ord(ch))
        if name is None:
            raise SystemExit(f"字体 {Path(self.font.reader.file.name).name} 缺少字符：{ch}")

        g = self.glyphs[name]
        bp = BoundsPen(self.glyphs)
        g.draw(bp)
        if bp.bounds is None:  # 空格之类
            return {"x0": 0, "x1": 0, "y0": 0, "y1": 0, "bw": 0, "bh": 0,
                    "area": 1e-6, "d": "", "adv": self.hmtx[name][0]}

        sp = SVGPathPen(self.glyphs)
        g.draw(sp)
        x0, y0, x1, y1 = bp.bounds
        bw, bh = x1 - x0, y1 - y0
        return {
            "x0": x0, "x1": x1, "y0": y0, "y1": y1,
            "bw": bw, "bh": bh, "area": max(bw * bh, 1e-6),
            "d": sp.getCommands() or "", "adv": self.hmtx[name][0],
        }


MIX_GAP = 0.10   # em，中英文直接相邻（没有空格）时的间距


def segment(text: str) -> list[tuple[str, str]]:
    """切成连续的中文段与拉丁段。

    不能用 all(is_cjk) 一刀切：只要混进一个英文，整串就会走拉丁路径，
    中文字的等面积校正全部丢失——而双语锁图恰恰是这种情形。
    """
    runs: list[tuple[str, str]] = []
    buf: list[str] = []
    kind: str | None = None
    for ch in text:
        k = "cjk" if is_cjk(ch) else "latin"
        if buf and k != kind:
            runs.append((kind, "".join(buf)))  # type: ignore[arg-type]
            buf = []
        buf.append(ch)
        kind = k
    if buf:
        runs.append((kind, "".join(buf)))  # type: ignore[arg-type]
    return runs


def typeset(text: str, face: Face, size: float = SIZE,
            tracking: float = 0.0, jitter: float = 0.0) -> str:
    """返回一段 <svg>，墨迹轮廓化，画布高度固定——所以 8 张卡的字号是同一个。

    中文段和拉丁段各用各的排版模型：中文等墨迹面积 + 等墨迹间隙，
    拉丁按前进宽度。两段各自算完宽度，再顺序摆放。
    """
    k = size / face.upm
    items: list[dict] = []
    cursor = 0.0

    runs = segment(text)
    for ri, (kind, run) in enumerate(runs):
        metrics = [face.ink(c) for c in run]

        if kind == "cjk":
            target = max(m["area"] for m in metrics)
            for m in metrics:
                m["s"] = min(max((target / m["area"]) ** 0.5, OPTICAL_CLAMP[0]), OPTICAL_CLAMP[1])
            gap = (BASE_INK_GAP + tracking) * size
            x = cursor
            for i, (ch, m) in enumerate(zip(run, metrics)):
                s = m["s"] * k
                dx = x - m["x0"] * s
                x += m["bw"] * s + gap
                dy = (jitter * size * (0.5 if i % 2 else -0.5)) if jitter else 0.0
                if m["d"]:
                    items.append({"dx": dx, "dy": dy, "s": s, "d": m["d"], **{
                        q: m[q] for q in ("x0", "x1", "y0", "y1")
                    }})
            run_w = x - cursor - gap
        else:
            x = cursor
            for i, (ch, m) in enumerate(zip(run, metrics)):
                s = k
                if m["d"]:
                    items.append({"dx": x, "dy": 0.0, "s": s, "d": m["d"], **{
                        q: m[q] for q in ("x0", "x1", "y0", "y1")
                    }})
                x += (m["adv"] + tracking * face.upm) * k
            run_w = x - cursor - tracking * face.upm * k

        cursor += run_w

        # 段间：本来就有空格就不再补，直接相邻才加一点呼吸
        if ri + 1 < len(runs):
            nxt = runs[ri + 1][1]
            if not (run[-1].isspace() or nxt[0].isspace()):
                cursor += MIX_GAP * size

    if not items:
        raise SystemExit(f"「{text}」没有可画的墨迹")

    left = min(it["dx"] + it["x0"] * it["s"] for it in items)
    right = max(it["dx"] + it["x1"] * it["s"] for it in items)
    width = right - left

    # 固定画布：基线放在 1.18em，画布 1.50em —— 所有卡片的字号因此一致
    canvas_h = size * 1.50
    baseline = size * 1.18

    parts = []
    for it in items:
        tx = it["dx"] - left
        ty = baseline + it["dy"]
        parts.append(
            f'<g transform="translate({tx:.2f} {ty:.2f}) '
            f'scale({it["s"]:.6f} {-it["s"]:.6f})">'
            f'<path d="{it["d"]}"/></g>'
        )

    return (
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {width:.2f} {canvas_h:.2f}" '
        f'fill="currentColor" role="img" aria-label="{html.escape(text)}">'
        + "".join(parts) + "</svg>"
    )


def spec_label(card: dict, fonts: dict[str, dict]) -> str:
    fam = fonts[card["font"]]["zh"]
    bits = [fam, str(card["wght"])]
    t = card.get("tracking", 0)
    bits.append("字距 " + (f"{t:+.0%}" if t else "0"))
    if card.get("baseline_jitter"):
        bits.append("基线微错")
    return " · ".join(bits)


def build_page(brand: str, cards: list[dict], fonts: dict[str, dict],
               exemplars: dict[str, list[str]], assets: dict[str, str],
               tiers: list[dict], size: float = SIZE) -> str:
    tier = make_tier(tiers)
    blocks = []
    for i, card in enumerate(cards, 1):
        face = Face(fonts[card["font"]]["file"], card["wght"])
        svg = typeset(brand, face, size=size, tracking=card.get("tracking", 0.0),
                      jitter=card.get("baseline_jitter", 0.0))

        thumbs = "".join(
            f'<img src="{html.escape(assets[name])}" alt="">'
            for name in exemplars.get(card["mood"], [])
        )

        blocks.append(f"""
  <article class="card" data-id="{i:02d}" data-mood="{card['mood']}"
           data-family="{family(card)}" data-tier="{tier(card['wght'])}">
    <div class="idx">{i:02d}</div>
    <div class="tick">✓</div>
    <div class="specimen">{svg}</div>
    <div class="mood"><b>{html.escape(card['zh'])}</b><span>{html.escape(card['meaning'])}</span></div>
    <div class="exemplars">{thumbs}</div>
    <div class="spec">{html.escape(spec_label(card, fonts))}</div>
  </article>""")

    return f"""<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(brand)} · 站 1 气质试排</title>
<style>
  :root {{
    --ink: #16181D;
    --muted: #6B6F76;
    --line: #E5DFD2;
    --paper: #FAF8F3;
    --card: #FFFFFF;
  }}
  * {{ box-sizing: border-box; }}
  body {{
    margin: 0; padding: 48px 24px 116px;
    background: var(--paper); color: var(--ink);
    font-family: -apple-system, "Segoe UI", "Microsoft YaHei", sans-serif;
    -webkit-font-smoothing: antialiased;
  }}
  header {{ max-width: 1120px; margin: 0 auto 32px; }}
  h1 {{ margin: 0 0 10px; font-size: 22px; font-weight: 600; letter-spacing: .01em; }}
  header p {{ margin: 0; font-size: 13px; line-height: 1.7; color: var(--muted); max-width: 640px; }}
  header b {{ color: var(--ink); font-weight: 600; }}
  .grid {{
    max-width: 1120px; margin: 0 auto;
    display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 18px;
  }}
  .card {{
    position: relative; background: var(--card);
    border: 1px solid var(--line); border-radius: 16px;
    padding: 26px 24px 18px;
    display: flex; flex-direction: column; gap: 14px;
    cursor: pointer; user-select: none;
    transition: border-color .15s, box-shadow .15s, background .15s;
  }}
  .card:hover {{ border-color: #CFC7B6; }}
  .card.picked {{ border-color: var(--ink); box-shadow: 0 0 0 1px var(--ink); }}
  .idx {{
    position: absolute; top: 14px; right: 18px;
    font-size: 11px; letter-spacing: .12em; color: #B9B3A6;
    font-variant-numeric: tabular-nums;
  }}
  .tick {{
    position: absolute; top: 11px; right: 14px;
    width: 20px; height: 20px; border-radius: 50%;
    background: var(--ink); color: #fff; font-size: 11px;
    display: none; align-items: center; justify-content: center;
  }}
  .card.picked .tick {{ display: flex; }}
  .card.picked .idx {{ display: none; }}
  .specimen {{ height: 54px; color: var(--ink); }}
  .specimen svg {{ height: 100%; width: auto; display: block; }}
  .mood {{ display: flex; align-items: baseline; gap: 10px; }}
  .mood b {{ font-size: 15px; font-weight: 600; }}
  .mood span {{ font-size: 12.5px; color: var(--muted); }}
  .exemplars {{ display: flex; align-items: center; gap: 10px; height: 30px; }}
  .exemplars img {{
    height: 30px; width: 46px; object-fit: contain;
    filter: grayscale(1); opacity: .5;
  }}
  .spec {{
    margin-top: auto; padding-top: 12px; border-top: 1px solid #F0ECE3;
    font-size: 11px; letter-spacing: .04em; color: #9A948A;
  }}
  footer {{
    max-width: 1120px; margin: 28px auto 0;
    font-size: 12px; line-height: 1.8; color: #9A948A;
  }}
  .bar {{
    position: fixed; left: 0; right: 0; bottom: 0; z-index: 10;
    background: rgba(250,248,243,.95); backdrop-filter: blur(10px);
    border-top: 1px solid var(--line); padding: 14px 24px;
  }}
  .bar .inner {{
    max-width: 1120px; margin: 0 auto;
    display: flex; align-items: center; gap: 14px; flex-wrap: wrap;
  }}
  .bar .num {{ font-size: 13px; color: var(--muted); white-space: nowrap; }}
  .bar .num b {{ color: var(--ink); font-variant-numeric: tabular-nums; font-size: 15px; }}
  .bar .hint {{ font-size: 12.5px; color: var(--muted); flex: 1; min-width: 180px; }}
  .bar .hint.warn {{ color: #B45309; }}
  .bar input {{
    width: 150px; font-family: ui-monospace, "SF Mono", Consolas, monospace;
    font-size: 13px; padding: 8px 10px; text-align: center;
    border: 1px solid var(--line); border-radius: 8px;
    background: #fff; color: var(--ink);
  }}
  .bar button {{
    padding: 8px 14px; font-size: 12.5px; border-radius: 8px;
    border: 1px solid var(--ink); background: var(--ink); color: #fff;
    cursor: pointer; white-space: nowrap;
  }}
</style>
</head>
<body>
<header>
  <h1>{html.escape(brand)}</h1>
  <p>
    <b>站 1 · 气质试排。</b>同一个名字，8 种手感。先不要管图形，<b>点卡片</b>选出 2–3 张
    「对，是这个感觉」的。选完把编号发回对话。<br>
    不要全挑同一个分量的（比如三张都很重）——那通常是在选「看起来专业」，不是在选你要的。
  </p>
</header>
<main class="grid">{"".join(blocks)}
</main>
<footer>
  灰度的三张小图是该气质的真实作品，只用来学构造，不用来对行业——库里样本偏科技/软件。<br>
  这不是最终字体：站 1 只定气质方向，中文字标的真正差异在站 4 的结构改装。
</footer>
<div class="bar">
  <div class="inner">
    <span class="num">已选 <b id="count">0</b> / 3</span>
    <span class="hint" id="hint">至少 2 张，最多 3 张</span>
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
    ap.add_argument("--out", type=Path)
    ap.add_argument("--size", type=float, default=100.0, help="内部排版字号，默认 100")
    args = ap.parse_args()

    spec = json.loads(AXES.read_text(encoding="utf-8"))
    zh = spec["zh"]
    fonts = {f["id"]: f for f in zh["fonts"]}
    cards = zh["station1_cards"]

    # 8 个 mood 必须一个不少：缺一个，那个气质就永远选不到
    require(spec, "mood", [c["mood"] for c in cards], where="站 1 · 气质")

    # mood → exemplars，取自轴表，不另抄一份
    mood_axis = next(a for a in spec["axes"] if a["id"] == "mood")
    by_mood = {v["id"]: v["exemplars"] for v in mood_axis["values"]}

    out = args.out or (SKILL / "runs" / args.brand / "round-1.html")
    out.parent.mkdir(parents=True, exist_ok=True)
    assets_dir = out.parent / "assets"
    assets_dir.mkdir(exist_ok=True)

    lib = Path(spec["library"]["root"])

    assets: dict[str, str] = {}
    exemplars: dict[str, list[str]] = {}
    missing: list[str] = []

    for card in cards:
        names = []
        for name in by_mood.get(card["mood"], []):
            src = lib / name
            if not src.exists():
                missing.append(name)
                continue
            dst = assets_dir / name
            if not dst.exists():
                shutil.copy2(src, dst)
            assets[name] = f"assets/{name}"
            names.append(name)
        exemplars[card["mood"]] = names

    page = build_page(args.brand, cards, fonts, exemplars, assets,
                      tiers=zh["weight_tiers"], size=args.size)
    out.write_text(page, encoding="utf-8")

    print(f"✅ {out}")
    print(f"   {len(cards)} 张卡 · {len(assets)} 个 exemplar · {len(missing)} 个缺失")
    if missing:
        print("   缺失：" + ", ".join(sorted(set(missing))))
    return 0


if __name__ == "__main__":
    sys.exit(main())
