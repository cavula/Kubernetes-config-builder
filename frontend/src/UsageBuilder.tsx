import { useState } from 'react';
import { ExportActions } from './ExportActions';
import { Preview } from './Preview';
import { ValidationStatus } from './ValidationStatus';
import { defaultUsageValues, renderUsage, usageTemplates, type UsageTemplateId, type UsageValues } from './usage-values';

export function UsageBuilder({ kubernetesVersion }: { kubernetesVersion: string }) {
  const [id, setId] = useState<UsageTemplateId>('application');
  return (
    <>
      <section className="step">
        <label>
          Usage template
          <select value={id} onChange={event => setId(event.target.value as UsageTemplateId)}>
            {Object.entries(usageTemplates).map(([key, template]) => <option key={key} value={key}>{template.label}</option>)}
          </select>
        </label>
      </section>
      <UsageForm key={id} id={id} kubernetesVersion={kubernetesVersion} />
    </>
  );
}

function UsageForm({ id, kubernetesVersion }: { id: UsageTemplateId; kubernetesVersion: string }) {
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
            ) : (
              <input type="text" value={String(values.current[field.id])} onChange={event => update(field.id, event.target.value)} />
            )}
            <small>{field.description}{'example' in field && field.example ? ` Example ${field.example}.` : ''}</small>
          </label>
        ))}
      </section>
      <section className="step builder-output" aria-labelledby="usage-preview">
        <h2 id="usage-preview">Usage YAML</h2>
        <label>
          Comment density
          <select value={learning ? 'learning' : 'compact'} onChange={event => setLearning(event.target.value === 'learning')}>
            <option value="learning">Learning</option>
            <option value="compact">Compact</option>
          </select>
        </label>
        <ExportActions text={current.text} filename={`${id}.yaml`} />
        <ValidationStatus tool="argo-cd" kubernetesVersion={kubernetesVersion} values={current.text} usage />
        <Preview current={current} previous={previous} />
      </section>
    </div>
  );
}
