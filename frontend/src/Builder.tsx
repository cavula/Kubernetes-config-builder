import { useEffect, useState } from 'react';
import { catalogs, defaultValues, dependencyHint, previewValues, type Value } from './builder-values';
import { Preview } from './Preview';
import { ChartComparison } from './ChartComparison';
import { ValidationStatus } from './ValidationStatus';
import { renderChart } from './render-chart';

export function Builder({ tool, kubernetesVersion }: { tool: keyof typeof catalogs; kubernetesVersion: string }) {
  const catalog = catalogs[tool];
  const [{ current, previous }, setValues] = useState(() => {
    const defaults = defaultValues(catalog);
    return { current: defaults, previous: defaults };
  });

  const [format, setFormat] = useState<'values' | 'manifests'>('values');
  const [compare, setCompare] = useState(false);
  const [manifest, setManifest] = useState({ current: '', previous: '', error: '' });
  const currentPreview = previewValues(catalog, current);

  useEffect(() => {
    if (format !== 'manifests' || compare) return;
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      try {
        const text = await renderChart(tool, kubernetesVersion, currentPreview.text, controller.signal);
        if (!controller.signal.aborted) {
          setManifest(({ current }) => ({ current: text, previous: current || text, error: '' }));
        }
      } catch (error) {
        if (!controller.signal.aborted) {
          setManifest(previous => ({
            ...previous,
            error: error instanceof Error ? error.message : 'Could not render the chart.',
          }));
        }
      }
    }, 100);
    return () => {
      clearTimeout(timer);
      controller.abort();
    };
  }, [compare, format, tool, kubernetesVersion, currentPreview.text]);

  function update(path: string, value: Value) {
    setValues(({ current }) => ({ current: { ...current, [path]: value }, previous: current }));
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
        <ValidationStatus tool={tool} kubernetesVersion={kubernetesVersion} values={currentPreview.text} />
        <label>
          Preview format
          <select value={format} onChange={event => setFormat(event.target.value === 'manifests' ? 'manifests' : 'values')}>
            <option value="values">Helm values</option>
            <option value="manifests">Kubernetes manifests</option>
          </select>
        </label>
        <button type="button" aria-pressed={compare} onClick={() => setCompare(open => !open)}>
          {compare ? 'Close comparison' : 'Compare Helm values and YAML'}
        </button>
        {compare ? (
          <ChartComparison tool={tool} kubernetesVersion={kubernetesVersion} values={current} />
        ) : format === 'values' ? (
          <Preview current={currentPreview} previous={previewValues(catalog, previous)} />
        ) : (
          <>
            {manifest.error && <p role="alert">{manifest.error}</p>}
            <Preview current={{ text: manifest.current, notes: new Map() }} previous={{ text: manifest.previous, notes: new Map() }} />
          </>
        )}
      </div>
    </div>
  );
}
