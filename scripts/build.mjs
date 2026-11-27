import { execFileSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { generateIconSyso } from './gen_icon_syso.mjs';

const root = process.cwd();
const distDir = path.join(root, 'dist');
fs.mkdirSync(distDir, { recursive: true });

generateIconSyso(path.join(root, 'cmd', 'ramkiller', 'rsrc_windows_amd64.syso'));

const outExe = path.join(distDir, 'ramkiller.exe');
const goBin = fs.existsSync('C:\\Program Files\\Go\\bin\\go.exe') ? 'C:\\Program Files\\Go\\bin\\go.exe' : 'go';
const env = { ...process.env, PATH: `${path.dirname(goBin)};${process.env.PATH || ''}`, CGO_ENABLED: '0', GOOS: 'windows', GOARCH: 'amd64' };

execFileSync(goBin, ['build', '-trimpath', '-ldflags=-H=windowsgui -s -w', '-o', outExe, './cmd/ramkiller'], {
  cwd: root,
  stdio: 'inherit',
  env
});
fs.rmSync(`${outExe}~`, { force: true });

const stat = fs.statSync(outExe);
if (!stat.isFile() || stat.size < 100_000) {
  console.error(`Build invalido: ${outExe} tem ${stat.size} bytes.`);
  process.exit(1);
}
console.log(`Build OK: ${outExe} (${(stat.size / (1024 * 1024)).toFixed(2)} MB)`);
