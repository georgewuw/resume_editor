// mygo serves native ES modules; Lucide assets are included at build time.
import fs from 'node:fs';
import path from 'node:path';
import { Pencil, Trash2 } from 'lucide-static';
import { ROOT, generateAssets } from './lib/assets.mjs';

const frontend = path.join(ROOT, 'frontend');
const dist = path.join(ROOT, 'dist');
const data = generateAssets();
const styles = ['base.css', 'titlebar.css', 'sidebar.css', 'workspace.css'];
const css = styles
  .map((name) => {
    const source = fs.readFileSync(path.join(frontend, 'styles', name), 'utf8');
    return '/* styles/' + name + ' */\n' + source;
  })
  .join('\n');

// dist is the dedicated generated directory directly beneath the project root.
fs.rmSync(dist, { recursive: true, force: true });
fs.mkdirSync(dist, { recursive: true });
fs.cpSync(frontend, dist, {
  recursive: true,
  filter: (source) => source !== path.join(frontend, 'styles') && !source.endsWith('.d.ts'),
});
let html = fs.readFileSync(path.join(frontend, 'index.html'), 'utf8');
for (const [name, svg] of [
  ['pencil', Pencil],
  ['trash-2', Trash2],
]) {
  const marker = '<!-- lucide:' + name + ' -->';
  if (!html.includes(marker)) throw new Error('Missing Lucide icon marker: ' + name);
  html = html.replaceAll(marker, svg.replace('<svg', '<svg aria-hidden="true" focusable="false"'));
}
fs.writeFileSync(path.join(dist, 'index.html'), html);
const licenseDirectory = path.join(dist, 'licenses');
fs.mkdirSync(licenseDirectory, { recursive: true });
fs.copyFileSync(
  path.join(ROOT, 'node_modules', 'lucide-static', 'LICENSE'),
  path.join(licenseDirectory, 'lucide.txt'),
);
fs.writeFileSync(path.join(dist, 'style.css'), css);
fs.writeFileSync(
  path.join(dist, 'data.js'),
  '// Generated. Do not edit.\nexport default ' + JSON.stringify(data) + ';\n',
);
console.log('dist/ 已生成：' + data.schema.length + ' 个字段，原生 ES 模块');
