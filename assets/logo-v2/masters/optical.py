"""字面率校正：先测量，再按「等墨高 + 等墨距」重排。

中文黑体的四个字在同一字号下，墨迹高度并不相等（「星」比「养」矮，等）。
做法不是拉长字，而是逐个等比缩放到同一墨高，再按墨迹边缘重新排距，
这样四个字看起来一样大、间距也一样。
"""

from pathlib import Path

from fontTools.pens.boundsPen import BoundsPen
from fontTools.pens.transformPen import TransformPen
from fontTools.ttLib import TTFont
from fontTools.varLib.instancer import instantiateVariableFont

FONT_PATH = r"C:\Windows\Fonts\NotoSansSC-VF.ttf"
TEXT = "养成星球"

font = instantiateVariableFont(TTFont(FONT_PATH), {"wght": 700}, inplace=False)
cmap = font.getBestCmap()
gs = font.getGlyphSet()
upm = font["head"].unitsPerEm

print(f"{'字':<3}{'墨宽':>7}{'墨高':>7}{'墨面积比':>9}{'宽高比':>8}")
rows = []
for ch in TEXT:
    name = cmap[ord(ch)]
    pen = BoundsPen(gs)
    gs[name].draw(pen)
    x0, y0, x1, y1 = pen.bounds
    w, h = x1 - x0, y1 - y0
    rows.append((ch, name, x0, y0, x1, y1, w, h))
    print(f"{ch:<3}{w:>7.0f}{h:>7.0f}{w*h/upm/upm:>9.3f}{w/h:>8.3f}")

hs = [r[7] for r in rows]
ws = [r[6] for r in rows]
print(f"\n墨高范围 {min(hs):.0f}–{max(hs):.0f}（差 {max(hs)-min(hs):.0f}，{(max(hs)/min(hs)-1)*100:.1f}%）")
print(f"墨宽范围 {min(ws):.0f}–{max(ws):.0f}（差 {max(ws)-min(ws):.0f}，{(max(ws)/min(ws)-1)*100:.1f}%）")
