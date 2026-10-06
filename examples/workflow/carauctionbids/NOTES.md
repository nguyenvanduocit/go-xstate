# workflow-car-auction-bids

Ported from `references/xstate/examples/workflow-car-auction-bids/main.ts`.

- `machine.go`: `Machine()` (= JS `workflow`: id `handleCarAuctionBid`, states `StoreCarAuctionBid` / `BiddingEnded`, `CarBidEvent` assign, `after: BiddingDelay` = 3000 ms, final-state `output`), `Bid`/`Bidder`/`Context`/`Output` structs with the JS key names, `WinningBid` (the reduce of the output function), `CarBidEvent` event builder.
- `run.go`: `Run(w)` = the runnable part of `main.ts` (inspect listener, subscriber, two bids one second apart, real clock).
- `inspect.go`: formats values like Bun's `console.log` so `Run` prints the same text as the JS entry.

## Not ported
- `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `.gitignore`: package manager and TypeScript setup (`vite-node` runner, `tsc && vite build` scripts); there is no UI, no HTML and no server in this app.
- The TypeScript `types: {} as {...}` declaration and the `Bid` interface: erased at runtime; their shape lives in the Go structs.
- `inspect.go` supports only the value shapes this example prints (structs, maps, slices, strings, numbers, booleans, nil), not a general `util.inspect`. Go maps carry no insertion order, so an `xs.E` event prints `type` first and its other keys sorted (JS prints insertion order; identical for the two event shapes of this app).

## Trace coverage
- `testdata/workflow-car-auction-bids.golden.json` (`scripts/trace/workflow-car-auction-bids/workflow-car-auction-bids.ts`, SimulatedClock, imports the example's real `workflow` export): initial snapshot, an unknown event (ignored), `CarBidEvent` x4 (increasing, lower, equal amounts), `advance` 999 (BiddingDelay not due) then 1 (fires, `StoreCarAuctionBid` -> final `BiddingEnded`, status `done`), a bid after completion (ignored), a further `advance`. The `after` timer starts at machine start and is not reset by bids.
- `testdata/no-bids.golden.json` (`no-bids.ts`): BiddingDelay fires with no bids; the final state's `output` reduces an empty array and throws, so the snapshot has status `error`, value still `StoreCarAuctionBid`, error `reduce of empty array with no initial value` (JavaScriptCore wording, Bun runtime).
- `testdata/winning-bid.golden.json` (`winning-bid.ts`): the root snapshot's `output` is always `null` because xstate only evaluates the root `output` (`packages/core/src/stateUtils.ts:1146`), never the final state's one, so the winning bid is not observable through a snapshot. The script calls the machine's own `BiddingEnded.output` function on 7 bid lists (single, increasing, decreasing, tie goes to the later bid, maximum in the middle, tie among three, empty -> throws); `TestWinningBid` checks `WinningBid` against it.
- `testdata/workflow-car-auction-bids.stdout.txt` (`workflow-car-auction-bids.stdout.ts`, runs `main.ts` with real timers, about 3 s): `TestRun` compares `Run`'s output byte for byte. It prints `workflow completed undefined` for the same reason as above.
- The machine has no guards, so there is no guard branch to cover; `WinningBid` is the only conditional and is covered by `winning-bid`.
