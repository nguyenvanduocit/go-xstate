// Runs the example's own entry (main.ts) deterministically: Math.random is a fixed
// sequence and setInterval hands its callback to this script, which fires it once
// per scripted tick instead of every 3000 ms. The printed text is main.ts's own output.
const picks = [0.0, 0.5, 0.9, 0.4, 0.7, 0.1]; // floor(r * 3) = 0, 1, 2, 1, 2, 0
let i = 0;
Math.random = () => picks[i++];
let tick: () => void = () => {};
(globalThis as any).setInterval = (fn: () => void) => {
  tick = fn;
  return 0;
};
await import('../../../references/xstate/examples/workflow-monitor-patient/main.ts');
for (let n = 0; n < picks.length; n++) tick();
process.exit(0);
