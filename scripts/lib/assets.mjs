import { execFileSync } from 'node:child_process';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

export const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');

// The runtime parser, field scanner and schema generator have one Go implementation.
export function generateAssets() {
  const output = execFileSync('go', ['run', './cmd/resume-assets'], {
    cwd: ROOT,
    encoding: 'utf8',
    maxBuffer: 8 * 1024 * 1024,
    stdio: ['ignore', 'pipe', 'inherit'],
    windowsHide: true,
  });
  return JSON.parse(output);
}
