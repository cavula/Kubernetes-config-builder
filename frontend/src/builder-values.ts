import prometheusARM64Values from '../../backend/prometheus-arm64-values.json' with { type: 'json' };
import prometheusOptions from '../../templates/prometheus/options.json' with { type: 'json' };
import prometheusNotes from '../../templates/prometheus/notes.json' with { type: 'json' };
import argoOptions from '../../templates/argo-cd/options.json' with { type: 'json' };
import argoNotes from '../../templates/argo-cd/notes.json' with { type: 'json' };

type Resources = Partial<Record<'requests' | 'limits', Partial<Record<'cpu' | 'memory', string>>>>;
export type Value = string | number | boolean | Resources;
type Option = {
  path: string;
  label: string;
  type: string;
  default: Value;
  requires?: string;
  choices?: string[];
  comparison?: string;
};

export type Catalog = { options: Option[]; fixedValues?: Record<string, string>; notes: { fields: Record<string, string>; dependencyHints: Partial<Record<string, string>> } };
export const catalogs: Record<'prometheus' | 'argo-cd', Catalog> = {
  prometheus: {
    options: prometheusOptions,
    notes: prometheusNotes,
    // Re-evaluate these values when updating the Prometheus chart pin.
    fixedValues: {
      'configmapReload.prometheus.image.tag': prometheusARM64Values.configmapReload.prometheus.image.tag,
      'alertmanager.configmapReload.image.tag': prometheusARM64Values.alertmanager.configmapReload.image.tag,
    },
  },
  'argo-cd': { options: argoOptions, notes: argoNotes },
};

export function catalogForChart(tool: keyof typeof catalogs, chartVersion: string): Catalog {
  const catalog = catalogs[tool];
  if (tool === 'prometheus' && chartVersion !== '29.45.1') {
    return { options: catalog.options, notes: catalog.notes };
  }
  return catalog;
}

export function defaultValues(catalog: Catalog): Record<string, Value> {
  return Object.fromEntries(catalog.options.map(option => [option.path, option.default]));
}

export function dependencyHint(catalog: Catalog, option: Option, values: Record<string, Value>) {
  return option.requires && !values[option.requires] ? catalog.notes.dependencyHints[option.path] : undefined;
}

export function comparisonValues(catalog: Catalog, path: string, values: Record<string, Value>) {
  const option = catalog.options.find(entry => entry.path === path);
  if (!option || (option.requires && !values[option.requires])) return undefined;
  const value = values[path];
  let alternate: Value;
  if (typeof value === 'boolean') {
    alternate = !value;
  } else if (typeof value === 'number') {
    alternate = value === 1 ? 2 : 1;
  } else if (typeof value === 'object') {
    const hasQuantities = Object.values(value).some(group => Object.values(group).some(Boolean));
    alternate = hasQuantities ? {} : { requests: { cpu: '100m', memory: '128Mi' } };
  } else {
    const sample = option.choices?.find(choice => choice !== value) ?? option.comparison;
    if (sample === undefined) return undefined;
    alternate = sample === value ? option.default : sample;
  }
  return { ...values, [path]: alternate };
}

type Field = { children: Map<string, Field>; value?: string | number | boolean };

export function previewValues(catalog: Catalog, values: Record<string, Value>) {
  const root: Field = { children: new Map() };

  function fieldAt(path: string, start = root) {
    let parent = start;
    for (const key of path.split('.')) {
      let field = parent.children.get(key);
      if (!field) {
        field = { children: new Map() };
        parent.children.set(key, field);
      }
      parent = field;
    }
    return parent;
  }

  for (const option of catalog.options) {
    if (option.requires && !values[option.requires]) continue;
    const field = fieldAt(option.path);
    const value = values[option.path];
    if (typeof value === 'object') {
      for (const [group, quantities] of Object.entries(value)) {
        for (const [resource, quantity] of Object.entries(quantities)) {
          if (quantity) {
            const resourceField = fieldAt(`${group}.${resource}`, field);
            resourceField.value = quantity;
          }
        }
      }
    } else {
      field.value = value;
    }
  }

  for (const [path, value] of Object.entries(catalog.fixedValues ?? {})) {
    fieldAt(path).value = value;
  }

  const lines: string[] = [];
  const notes = new Map<number, string>();
  const fieldLines = new Map<string, number>();
  function writeFields(parent: Field, depth: number, prefix = '') {
    const indent = '  '.repeat(depth);
    for (const [key, field] of parent.children) {
      const path = prefix ? `${prefix}.${key}` : key;
      const note = catalog.notes.fields[path];
      if (!note) throw new Error(`Missing chart explanation for ${path}`);
      lines.push(`${indent}# ${note}`);
      notes.set(lines.length + 1, note);
      fieldLines.set(path, lines.length + 1);
      const suffix = field.children.size ? '' : ` ${field.value === undefined ? '{}' : JSON.stringify(field.value)}`;
      lines.push(`${indent}${key}:${suffix}`);
      writeFields(field, depth + 1, path);
    }
  }
  writeFields(root, 0);
  return { text: lines.join('\n') + '\n', notes, fieldLines };
}


export function describeValuesPaths(preview: ReturnType<typeof previewValues>) {
  const lines = preview.text.split('\n');
  for (const [path, line] of preview.fieldLines) {
    const indent = lines[line - 2].match(/^ */)?.[0] ?? '';
    lines[line - 2] = `${indent}# ${path}: ${preview.notes.get(line)}`;
  }
  return lines.join('\n');
}

export const prometheusAddons = [
  { path: 'alertmanager.enabled', label: 'Alertmanager' },
  { path: 'kube-state-metrics.enabled', label: 'kube-state-metrics' },
  { path: 'prometheus-node-exporter.enabled', label: 'Node exporter' },
  { path: 'prometheus-pushgateway.enabled', label: 'Pushgateway' },
] as const;

export const presets = {
  minimal: 'Minimal (Prometheus server only, best for learning)',
  recommended: 'Recommended',
  full: 'Full chart defaults',
} as const;

export type Preset = keyof typeof presets;

export function presetValues(catalog: Catalog, preset: Preset) {
  const values = defaultValues(catalog);
  if (preset === 'minimal') {
    for (const addon of prometheusAddons) values[addon.path] = false;
  }
  if (preset === 'recommended') values['prometheus-pushgateway.enabled'] = false;
  return values;
}
