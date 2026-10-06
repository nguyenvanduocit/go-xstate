import { record } from './lib/record.ts';
await record('orders',[{id:'123',item:'laptop',quantity:'10'},{id:'456',item:'desktop',quantity:'4'}]);
