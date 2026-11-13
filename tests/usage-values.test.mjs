import { deepEqual, doesNotMatch, equal, match, ok } from 'node:assert/strict';
import { test } from 'node:test';
import { defaultUsageValues, renderUsage, usageFilename, usageTemplateIds, usageTemplates } from '../frontend/src/usage-values.ts';

test('each usage YAML line has a plain-language comment in Learning mode', () => {
  for (const id of Object.keys(usageTemplates)) {
    const values = defaultUsageValues(id);
    const learning = renderUsage(id, values, true);
    const compact = renderUsage(id, values, false);
    const yamlLines = compact.text.trimEnd().split('\n').filter(line => line.trim() && !line.trimStart().startsWith('#'));
    const explainedLines = learning.text.trimEnd().split('\n');
    const yamlWithIndexes = explainedLines.flatMap((line, index) =>
      line.trim() && !line.trimStart().startsWith('#') ? [[line, index]] : []);

    equal(learning.notes.size, yamlLines.length, id);
    deepEqual(yamlWithIndexes.map(([line]) => line), yamlLines, id);
    for (const [yaml, index] of yamlWithIndexes) {
      const note = learning.notes.get(index + 1);
      ok(note, `${id} lacks an explanation for ${yaml}`);
      equal(explainedLines[index - 1], `${yaml.match(/^ */)[0]}# ${note}`, id);
    }
  }
});

test('Application defaults to manual sync and exposes explicit deletion options', () => {
  const defaults = renderUsage('application', {}, false).text;
  match(defaults, /kind: Application\n/);
  match(defaults, /- CreateNamespace=true\n/);
  doesNotMatch(defaults, /automated:|prune:|selfHeal:|finalizers:/);

  const enabled = renderUsage('application', { automated: true, prune: true, selfHeal: true, cascade: true }, true);
  match(enabled.text, /resources-finalizer\.argocd\.argoproj\.io/);
  match(enabled.text, /prune: true\n/);
  match(enabled.text, /selfHeal: true\n/);
  ok([...enabled.notes.values()].some(note => note.includes('Delete the managed resources')));
});

test('AppProject scopes its repository and destination to form values', () => {
  const { text } = renderUsage('app-project', {
    repoURL: 'https://example.org/my/repo.git',
    destinationServer: 'https://kubernetes.example.org',
    destinationNamespace: 'restricted',
  }, false);
  match(text, /kind: AppProject\n/);
  match(text, /sourceRepos:\n    - "https:\/\/example\.org\/my\/repo\.git"/);
  match(text, /destinations:\n    - server: "https:\/\/kubernetes\.example\.org"\n      namespace: "restricted"/);
});

test('ApplicationSet generates distinct named Applications from its list inputs', () => {
  const { text } = renderUsage('application-set', {
    name: 'preview',
    firstEnvironment: 'alpha',
    secondEnvironment: 'beta',
    destinationNamespace: 'preview-{{ .environment }}',
  }, false);
  match(text, /kind: ApplicationSet\n/);
  match(text, /goTemplate: true/);
  match(text, /- environment: "alpha"\n          - environment: "beta"/);
  match(text, /name: "preview-\{\{ \.environment \}\}"/);
  match(text, /namespace: "preview-\{\{ \.environment \}\}"/);
});

test('form text stays quoted on one YAML line', () => {
  const revision = 'HEAD"\n  project: injected #\u2028';
  const { text } = renderUsage('application', { targetRevision: revision }, false);
  ok(text.split('\n').includes(`    targetRevision: ${JSON.stringify(revision).replace('\u2028', '\\u2028')}`));
  doesNotMatch(text, /^  project: injected/m);
});


test('standalone Prometheus templates generate Service annotations and Helm values', () => {
  const service = renderUsage('service-scrape', { servicePort: 80, targetPort: '9100', scheme: 'https' }, false).text;
  match(service, /^apiVersion: v1\nkind: Service\n/);
  match(service, /prometheus.io\/scrape: "true"/);
  match(service, /prometheus.io\/scheme: "https"/);
  match(service, /prometheus.io\/path: "\/metrics"/);
  match(service, /prometheus.io\/port: "9100"/);
  match(service, /- port: 80\n      targetPort: 9100/);

  const job = renderUsage('scrape-job', { jobName: 'other-job', target: 'metrics.example:9100' }, false).text;
  match(job, /scrapeConfigs:\n  "other-job":\n    enabled: true/);
  match(job, /static_configs:\n      - targets:\n          - "metrics.example:9100"/);

  const rule = renderUsage('alert-rule', { duration: '10m' }, false).text;
  match(rule, /serverFiles:\n  alerting_rules.yml:\n    groups:/);
  match(rule, /for: "10m"/);
  match(rule, /# Merge existing rule groups into this file first/);
});

test('both Helm fragment exports keep the pinned upgrade command in every mode', () => {
  for (const id of usageTemplateIds.prometheus.filter(id => id !== 'service-scrape')) {
    for (const learning of [true, false]) {
      const { text } = renderUsage(id, {}, learning);
      match(text, /^# Helm values to add to an existing standalone Prometheus install\. Do not use kubectl apply\./);
      ok(text.includes(`# helm upgrade prometheus prometheus-community/prometheus --namespace monitoring --version 29.45.1 --reuse-values --values ${usageFilename(id)}\n`));
    }
  }
});
