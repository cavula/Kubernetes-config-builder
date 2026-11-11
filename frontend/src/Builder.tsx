import { useState } from 'react';
import { catalogs, defaultValues, dependencyHint, previewValues, type Value } from './builder-values';
import { Preview } from './Preview';

export function Builder({ tool }: { tool: keyof typeof catalogs }) {
  const catalog = catalogs[tool];
  const [{ current, previous }, setValues] = useState(() => {
    const defaults = defaultValues(catalog);
    return { current: defaults, previous: defaults };
  });

  function update(path: string, value: Value) {
    setValues(({ current }) => ({ current: { ...current, [path]: value }, previous: current }));
  }

  return (
    <div className="builder">
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
        <h2>Helm values preview</h2>
        <Preview current={previewValues(catalog, current)} previous={previewValues(catalog, previous)} />
      </div>
    </div>
  );
}
