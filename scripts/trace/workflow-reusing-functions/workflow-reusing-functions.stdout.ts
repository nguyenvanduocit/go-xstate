import {view} from '../trace.ts';
const log=console.log;
console.log=(...args:any[])=>{
 if(args.length===1 && args[0]?.machine) log(JSON.stringify(view(args[0])));
 else log(...args);
};
const timer=globalThis.setTimeout;
globalThis.setTimeout=((fn:any,_ms:number,...args:any[])=>timer(fn,1,...args)) as any;
await import('../../../references/xstate/examples/workflow-reusing-functions/main.ts');
