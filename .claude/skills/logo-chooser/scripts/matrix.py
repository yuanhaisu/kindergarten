"""站 3 渲染器：类型试穿矩阵。

3 句 × 5 类 = 15 格。同一个想法，用五种方式做出来，问「哪种最像你们」。

这个脚本**不把句子画成形**。站 3 的符号只到「示意」级别——它是用来判断
「这个想法适合哪种类型」的，不是用来判断画得好不好的。真正的构造在站 4。

所以每句需要两份手绘示意，放在 runs/<品牌>/sketches/：

    S07.svg           具象：把物件画出来的样子
    S07.abstract.svg  抽象：只留几何关系的样子（可缺省）

缺 abstract 时复用具象，并在格子上标注「无更简版」——那本身是个有用的信号。

类型只有 5 列。8 种里砍掉 emblem 和 mascot：16px 基本报废，除非用户明确
要机构感或要角色。字母列对中文品牌用首字代替（中文没有字母组合）。

用法：
    python -I scripts/matrix.py --brand 养成星球
    python -I scripts/matrix.py --brand 养成星球 --ids S07,S01,S13
"""

from __future__ import annotations

import argparse
import html
import json
import re
import sys
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

SKILL = Path(__file__).resolve().parent.parent
AXES = SKILL / "axes.json"

sys.path.insert(0, str(Path(__file__).resolve().parent))
from typeset import Face, typeset  # noqa: E402
from coverage_check import require  # noqa: E402
import picks  # noqa: E402
import survive  # noqa: E402

def load_types() -> list[dict]:
    """类型清单只在 axes.json，渲染器和记录器读同一份。"""
    spec = json.loads(AXES.read_text(encoding="utf-8"))
    types = spec["station3"]["types"]

    # 砍掉一个类型可以，但必须申报理由——未申报的裁剪会把正确答案挡在门外
    require(spec, "mark_type", [t["id"] for t in types],
            allow_cut=spec["station3"].get("cut"), where="站 3 · 类型")

    return types


def _badge_measured(m: dict, kind: str = "符号") -> str:
    a, b = m["small"], m["ref"]
    tip = (f"{kind} 16px：墨 {a['ink']:.1%} · {a['components']} 块 · {a['holes']} 洞\n"
           f"{kind} 96px：墨 {b['ink']:.1%} · {b['components']} 块 · {b['holes']} 洞")
    if m["reasons"]:
        tip += "\n" + "\n".join(m["reasons"])
    return f'<span title="{html.escape(tip)}">16px {m["verdict"]}·{a["ink"]:.0%}</span>'


def _badge_text() -> str:
    return ('<span class="na" title="16px 上文字不可读；这类标志在符号位需要另做简化版">'
            '16px 不适用</span>')

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

    let msg = '选 2–3 格，且至少跨 2 种类型';
    let warn = false;
    if (ids.length >= 2) {
      const types = cells.filter(c => picked.has(c.dataset.id)).map(c => c.dataset.type);
      if (new Set(types).size === 1) {
        msg = '都在同一种类型里 —— 那样没有在比较类型';
        warn = true;
      } else if (ids.length >= 2) {
        msg = '可以了 —— 把编号发回对话';
      }
    }
    hint.textContent = msg;
    hint.classList.toggle('warn', warn);
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


def read_svg(path: Path, cls: str = "") -> str:
    """读一张示意 SVG，剥掉 XML 声明与固定宽高，交给 CSS 控制尺寸。"""
    text = path.read_text(encoding="utf-8").strip()
    if text.startswith("<?xml"):
        text = text.split("?>", 1)[1].strip()
    for attr in ('width="', 'height="'):
        i = text.find(attr)
        if i != -1 and i < text.find(">"):
            j = text.index('"', i + len(attr))
            text = text[:i] + text[j + 2:]
    if cls:
        text = text.replace("<svg", f'<svg class="{cls}"', 1)
    return text


def build_page(brand: str, rows: list[dict], run_dir: Path, fonts: dict,
               wordmark: dict, mask_name: bool = False) -> str:
    types = load_types()
    default_font = fonts["noto-sans-sc"]["file"]
    # 站 1 锁定的字重与字距必须真的被用上，否则上游的选择等于没发生过
    face = Face(default_font, wordmark["wght"])
    name_svg = typeset(brand, face, size=100, tracking=wordmark["tracking"])
    mono_svg = typeset(brand[0], Face(default_font, 900), size=100, tracking=0.0)

    # 变体：组合列的名字换成灰色占位块。
    # 不能直接删掉——删了就与具象列完全重复，测的就不是「看到自己名字的吸引力」，
    # 而是「两列一模一样时怎么选」。
    if mask_name:
        vb = re.search(r'viewBox="0 0 ([\d.]+) ([\d.]+)"', name_svg)
        nw, nh = float(vb.group(1)), float(vb.group(2))
        combo_name = f'<div class="namebar" style="width:{26 * nw / nh:.0f}px"></div>'
    else:
        combo_name = name_svg

    # 字标不随想法变化：它只画名字。做成全局卡，而不是每行一格——
    # 否则三行字标逐像素相同，用户点哪行纯属偶然，编号里的行号是假的。
    wordmark_card = f"""
  <section class="global">
    <div class="glabel"><b>只用名字</b><span>不随哪个想法变化——这是品牌名本身</span></div>
    <button class="cell wide" data-id="wordmark" data-type="wordmark" data-sid="">
      <div class="tick">✓</div>
      <div class="stage text">{name_svg}</div>
      <div class="badges">{_badge_text()}<span>文字 强</span></div>
    </button>
  </section>"""

    body_rows = []
    for s in rows:
        sid = s["id"]
        pic_svg = read_svg(run_dir / "sketches" / f"{sid}.svg", "sym")
        m_pic = survive.measure(pic_svg)

        cells = []
        for t in types:
            key = t["id"]
            if key == "wordmark":
                continue
            if key == "pictorial":
                stage = f'<div class="stage">{pic_svg}</div>'
                badge = _badge_measured(m_pic)
            else:
                stage = f'<div class="stage combo">{pic_svg}{combo_name}</div>'
                badge = _badge_measured(m_pic, "组合里的符号")

            text_badge = f'<span>文字 {"强" if t["text"] else "弱"}</span>'

            cells.append(f"""
        <button class="cell" data-id="{sid}-{key}" data-type="{key}" data-sid="{sid}">
          <div class="tick">✓</div>
          {stage}
          <div class="badges">{badge}{text_badge}</div>
        </button>""")

        body_rows.append(f"""
  <section class="row">
    <header class="rowhead">
      <div class="sid">{sid}</div>
      <p class="sentence">{html.escape(s['sentence'])}</p>
      <div class="parts">
        <span>形 {html.escape(s['shape'])}</span>
        <span>位 {html.escape(s['relation'])}</span>
        <span>连 {html.escape(s['join'])}</span>
      </div>
    </header>
    <div class="cells">{''.join(cells)}</div>
  </section>""")

    heads = "".join(
        f'<div class="colhead"><b>{html.escape(t["zh"])}</b>'
        f'<span>{html.escape(t["plain"])}</span></div>'
        for t in types if t["id"] != "wordmark"
    )

    return f"""<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(brand)} · 站 3 类型试穿</title>
<style>
  :root {{
    --ink: #16181D; --muted: #6B6F76; --line: #E5DFD2;
    --paper: #FAF8F3; --card: #FFFFFF;
  }}
  * {{ box-sizing: border-box; }}
  body {{
    margin: 0; padding: 44px 24px 116px;
    background: var(--paper); color: var(--ink);
    font-family: -apple-system, "Segoe UI", "Microsoft YaHei", sans-serif;
    -webkit-font-smoothing: antialiased;
  }}
  header.top {{ max-width: 1180px; margin: 0 auto 30px; }}
  h1 {{ margin: 0 0 10px; font-size: 22px; font-weight: 600; }}
  header.top p {{ margin: 0 0 6px; font-size: 13px; line-height: 1.75; color: var(--muted); max-width: 720px; }}
  header.top b {{ color: var(--ink); font-weight: 600; }}
  header.top .lock {{
    display: inline-block; margin-top: 4px; padding: 3px 9px; border-radius: 6px;
    background: #F1EDE3; color: #8C8578; font-size: 11.5px;
  }}
  .matrix {{ max-width: 1180px; margin: 0 auto; }}
  /* 列头必须和行用同一套栅格，否则越往右错得越多 */
  .cols {{
    display: grid; grid-template-columns: 250px 1fr; gap: 12px;
    padding: 0 0 8px; border-bottom: 1px solid var(--line); margin-bottom: 12px;
  }}
  .cellheads {{
    display: grid; grid-template-columns: repeat(2, minmax(0, 360px));
    gap: 12px; justify-content: start;
  }}
  .global {{
    max-width: 1180px; margin: 0 auto 22px;
    display: grid; grid-template-columns: 250px 1fr; gap: 12px;
    padding-bottom: 22px; border-bottom: 1px solid var(--line);
  }}
  .glabel {{ padding: 6px 4px 0; }}
  .glabel b {{ display: block; font-size: 13px; font-weight: 600; }}
  .glabel span {{ display: block; margin-top: 4px; font-size: 11px; line-height: 1.5; color: #9A948A; }}
  .cell.wide {{ max-width: 340px; }}
  .colhead {{ padding-left: 4px; }}
  .colhead b {{ display: block; font-size: 13px; font-weight: 600; }}
  .colhead span {{ display: block; margin-top: 2px; font-size: 11px; color: #9A948A; }}
  .row {{
    display: grid; grid-template-columns: 250px 1fr; gap: 12px;
    padding: 14px 0; border-top: 1px solid #F0ECE3;
  }}
  .row:first-of-type {{ border-top: none; }}
  .rowhead {{ padding: 6px 8px 0 4px; }}
  .sid {{ font-size: 10.5px; letter-spacing: .1em; color: #B9B3A6; }}
  .sentence {{ margin: 6px 0 10px; font-size: 14.5px; line-height: 1.6; }}
  .parts {{ display: flex; flex-direction: column; gap: 3px; }}
  .parts span {{ font-size: 11px; line-height: 1.45; color: var(--muted); }}
  .cells {{
    display: grid; grid-template-columns: repeat(2, minmax(0, 360px));
    gap: 12px; justify-content: start;
  }}
  .cell {{
    position: relative; background: var(--card); border: 1px solid var(--line);
    border-radius: 14px; padding: 14px 10px 10px; cursor: pointer;
    display: flex; flex-direction: column; align-items: center; gap: 10px;
    font: inherit; color: inherit; transition: border-color .15s, box-shadow .15s;
  }}
  .cell:hover {{ border-color: #CFC7B6; }}
  .cell.picked {{ border-color: var(--ink); box-shadow: 0 0 0 1px var(--ink); }}
  .tick {{
    position: absolute; top: 7px; right: 7px; width: 18px; height: 18px;
    border-radius: 50%; background: var(--ink); color: #fff; font-size: 10px;
    display: none; align-items: center; justify-content: center;
  }}
  .cell.picked .tick {{ display: flex; }}
  .stage {{
    height: 76px; display: flex; align-items: center; justify-content: center; gap: 8px;
  }}
  .stage svg {{ height: 76px; width: auto; max-width: 100%; }}
  .stage.text svg {{ height: 30px; }}
  .stage.mono svg {{ height: 62px; }}
  .stage.combo {{ gap: 10px; }}
  .stage.combo svg.sym {{ height: 48px; }}
  .stage.combo svg:not(.sym) {{ height: 26px; }}
  .namebar {{
    height: 13px; border-radius: 3px; background: #DED7C8;
  }}
  .badges {{ display: flex; gap: 6px; flex-wrap: wrap; justify-content: center; }}
  .badges span {{
    font-size: 10px; color: #9A948A; background: #F6F3EC;
    padding: 2px 6px; border-radius: 5px; white-space: nowrap;
  }}
  .note {{ font-size: 10px; color: #B45309; }}
  .bar {{
    position: fixed; left: 0; right: 0; bottom: 0; z-index: 10;
    background: rgba(250,248,243,.95); backdrop-filter: blur(10px);
    border-top: 1px solid var(--line); padding: 14px 24px;
  }}
  .bar .inner {{ max-width: 1180px; margin: 0 auto; display: flex; align-items: center; gap: 14px; flex-wrap: wrap; }}
  .bar .num {{ font-size: 13px; color: var(--muted); white-space: nowrap; }}
  .bar .num b {{ color: var(--ink); font-variant-numeric: tabular-nums; font-size: 15px; }}
  .bar .hint {{ font-size: 12.5px; color: var(--muted); flex: 1; min-width: 200px; }}
  .bar .hint.warn {{ color: #B45309; }}
  .bar input {{
    width: 310px; font-family: ui-monospace, "SF Mono", Consolas, monospace;
    font-size: 12px; padding: 8px 10px; text-align: center;
    border: 1px solid var(--line); border-radius: 8px; background: #fff; color: var(--ink);
  }}
  .bar button {{
    padding: 8px 14px; font-size: 12.5px; border-radius: 8px;
    border: 1px solid var(--ink); background: var(--ink); color: #fff; cursor: pointer; white-space: nowrap;
  }}
</style>
</head>
<body>
<header class="top">
  <h1>{html.escape(brand)}</h1>
  <p>
    <b>站 3 · 类型试穿。</b>每个想法只有两种形态：<b>只用图形</b>、<b>图形 + 名字</b>；
    上面另有一张<b>只用名字</b>的全局卡。这就是真实世界里标志的全部选择。
    选 <b>2–3 格</b>，且至少跨 <b>2 种类型</b>——全在同一种类型里，就没在比较类型。
  </p>
  <p>格子里的符号只到示意级别，用来判断「这个想法适不适合这种类型」，不是终稿。</p>
  <p class="lock">字标按站 1 的第一个选择排：wght {wordmark['wght']:g} · 字距 {wordmark['tracking']:+.0%}</p>
</header>
{wordmark_card}
<div class="matrix">
  <div class="cols"><div></div><div class="cellheads">{heads}</div></div>
  {''.join(body_rows)}
</div>
<div class="bar">
  <div class="inner">
    <span class="num">已选 <b id="count">0</b> / 3</span>
    <span class="hint" id="hint">选 2–3 格，且至少跨 2 种类型</span>
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
    ap.add_argument("--ids", help="句子编号，逗号分隔；默认读 picks.json 的站 2 选择")
    ap.add_argument("--wght", type=float, help="覆盖站 1 锁定的字重")
    ap.add_argument("--tracking", type=float, help="覆盖站 1 锁定的字距")
    ap.add_argument("--mask-name", action="store_true",
                    help="组合列的名字换成灰色占位块——检验「组合列看起来更完整」是不是假偏好")
    ap.add_argument("--out", type=Path)
    args = ap.parse_args()

    run_dir = SKILL / "runs" / args.brand
    bank_path = run_dir / "round-2.json"
    if not bank_path.exists():
        print(f"❌ 找不到 {bank_path}，先跑 scripts/draft.py")
        return 1
    bank = json.loads(bank_path.read_text(encoding="utf-8"))
    by_id = {s["id"]: s for s in bank["sentences"]}

    data = picks.load(run_dir / "picks.json")

    if args.ids:
        ids = [x.strip() for x in args.ids.replace("，", ",").split(",") if x.strip()]
        print("⚠️  --ids 绕过了站 2 的记录：这次产出的矩阵不在依赖链上，不会让任何记录失效")
    else:
        entry = picks.active(data, 2)
        if entry is None:
            print("❌ 没有有效的站 2 选择（可能已被标记为过期），先跑 narrow.py --station 2")
            return 1
        ids = [p["id"] for p in entry["picks"]]

    unknown = [i for i in ids if i not in by_id]
    if unknown:
        print(f"❌ 没有这些句子：{unknown}")
        return 1
    if len(ids) != 3:
        print(f"❌ 站 3 需要正好 3 句，现在 {len(ids)} 句")
        return 1

    rows = [by_id[i] for i in ids]
    missing = [s["id"] for s in rows if not (run_dir / "sketches" / f"{s['id']}.svg").exists()]
    if missing:
        print("❌ 缺少示意 SVG：")
        for m in missing:
            print(f"   runs/{args.brand}/sketches/{m}.svg")
        print("   站 3 不画形——示意需要先由人（或 AI）画好。")
        return 1

    spec = json.loads(AXES.read_text(encoding="utf-8"))
    fonts = {f["id"]: f for f in spec["zh"]["fonts"]}

    # 站 1 锁定的字重/字距，下游必须消费掉
    wordmark = {"wght": 500.0, "tracking": 0.02}
    lock = picks.active(data, 1)
    if lock and lock["picks"]:
        p = lock["picks"][0]
        wordmark = {"wght": float(p["wght"]), "tracking": float(p.get("tracking", 0.0))}
    if args.wght is not None:
        wordmark["wght"] = args.wght
    if args.tracking is not None:
        wordmark["tracking"] = args.tracking

    out = args.out or (run_dir / ("round-3-masked.html" if args.mask_name else "round-3.html"))
    out.write_text(build_page(args.brand, rows, run_dir, fonts, wordmark,
                              mask_name=args.mask_name), encoding="utf-8")

    # 给 narrow.py 读的机器可读副本：它需要知道这一轮到底摆了几行几列、是哪个变体
    (run_dir / "round-3.json").write_text(json.dumps({
        "brand": args.brand,
        "variant": "masked" if args.mask_name else "full",
        "types": [t["id"] for t in load_types()],
        "sentences": rows,
    }, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")

    n_types = len(load_types())
    cells = 1 + len(rows) * (n_types - 1)   # 字标是全局卡，不占行
    print(f"✅ {out}")
    print(f"   1 张全局字标 + {len(rows)} 句 × {n_types - 1} 形态 = {cells} 格")
    return 0


if __name__ == "__main__":
    sys.exit(main())
