import { manifestNotes } from './manifest-notes';

export function manifestComponents(text: string, tool: string) {
  const documents = new Map<string, string[]>();
  const main = tool === 'prometheus' ? 'Prometheus server' : 'Argo CD';
  const [header, ...parts] = text.split(/^---[ \t]*$/m);
  let component = main;
  for (const part of parts) {
    component = part.match(/^# Component: (.+)$/m)?.[1] ?? component;
    const label = /^kind: CustomResourceDefinition[ \t]*$/m.test(part) ? 'CRD schemas' : component;
    const group = documents.get(label) ?? [];
    group.push(`---\n${part.trim()}\n`);
    documents.set(label, group);
  }
  return Array.from(documents, ([name, parts]) => {
    const body = parts.join('');
    const text = name === main ? header + body : body;
    return { name, text, lines: body.trimEnd().split('\n').length, notes: manifestNotes(text) };
  }).sort((a, b) => Number(b.name === main) - Number(a.name === main));
}
