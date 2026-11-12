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
    ['Alertmanager', 'check'],
    ['Kube-state-metrics', 'check'],
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

    const current = () => page.evaluate(() => globalThis.testMonaco.getModels().at(-1)?.getValue() || '');
    const chooser = page.locator('section[aria-label="Configuration selection"] select').last();
    const formatControl = page.locator('.builder-output select').first();
    const elapsed = [];
    for (const format of ['values', 'manifests']) {
      for (const [tool, edits] of Object.entries(editsByTool)) {
        await chooser.selectOption(tool === 'prometheus' ? 'argo-cd' : 'prometheus');
        await chooser.selectOption(tool);
        await formatControl.selectOption(format);
        await page.waitForFunction(() => Boolean(globalThis.testMonaco.getModels().at(-1)?.getValue()));
        for (const [name, type, value] of edits) {
          const control = optionControl(page, name, type);
          const before = await current();
          const start = performance.now();
          if (type === 'check') await control.setChecked(!(await control.isChecked()));
          else if (type === 'select') await control.selectOption(value);
          else await control.fill(value);
          await page.waitForFunction(old => {
            const next = globalThis.testMonaco.getModels().at(-1)?.getValue();
            return Boolean(next && next !== old);
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

    console.log(`All ${elapsed.length} option edits updated the preview in under one second. Slowest ${Math.max(...elapsed).toFixed(0)} ms.`);
  } finally {
    await browser.close();
  }
});
