// All fields present: ProvisionOrder -> ApplyOrder -> End.
import { workflow, record } from './lib/record.ts';

await record('workflow-provision-orders-success', workflow, {
  order: { id: 'o-1', item: 'laptop', quantity: '10' }
});
