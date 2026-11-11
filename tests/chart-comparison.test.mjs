import assert from 'node:assert/strict';
import { test } from 'node:test';
import { catalogs, comparisonValues, defaultValues, previewValues } from '../frontend/src/builder-values.ts';

for (const [tool, catalog] of Object.entries(catalogs)) {
  test(`${tool} comparison varies each approved setting independently`, () => {
    const current = defaultValues(catalog);
    for (const { path } of catalog.options) {
      const alternate = comparisonValues(catalog, path, current);
      assert.ok(alternate, path);
      assert.deepEqual(Object.keys(alternate).filter(key => alternate[key] !== current[key]), [path]);
      assert.notEqual(previewValues(catalog, alternate).text, previewValues(catalog, current).text);
      assert.ok(previewValues(catalog, current).fieldLines.has(path));
    }
  });
}

test('comparison omits a storage size contrast when persistence is disabled', () => {
  const catalog = catalogs.prometheus;
  const current = { ...defaultValues(catalog), 'server.persistentVolume.enabled': false };
  assert.equal(comparisonValues(catalog, 'server.persistentVolume.size', current), undefined);
  const alternate = comparisonValues(catalog, 'server.persistentVolume.enabled', current);
  assert.equal(alternate['server.persistentVolume.enabled'], true);
  assert.match(previewValues(catalog, alternate).text, /size: "8Gi"/);
});
