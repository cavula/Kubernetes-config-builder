import { prometheusAddons } from './builder-values';

export function manifestComponents(text: string, tool: string) {
  const components = new Map<string, number>();
  const workloads: string[] = [];
  const schemas: string[] = [];
  for (const part of text.split(/^---[ \t]*$/m)) {
    if (!/^kind: /m.test(part)) {
      if (part.trim()) workloads.push(part.trim());
      continue;
    }
    const schema = /^kind: CustomResourceDefinition\s*$/m.test(part);
    const dependency = part.match(/^# Source: prometheus\/charts\/([^/]+)\//m)?.[1];
    const addon = prometheusAddons.find(entry => entry.path === `${dependency}.enabled`);
    const label = schema ? 'CRD schemas' : addon?.label ?? (tool === 'prometheus' ? 'Prometheus server' : 'Argo CD');
    components.set(label, (components.get(label) ?? 0) + part.trim().split('\n').length + 1);
    (schema ? schemas : workloads).push(part.trim());
  }
  // Header comments, if any, must stay ahead of the first document.
  const join = (parts: string[]) => parts.map(part => /^kind: /m.test(part) ? `---\n${part}\n` : `${part}\n`).join('');
  return { components, workloads: join(workloads), schemas: join(schemas) };
}
