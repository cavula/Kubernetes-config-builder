import assert from 'node:assert/strict';
import { test } from 'node:test';
import { catalogs, defaultValues, dependencyHint, previewValues } from '../frontend/src/builder-values.ts';

const edits = {
  prometheus: ['30s', '7d', false, '4Gi', { requests: { cpu: '50m', memory: '64Mi' } }, false, false],
  'argo-cd': ['NodePort', 2, 2, { requests: { cpu: '100m' } }, { limits: { memory: '256Mi' } }, true, false],
};

for (const [tool, catalog] of Object.entries(catalogs)) {
  test(`${tool} options update the preview within one second and the latest edit wins`, () => {
    const defaults = defaultValues(catalog);
    const original = previewValues(catalog, defaults).text;
    assert.equal(catalog.options.length, edits[tool].length);
    const start = performance.now();
    for (const [index, option] of catalog.options.entries()) {
      const changed = previewValues(catalog, { ...defaults, [option.path]: edits[tool][index] });
      assert.notEqual(changed.text, original, option.path);
      const restored = previewValues(catalog, { ...defaults, [option.path]: option.default });
      assert.equal(restored.text, original, option.path);
    }
    assert.ok(performance.now() - start < 1000);
  });

  test(`${tool} output comments match hover explanations for every field`, () => {
    const values = defaultValues(catalog);
    for (const option of catalog.options.filter(option => option.type === 'resources')) {
      values[option.path] = {
        requests: { cpu: '50m', memory: '64Mi' },
        limits: { cpu: '100m', memory: '128Mi' },
      };
    }
    const { text, notes } = previewValues(catalog, values);
    const lines = text.trimEnd().split('\n');
    for (let index = 1; index < lines.length; index += 2) {
      const comment = lines[index - 1].trimStart();
      assert.match(comment, /^# .+/);
      assert.equal(notes.get(index + 1), comment.slice(2));
      assert.ok(Object.values(catalog.notes.fields).includes(comment.slice(2)));
    }
  });
}

test('disabling persistence removes the size field and uses the catalog dependency hint', () => {
  const catalog = catalogs.prometheus;
  const values = defaultValues(catalog);
  const size = catalog.options.find(option => option.path === 'server.persistentVolume.size');
  assert.equal(dependencyHint(catalog, size, values), undefined);
  assert.match(previewValues(catalog, values).text, /size: "8Gi"/);
  values[size.requires] = false;
  assert.equal(dependencyHint(catalog, size, values), catalog.notes.dependencyHints[size.path]);
  assert.doesNotMatch(previewValues(catalog, values).text, /size:/);
});
