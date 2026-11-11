import assert from 'node:assert/strict';
import { test } from 'node:test';
import { selectConfiguration } from '../frontend/src/selection.ts';

test('resolves both chart pins across all supported Kubernetes minors', () => {
  for (const version of ['1.37', '1.36', '1.35']) {
    for (const [tool, pin] of [['prometheus', '29.45.1'], ['argo-cd', '10.17.1']]) {
      const selection = selectConfiguration('k3s', version, tool);
      assert.equal(selection.platform.id, 'k3s');
      assert.equal(selection.kubernetesVersion, version);
      assert.equal(selection.tool.id, tool);
      assert.equal(selection.tool.chartVersion, pin);
    }
  }
});

test('rejects unsupported platform, version, and tool identifiers', () => {
  for (const invalid of [
    ['eks', '1.37', 'prometheus'],
    ['k3s', '1.34', 'prometheus'],
    ['k3s', '1.37', 'flux'],
  ]) {
    assert.throws(() => selectConfiguration(...invalid), RangeError);
  }
});
