import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { test } from 'node:test';

const require = createRequire(new URL('../frontend/package.json', import.meta.url));
const { chromium } = require('playwright-core');

const editsByTool = {
  prometheus: [
    ['Scrape interval', 'input', '30s'],
    ['Retention period', 'input', '7d'],
    ['Storage size', 'input', '4Gi'],
    ['CPU and memory', 'fieldset', '75m'],
    ['Persistent storage', 'check'],
    ['Enable Alertmanager', 'check'],
    ['Enable kube-state-metrics', 'check'],
    ['Enable Node exporter', 'check'],
    ['Enable Pushgateway', 'check'],
  ],
  'argo-cd': [
    ['Server service type', 'select', 'NodePort'],
    ['API and UI server replicas', 'input', '2'],
    ['Repository server replicas', 'input', '2'],
    ['API and UI server CPU and memory', 'fieldset', '75m'],
    ['Repository server CPU and memory', 'fieldset', '80m'],
    ['API server metrics', 'check'],
    ['Dex', 'check'],
  ],
};

function optionControl(page, name, type) {
  const panel = page.locator('.builder-options');
  if (type === 'fieldset') {
    return panel.locator('fieldset').filter({ has: page.getByText(name, { exact: true }) }).locator('input').first();
  }
  return panel.locator('label').filter({ hasText: new RegExp('^' + name) })
    .locator(type === 'check' ? 'input[type=checkbox]' : type);
}

test('both tools update both preview formats, highlight edits, and explain YAML fields', { timeout: 60000 }, async () => {
  const browser = await chromium.launch({
    executablePath: process.env.KCB_CHROME_PATH,
    timeout: 15000,
    args: ['--disable-gpu'],
  });
  try {
    const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
    page.setDefaultTimeout(8000);
    await page.goto(process.env.KCB_APP_URL ?? 'http://127.0.0.1:5173');
    await page.locator('.view-line').first().waitFor();
    await page.evaluate(async () => {
      const url = performance.getEntriesByType('resource')
        .find(entry => /\/monaco-editor\.js(?:\?|$)/.test(entry.name))?.name;
      if (!url) throw new Error('Monaco dev module could not be found');
      globalThis.testMonaco = (await import(url)).editor;
    });

    const currentVersion = () => page.evaluate(() => globalThis.testMonaco.getModels().at(-1)?.getVersionId());
    const chooser = page.getByRole('combobox', { name: /^Tool/ });
    const preset = page.getByRole('combobox', { name: /^Preset/ });
    const formatControl = page.locator('.builder-output select').first();
    await preset.selectOption('minimal');
    await page.getByRole('heading', { name: '1. Choose a preset' }).waitFor();
    await page.getByRole('heading', { name: '2. Customize (optional)' }).waitFor();
    await page.getByRole('heading', { name: '3. Deploy' }).waitFor();
    for (const [name] of editsByTool.prometheus.filter(([name]) => name.startsWith('Enable '))) {
      assert.equal(await optionControl(page, name, 'check').isChecked(), false, name);
    }
    await page.getByText(/Prometheus server · about/).waitFor();
    assert.equal(await page.getByText(/Pushgateway · about/).count(), 0);
    await optionControl(page, 'Enable Pushgateway', 'check').check();
    await page.getByText(/Pushgateway · about/).waitFor();
    await preset.selectOption('recommended');
    assert.equal(await optionControl(page, 'Enable Alertmanager', 'check').isChecked(), true);
    assert.equal(await optionControl(page, 'Enable Pushgateway', 'check').isChecked(), false);
    await preset.selectOption('full');
    const elapsed = [];
    for (const format of ['values', 'manifests']) {
      for (const [tool, edits] of Object.entries(editsByTool)) {
        await chooser.selectOption(tool === 'prometheus' ? 'argo-cd' : 'prometheus');
        await chooser.selectOption(tool);
        await formatControl.selectOption(format);
        await page.waitForFunction(() => Boolean(globalThis.testMonaco.getModels().at(-1)?.getValue()));
        for (const [name, type, value] of edits) {
          const control = optionControl(page, name, type);
          const before = await currentVersion();
          const start = performance.now();
          if (type === 'check') await control.setChecked(!(await control.isChecked()));
          else if (type === 'select') await control.selectOption(value);
          else await control.fill(value);
          await page.waitForFunction(old => {
            const model = globalThis.testMonaco.getModels().at(-1);
            return Boolean(model && model.getValueLength() > 0 && model.getVersionId() !== old);
          }, before);
          const milliseconds = performance.now() - start;
          elapsed.push(milliseconds);
          assert.ok(milliseconds < 1000, `${tool} ${format} ${name} took ${milliseconds.toFixed(0)} ms`);
        }
      }
    }

    await chooser.selectOption('prometheus');
    const interval = optionControl(page, 'Scrape interval', 'input');
    for (const value of ['31s', '32s', '33s', '35s']) await interval.fill(value);
    const field = page.locator('.view-line').filter({ hasText: /scrape_interval.*35s/ }).last();
    await field.waitFor();
    await page.locator('.line-insert, .char-insert').first().waitFor({ state: 'attached' });
    await field.locator('span').filter({ hasText: 'scrape_interval' }).last().hover();
    await page.locator('.monaco-hover').getByText(/How often Prometheus collects metrics/).waitFor();

    await formatControl.selectOption('manifests');
    await page.waitForFunction(() => globalThis.testMonaco.getModels().at(-1)?.getValue().includes('automountServiceAccountToken:'));
    const manifestHover = page.locator('.monaco-hover').getByText(/automountServiceAccountToken|service account token/i);
    for (let attempt = 0; attempt < 10; attempt++) {
      await page.evaluate(() => {
        const model = globalThis.testMonaco.getModels().at(-1);
        const { startLineNumber, startColumn } = model.findMatches('automountServiceAccountToken:', false, false, true, null, false)[0].range;
        const editor = globalThis.testMonaco.getEditors().find(editor => editor.getModel() === model);
        editor.setPosition({ lineNumber: startLineNumber, column: startColumn + 1 });
        editor.revealLineInCenter(startLineNumber);
        editor.focus();
        editor.trigger('test', 'editor.action.showHover', {});
      });
      if (await manifestHover.isVisible()) break;
      await page.waitForTimeout(100);
    }
    await manifestHover.waitFor();

    async function download(label) {
      const pending = page.waitForEvent('download');
      await page.getByRole('button', { name: label }).click();
      const file = await pending;
      const stream = await file.createReadStream();
      const chunks = [];
      for await (const chunk of stream) chunks.push(chunk);
      return { name: file.suggestedFilename(), text: Buffer.concat(chunks).toString() };
    }

    await chooser.selectOption('argo-cd');
    const exportActions = page.getByLabel('Export active format');
    assert.deepEqual(await exportActions.getByRole('button').allTextContents(), ['Copy', 'Download']);
    const installCommands = page.getByLabel('Install commands');
    const values = await download('Download');
    assert.equal(values.name, 'argo-cd-values.yaml');
    assert.match(values.text, /^# helm repo add argo /);
    assert.match(values.text, /helm install argo-cd argo\/argo-cd --version 10\.17\.1/);
    assert.match(values.text, /Indentation groups settings/);
    assert.match(values.text, /argocd admin initial-password/);
    for (const command of (await installCommands.textContent()).split('\n')) {
      assert.ok(values.text.includes('# ' + command));
    }
    await formatControl.selectOption('manifests');
    await page.getByText('CRD schemas', { exact: true }).waitFor();
    assert.equal(await page.locator('details').getAttribute('open'), null);
    assert.deepEqual(await exportActions.getByRole('button').allTextContents(), ['Copy', 'Download']);
    const comparison = page.getByRole('button', { name: 'Compare Helm and Kubernetes YAML manifest' });
    await comparison.click();
    const panes = page.locator('.comparison-panes .preview');
    await panes.nth(1).waitFor();
    assert.equal(await panes.count(), 2);
    for (const pane of await panes.all()) {
      await pane.locator('.modified .line-numbers').first().waitFor();
      assert.equal(await pane.locator('.original .line-numbers').count(), 0);
    }
    await comparison.click();
    const workloads = await download('Download');
    assert.equal(workloads.name, 'argo-cd-manifests.yaml');
    assert.ok(workloads.text.startsWith('# ' + (await installCommands.textContent())));
    assert.match(workloads.text, /Server-side apply avoids the annotation size limit/);
    assert.match(workloads.text, /# Why: Use apps\/v1 because/);
    assert.match(workloads.text, /REQUIRED IF you enable/);
    assert.match(workloads.text, /^kind: CustomResourceDefinition$/m);
    assert.match(workloads.text, /Run this Redis initialization Job first/);

    console.log(`All ${elapsed.length} option edits updated the preview in under one second. Slowest ${Math.max(...elapsed).toFixed(0)} ms.`);
  } finally {
    await browser.close();
  }
});
