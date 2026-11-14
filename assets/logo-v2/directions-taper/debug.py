import re
from pathlib import Path

s = Path("taper-a.svg").read_text(encoding="utf-8")
d = re.search(r'd="([^"]+)"', s).group(1)
print("len", len(d))
print(d[:500])
print("...")
print(d[-260:])
