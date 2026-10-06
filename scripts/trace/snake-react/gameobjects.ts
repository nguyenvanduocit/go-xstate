// getGamObjectAtPos over every cell (plus one cell outside) of a mid-game context.
import { getGamObjectAtPos, createInitialContext } from '../../../references/xstate/examples/snake-react/src/snakeMachine.ts';

const base = createInitialContext();
const contexts = {
  initial: base,
  // head turned Up, longer body with differing dirs, apple elsewhere
  midgame: {
    ...base,
    dir: 'Up' as const,
    apple: { x: 3, y: 3 },
    snake: [
      { x: 10, y: 5, dir: 'Up' as const },
      { x: 10, y: 6, dir: 'Up' as const },
      { x: 11, y: 6, dir: 'Right' as const },
      { x: 12, y: 6, dir: 'Right' as const }
    ]
  }
};

const cases: any[] = [];
for (const [name, context] of Object.entries(contexts)) {
  for (let y = -1; y <= base.gridSize.y; y++) {
    for (let x = -1; x <= base.gridSize.x; x++) {
      const result = getGamObjectAtPos(context, { x, y });
      cases.push({ context: name, p: { x, y }, result: result ?? null });
    }
  }
}
console.log(JSON.stringify({ name: 'snake-react/gameobjects', contexts, cases }, null, 2));
