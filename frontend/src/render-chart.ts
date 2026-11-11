export async function renderChart(tool: string, kubernetesVersion: string, values: string, signal: AbortSignal) {
  const response = await fetch('/api/render', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tool, kubernetesVersion, values }),
    signal,
  });
  const text = await response.text();
  if (!response.ok) {
    throw new Error(response.status === 422 ? text.trim() : 'The local chart renderer is unavailable.');
  }
  return text;
}
