export async function renderChart(
  tool: string,
  kubernetesVersion: string,
  values: string,
  signal: AbortSignal,
  onManifest: (text: string) => void,
) {
  const response = await fetch('/api/render', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/x-ndjson' },
    body: JSON.stringify({ tool, kubernetesVersion, values }),
    signal,
  });
  if (!response.ok || !response.body) {
    throw new Error(response.status === 422 ? (await response.text()).trim() : 'The local chart renderer is unavailable.');
  }

  const reader = response.body.pipeThrough(new TextDecoderStream()).getReader();
  let buffer = '';
  async function readLine() {
    let newline = buffer.indexOf('\n');
    while (newline === -1) {
      const { done, value } = await reader.read();
      if (done) throw new Error('The chart renderer returned an incomplete preview.');
      buffer += value;
      newline = buffer.indexOf('\n');
    }
    const line = buffer.slice(0, newline);
    buffer = buffer.slice(newline + 1);
    return line;
  }

  try {
    const text: string = JSON.parse(await readLine());
    onManifest(text);
    const notes: Record<string, string> = JSON.parse(await readLine());
    return { text, notes };
  } finally {
    reader.releaseLock();
  }
}
