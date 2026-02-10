import { cpSync, existsSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.resolve(__dirname, '..');

const sourceDir = path.resolve(projectRoot, 'dist', 'web');
const targetDir = path.resolve(projectRoot, '..', '..', 'webassets', 'vtuber');

if (!existsSync(sourceDir)) {
  console.error(`[sync:webassets] Build output not found: ${sourceDir}`);
  process.exit(1);
}

rmSync(targetDir, { recursive: true, force: true });
mkdirSync(targetDir, { recursive: true });
cpSync(sourceDir, targetDir, { recursive: true, force: true });

console.log(`[sync:webassets] Copied ${sourceDir} -> ${targetDir}`);
