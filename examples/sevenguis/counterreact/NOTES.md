# 7guis-counter-react

Ported from `references/xstate/examples/7guis-counter-react/src/counterMachine.ts`.

Go package name is `sevenguiscounterreact`: the directory name starts with a digit, which is not a valid Go identifier.

## Not ported
- `src/App.tsx`, `src/main.tsx`: React components, `useMachine` hook and `ReactDOM` rendering.
- `src/App.css`, `src/index.css`, `src/favicon.svg`, `index.html`: styling and HTML shell.
- `vite.config.ts`, `tsconfig.json`, `package.json`, `pnpm-lock.yaml`, `src/vite-env.d.ts`: bundler and TypeScript setup.

## Trace coverage (`testdata/counter.golden.json`, recorded by `scripts/trace/7guis-counter-react/counter.ts`)
- initial context, `INCREMENT` x3, and an unknown event (no change).
- The machine has no states, guards, delays or invokes, so there is nothing else to cover.
