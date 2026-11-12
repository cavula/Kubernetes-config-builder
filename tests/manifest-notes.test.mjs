import assert from 'node:assert/strict';
import { test } from 'node:test';
import { manifestNotes } from '../frontend/src/manifest-notes.ts';

test('rendered YAML links resource purposes and inline explanations to fields', () => {
  const notes = manifestNotes(`---
# Purpose: Maintain the requested replicas.
# Source: chart/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
        - name: example
          # This tag supplies ARM64 images.
          image: example:1
---
apiVersion: v1
kind: ConfigMap
`);
  assert.equal(notes.get(5), 'Maintain the requested replicas.');
  assert.equal(notes.get(12), 'This tag supplies ARM64 images.');
  assert.equal(notes.get(15), undefined);
});
