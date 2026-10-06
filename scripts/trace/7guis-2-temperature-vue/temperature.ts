import { tempMachine } from '../../../references/xstate/examples/7guis-2-temperature-vue/src/tempMachine.ts';
import { runTrace } from '../trace.ts';
const c = (value: string) => ({ send: { type: 'changeC', value } });
const f = (value: string) => ({ send: { type: 'changeF', value } });
const t = await runTrace('7guis-2-temperature-vue', tempMachine, [
  c('100'), // guard true, onChangeC non-empty
  c(''), // onChangeC whitespace-only branch (empty)
  c('-40'),
  c('36.6'),
  c('2.5'), // Math.round half up (36.5 -> 37)
  c('-37.5'), // Math.round of -35.5 is -35 (half toward +Infinity)
  c('1e2'), // exponent literal
  c(' 12 '), // surrounding whitespace is trimmed by unary +
  c('0x10'), // hex literal
  c('0b11'), // binary literal
  c('0o17'), // octal literal
  c('.5'),
  c('5.'),
  c('+7'),
  c(' '), // whitespace only: onChangeC clears both fields
  c('abc'), // guard false (NaN): no change
  c('1_0'), // underscore separators are NaN in JS
  c('1e'), // incomplete exponent is NaN
  c('-0x10'), // signed hex is NaN
  c('Infinity'), // Infinity serialises as null
  c('-0'), // negative zero serialises as 0
  f('212'), // guard true, onChangeF non-empty
  f(''), // onChangeF whitespace-only branch
  f('-40'),
  f('98.6'),
  f('-39.1'), // Math.round of -39.5 is -39
  f('.5'),
  f('\t32\n'), // tab/newline whitespace
  f('abc'), // guard false: no change
  f('-Infinity'),
  f('0x20'),
  { send: { type: 'unknown' } }
]);
console.log(JSON.stringify(t, null, 2));
