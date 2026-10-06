import { load } from './lib/load.ts';
let calls = 0;
Math.random = () => calls++ === 0 ? 0 : 1;
const timer = globalThis.setTimeout;
globalThis.setTimeout = ((fn: any, _ms: number, ...args: any[]) => timer(fn, 1, ...args)) as any;
await load(true);
