import csv
import json
import ssl
import urllib.request
from pathlib import Path

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\docs\research\2026-11-02-kindergarten-logo-landscape")
CSV_PATH = BASE.parent / "2026-11-02-kindergarten-logo-landscape.csv"
REPORT = BASE / "download-report.json"

rows = list(csv.DictReader(CSV_PATH.open(encoding="utf-8-sig")))
idx = next(i for i, r in enumerate(rows, 1) if r["brand"] == "幼视通")
row = rows[idx - 1]
req = urllib.request.Request(
    row["logo_url"],
    headers={
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/124.0 Safari/537.36",
        "Referer": row["site_url"],
        "Accept": "image/*,*/*;q=0.8",
    },
)
data = urllib.request.urlopen(req, context=ssl._create_unverified_context(), timeout=20).read()
out = BASE / "logos" / "L1-saas" / f"L1-{idx:02d}.png"
out.parent.mkdir(parents=True, exist_ok=True)
out.write_bytes(data)

report = json.loads(REPORT.read_text(encoding="utf-8"))
for item in report:
    if item["brand"] == "幼视通":
        item.update(
            ok=True,
            file=str(out.relative_to(BASE)),
            status=200,
            content_type="image/png",
            bytes=len(data),
        )
        item.pop("error", None)
REPORT.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
print(out, len(data))
