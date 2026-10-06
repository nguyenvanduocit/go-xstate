// The console.log lines of the example's actions and guards for the happy path
// (saveReport x3, allSucceeded, saveCreditProfile, emailUser, emailSalesTeam).
import { createActor } from 'xstate';
import { creditCheckMachine, logs } from './lib/load.ts';

const actor = createActor(creditCheckMachine).start();
actor.send({ type: 'Submit', SSN: '123456789', firstName: 'Gavin', lastName: 'Bauman' });
await new Promise((r) => setTimeout(r, 700));
actor.stop();
process.stdout.write(logs.join('\n') + '\n');
