// Empty quantity: onError guard 3 -> Exception.MissingQuantity.
import { workflow, record } from './lib/record.ts';

await record('workflow-provision-orders-missing-quantity', workflow, {
  order: { id: 'o-1', item: 'laptop', quantity: '' }
});
