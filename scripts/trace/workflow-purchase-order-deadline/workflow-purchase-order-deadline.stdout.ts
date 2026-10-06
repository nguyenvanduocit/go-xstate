import { load } from './lib/load.ts';
const timer = globalThis.setTimeout;
globalThis.setTimeout = ((fn: any, ms: number, ...args: any[]) => timer(fn, ms/100, ...args)) as any;
Math.random = () => 1;
await load(true);
