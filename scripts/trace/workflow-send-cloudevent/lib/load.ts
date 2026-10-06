import { plugin } from 'bun';
import { readFileSync } from 'node:fs';
plugin({name:'cloud-reference',setup(build){build.onLoad({filter:/workflow-send-cloudevent\/main\.ts$/},args=>({contents:readFileSync(args.path,'utf8').split('const actor = createActor(workflow,')[0],loader:'ts'}));}});
export const {workflow}=await import('../../../../references/xstate/examples/workflow-send-cloudevent/main.ts');
