import csv
import json
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

BASE = Path(r"D:\yuanhaisu\personal_projects\kindergarten\docs\research\2026-11-02-keyword-logos")
CSV_PATH = BASE.parent / "2026-11-02-keyword-logos.csv"
LOGOS = BASE / "logos"
REPORT = BASE / "download-report.json"

KEYWORD_DIR = {"星球": "planet", "养成": "nurture", "成长": "growth"}
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
    "image/x-icon": ".ico",
    "image/vnd.microsoft.icon": ".ico",
}


def pick_ext(url: str, ctype: str) -> str:
    if ctype in EXT:
        return EXT[ctype]
    suffix = Path(url.split("?")[0]).suffix.lower()
    return suffix if suffix in {".svg", ".png", ".jpg", ".jpeg", ".webp", ".gif", ".ico"} else ".img"


def fetch(row, idx):
    url = (row.get("logo_url") or "").strip()
    keyword = row["keyword"]
    brand = row["brand"]
    if not url:
        return {"keyword": keyword, "brand": brand, "url": "", "file": "", "ok": False, "error": "no logo_url"}
    folder = LOGOS / KEYWORD_DIR.get(keyword, keyword)
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
        with urllib.request.urlopen(req, timeout=25) as resp:
            data = resp.read()
            ctype = (resp.headers.get("Content-Type") or "").split(";")[0].strip().lower()
            ext = pick_ext(url, ctype)
            path = folder / f"{KEYWORD_DIR.get(keyword, keyword)}-{idx:02d}{ext}"
            path.write_bytes(data)
            return {
                "keyword": keyword,
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
            "keyword": keyword,
            "brand": brand,
            "url": url,
            "file": "",
            "ok": False,
            "error": f"{type(exc).__name__}: {exc}",
        }


def main():
    rows = list(csv.DictReader(CSV_PATH.open(encoding="utf-8-sig")))
    LOGOS.mkdir(parents=True, exist_ok=True)
    counters = {k: 0 for k in KEYWORD_DIR}
    results = []
    with ThreadPoolExecutor(max_workers=6) as pool:
        futures = []
        for row in rows:
            counters[row["keyword"]] = counters.get(row["keyword"], 0) + 1
            futures.append(pool.submit(fetch, row, counters[row["keyword"]]))
        for fut in as_completed(futures):
            result = fut.result()
            results.append(result)
            print(("OK " if result["ok"] else "ERR"), result["keyword"], result["brand"])
    results.sort(key=lambda r: (r["keyword"], r["brand"]))
    REPORT.write_text(json.dumps(results, ensure_ascii=False, indent=2), encoding="utf-8")
    ok = sum(1 for r in results if r["ok"])
    print(f"downloaded {ok}/{len(results)}; report: {REPORT}")


if __name__ == "__main__":
    main()
