import { mock } from 'bun:test';
import { createActor, fromCallback, SimulatedClock } from 'xstate';
import { view, type Step } from '../../trace.ts';

// Avoid loading the upstream logger and launching ffprobe or filesystem writes.
mock.module('../../../../references/xstate/examples/workflow-media-scanner/src/fileHandlers.ts', () => ({
  scanDirectories: async () => { throw new Error('fixture scan failure'); },
  checkFilePermissions: async () => { throw new Error('unexpected permission check'); },
  evaluateFiles: async () => { throw new Error('unexpected probe'); },
  moveFiles: async () => { throw new Error('unexpected move'); }
}));
export const { mediaScannerMachine } = await import('../../../../references/xstate/examples/workflow-media-scanner/src/mediaScannerMachine.ts');

export const input = { basePath: '/library', destinationPath: '/archive' };
export async function record(scenario: string) {
  const clock = new SimulatedClock();
  const service = (id: string, output: unknown, failure?: unknown) => fromCallback(({sendBack}) => {
    const timer = clock.setTimeout(() => sendBack(failure === undefined
      ? {type: `xstate.done.actor.${id}`, output, actorId: id}
      : {type: `xstate.error.actor.${id}`, error: failure, actorId: id}), 1);
    return () => clock.clearTimeout(timer);
  });
  const machine = mediaScannerMachine.provide({
    actors: {
      scanLibrary: service('scanLibrary', ['/library/movie', '/library/denied'], scenario === 'scan-error' ? 'scan failed' : undefined),
      checkFilePermissions: service('checkFilePermissions', {dirsToEvaluate: ['/library/movie'], dirsToReport: ['/library/denied']},
        scenario === 'permissions-error' ? {message: 'Error checking file permissions', error: {message: 'No accessible files found to move', dirsToReport: ['/library/denied']}} :
        scenario === 'permissions-report' ? {dirsToReport: ['/library/denied']} : undefined),
      evaluateFiles: service('evaluatingFiles', {dirsToMove: ['/library/movie/video.mkv']}, scenario === 'evaluate-error' ? 'evaluate failed' : undefined),
      moveFiles: service('moveFiles', {message: scenario === 'partial-move' ? 'files moved with errors' : 'all files moved successfully'}, scenario === 'move-error' ? 'move failed' : undefined)
    },
    actions: {emailErrors: () => {}}
  });
  const steps: Step[] = [{send: {type: 'UNKNOWN'}}, {send: {type: 'START_SCAN'}}, {send: {type: 'START_SCAN'}}, {advance: 1}, {advance: 1}, {advance: 1}, {advance: 1}, {send: {type: 'RESTART'}}, {send: {type: 'START_SCAN'}}, {advance: 1}];
  const actor = createActor(machine, {input, clock});
  actor.subscribe({error: () => {}});
  actor.start();
  const snapshots = [{step: 'start' as string | Step, snapshot: view(actor.getSnapshot())}];
  for (const step of steps) {
    if ('send' in step) actor.send(step.send);
    else if ('advance' in step) clock.increment(step.advance);
    snapshots.push({step, snapshot: view(actor.getSnapshot())});
  }
  actor.stop();
  console.log(JSON.stringify({name: `workflow-media-scanner/${scenario}`, clock: true, input, steps: snapshots}, null, 2));
}
