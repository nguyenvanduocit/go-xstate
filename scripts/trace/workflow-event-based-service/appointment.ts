import { runTrace } from '../trace.ts';
import { workflow, appointmentStub, finish } from './lib/load.ts';

const logic = workflow.provide({ actors: { MakeAppointmentAction: appointmentStub } });

// Same input as main.ts (the machine does not read it).
const input = { person: { name: 'Jenny' } };
const t = await runTrace(
  'workflow-event-based-service',
  logic,
  [
    { send: { type: 'unknown' } }, // ignored in Idle
    {
      send: {
        type: 'MakeVetAppointment',
        patientInfo: { name: 'Jenny', pet: 'Ato', reason: 'Annual checkup' }
      }
    }, // Idle -> MakeVetAppointmentState, patientInfo assigned, actor invoked
    { wait: 30 }, // MakeAppointmentAction still pending
    {
      send: {
        type: 'MakeVetAppointment',
        patientInfo: { name: 'Max', pet: 'Rex', reason: 'Vaccination' }
      }
    }, // no handler in MakeVetAppointmentState: ignored, patientInfo unchanged
    { send: { type: 'unknown' } }, // ignored while invoking
    { wait: 150 }, // promise resolved: onDone -> Idle, appointmentInfo = {appointmentInfo: {...}}
    {
      send: {
        type: 'MakeVetAppointment',
        patientInfo: { name: 'Max', pet: 'Rex', reason: 'Vaccination' }
      }
    }, // second round: patientInfo replaced, appointmentInfo kept until done
    { wait: 150 } // second onDone: appointmentInfo replaced
  ],
  { input }
);
finish({ ...t, clock: false, input });
