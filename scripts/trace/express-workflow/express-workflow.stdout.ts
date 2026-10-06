// Runs index.ts with a fake router whose listen() invokes its callback; index.ts
// prints "Server listening on port 4242" there.
import { loadServer } from './lib/fake-express.ts';

const srv = await loadServer([]);
for (const line of srv.printed) console.log(line);
