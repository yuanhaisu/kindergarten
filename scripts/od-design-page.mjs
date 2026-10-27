#!/usr/bin/env node
/**
 * Drive one OpenDesign page-design run end to end.
 *
 *   node scripts/od-design-page.mjs \
 *     --slug teacher-index --surface teacher --title 老师工作台 \
 *     --brief path/to/brief.md --out docs/design-previews/<batch>/teacher/01-workbench
 *
 * Creates a project, starts a `claude` run bound to the nurture-planet design
 * system, waits for a terminal status, copies the produced files into --out,
 * writes _run.json and best-effort renders preview.png.
 *
 * Exit codes: 0 succeeded, 1 run failed/timed out or artifacts missing, 2 bad args.
 */
import fs from 'node:fs';
import path from 'node:path';
import { randomUUID } from 'node:crypto';
import { execFile } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const DAEMON = process.env.OD_DAEMON || 'http://127.0.0.1:7456';
const DESIGN_SYSTEM = process.env.OD_DESIGN_SYSTEM || 'user:nurture-planet-local';
const PROJECTS_ROOT =
  process.env.OD_PROJECTS_ROOT ||
  'D:/yuanhaisu/personal_projects/open-design/.od/projects';
const RUNS_ROOT =
  process.env.OD_RUNS_ROOT ||
  'D:/yuanhaisu/personal_projects/open-design/.od/runs';

function runStateOnDisk(runId) {
  try {
    const state = JSON.parse(fs.readFileSync(path.join(RUNS_ROOT, runId, 'state.json'), 'utf8'));
    return state?.status ?? null;
  } catch {
    return null;
  }
}

function parseArgs(argv) {
  const out = {};
  for (let i = 0; i < argv.length; i += 1) {
    const key = argv[i];
    if (!key.startsWith('--')) continue;
    const name = key.slice(2);
    const value = argv[i + 1];
    if (value === undefined || value.startsWith('--')) {
      out[name] = true;
    } else {
      out[name] = value;
      i += 1;
    }
  }
  return out;
}

const args = parseArgs(process.argv.slice(2));
const slug = args.slug;
const surface = args.surface;
const title = args.title;
const briefPath = args.brief;
const outDir = args.out;

if (!slug || !surface || !briefPath || !outDir) {
  console.error(
    'usage: node scripts/od-design-page.mjs --slug <slug> --surface teacher|parent --title <中文标题> --brief <file> --out <dir> [--timeout-min 25]',
  );
  process.exit(2);
}
if (!['teacher', 'parent'].includes(surface)) {
  console.error(`surface must be teacher or parent, got ${surface}`);
  process.exit(2);
}

const timeoutMin = Number(args['timeout-min'] || 25);
const message = fs.readFileSync(briefPath, 'utf8');

async function api(pathname, init = {}) {
  const res = await fetch(`${DAEMON}${pathname}`, {
    ...init,
    headers: { 'content-type': 'application/json', ...(init.headers || {}) },
  });
  const text = await res.text();
  let body;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = { raw: text.slice(0, 500) };
  }
  if (!res.ok) {
    throw new Error(`${init.method || 'GET'} ${pathname} -> ${res.status} ${JSON.stringify(body).slice(0, 400)}`);
  }
  return body;
}

function log(step, detail) {
  console.log(`[od-page] ${step}${detail ? ` ${detail}` : ''}`);
}

function copyProjectFiles(projectId, dest) {
  const src = path.join(PROJECTS_ROOT, projectId);
  if (!fs.existsSync(src)) return [];
  fs.mkdirSync(dest, { recursive: true });
  // .mcp.json is injected by OpenDesign and carries real API keys — it must
  // never be copied into the repo.
  const skip = new Set(['CLAUDE.md', '.git', 'node_modules', '.DS_Store', '.mcp.json']);
  const copied = [];
  const walk = (rel) => {
    for (const entry of fs.readdirSync(path.join(src, rel), { withFileTypes: true })) {
      if (skip.has(entry.name)) continue;
      const relPath = path.join(rel, entry.name);
      const abs = path.join(src, relPath);
      if (entry.isDirectory()) {
        fs.mkdirSync(path.join(dest, relPath), { recursive: true });
        walk(relPath);
      } else {
        fs.mkdirSync(path.dirname(path.join(dest, relPath)), { recursive: true });
        fs.copyFileSync(abs, path.join(dest, relPath));
        copied.push(relPath.split(path.sep).join('/'));
      }
    }
  };
  walk('');
  return copied;
}

function renderPreview(htmlPath, pngPath) {
  // Delegate to the CDP shot: mixing --force-device-scale-factor with
  // --window-size gives an inconsistent CSS viewport on scaled Windows hosts,
  // which produced cropped previews. od-shot pins the viewport explicitly.
  const shot = path.join(path.dirname(fileURLToPath(import.meta.url)), 'od-shot.mjs');
  return new Promise((resolve) => {
    execFile(
      process.execPath,
      [shot, htmlPath, pngPath, '--width', '390', '--dsf', '2'],
      { timeout: 90000 },
      (err) => resolve(!err && fs.existsSync(pngPath)),
    );
  });
}

async function main() {
  const started = Date.now();
  const projectId = randomUUID();
  const project = await api('/api/projects', {
    method: 'POST',
    body: JSON.stringify({
      id: projectId,
      name: `odpage ${slug}`,
      designSystemId: DESIGN_SYSTEM,
      metadata: { odPageSlug: slug, odPageSurface: surface },
    }),
  });
  const conversationId = project.conversationId;
  if (!conversationId) throw new Error('project created without conversationId');
  log('project', projectId);

  const run = await api('/api/runs', {
    method: 'POST',
    body: JSON.stringify({
      projectId,
      conversationId,
      agentId: 'claude',
      designSystemId: DESIGN_SYSTEM,
      reasoning: 'low',
      clientRequestId: `odpage-${slug}-${Date.now()}`,
      message,
    }),
  });
  const runId = run.runId;
  if (!runId) throw new Error(`run not started: ${JSON.stringify(run).slice(0, 300)}`);
  log('run', runId);

  const deadline = started + timeoutMin * 60 * 1000;
  let status = null;
  let lastNote = 0;
  // The daemon keeps run state in memory: a restart mid-run makes /api/runs/:id
  // 404 while the agent keeps writing files. Fall back to watching the project
  // directory, and treat a stable index.html as a finished run.
  let lastSize = -1;
  let stableSince = 0;
  const projectSig = () => {
    const dir = path.join(PROJECTS_ROOT, projectId);
    return ['index.html', 'styles.css', 'app.js']
      .map((name) => {
        const file = path.join(dir, name);
        return fs.existsSync(file) ? fs.statSync(file).size : -1;
      })
      .join(':');
  };
  while (Date.now() < deadline) {
    await new Promise((r) => setTimeout(r, 5000));
    try {
      status = await api(`/api/runs/${runId}`);
    } catch (error) {
      status = null;
      log('waiting', 'run record unavailable, watching project files');
    }
    const state = status?.status;
    if (status && state !== 'running' && state !== 'queued' && state !== 'starting') break;

    // state.json on disk is written when the run reaches a terminal status and
    // survives the daemon forgetting its in-memory record.
    const disk = runStateOnDisk(runId);
    if (disk && disk !== 'running' && disk !== 'queued' && disk !== 'starting') {
      status = { status: disk, inferred: true, error: null };
      log('settled', `state.json=${disk}`);
      break;
    }

    // The in-memory status can stay 'running' after the files are final (and the
    // list endpoint forgets finished runs entirely). All three artifacts present
    // and byte-stable for 30s is the only signal we trust in that case.
    // NOTE: must require every artifact (-1 = missing); checking only index.html
    // exits while the agent is still writing styles.css/app.js, which shipped
    // incomplete pages in the 2026-10-26 parent batch.
    const sig = projectSig();
    if (!sig.includes('-1') && sig === String(lastSize)) {
      if (!stableSince) stableSince = Date.now();
      if (Date.now() - stableSince > 30000) {
        status = { status: 'succeeded', inferred: true, error: null };
        log('inferred', 'artifacts stable; daemon status never settled');
        break;
      }
    } else {
      stableSince = 0;
      lastSize = sig;
    }

    if (Date.now() - lastNote > 60000) {
      lastNote = Date.now();
      log('waiting', `${Math.round((Date.now() - started) / 1000)}s status=${state ?? 'unknown'} files=${sig}`);
    }
  }

  const terminal = status && !['running', 'queued', 'starting'].includes(status.status);
  if (!terminal) {
    log('timeout', `after ${timeoutMin}min; cancelling`);
    await fetch(`${DAEMON}/api/runs/${runId}/cancel`, { method: 'POST' }).catch(() => {});
    status = await api(`/api/runs/${runId}`).catch(() => status);
  }

  const copied = copyProjectFiles(projectId, path.resolve(outDir));
  // The brand tokens are the one file the run must never invent. Guarantee it
  // exists next to the prototype even when the inner agent didn't write one.
  const tokenSrc =
    process.env.OD_TOKENS_CSS ||
    'D:/yuanhaisu/personal_projects/open-design/.od/design-systems/nurture-planet-local/tokens.css';
  const tokenDest = path.resolve(outDir, 'tokens.css');
  if (!fs.existsSync(tokenDest) && fs.existsSync(tokenSrc)) {
    fs.mkdirSync(path.dirname(tokenDest), { recursive: true });
    fs.copyFileSync(tokenSrc, tokenDest);
    copied.push('tokens.css');
  }
  const hasHtml = copied.includes('index.html');
  const summary = {
    slug,
    surface,
    title,
    projectId,
    conversationId,
    runId,
    status: status?.status ?? null,
    error: status?.error ?? null,
    elapsedSec: Math.round((Date.now() - started) / 1000),
    files: copied,
    hasHtml,
    daemonUrl: `${DAEMON}/projects/${projectId}`,
  };

  const htmlPath = path.resolve(outDir, 'index.html');
  if (hasHtml) {
    const png = path.resolve(outDir, 'preview.png');
    summary.previewRendered = await renderPreview(htmlPath, png);
  }

  fs.mkdirSync(path.resolve(outDir), { recursive: true });
  fs.writeFileSync(
    path.resolve(outDir, '_run.json'),
    `${JSON.stringify(summary, null, 2)}\n`,
    'utf8',
  );
  console.log(`[od-page] RESULT ${JSON.stringify(summary)}`);
  if (summary.status !== 'succeeded' || !hasHtml) process.exit(1);
}

main().catch((error) => {
  console.error(`[od-page] FAILED ${error.stack || error.message}`);
  process.exit(1);
});
