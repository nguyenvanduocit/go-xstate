// The machine's three promise actors call the Rick and Morty API through axios; they are
// replaced with deterministic stubs (the Go test installs the same stubs through Provide).
import { fromPromise } from 'xstate';
import triviaMachine from '../../../references/xstate/examples/trivia-game-example/src/triviaMachine.ts';
import { runTrace, type Step } from '../trace.ts';

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
const character = (id: number) => ({
  id,
  name: `Character ${id}`,
  status: 'Alive',
  species: 'Human',
  gender: 'Male',
  image: `https://example.test/avatar/${id}.jpeg`,
  episode: [`https://example.test/episode/${id}`]
});

// loadSingleCharacter resolves with ids 101, 102, ... in call order.
let singles = 0;
const logic = triviaMachine.provide({
  actors: {
    // The 10 ms delays keep the snapshot taken right after a send in the loading state on both sides
    // (a Go goroutine would otherwise resolve before the test reads it; JS cannot, being synchronous).
    loadHomePageCharacters: fromPromise(async () => {
      await sleep(10);
      return [character(1), character(2)];
    }),
    loadSingleCharacter: fromPromise(async () => {
      await sleep(10);
      return character(100 + ++singles);
    }),
    loadRandomCharacters: fromPromise(async () => [character(201), character(202), character(203)])
  }
});

const send = (type: string, extra: Record<string, unknown> = {}): Step => ({ send: { type, ...extra } });
const load: Step = { wait: 100 };

const steps: Step[] = [
  // homepage.loadingData (initial) -> dataLoaded
  send('user.play'), // ignored while loading
  load,
  send('user.toggleClue'), // ignored on the homepage
  send('user.selectAnswer', { answer: 1 }), // ignored on the homepage
  send('user.play'), // dataLoaded -> instructionModal
  send('user.close'), // -> homepage.dataLoaded (data is not reloaded)
  send('user.play'),
  send('user.reject'), // -> homepage.dataLoaded
  send('user.play'),
  send('user.accept'), // -> startTrivia.loadQuestionData.loadCharacter
  load, // question 1 ready, character 101
  send('user.nextQuestion'), // ignored in questionStart
  send('user.playAgain'), // ignored in questionStart
  send('user.toggleClue'), // clue opens
  send('user.toggleClue'), // clue closes
  send('user.toggleClue'), // clue opens again (stays open across questions)
  send('user.selectAnswer', { answer: 999 }), // wrong -> incorrectAnswer, lifes 2
  send('user.selectAnswer', { answer: 101 }), // ignored in incorrectAnswer
  send('user.toggleClue'), // handled by questionReady in a child state
  send('user.nextQuestion'), // -> loadQuestionData again
  load, // question 2, character 102
  send('user.selectAnswer', { answer: 102 }), // correct -> points 10
  send('user.selectAnswer', { answer: 999 }), // ignored in correctAnswer
  send('user.nextQuestion'),
  load, // question 3, character 103
  send('user.selectAnswer', { answer: 999 }), // wrong, lifes 1
  send('user.nextQuestion'),
  load, // question 4, character 104
  send('user.selectAnswer', { answer: 999 }), // wrong, lifes 0 -> always: hasLostGame -> lostGame
  send('user.nextQuestion'), // ignored in lostGame
  send('user.selectAnswer', { answer: 104 }), // ignored in lostGame
  send('user.toggleClue'), // still handled in lostGame
  send('user.playAgain'), // -> startTrivia re-entered, context reset
  load // question 1, character 105
];

// Win: one wrong answer (character 105), then ten correct ones (characters 106..115, 100 points)
// -> always: hasWonGame -> wonGame.
steps.push(send('user.selectAnswer', { answer: 999 }), send('user.nextQuestion'), load);
for (let i = 0; i < 10; i++) {
  steps.push(send('user.selectAnswer', { answer: 106 + i }));
  if (i < 9) steps.push(send('user.nextQuestion'), load);
}
steps.push(
  send('user.nextQuestion'), // ignored in wonGame
  send('user.selectAnswer', { answer: 1 }), // ignored in wonGame
  send('user.playAgain'), // -> startTrivia, context reset
  load,
  send('user.toggleClue')
);

const t = await runTrace('trivia-game-example', logic, steps);
console.log(JSON.stringify(t, null, 2));
