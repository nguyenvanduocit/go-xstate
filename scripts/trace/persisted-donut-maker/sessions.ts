import { runSessions } from './lib/sessions.ts';
console.log(JSON.stringify({ name: 'persisted-donut-maker-sessions', sessions: await runSessions() }, null, 2));
