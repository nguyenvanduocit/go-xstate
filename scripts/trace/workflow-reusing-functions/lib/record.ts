import { createActor,fromPromise } from 'xstate';
import { view } from '../../trace.ts';
import { workflow,parentWorkflow } from './load.ts';
export async function record(name:string,amount:number,failure=''){
 const log=console.log;const events:any[]=[];console.log=(...args:any[])=>{if(args[0]==='Received event') events.push(JSON.parse(JSON.stringify(args[1])));};
 const timer=globalThis.setTimeout;
 globalThis.setTimeout=((fn:any,ms:number,...args:any[])=>timer(fn,ms===1000?200:ms,...args)) as any;
 if(failure)(workflow as any).implementations.actors[failure]=fromPromise(async()=>{await new Promise(r=>timer(r,200));throw new Error('service failed');});
 const actor=createActor(parentWorkflow);actor.subscribe({error:()=>{}});actor.start();
 const snapshot=()=>({parent:view(actor.getSnapshot()),child:view(actor.getSnapshot().children.paymentconfirmation!.getSnapshot()),events:[...events]});
 const steps:any[]=[{step:'start',snapshot:snapshot()}];
 const script:any[]=[{send:{type:'OTHER'}},{send:{type:'PaymentReceivedEvent',accountId:'1234',payment:{amount},customer:{name:'John Doe'},funds:{available:false}}},{wait:250},{wait:250},{send:{type:'PaymentReceivedEvent',accountId:'other',payment:{amount:5},customer:{name:'Other'},funds:{available:true}}}];
 for(const step of script){if(step.send)actor.send(step.send);else await new Promise(r=>timer(r,step.wait));steps.push({step,snapshot:snapshot()});}
 actor.stop();log(JSON.stringify({name,steps},null,2));
}
