#!/usr/bin/env node
/**
 * 校验 DNA logo 的三档 PNG 导出（ticket #48）。
 *
 *   node _shared/verify-logo-png.mjs
 *
 * 断言用户可见的事实：尺寸、圆盘外透明、圆盘内三色（cream/pine/marigold）、
 * 环在左上缺口的开口、圆盘在画布中不裁切。坐标与色值来自 logo-dna.svg：
 *   viewBox 256，圆盘 (128,128) r=126，松绿环 r=84 粗 22（左上 90° 缺口），
 *   marigold 点 (80,80) r=18。
 */
import fs from "node:fs";
import path from "node:path";
import zlib from "node:zlib";
import { fileURLToPath } from "node:url";

const SHARED = path.dirname(fileURLToPath(import.meta.url));
const ASSETS = [
  { file: "logo-dna-168.png", size: 168 },
  { file: "logo-dna-84.png", size: 84 },
  { file: "logo-dna-48.png", size: 48 },
];

const CREAM = [0xfc, 0xfa, 0xf4];
const PINE = [0x17, 0x32, 0x20];
const MARIGOLD = [0xdd, 0xb1, 0x4a];

let failures = 0;
let checks = 0;
function check(label, ok, detail = "") {
  checks += 1;
  if (ok) return;
  failures += 1;
  console.error(`  FAIL ${label}${detail ? ` — ${detail}` : ""}`);
}

/** Minimal PNG reader: 8-bit RGBA, non-interlaced only. */
function decodePng(buffer) {
  const sig = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
  if (!buffer.subarray(0, 8).equals(sig)) throw new Error("not a PNG");
  let offset = 8;
  let width = 0;
  let height = 0;
  let bitDepth = 0;
  let colorType = 0;
  let interlace = 0;
  const idat = [];
  while (offset < buffer.length) {
    const length = buffer.readUInt32BE(offset);
    const type = buffer.toString("ascii", offset + 4, offset + 8);
    const data = buffer.subarray(offset + 8, offset + 8 + length);
    offset += 12 + length;
    if (type === "IHDR") {
      width = data.readUInt32BE(0);
      height = data.readUInt32BE(4);
      bitDepth = data[8];
      colorType = data[9];
      interlace = data[12];
    } else if (type === "IDAT") {
      idat.push(data);
    } else if (type === "IEND") {
      break;
    }
  }
  if (bitDepth !== 8 || colorType !== 6 || interlace !== 0) {
    throw new Error(
      `unsupported PNG (bitDepth=${bitDepth} colorType=${colorType} interlace=${interlace}); expected 8-bit RGBA`,
    );
  }
  const raw = zlib.inflateSync(Buffer.concat(idat));
  const stride = width * 4;
  const pixels = Buffer.alloc(stride * height);
  for (let y = 0; y < height; y += 1) {
    const filter = raw[y * (stride + 1)];
    const line = raw.subarray(y * (stride + 1) + 1, (y + 1) * (stride + 1));
    const out = pixels.subarray(y * stride, (y + 1) * stride);
    const prev = y > 0 ? pixels.subarray((y - 1) * stride, y * stride) : null;
    for (let x = 0; x < stride; x += 1) {
      const left = x >= 4 ? out[x - 4] : 0;
      const up = prev ? prev[x] : 0;
      const upLeft = prev && x >= 4 ? prev[x - 4] : 0;
      let value;
      switch (filter) {
        case 0:
          value = line[x];
          break;
        case 1:
          value = line[x] + left;
          break;
        case 2:
          value = line[x] + up;
          break;
        case 3:
          value = line[x] + Math.floor((left + up) / 2);
          break;
        case 4: {
          const p = left + up - upLeft;
          const pa = Math.abs(p - left);
          const pb = Math.abs(p - up);
          const pc = Math.abs(p - upLeft);
          const predictor = pa <= pb && pa <= pc ? left : pb <= pc ? up : upLeft;
          value = line[x] + predictor;
          break;
        }
        default:
          throw new Error(`unknown PNG filter ${filter} on row ${y}`);
      }
      out[x] = value & 0xff;
    }
  }
  return { width, height, pixels };
}

function pixelAt(image, x, y) {
  const i = (y * image.width + x) * 4;
  return [image.pixels[i], image.pixels[i + 1], image.pixels[i + 2], image.pixels[i + 3]];
}

function closeTo(actual, expected, tolerance = 10) {
  // alpha 由调用方单独断言，这里只比 RGB。
  return expected.every((v, i) => Math.abs(actual[i] - v) <= tolerance);
}

function describe(pixel) {
  return `rgba(${pixel.join(",")})`;
}

/** Map a 256-space SVG point to the exported pixel grid. */
function samplePoint(size, svgX, svgY) {
  const scale = size / 256;
  return [
    Math.min(size - 1, Math.max(0, Math.round(svgX * scale))),
    Math.min(size - 1, Math.max(0, Math.round(svgY * scale))),
  ];
}

for (const { file, size } of ASSETS) {
  const asset = path.join(SHARED, file);
  console.log(`${file}`);
  if (!fs.existsSync(asset)) {
    check(`${file} exists`, false, "file missing");
    continue;
  }
  let image;
  try {
    image = decodePng(fs.readFileSync(asset));
  } catch (error) {
    check(`${file} is a decodable 8-bit RGBA PNG`, false, error.message);
    continue;
  }

  check(`${file} width`, image.width === size, `got ${image.width}, want ${size}`);
  check(`${file} height`, image.height === size, `got ${image.height}, want ${size}`);
  if (image.width !== size || image.height !== size) continue;

  // 圆盘外必须透明。只看四角：边中点距圆心约 126.5（SVG 单位），
  // 仍落在 r=127 的 12% 描边里，会带出非零 alpha，不适合当断言点。
  for (const [x, y] of [
    [1, 1],
    [size - 2, 1],
    [1, size - 2],
    [size - 2, size - 2],
  ]) {
    const pixel = pixelAt(image, x, y);
    check(`${file} transparent outside disc at (${x},${y})`, pixel[3] === 0, describe(pixel));
  }

  // 圆盘内：松绿环 / marigold 点 / cream 底。坐标从 SVG 几何推出。
  const samples = [
    { name: "ring right", svg: [212, 128], expected: PINE, alpha: "opaque" },
    { name: "ring bottom", svg: [128, 212], expected: PINE, alpha: "opaque" },
    { name: "marigold dot", svg: [80, 80], expected: MARIGOLD, alpha: "opaque" },
    { name: "cream center", svg: [128, 128], expected: CREAM, alpha: "opaque" },
    { name: "cream in upper-left ring gap", svg: [51.9, 92.5], expected: CREAM, alpha: "opaque" },
  ];
  for (const { name, svg, expected, alpha } of samples) {
    const [x, y] = samplePoint(size, svg[0], svg[1]);
    const pixel = pixelAt(image, x, y);
    check(`${file} ${name} alpha`, pixel[3] >= 250, describe(pixel));
    check(`${file} ${name} color`, closeTo(pixel, expected), `${describe(pixel)} want rgb(${expected.join(",")})`);
  }

  // 不裁切、不偏心：中心横向扫描，圆盘左右边界应各留约 1 个源像素的空隙。
  const midY = Math.round(size / 2);
  let left = -1;
  let right = -1;
  for (let x = 0; x < size; x += 1) {
    if (pixelAt(image, x, midY)[3] > 0) {
      if (left === -1) left = x;
      right = x;
    }
  }
  const expectedInset = Math.round(size / 256);
  const inset = Math.max(1, expectedInset);
  check(
    `${file} disc not cropped horizontally`,
    left >= inset - 1 && right <= size - 1 - (inset - 1) && left <= inset + 1 && size - 1 - right <= inset + 1,
    `left=${left} right=${right} size=${size}`,
  );
}

if (failures) {
  console.error(`\n${failures} of ${checks} checks failed`);
  process.exitCode = 1;
} else {
  console.log(`\nall ${checks} checks passed`);
}
