/**
 * 用一次无头 Chrome 会话渲染多个原型页，返回渲染后的可见文本 / DOM 属性 /
 * 交互结果。
 *
 * chosen 检查需要三类静态 HTML 给不了的事实：
 * - `?state=` 由 app.js 在运行期切换，静态文件里读到的是全部分支；
 * - `within` 要按 CSS 选择器限定到容器；
 * - 默认勾选态与「点登录后是否离开当前页」只有跑起来才能判定。
 * 三者都只有渲染后才能判定，因此这里复用共享 CDP 驱动。
 *
 * 渲染失败（导航超时、选择器未命中等）不再落成「页面不存在」：每个请求返回
 * `{ error }`，由检查器按页报出具体原因；启动阶段失败（没有 Chrome）才整体抛出。
 */
import path from "node:path";
import { pathToFileURL } from "node:url";
import { launchChrome, sleep, TIMEOUT_MS } from "./cdp.mjs";

const describe = (request) =>
  request.label || (request.file ? path.basename(path.dirname(request.file)) : "unknown");

const buildUrl = (request) => {
  const url = pathToFileURL(path.resolve(request.file));
  if (request.query) url.search = request.query;
  return url.href;
};

const TEXT_EXPRESSION = (selectors, props) => `(() => {
  const normalize = (value) => (value || "").replace(/\\s+/g, " ").trim();
  const scoped = {};
  const counts = {};
  for (const selector of ${JSON.stringify(selectors)}) {
    const nodes = document.querySelectorAll(selector);
    counts[selector] = nodes.length;
    const element = nodes[0] || null;
    scoped[selector] = element ? normalize(element.innerText) : null;
  }
  const props = {};
  for (const spec of ${JSON.stringify(props)}) {
    const element = document.querySelector(spec.selector);
    if (spec.check === "checked") {
      props[spec.selector + "|checked"] =
        element && typeof element.checked === "boolean" ? element.checked : null;
    } else if (spec.check === "tagName") {
      props[spec.selector + "|tagName"] = element ? element.tagName.toLowerCase() : null;
    } else if (spec.check === "bottomSheet") {
      if (!element) {
        props[spec.selector + "|bottomSheet"] = null;
      } else {
        const rect = element.getBoundingClientRect();
        const rendered = rect.width > 0 && rect.height > 0;
        props[spec.selector + "|bottomSheet"] =
          rendered &&
          rect.bottom >= window.innerHeight - 2 &&
          rect.left <= 2 &&
          rect.right >= window.innerWidth - 2;
      }
    } else {
      props[spec.selector + "|" + spec.check] = null;
    }
  }
  return {
    title: document.title || "",
    text: normalize((document.title || "") + " " + (document.body ? document.body.innerText : "")),
    scoped,
    counts,
    props,
  };
})()`;

const INTERACTION_EXPRESSION = (probes) => `(() => {
  const normalize = (value) => (value || "").replace(/\\s+/g, " ").trim();
  const visible = {};
  const visibleText = {};
  for (const selector of ${JSON.stringify(probes)}) {
    const element = document.querySelector(selector);
    if (!element) {
      visible[selector] = false;
      visibleText[selector] = null;
      continue;
    }
    const rect = element.getBoundingClientRect();
    const style = getComputedStyle(element);
    visible[selector] =
      !element.hidden &&
      rect.width > 0 &&
      rect.height > 0 &&
      style.display !== "none" &&
      style.visibility !== "hidden";
    visibleText[selector] = normalize(element.innerText);
  }
  return { href: location.href, visible, visibleText };
})()`;

async function clickSelector(session, selector, label) {
  const clicked = await session.evaluate(`(() => {
    const element = document.querySelector(${JSON.stringify(selector)});
    if (!element) return false;
    element.click();
    return true;
  })()`);
  if (!clicked) throw new Error(`${label}：交互目标未找到 ${selector}`);
  await sleep(80);
}

/** 导航期间执行上下文会短暂失效；重试到文档可读，失败时带出最后原因。 */
async function evaluateAfterNavigation(session, expression, label) {
  const deadline = Date.now() + 3000;
  let lastError = null;
  while (Date.now() < deadline) {
    try {
      return await session.evaluate(expression);
    } catch (error) {
      lastError = error;
      await sleep(100);
    }
  }
  throw new Error(
    `${label}：导航后无法读取页面（${lastError ? lastError.message : "执行上下文超时"}）`,
  );
}

async function renderInteraction(session, request) {
  const { interaction } = request;
  const label = describe(request);
  await session.navigate(buildUrl(request), { timeoutMs: TIMEOUT_MS });
  // defer 脚本在 load 之前已经跑过；留一帧给 transition / 状态写入。
  await sleep(120);

  for (const step of interaction.steps || []) {
    await clickSelector(session, step.click, label);
  }

  const probes = [interaction.expectVisible, interaction.expectHidden].filter(Boolean);
  const startHref = await session.evaluate("location.href");
  await clickSelector(session, interaction.click, label);

  let navigateError = null;
  if (interaction.expectNavigateTo) {
    const deadline = Date.now() + (interaction.timeoutMs || 6000);
    let moved = false;
    while (Date.now() < deadline) {
      await sleep(100);
      try {
        const href = await session.evaluate("location.href");
        if (href !== startHref) {
          moved = true;
          break;
        }
      } catch (error) {
        // 正在导航，旧执行上下文被销毁属预期；记录原因，超时仍未离开时抛出。
        navigateError = error;
      }
    }
    if (!moved) {
      throw new Error(
        `${label}：点击 ${interaction.click} 后未离开当前页（期望到 ${interaction.expectNavigateTo}）` +
          (navigateError ? `；最后错误：${navigateError.message}` : ""),
      );
    }
    await sleep(200);
  } else {
    // 没有导航期望时等足够久，确保「不该跳」的场景真的没有延迟跳转。
    await sleep(interaction.settleMs || 1500);
  }

  const result = await evaluateAfterNavigation(session, INTERACTION_EXPRESSION(probes), label);
  return { interaction: result };
}

async function renderPage(session, request) {
  const label = describe(request);
  await session.navigate(buildUrl(request), { timeoutMs: TIMEOUT_MS });
  // defer 脚本在 load 之前已经跑过；留一帧给 transition / 状态写入。
  await sleep(120);
  return session.evaluate(TEXT_EXPRESSION(request.selectors || [], request.props || []));
}

/**
 * @param {{file: string, query?: string, selectors?: string[], props?: object[], interaction?: object, label?: string}[]} requests
 * @returns {Promise<Map<number, object>>} 每项或为渲染结果，或为 `{ error }`
 */
export async function renderPages(requests) {
  const session = await launchChrome();
  const results = new Map();
  try {
    await session.send("Page.enable");
    await session.send("Runtime.enable");
    await session.setViewport({ width: 390, height: 844, deviceScaleFactor: 2, mobile: true });

    for (let index = 0; index < requests.length; index += 1) {
      const request = requests[index];
      try {
        results.set(
          index,
          request.interaction ? await renderInteraction(session, request) : await renderPage(session, request),
        );
      } catch (error) {
        results.set(index, { error: `${describe(request)}：${error.message}` });
      }
    }
  } finally {
    await session.close();
  }
  return results;
}
