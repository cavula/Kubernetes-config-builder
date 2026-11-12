import exportGuides from '../../backend/export-guides.json';
import acceptedRisks from '../../security/accepted-risks.json';
import { useEffect, useState } from 'react';
import { catalogForChart, defaultValues, dependencyHint, previewValues, type catalogs, type Value } from './builder-values';
import { Preview } from './Preview';
import { ChartComparison } from './ChartComparison';
import { ExportActions } from './ExportActions';
import { ValidationStatus } from './ValidationStatus';
import { manifestNotes } from './manifest-notes';
import { explainManifestField, renderChart } from './render-chart';

export function Builder({ tool, chartVersion, kubernetesVersion }: {
  tool: keyof typeof catalogs;
  chartVersion: string;
  kubernetesVersion: string;
}) {
  const catalog = catalogForChart(tool, chartVersion);
  const hasAcceptedRisks = acceptedRisks.charts[tool].pinnedChart === chartVersion &&
    Object.values(acceptedRisks.images).some(entry => entry.tool === tool);
  const [{ current, previous }, setValues] = useState(() => {
    const defaults = defaultValues(catalog);
    return { current: defaults, previous: defaults };
  });

  const [format, setFormat] = useState<'values' | 'manifests'>('values');
  const [compare, setCompare] = useState(false);
  const [manifest, setManifest] = useState({ key: '', current: '', previous: '', notes: new Map<number, string>(), previousNotes: new Map<number, string>(), error: '' });
  const currentPreview = previewValues(catalog, current);
  const renderKey = JSON.stringify([tool, kubernetesVersion, currentPreview.text]);

  useEffect(() => {
    if (format !== 'manifests') return;
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      try {
        const text = await renderChart(tool, kubernetesVersion, currentPreview.text, controller.signal);
        if (!controller.signal.aborted) {
          const currentNotes = manifestNotes(text);
          setManifest(({ current, notes }) => ({
            key: renderKey, current: text, previous: current || text,
            notes: currentNotes, previousNotes: current ? notes : currentNotes, error: '',
          }));
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setManifest(previous => ({
            ...previous,
            key: renderKey,
            current: previous.key === renderKey ? previous.current : '',
            error: error instanceof Error ? error.message : 'Could not render the chart.',
          }));
        }
      }
    }, 100);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [format, tool, kubernetesVersion, currentPreview.text, renderKey]);

  function update(path: string, value: Value) {
    setValues(({ current }) => ({ current: { ...current, [path]: value }, previous: current }));
  }

  const currentManifest = manifest.key === renderKey ? manifest.current : '';
  const valuesExport = exportGuides[tool].map(line => '# ' + line).join('\n') + '\n\n' + currentPreview.text;
  async function loadExport(part: 'manifests' | 'crds') {
    const response = await fetch('/api/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tool, kubernetesVersion, values: currentPreview.text }),
    });
    if (!response.ok) throw new Error((await response.text()).trim() || 'Could not prepare the export.');
    return (await response.json())[part] as string;
  }

  return (
    <div className={compare ? "builder builder-comparing" : "builder"}>
      <section aria-label="Builder options" className="builder-options">
        {catalog.options.map(option => {
          const value = current[option.path];
          const hint = dependencyHint(catalog, option, current);
          const note = catalog.notes.fields[option.path];
          if (typeof value === 'object') {
            return (
              <fieldset key={option.path}>
                <legend>{option.label}</legend>
                {(['requests', 'limits'] as const).flatMap(group =>
                  (['cpu', 'memory'] as const).map(resource => (
                    <label key={`${group}.${resource}`}>
                      {group === 'requests' ? 'Requested' : 'Maximum'} {resource === 'cpu' ? 'CPU' : 'memory'}
                      <input value={value[group]?.[resource] ?? ''} onChange={event =>
                        update(option.path, { ...value, [group]: { ...value[group], [resource]: event.target.value } })
                      } />
                    </label>
                  )),
                )}
              </fieldset>
            );
          }
          return (
            <label key={option.path}>
              {option.label}
              {typeof value === 'boolean' ? (
                <input type="checkbox" checked={value} onChange={event => update(option.path, event.target.checked)} />
              ) : option.choices ? (
                <select value={value} onChange={event => update(option.path, event.target.value)}>
                  {option.choices.map(choice => <option key={choice}>{choice}</option>)}
                </select>
              ) : (
                <input
                  value={value}
                  type={option.type === 'number' ? 'number' : 'text'}
                  min={option.type === 'number' ? 1 : undefined}
                  disabled={!!option.requires && !current[option.requires]}
                  onChange={event => {
                    if (typeof value === 'string') update(option.path, event.target.value);
                    else if (event.target.validity.valid && Number.isInteger(event.target.valueAsNumber)) {
                      update(option.path, event.target.valueAsNumber);
                    }
                  }}
                />
              )}
              <small>{hint ?? note}</small>
            </label>
          );
        })}
      </section>
      <div className="builder-output">
        {hasAcceptedRisks && (
          <p className="security-note">
            <strong>Chart image security note.</strong> The pinned {tool === 'argo-cd' ? 'Argo CD' : 'Prometheus'} chart
            includes images with known vulnerabilities that remain in the latest official chart reviewed.
            Review date <time dateTime={acceptedRisks.reviewDate}>{acceptedRisks.reviewDate}</time>.
          </p>
        )}
        <ValidationStatus tool={tool} kubernetesVersion={kubernetesVersion} values={currentPreview.text} />
        <label>
          Preview format
          <select value={format} onChange={event => setFormat(event.target.value === 'manifests' ? 'manifests' : 'values')}>
            <option value="values">Helm values</option>
            <option value="manifests">See what this creates</option>
          </select>
        </label>
        <ExportActions key={tool} text={valuesExport} filename={`${tool}-values.yaml`} label="Download Helm values" copyLabel="Copy Helm values" />
        {format === 'manifests' && (
          <>
            <ExportActions text={currentManifest} filename={`${tool}-manifests.yaml`}
              label="Download explained manifests" copyLabel="Copy explained manifests" load={() => loadExport('manifests')} />
            {tool === 'argo-cd' && (
              <ExportActions text={currentManifest} filename="argo-cd-crds.yaml"
                label="Download CRD schemas" copyLabel="Copy CRD schemas" load={() => loadExport('crds')} />
            )}
          </>
        )}
        <button type="button" aria-pressed={compare} onClick={() => setCompare(open => !open)}>
          {compare ? 'Close comparison' : 'Compare Helm values and YAML'}
        </button>
        {compare ? (
          <ChartComparison catalog={catalog} tool={tool} kubernetesVersion={kubernetesVersion} values={current} />
        ) : format === 'values' ? (
          <Preview current={currentPreview} previous={previewValues(catalog, previous)} />
        ) : (
          <>
            {manifest.key === renderKey && manifest.error && <p role="alert">{manifest.error}</p>}
            <Preview explainField={(text, line, signal) => explainManifestField(text, kubernetesVersion, line, signal)}
              current={{ text: manifest.current, notes: manifest.notes }} previous={{ text: manifest.previous, notes: manifest.previousNotes }} />
          </>
        )}
      </div>
    </div>
  );
}
