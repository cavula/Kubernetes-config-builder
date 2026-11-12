export async function renderExplainedChart(tool: string, kubernetesVersion: string, values: string, signal: AbortSignal) {
  const response = await fetch('/api/export', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ tool, kubernetesVersion, values }),
    signal,
  });
  if (!response.ok) {
    throw new Error(response.status === 422 ? (await response.text()).trim() : 'The local chart renderer is unavailable.');
  }
  const { yaml } = await response.json();
  if (typeof yaml !== 'string' || !yaml.startsWith('# kubectl apply ') || !yaml.includes('\n# Why: ')) {
    throw new Error('The local renderer did not return an explained YAML export.');
  }
  return yaml;
}

export async function explainManifestField(manifest: string, kubernetesVersion: string, line: number, signal: AbortSignal) {
  const response = await fetch('/api/hover', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ manifest, kubernetesVersion, line }),
    signal,
  });
  return response.ok ? await response.json() as string : '';
}
