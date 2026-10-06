import { record } from './lib/record.ts';
await record('success',['OrderCreatedEvent','OrderConfirmedEvent','ShipmentSentEvent']);
