import { plugin } from 'bun';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

export async function load(entry = false) {
  plugin({ name: 'patient-reference', setup(build) {
    build.onResolve({ filter: /^cockatiel$/ }, () => ({ path: resolve(import.meta.dir, './deps/node_modules/cockatiel/dist/index.js') }));
    if (!entry) build.onLoad({ filter: /workflow-new-patient-onboarding\/main\.ts$/ }, args => ({
      contents: readFileSync(args.path, 'utf8').split('const actor = createActor(workflow);')[0], loader: 'ts'
    }));
  }});
  return import('../../../../references/xstate/examples/workflow-new-patient-onboarding/main.ts');
}
