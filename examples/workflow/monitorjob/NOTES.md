# Monitor job

Ports [`workflow-monitor-job/main.ts`](../../../references/xstate/examples/workflow-monitor-job/main.ts):
submit a job, wait five seconds, poll its status, then report success or failure.
An unfinished status returns to the wait state. `Run` prints the upstream demo's
output; `RunWith` accepts a clock for tests.

`succeeded.golden.json` covers an unfinished poll followed by success.
`failed.golden.json` covers the failure branch. Both use the JavaScript recorder
in `scripts/trace/workflow-monitor-job/` and matching Go actor stubs that resolve
after 100 ms. The wait state uses a simulated clock. `TestStdout` compares the
demo with `workflow-monitor-job.stdout.txt` while advancing its five-second wait
through the simulated clock.

The upstream services are logging stubs; this port does not submit or monitor
real jobs. Actor rejection paths are not covered by these fixtures, and the
upstream machine defines no error transitions for them. Console formatting is
matched for the supplied demo inputs, not arbitrary JavaScript values.

Run from `examples/`:

```sh
go test -race -count=1 ./workflow/monitorjob
```
