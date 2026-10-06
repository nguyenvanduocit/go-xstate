import { createActor, fromCallback, SimulatedClock } from 'xstate';
import { view, type Step } from '../../trace.ts';
import {
  verifyCredentialsResult,
  checkReportsTableResult,
  checkBureauServiceResult,
  determineMiddleScoreResult,
  generateInterestRateResult
} from './stub-services.ts';

export async function runTrace(name: string, machine: any, steps: Step[]) {
  const clock = new SimulatedClock();
  const service = (delay: number | ((input: any) => number), result: (input: any) => unknown) =>
    fromCallback(({ input, self, sendBack }) => {
      const timer = clock.setTimeout(() => {
        let output: unknown;
        try {
          output = result(input);
        } catch (error) {
          sendBack({ type: `xstate.error.actor.${self.id}`, error, actorId: self.id });
          return;
        }
        sendBack({ type: `xstate.done.actor.${self.id}`, output, actorId: self.id });
      }, typeof delay === 'number' ? delay : delay(input));
      return () => clock.clearTimeout(timer);
    });
  const logic = machine.provide({ actors: {
    verifyCredentials: service(20, verifyCredentialsResult),
    checkReportsTable: service(40, checkReportsTableResult),
    checkBureau: service((input) => ({ EquiGavin: 100, GavUnion: 200, Gavperian: 300 })[input.bureauName]!, checkBureauServiceResult),
    determineMiddleScore: service(40, determineMiddleScoreResult),
    generateInterestRates: service(60, generateInterestRateResult)
  } });
  const actor = createActor(logic, { clock });
  actor.subscribe({ error: () => {} });
  actor.start();
  const snapshots = [{ step: 'start' as string | Step, snapshot: view(actor.getSnapshot()) }];
  for (const step of steps) {
    if ('send' in step) actor.send(step.send);
    else if ('advance' in step) clock.increment(step.advance);
    else throw new Error('Credit-check machine traces require virtual advance steps');
    snapshots.push({ step, snapshot: view(actor.getSnapshot()) });
  }
  actor.stop();
  return { name, clock: true, steps: snapshots };
}
