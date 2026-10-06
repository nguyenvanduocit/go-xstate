// Store config of the module-level `createStore` call in
// ../../../../references/xstate/examples/store-counter-react/src/App.tsx:6-21.
// App.tsx cannot be imported (React, CSS, @xstate/store-react and
// @statelyai/inspect are not resolvable by the trace runner), so the config is
// repeated here verbatim.
export const counterConfig = {
  context: {
    count: 0
  },
  on: {
    inc: (context: { count: number }, event: { by: number }) => {
      return {
        count: context.count + event.by
      };
    },
    reset: () => {
      return {
        count: 0
      };
    }
  }
};

// Steps shared by every trace. Covers both `on` handlers, a repeated inc, an
// inc after reset, a negative `by`, a reset of an already-zero count, and an
// event type with no handler.
export const steps = [
  { send: { type: 'inc', by: 1 } },
  { send: { type: 'inc', by: 1 } },
  { send: { type: 'inc', by: 5 } },
  { send: { type: 'reset' } },
  { send: { type: 'inc', by: 1 } },
  { send: { type: 'inc', by: -3 } },
  { send: { type: 'unknown' } },
  { send: { type: 'reset' } },
  { send: { type: 'reset' } }
];
