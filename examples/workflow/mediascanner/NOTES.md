# Media scanner

Ports `references/xstate/examples/workflow-media-scanner/src/`: the machine,
four file helpers, and the console entry. The machine retains all six states,
four invokes, assignments, event names, and the `emailErrors` action.
There are no machine guards or delays.

`NewMachine` accepts actor implementations and an output writer.
`DefaultActors` connects the promise actors to `Handlers`. `DefaultHandlers`
uses the filesystem, read/write access checks, an `ffprobe` subprocess, and
directory moves with overwrite. `Run` uses the upstream placeholder paths;
`RunWith` accepts paths, services, and a cancellation context for one scan.
The machine itself remains restartable and has no final state.

## Recorded evidence

The scripts import the original upstream machine without changing its states.
Clock-driven callback actors replace only the four promise boundaries and send
their completion/error events after one simulated millisecond. This makes every
intermediate state reproducible without filesystem writes or real-time waits.
Go replays all snapshots with `tracetest.Run` and checks each actor's input.
The actual Go promise adapters also run a complete scan against temporary files.

- `success`: all successful workflow transitions, ignored events, return to
  idle, and a second scan retaining context from the first.
- `scan-error`: scanning rejection, error report, restart, and scan again.
- `permissions-error`: helper-shaped nested permission rejection; the machine
  reads the absent top-level `dirsToReport`, so the JSON property disappears.
- `permissions-report`: top-level permission report assignment and restart.
- `evaluate-error`: unhandled evaluation rejection stops the actor in
  `EvaluatingFiles`; subsequent events have no effect.
- `move-error`: move actor rejection and restart.
- `partial-move`: a resolved move result containing failures still returns idle.
- `workflow-media-scanner.stdout`: imports the original `src/index.ts` with
  a deterministic scan rejection and compares its console output to `RunWith`.

## Preserved helper behavior

Directory scans follow symlinks when checking whether entries are directories.
File extensions are case-sensitive; both dimensions must exceed 1920 by 1080.
Only the first probe stream is examined. A read or probe error skips the rest
of that directory and evaluation continues with the next one. No matches reject.
Moves operate on each selected file's whole parent directory. Multiple selected
files from one directory can therefore produce a second move failure. Individual
move failures resolve as a report rather than rejecting the machine's invoke.

Tests cover these behaviors, inaccessible and missing paths, overwrites, retained
sidecar files, the subprocess arguments and JSON contract, cancellation, and
recursive copying with symlinks. All file writes and moves in tests use temporary
directories. The copy fallback is tested directly; a real cross-device mount is
not required by the suite.

## Environment and exclusions

- A real `ffprobe` installation and real media codecs are environment requirements.
  Tests use an injected probe and a temporary executable to verify its adapter.
- Winston's file transports (`error.log`, `combined.log`), structured metadata,
  and platform-specific error text are replaced by messages to an injected writer.
  No email is sent upstream: `emailErrors` only prints its message here as well.
- Permission checks run in input order instead of `Promise.all`; their result
  arrays keep the upstream input order. Access checks use Unix `access(2)`.
- Go errors expose nested failures through `FileError` and `Unwrap`; JavaScript
  plain-object rejections become typed errors. Undefined `dirsToReport` remains
  distinct from an empty array in serialized context.
- Directory moves reject replacing a source ancestor before overwrite. The
  recursive fallback copies regular files and symlinks, rejecting special files.
- Entry stdout parity covers the recorded placeholder-path rejection. The compact
  snapshot formatter does not implement Bun's general-purpose object inspector
  for arbitrary path lengths or nested error objects.
- Node package metadata, lockfiles, TypeScript configuration, and README install
  instructions are replaced by the enclosing Go module and this package's API.
