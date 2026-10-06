import { fromPromise } from 'xstate';
import { runTrace } from '../../trace.ts';
import { load } from './load.ts';
export async function record(failure = '') {
  const { workflow } = await load();
  const actors = Object.fromEntries(['StoreNewPatientInfo', 'AssignDoctor', 'ScheduleAppt'].map(name => [name, fromPromise(async () => {
    await new Promise(resolve => setTimeout(resolve, 200));
    if (name === failure) throw new Error('service failed');
  })]));
  const steps: any[] = [{send:{type:'ignored'}}, {send:{type:'NewPatientEvent',name:'John Doe',condition:'Broken Arm'}}];
  for (let i=0;i<3;i++) steps.push({wait:250});
  steps.push({send:{type:'NewPatientEvent',name:'Jane',condition:'Cold'}});
  console.log(JSON.stringify(await runTrace(failure || 'success', workflow.provide({actors}), steps), null, 2));
}
