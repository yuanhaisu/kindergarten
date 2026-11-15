/**
 * 票 #48：DNA logo 三档 PNG 导出（168 / 84 / 48）。
 *
 * 原先由 `_shared/verify-logo-png.mjs` 单独校验，等于在 checker 之外多开了一条
 * 验收缝；#41 明确要求 checker 是唯一新增验收缝，因此尺寸 / 透明 / 取样色 /
 * 不裁切断言并入这里，导出脚本 `export-logo-png.mjs` 仍只负责产出资产。
 *
 * 坐标与色值来自 logo-dna.svg：viewBox 256，圆盘 (128,128) r=126，
 * 松绿环 r=84 粗 22（左上 90° 缺口），marigold 点 (80,80) r=18。
 */
const CREAM = [0xfc, 0xfa, 0xf4];
const PINE = [0x17, 0x32, 0x20];
const MARIGOLD = [0xdd, 0xb1, 0x4a];

const samples = [
  { name: "松绿环右侧", svg: [212, 128], rgb: PINE },
  { name: "松绿环底部", svg: [128, 212], rgb: PINE },
  { name: "marigold 圆点", svg: [80, 80], rgb: MARIGOLD },
  { name: "cream 圆心", svg: [128, 128], rgb: CREAM },
  { name: "环缺口处 cream", svg: [51.9, 92.5], rgb: CREAM },
];

export default {
  assets: [168, 84, 48].map((size) => ({
    file: `logo-dna-${size}.png`,
    width: size,
    height: size,
    viewBox: 256,
    // 圆盘外透明只看四角：边中点仍在描边附近，会带出非零 alpha。
    transparentCorners: true,
    discInset: true,
    samples,
  })),
};
