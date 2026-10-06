import { temperatureMachine } from '../../../references/xstate/examples/7guis-temperature-react/src/temperatureMachine.ts';
import { runTrace } from '../trace.ts';
const c = (value: string) => ({ send: { type: 'CELSIUS', value } });
const f = (value: string) => ({ send: { type: 'FAHRENHEIT', value } });
const t = await runTrace('7guis-temperature-react', temperatureMachine, [
  c('100'), // non-empty CELSIUS branch
  c(''), // empty CELSIUS branch
  c('-40'),
  c('36.6'),
  c('1e2'), // exponent literal
  c(' 12 '), // surrounding whitespace is trimmed by unary +
  c('0x10'), // hex literal
  c(' '), // whitespace only: non-empty string, +' ' === 0
  c('abc'), // NaN serialises as null
  c('Infinity'), // Infinity serialises as null
  f('212'), // non-empty FAHRENHEIT branch
  f(''), // empty FAHRENHEIT branch
  f('-40'),
  f('98.6'),
  f('.5'),
  f('abc'),
  f('-Infinity'),
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify(t, null, 2));
