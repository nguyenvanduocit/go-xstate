// Runs the example's own entry (main.ts) as a child process and answers its two prompts, so the
// recorded stdout is the real program's output. Each answer is written only after the matching
// question has been printed: readline drops lines that arrive while no question is pending.
const answers = ['Ada\n', '\n'];
const proc = Bun.spawn(['bun', 'run', '../../references/xstate/examples/workflow-async-subflow/main.ts'], {
  stdin: 'pipe',
  stdout: 'pipe',
  stderr: 'inherit'
});

let out = '';
let answered = 0;
const decoder = new TextDecoder();
for await (const chunk of proc.stdout) {
  const text = decoder.decode(chunk);
  out += text;
  process.stdout.write(text);
  const questions = (out.match(/\?|finish the onboarding process/g) ?? []).length;
  while (answered < answers.length && questions > answered) {
    proc.stdin.write(answers[answered++]);
    proc.stdin.flush();
  }
}
proc.stdin.end();
await proc.exited;
