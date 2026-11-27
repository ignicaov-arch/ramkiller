import { execFileSync } from 'child_process';
import fs from 'fs';
import path from 'path';

const root = process.cwd();
const IGNORE = new Set(['.git', 'node_modules', 'dist', '.skills-gates', '.visual-audit', 'scratch']);
const EXTS = new Set(['.go', '.mjs', '.js', '.ts']);

function resolveGoBinary() {
  const candidates = ['C:\\Program Files\\Go\\bin\\go.exe', 'go'];
  for (const c of candidates) {
    if (c === 'go' || fs.existsSync(c)) return c;
  }
  return 'go';
}

function walk(dir, out = []) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (IGNORE.has(entry.name)) continue;
    const full = path.join(dir, entry.name);
    if (entry.isDirectory()) walk(full, out);
    else if (entry.isFile() && EXTS.has(path.extname(entry.name).toLowerCase())) out.push(full);
  }
  return out;
}

const files = walk(root);
const violations = [];
for (const f of files) {
  const rel = path.relative(root, f).replace(/\\/g, '/');
  const raw = fs.readFileSync(f, 'utf8');
  const trimmed = raw.endsWith('\n') ? raw.slice(0, -1).replace(/\r$/, '') : raw;
  const lines = trimmed ? trimmed.split('\n').length : 0;
  if (lines > 200) violations.push(`${rel}: ${lines} linhas (limite 200)`);
}

if (violations.length > 0) {
  console.error('Violacao da regra modular (< 200 linhas):\n' + violations.join('\n'));
  process.exit(1);
}

const goBin = resolveGoBinary();
const goDir = path.dirname(goBin);
const env = { ...process.env, PATH: `${goDir};${process.env.PATH || ''}` };
execFileSync(goBin, ['vet', './...'], { cwd: root, stdio: 'inherit', env });
console.log(`Lint OK: ${files.length} arquivos verificados (<=200L) e go vet aprovado.`);
