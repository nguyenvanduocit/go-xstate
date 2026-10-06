import { createActor, fromPromise } from 'xstate';
import { view } from '../../trace.ts';
import { workflow } from './load.ts';
export async function record(name: string,orders: any[],fail=false){
 const log=console.log;console.log=()=>{};
 const timer=globalThis.setTimeout;
 globalThis.setTimeout=((fn:any,ms:number,...args:any[])=>timer(fn,ms===1000?100:ms,...args)) as any;
 const logic=fail?workflow.provide({actors:{provisionOrdersFunction:fromPromise(async()=>{await new Promise(r=>timer(r,100));throw new Error('provision failed');})}}):orders.length===0?workflow.provide({actors:{provisionOrdersFunction:fromPromise(async()=>{await new Promise(r=>timer(r,100));return [];})}}):workflow;
 const actor=createActor(logic,{input:{orders}});actor.subscribe({error:()=>{}});actor.start();
 const steps:any[]=[{step:'start',snapshot:view(actor.getSnapshot())}];
 await new Promise(r=>timer(r,150));steps.push({step:{wait:150},snapshot:view(actor.getSnapshot())});
 actor.send({type:'ignored'} as any);steps.push({step:{send:{type:'ignored'}},snapshot:view(actor.getSnapshot())});actor.stop();
 log(JSON.stringify({name,input:{orders},steps},null,2));
}
