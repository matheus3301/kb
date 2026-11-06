type Step = {
  label: string;
  heading: string;
  text?: string;
  command?: string;
  path?: string;
  diff?: [string, string];
  duration: number;
};

type Session = {
  title: string;
  prompt: string;
  steps: Step[];
};

export const agentSessions: Session[] = [
  {
    title: 'Recover a decision',
    prompt: 'Why did we cap export batches at 500 rows?',
    steps: [
      { label: 'Plan', heading: 'Find the original decision', text: 'I’ll look up the batch limit and check what led to it.', duration: 2200 },
      { label: 'Search', heading: 'One matching note', command: 'kb search "export batches" --limit 3', path: '01_Projects/exports/batching.md', duration: 3000 },
      {
        label: 'Read', heading: 'Read the rationale',
        command: 'kb read 01_Projects/exports/batching.md',
        text: '“Split large exports into 500-row batches so retries stay within the worker’s memory budget.”',
        path: '01_Projects/exports/batching.md:8', duration: 4000,
      },
      {
        label: 'Reply', heading: 'Agent reply',
        text: 'The 500-row cap keeps retries within the worker’s memory budget. Larger exports are split into batches so a retry only needs to process one batch.',
        path: '01_Projects/exports/batching.md:8', duration: 9000,
      },
    ],
  },
  {
    title: 'Pick up an incident',
    prompt: 'The nightly import stalled again. What should I check first?',
    steps: [
      { label: 'Plan', heading: 'Check the last incident', text: 'I’ll look for the previous handoff and the conditions it describes.', duration: 2200 },
      {
        label: 'Context', heading: 'Found a handoff excerpt',
        command: 'kb context "nightly import stalled" --max-tokens 600',
        text: '“If the queue grows while processed jobs stay flat, check the worker logs for an expired API token.”',
        path: '01_Projects/imports/handoff.md:8', duration: 4000,
      },
      {
        label: 'Read', heading: 'Check the recovery instructions',
        command: 'kb read 01_Projects/imports/handoff.md',
        text: '“If the logs show HTTP 401, renew the token and retry the stalled job.”',
        path: '01_Projects/imports/handoff.md:9', duration: 4200,
      },
      {
        label: 'Reply', heading: 'Agent reply',
        text: 'Check whether the queue is growing while the processed count stays flat. If it is, look for HTTP 401 in the worker logs. Renew the token and retry the job if you see that error.',
        path: '01_Projects/imports/handoff.md:8–9', duration: 9000,
      },
    ],
  },
  {
    title: 'Record a decision',
    prompt: 'Add this to the Knowledge Base: keep the previous webhook signing key active for 24 hours during rotation.',
    steps: [
      { label: 'Plan', heading: 'Find the existing runbook', text: 'I’ll look for the rotation note and preview an update to it.', duration: 2400 },
      { label: 'Search', heading: 'Found the rotation runbook', command: 'kb search "webhook key rotation" --limit 3', path: '03_Resources/webhook-rotation.md', duration: 3000 },
      {
        label: 'Read', heading: 'Read the note and its current revision',
        command: 'kb read 03_Resources/webhook-rotation.md',
        text: '“Retire the previous signing key immediately.”',
        path: '03_Resources/webhook-rotation.md:8', duration: 3600,
      },
      {
        label: 'Preview', heading: 'Preview the revision-checked edit',
        command: 'kb patch --input patch.json --dry-run',
        diff: ['Retire the previous signing key immediately.', 'Keep the previous signing key active for 24 hours.'],
        duration: 5200,
      },
      {
        label: 'Save', heading: 'Update applied',
        command: 'kb patch --input patch.json',
        path: '03_Resources/webhook-rotation.md', duration: 3100,
      },
      {
        label: 'Reply', heading: 'Agent reply',
        text: 'I updated the rotation runbook. It now says to keep the previous webhook signing key active for 24 hours.',
        path: '03_Resources/webhook-rotation.md:11', duration: 9000,
      },
    ],
  },
];
