# trivia-game-example

Ported from `references/xstate/examples/trivia-game-example/src/triviaMachine.ts` plus the non-UI helpers it uses
(`src/services/RickApi.tsx`, `src/common/constants.ts`, `src/common/types.ts`).
Go: `machine.go` (`Machine`, `NewMachine`, `Context`), `rickapi.go` (`RickAPI`, `Character`, `Episode`).

## Not ported
- All React UI: `src/App.tsx`, `src/main.tsx`, `src/pages/`, `src/components/`, `src/context/AppContext.tsx`
  (`AppContext` is a React provider around the machine), the types `ClueProps`, `PropsNode`, `State` in `src/common/types.ts`.
- Styling and shell: `src/*.css`, `src/styles/global.scss`, `tailwind.config.js`, `postcss.config.js`, `index.html`.
- Bundler / tooling: `vite.config.ts`, `tsconfig*.json`, `.eslintrc.cjs`, `package.json`, `pnpm-lock.yaml`, `README.md`.
- `axios` is replaced by `net/http` in `RickAPI`; the base URL, `http.Client`, `Math.random` and the error log are fields of `RickAPI`
  so tests use `httptest`. A failed request keeps the JS `.catch(console.log)` behaviour: it is logged and yields the zero value
  (`nil` slice / `nil` pointer, where JS resolves with `undefined`). In the snapshot that is JSON `null`, where the JS context key would be
  dropped by `JSON.stringify` (not exercised by the trace, which uses stubs).
- The real network actors are only exercised against a local fake server (`TestMachineWithRickAPI`, `TestRickAPI*`); the public
  Rick and Morty API is never called by the tests.
- The `isAnswerCorrect` guard's `assertEvent` becomes a panic on any other event type (the same failure mode as the JS throw); the guard is only wired to
  `user.selectAnswer`, so it is unreachable in practice.
- Unreachable branches kept as in JS, not covered by any trace: guard `hasLostGame` in `correctAnswer` and `hasWonGame` in `incorrectAnswer`
  (lives only drop in `incorrectAnswer`, points only rise in `correctAnswer`), and the `!context.currentCharacter` early return of `isAnswerCorrect`
  (a question is only ready after `loadCharacter` assigned it, except when the real API failed and returned `undefined`).

## Trace coverage (`testdata/trivia-game-example.golden.json`, recorded by `scripts/trace/trivia-game-example/trivia-game-example.ts`)
The three promise actors are replaced with deterministic stubs through `machine.provide` (Go: `Machine().Provide`): the home page resolves
characters 1 and 2, `loadSingleCharacter` resolves ids 101, 102, ... in call order, `loadRandomCharacters` resolves 201..203. Waits use real time
(`{wait: 100}`); the first two stubs sleep 10 ms so the snapshot read right after a send is still `loading` in Go (a goroutine would otherwise win the race;
JS promises cannot resolve before the synchronous read).
- every state: `homepage.loadingData`, `homepage.dataLoaded`, `instructionModal`, `startTrivia.loadQuestionData.loadCharacter`, `.loadRandomCharacters`
  (transient, not observable between steps), `questionReady.questionStart`, `correctAnswer`, `incorrectAnswer`, `lostGame`, `wonGame`.
- invoke `onDone` of all three actors with their `assign`s (`homePageCharacters` + `hasLoaded`, `currentCharacter`, `randomCharacters` + `question + 1` + `hasLoaded`);
  `children` shows the invoked actor while loading.
- `user.play`, `user.close`, `user.reject` (both back to `homepage.dataLoaded` without reloading), `user.accept` (assign `hasLoaded: false`, enters `startTrivia`: `goToTriviaPage`, `resetTriviaData`).
- `user.selectAnswer`: both guard branches (`isAnswerCorrect` and `not(isAnswerCorrect)`); entry assigns `points + 10` and `lifes - 1`.
- eventless `always` after an answer: neither guard true (stay), `hasLostGame` (3 wrong answers across 4 questions), `hasWonGame` (10 correct answers, 100 points, one wrong in between).
- `user.nextQuestion` -> `#loadQuestionData` (entry re-run), `user.playAgain` from `lostGame` and `wonGame` -> `#startTrivia` (reset of points, lifes, question, currentCharacter, randomCharacters; `isClueOpened` is kept).
- `user.toggleClue` handled by `questionReady` from `questionStart`, `incorrectAnswer` and `lostGame`; open/close/open and carry-over across questions.
- ignored events: `user.play`/`toggleClue`/`selectAnswer` on the homepage, `nextQuestion`/`playAgain` in `questionStart`, `selectAnswer` in `incorrectAnswer`, `correctAnswer`, `lostGame` and `wonGame`, `nextQuestion` in `lostGame` and `wonGame`.

Go-only tests: `TestRickAPI` (URLs: `?page=N`, `/<id>`, `/a,b,c`, episode clue; `RandomNumber` = floor(random * 400)), `TestRickAPIFailure` (log + zero value on HTTP 500),
`TestMachineWithRickAPI` (the machine's real actors against the fake server: home page -> accept -> question ready).
