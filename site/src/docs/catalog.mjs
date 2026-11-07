export const repository = 'https://github.com/matheus3301/kb';
export const documents = [
  { id: 'README', title: 'Documentation overview', group: 'Get started' },
  { id: 'installation', title: 'Installation', group: 'Get started' },
  { id: 'quickstart', title: 'Quickstart', group: 'Get started' },
  { id: 'agent-setup', title: 'Agent setup · CLI + skill', group: 'Get started' },
  { id: 'cli', title: 'CLI reference', group: 'Use KB' },
  { id: 'backups', title: 'Backups & recovery', group: 'Use KB' },
  { id: 'setup', title: 'Setup by harness', group: 'Use KB' },
  { id: 'mcp', title: 'MCP · independent interface', group: 'Integrate' },
  { id: 'interfaces', title: 'Interface contract', group: 'Integrate' },
  { id: 'spec', title: 'Specification', group: 'Understand' },
  { id: 'project-overview', title: 'Architecture', group: 'Understand' },
  { id: 'compatibility', title: 'Versions & support', group: 'Understand' },
  { id: 'quality', title: 'Quality gates', group: 'Understand' },
];
export function docsPath(id, base = process.env.KB_SITE_BASE ?? '/kb/') {
  return `${base.replace(/\/$/, '')}/docs/${id === 'README' ? '' : `${id}/`}`;
}
