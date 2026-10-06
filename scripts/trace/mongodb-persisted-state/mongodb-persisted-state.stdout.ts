// Runs the example's own main.ts (references/xstate/examples/mongodb-persisted-state/main.ts)
// against an in-memory fake of the `mongodb` package, three times against the same
// "database" (a fresh start, a restart that restores the persisted state, and a
// failed connection). stdin is an EventEmitter fed with one chunk per event, as the
// terminal does.
//
// console.log is replaced by a formatter with the same string/space-join behaviour
// as Node's, except that objects are printed as JSON with sorted keys (Node's
// util.inspect layout is not reproduced by the Go port) and Errors as "Name: message".
import { plugin } from 'bun';
import { EventEmitter } from 'node:events';

const sortKeys = (v: any): any =>
  Array.isArray(v)
    ? v.map(sortKeys)
    : v && typeof v === 'object'
      ? Object.fromEntries(Object.keys(v).sort().map((k) => [k, sortKeys(v[k])]))
      : v;
const canon = (v: unknown) => JSON.stringify(sortKeys(v));
// The MongoDB Node driver serializes `undefined` as null by default (ignoreUndefined: false),
// so the fake stores documents the same way (snapshot.output / snapshot.error are undefined).
const bsonClone = (v: unknown) =>
  JSON.parse(JSON.stringify(v, (_k, val) => (val === undefined ? null : val)));

let doc: any = null; // the single document of the donut-maker.donuts collection
let failConnect = false;

class FakeClient {
  constructor(_uri: string, _options: unknown) {}
  async connect() {
    if (failConnect) throw new Error('connect ECONNREFUSED');
  }
  db(_name: string) {
    return {
      collection(_name: string) {
        return {
          async findOne() {
            return doc ? bsonClone(doc) : null;
          },
          async updateOne(_filter: unknown, update: any, options: any) {
            const persistedState = bsonClone(update.$set.persistedState);
            if (doc?.persistedState !== undefined) {
              const modified = canon(doc.persistedState) !== canon(persistedState);
              doc.persistedState = persistedState;
              return {
                acknowledged: true,
                modifiedCount: modified ? 1 : 0,
                upsertedId: null,
                upsertedCount: 0,
                matchedCount: 1
              };
            }
            if (!options.upsert) {
              return { acknowledged: true, modifiedCount: 0, upsertedId: null, upsertedCount: 0, matchedCount: 0 };
            }
            doc = { _id: 'doc-1', persistedState };
            return { acknowledged: true, modifiedCount: 0, upsertedId: 'doc-1', upsertedCount: 1, matchedCount: 0 };
          }
        };
      }
    };
  }
  async close() {}
}

plugin({
  name: 'fake-mongodb',
  setup(build) {
    build.module('mongodb', () => ({
      exports: { MongoClient: FakeClient, ServerApiVersion: { v1: '1' } },
      loader: 'object'
    }));
  }
});

const stdin = new EventEmitter();
Object.defineProperty(process, 'stdin', { value: stdin });

const out: string[] = [];
const fmt = (a: unknown) =>
  typeof a === 'string' ? a : a instanceof Error ? `${a.name}: ${a.message}` : canon(a);
console.log = (...args: unknown[]) => void out.push(args.map(fmt).join(' '));

const settle = () => new Promise((r) => setTimeout(r, 10));

async function session(n: number, events: string[]) {
  out.push(`--- session ${n} ---`);
  stdin.removeAllListeners('data');
  await import(`../../../references/xstate/examples/mongodb-persisted-state/main.ts?session=${n}`);
  await settle();
  for (const e of events) {
    stdin.emit('data', Buffer.from(e + '\n'));
    await settle();
  }
}

await session(1, ['NEXT', 'BOGUS', 'NEXT', 'MIXED_DRY', '']);
await session(2, ['MIXED_WET', 'NEXT', 'NEXT', 'NEXT', 'NEXT', 'ANOTHER_DONUT']);
failConnect = true;
await session(3, []);

process.stdout.write(out.join('\n') + '\n');
process.exit(0);
