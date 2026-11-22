/**
 * chosen 检查器的 PNG 资产断言（原 `_shared/verify-logo-png.mjs` 的 dims/alpha
 * 校验并入 checker 后留下的解码与取样实现）。
 *
 * 只支持 8-bit RGBA 非交错 PNG —— Chrome 的 `Page.captureScreenshot` 就是这种。
 */
import zlib from "node:zlib";

const SIGNATURE = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);

/** Minimal PNG reader: 8-bit RGBA, non-interlaced only. */
export function decodePng(buffer) {
  if (!buffer.subarray(0, 8).equals(SIGNATURE)) throw new Error("not a PNG");
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

export function pixelAt(image, x, y) {
  const i = (y * image.width + x) * 4;
  return [image.pixels[i], image.pixels[i + 1], image.pixels[i + 2], image.pixels[i + 3]];
}

const closeTo = (actual, expected, tolerance = 10) =>
  expected.every((value, index) => Math.abs(actual[index] - value) <= tolerance);

export const describePixel = (pixel) => `rgba(${pixel.join(",")})`;

/** Map a viewBox point to the exported pixel grid. */
function samplePoint(size, viewBox, svgX, svgY) {
  const scale = size / viewBox;
  return [
    Math.min(size - 1, Math.max(0, Math.round(svgX * scale))),
    Math.min(size - 1, Math.max(0, Math.round(svgY * scale))),
  ];
}

/**
 * 断言一个 PNG 资产的用户可见事实：尺寸、圆盘外透明、取样点颜色、圆盘不裁切。
 *
 * @param {{width:number,height:number,viewBox?:number,transparentCorners?:boolean,discInset?:boolean,samples?:object[]}} spec
 * @returns {{checks: number, failures: {label: string, detail: string}[]}}
 */
export function verifyAsset(spec, buffer) {
  const failures = [];
  let checks = 0;
  const check = (label, ok, detail = "") => {
    checks += 1;
    if (!ok) failures.push({ label, detail });
  };

  let image;
  try {
    image = decodePng(buffer);
  } catch (error) {
    check(`${spec.file} 是 8-bit RGBA PNG`, false, error.message);
    return { checks, failures };
  }

  check(`${spec.file} 宽度 ${spec.width}`, image.width === spec.width, `实际 ${image.width}`);
  check(`${spec.file} 高度 ${spec.height}`, image.height === spec.height, `实际 ${image.height}`);
  if (image.width !== spec.width || image.height !== spec.height) return { checks, failures };

  const size = spec.width;
  if (spec.transparentCorners) {
    // 只看四角：边中点仍在圆盘描边附近，会带出非零 alpha。
    for (const [x, y] of [
      [1, 1],
      [size - 2, 1],
      [1, size - 2],
      [size - 2, size - 2],
    ]) {
      const pixel = pixelAt(image, x, y);
      check(`${spec.file} (${x},${y}) 圆盘外透明`, pixel[3] === 0, describePixel(pixel));
    }
  }

  const viewBox = spec.viewBox || 256;
  for (const sample of spec.samples || []) {
    const [x, y] = samplePoint(size, viewBox, sample.svg[0], sample.svg[1]);
    const pixel = pixelAt(image, x, y);
    if (sample.opaque !== false) check(`${spec.file} ${sample.name} 不透明`, pixel[3] >= 250, describePixel(pixel));
    check(
      `${spec.file} ${sample.name} 颜色`,
      closeTo(pixel, sample.rgb),
      `${describePixel(pixel)}，期望 rgb(${sample.rgb.join(",")})`,
    );
  }

  if (spec.discInset) {
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
    const inset = Math.max(1, Math.round(size / viewBox));
    check(
      `${spec.file} 圆盘未被裁切`,
      left >= inset - 1 && right <= size - 1 - (inset - 1) && left <= inset + 1 && size - 1 - right <= inset + 1,
      `left=${left} right=${right} size=${size}`,
    );
  }

  return { checks, failures };
}
