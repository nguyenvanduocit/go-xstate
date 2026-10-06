import { gameStore, getGameOutcome } from './lib/load.ts';
import { steps } from './lib/steps.ts';

// The real code path of App.tsx: the exported `gameStore`, `store.inspect(fn)`
// (a recorder), and the three selectors App.tsx reads through useSelector:
// `context.board`, `context.currentPlayer`, `context.status`.
// Golden: { name, steps: [ { step, snapshot: {status, context, outcome, inspected, notified} } ] }
//   outcome:   getGameOutcome(context.board)
//   inspected: inspection events received during the step: { type, event, context }
//   notified:  values the selector subscriptions received during the step
const board = gameStore.select((ctx) => ctx.board);
const currentPlayer = gameStore.select((ctx) => ctx.currentPlayer);
const status = gameStore.select((ctx) => ctx.status);

let inspected: any[] = [];
let notified: any = { board: [], currentPlayer: [], status: [] };
gameStore.inspect((e: any) => {
  inspected.push({ type: e.type, event: e.event, context: e.snapshot.context });
});
board.subscribe((v) => notified.board.push(v));
currentPlayer.subscribe((v) => notified.currentPlayer.push(v));
status.subscribe((v) => notified.status.push(v));

const view = () => {
  const s = gameStore.getSnapshot();
  const out = JSON.parse(
    JSON.stringify({
      status: s.status,
      context: s.context,
      outcome: getGameOutcome(s.context.board),
      inspected,
      notified
    })
  );
  inspected = [];
  notified = { board: [], currentPlayer: [], status: [] };
  return out;
};

const recorded: any[] = [{ step: 'start', snapshot: view() }];
for (const step of steps) {
  gameStore.send(step.send as any);
  recorded.push({ step, snapshot: view() });
}
console.log(JSON.stringify({ name: 'store-tic-tac-toe/store', steps: recorded }, null, 2));
