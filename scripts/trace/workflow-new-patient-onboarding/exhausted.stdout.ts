import { load } from './lib/load.ts';
Math.random = () => 0;
const timer = globalThis.setTimeout;
globalThis.setTimeout = ((fn: any, _ms: number, ...args: any[]) => timer(fn, 1, ...args)) as any;
await load(true);
