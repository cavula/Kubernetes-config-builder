import { useEffect, useState } from 'react';

type Diagnostic = { check: string; severity: string; message: string };
type Result = { status: string; diagnostics: Diagnostic[] };

export function ValidationStatus({ tool, kubernetesVersion, values }: {
  tool: string;
  kubernetesVersion: string;
  values: string;
}) {
  const body = JSON.stringify({ tool, kubernetesVersion, values });
  const [response, setResponse] = useState<{ body: string; result: Result }>();

  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      try {
        const request = await fetch('/api/validate', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body,
          signal: controller.signal,
        });
        if (!request.ok) throw new Error('The local validator is unavailable.');
        const result: Result = await request.json();
        if (!controller.signal.aborted) setResponse({ body, result });
      } catch {
        if (!controller.signal.aborted) {
          setResponse({ body, result: {
            status: 'incomplete',
            diagnostics: [{ check: 'Validation', severity: 'incomplete', message: 'Start the local Go backend to run validation.' }],
          } });
        }
      }
    }, 250);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [body]);

  const result = response?.body === body ? response.result : undefined;
  return (
    <div className="validation-status" aria-live="polite">
      <h2>Validation</h2>
      {!result && <p>Checking configuration…</p>}
      {result?.status === 'passed' && <p>Helm lint and schema validation passed.</p>}
      {result?.diagnostics.length ? (
        <ul>
          {result.diagnostics.map(({ check, severity, message }, index) => (
            <li key={index}><strong>{check} ({severity})</strong> {message}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
