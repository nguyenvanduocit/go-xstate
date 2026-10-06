import { createActor, fromPromise, SimulatedClock } from 'xstate';
import { view } from '../../trace.ts';
import { load } from './load.ts';
export async function record(name: string, events: string[], deadline = false, fail = false) {
  const { workflow } = await load();
  const clock = new SimulatedClock();
  const actor = createActor(workflow.provide({actions:Object.fromEntries(['logNewOrderCreated','logOrderConfirmed','logOrderShipped','logOrderFinished','logOrderCancelled'].map(name=>[name,()=>{}])), actors:{CancelOrder:fromPromise(async()=>{
    await new Promise(resolve=>setTimeout(resolve,100));
    if(fail) throw new Error('cancel failed');
  })}}), {clock});
  actor.subscribe({error:()=>{}});
  actor.start();
  const steps: any[] = [{step:'start',snapshot:view(actor.getSnapshot())}];
  const script: any[] = [{send:{type:'OrderFinishedEvent'}},...events.map(type=>({send:{type}})), {advance:14999}, {advance:1}];
  if(deadline) script.push({wait:150});
  script.push({send:{type:'ShipmentSentEvent'}},{advance:15000});
  for(const step of script) {
    if(step.send) actor.send(step.send);
    else if(step.advance) clock.increment(step.advance);
    else await new Promise(resolve=>setTimeout(resolve,step.wait));
    steps.push({step,snapshot:view(actor.getSnapshot())});
  }
  actor.stop();
  console.log(JSON.stringify({name,clock:true,steps},null,2));
}
