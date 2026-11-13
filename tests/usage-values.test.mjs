import { deepEqual, doesNotMatch, equal, match, ok } from 'node:assert/strict';
import { test } from 'node:test';
import { defaultUsageValues, renderUsage, usageTemplates } from '../frontend/src/usage-values.ts';

test('each usage YAML line has a plain-language comment in Learning mode', () => {
  for (const id of Object.keys(usageTemplates)) {
    const values = defaultUsageValues(id);
    const learning = renderUsage(id, values, true);
    const compact = renderUsage(id, values, false);
    const yamlLines = compact.text.trimEnd().split('\n');
    const explainedLines = learning.text.trimEnd().split('\n');

    equal(explainedLines.length, yamlLines.length * 2, id);
    equal(learning.notes.size, yamlLines.length, id);
    deepEqual(explainedLines.filter((_, index) => index % 2 === 1), yamlLines, id);
    for (const [index, yaml] of yamlLines.entries()) {
      const note = learning.notes.get(index * 2 + 2);
      ok(note, `${id} lacks an explanation for ${yaml}`);
      equal(explainedLines[index * 2], `${yaml.match(/^ */)[0]}# ${note}`, id);
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
