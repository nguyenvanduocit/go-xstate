import { createActor, toPromise } from 'xstate';
import { workflow } from './lib/load.ts';
const actor=createActor((workflow as any).implementations.actors.provisionOrdersFunction,{input:{orders:[]}});
actor.start();
console.log(JSON.stringify(await toPromise(actor)));
