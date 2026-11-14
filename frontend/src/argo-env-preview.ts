const explanation = 'These environment variables configure the container. Settings read from the argocd-cmd-params-cm ConfigMap are all optional.';

export function argoEnvironmentPreview(text: string) {
  const lines = text.split('\n');
  const preview: string[] = [];
  const folds: { start: number; end: number }[] = [];

  for (let index = 0; index < lines.length; index++) {
    const line = lines[index];
    const env = /^( +)(?:- )?env:\s*$/.exec(line);
    if (!env) {
      preview.push(line);
      continue;
    }
    // Helm may indent a sequence at the same column as its key.
    const indent = env[1].length + (line.trimStart().startsWith('- ') ? 2 : 0);
    let end = index + 1;
    let lastEntry = index;
    let entries = 0;
    for (; end < lines.length; end++) {
      const child = lines[end].trimStart();
      if (!child || child.startsWith('#')) continue;
      const depth = lines[end].length - child.length;
      if (depth < indent || (depth === indent && !child.startsWith('- '))) break;
      if (/^\s*- name:/.test(lines[end])) entries++;
      lastEntry = end;
    }
    const block = lines.slice(index + 1, lastEntry + 1);
    if (entries < 4 || !block.some(child => /^\s+name: argocd-cmd-params-cm\s*$/.test(child))) {
      preview.push(line);
      continue;
    }
    preview.push(env[1] + '# ' + explanation, line);
    const start = preview.length;
    preview.push(...block);
    folds.push({ start, end: preview.length });
    index = lastEntry;
  }

  return { text: preview.join('\n'), folds };
}
