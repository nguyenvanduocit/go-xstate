import { plugin } from 'bun';
import { readFileSync } from 'node:fs';
plugin({name:'payment-reference',setup(build){build.onLoad({filter:/workflow-reusing-functions\/main\.ts$/},args=>({contents:readFileSync(args.path,'utf8').split('const actor = createActor(parentWorkflow);')[0].replace('const parentWorkflow =','export const parentWorkflow ='),loader:'ts'}));}});
export const {workflow,parentWorkflow}=await import('../../../../references/xstate/examples/workflow-reusing-functions/main.ts');
