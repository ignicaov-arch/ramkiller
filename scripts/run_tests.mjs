import { execFileSync } from 'child_process';
import fs from 'fs';
import path from 'path';

const root = process.cwd();
const goBin = fs.existsSync('C:\\Program Files\\Go\\bin\\go.exe') ? 'C:\\Program Files\\Go\\bin\\go.exe' : 'go';
const env = { ...process.env, PATH: `${path.dirname(goBin)};${process.env.PATH || ''}` };

execFileSync(goBin, ['test', '-v', '-cover', './...'], { cwd: root, stdio: 'inherit', env });
