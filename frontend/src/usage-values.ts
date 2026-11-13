import argoUsage from '../../templates/argo-cd/usage.json' with { type: 'json' };

import prometheusUsage from '../../templates/prometheus/usage.json' with { type: 'json' };

export type UsageTemplateId = keyof typeof argoUsage | keyof typeof prometheusUsage;
export type UsageValues = Record<string, string | boolean | number>;

export const usageTemplateIds = {
  'argo-cd': ['application', 'app-project', 'application-set'],
  prometheus: ['service-scrape', 'scrape-job', 'alert-rule'],
} as const satisfies Record<string, readonly UsageTemplateId[]>;

export type UsageTool = keyof typeof usageTemplateIds;

type Field = {
  id: string;
  label: string;
  description: string;
  example?: string;
} & (
  { type: 'text'; default: string } |
  { type: 'number'; default: number } |
  { type: 'select'; default: string; choices: string[] } |
  { type: 'boolean'; default: boolean }
);

export const usageTemplates = { ...argoUsage, ...prometheusUsage } as Record<UsageTemplateId, {
  label: string;
  description: string;
  fields: Field[];
}>;

export function defaultUsageValues(id: UsageTemplateId): UsageValues {
  return Object.fromEntries(usageTemplates[id].fields.map(field => [field.id, field.default]));
}

export function usageFilename(id: UsageTemplateId) {
  return id === 'scrape-job' || id === 'alert-rule' ? `prometheus-${id}-values.yaml` : `${id}.yaml`;
}

export function renderUsage(id: UsageTemplateId, values: UsageValues, learning: boolean) {
  const lines: string[] = [];
  const notes = new Map<number, string>();
  const input = { ...defaultUsageValues(id), ...values };
  const quote = (value: UsageValues[string]) => JSON.stringify(value).replace(/[\u0085\u2028\u2029]/g,
    character => '\\u' + character.charCodeAt(0).toString(16).padStart(4, '0'),
  );
  const quoted = (key: string) => quote(input[key]);

  function write(indent: number, yaml: string, note: string) {
    const padding = ' '.repeat(indent);
    if (learning) lines.push(`${padding}# ${note}`);
    lines.push(padding + yaml);
    notes.set(lines.length, note);
  }

  if (id === 'service-scrape') {
    function port(key: 'servicePort' | 'targetPort') {
      const value = input[key];
      return typeof value === 'string' && /^\d+$/.test(value) ? Number(value) : value;
    }

    write(0, 'apiVersion: v1', 'Use the core Kubernetes API version that provides Services.');
    write(0, 'kind: Service', 'Expose the selected Pods through a stable network address.');
    write(0, 'metadata:', 'Name the Service and declare its scrape settings.');
    write(2, `name: ${quoted('name')}`, 'Give clients a stable Service name.');
    write(2, `namespace: ${quoted('namespace')}`, 'Create the Service in the namespace of the selected Pods.');
    write(2, 'annotations:', 'Tell the chart’s Prometheus discovery job how to scrape these targets.');
    write(4, 'prometheus.io/scrape: "true"', 'Include the Service’s EndpointSlice targets in the default scrape job.');
    write(4, `prometheus.io/scheme: ${quoted('scheme')}`, 'Use this protocol to connect to each target.');
    write(4, `prometheus.io/path: ${quoted('metricsPath')}`, 'Request metrics from this URL path.');
    write(4, `prometheus.io/port: ${quote(String(port('targetPort')))}`, 'Scrape the port the selected Pods listen on.');
    write(0, 'spec:', 'Define which Pods the Service reaches and which port it exposes.');
    write(2, 'selector:', 'Find the Pods by matching their labels.');
    write(4, `${quoted('selectorKey')}: ${quoted('selectorValue')}`, 'Select only Pods with this label key and value.');
    write(2, 'ports:', 'Expose a port for clients that use the Service.');
    write(4, `- port: ${quote(port('servicePort'))}`, 'Listen for clients on this Service port.');
    write(6, `targetPort: ${quote(port('targetPort'))}`, 'Send traffic to this port on the selected Pods.');
    return { text: lines.join('\n') + '\n', notes };
  }

  if (id === 'scrape-job' || id === 'alert-rule') {
    lines.push(
      '# Helm values to add to an existing standalone Prometheus install. Do not use kubectl apply.',
      '# Add the chart repository if needed.',
      '# helm repo add prometheus-community https://prometheus-community.github.io/helm-charts',
      '# The command uses the dashboard’s release name and namespace. Adjust them for your install.',
    );
    lines.push(id === 'alert-rule'
      ? '# Merge existing rule groups into this file first. Helm replaces the groups list during upgrade.'
      : '# Use a unique job name to preserve other jobs in the release.');
    lines.push(`# helm upgrade prometheus prometheus-community/prometheus --namespace monitoring --version 29.45.1 --reuse-values --values ${usageFilename(id)}`, '');

    if (id === 'scrape-job') {
      write(0, 'scrapeConfigs:', 'Add a job to the chart’s Prometheus scrape configuration.');
      write(2, `${quoted('jobName')}:`, 'Use this key as the job name shown by Prometheus.');
      write(4, 'enabled: true', 'Include this job when Helm builds the Prometheus configuration.');
      write(4, `scrape_interval: ${quoted('interval')}`, 'Wait this long between requests to the target.');
      write(4, `metrics_path: ${quoted('metricsPath')}`, 'Request metrics from this URL path.');
      write(4, `scheme: ${quoted('scheme')}`, 'Use this protocol to connect to the target.');
      write(4, 'static_configs:', 'Use an explicit target address for this job.');
      write(6, '- targets:', 'List the addresses to scrape in this group.');
      write(10, `- ${quoted('target')}`, 'Connect to this host and port from Prometheus.');
    } else {
      write(0, 'serverFiles:', 'Add a file to the chart’s Prometheus configuration directory.');
      write(2, 'alerting_rules.yml:', 'Use the rule file already loaded by the chart’s default configuration.');
      write(4, 'groups:', 'Organize rules into groups that Prometheus evaluates together.');
      write(6, `- name: ${quoted('groupName')}`, 'Name this evaluation group.');
      write(8, 'rules:', 'List the alerts to evaluate in this group.');
      write(10, `- alert: ${quoted('alertName')}`, 'Name the alert shown when its condition is met.');
      write(12, `expr: ${quoted('expression')}`, 'Evaluate this PromQL condition to find affected targets.');
      write(12, `for: ${quoted('duration')}`, 'Require the condition to stay true for this long before firing.');
      write(12, 'labels:', 'Attach values that can be used for filtering and routing.');
      write(14, `severity: ${quoted('severity')}`, 'Classify how urgent the alert is.');
      write(12, 'annotations:', 'Attach readable context to the alert.');
      write(14, `summary: ${quoted('summary')}`, 'Show a brief description of the problem.');
    }
    return { text: lines.join('\n') + '\n', notes };
  }

  function applicationSpec(indent: number) {
    write(indent, `project: ${quoted('project')}`, 'Use this AppProject to decide which sources and destinations are allowed.');
    write(indent, 'source:', 'Choose the Git content Argo CD will render.');
    write(indent + 2, `repoURL: ${quoted('repoURL')}`, 'Read manifests from this Git repository.');
    write(indent + 2, `targetRevision: ${quoted('targetRevision')}`, 'Select the Git branch, tag, or commit to read.');
    write(indent + 2, `path: ${quoted('path')}`, 'Read manifests from this directory inside the repository.');
    write(indent, 'destination:', 'Set where Argo CD sends the manifests.');
    write(indent + 2, `server: ${quoted('destinationServer')}`, 'Select the target Kubernetes API server by its URL.');
    write(indent + 2, `namespace: ${quoted('destinationNamespace')}`, 'Use this namespace for resources that do not specify one.');
  }

  const kind = usageTemplates[id].label;
  write(0, 'apiVersion: argoproj.io/v1alpha1', 'Use the Argo CD API group and version that serve this resource.');
  write(0, `kind: ${kind}`, usageTemplates[id].description);
  write(0, 'metadata:', 'Identify this resource in Kubernetes.');
  write(2, `name: ${quoted('name')}`, `Give this ${kind} a name in its namespace.`);
  write(2, `namespace: ${quoted('namespace')}`, 'Place the resource in a namespace that Argo CD watches.');

  if (id === 'application' && input.cascade) {
    write(2, 'finalizers:', 'Let Argo CD finish cleanup before Kubernetes removes the Application.');
    write(4, '- resources-finalizer.argocd.argoproj.io', 'Delete the managed resources when this Application is deleted.');
  }

  if (id === 'application') {
    write(0, 'spec:', 'Describe the app to deploy and how it should sync.');
    applicationSpec(2);
    if (input.automated || input.createNamespace) {
      write(2, 'syncPolicy:', 'Choose how Argo CD applies changes to the target cluster.');
    }
    if (input.automated) {
      write(4, 'automated:', 'Automatically sync changes from Git when this app is out of sync.');
      write(6, `prune: ${input.prune === true}`, input.prune ? 'Delete resources that have been removed from Git.' : 'Keep live resources when their manifests are removed from Git.');
      write(6, `selfHeal: ${input.selfHeal === true}`, input.selfHeal ? 'Restore the Git state if someone changes a live resource.' : 'Leave live changes alone until a sync is triggered for another reason.');
    }
    if (input.createNamespace) {
      write(4, 'syncOptions:', 'Pass options to the sync operation.');
      write(6, '- CreateNamespace=true', 'Create the destination namespace during sync if it is missing.');
    }
  } else if (id === 'app-project') {
    write(0, 'spec:', 'Set the deployment boundaries for Applications that use this project.');
    write(2, 'sourceRepos:', 'List Git repositories this project allows.');
    write(4, `- ${quoted('repoURL')}`, 'Permit Applications to read manifests from this repository.');
    write(2, 'destinations:', 'List cluster and namespace pairs where the project can deploy.');
    write(4, `- server: ${quoted('destinationServer')}`, 'Permit deployment to this Kubernetes API server.');
    write(6, `namespace: ${quoted('destinationNamespace')}`, 'Limit deployments on that server to this namespace.');
  } else {
    write(0, 'spec:', 'Describe how to generate Applications and what each one should deploy.');
    write(2, 'goTemplate: true', 'Let the controller substitute Go template expressions in string fields.');
    write(2, 'goTemplateOptions:', 'Set how Go templates handle input errors.');
    write(4, '- missingkey=error', 'Fail generation if a template refers to a missing field.');
    write(2, 'generators:', 'List where the controller gets its template inputs.');
    write(4, '- list:', 'Generate one Application for each item in this list.');
    write(8, 'elements:', 'Provide the environment value for each generated Application.');
    write(10, `- environment: ${quoted('firstEnvironment')}`, 'Generate an Application for the first environment.');
    write(10, `- environment: ${quoted('secondEnvironment')}`, 'Generate an Application for the second environment.');
    write(2, 'template:', 'Provide the Application definition that each environment fills in.');
    write(4, 'metadata:', 'Set the identity of each generated Application.');
    write(6, `name: ${quote(`${input.name}-{{ .environment }}`)}`, 'Append the environment to give each generated Application a distinct name.');
    write(4, 'spec:', 'Describe the app each generated Application will manage.');
    applicationSpec(6);
  }

  return { text: lines.join('\n') + '\n', notes };
}
