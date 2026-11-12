import { useEffect, useState } from 'react';
import { comparisonValues, dependencyHint, previewValues, type Catalog, type Value } from './builder-values';
import { Preview, type LineRange } from './Preview';
import { manifestNotes } from './manifest-notes';

type Change = {
  resource: { apiVersion: string; kind: string; namespace?: string; name: string };
  type: 'added' | 'changed' | 'removed';
  current?: LineRange;
  prior?: LineRange;
};
type Comparison = {
  current: string; prior: string; changes: Change[];
  currentNotes: Record<string, string>; priorNotes: Record<string, string>;
};

function rangeLabel(range: LineRange) {
  return range.start === range.end ? `line ${range.start}` : `lines ${range.start}–${range.end}`;
}

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
  const [result, setResult] = useState<{ key: string; comparison?: Comparison; error?: string }>();
  const [navigation, setNavigation] = useState<{ key: string; side: 'current' | 'prior'; range: LineRange }>();

  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      try {
        const response = await fetch('/api/compare', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ tool, kubernetesVersion, values: currentText, alternateValues: alternateText }),
          signal: controller.signal,
        });
        if (!response.ok) {
          throw new Error(response.status === 422 ? (await response.text()).trim() : 'The local chart renderer is unavailable.');
        }
        const comparison: Comparison = await response.json();
        if (!controller.signal.aborted) setResult({ key: requestKey, comparison });
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
  const comparison = visible?.comparison;
  const selected = navigation?.key === requestKey ? navigation : undefined;
  const side = selected?.side ?? 'current';
  const ranges = comparison?.changes.flatMap(change => change[side] ? [change[side]] : []);
  const text = comparison?.[side] ?? '';
  const rendered = { text, notes: manifestNotes(text, side === 'current' ? comparison?.currentNotes : comparison?.priorNotes) };
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
          Select a line range to jump to its YAML.
        </p>
      )}
      {!visible && <p role="status">Rendering comparison…</p>}
      {visible?.error && <p role="alert">{visible.error}</p>}
      {comparison?.changes.length === 0 && !hint && (
        <p role="status">This option produced no YAML changes for the current configuration.</p>
      )}
      <div className="comparison-panes">
        <div>
          <h2>Helm values</h2>
          <Preview current={current} previous={current} focusLine={current.fieldLines.get(path)} />
        </div>
        <div>
          <h2>{side === 'current' ? 'Rendered YAML' : 'YAML with comparison value'}</h2>
          {comparison && (
            <Preview current={rendered} previous={rendered} highlightRanges={ranges} focusRange={selected?.range} />
          )}
        </div>
      </div>
      {!!comparison?.changes.length && (
        <ul className="comparison-ranges" aria-label="Affected YAML ranges">
          {comparison.changes.map(({ resource, type, current, prior }, index) => (
            <li key={index}>
              <strong>{resource.kind} {resource.namespace ? `${resource.namespace}/` : ''}{resource.name}</strong>
              {' '}({type})
              {current && (
                <button type="button" onClick={() => setNavigation({ key: requestKey, side: 'current', range: { ...current } })}>
                  Current {rangeLabel(current)}
                </button>
              )}
              {prior && (
                <button type="button" onClick={() => setNavigation({ key: requestKey, side: 'prior', range: { ...prior } })}>
                  Comparison {rangeLabel(prior)}
                </button>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
