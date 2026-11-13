// Syntax checking only; this command does not run tests or modify source files.
import fs from 'node:fs';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { ROOT } from './lib/assets.mjs';

function sourceFiles(directory) {
  return fs.readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const file = path.join(directory, entry.name);
    if (entry.isDirectory()) return sourceFiles(file);
    return /\.(?:js|mjs)$/.test(entry.name) ? [file] : [];
  });
}

const files = ['frontend', 'scripts'].flatMap((name) => sourceFiles(path.join(ROOT, name)));
for (const file of files) {
  const result = spawnSync(process.execPath, ['--check', file], {
    stdio: 'inherit',
    windowsHide: true,
  });
  if (result.error) throw result.error;
  if (result.status !== 0) process.exit(result.status ?? 1);
}
console.log('JavaScript 语法检查通过：' + files.length + ' 个文件');
