#!/usr/bin/env node
/**
 * 从 _shared/logo-dna.svg 导出小程序用 PNG（ticket #48）。
 *
 *   node _shared/export-logo-png.mjs
 *
 * 三档 @3x：168 / 84 / 48，对应 56 / 28 / 16 展示尺寸。
 * 用本机 Chrome（本批截图链路已依赖它）在精确视口下栅格化，圆盘外透明；
 * 不联网、不引入 npm 依赖。导出后由 `node _shared/check-variants.mjs --chosen`
 * 的 48-logo-png 资产断言校验尺寸 / 透明 / 取样色 / 不裁切。
 */
import fs from "node:fs";
import path from "node:path";
import { randomUUID } from "node:crypto";
import { fileURLToPath, pathToFileURL } from "node:url";
import { launchChrome, sleep } from "./lib/cdp.mjs";

const SHARED = path.dirname(fileURLToPath(import.meta.url));
const SOURCE = path.join(SHARED, "logo-dna.svg");
const SIZES = [168, 84, 48];

/** Inline the SVG at a 1:1 viewBox so Chrome rasterizes vectors at the target size. */
function buildHtml(svg, size) {
  const inline = svg
    .replace(/\swidth="[^"]*"/, "")
    .replace(/\sheight="[^"]*"/, "")
    .replace("<svg ", `<svg width="${size}" height="${size}" `);
  return `<!doctype html>
<html><head><meta charset="utf-8">
<style>
  html, body { margin: 0; padding: 0; width: ${size}px; height: ${size}px; background: transparent; overflow: hidden; }
  svg { display: block; width: ${size}px; height: ${size}px; }
</style></head>
<body>${inline}</body></html>`;
}

async function exportAll(svg) {
  const workDir = path.join(process.env.TEMP || "/tmp", `od-logo-${randomUUID()}`);
  fs.mkdirSync(workDir, { recursive: true });
  const session = await launchChrome({ extraArgs: ["--hide-scrollbars", "--force-color-profile=srgb"] });

  try {
    await session.send("Page.enable");
    await session.send("Emulation.setDefaultBackgroundColorOverride", { color: { r: 0, g: 0, b: 0, a: 0 } });

    for (const size of SIZES) {
      const htmlPath = path.join(workDir, `${size}.html`);
      fs.writeFileSync(htmlPath, buildHtml(svg, size), "utf8");
      const url = pathToFileURL(htmlPath).href;

      await session.setViewport({ width: size, height: size, deviceScaleFactor: 1, mobile: false });
      await session.navigate(url);
      // 导航完成不等于矢量已栅格化；留半秒确保截图拿到稳定像素。
      await sleep(500);

      const probe = await session.evaluate(
        "(() => ({ vw: window.innerWidth, vh: window.innerHeight, href: location.href, svg: !!document.querySelector('svg') }))()",
      );
      if (probe.href !== url) throw new Error(`页面被导航到别处：${probe.href}`);
      if (probe.vw !== size || probe.vh !== size) {
        throw new Error(`视口为 ${probe.vw}x${probe.vh}，期望 ${size}x${size}`);
      }
      if (!probe.svg) throw new Error(`${size}px 页面里找不到 svg`);

      const shot = await session.send("Page.captureScreenshot", {
        format: "png",
        clip: { x: 0, y: 0, width: size, height: size, scale: 1 },
        captureBeyondViewport: false,
      });
      const out = path.join(SHARED, `logo-dna-${size}.png`);
      fs.writeFileSync(out, Buffer.from(shot.data, "base64"));
      console.log(`wrote ${path.basename(out)} (${size}x${size}, ${fs.statSync(out).size} bytes)`);
    }
  } finally {
    await session.close();
    try {
      fs.rmSync(workDir, { recursive: true, force: true });
    } catch (error) {
      console.error(`[export-logo-png] 清理临时目录失败：${error.message}`);
    }
  }
}

await exportAll(fs.readFileSync(SOURCE, "utf8"));
