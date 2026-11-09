#!/usr/bin/env node
/**
 * Scaffold a design-change note under docs/design-changes/.
 *
 *   npm run design:change -- pickup-polish "接送页三态与完成态收口"
 */
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const dir = path.join(repoRoot, 'docs/design-changes');
const template = path.join(dir, '_template.md');

const slug = String(process.argv[2] || '').trim();
if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(slug)) {
  console.error('用法: npm run design:change -- <kebab-slug> [标题]');
  process.exit(1);
}

const title = process.argv.slice(3).join(' ').trim() || slug;
const date = new Date();
const pad = (value) => String(value).padStart(2, '0');
const today = `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`;
const target = path.join(dir, `${today}-${slug}.md`);

if (fs.existsSync(target)) {
  console.error(`已存在: ${path.relative(repoRoot, target)}`);
  process.exit(1);
}

const text = fs.readFileSync(template, 'utf8')
  .replaceAll('{{DATE}}', today)
  .replaceAll('{{SLUG}}', slug)
  .replaceAll('{{TITLE}}', title);
fs.writeFileSync(target, text);
console.log(path.relative(repoRoot, target));
