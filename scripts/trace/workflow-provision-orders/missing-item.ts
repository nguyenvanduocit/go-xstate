// Empty item: onError guard 2 -> Exception.MissingItem.
import { workflow, record } from './lib/record.ts';

await record('workflow-provision-orders-missing-item', workflow, {
  order: { id: 'o-1', item: '', quantity: '10' }
});
