// Empty id: onError guard 1 -> Exception.MissingId -> Exception.End -> End.
import { workflow, record } from './lib/record.ts';

await record('workflow-provision-orders-missing-id', workflow, {
  order: { id: '', item: 'laptop', quantity: '10' }
});
