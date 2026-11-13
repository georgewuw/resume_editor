// Build on Windows, install the Debian package in Ubuntu, then launch it through WSLg.
import fs from 'node:fs';
import path from 'node:path';
import { spawn, spawnSync } from 'node:child_process';
import { once } from 'node:events';
import { parseArgs } from 'node:util';
import { ROOT } from './lib/assets.mjs';

const help = `用法：npm run wsl -- [选项]
  --skip-build       安装现有的最新 Ubuntu x64 .deb，不重新构建
  --package <路径>   安装指定 .deb，自动跳过构建
  --distro <名称>    WSL 发行版名称，默认 Ubuntu
  --repair-wslg     图形通道故障时重建 WSLg，会关闭所有 Linux 图形窗口
  --help            显示帮助`;

function run(command, args, { capture = false, allowFailure = false } = {}) {
  const result = spawnSync(command, args, {
    cwd: ROOT,
    windowsHide: true,
    stdio: ['ignore', capture ? 'pipe' : 'inherit', 'inherit'],
    encoding: 'utf8',
  });
  if (result.error) throw result.error;
  if (result.status !== 0 && !allowFailure) {
    throw new Error(`${command} 执行失败（退出码 ${result.status ?? result.signal}）`);
  }
  return result;
}

function packageFile(config, supplied) {
  if (supplied) return path.resolve(ROOT, supplied);
  const directory = path.resolve(ROOT, config.out ?? 'build', 'linux-amd64');
  const packages = fs
    .readdirSync(directory)
    .filter((name) => name.startsWith(`${config.slug}_`) && name.endsWith('_amd64.deb'))
    .map((name) => path.join(directory, name))
    .sort((a, b) => fs.statSync(b).mtimeMs - fs.statSync(a).mtimeMs);
  if (!packages.length) throw new Error('未找到 Ubuntu x64 .deb，请先运行 npm run build:ubuntu');
  return packages[0];
}

async function deploy(options) {
  if (process.platform !== 'win32')
    throw new Error('此脚本需要在 Windows 上运行，目标为 WSL2 Ubuntu x64');
  const config = JSON.parse(fs.readFileSync(path.join(ROOT, 'mygo.json'), 'utf8'));
  const app = config.slug;
  if (!/^[a-z0-9][a-z0-9.+-]*$/.test(app))
    throw new Error('mygo.json 的 slug 必须是有效的 Debian 包名');
  const distro = options.distro;
  const wsl = (args, settings) => run('wsl.exe', ['--distribution', distro, ...args], settings);
  const linuxPath = (file) =>
    wsl(['--exec', 'wslpath', '-a', '-u', file], { capture: true }).stdout.trim();

  // Keep a WSL client connected between short commands, including while Go builds on Windows.
  // Closing stdin ends this helper without leaving a sleep process behind.
  const keeper = spawn(
    'wsl.exe',
    ['--distribution', distro, '--exec', 'sh', '-c', 'read -r keep_alive'],
    { windowsHide: true, stdio: ['pipe', 'ignore', 'inherit'] },
  );
  await once(keeper, 'spawn');
  try {
    const script = linuxPath(path.join(ROOT, 'scripts', 'wsl', 'ubuntu.sh'));
    const ubuntu = (operation, args = [], settings) =>
      wsl(['--exec', 'bash', script, operation, app, ...args], settings);

    console.log(`\n[WSL] 检查 ${distro} 环境`);
    ubuntu('check');
    const graphics = ubuntu('graphics', [], { allowFailure: true });
    if (graphics.status === 42) {
      if (!options['repair-wslg']) {
        throw new Error(
          'WSLg 图形通道处于 COPY MODE。关闭其他 Linux 图形窗口后，加 --repair-wslg 重试',
        );
      }
      console.log('[WSL] 重建 WSLg 图形会话');
      run('wsl.exe', ['--system', '--exec', 'pkill', '-TERM', '-x', 'weston']);
      ubuntu('graphics', ['wait']);
    } else if (graphics.status !== 0) {
      throw new Error('WSLg 图形通道未就绪，请查看上方错误');
    }

    if (!options['skip-build'] && !options.package) {
      console.log('\n[WSL] 构建 Ubuntu x64 安装包');
      run('go', ['tool', 'mygo', 'build', '-platform', 'linux/amd64', '-skip-installer']);
    }
    const deb = packageFile(config, options.package);
    if (!fs.statSync(deb).isFile()) throw new Error(`安装包不存在：${deb}`);
    const linuxDeb = linuxPath(deb);
    console.log(`\n[WSL] 安装 ${deb}`);
    ubuntu('stop');
    wsl(['--user', 'root', '--exec', 'bash', script, 'install', app, linuxDeb]);

    console.log('\n[WSL] 通过 WSLg 打开已安装的应用');
    ubuntu('graphics');
    ubuntu('prepare');
    // This Windows WSL client remains attached to the GUI process after npm exits.
    const gui = spawn(
      'wsl.exe',
      ['--distribution', distro, '--exec', 'bash', script, 'launch', app],
      { detached: true, windowsHide: true, stdio: 'ignore' },
    );
    await once(gui, 'spawn');
    gui.unref();
    ubuntu('ready');
    console.log(`[WSL] 完成：${distro} 已安装并启动 ${app}`);
  } finally {
    keeper.stdin.end();
  }
}

try {
  const { values } = parseArgs({
    options: {
      distro: { type: 'string', default: 'Ubuntu' },
      package: { type: 'string' },
      'skip-build': { type: 'boolean' },
      'repair-wslg': { type: 'boolean' },
      help: { type: 'boolean' },
    },
  });
  if (values.help) console.log(help);
  else await deploy(values);
} catch (error) {
  console.error(`\n[WSL] ${error.message}`);
  process.exitCode = 1;
}
