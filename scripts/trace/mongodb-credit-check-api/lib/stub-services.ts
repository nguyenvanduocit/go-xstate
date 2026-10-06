// Scripted stand-in for references/xstate/examples/mongodb-credit-check-api/services/machineLogicService.ts
// (substituted by lib/load.ts). Same exports, same signatures; the random bureau
// calls and the Mongo lookups become scripted, SSN-keyed behaviour with short fixed delays.
//
// Timeline (ms after the service is called):
//   verifyCredentials 20 · checkReportsTable 40 · checkBureau 100/200/300 (EquiGavin/GavUnion/Gavperian)
//   determineMiddleScore 40 · generateInterestRate 60
//
// SSN scenarios:
//   777777777  checkReportsTable rejects for every bureau
//   999999999  checkBureau rejects for EquiGavin and GavUnion
//   999999998  checkBureau rejects for Gavperian
//   888888888  stored reports: EquiGavin 690, GavUnion 710, Gavperian 640
//   666666666  every bureau scores 666 (generateInterestRate rejects for 666)
//   555555555  every bureau scores 555 (determineMiddleScore rejects when it sees 555)
//   other      EquiGavin 720, GavUnion 580, Gavperian 650
// determineMiddleScore is the verbatim body of the real one (the real module needs zod and mongodb at import time).

export type userCredential = { firstName: string; lastName: string; SSN: string };

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

const BUREAU_DELAY: Record<string, number> = { EquiGavin: 100, GavUnion: 200, Gavperian: 300 };
const BUREAU_SCORE: Record<string, number> = { EquiGavin: 720, GavUnion: 580, Gavperian: 650 };
const STORED: Record<string, number> = {
  '888888888|EquiGavin': 690,
  '888888888|GavUnion': 710,
  '888888888|Gavperian': 640
};
const BUREAU_FAIL: Record<string, string[]> = {
  '999999999': ['EquiGavin', 'GavUnion'],
  '999999998': ['Gavperian']
};

function validString(v: unknown) {
  return typeof v === 'string' && v.length >= 1 && v.length <= 255;
}

export function verifyCredentialsResult(credentials: userCredential) {
  const c: any = credentials;
  const bad = !validString(c.firstName)
    ? 'firstName'
    : !validString(c.lastName)
      ? 'lastName'
      : typeof c.SSN !== 'string' || c.SSN.length !== 9
        ? 'SSN'
        : '';
  if (bad) throw new Error('Invalid Credentials. Details: invalid ' + bad);
  return credentials;
}

export async function verifyCredentials(credentials: userCredential) {
  await sleep(20);
  return verifyCredentialsResult(credentials);
}

export function determineMiddleScoreResult(scores: number[]) {
  if (scores.includes(555)) throw new Error('score service unavailable');
  scores.sort();
  return scores[1];
}

export async function determineMiddleScore(scores: number[]) {
  await sleep(40);
  return determineMiddleScoreResult(scores);
}

export function checkReportsTableResult({ ssn, bureauName }: { ssn: string; bureauName: string }) {
  if (ssn === '777777777') throw new Error('reports table unavailable');
  const creditScore = STORED[`${ssn}|${bureauName}`];
  return creditScore === undefined ? null : { ssn, bureauName, creditScore };
}

export async function checkReportsTable({ ssn, bureauName }: { ssn: string; bureauName: string }) {
  await sleep(40);
  return checkReportsTableResult({ ssn, bureauName });
}

export function checkBureauServiceResult({ ssn, bureauName }: { ssn: string; bureauName: string }) {
  if (BUREAU_FAIL[ssn]?.includes(bureauName)) throw new Error(bureauName + ' unavailable');
  if (ssn === '666666666') return 666;
  if (ssn === '555555555') return 555;
  return BUREAU_SCORE[bureauName];
}

export async function checkBureauService({ ssn, bureauName }: { ssn: string; bureauName: string }) {
  await sleep(BUREAU_DELAY[bureauName]!);
  return checkBureauServiceResult({ ssn, bureauName });
}

export function generateInterestRateResult(creditScore: number) {
  if (creditScore === 666) throw new Error('rate service unavailable');
  if (creditScore > 700) return 3.5;
  if (creditScore > 600) return 5;
  return 200;
}

export async function generateInterestRate(creditScore: number) {
  await sleep(60);
  return generateInterestRateResult(creditScore);
}

export async function saveCreditReport(_report: unknown) {}
export async function saveCreditProfile(_profile: unknown) {}
