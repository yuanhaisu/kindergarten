"""站 2 渲染器：15 句意象，校验 + 出可指认的卡片页。

这个脚本**不生成句子**。15 句「这是一个 ______」是创作，只能由人（或 AI）写；
脚本的职责是让不合格的句子进不来，以及把指认结果记下来。

一条句子要过四关：

1. ≤ 20 字
2. 不含抽象词——句子在说形，不在说意义
3. `shape` / `relation` / `join` 三段都填得出来
4. 五个词各出三句，一句不少

第 3 关是从 Airbnb 用例补的硬规则：

    像什么  →  「一颗心被拉成房子」        ❌ 不能直接构造
    怎么连  →  「一条线绕成 A，中途抱住心」 ✅ 能直接构造

形（画什么）+ 位（放在哪）+ 连（怎么接），三样齐了才是可构造的句子；
缺一样，站 4 就会开始自由发挥——那是「坐标对、看着不对」的来源。

词库格式见 examples/养成星球.words.json。

用法：
    python -I scripts/draft.py --brand 养成星球
    python -I scripts/draft.py --brand X --words path/to/x.words.json
"""

from __future__ import annotations

import argparse
import html
import json
import sys
from pathlib import Path

for _s in (sys.stdout, sys.stderr):
    if hasattr(_s, "reconfigure"):
        _s.reconfigure(encoding="utf-8")

SKILL = Path(__file__).resolve().parent.parent

MAX_LEN = 20

# 出现即作废：这些词描述意义，不描述形
ABSTRACT = [
    "成长", "未来", "关怀", "生态", "温暖", "陪伴", "理念", "价值", "赋能",
    "连接", "共育", "美好", "幸福", "梦想", "智能", "专业", "高端", "国际化",
    "品质", "生活方式", "情感", "体验", "守护", "呵护", "希望", "责任", "传承",
]

SELECTION_JS = """
(function () {
  const cards = Array.from(document.querySelectorAll('.card'));
  const picked = new Set();
  const out = document.getElementById('out');
  const count = document.getElementById('count');
  const hint = document.getElementById('hint');
  const copyBtn = document.getElementById('copy');

  function render() {
    cards.forEach(c => c.classList.toggle('picked', picked.has(c.dataset.id)));
    const ids = Array.from(picked).sort();
    out.value = ids.join(',');
    count.textContent = ids.length;

    let msg = '选 3 句，且必须来自至少 2 个不同的词';
    let warn = false;
    if (ids.length >= 2) {
      const words = cards.filter(c => picked.has(c.dataset.id)).map(c => c.dataset.word);
      if (new Set(words).size === 1) {
        msg = '三句都在同一个词下 —— 这其实是同一个想法';
        warn = true;
      } else if (ids.length === 3) {
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


def validate(bank: dict) -> tuple[list[dict], list[str]]:
    """返回 (合格句子, 失败原因)。有任何失败就不出页面。"""
    failures: list[str] = []
    out: list[dict] = []

    words = bank.get("words") or []
    if len(words) != 5:
        failures.append(f"词库必须是 5 个词，现在是 {len(words)} 个")

    for w in words:
        objs = w.get("objects") or []
        if len(objs) != 3:
            failures.append(f"「{w.get('word')}」必须给 3 个具象，现在是 {len(objs)} 个")

    for s in bank.get("sentences") or []:
        sid = s.get("id", "?")
        text = (s.get("sentence") or "").strip()

        if not text:
            failures.append(f"{sid}：句子为空")
            continue
        if len(text) > MAX_LEN:
            failures.append(f"{sid}：{len(text)} 字，超过 {MAX_LEN} —— 「{text}」")
        hit = [w for w in ABSTRACT if w in text]
        if hit:
            failures.append(f"{sid}：在说意义不在说形（{', '.join(hit)}）—— 「{text}」")
        for field, zh in (("shape", "形"), ("relation", "位"), ("join", "连")):
            if not (s.get(field) or "").strip():
                failures.append(f"{sid}：缺「{zh}」，不可构造 —— 「{text}」")
        out.append(s)

    expect = sum(len(w.get("objects") or []) for w in words)
    if out and len(bank.get("sentences") or []) != expect:
        failures.append(f"应有 {expect} 句，实际 {len(bank.get('sentences') or [])} 句")

    return out, failures


def build_page(bank: dict, sentences: list[dict]) -> str:
    by_word: dict[str, list[dict]] = {}
    for s in sentences:
        by_word.setdefault(s["word"], []).append(s)

    groups = []
    for w in bank["words"]:
        word = w["word"]
        cards = []
        for s in by_word.get(word, []):
            cards.append(f"""
      <article class="card" data-id="{s['id']}" data-word="{html.escape(s['word'])}"
               data-object="{html.escape(s['object'])}">
        <div class="sid">{s['id']}</div>
        <div class="tick">✓</div>
        <p class="sentence">{html.escape(s['sentence'])}</p>
        <div class="parts">
          <span><i>形</i>{html.escape(s['shape'])}</span>
          <span><i>位</i>{html.escape(s['relation'])}</span>
          <span><i>连</i>{html.escape(s['join'])}</span>
        </div>
        <div class="obj">物件：{html.escape(s['object'])}</div>
      </article>""")
        groups.append(f"""
  <section class="group">
    <h2>{html.escape(word)}<span>{html.escape(w.get('kind', ''))}</span></h2>
    <div class="row">{''.join(cards)}</div>
  </section>""")

    brief = bank.get("brief") or {}
    brand = bank.get("brand", "")

    return f"""<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{html.escape(brand)} · 站 2 意象</title>
<style>
  :root {{
    --ink: #16181D; --muted: #6B6F76; --line: #E5DFD2;
    --paper: #FAF8F3; --card: #FFFFFF;
  }}
  * {{ box-sizing: border-box; }}
  body {{
    margin: 0; padding: 48px 24px 116px;
    background: var(--paper); color: var(--ink);
    font-family: -apple-system, "Segoe UI", "Microsoft YaHei", sans-serif;
    -webkit-font-smoothing: antialiased;
  }}
  header {{ max-width: 1120px; margin: 0 auto 34px; }}
  h1 {{ margin: 0 0 10px; font-size: 22px; font-weight: 600; }}
  header p {{ margin: 0 0 8px; font-size: 13px; line-height: 1.75; color: var(--muted); max-width: 680px; }}
  header b {{ color: var(--ink); font-weight: 600; }}
  .brief {{ font-size: 12.5px; color: #9A948A; }}
  .group {{ max-width: 1120px; margin: 0 auto 26px; }}
  .group h2 {{
    margin: 0 0 10px; font-size: 13px; font-weight: 600; letter-spacing: .06em;
    color: var(--muted); text-transform: none;
  }}
  .group h2 span {{ margin-left: 8px; font-weight: 400; color: #B9B3A6; }}
  .row {{ display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 14px; }}
  .card {{
    position: relative; background: var(--card);
    border: 1px solid var(--line); border-radius: 16px;
    padding: 22px 20px 16px; cursor: pointer; user-select: none;
    display: flex; flex-direction: column; gap: 12px;
    transition: border-color .15s, box-shadow .15s;
  }}
  .card:hover {{ border-color: #CFC7B6; }}
  .card.picked {{ border-color: var(--ink); box-shadow: 0 0 0 1px var(--ink); }}
  .sid {{
    position: absolute; top: 12px; right: 16px;
    font-size: 10.5px; letter-spacing: .1em; color: #B9B3A6;
    font-variant-numeric: tabular-nums;
  }}
  .tick {{
    position: absolute; top: 9px; right: 12px;
    width: 20px; height: 20px; border-radius: 50%;
    background: var(--ink); color: #fff; font-size: 11px;
    display: none; align-items: center; justify-content: center;
  }}
  .card.picked .tick {{ display: flex; }}
  .card.picked .sid {{ display: none; }}
  .sentence {{
    margin: 0; font-size: 16.5px; line-height: 1.65; letter-spacing: .01em;
    min-height: 3.3em;
  }}
  .parts {{ display: flex; flex-direction: column; gap: 4px; }}
  .parts span {{ font-size: 11.5px; line-height: 1.5; color: var(--muted); }}
  .parts i {{
    display: inline-block; width: 16px; margin-right: 6px;
    font-style: normal; font-size: 10px; color: #B9B3A6;
  }}
  .obj {{
    margin-top: auto; padding-top: 10px; border-top: 1px solid #F0ECE3;
    font-size: 11px; color: #9A948A;
  }}
  .bar {{
    position: fixed; left: 0; right: 0; bottom: 0; z-index: 10;
    background: rgba(250,248,243,.95); backdrop-filter: blur(10px);
    border-top: 1px solid var(--line); padding: 14px 24px;
  }}
  .bar .inner {{ max-width: 1120px; margin: 0 auto; display: flex; align-items: center; gap: 14px; flex-wrap: wrap; }}
  .bar .num {{ font-size: 13px; color: var(--muted); white-space: nowrap; }}
  .bar .num b {{ color: var(--ink); font-variant-numeric: tabular-nums; font-size: 15px; }}
  .bar .hint {{ font-size: 12.5px; color: var(--muted); flex: 1; min-width: 200px; }}
  .bar .hint.warn {{ color: #B45309; }}
  .bar input {{
    width: 170px; font-family: ui-monospace, "SF Mono", Consolas, monospace;
    font-size: 13px; padding: 8px 10px; text-align: center;
    border: 1px solid var(--line); border-radius: 8px; background: #fff; color: var(--ink);
  }}
  .bar button {{
    padding: 8px 14px; font-size: 12.5px; border-radius: 8px;
    border: 1px solid var(--ink); background: var(--ink); color: #fff; cursor: pointer; white-space: nowrap;
  }}
</style>
</head>
<body>
<header>
  <h1>{html.escape(brand)}</h1>
  <p>
    <b>站 2 · 意象。</b>15 句「这是一个 ___」，分成 5 组。选 <b>3 句</b>你愿意指着说「就是这个」的。
    先只读字，不要想象它长什么样——如果一句话需要你脑补，它就不该被选中。
  </p>
  <p>三句必须来自 <b>至少 2 个不同的词</b>。三句都出自同一个词，说明你其实只有一个想法。</p>
  <p class="brief">给谁：{html.escape(brief.get('who', '—'))}　·　解决什么：{html.escape(brief.get('problem', '—'))}</p>
</header>
{''.join(groups)}
<div class="bar">
  <div class="inner">
    <span class="num">已选 <b id="count">0</b> / 3</span>
    <span class="hint" id="hint">选 3 句，且必须来自至少 2 个不同的词</span>
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
    ap.add_argument("--words", type=Path, help="词库 JSON，默认 runs/<品牌>/words.json")
    ap.add_argument("--out", type=Path)
    args = ap.parse_args()

    run_dir = SKILL / "runs" / args.brand
    src = args.words or (run_dir / "words.json")
    if not src.exists():
        print(f"❌ 找不到词库：{src}")
        print("   站 2 的句子需要人工/AI 先写成语库，脚本不生成它们。")
        return 1

    bank = json.loads(src.read_text(encoding="utf-8"))
    sentences, failures = validate(bank)

    if failures:
        print(f"❌ {len(failures)} 处不合格，未生成页面：\n")
        for f in failures:
            print("  -", f)
        return 1

    out = args.out or (run_dir / "round-2.html")
    out.parent.mkdir(parents=True, exist_ok=True)
    out.write_text(build_page(bank, sentences), encoding="utf-8")

    (run_dir / "round-2.json").write_text(
        json.dumps({"brand": bank.get("brand"), "brief": bank.get("brief"),
                    "words": bank.get("words"), "sentences": sentences},
                   ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )

    print(f"✅ {out}")
    print(f"   {len(sentences)} 句 · {len(bank['words'])} 组")
    return 0


if __name__ == "__main__":
    sys.exit(main())
