import csv
import json
import re
import ssl
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\docs\research\2026-11-02-kindergarten-logo-landscape")
CSV_PATH = BASE.parent / "2026-11-02-kindergarten-logo-landscape.csv"
LOGOS = BASE / "logos"
REPORT = BASE / "download-report.json"

LAYER_DIR = {"L1": "L1-saas", "L2": "L2-kindergarten", "L3": "L3-content", "L4": "L4-international"}
UA = (
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 "
    "(KHTML, like Gecko) Chrome/124.0 Safari/537.36"
)
EXT = {
    "image/svg+xml": ".svg",
    "image/png": ".png",
    "image/jpeg": ".jpg",
    "image/webp": ".webp",
    "image/gif": ".gif",
}


def pick_ext(url: str, ctype: str) -> str:
    if ctype in EXT:
        return EXT[ctype]
    suffix = Path(url.split("?")[0]).suffix.lower()
    return suffix if suffix in {".svg", ".png", ".jpg", ".jpeg", ".webp", ".gif"} else ".img"


def fetch(row, idx):
    url = (row.get("logo_url") or "").strip()
    layer = row["layer"]
    brand = row["brand"]
    if not url:
        return {"layer": layer, "brand": brand, "url": "", "file": "", "ok": False, "error": "no logo_url"}
    folder = LOGOS / LAYER_DIR.get(layer, layer)
    folder.mkdir(parents=True, exist_ok=True)
    req = urllib.request.Request(
        url,
        headers={
            "User-Agent": UA,
            "Accept": "image/avif,image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8",
            "Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
            "Referer": row.get("site_url") or url,
        },
    )
    try:
        with urllib.request.urlopen(req, timeout=20) as resp:
            data = resp.read()
            ctype = (resp.headers.get("Content-Type") or "").split(";")[0].strip().lower()
            ext = pick_ext(url, ctype)
            path = folder / f"{layer}-{idx:02d}{ext}"
            path.write_bytes(data)
            return {
                "layer": layer,
                "brand": brand,
                "url": url,
                "file": str(path.relative_to(BASE)),
                "ok": True,
                "status": resp.status,
                "content_type": ctype,
                "bytes": len(data),
            }
    except Exception as exc:
        return {
            "layer": layer,
            "brand": brand,
            "url": url,
            "file": "",
            "ok": False,
            "error": f"{type(exc).__name__}: {exc}",
        }


def main():
    rows = list(csv.DictReader(CSV_PATH.open(encoding="utf-8-sig")))
    LOGOS.mkdir(parents=True, exist_ok=True)
    results = []
    with ThreadPoolExecutor(max_workers=6) as pool:
        futures = [pool.submit(fetch, row, i) for i, row in enumerate(rows, 1)]
        for fut in as_completed(futures):
            result = fut.result()
            results.append(result)
            flag = "OK " if result["ok"] else "ERR"
            print(f"{flag} {result['layer']} {result['brand']}")
    results.sort(key=lambda r: (r["layer"], r["brand"]))
    REPORT.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
    ok = sum(1 for r in results if r["ok"])
    print(f"downloaded {ok}/{len(results)}; report: {REPORT}")


if __name__ == "__main__":
    main()
