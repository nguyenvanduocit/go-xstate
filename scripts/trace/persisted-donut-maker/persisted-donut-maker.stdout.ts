import { runSessions } from './lib/sessions.ts';
const sessions = await runSessions();
process.stdout.write(sessions.map((s) => s.stdout).join('=== next session ===\n'));
