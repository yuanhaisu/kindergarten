# 老师端入口图标（位图）

8 个功能入口的插画图标，**统一暖橙渐变**。替代原先「中文单字 + 8 种底色」的方案：
用同一套视觉让入口看起来是一家人，语义区分交给图形本身。

- 来源：SiliconFlow `Tongyi-MAI/Z-Image-Turbo` 生成（2026-10-25）+ 本地后处理
- 尺寸：144px（视网膜 2x）/ 96px（1x），8 张合计约 176KB
- ⚠️ **位图不随主题换色**。改品牌色需要重新生成，或换回矢量图标字体

## 生成 prompt 模板

所有图标共用同一段风格描述，只替换 `{SUBJECT}`：

```text
A single app icon of {SUBJECT}, centered on a pure white square background with
generous padding. Soft rounded friendly style for a kindergarten parent app:
thick rounded strokes, no sharp corners, solid filled shapes with NO outline
stroke, all filled with the same warm amber-orange gradient (#FFA866 to #FF8A3D),
with a soft cream halo behind it. Flat vector illustration look, minimal detail
so it stays readable when shrunk to 24 pixels. No text, no letters, no
background patterns. Clean, gentle, trustworthy, like a warm sunlit kindergarten.
```

| 文件 | 入口 | `{SUBJECT}` |
|---|---|---|
| `notice.png` | 发布通知 | a notification bell |
| `growth.png` | 成长记录 | a young sprout with two rounded leaves growing from a small mound |
| `pickup.png` | 接送管理 | a small rounded car seen from the side, two round wheels |
| `task.png` | 亲子任务 | one rounded five-point star with softly blunted tips |
| `calendar.png` | 班级日历 | a calendar page with two binding rings and a row of day dots |
| `leave.png` | 请假审批 | a sheet of paper with a rounded check mark |
| `album.png` | 班级相册 | two overlapping rounded photo cards with a mountain and sun dot |
| `more.png` | 更多 | a 2 by 2 grid of four rounded squares |

## 后处理（两步，必须做）

模型有稳定行为：**无视「no card」指令**，每张都画白色圆角卡片 + 投影；且各图内容
占比在 **55%~80%** 之间浮动（嫩芽 55%、日历 80%），直接使用会大小不一。

1. **抠底**：从图片四边泛洪，把「中性浅色」像素（`max-min < 14` 且 `min > 195`，
   即白底与灰投影）的 alpha 置 0。只从边界扩散，因此图标内部的白色区域不受影响。
2. **归一化**：裁到 alpha 边界，等比缩放到画布 82%，居中 —— 统一光学尺寸。

```python
from PIL import Image
from collections import deque

def key_out(im):
    im = im.convert('RGBA'); w, h = im.size; px = im.load()
    seen = bytearray(w * h); q = deque()
    for x in range(w): q.extend(((x, 0), (x, h - 1)))
    for y in range(h): q.extend(((0, y), (w - 1, y)))
    while q:
        x, y = q.popleft()
        if not (0 <= x < w and 0 <= y < h): continue
        i = y * w + x
        if seen[i]: continue
        seen[i] = 1
        r, g, b, a = px[x, y]
        if not (max(r, g, b) - min(r, g, b) < 14 and min(r, g, b) > 195): continue
        px[x, y] = (r, g, b, 0)
        q.extend(((x + 1, y), (x - 1, y), (x, y + 1), (x, y - 1)))
    return im

def normalize(im, canvas=512, fill=0.82):
    glyph = im.crop(im.split()[3].getbbox())
    s = canvas * fill / max(glyph.size)
    glyph = glyph.resize((round(glyph.width * s), round(glyph.height * s)), Image.LANCZOS)
    out = Image.new('RGBA', (canvas, canvas), (0, 0, 0, 0))
    out.paste(glyph, ((canvas - glyph.width) // 2, (canvas - glyph.height) // 2), glyph)
    return out
```

## 已知遗留

- 残留的米色光圈（烘焙在图里）在浅底格子上会留下很淡的一圈
- 描边派（日历 / 请假 / 相册）比实心派（铃铛 / 嫩芽 / 小车 / 网格）视觉重量轻一档
- 若要修这两点，需重新生成或换成矢量图标体系
