// 生成 dist/：纯静态前端，不需要 Vite / Bun / npm 依赖。
// mygo build 会执行本脚本并把 dist/ 嵌入可执行文件；
// 直接用浏览器打开 dist/index.html 也能以「降级模式」完整使用编辑器。
//
// 目录参考 MyGit：frontend/ 按界面区域拆成 styles/ 与 components/，
// 构建时把样式按序拼接为单文件 style.css，组件与入口脚本原样复制。
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const DIST = path.join(ROOT, 'dist');
const FRONTEND = path.join(ROOT, 'frontend');
const A = name => fs.readFileSync(path.join(ROOT, 'assets', name), 'utf8');

// docs/ 是模板与内容文档的唯一真相源；同步一份到 assets/ 供 go:embed 打包。
for (const name of ['简历模板.html', '简历内容.md']) {
  const src = path.join(ROOT, 'docs', name);
  if (fs.existsSync(src)) fs.copyFileSync(src, path.join(ROOT, 'assets', name));
}

// 与 Go / 浏览器端一致的 Markdown 字段解析
function parseContent(src) {
  const fields = {};
  let current = '';
  let inFence = false;
  for (const raw of src.split(/\r?\n/)) {
    const line = raw.trim();
    if (line.startsWith('```')) { inFence = !inFence; continue; }
    if (inFence) continue;
    if (line.startsWith('@@')) {
      current = line.slice(2).trim();
      if (!current || current in fields) throw new Error(`字段 ID 异常：${current}`);
      fields[current] = [];
      continue;
    }
    if (line.startsWith('## ') || line === '---') { current = ''; continue; }
    if (line.startsWith('::') || line === '') continue;
    if (current) fields[current].push(line);
  }
  for (const [k, v] of Object.entries(fields)) {
    if (!v.length) throw new Error(`字段 ${k} 没有内容`);
  }
  return fields;
}

const tpl = A('简历模板.html');
const schema = JSON.parse(A('schema.json'));
const initial = parseContent(A('简历内容.md'));

const keysInTemplate = [...tpl.matchAll(/\bdata-f="([^"]+)"/g)].map(m => m[1]);
const uniqueKeys = [...new Set(keysInTemplate)];
const fieldKeys = Object.keys(initial);
const missing = uniqueKeys.filter(k => !fieldKeys.includes(k));
const extra = fieldKeys.filter(k => !uniqueKeys.includes(k));
if (missing.length || extra.length) {
  throw new Error(`字段与模板不一致：缺失 ${missing.join(',')} / 多余 ${extra.join(',')}`);
}

// 样式按「全局 → 顶栏 → 侧栏 → 工作区」拼接，后者覆盖前者
const CSS_ORDER = ['base.css', 'titlebar.css', 'sidebar.css', 'workspace.css'];
const css = CSS_ORDER.map(name => {
  const file = path.join(FRONTEND, 'styles', name);
  if (!fs.existsSync(file)) throw new Error(`缺少样式模块：${name}`);
  return `/* ===== styles/${name} ===== */\n` + fs.readFileSync(file, 'utf8').trim() + '\n';
}).join('\n');

const COMPONENTS = ['titlebar.js', 'panes.js'];

fs.rmSync(DIST, { recursive: true, force: true });
fs.mkdirSync(path.join(DIST, 'components'), { recursive: true });
fs.copyFileSync(path.join(FRONTEND, 'index.html'), path.join(DIST, 'index.html'));
fs.copyFileSync(path.join(FRONTEND, 'app.js'), path.join(DIST, 'app.js'));
for (const name of COMPONENTS) {
  const file = path.join(FRONTEND, 'components', name);
  if (!fs.existsSync(file)) throw new Error(`缺少界面组件：${name}`);
  fs.copyFileSync(file, path.join(DIST, 'components', name));
}
fs.writeFileSync(path.join(DIST, 'style.css'), css, 'utf8');

// 浏览器降级模式所需的模板、字段定义与初始内容
const data = [
  '/* 由 scripts/build-web.mjs 生成，请勿手工编辑 */',
  `window.__RESUME_TEMPLATE__ = ${JSON.stringify(tpl)};`,
  `window.__RESUME_SCHEMA__ = ${JSON.stringify(schema)};`,
  `window.__RESUME_INITIAL__ = ${JSON.stringify({ fields: initial })};`,
  '',
].join('\n');
fs.writeFileSync(path.join(DIST, 'data.js'), data, 'utf8');

console.log(`dist/ 已生成：${schema.length} 个字段，模板 ${tpl.length} 字符`);
console.log(`样式模块 ${CSS_ORDER.length} 个，界面组件 ${COMPONENTS.length} 个`);
