import argoUsage from '../../templates/argo-cd/usage.json' with { type: 'json' };

export type UsageTemplateId = keyof typeof argoUsage;
export type UsageValues = Record<string, string | boolean>;

type Field = {
  id: string;
  label: string;
  description: string;
  example?: string;
} & ({ type: 'text'; default: string } | { type: 'boolean'; default: boolean });

export const usageTemplates = argoUsage as Record<UsageTemplateId, {
  label: string;
  description: string;
  fields: Field[];
}>;

export function defaultUsageValues(id: UsageTemplateId): UsageValues {
  return Object.fromEntries(usageTemplates[id].fields.map(field => [field.id, field.default]));
}

export function renderUsage(id: UsageTemplateId, values: UsageValues, learning: boolean) {
  const lines: string[] = [];
  const notes = new Map<number, string>();
  const input = { ...defaultUsageValues(id), ...values };
  const quote = (value: string | boolean) => JSON.stringify(value).replace(/[\u0085\u2028\u2029]/g,
    character => '\\u' + character.charCodeAt(0).toString(16).padStart(4, '0'),
  );
  const quoted = (key: string) => quote(input[key]);

  function write(indent: number, yaml: string, note: string) {
    const padding = ' '.repeat(indent);
    if (learning) lines.push(`${padding}# ${note}`);
    lines.push(padding + yaml);
    notes.set(lines.length, note);
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
