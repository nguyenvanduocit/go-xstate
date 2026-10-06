// Store config of <Counter initialCount={...}/> in
// ../../../../references/xstate/examples/local-store-counter-react/src/App.tsx:5-20.
// App.tsx cannot be imported (React, CSS and @xstate/store-react are not
// resolvable by the trace runner), so the config is repeated here verbatim.
export function counterConfig(initialCount: number) {
  return {
    context: {
      count: initialCount
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
}

// Steps shared by every trace. Covers both `on` handlers, a repeated inc, an
// inc after reset, a negative `by`, and an event type with no handler.
export const steps = [
  { send: { type: 'inc', by: 1 } },
  { send: { type: 'inc', by: 5 } },
  { send: { type: 'reset' } },
  { send: { type: 'inc', by: 1 } },
  { send: { type: 'inc', by: -3 } },
  { send: { type: 'unknown' } },
  { send: { type: 'reset' } }
];
