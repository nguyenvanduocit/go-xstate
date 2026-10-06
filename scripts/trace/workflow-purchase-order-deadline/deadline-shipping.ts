import { record } from './lib/record.ts';
await record('deadline-shipping',['OrderCreatedEvent','OrderConfirmedEvent'],true);
