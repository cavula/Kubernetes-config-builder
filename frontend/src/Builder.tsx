import deploymentCommands from '../../backend/deployment-commands.json';
import exportGuides from '../../backend/export-guides.json';
import acceptedRisks from '../../security/accepted-risks.json';
import { useEffect, useMemo, useState } from 'react';
import { catalogForChart, presetValues, prometheusAddons, dependencyHint, describeValuesPaths, previewValues, type catalogs, type Value, type Preset } from './builder-values';
import { Preview } from './Preview';
import { ChartComparison } from './ChartComparison';
import { ExportActions } from './ExportActions';
import { ValidationStatus } from './ValidationStatus';
import { manifestNotes } from './manifest-notes';
import { manifestComponents } from './manifest-components';
import { explainManifestField, renderChart } from './render-chart';

export function Builder({ tool, chartVersion, kubernetesVersion, preset }: {
  tool: keyof typeof catalogs;
  chartVersion: string;
  preset: Preset;
  kubernetesVersion: string;
}) {
  const catalog = catalogForChart(tool, chartVersion);
  const hasAcceptedRisks = acceptedRisks.charts[tool].pinnedChart === chartVersion &&
    Object.values(acceptedRisks.images).some(entry => entry.tool === tool);
  const [{ current, previous }, setValues] = useState(() => {
    const defaults = presetValues(catalog, preset);
    return { current: defaults, previous: defaults };
  });

  const [format, setFormat] = useState<'values' | 'manifests'>('values');
  const [compare, setCompare] = useState(false);
  const [manifest, setManifest] = useState({ key: '', current: '', previous: '', notes: new Map<number, string>(), previousNotes: new Map<number, string>(), error: '' });
  const currentPreview = previewValues(catalog, current);
  const renderKey = JSON.stringify([tool, kubernetesVersion, currentPreview.text]);

  useEffect(() => {
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      try {
        const text = await renderChart(tool, kubernetesVersion, currentPreview.text, controller.signal);
        if (!controller.signal.aborted) {
          const currentNotes = manifestNotes(manifestComponents(text, tool).workloads);
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
  }, [tool, kubernetesVersion, currentPreview.text, renderKey]);

  function update(path: string, value: Value) {
    setValues(({ current }) => ({ current: { ...current, [path]: value }, previous: current }));
  }

  const currentManifest = manifest.key === renderKey ? manifest.current : '';
  const valuesExport = [
    ...deploymentCommands[tool].helm,
    'Indentation groups settings under their parent keys. A dotted name shows the full path to a setting.',
    ...exportGuides[tool],
  ].map(line => '# ' + line).join('\n') + '\n\n' + describeValuesPaths(currentPreview);
  async function loadExport() {
    const response = await fetch('/api/export', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ tool, kubernetesVersion, values: currentPreview.text }),
    });
    if (!response.ok) throw new Error((await response.text()).trim() || 'Could not prepare the export.');
    return (await response.json()).yaml as string;
  }

  const rendered = useMemo(() => manifestComponents(manifest.current, tool), [manifest.current, tool]);
  const prior = useMemo(() => manifestComponents(manifest.previous, tool), [manifest.previous, tool]);
  const commands = deploymentCommands[tool][format === 'values' ? 'helm' : 'yaml'];
  return (
    <div className={compare ? "builder builder-comparing" : "builder"}>
      <section aria-labelledby="customize" className="step builder-options">
        <h2 id="customize">2. Customize (optional)</h2>
        {catalog.options.map(option => {
          const value = current[option.path];
          const hint = dependencyHint(catalog, option, current);
          const note = catalog.notes.fields[option.path];
          const addon = tool === 'prometheus' && prometheusAddons.some(entry => entry.path === option.path);
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
                      <small>{catalog.notes.fields[`${option.path}.${group}.${resource}`]} Example {resource === 'cpu' ? '250m' : '512Mi'}.</small>
                    </label>
                  )),
                )}
              </fieldset>
            );
          }
          const control = (
            <label key={option.path}>
              {addon ? `Enable ${option.label}` : option.label}
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
              <small>{hint ?? note} Example {String(typeof value === 'boolean' ? 'On' : option.comparison ?? option.default)}.</small>
            </label>
          );
          return addon ? <fieldset key={option.path}><legend>{option.label}</legend>{control}</fieldset> : control;
        })}
      </section>
      <section className="step builder-output" aria-labelledby="deploy">
        <h2 id="deploy">3. Deploy</h2>
        <p>{tool === 'prometheus' ? 'Prometheus' : 'Argo CD'} chart <strong>{chartVersion}</strong></p>
        {hasAcceptedRisks && (
          <p className="security-note">
            <strong>Chart image security note.</strong> The pinned {tool === 'argo-cd' ? 'Argo CD' : 'Prometheus'} chart
            includes images with known vulnerabilities that remain in the latest official chart reviewed.
            Review date <time dateTime={acceptedRisks.reviewDate}>{acceptedRisks.reviewDate}</time>.
          </p>
        )}
        <ValidationStatus tool={tool} kubernetesVersion={kubernetesVersion} values={currentPreview.text} />
        <label>
          Template
          <select value={format} onChange={event => setFormat(event.target.value === 'manifests' ? 'manifests' : 'values')}>
            <option value="values">Helm chart</option>
            <option value="manifests">YAML manifest</option>
          </select>
          <small>Format to copy, download, and deploy. Example Helm chart.</small>
        </label>
        <ExportActions key={format} text={format === 'values' ? valuesExport : currentManifest}
          filename={`${tool}-${format}.yaml`} load={format === 'manifests' ? loadExport : undefined} />
        <button className="compare-toggle" type="button" aria-pressed={compare} onClick={() => setCompare(open => !open)}>
          Compare Helm and Kubernetes YAML manifest
        </button>
        <pre aria-label="Install commands"><code>{commands.join('\n')}</code></pre>
        {currentManifest && (
          <div aria-label="Included components">
            <h3>Included components</h3>
            <ul>{Array.from(rendered.components, ([name, lines]) => (
              <li key={name}>{name} · about {lines} lines</li>
            ))}</ul>
          </div>
        )}
        {compare ? (
          <ChartComparison catalog={catalog} tool={tool} kubernetesVersion={kubernetesVersion} values={current} />
        ) : format === 'values' ? (
          <Preview current={currentPreview} previous={previewValues(catalog, previous)} />
        ) : (
          <>
            {manifest.key === renderKey && manifest.error && <p role="alert">{manifest.error}</p>}
            <Preview explainField={(text, line, signal) => explainManifestField(text, kubernetesVersion, line, signal)}
              current={{ text: rendered.workloads, notes: manifest.notes }} previous={{ text: prior.workloads, notes: manifest.previousNotes }} />
            {rendered.schemas && (
              <details>
                <summary>CRD schemas</summary>
                <p>A CustomResourceDefinition registers a new resource type and its validation schema.</p>
                <pre>{rendered.schemas}</pre>
              </details>
            )}
          </>
        )}
      </section>
    </div>
  );
}
