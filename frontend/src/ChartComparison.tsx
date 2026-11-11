import { useEffect, useState } from 'react';
import { comparisonValues, dependencyHint, previewValues, type Catalog, type Value } from './builder-values';
import { Preview } from './Preview';
import { renderChart } from './render-chart';

const noNotes = new Map<number, string>();

export function ChartComparison({ catalog, tool, kubernetesVersion, values }: {
  catalog: Catalog;
  tool: string;
  kubernetesVersion: string;
  values: Record<string, Value>;
}) {
  const [path, setPath] = useState(catalog.options[0].path);
  const option = catalog.options.find(entry => entry.path === path)!;
  const current = previewValues(catalog, values);
  const alternate = comparisonValues(catalog, path, values);
  const alternatePreview = alternate ? previewValues(catalog, alternate) : undefined;
  const alternateText = alternatePreview?.text;
  const alternateLine = alternatePreview?.fieldLines.get(path);
  const alternateField = alternateLine === undefined ? '' : alternateText?.split('\n')[alternateLine - 1].trim();
  const currentText = current.text;
  const requestKey = JSON.stringify([tool, kubernetesVersion, path, currentText, alternateText]);
  const [result, setResult] = useState<{ key: string; yaml?: string; baseline?: string; error?: string }>();

  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      try {
        const currentRender = renderChart(tool, kubernetesVersion, currentText, controller.signal);
        const alternateRender = alternateText
          ? renderChart(tool, kubernetesVersion, alternateText, controller.signal)
          : currentRender;
        const [yaml, baseline] = await Promise.all([currentRender, alternateRender]);
        if (!controller.signal.aborted) setResult({ key: requestKey, yaml, baseline });
      } catch (error) {
        if (!controller.signal.aborted) {
          setResult({ key: requestKey, error: error instanceof Error ? error.message : 'Could not render the chart.' });
        }
      }
    }, 100);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [requestKey, tool, kubernetesVersion, currentText, alternateText]);

  const visible = result?.key === requestKey ? result : undefined;
  const hint = dependencyHint(catalog, option, values);
  return (
    <div className="chart-comparison" aria-label="Helm to YAML comparison">
      <label>
        Helm option to inspect
        <select value={path} onChange={event => setPath(event.target.value)}>
          {catalog.options.map(entry => <option key={entry.path} value={entry.path}>{entry.label}</option>)}
        </select>
      </label>
      {hint ? <p>{hint}</p> : (
        <p>
          Comparing <code>{path}</code> with <code>{alternateField}</code>.
          Green marks current YAML. Red marks YAML from the comparison, including removed resources.
        </p>
      )}
      {!visible && <p role="status">Rendering comparison…</p>}
      {visible?.error && <p role="alert">{visible.error}</p>}
      {visible?.yaml !== undefined && visible.yaml === visible.baseline && !hint && (
        <p role="status">This option produced no YAML changes for the current configuration.</p>
      )}
      <div className="comparison-panes">
        <div>
          <h2>Helm values</h2>
          <Preview current={current} previous={current} focusLine={current.fieldLines.get(path)} />
        </div>
        <div>
          <h2>Rendered YAML</h2>
          {visible?.yaml !== undefined && (
            <Preview
              current={{ text: visible.yaml, notes: noNotes }}
              previous={{ text: visible.baseline ?? visible.yaml, notes: noNotes }}
              revealChanges
            />
          )}
        </div>
      </div>
    </div>
  );
}
