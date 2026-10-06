// Runs the example's own entry (main.ts) as a child process, once per session,
// in one shared temp directory so persisted-state.json carries over between
// sessions. Each stdin line is written on its own and given time to be handled,
// because main.ts treats every stdin chunk as one event.
import { mkdtempSync, readFileSync, existsSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const main = resolve(import.meta.dir, '../../../../references/xstate/examples/persisted-donut-maker/main.ts');
const preload = resolve(import.meta.dir, '../../preload.ts');
const LINE_DELAY_MS = 250;

// Session 1 starts without a persisted file and stops mid-mix; session 2 restores
// it, finishes the donut and starts another; session 3 restores the final state.
export const sessionInputs: string[][] = [
  ['NEXT', 'NEXT', 'MIXED_DRY', 'BOGUS'],
  ['MIXED_WET', 'NEXT', 'NEXT', 'NEXT', 'NEXT', 'ANOTHER_DONUT', 'NEXT'],
  ['NEXT']
];

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

export async function runSessions() {
  const dir = mkdtempSync(join(tmpdir(), 'donut-'));
  try {
    const out: { input: string[]; stdout: string; persisted: unknown }[] = [];
    for (const input of sessionInputs) {
      const proc = Bun.spawn(['bun', '--preload', preload, main], {
        cwd: dir,
        stdin: 'pipe',
        stdout: 'pipe',
        stderr: 'inherit'
      });
      const stdout = new Response(proc.stdout).text();
      await sleep(LINE_DELAY_MS * 2);
      for (const line of input) {
        proc.stdin.write(line + '\n');
        proc.stdin.flush();
        await sleep(LINE_DELAY_MS);
      }
      proc.stdin.end();
      await proc.exited;
      const file = join(dir, 'persisted-state.json');
      out.push({
        input,
        stdout: await stdout,
        persisted: existsSync(file) ? JSON.parse(readFileSync(file, 'utf8')) : null
      });
    }
    return out;
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
}
