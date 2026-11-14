import sys
import xlrd

path, out, limit = sys.argv[1], sys.argv[2], int(sys.argv[3])
wb = xlrd.open_workbook(path, formatting_info=False)
lines = ["sheets: %r" % (wb.sheet_names(),)]
for sh in wb.sheets():
    lines.append("=" * 60)
    lines.append("SHEET %r rows=%d cols=%d" % (sh.name, sh.nrows, sh.ncols))
    for r in range(min(sh.nrows, limit)):
        vals = []
        for c in range(sh.ncols):
            v = sh.cell_value(r, c)
            if isinstance(v, float) and v == int(v):
                v = int(v)
            vals.append(str(v).strip())
        lines.append("R%02d | %s" % (r, " | ".join(vals)))
with open(out, "w", encoding="utf-8") as fh:
    fh.write("\n".join(lines) + "\n")
print("wrote", out, len(lines), "lines")
