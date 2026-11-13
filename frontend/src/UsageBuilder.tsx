import { useState } from 'react';
import { ExportActions } from './ExportActions';
import { Preview } from './Preview';
import { ValidationStatus } from './ValidationStatus';
import { defaultUsageValues, renderUsage, usageFilename, usageTemplateIds, usageTemplates, type UsageTemplateId, type UsageTool, type UsageValues } from './usage-values';

export function UsageBuilder({ tool, kubernetesVersion }: { tool: UsageTool; kubernetesVersion: string }) {
  const ids = usageTemplateIds[tool];
  const [id, setId] = useState<UsageTemplateId>(ids[0]);
  return (
    <>
      <section className="step">
        <label>
          Usage template
          <select value={id} onChange={event => setId(event.target.value as UsageTemplateId)}>
            {ids.map(key => <option key={key} value={key}>{usageTemplates[key].label}</option>)}
          </select>
        </label>
      </section>
      <UsageForm key={id} id={id} tool={tool} kubernetesVersion={kubernetesVersion} />
    </>
  );
}

function UsageForm({ id, tool, kubernetesVersion }: { id: UsageTemplateId; tool: UsageTool; kubernetesVersion: string }) {
  const template = usageTemplates[id];
  const [learning, setLearning] = useState(true);
  const [values, setValues] = useState(() => {
    const defaults = defaultUsageValues(id);
    return { current: defaults, previous: defaults };
  });
  const current = renderUsage(id, values.current, learning);
  const previous = renderUsage(id, values.previous, learning);

  function update(field: string, value: UsageValues[string]) {
    setValues(({ current }) => ({ current: { ...current, [field]: value }, previous: current }));
  }

  return (
    <div className="builder">
      <section className="step builder-options" aria-labelledby="usage-fields">
        <h2 id="usage-fields">{template.label} settings</h2>
        <p className="usage-description">{template.description}</p>
        {template.fields.map(field => (
          <label key={field.id}>
            {field.label}
            {field.type === 'boolean' ? (
              <input type="checkbox" checked={values.current[field.id] === true} onChange={event => update(field.id, event.target.checked)} />
            ) : field.type === 'select' ? (
              <select value={String(values.current[field.id])} onChange={event => update(field.id, event.target.value)}>
                {field.choices.map(choice => <option key={choice}>{choice}</option>)}
              </select>
            ) : field.type === 'number' ? (
              <input type="number" min={1} max={65535} value={String(values.current[field.id])}
                onChange={event => update(field.id, event.target.value === '' ? '' : event.target.valueAsNumber)} />
            ) : (
              <input type="text" value={String(values.current[field.id])} onChange={event => update(field.id, event.target.value)} />
            )}
            <small>{field.description}{'example' in field && field.example ? ` Example ${field.example}.` : ''}</small>
          </label>
        ))}
      </section>
      <section className="step builder-output" aria-labelledby="usage-preview">
        <h2 id="usage-preview">{id === 'scrape-job' || id === 'alert-rule' ? 'Helm values' : 'Usage YAML'}</h2>
        <label>
          Comment density
          <select value={learning ? 'learning' : 'compact'} onChange={event => setLearning(event.target.value === 'learning')}>
            <option value="learning">Learning</option>
            <option value="compact">Compact</option>
          </select>
        </label>
        <ExportActions text={current.text} filename={usageFilename(id)} />
        <ValidationStatus tool={tool} kubernetesVersion={kubernetesVersion} values={current.text} usageTemplate={id} />
        <Preview current={current} previous={previous} />
      </section>
    </div>
  );
}
