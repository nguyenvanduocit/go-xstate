const timer=globalThis.setTimeout;
globalThis.setTimeout=((fn:any,_ms:number,...args:any[])=>timer(fn,1,...args)) as any;
await import('../../../references/xstate/examples/workflow-send-cloudevent/main.ts');
