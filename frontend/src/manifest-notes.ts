export function manifestNotes(manifest: string) {
  const notes = new Map<number, string>();
  let section = '';
  let field = '';
  for (const [index, line] of manifest.split('\n').entries()) {
    const content = line.trimStart();
    if (line === '---' || !content) {
      section = line === '---' ? '' : section;
      field = '';
    } else if (content.startsWith('# Purpose: ')) {
      section = content.slice('# Purpose: '.length);
    } else if (content.startsWith('# Source: ')) {
      // Source markers identify the Helm template behind the resource.
      continue;
    } else if (content.startsWith('# ')) {
      field = content.slice(2);
    } else {
      if (line.startsWith('kind: ') && section) {
        notes.set(index + 1, section);
      } else if (field) {
        notes.set(index + 1, field);
      }
      field = '';
    }
  }
  return notes;
}
